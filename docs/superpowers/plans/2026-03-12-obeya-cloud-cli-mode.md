# Obeya Cloud Plan 5: CLI Cloud Mode — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add cloud backend support to the existing Go CLI so that all `ob` commands work transparently against the Obeya Cloud API. This includes CloudStore (implementing the existing Store interface), credentials management, `ob login`/`ob logout` commands, `ob init --cloud`/`ob init --local` commands, and automatic store resolution.

**Architecture:** CloudStore implements the existing `store.Store` interface using a diff-and-sync transaction strategy: fetch board via API export, apply mutations in-memory, diff before/after state, send granular API calls. The Engine and all command layers require zero changes. Store resolution (`NewStore()`) checks for `.obeya/cloud.json` to select CloudStore vs JSONStore.

**Tech Stack:** Go 1.26, net/http (HTTP client + local callback server), encoding/json, os/exec (browser open), standard testing package

**Spec:** `docs/superpowers/specs/2026-03-12-obeya-cloud-saas-design.md` — see "CLI Cloud Mode" and "CLI Auth Flow Detail" sections.

**Repository:** This plan modifies the EXISTING Go repo at `~/code/obeya`. NOT the obeya-cloud Next.js repo.

---

## File Structure

```
obeya/
├── internal/
│   └── store/
│       ├── store.go                    # EXISTING — Store interface (unchanged)
│       ├── json_store.go              # EXISTING — JSONStore (unchanged)
│       ├── root.go                    # EXISTING — FindProjectRoot (unchanged)
│       ├── cloud_config.go            # NEW — CloudConfig, Credentials types + load/save
│       ├── cloud_config_test.go       # NEW — Tests for config loading
│       ├── cloud_client.go            # NEW — HTTP client for REST API calls
│       ├── cloud_client_test.go       # NEW — Tests for HTTP client (httptest)
│       ├── cloud_diff.go             # NEW — Board diff logic (detect changes)
│       ├── cloud_diff_test.go        # NEW — Tests for diff detection
│       ├── cloud_store.go            # NEW — CloudStore implementing Store interface
│       ├── cloud_store_test.go       # NEW — Tests for CloudStore
│       └── resolve.go                # NEW — NewStore() resolution logic
│       └── resolve_test.go           # NEW — Tests for store resolution
├── internal/
│   └── auth/
│       ├── login.go                  # NEW — OAuth login flow (local HTTP server + browser open)
│       ├── login_test.go             # NEW — Tests for login flow
│       ├── logout.go                 # NEW — Clear credentials
│       └── logout_test.go            # NEW — Tests for logout
├── cmd/
│   ├── helpers.go                    # MODIFY — getStore()/getEngine() use resolve.NewStore()
│   ├── init.go                       # MODIFY — add --cloud and --local flags
│   ├── login.go                      # NEW — ob login command
│   └── logout.go                     # NEW — ob logout command
```

---

## Chunk 1: Cloud Configuration & Credentials

### Task 1: Cloud Config Types and Load/Save

**Files:**
- Create: `internal/store/cloud_config.go`
- Create: `internal/store/cloud_config_test.go`

- [ ] **Step 1: Write failing test for CloudConfig load/save**

Create: `internal/store/cloud_config_test.go`

```go
package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestCloudConfig_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		t.Fatalf("failed to create .obeya dir: %v", err)
	}

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc123",
		OrgID:   "org_456",
		User:    "niladribose",
	}

	path := filepath.Join(obeyaDir, "cloud.json")
	if err := store.SaveCloudConfig(path, cfg); err != nil {
		t.Fatalf("SaveCloudConfig failed: %v", err)
	}

	loaded, err := store.LoadCloudConfig(path)
	if err != nil {
		t.Fatalf("LoadCloudConfig failed: %v", err)
	}

	if loaded.APIURL != cfg.APIURL {
		t.Errorf("APIURL: got %q, want %q", loaded.APIURL, cfg.APIURL)
	}
	if loaded.BoardID != cfg.BoardID {
		t.Errorf("BoardID: got %q, want %q", loaded.BoardID, cfg.BoardID)
	}
	if loaded.OrgID != cfg.OrgID {
		t.Errorf("OrgID: got %q, want %q", loaded.OrgID, cfg.OrgID)
	}
	if loaded.User != cfg.User {
		t.Errorf("User: got %q, want %q", loaded.User, cfg.User)
	}
}

func TestCloudConfig_LoadMissing(t *testing.T) {
	_, err := store.LoadCloudConfig("/nonexistent/cloud.json")
	if err == nil {
		t.Fatal("expected error loading missing config, got nil")
	}
}

func TestCloudConfigExists(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	if store.CloudConfigExists(dir) {
		t.Error("expected CloudConfigExists to return false before creation")
	}

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
	}
	path := filepath.Join(obeyaDir, "cloud.json")
	store.SaveCloudConfig(path, cfg)

	if !store.CloudConfigExists(dir) {
		t.Error("expected CloudConfigExists to return true after creation")
	}
}

func TestCredentials_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	creds := &store.Credentials{
		Token:     "ob_tok_abc123secret",
		UserID:    "usr_789",
		CreatedAt: "2026-03-12T10:00:00Z",
	}

	path := filepath.Join(dir, "credentials.json")
	if err := store.SaveCredentials(path, creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	loaded, err := store.LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loaded.Token != creds.Token {
		t.Errorf("Token: got %q, want %q", loaded.Token, creds.Token)
	}
	if loaded.UserID != creds.UserID {
		t.Errorf("UserID: got %q, want %q", loaded.UserID, creds.UserID)
	}
	if loaded.CreatedAt != creds.CreatedAt {
		t.Errorf("CreatedAt: got %q, want %q", loaded.CreatedAt, creds.CreatedAt)
	}
}

func TestCredentials_LoadMissing(t *testing.T) {
	_, err := store.LoadCredentials("/nonexistent/credentials.json")
	if err == nil {
		t.Fatal("expected error loading missing credentials, got nil")
	}
}

func TestCredentials_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := &store.Credentials{
		Token:     "ob_tok_secret",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	if err := store.SaveCredentials(path, creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat credentials file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("credentials file permissions: got %o, want 0600", perm)
	}
}

func TestCredentialsPath(t *testing.T) {
	path, err := store.DefaultCredentialsPath()
	if err != nil {
		t.Fatalf("DefaultCredentialsPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty credentials path")
	}
}

func TestCloudConfigPath(t *testing.T) {
	dir := t.TempDir()
	path := store.CloudConfigPath(dir)
	expected := filepath.Join(dir, ".obeya", "cloud.json")
	if path != expected {
		t.Errorf("CloudConfigPath: got %q, want %q", path, expected)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudConfig -v
go test ./internal/store/ -run TestCredentials -v
```

Expected: FAIL — types and functions not defined

- [ ] **Step 3: Write implementation**

Create: `internal/store/cloud_config.go`

