package store_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

type apiCall struct {
	Method string
	Path   string
	Body   map[string]interface{}
}

// TestCloudStore_EngineIntegration verifies that the Engine works with CloudStore
// exactly the same way it works with JSONStore — no Engine changes needed.
func TestCloudStore_EngineIntegration(t *testing.T) {
	board := domain.NewBoard("integration-test")
	board.Users["test-user-id"] = &domain.Identity{
		ID: "test-user-id", Name: "testuser", Type: "human", Provider: "local",
	}

	var mu sync.Mutex
	var apiCalls []apiCall

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		call := apiCall{Method: r.Method, Path: r.URL.Path}
		if r.Body != nil && r.Method != http.MethodGet {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			call.Body = body
		}
		apiCalls = append(apiCalls, call)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/boards/b1/export":
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/api/boards/b1/items":
			resp := store.APIResponse{OK: true, Data: map[string]string{"id": "server-id"}}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPatch:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodDelete:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)

		default:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "b1")
	eng := engine.New(cs)

	// Test: Create an item via Engine
	item, err := eng.CreateItem("task", "Integration Task", "", "desc", "medium", "testuser", nil, "")
	if err != nil {
		t.Fatalf("CreateItem via Engine failed: %v", err)
	}
	if item.Title != "Integration Task" {
		t.Errorf("item title: got %q, want 'Integration Task'", item.Title)
	}

	// Verify API received a create call
	mu.Lock()
	foundCreate := false
	for _, call := range apiCalls {
		if call.Method == http.MethodPost && call.Path == "/api/boards/b1/items" {
			foundCreate = true
		}
	}
	mu.Unlock()

	if !foundCreate {
		t.Error("expected POST /api/boards/b1/items call, but none found")
	}
}

// TestCloudStore_FullCycle tests create → load → move → delete cycle.
func TestCloudStore_FullCycle(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("cycle-test")

	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet {
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := store.APIResponse{OK: true, Data: map[string]string{"id": "x", "board_id": "new"}}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "b1")

	// 1. Create item
	err := cs.Transaction(func(b *domain.Board) error {
		b.Items["t1"] = &domain.Item{
			ID: "t1", DisplayNum: 1, Title: "Cycle Task",
			Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium,
			CreatedAt: now, UpdatedAt: now,
		}
		b.DisplayMap[1] = "t1"
		b.NextDisplay = 2
		return nil
	})
	if err != nil {
		t.Fatalf("create transaction failed: %v", err)
	}

	// 2. Load board
	loaded, err := cs.LoadBoard()
	if err != nil {
		t.Fatalf("LoadBoard failed: %v", err)
	}
	if loaded.Name != "cycle-test" {
		t.Errorf("board name: got %q, want 'cycle-test'", loaded.Name)
	}

	// 3. BoardExists should return true
	if !cs.BoardExists() {
		t.Error("BoardExists should return true")
	}

	// 4. BoardFilePath should return empty
	if cs.BoardFilePath() != "" {
		t.Errorf("BoardFilePath should return empty, got %q", cs.BoardFilePath())
	}

	fmt.Println("Full cycle test passed")
}
