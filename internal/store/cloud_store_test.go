package store_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestCloudStore_LoadBoard(t *testing.T) {
	board := domain.NewBoard("cloud-test")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Cloud Task",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "item1"
	board.NextDisplay = 2

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := store.APIResponse{OK: true, Data: board}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "board1")
	got, err := cs.LoadBoard()
	if err != nil {
		t.Fatalf("LoadBoard failed: %v", err)
	}
	if got.Name != "cloud-test" {
		t.Errorf("Name: got %q, want %q", got.Name, "cloud-test")
	}
	if len(got.Items) != 1 {
		t.Errorf("Items: got %d, want 1", len(got.Items))
	}
}

func TestCloudStore_BoardExists(t *testing.T) {
	cs := store.NewCloudStore("http://example.com/api", "tok", "board1")
	if !cs.BoardExists() {
		t.Error("CloudStore.BoardExists should always return true")
	}
}

func TestCloudStore_BoardFilePath(t *testing.T) {
	cs := store.NewCloudStore("http://example.com/api", "tok", "board1")
	if cs.BoardFilePath() != "" {
		t.Errorf("CloudStore.BoardFilePath should return empty string, got %q", cs.BoardFilePath())
	}
}

func TestCloudStore_Transaction_CreateItem(t *testing.T) {
	board := domain.NewBoard("txn-test")

	var mu sync.Mutex
	var createdItems []map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/boards/board1/export":
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/api/boards/board1/items":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			createdItems = append(createdItems, body)
			resp := store.APIResponse{OK: true, Data: map[string]string{"id": "new"}}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)

		default:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "board1")

	err := cs.Transaction(func(b *domain.Board) error {
		item := &domain.Item{
			ID: "newtask", DisplayNum: b.NextDisplay, Title: "Created in txn",
			Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium,
		}
		b.Items[item.ID] = item
		b.DisplayMap[item.DisplayNum] = item.ID
		b.NextDisplay++
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(createdItems) != 1 {
		t.Errorf("expected 1 create call, got %d", len(createdItems))
	}
}

func TestCloudStore_Transaction_MoveItem(t *testing.T) {
	board := domain.NewBoard("txn-move")
	board.Items["mv1"] = &domain.Item{
		ID: "mv1", DisplayNum: 1, Title: "Move Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "mv1"
	board.NextDisplay = 2

	var movedStatus string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/boards/board1/export":
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/api/items/mv1/move":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			movedStatus = body["status"]
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)

		default:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "board1")

	err := cs.Transaction(func(b *domain.Board) error {
		b.Items["mv1"].Status = "done"
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if movedStatus != "done" {
		t.Errorf("expected move to 'done', got %q", movedStatus)
	}
}

func TestCloudStore_Transaction_DeleteItem(t *testing.T) {
	board := domain.NewBoard("txn-del")
	board.Items["del1"] = &domain.Item{
		ID: "del1", DisplayNum: 1, Title: "Delete Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "del1"
	board.NextDisplay = 2

	var deletedPath string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/boards/board1/export":
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodDelete:
			deletedPath = r.URL.Path
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)

		default:
			resp := store.APIResponse{OK: true, Data: nil}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "board1")

	err := cs.Transaction(func(b *domain.Board) error {
		delete(b.Items, "del1")
		delete(b.DisplayMap, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if deletedPath != "/api/items/del1" {
		t.Errorf("expected delete path '/api/items/del1', got %q", deletedPath)
	}
}

func TestCloudStore_Transaction_FnError_NoAPICalls(t *testing.T) {
	board := domain.NewBoard("txn-err")
	apiCallCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.URL.Path == "/api/boards/board1/export" {
			resp := store.APIResponse{OK: true, Data: board}
			json.NewEncoder(w).Encode(resp)
			return
		}
		apiCallCount++
		resp := store.APIResponse{OK: true, Data: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cs := store.NewCloudStore(server.URL+"/api", "tok", "board1")

	err := cs.Transaction(func(b *domain.Board) error {
		b.Items["x"] = &domain.Item{ID: "x", Title: "will fail"}
		return fmt.Errorf("intentional error")
	})

	if err == nil {
		t.Fatal("expected error from transaction, got nil")
	}

	mu.Lock()
	defer mu.Unlock()
	if apiCallCount != 0 {
		t.Errorf("expected 0 API calls after fn error, got %d", apiCallCount)
	}
}

func TestCloudStore_InitBoard(t *testing.T) {
	cs := store.NewCloudStore("http://example.com/api", "tok", "board1")
	err := cs.InitBoard("test", nil)
	if err == nil {
		t.Error("expected InitBoard to return error for CloudStore")
	}
}