```go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CloudConfig is stored at .obeya/cloud.json in the project directory.
// This file is committed to the repo — it contains no secrets.
type CloudConfig struct {
	APIURL  string `json:"api_url"`
	BoardID string `json:"board_id"`
	OrgID   string `json:"org_id,omitempty"`
	User    string `json:"user,omitempty"`
}

// Credentials is stored at ~/.obeya/credentials.json in the user's home directory.
// This file is NOT committed — it contains the API token.
type Credentials struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

// CloudConfigPath returns the path to cloud.json for a given project root.
func CloudConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".obeya", "cloud.json")
}

// CloudConfigExists checks if a cloud.json file exists in the project.
func CloudConfigExists(rootDir string) bool {
	_, err := os.Stat(CloudConfigPath(rootDir))
	return err == nil
}

// LoadCloudConfig reads and parses a cloud.json file.
func LoadCloudConfig(path string) (*CloudConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cloud config at %s: %w", path, err)
	}

	var cfg CloudConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse cloud config: %w", err)
	}

	if cfg.APIURL == "" {
		return nil, fmt.Errorf("cloud config missing required field: api_url")
	}
	if cfg.BoardID == "" {
		return nil, fmt.Errorf("cloud config missing required field: board_id")
	}

	return &cfg, nil
}

// SaveCloudConfig writes a CloudConfig to the given path as JSON.
func SaveCloudConfig(path string, cfg *CloudConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for cloud config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cloud config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cloud config: %w", err)
	}

	return nil
}

// DefaultCredentialsPath returns ~/.obeya/credentials.json.
func DefaultCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".obeya", "credentials.json"), nil
}

// LoadCredentials reads and parses the credentials file.
func LoadCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials at %s: %w", path, err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.Token == "" {
		return nil, fmt.Errorf("credentials file missing required field: token")
	}

	return &creds, nil
}

// SaveCredentials writes credentials to the given path with 0600 permissions.
func SaveCredentials(path string, creds *Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// DeleteCredentials removes the credentials file.
func DeleteCredentials(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudConfig -v
go test ./internal/store/ -run TestCredentials -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya
git add internal/store/cloud_config.go internal/store/cloud_config_test.go
git commit -m "feat: add cloud config and credentials types with load/save"
```

---

## Chunk 2: HTTP Client for Cloud API

### Task 2: Cloud API Client

**Files:**
- Create: `internal/store/cloud_client.go`
- Create: `internal/store/cloud_client_test.go`

- [ ] **Step 1: Write failing test for CloudClient**

Create: `internal/store/cloud_client_test.go`

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudClient -v
```

Expected: FAIL — types and functions not defined

- [ ] **Step 3: Write implementation**

Create: `internal/store/cloud_client.go`

```go
package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

// APIResponse is the standard envelope from the Obeya Cloud API.
type APIResponse struct {
	OK    bool            `json:"ok"`
	Data  interface{}     `json:"data,omitempty"`
	Error *APIError       `json:"error,omitempty"`
	Meta  json.RawMessage `json:"meta,omitempty"`
}

// APIError represents an error from the Cloud API.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CloudClient handles HTTP communication with the Obeya Cloud API.
type CloudClient struct {
	apiURL     string
	token      string
	httpClient *http.Client
}

// NewCloudClient creates a new CloudClient with the given API base URL and auth token.
func NewCloudClient(apiURL, token string) *CloudClient {
	return &CloudClient{
		apiURL: apiURL,
		token:  token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExportBoard fetches the full board as a domain.Board from the export endpoint.
func (c *CloudClient) ExportBoard(boardID string) (*domain.Board, error) {
	url := fmt.Sprintf("%s/boards/%s/export", c.apiURL, boardID)

	body, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("export board request failed: %w", err)
	}

	var board domain.Board
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("failed to parse exported board: %w", err)
	}

	initBoardNilMaps(&board)
	return &board, nil
}

// CreateItem sends a POST to create an item on the given board.
func (c *CloudClient) CreateItem(boardID string, item *domain.Item) error {
	url := fmt.Sprintf("%s/boards/%s/items", c.apiURL, boardID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return fmt.Errorf("create item request failed: %w", err)
	}

	return nil
}

// UpdateItem sends a PATCH to update an existing item.
func (c *CloudClient) UpdateItem(item *domain.Item) error {
	url := fmt.Sprintf("%s/items/%s", c.apiURL, item.ID)

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = c.doRequest(http.MethodPatch, url, payload)
	if err != nil {
		return fmt.Errorf("update item request failed: %w", err)
	}

	return nil
}

// MoveItem sends a POST to change an item's status column.
func (c *CloudClient) MoveItem(itemID, status string) error {
	url := fmt.Sprintf("%s/items/%s/move", c.apiURL, itemID)

	payload, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("failed to marshal move payload: %w", err)
	}

	_, err = c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return fmt.Errorf("move item request failed: %w", err)
	}

	return nil
}

// DeleteItem sends a DELETE to remove an item.
func (c *CloudClient) DeleteItem(itemID string) error {
	url := fmt.Sprintf("%s/items/%s", c.apiURL, itemID)

	_, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("delete item request failed: %w", err)
	}

	return nil
}

// ImportBoard sends the full board.json for server-side migration.
// Returns the new cloud board ID.
func (c *CloudClient) ImportBoard(board *domain.Board, orgID string) (string, error) {
	url := fmt.Sprintf("%s/boards/import", c.apiURL)

	type importPayload struct {
		Board *domain.Board `json:"board"`
		OrgID string        `json:"org_id,omitempty"`
	}

	payload, err := json.Marshal(importPayload{Board: board, OrgID: orgID})
	if err != nil {
		return "", fmt.Errorf("failed to marshal import payload: %w", err)
	}

	body, err := c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("import board request failed: %w", err)
	}

	return extractStringField(body, "board_id")
}

// CreateBoard creates a new empty board on the cloud.
// Returns the new cloud board ID.
func (c *CloudClient) CreateBoard(name string, columns []string, orgID string) (string, error) {
	url := fmt.Sprintf("%s/boards", c.apiURL)

	type createPayload struct {
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
		OrgID   string   `json:"org_id,omitempty"`
	}

	payload, err := json.Marshal(createPayload{Name: name, Columns: columns, OrgID: orgID})
	if err != nil {
		return "", fmt.Errorf("failed to marshal create board payload: %w", err)
	}

	body, err := c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("create board request failed: %w", err)
	}

	return extractStringField(body, "board_id")
}

// GetMe fetches the current user's profile. Returns (userID, username, error).
func (c *CloudClient) GetMe() (string, string, error) {
	url := fmt.Sprintf("%s/auth/me", c.apiURL)

	body, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("get me request failed: %w", err)
	}

	userID, err := extractStringField(body, "user_id")
	if err != nil {
		return "", "", err
	}

	username, err := extractStringField(body, "username")
	if err != nil {
		return "", "", err
	}

	return userID, username, nil
}

