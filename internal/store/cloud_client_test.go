package store_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestCloudClient_ExportBoard(t *testing.T) {
	board := domain.NewBoard("test-board")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Task One",
		Type: domain.ItemTypeTask, Status: "todo",
		Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "item1"
	board.NextDisplay = 2

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/boards/board123/export" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}

		resp := store.APIResponse{
			OK:   true,
			Data: board,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	got, err := client.ExportBoard("board123")
	if err != nil {
		t.Fatalf("ExportBoard failed: %v", err)
	}

	if got.Name != "test-board" {
		t.Errorf("Name: got %q, want %q", got.Name, "test-board")
	}
	if len(got.Items) != 1 {
		t.Errorf("Items count: got %d, want 1", len(got.Items))
	}
}

func TestCloudClient_ExportBoard_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := store.APIResponse{
			OK: false,
			Error: &store.APIError{
				Code:    "BOARD_NOT_FOUND",
				Message: "Board xyz not found",
			},
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	_, err := client.ExportBoard("xyz")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestCloudClient_CreateItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/boards/board123/items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "New Task" {
			t.Errorf("title: got %v, want 'New Task'", body["title"])
		}

		resp := store.APIResponse{OK: true, Data: map[string]string{"id": "new-id"}}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	item := &domain.Item{
		ID: "new-id", Title: "New Task", Type: domain.ItemTypeTask,
		Status: "backlog", Priority: domain.PriorityMedium,
	}
	err := client.CreateItem("board123", item)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
}

func TestCloudClient_UpdateItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/items/item1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := store.APIResponse{OK: true, Data: map[string]string{"id": "item1"}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	item := &domain.Item{
		ID: "item1", Title: "Updated Task", Type: domain.ItemTypeTask,
		Status: "todo", Priority: domain.PriorityHigh,
	}
	err := client.UpdateItem(item)
	if err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}
}

func TestCloudClient_MoveItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/items/item1/move" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "done" {
			t.Errorf("status: got %v, want 'done'", body["status"])
		}

		resp := store.APIResponse{OK: true, Data: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	err := client.MoveItem("item1", "done")
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}
}

func TestCloudClient_DeleteItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/items/item1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := store.APIResponse{OK: true, Data: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	err := client.DeleteItem("item1")
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
}

func TestCloudClient_ImportBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/boards/import" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := store.APIResponse{OK: true, Data: map[string]string{"board_id": "new-board-id"}}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	board := domain.NewBoard("migrate-me")

	boardID, err := client.ImportBoard(board, "")
	if err != nil {
		t.Fatalf("ImportBoard failed: %v", err)
	}
	if boardID != "new-board-id" {
		t.Errorf("boardID: got %q, want 'new-board-id'", boardID)
	}
}

func TestCloudClient_CreateBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/boards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := store.APIResponse{OK: true, Data: map[string]string{"board_id": "created-id"}}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	boardID, err := client.CreateBoard("my-board", []string{"backlog", "todo", "done"}, "")
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	if boardID != "created-id" {
		t.Errorf("boardID: got %q, want 'created-id'", boardID)
	}
}

func TestCloudClient_GetMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/auth/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := store.APIResponse{
			OK: true,
			Data: map[string]string{
				"user_id":  "usr_123",
				"username": "niladri",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := store.NewCloudClient(server.URL+"/api", "test-token")
	userID, username, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}
	if userID != "usr_123" {
		t.Errorf("userID: got %q, want 'usr_123'", userID)
	}
	if username != "niladri" {
		t.Errorf("username: got %q, want 'niladri'", username)
	}
}