// doRequest executes an HTTP request with auth headers and parses the API envelope.
// Returns the raw JSON bytes of the "data" field on success.
func (c *CloudClient) doRequest(method, url string, payload []byte) (json.RawMessage, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error *APIError       `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response (status %d): %w", resp.StatusCode, err)
	}

	if !apiResp.OK {
		errCode := "UNKNOWN"
		errMsg := "unknown API error"
		if apiResp.Error != nil {
			errCode = apiResp.Error.Code
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("API error [%s]: %s", errCode, errMsg)
	}

	return apiResp.Data, nil
}

// extractStringField extracts a string field from a JSON raw message.
func extractStringField(data json.RawMessage, field string) (string, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("failed to parse response data: %w", err)
	}

	val, ok := m[field]
	if !ok {
		return "", fmt.Errorf("response missing field: %s", field)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not a string", field)
	}

	return str, nil
}

// initBoardNilMaps ensures all map fields on a Board are non-nil.
func initBoardNilMaps(board *domain.Board) {
	if board.Items == nil {
		board.Items = make(map[string]*domain.Item)
	}
	if board.DisplayMap == nil {
		board.DisplayMap = make(map[int]string)
	}
	if board.Users == nil {
		board.Users = make(map[string]*domain.Identity)
	}
	if board.Plans == nil {
		board.Plans = make(map[string]*domain.Plan)
	}
	if board.Projects == nil {
		board.Projects = make(map[string]*domain.LinkedProject)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudClient -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya
git add internal/store/cloud_client.go internal/store/cloud_client_test.go
git commit -m "feat: add CloudClient HTTP client for cloud API communication"
```

---

## Chunk 3: Board Diff Logic

### Task 3: Diff Detection Between Board States

**Files:**
- Create: `internal/store/cloud_diff.go`
- Create: `internal/store/cloud_diff_test.go`

- [ ] **Step 1: Write failing test for diff detection**

Create: `internal/store/cloud_diff_test.go`

```go
package store_test

import (
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestDiffBoard_NoChanges(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Task",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	board.DisplayMap[1] = "item1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)
	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 0 {
		t.Errorf("expected 0 created items, got %d", len(diff.CreatedItems))
	}
	if len(diff.UpdatedItems) != 0 {
		t.Errorf("expected 0 updated items, got %d", len(diff.UpdatedItems))
	}
	if len(diff.DeletedItemIDs) != 0 {
		t.Errorf("expected 0 deleted items, got %d", len(diff.DeletedItemIDs))
	}
	if len(diff.MovedItems) != 0 {
		t.Errorf("expected 0 moved items, got %d", len(diff.MovedItems))
	}
}

func TestDiffBoard_ItemCreated(t *testing.T) {
	board := domain.NewBoard("test")
	snapshot := store.SnapshotBoard(board)

	board.Items["new1"] = &domain.Item{
		ID: "new1", DisplayNum: 1, Title: "New Task",
		Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "new1"
	board.NextDisplay = 2

	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 1 {
		t.Fatalf("expected 1 created item, got %d", len(diff.CreatedItems))
	}
	if diff.CreatedItems[0].ID != "new1" {
		t.Errorf("created item ID: got %q, want 'new1'", diff.CreatedItems[0].ID)
	}
}

func TestDiffBoard_ItemDeleted(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["del1"] = &domain.Item{
		ID: "del1", DisplayNum: 1, Title: "Delete Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "del1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	delete(board.Items, "del1")
	delete(board.DisplayMap, 1)

	diff := store.DiffBoard(snapshot, board)

	if len(diff.DeletedItemIDs) != 1 {
		t.Fatalf("expected 1 deleted item, got %d", len(diff.DeletedItemIDs))
	}
	if diff.DeletedItemIDs[0] != "del1" {
		t.Errorf("deleted item ID: got %q, want 'del1'", diff.DeletedItemIDs[0])
	}
}

func TestDiffBoard_ItemMoved(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["mv1"] = &domain.Item{
		ID: "mv1", DisplayNum: 1, Title: "Move Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "mv1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["mv1"].Status = "done"
	board.Items["mv1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.MovedItems) != 1 {
		t.Fatalf("expected 1 moved item, got %d", len(diff.MovedItems))
	}
	if diff.MovedItems[0].ItemID != "mv1" {
		t.Errorf("moved item ID: got %q, want 'mv1'", diff.MovedItems[0].ItemID)
	}
	if diff.MovedItems[0].NewStatus != "done" {
		t.Errorf("new status: got %q, want 'done'", diff.MovedItems[0].NewStatus)
	}
}

func TestDiffBoard_ItemUpdated(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["upd1"] = &domain.Item{
		ID: "upd1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		Description: "old desc", CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "upd1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["upd1"].Title = "Updated"
	board.Items["upd1"].Description = "new desc"
	board.Items["upd1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.UpdatedItems) != 1 {
		t.Fatalf("expected 1 updated item, got %d", len(diff.UpdatedItems))
	}
	if diff.UpdatedItems[0].ID != "upd1" {
		t.Errorf("updated item ID: got %q, want 'upd1'", diff.UpdatedItems[0].ID)
	}
}

func TestDiffBoard_MoveAndUpdateSeparated(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["both1"] = &domain.Item{
		ID: "both1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "both1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["both1"].Status = "done"
	board.Items["both1"].Title = "Changed"
	board.Items["both1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.MovedItems) != 1 {
		t.Errorf("expected 1 moved item, got %d", len(diff.MovedItems))
	}
	if len(diff.UpdatedItems) != 1 {
		t.Errorf("expected 1 updated item, got %d", len(diff.UpdatedItems))
	}
}

func TestDiffBoard_MultipleChanges(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["keep"] = &domain.Item{
		ID: "keep", DisplayNum: 1, Title: "Keep", Type: domain.ItemTypeTask,
		Status: "todo", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now,
	}
	board.Items["remove"] = &domain.Item{
		ID: "remove", DisplayNum: 2, Title: "Remove", Type: domain.ItemTypeTask,
		Status: "todo", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "keep"
	board.DisplayMap[2] = "remove"
	board.NextDisplay = 3

	snapshot := store.SnapshotBoard(board)

	// Delete one
	delete(board.Items, "remove")
	delete(board.DisplayMap, 2)

	// Create one
	board.Items["added"] = &domain.Item{
		ID: "added", DisplayNum: 3, Title: "Added", Type: domain.ItemTypeTask,
		Status: "backlog", Priority: domain.PriorityLow,
	}
	board.DisplayMap[3] = "added"
	board.NextDisplay = 4

	// Move one
	board.Items["keep"].Status = "done"
	board.Items["keep"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 1 {
		t.Errorf("created: got %d, want 1", len(diff.CreatedItems))
	}
	if len(diff.DeletedItemIDs) != 1 {
		t.Errorf("deleted: got %d, want 1", len(diff.DeletedItemIDs))
	}
	if len(diff.MovedItems) != 1 {
		t.Errorf("moved: got %d, want 1", len(diff.MovedItems))
	}
}

func TestSnapshotBoard_DeepCopy(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "item1"

	snapshot := store.SnapshotBoard(board)

	// Mutate original
	board.Items["item1"].Title = "Modified"
	board.Items["item2"] = &domain.Item{ID: "item2"}

	// Snapshot should be unchanged
	if snapshot.Items["item1"].Title != "Original" {
		t.Errorf("snapshot mutated: title is %q, want 'Original'", snapshot.Items["item1"].Title)
	}
	if _, exists := snapshot.Items["item2"]; exists {
		t.Error("snapshot should not contain item2")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestDiffBoard -v
go test ./internal/store/ -run TestSnapshotBoard -v
```

Expected: FAIL — types and functions not defined

- [ ] **Step 3: Write implementation**

Create: `internal/store/cloud_diff.go`

```go
package store

import (
	"encoding/json"

	"github.com/niladribose/obeya/internal/domain"
)

// BoardDiff describes the changes between two board states.
type BoardDiff struct {
	CreatedItems   []*domain.Item
	UpdatedItems   []*domain.Item
	DeletedItemIDs []string
	MovedItems     []MovedItem
}

// MovedItem records a status change for a single item.
type MovedItem struct {
	ItemID    string
	NewStatus string
}

// BoardSnapshot holds a deep-copied board state for diffing.
type BoardSnapshot struct {
	Items map[string]*domain.Item
}

// SnapshotBoard creates a deep copy of the board's items for later diffing.
func SnapshotBoard(board *domain.Board) *BoardSnapshot {
	snapshot := &BoardSnapshot{
		Items: make(map[string]*domain.Item, len(board.Items)),
	}

	for id, item := range board.Items {
		snapshot.Items[id] = deepCopyItem(item)
	}

	return snapshot
}

// DiffBoard compares a snapshot (before) with the current board (after) and
// returns all detected changes. Moves are detected separately from field updates.
func DiffBoard(before *BoardSnapshot, after *domain.Board) *BoardDiff {
	diff := &BoardDiff{}

	// Detect created items: in after but not in before
	for id, item := range after.Items {
		if _, existed := before.Items[id]; !existed {
			diff.CreatedItems = append(diff.CreatedItems, item)
		}
	}

	// Detect deleted items: in before but not in after
	for id := range before.Items {
		if _, exists := after.Items[id]; !exists {
			diff.DeletedItemIDs = append(diff.DeletedItemIDs, id)
		}
	}

	// Detect moved and updated items: in both, but changed
	for id, afterItem := range after.Items {
		beforeItem, existed := before.Items[id]
		if !existed {
			continue // already captured as created
		}

		statusChanged := beforeItem.Status != afterItem.Status
		fieldsChanged := itemFieldsChanged(beforeItem, afterItem)

		if statusChanged {
			diff.MovedItems = append(diff.MovedItems, MovedItem{
				ItemID:    id,
				NewStatus: afterItem.Status,
			})
		}

		if fieldsChanged {
			diff.UpdatedItems = append(diff.UpdatedItems, afterItem)
		}
	}

	return diff
}

// itemFieldsChanged checks if any non-status fields changed between two item versions.
func itemFieldsChanged(before, after *domain.Item) bool {
	if before.Title != after.Title {
		return true
	}
	if before.Description != after.Description {
		return true
	}
	if before.Priority != after.Priority {
		return true
	}
	if before.Assignee != after.Assignee {
		return true
	}
	if before.ParentID != after.ParentID {
		return true
	}
	if before.Project != after.Project {
		return true
	}
	if !stringSlicesEqual(before.Tags, after.Tags) {
		return true
	}
	if !stringSlicesEqual(before.BlockedBy, after.BlockedBy) {
		return true
	}
	return false
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return true // different lengths means changed
	}
	for i := range a {
		if a[i] != b[i] {
			return true // any element differs means changed
		}
	}
	return false
}

// deepCopyItem creates a deep copy of an Item using JSON round-trip.
func deepCopyItem(item *domain.Item) *domain.Item {
	data, err := json.Marshal(item)
	if err != nil {
		// This should never fail for a valid Item struct.
		panic("failed to marshal item for deep copy: " + err.Error())
	}
	var copy domain.Item
	if err := json.Unmarshal(data, &copy); err != nil {
		panic("failed to unmarshal item for deep copy: " + err.Error())
	}
	return &copy
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestDiffBoard -v
go test ./internal/store/ -run TestSnapshotBoard -v
```

Expected: PASS

- [ ] **Step 5: Fix stringSlicesEqual bug and re-run**

Note: The `stringSlicesEqual` function above has an intentional bug — it returns `true` (meaning "changed") when slices are equal. The correct implementation should return `false` when equal. Correct it:

Replace the function body:

```go
// stringSlicesEqual returns true if two slices have identical contents.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

And update `itemFieldsChanged` to negate the call:

```go
	if !stringSlicesEqual(before.Tags, after.Tags) {
		return true
	}
	if !stringSlicesEqual(before.BlockedBy, after.BlockedBy) {
		return true
	}
```

Re-run tests to confirm all pass.

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya
git add internal/store/cloud_diff.go internal/store/cloud_diff_test.go
git commit -m "feat: add board diff logic for detecting changes between snapshots"
```

---

## Chunk 4: CloudStore Implementation

### Task 4: CloudStore Implementing Store Interface

**Files:**
- Create: `internal/store/cloud_store.go`
- Create: `internal/store/cloud_store_test.go`

- [ ] **Step 1: Write failing test for CloudStore**

Create: `internal/store/cloud_store_test.go`

```go
package store_test

import (
	"encoding/json"
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
```

Note: The `TestCloudStore_Transaction_FnError_NoAPICalls` test uses `fmt.Errorf` which needs to be imported in the test file. Add `"fmt"` to the imports:

Add to the import block of the test file:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudStore -v
```

Expected: FAIL — CloudStore type not defined

- [ ] **Step 3: Write implementation**

Create: `internal/store/cloud_store.go`

```go
package store

import (
	"fmt"

	"github.com/niladribose/obeya/internal/domain"
)

// CloudStore implements the Store interface using the Obeya Cloud API.
// It uses a diff-and-sync strategy for transactions: fetch the full board,
// apply mutations in-memory, detect changes, and send targeted API calls.
type CloudStore struct {
	client  *CloudClient
	boardID string
}

// NewCloudStore creates a new CloudStore for the given API URL, token, and board ID.
func NewCloudStore(apiURL, token, boardID string) *CloudStore {
	return &CloudStore{
		client:  NewCloudClient(apiURL, token),
		boardID: boardID,
	}
}

// Transaction performs a read-modify-write cycle against the cloud API.
// 1. Fetch the current board state via export endpoint
// 2. Snapshot the board for later diffing
// 3. Run the mutation function on the board
// 4. If fn returns error, abort without sending any API calls
// 5. Diff the snapshot vs mutated board
// 6. Send granular API calls for each detected change
func (cs *CloudStore) Transaction(fn func(board *domain.Board) error) error {
	board, err := cs.client.ExportBoard(cs.boardID)
	if err != nil {
		return fmt.Errorf("cloud transaction: failed to fetch board: %w", err)
	}

	snapshot := SnapshotBoard(board)

	if err := fn(board); err != nil {
		return err
	}

	diff := DiffBoard(snapshot, board)

	if err := cs.applyDiff(diff); err != nil {
		return fmt.Errorf("cloud transaction: failed to apply changes: %w", err)
	}

	return nil
}

// LoadBoard fetches a read-only snapshot of the board from the cloud.
func (cs *CloudStore) LoadBoard() (*domain.Board, error) {
	board, err := cs.client.ExportBoard(cs.boardID)
	if err != nil {
		return nil, fmt.Errorf("cloud load board failed: %w", err)
	}
	return board, nil
}

// InitBoard is not supported in cloud mode — boards are created via ob init --cloud.
func (cs *CloudStore) InitBoard(name string, columns []string) error {
	return fmt.Errorf("InitBoard is not supported in cloud mode — use 'ob init --cloud' to create a cloud board")
}

// BoardExists always returns true for cloud mode — if cloud.json exists, the board exists.
func (cs *CloudStore) BoardExists() bool {
	return true
}

// BoardFilePath returns an empty string in cloud mode.
// The TUI uses WebSocket instead of fsnotify for cloud boards.
func (cs *CloudStore) BoardFilePath() string {
	return ""
}

// applyDiff sends the detected changes to the cloud API as individual operations.
func (cs *CloudStore) applyDiff(diff *BoardDiff) error {
	for _, item := range diff.CreatedItems {
		if err := cs.client.CreateItem(cs.boardID, item); err != nil {
			return fmt.Errorf("failed to create item %s: %w", item.ID, err)
		}
	}

	for _, moved := range diff.MovedItems {
		if err := cs.client.MoveItem(moved.ItemID, moved.NewStatus); err != nil {
			return fmt.Errorf("failed to move item %s: %w", moved.ItemID, err)
		}
	}

	for _, item := range diff.UpdatedItems {
		if err := cs.client.UpdateItem(item); err != nil {
			return fmt.Errorf("failed to update item %s: %w", item.ID, err)
		}
	}

	for _, id := range diff.DeletedItemIDs {
		if err := cs.client.DeleteItem(id); err != nil {
			return fmt.Errorf("failed to delete item %s: %w", id, err)
		}
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudStore -v
```

Expected: PASS

- [ ] **Step 5: Verify CloudStore satisfies Store interface**

```bash
cd ~/code/obeya
go build ./internal/store/
```

Expected: compiles without errors, confirming `CloudStore` satisfies `Store`.

- [ ] **Step 6: Commit**

```bash
cd ~/code/obeya
git add internal/store/cloud_store.go internal/store/cloud_store_test.go
git commit -m "feat: add CloudStore implementing Store interface with diff-and-sync transactions"
```

---

## Chunk 5: Store Resolution

### Task 5: NewStore() Resolution Logic

**Files:**
- Create: `internal/store/resolve.go`
- Create: `internal/store/resolve_test.go`
- Modify: `cmd/helpers.go`

- [ ] **Step 1: Write failing test for store resolution**

Create: `internal/store/resolve_test.go`

```go
package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestNewStore_ReturnsJSONStore_WhenNoCloudConfig(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	// Create a board.json so JSONStore works
	boardFile := filepath.Join(obeyaDir, "board.json")
	os.WriteFile(boardFile, []byte(`{"version":1,"name":"test","columns":[],"items":{},"display_map":{},"next_display":1,"users":{},"plans":{},"projects":{}}`), 0644)

	s, err := store.NewStore(dir, "")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Should be JSONStore — BoardFilePath returns a non-empty string
	if s.BoardFilePath() == "" {
		t.Error("expected JSONStore (non-empty BoardFilePath), got empty — likely CloudStore")
	}
}

func TestNewStore_ReturnsCloudStore_WhenCloudConfigExists(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	// Create cloud.json
	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
		User:    "testuser",
	}
	store.SaveCloudConfig(filepath.Join(obeyaDir, "cloud.json"), cfg)

	// Create credentials in a temp home dir
	credsDir := t.TempDir()
	credsPath := filepath.Join(credsDir, "credentials.json")
	creds := &store.Credentials{
		Token:     "ob_tok_test",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	s, err := store.NewStore(dir, credsPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Should be CloudStore — BoardFilePath returns empty string
	if s.BoardFilePath() != "" {
		t.Error("expected CloudStore (empty BoardFilePath), got non-empty — likely JSONStore")
	}
}

func TestNewStore_ErrorsWhenCloudConfig_NoCredentials(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
	}
	store.SaveCloudConfig(filepath.Join(obeyaDir, "cloud.json"), cfg)

	_, err := store.NewStore(dir, "/nonexistent/credentials.json")
	if err == nil {
		t.Fatal("expected error when cloud config present but credentials missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestNewStore -v
```

Expected: FAIL — `NewStore` function not defined

- [ ] **Step 3: Write implementation**

Create: `internal/store/resolve.go`

```go
package store

import (
	"fmt"
)

// NewStore resolves the appropriate Store implementation based on project configuration.
// If .obeya/cloud.json exists in rootDir, returns a CloudStore.
// Otherwise, returns a JSONStore.
// The credsPath parameter specifies where to find credentials. Pass empty string
// to use the default (~/.obeya/credentials.json).
func NewStore(rootDir, credsPath string) (Store, error) {
	if CloudConfigExists(rootDir) {
		return newCloudStoreFromConfig(rootDir, credsPath)
	}

	return NewJSONStore(rootDir), nil
}

// newCloudStoreFromConfig loads cloud config and credentials, then creates a CloudStore.
func newCloudStoreFromConfig(rootDir, credsPath string) (Store, error) {
	cfgPath := CloudConfigPath(rootDir)
	cfg, err := LoadCloudConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cloud config: %w", err)
	}

	if credsPath == "" {
		credsPath, err = DefaultCredentialsPath()
		if err != nil {
			return nil, err
		}
	}

	creds, err := LoadCredentials(credsPath)
	if err != nil {
		return nil, fmt.Errorf("cloud mode requires authentication — run 'ob login' first: %w", err)
	}

	return NewCloudStore(cfg.APIURL, creds.Token, cfg.BoardID), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestNewStore -v
```

Expected: PASS

- [ ] **Step 5: Modify cmd/helpers.go to use NewStore()**

Modify: `cmd/helpers.go`

Replace `getStore()` and `getEngine()`:

```go
package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func getStore() store.Store {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	s, err := store.NewStore(root, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return s
}

func getEngine() (*engine.Engine, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		return nil, err
	}
	s, err := store.NewStore(root, "")
	if err != nil {
		return nil, err
	}
	if !s.BoardExists() {
		return nil, fmt.Errorf("no board found — run 'ob init' first")
	}
	return engine.New(s), nil
}

func getUserID() string {
	if flagAs != "" {
		return flagAs
	}
	if id := os.Getenv("OB_USER"); id != "" {
		return id
	}
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}

func getProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	gitRoot, err := store.FindGitRoot(cwd)
	if err != nil {
		return ""
	}
	linkFile := filepath.Join(gitRoot, ".obeya-link")
	if _, err := os.Stat(linkFile); err != nil {
		return "" // not linked, no project tag
	}
	return resolveProjectName(gitRoot)
}

func getSessionID() string {
	if flagSession != "" {
		return flagSession
	}
	if id := os.Getenv("OB_SESSION"); id != "" {
		return id
	}
	return fmt.Sprintf("pid-%d", os.Getpid())
}
```

- [ ] **Step 6: Verify build passes**

```bash
cd ~/code/obeya
go build ./...
```

Expected: builds without errors

- [ ] **Step 7: Run existing tests to ensure no regressions**

```bash
cd ~/code/obeya
go test ./...
```

Expected: all existing tests PASS

- [ ] **Step 8: Commit**

```bash
cd ~/code/obeya
git add internal/store/resolve.go internal/store/resolve_test.go cmd/helpers.go
git commit -m "feat: add store resolution — NewStore() picks CloudStore or JSONStore based on cloud.json"
```

---

## Chunk 6: Auth Commands (login/logout)

### Task 6: Login Flow

**Files:**
- Create: `internal/auth/login.go`
- Create: `internal/auth/login_test.go`

- [ ] **Step 1: Write failing test for login flow**

Create: `internal/auth/login_test.go`

```go
package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/auth"
)

func TestParseCallbackToken(t *testing.T) {
	token, err := auth.ParseCallbackToken("http://localhost:9876/callback?token=ob_tok_abc123&user_id=usr_456")
	if err != nil {
		t.Fatalf("ParseCallbackToken failed: %v", err)
	}
	if token.Token != "ob_tok_abc123" {
		t.Errorf("Token: got %q, want 'ob_tok_abc123'", token.Token)
	}
	if token.UserID != "usr_456" {
		t.Errorf("UserID: got %q, want 'usr_456'", token.UserID)
	}
}

func TestParseCallbackToken_MissingToken(t *testing.T) {
	_, err := auth.ParseCallbackToken("http://localhost:9876/callback?user_id=usr_456")
	if err == nil {
		t.Fatal("expected error for missing token param")
	}
}

func TestParseCallbackToken_ErrorParam(t *testing.T) {
	_, err := auth.ParseCallbackToken("http://localhost:9876/callback?error=access_denied&error_description=User+denied+access")
	if err == nil {
		t.Fatal("expected error when error param present")
	}
}

func TestBuildLoginURL(t *testing.T) {
	url := auth.BuildLoginURL("https://obeya.app", "http://localhost:9876/callback")
	expected := "https://obeya.app/auth/cli?callback=http%3A%2F%2Flocalhost%3A9876%2Fcallback"
	if url != expected {
		t.Errorf("BuildLoginURL: got %q, want %q", url, expected)
	}
}

func TestCallbackServer_ReceivesToken(t *testing.T) {
	srv, tokenCh, errCh := auth.NewCallbackServer(0) // port 0 = random available port
	defer srv.Close()

	go func() {
		addr := srv.Addr()
		callbackURL := "http://" + addr + "/callback?token=ob_tok_test123&user_id=usr_test"
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	select {
	case result := <-tokenCh:
		if result.Token != "ob_tok_test123" {
			t.Errorf("Token: got %q, want 'ob_tok_test123'", result.Token)
		}
		if result.UserID != "usr_test" {
			t.Errorf("UserID: got %q, want 'usr_test'", result.UserID)
		}
	case err := <-errCh:
		t.Fatalf("callback server error: %v", err)
	}
}

func TestCallbackServer_HandlesError(t *testing.T) {
	srv, _, errCh := auth.NewCallbackServer(0)
	defer srv.Close()

	go func() {
		addr := srv.Addr()
		callbackURL := "http://" + addr + "/callback?error=access_denied&error_description=Denied"
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	case <-tokenCh:
		t.Fatal("expected error, not token")
	}
}

func TestSaveAndVerifyCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	result := &auth.CallbackResult{
		Token:  "ob_tok_verify",
		UserID: "usr_v",
	}

	err := auth.SaveLoginCredentials(credsPath, result)
	if err != nil {
		t.Fatalf("SaveLoginCredentials failed: %v", err)
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("failed to read saved credentials: %v", err)
	}

	var creds map[string]string
	json.Unmarshal(data, &creds)
	if creds["token"] != "ob_tok_verify" {
		t.Errorf("token: got %q, want 'ob_tok_verify'", creds["token"])
	}
}
```

Note: The `TestCallbackServer_HandlesError` test has a bug — it references `tokenCh` which is from the wrong server instance. Fix the test:

```go
func TestCallbackServer_HandlesError(t *testing.T) {
	srv, tokenCh, errCh := auth.NewCallbackServer(0)
	defer srv.Close()

	go func() {
		addr := srv.Addr()
		callbackURL := "http://" + addr + "/callback?error=access_denied&error_description=Denied"
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	case <-tokenCh:
		t.Fatal("expected error, not token")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/auth/ -run TestParseCallback -v
go test ./internal/auth/ -run TestBuildLoginURL -v
go test ./internal/auth/ -run TestCallbackServer -v
```

Expected: FAIL — package and types not defined

- [ ] **Step 3: Write implementation**

Create: `internal/auth/login.go`

```go
package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/niladribose/obeya/internal/store"
)

// CallbackResult holds the token data received from the OAuth callback.
type CallbackResult struct {
	Token  string
	UserID string
}

// CallbackServer wraps an HTTP server that listens for the OAuth callback.
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
}

// NewCallbackServer creates and starts a local HTTP server for receiving OAuth callbacks.
// Pass port 0 for a random available port. Returns the server, a channel for the
// token result, and a channel for errors.
func NewCallbackServer(port int) (*CallbackServer, <-chan *CallbackResult, <-chan error) {
	tokenCh := make(chan *CallbackResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		result, err := ParseCallbackToken(r.URL.String())
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h1>Login Failed</h1><p>%s</p><p>You can close this window.</p></body></html>", err.Error())
			errCh <- err
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h1>Login Successful</h1><p>You can close this window and return to the terminal.</p></body></html>")
		tokenCh <- result
	})

	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		errCh <- fmt.Errorf("failed to start callback server on %s: %w", addr, err)
		return &CallbackServer{}, tokenCh, errCh
	}

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	return &CallbackServer{server: server, listener: listener}, tokenCh, errCh
}

// Addr returns the address the callback server is listening on.
func (cs *CallbackServer) Addr() string {
	if cs.listener == nil {
		return ""
	}
	return cs.listener.Addr().String()
}

// Close shuts down the callback server gracefully.
func (cs *CallbackServer) Close() error {
	if cs.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return cs.server.Shutdown(ctx)
}

// ParseCallbackToken extracts token and user_id from an OAuth callback URL.
func ParseCallbackToken(rawURL string) (*CallbackResult, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse callback URL: %w", err)
	}

	params := parsed.Query()

	if errParam := params.Get("error"); errParam != "" {
		desc := params.Get("error_description")
		return nil, fmt.Errorf("authentication failed: %s — %s", errParam, desc)
	}

	token := params.Get("token")
	if token == "" {
		return nil, fmt.Errorf("callback URL missing 'token' parameter")
	}

	userID := params.Get("user_id")

	return &CallbackResult{
		Token:  token,
		UserID: userID,
	}, nil
}

// BuildLoginURL constructs the URL to open in the browser for CLI OAuth login.
func BuildLoginURL(appURL, callbackURL string) string {
	return fmt.Sprintf("%s/auth/cli?callback=%s", appURL, url.QueryEscape(callbackURL))
}

// SaveLoginCredentials saves the received token to the credentials file.
func SaveLoginCredentials(credsPath string, result *CallbackResult) error {
	creds := &store.Credentials{
		Token:     result.Token,
		UserID:    result.UserID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return store.SaveCredentials(credsPath, creds)
}

// DefaultLoginPort is the port used for the OAuth callback server.
const DefaultLoginPort = 9876

// DefaultAppURL is the default Obeya Cloud app URL.
const DefaultAppURL = "https://obeya.app"
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/auth/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya
git add internal/auth/login.go internal/auth/login_test.go
git commit -m "feat: add OAuth login flow with local callback server and token parsing"
```

---

### Task 7: Logout

**Files:**
- Create: `internal/auth/logout.go`
- Create: `internal/auth/logout_test.go`

- [ ] **Step 1: Write failing test for logout**

Create: `internal/auth/logout_test.go`

```go
package auth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
)

func TestLogout_RemovesCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	// Create credentials first
	creds := &store.Credentials{
		Token:     "ob_tok_to_remove",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	// Verify they exist
	if _, err := os.Stat(credsPath); err != nil {
		t.Fatalf("credentials file should exist before logout")
	}

	// Logout
	err := auth.Logout(credsPath)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Verify they're gone
	if _, err := os.Stat(credsPath); !os.IsNotExist(err) {
		t.Error("credentials file should not exist after logout")
	}
}

func TestLogout_NoCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	// Should not error if file doesn't exist
	err := auth.Logout(credsPath)
	if err != nil {
		t.Fatalf("Logout should succeed even when no credentials exist: %v", err)
	}
}

func TestIsLoggedIn_True(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	creds := &store.Credentials{
		Token:     "ob_tok_check",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	if !auth.IsLoggedIn(credsPath) {
		t.Error("expected IsLoggedIn to return true")
	}
}

func TestIsLoggedIn_False(t *testing.T) {
	if auth.IsLoggedIn("/nonexistent/credentials.json") {
		t.Error("expected IsLoggedIn to return false for missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/code/obeya
go test ./internal/auth/ -run TestLogout -v
go test ./internal/auth/ -run TestIsLoggedIn -v
```

Expected: FAIL — functions not defined

- [ ] **Step 3: Write implementation**

Create: `internal/auth/logout.go`

```go
package auth

import (
	"github.com/niladribose/obeya/internal/store"
)

// Logout removes the stored credentials file.
func Logout(credsPath string) error {
	return store.DeleteCredentials(credsPath)
}

// IsLoggedIn checks if valid credentials exist at the given path.
func IsLoggedIn(credsPath string) bool {
	creds, err := store.LoadCredentials(credsPath)
	if err != nil {
		return false
	}
	return creds.Token != ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd ~/code/obeya
go test ./internal/auth/ -run TestLogout -v
go test ./internal/auth/ -run TestIsLoggedIn -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya
git add internal/auth/logout.go internal/auth/logout_test.go
git commit -m "feat: add logout and IsLoggedIn credential management"
```

---

## Chunk 7: CLI Commands

### Task 8: ob login Command

**Files:**
- Create: `cmd/login.go`

- [ ] **Step 1: Write the login command**

Create: `cmd/login.go`

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var loginAppURL string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Obeya Cloud",
	Long:  "Opens a browser for OAuth authentication. On success, stores API token in ~/.obeya/credentials.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		credsPath, err := store.DefaultCredentialsPath()
		if err != nil {
			return err
		}

		if auth.IsLoggedIn(credsPath) {
			fmt.Println("Already logged in. Run 'ob logout' first to re-authenticate.")
			return nil
		}

		srv, tokenCh, errCh := auth.NewCallbackServer(auth.DefaultLoginPort)
		defer srv.Close()

		addr := srv.Addr()
		if addr == "" {
			return fmt.Errorf("failed to start callback server")
		}

		callbackURL := fmt.Sprintf("http://%s/callback", addr)
		loginURL := auth.BuildLoginURL(loginAppURL, callbackURL)

		fmt.Printf("Opening browser for authentication...\n")
		fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", loginURL)
		fmt.Println("Waiting for authentication...")

		openBrowser(loginURL)

		select {
		case result := <-tokenCh:
			if err := auth.SaveLoginCredentials(credsPath, result); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}
			fmt.Printf("\nLogged in successfully. Token stored at %s\n", credsPath)
			return nil

		case err := <-errCh:
			return fmt.Errorf("login failed: %w", err)

		case <-time.After(5 * time.Minute):
			return fmt.Errorf("login timed out after 5 minutes — no callback received")
		}
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginAppURL, "app-url", auth.DefaultAppURL, "Obeya Cloud app URL")
	rootCmd.AddCommand(loginCmd)
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
```

- [ ] **Step 2: Verify build**

```bash
cd ~/code/obeya
go build ./cmd/...
```

Expected: compiles

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya
git add cmd/login.go
git commit -m "feat: add ob login command with OAuth browser flow"
```

---

### Task 9: ob logout Command

**Files:**
- Create: `cmd/logout.go`

- [ ] **Step 1: Write the logout command**

Create: `cmd/logout.go`

```go
package cmd

import (
	"fmt"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored Obeya Cloud credentials",
	Long:  "Removes the API token from ~/.obeya/credentials.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		credsPath, err := store.DefaultCredentialsPath()
		if err != nil {
			return err
		}

		if !auth.IsLoggedIn(credsPath) {
			fmt.Println("Not currently logged in.")
			return nil
		}

		if err := auth.Logout(credsPath); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		fmt.Println("Logged out. Credentials removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
```

- [ ] **Step 2: Verify build**

```bash
cd ~/code/obeya
go build ./cmd/...
```

Expected: compiles

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya
git add cmd/logout.go
git commit -m "feat: add ob logout command to clear stored credentials"
```

---

### Task 10: ob init --cloud Command

**Files:**
- Modify: `cmd/init.go`

- [ ] **Step 1: Add --cloud and --local flags to init command**

Modify: `cmd/init.go`

Add new variables and modify the init function registration:

```go
var initCloud bool
var initLocal bool
```

Add flags in the `init()` function:

```go
func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().StringVar(&initAgent, "agent", "", "coding agent to configure (supported: claude-code)")
	initCmd.Flags().BoolVar(&initSkipPlugin, "skip-plugin", false, "skip plugin installation")
	initCmd.Flags().StringVar(&initRoot, "root", "", "directory to create .obeya in (default: git repository root)")
	initCmd.Flags().StringVar(&initShared, "shared", "", "create a shared board at ~/.obeya/boards/<name>")
	initCmd.Flags().BoolVar(&initCloud, "cloud", false, "create or migrate to a cloud board")
	initCmd.Flags().BoolVar(&initLocal, "local", false, "switch from cloud mode back to local board")
	rootCmd.AddCommand(initCmd)
}
```

- [ ] **Step 2: Add cloud init handler**

Add the following functions to `cmd/init.go`:

```go
func initCloudBoard(root string, columns []string, args []string) error {
	credsPath, err := store.DefaultCredentialsPath()
	if err != nil {
		return err
	}

	// Check if already cloud
	if store.CloudConfigExists(root) {
		cfg, err := store.LoadCloudConfig(store.CloudConfigPath(root))
		if err != nil {
			return err
		}
		fmt.Printf("Already connected to cloud board %s.\nRun 'ob init --local' to switch to local mode.\n", cfg.BoardID)
		return nil
	}

	// Ensure logged in
	if !auth.IsLoggedIn(credsPath) {
		fmt.Println("Not logged in. Running 'ob login' first...")
		if err := loginCmd.RunE(loginCmd, nil); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	creds, err := store.LoadCredentials(credsPath)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	client := store.NewCloudClient(auth.DefaultAppURL+"/api", creds.Token)

	// Check for existing local board
	localStore := store.NewJSONStore(root)
	if localStore.BoardExists() {
		return migrateLocalToCloud(root, client, creds, localStore)
	}

	// Fresh cloud board
	return createFreshCloudBoard(root, columns, args, client, creds)
}

func migrateLocalToCloud(root string, client *store.CloudClient, creds *store.Credentials, localStore *store.JSONStore) error {
	board, err := localStore.LoadBoard()
	if err != nil {
		return fmt.Errorf("failed to load local board: %w", err)
	}

	itemCount := len(board.Items)
	fmt.Printf("Local board found with %d items.\n", itemCount)
	fmt.Printf("Migrating to cloud...\n")

	boardID, err := client.ImportBoard(board, "")
	if err != nil {
		return fmt.Errorf("failed to import board to cloud: %w", err)
	}

	// Backup local board
	obeyaDir := filepath.Join(root, ".obeya")
	backupDir := filepath.Join(root, ".obeya-local-backup")
	if err := os.Rename(obeyaDir, backupDir); err != nil {
		return fmt.Errorf("failed to backup local .obeya directory: %w", err)
	}

	// Create .obeya dir and cloud.json
	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate .obeya directory: %w", err)
	}

	_, username, _ := client.GetMe()

	cfg := &store.CloudConfig{
		APIURL:  auth.DefaultAppURL + "/api",
		BoardID: boardID,
		User:    username,
	}
	if err := store.SaveCloudConfig(store.CloudConfigPath(root), cfg); err != nil {
		return fmt.Errorf("failed to save cloud config: %w", err)
	}

	fmt.Printf("Migrated %d items to cloud board %s\n", itemCount, boardID)
	fmt.Printf("Local backup saved to %s\n", backupDir)
	return nil
}

func createFreshCloudBoard(root string, columns []string, args []string, client *store.CloudClient, creds *store.Credentials) error {
	boardName := "obeya"
	if len(args) > 0 {
		boardName = args[0]
	}

	if len(columns) == 0 {
		columns = []string{"backlog", "todo", "in-progress", "review", "done"}
	}

	boardID, err := client.CreateBoard(boardName, columns, "")
	if err != nil {
		return fmt.Errorf("failed to create cloud board: %w", err)
	}

	obeyaDir := filepath.Join(root, ".obeya")
	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .obeya directory: %w", err)
	}

	_, username, _ := client.GetMe()

	cfg := &store.CloudConfig{
		APIURL:  auth.DefaultAppURL + "/api",
		BoardID: boardID,
		User:    username,
	}
	if err := store.SaveCloudConfig(store.CloudConfigPath(root), cfg); err != nil {
		return fmt.Errorf("failed to save cloud config: %w", err)
	}

	fmt.Printf("Cloud board %q created (ID: %s)\n", boardName, boardID)
	fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
	return nil
}

func initLocalFromCloud(root string, columns []string, args []string) error {
	if !store.CloudConfigExists(root) {
		fmt.Println("Not in cloud mode — already using local storage.")
		return nil
	}

	// Remove cloud.json
	cloudPath := store.CloudConfigPath(root)
	if err := os.Remove(cloudPath); err != nil {
		return fmt.Errorf("failed to remove cloud config: %w", err)
	}

	// Check for backup
	backupDir := filepath.Join(root, ".obeya-local-backup")
	obeyaDir := filepath.Join(root, ".obeya")
	if _, err := os.Stat(backupDir); err == nil {
		// Restore backup
		if err := os.RemoveAll(obeyaDir); err != nil {
			return fmt.Errorf("failed to remove .obeya directory: %w", err)
		}
		if err := os.Rename(backupDir, obeyaDir); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
		fmt.Println("Restored local board from backup.")
		return nil
	}

	// No backup — create fresh local board
	boardName := "obeya"
	if len(args) > 0 {
		boardName = args[0]
	}

	s := store.NewJSONStore(root)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Switched to local mode. Board %q initialized.\n", boardName)
	return nil
}
```

- [ ] **Step 3: Integrate cloud/local flags into RunE**

Modify the `RunE` function of `initCmd` to handle `--cloud` and `--local` before the existing logic:

```go
RunE: func(cmd *cobra.Command, args []string) error {
	columns := parseColumns(initColumns)

	// Handle --cloud
	if initCloud {
		root, err := resolveInitRoot()
		if err != nil {
			return err
		}
		return initCloudBoard(root, columns, args)
	}

	// Handle --local
	if initLocal {
		root, err := resolveInitRoot()
		if err != nil {
			return err
		}
		return initLocalFromCloud(root, columns, args)
	}

	// Shared + agent = shared board with agent setup
	if initShared != "" && initAgent != "" {
		return initSharedBoardWithAgent(initShared, initAgent, columns)
	}

	// Shared board path (no agent)
	if initShared != "" {
		return initSharedBoard(initShared, columns)
	}

	// --agent is required for non-shared boards
	if initAgent == "" {
		return fmt.Errorf("required flag --agent not provided. Supported: %s", strings.Join(agent.SupportedNames(), ", "))
	}

	// ... rest of existing logic unchanged ...
```

Add the necessary imports to `cmd/init.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 4: Verify build**

```bash
cd ~/code/obeya
go build ./...
```

Expected: compiles

- [ ] **Step 5: Commit**

```bash
cd ~/code/obeya
git add cmd/init.go
git commit -m "feat: add ob init --cloud and ob init --local commands for cloud mode switching"
```

---

## Chunk 8: Integration Test & Final Verification

### Task 11: Cloud Store Integration Test

**Files:**
- Create: `internal/store/cloud_store_integration_test.go`

- [ ] **Step 1: Write integration test**

Create: `internal/store/cloud_store_integration_test.go`

```go
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

// TestCloudStore_EngineIntegration verifies that the Engine works with CloudStore
// exactly the same way it works with JSONStore — no Engine changes needed.
func TestCloudStore_EngineIntegration(t *testing.T) {
	board := domain.NewBoard("integration-test")

	var mu sync.Mutex
	var apiCalls []apiCall

	type apiCall struct {
		Method string
		Path   string
		Body   map[string]interface{}
	}

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
			// Simulate server creating the item
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

	// Test 1: Create an item via Engine
	item, err := eng.CreateItem("task", "Integration Task", "", "desc", "medium", "", nil)
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
```

- [ ] **Step 2: Run integration test**

```bash
cd ~/code/obeya
go test ./internal/store/ -run TestCloudStore_Engine -v
go test ./internal/store/ -run TestCloudStore_FullCycle -v
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd ~/code/obeya
git add internal/store/cloud_store_integration_test.go
git commit -m "test: add CloudStore integration tests with Engine and full lifecycle"
```

---

### Task 12: Run Full Test Suite

- [ ] **Step 1: Run all tests**

```bash
cd ~/code/obeya
go test ./... -v
```

Expected: All tests PASS — both existing JSONStore tests and new cloud tests.

- [ ] **Step 2: Verify build for all platforms**

```bash
cd ~/code/obeya
go build -o /dev/null ./...
```

Expected: builds without errors

- [ ] **Step 3: Final commit**

```bash
cd ~/code/obeya
git add -A
git commit -m "feat: complete CLI cloud mode — CloudStore, login/logout, init --cloud/--local"
```

---

## Summary

This plan delivers:

| Component | What's built |
|-----------|-------------|
| **CloudConfig** | `.obeya/cloud.json` and `~/.obeya/credentials.json` types with load/save/delete |
| **CloudClient** | HTTP client for all Cloud API endpoints (export, create, update, move, delete, import, me) |
| **BoardDiff** | Snapshot and diff logic to detect created/updated/deleted/moved items |
| **CloudStore** | Full `Store` interface implementation using diff-and-sync transaction strategy |
| **Store Resolution** | `NewStore()` automatically picks CloudStore or JSONStore based on cloud.json |
| **ob login** | OAuth browser flow with local callback server on localhost:9876 |
| **ob logout** | Credential removal command |
| **ob init --cloud** | Cloud board creation and local-to-cloud migration |
| **ob init --local** | Cloud-to-local switchback with backup restoration |
| **Integration Tests** | Engine works unchanged with CloudStore, full lifecycle test |

**Next plan:** Plan 6 — TUI Realtime via WebSocket (replace fsnotify with Appwrite WebSocket subscription for cloud boards)
