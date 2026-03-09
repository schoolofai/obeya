# Obeya CLI Kanban Board — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI Kanban board manager (`ob`) that serves both humans (TUI) and AI agents (CLI), with a storage abstraction enabling future cloud (Pro) edition.

**Architecture:** Layered — thin CLI/TUI on top of a domain engine, backed by a storage interface. Lite edition uses local JSON. Concurrency via file locking + optimistic versioning. Dual ID system (hash + display number).

**Tech Stack:** Go 1.22+, Cobra (CLI), Bubble Tea (TUI), gofrs/flock (file locking)

**Design Doc:** `docs/plans/2026-03-09-obeya-kanban-design.md`

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/niladribose/obeya`
Expected: `go.mod` created

**Step 2: Install dependencies**

Run: `go get github.com/spf13/cobra@latest && go get github.com/charmbracelet/bubbletea@latest && go get github.com/charmbracelet/lipgloss@latest && go get github.com/gofrs/flock@latest`
Expected: Dependencies added to go.mod

**Step 3: Create root command**

Create `cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ob",
	Short: "Obeya — CLI Kanban board for humans and AI agents",
	Long:  "A CLI-based Kanban board manager that serves both humans (via TUI) and AI agents (via CLI commands).",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Create main.go**

Create `main.go`:

```go
package main

import "github.com/niladribose/obeya/cmd"

func main() {
	cmd.Execute()
}
```

**Step 5: Verify it builds and runs**

Run: `go build -o ob . && ./ob --help`
Expected: Help output showing "Obeya — CLI Kanban board for humans and AI agents"

**Step 6: Commit**

```bash
git add go.mod go.sum main.go cmd/root.go
git commit -m "feat: scaffold project with Cobra root command"
```

---

## Task 2: Domain Types

**Files:**
- Create: `internal/domain/types.go`
- Create: `internal/domain/types_test.go`

**Step 1: Write tests for domain types**

Create `internal/domain/types_test.go`:

```go
package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestItemTypeValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"epic", true},
		{"story", true},
		{"task", true},
		{"bug", false},
		{"", false},
	}
	for _, tt := range tests {
		err := domain.ItemType(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("ItemType(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ItemType(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestPriorityValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"critical", true},
		{"urgent", false},
		{"", false},
	}
	for _, tt := range tests {
		err := domain.Priority(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("Priority(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("Priority(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestIdentityTypeValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"human", true},
		{"agent", true},
		{"bot", false},
	}
	for _, tt := range tests {
		err := domain.IdentityType(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("IdentityType(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("IdentityType(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestNewBoardDefaults(t *testing.T) {
	board := domain.NewBoard("test-board")
	if board.Name != "test-board" {
		t.Errorf("expected name 'test-board', got %q", board.Name)
	}
	if len(board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(board.Columns))
	}
	expectedCols := []string{"backlog", "todo", "in-progress", "review", "done"}
	for i, col := range board.Columns {
		if col.Name != expectedCols[i] {
			t.Errorf("column %d: expected %q, got %q", i, expectedCols[i], col.Name)
		}
	}
	if board.Version != 1 {
		t.Errorf("expected version 1, got %d", board.Version)
	}
	if board.NextDisplay != 1 {
		t.Errorf("expected NextDisplay 1, got %d", board.NextDisplay)
	}
	if board.AgentRole != "admin" {
		t.Errorf("expected AgentRole 'admin', got %q", board.AgentRole)
	}
}

func TestNewBoardCustomColumns(t *testing.T) {
	board := domain.NewBoardWithColumns("custom", []string{"todo", "doing", "done"})
	if len(board.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(board.Columns))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Implement domain types**

Create `internal/domain/types.go`:

```go
package domain

import (
	"fmt"
	"time"
)

type ItemType string

const (
	ItemTypeEpic  ItemType = "epic"
	ItemTypeStory ItemType = "story"
	ItemTypeTask  ItemType = "task"
)

func (t ItemType) Validate() error {
	switch t {
	case ItemTypeEpic, ItemTypeStory, ItemTypeTask:
		return nil
	default:
		return fmt.Errorf("invalid item type: %q (must be epic, story, or task)", string(t))
	}
}

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

func (p Priority) Validate() error {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return nil
	default:
		return fmt.Errorf("invalid priority: %q (must be low, medium, high, or critical)", string(p))
	}
}

type IdentityType string

const (
	IdentityHuman IdentityType = "human"
	IdentityAgent IdentityType = "agent"
)

func (t IdentityType) Validate() error {
	switch t {
	case IdentityHuman, IdentityAgent:
		return nil
	default:
		return fmt.Errorf("invalid identity type: %q (must be human or agent)", string(t))
	}
}

type Column struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
}

type ChangeRecord struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

type Identity struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Type     IdentityType `json:"type"`
	Provider string       `json:"provider"`
}

type Item struct {
	ID          string         `json:"id"`
	DisplayNum  int            `json:"display_num"`
	Type        ItemType       `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Priority    Priority       `json:"priority"`
	Assignee    string         `json:"assignee,omitempty"`
	ParentID    string         `json:"parent_id,omitempty"`
	BlockedBy   []string       `json:"blocked_by,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	History     []ChangeRecord `json:"history,omitempty"`
}

type Board struct {
	Version     int                  `json:"version"`
	Name        string               `json:"name"`
	Columns     []Column             `json:"columns"`
	Items       map[string]*Item     `json:"items"`
	DisplayMap  map[int]string       `json:"display_map"`
	NextDisplay int                  `json:"next_display"`
	Users       map[string]*Identity `json:"users"`
	AgentRole   string               `json:"agent_role"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

var defaultColumns = []string{"backlog", "todo", "in-progress", "review", "done"}

func NewBoard(name string) *Board {
	return NewBoardWithColumns(name, defaultColumns)
}

func NewBoardWithColumns(name string, columnNames []string) *Board {
	cols := make([]Column, len(columnNames))
	for i, n := range columnNames {
		cols[i] = Column{Name: n}
	}
	now := time.Now()
	return &Board{
		Version:     1,
		Name:        name,
		Columns:     cols,
		Items:       make(map[string]*Item),
		DisplayMap:  make(map[int]string),
		NextDisplay: 1,
		Users:       make(map[string]*Identity),
		AgentRole:   "admin",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// HasColumn checks if a column name exists on the board.
func (b *Board) HasColumn(name string) bool {
	for _, c := range b.Columns {
		if c.Name == name {
			return true
		}
	}
	return false
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain types — Item, Board, Identity, enums with validation"
```

---

## Task 3: ID Generation

**Files:**
- Create: `internal/domain/id.go`
- Create: `internal/domain/id_test.go`

**Step 1: Write tests for ID generation and resolution**

Create `internal/domain/id_test.go`:

```go
package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestGenerateID(t *testing.T) {
	id := domain.GenerateID()
	if len(id) != 8 {
		t.Errorf("expected ID length 8, got %d: %q", len(id), id)
	}
	// IDs should be unique
	id2 := domain.GenerateID()
	if id == id2 {
		t.Errorf("two generated IDs should not be equal: %q", id)
	}
}

func TestResolveID_ByHash(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("abc12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_ByHashPrefix(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("abc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_ByDisplayNum(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_NotFound(t *testing.T) {
	board := domain.NewBoard("test")
	_, err := board.ResolveID("xyz")
	if err == nil {
		t.Error("expected error for unknown ID, got nil")
	}
}

func TestResolveID_AmbiguousPrefix(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.Items["abc12999"] = &domain.Item{ID: "abc12999", DisplayNum: 2}
	board.DisplayMap[1] = "abc12345"
	board.DisplayMap[2] = "abc12999"

	_, err := board.ResolveID("abc1")
	if err == nil {
		t.Error("expected ambiguous error, got nil")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/ -v -run TestGenerateID`
Expected: FAIL — function not defined

**Step 3: Implement ID generation and resolution**

Create `internal/domain/id.go`:

```go
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// GenerateID creates an 8-character hex ID from 4 random bytes.
func GenerateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random ID: %v", err))
	}
	return hex.EncodeToString(b)
}

// ResolveID resolves a user-provided reference (display number or hash prefix) to a canonical ID.
func (b *Board) ResolveID(ref string) (string, error) {
	// Try exact match first
	if _, ok := b.Items[ref]; ok {
		return ref, nil
	}

	// Try as display number
	if num, err := strconv.Atoi(ref); err == nil {
		if id, ok := b.DisplayMap[num]; ok {
			return id, nil
		}
		return "", fmt.Errorf("no item with display number %d", num)
	}

	// Try as hash prefix
	var matches []string
	for id := range b.Items {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no item found matching %q", ref)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous reference %q matches %d items: %s", ref, len(matches), strings.Join(matches, ", "))
	}
}

// ResolveUserID resolves a user reference (display-style or hash) to a user ID.
func (b *Board) ResolveUserID(ref string) (string, error) {
	// Exact match
	if _, ok := b.Users[ref]; ok {
		return ref, nil
	}

	// Try by name (case-insensitive)
	for id, u := range b.Users {
		if strings.EqualFold(u.Name, ref) {
			return id, nil
		}
	}

	// Try as hash prefix
	var matches []string
	for id := range b.Users {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no user found matching %q", ref)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous user reference %q matches %d users", ref, len(matches))
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/domain/id.go internal/domain/id_test.go
git commit -m "feat: add ID generation and dual ID resolution (hash + display number)"
```

---

## Task 4: Storage Interface + JSON Store

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/json_store.go`
- Create: `internal/store/json_store_test.go`

**Step 1: Write the storage interface**

Create `internal/store/store.go`:

```go
package store

import "github.com/niladribose/obeya/internal/domain"

// Store abstracts board persistence. Lite uses JSON files; Pro will use a cloud API.
type Store interface {
	// Transaction performs an atomic read-modify-write on the board.
	// The provided function receives the current board state and may mutate it.
	// Changes are persisted atomically when the function returns nil.
	Transaction(fn func(board *domain.Board) error) error

	// LoadBoard returns a read-only snapshot of the board (no lock).
	LoadBoard() (*domain.Board, error)

	// InitBoard creates a new board with the given config.
	InitBoard(name string, columns []string) error

	// BoardExists checks if a board has been initialized in the current directory.
	BoardExists() bool
}
```

**Step 2: Write tests for JSON store**

Create `internal/store/json_store_test.go`:

```go
package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func setupTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func TestJSONStore_InitBoard(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)

	if s.BoardExists() {
		t.Error("board should not exist before init")
	}

	err := s.InitBoard("test-board", nil)
	if err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}

	if !s.BoardExists() {
		t.Error("board should exist after init")
	}

	// Verify file exists
	boardFile := filepath.Join(dir, ".obeya", "board.json")
	if _, err := os.Stat(boardFile); os.IsNotExist(err) {
		t.Error("board.json file should exist")
	}
}

func TestJSONStore_InitBoard_AlreadyExists(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)

	_ = s.InitBoard("test", nil)
	err := s.InitBoard("test", nil)
	if err == nil {
		t.Error("expected error when board already exists")
	}
}

func TestJSONStore_InitBoard_CustomColumns(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)

	err := s.InitBoard("test", []string{"todo", "doing", "done"})
	if err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}

	board, err := s.LoadBoard()
	if err != nil {
		t.Fatalf("LoadBoard failed: %v", err)
	}

	if len(board.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(board.Columns))
	}
}

func TestJSONStore_LoadBoard(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)

	_ = s.InitBoard("test-board", nil)

	board, err := s.LoadBoard()
	if err != nil {
		t.Fatalf("LoadBoard failed: %v", err)
	}

	if board.Name != "test-board" {
		t.Errorf("expected name 'test-board', got %q", board.Name)
	}
	if board.Version != 1 {
		t.Errorf("expected version 1, got %d", board.Version)
	}
}

func TestJSONStore_LoadBoard_NotInitialized(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)

	_, err := s.LoadBoard()
	if err == nil {
		t.Error("expected error when board not initialized")
	}
}

func TestJSONStore_Transaction(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)
	_ = s.InitBoard("test", nil)

	err := s.Transaction(func(board *domain.Board) error {
		board.Name = "modified"
		return nil
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	board, _ := s.LoadBoard()
	if board.Name != "modified" {
		t.Errorf("expected name 'modified', got %q", board.Name)
	}
	if board.Version != 2 {
		t.Errorf("expected version 2 after transaction, got %d", board.Version)
	}
}

func TestJSONStore_Transaction_ErrorRollback(t *testing.T) {
	dir := setupTempDir(t)
	s := store.NewJSONStore(dir)
	_ = s.InitBoard("test", nil)

	err := s.Transaction(func(board *domain.Board) error {
		board.Name = "should-not-persist"
		return fmt.Errorf("simulated error")
	})
	if err == nil {
		t.Error("expected error from transaction")
	}

	board, _ := s.LoadBoard()
	if board.Name != "test" {
		t.Errorf("board should not have been modified, got name %q", board.Name)
	}
}
```

Note: Add `"fmt"` and the domain import to the test file imports:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	domain "github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)
```

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/store/ -v`
Expected: FAIL — package doesn't exist yet

**Step 4: Implement JSON store**

Create `internal/store/json_store.go`:

```go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/niladribose/obeya/internal/domain"
)

type JSONStore struct {
	rootDir   string
	obeyaDir  string
	boardFile string
	lockFile  string
}

func NewJSONStore(rootDir string) *JSONStore {
	obeyaDir := filepath.Join(rootDir, ".obeya")
	return &JSONStore{
		rootDir:   rootDir,
		obeyaDir:  obeyaDir,
		boardFile: filepath.Join(obeyaDir, "board.json"),
		lockFile:  filepath.Join(obeyaDir, "board.lock"),
	}
}

func (s *JSONStore) BoardExists() bool {
	_, err := os.Stat(s.boardFile)
	return err == nil
}

func (s *JSONStore) InitBoard(name string, columns []string) error {
	if s.BoardExists() {
		return fmt.Errorf("board already initialized in %s", s.obeyaDir)
	}

	if err := os.MkdirAll(s.obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .obeya directory: %w", err)
	}

	var board *domain.Board
	if len(columns) > 0 {
		board = domain.NewBoardWithColumns(name, columns)
	} else {
		board = domain.NewBoard(name)
	}

	return s.writeBoard(board)
}

func (s *JSONStore) LoadBoard() (*domain.Board, error) {
	if !s.BoardExists() {
		return nil, fmt.Errorf("no board found — run 'ob init' first")
	}

	data, err := os.ReadFile(s.boardFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read board file: %w", err)
	}

	var board domain.Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to parse board file: %w", err)
	}

	return &board, nil
}

func (s *JSONStore) Transaction(fn func(board *domain.Board) error) error {
	fileLock := flock.New(s.lockFile)

	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire board lock: %w", err)
	}
	defer fileLock.Unlock()

	board, err := s.LoadBoard()
	if err != nil {
		return err
	}

	savedVersion := board.Version

	if err := fn(board); err != nil {
		return err
	}

	if board.Version != savedVersion {
		return fmt.Errorf("concurrent modification detected: board version changed during transaction")
	}

	board.Version++
	board.UpdatedAt = time.Now()

	return s.writeBoard(board)
}

func (s *JSONStore) writeBoard(board *domain.Board) error {
	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}

	tmpFile := s.boardFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp board file: %w", err)
	}

	if err := os.Rename(tmpFile, s.boardFile); err != nil {
		return fmt.Errorf("failed to atomically replace board file: %w", err)
	}

	return nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add internal/store/
git commit -m "feat: add storage interface and JSON store with file locking"
```

---

## Task 5: Engine — Core Business Logic

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

**Step 1: Write tests for engine operations**

Create `internal/engine/engine_test.go`:

```go
package engine_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func setupEngine(t *testing.T) *engine.Engine {
	t.Helper()
	dir := t.TempDir()
	s := store.NewJSONStore(dir)
	_ = s.InitBoard("test", nil)
	return engine.New(s)
}

func TestCreateItem(t *testing.T) {
	eng := setupEngine(t)

	item, err := eng.CreateItem("epic", "Build auth system", "", "", "medium", "", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if item.Type != "epic" {
		t.Errorf("expected type 'epic', got %q", item.Type)
	}
	if item.Title != "Build auth system" {
		t.Errorf("expected title 'Build auth system', got %q", item.Title)
	}
	if item.Status != "backlog" {
		t.Errorf("expected default status 'backlog', got %q", item.Status)
	}
	if item.DisplayNum != 1 {
		t.Errorf("expected display num 1, got %d", item.DisplayNum)
	}
}

func TestCreateItem_WithParent(t *testing.T) {
	eng := setupEngine(t)

	epic, _ := eng.CreateItem("epic", "Epic", "", "", "medium", "", nil)
	story, err := eng.CreateItem("story", "Story", epic.ID, "", "medium", "", nil)
	if err != nil {
		t.Fatalf("CreateItem with parent failed: %v", err)
	}
	if story.ParentID != epic.ID {
		t.Errorf("expected parent %q, got %q", epic.ID, story.ParentID)
	}
}

func TestCreateItem_InvalidParent(t *testing.T) {
	eng := setupEngine(t)

	_, err := eng.CreateItem("story", "Story", "nonexistent", "", "medium", "", nil)
	if err == nil {
		t.Error("expected error for invalid parent")
	}
}

func TestCreateItem_InvalidType(t *testing.T) {
	eng := setupEngine(t)

	_, err := eng.CreateItem("bug", "Bug report", "", "", "medium", "", nil)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestMoveItem(t *testing.T) {
	eng := setupEngine(t)

	item, _ := eng.CreateItem("task", "Fix bug", "", "", "medium", "", nil)

	err := eng.MoveItem(item.ID, "in-progress", "", "")
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	updated, _ := eng.GetItem(item.ID)
	if updated.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", updated.Status)
	}
}

func TestMoveItem_InvalidStatus(t *testing.T) {
	eng := setupEngine(t)

	item, _ := eng.CreateItem("task", "Fix bug", "", "", "medium", "", nil)

	err := eng.MoveItem(item.ID, "invalid-column", "", "")
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestBlockItem(t *testing.T) {
	eng := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "", nil)

	err := eng.BlockItem(task2.ID, task1.ID, "", "")
	if err != nil {
		t.Fatalf("BlockItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task2.ID)
	if len(updated.BlockedBy) != 1 || updated.BlockedBy[0] != task1.ID {
		t.Errorf("expected BlockedBy [%s], got %v", task1.ID, updated.BlockedBy)
	}
}

func TestBlockItem_SelfBlock(t *testing.T) {
	eng := setupEngine(t)

	task, _ := eng.CreateItem("task", "Task", "", "", "medium", "", nil)

	err := eng.BlockItem(task.ID, task.ID, "", "")
	if err == nil {
		t.Error("expected error for self-blocking")
	}
}

func TestUnblockItem(t *testing.T) {
	eng := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "", nil)

	_ = eng.BlockItem(task2.ID, task1.ID, "", "")
	err := eng.UnblockItem(task2.ID, task1.ID, "", "")
	if err != nil {
		t.Fatalf("UnblockItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task2.ID)
	if len(updated.BlockedBy) != 0 {
		t.Errorf("expected empty BlockedBy, got %v", updated.BlockedBy)
	}
}

func TestAssignItem(t *testing.T) {
	eng := setupEngine(t)

	task, _ := eng.CreateItem("task", "Task", "", "", "medium", "", nil)
	_ = eng.AddUser("Dev", "human", "local")

	board, _ := eng.ListBoard()
	var userID string
	for id := range board.Users {
		userID = id
	}

	err := eng.AssignItem(task.ID, userID, "", "")
	if err != nil {
		t.Fatalf("AssignItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task.ID)
	if updated.Assignee != userID {
		t.Errorf("expected assignee %q, got %q", userID, updated.Assignee)
	}
}

func TestDeleteItem_WithChildren(t *testing.T) {
	eng := setupEngine(t)

	epic, _ := eng.CreateItem("epic", "Epic", "", "", "medium", "", nil)
	_, _ = eng.CreateItem("story", "Story", epic.ID, "", "medium", "", nil)

	err := eng.DeleteItem(epic.ID, "", "")
	if err == nil {
		t.Error("expected error when deleting item with children")
	}
}

func TestAddUser(t *testing.T) {
	eng := setupEngine(t)

	err := eng.AddUser("Claude Agent", "agent", "claude-code")
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	board, _ := eng.ListBoard()
	if len(board.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(board.Users))
	}

	for _, u := range board.Users {
		if u.Name != "Claude Agent" {
			t.Errorf("expected name 'Claude Agent', got %q", u.Name)
		}
		if u.Provider != "claude-code" {
			t.Errorf("expected provider 'claude-code', got %q", u.Provider)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Implement engine**

Create `internal/engine/engine.go`:

```go
package engine

import (
	"fmt"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

type Engine struct {
	store store.Store
}

func New(s store.Store) *Engine {
	return &Engine{store: s}
}

func (e *Engine) CreateItem(itemType, title, parentRef, description, priority, assignee string, tags []string) (*domain.Item, error) {
	if err := domain.ItemType(itemType).Validate(); err != nil {
		return nil, err
	}
	if priority == "" {
		priority = "medium"
	}
	if err := domain.Priority(priority).Validate(); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	var created *domain.Item
	err := e.store.Transaction(func(board *domain.Board) error {
		// Resolve parent if provided
		var parentID string
		if parentRef != "" {
			resolved, err := board.ResolveID(parentRef)
			if err != nil {
				return fmt.Errorf("invalid parent: %w", err)
			}
			parentID = resolved
		}

		// Default status = first column
		status := board.Columns[0].Name

		item := &domain.Item{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Type:        domain.ItemType(itemType),
			Title:       title,
			Description: description,
			Status:      status,
			Priority:    domain.Priority(priority),
			Assignee:    assignee,
			ParentID:    parentID,
			Tags:        tags,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			History: []domain.ChangeRecord{
				{Action: "created", Detail: fmt.Sprintf("created %s: %s", itemType, title), Timestamp: time.Now()},
			},
		}

		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++

		created = item
		return nil
	})

	return created, err
}

func (e *Engine) MoveItem(ref, status, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		if !board.HasColumn(status) {
			return fmt.Errorf("invalid status %q — available columns: %s", status, columnNames(board))
		}

		item := board.Items[id]
		oldStatus := item.Status
		item.Status = status
		item.UpdatedAt = time.Now()
		item.History = append(item.History, domain.ChangeRecord{
			UserID:    userID,
			SessionID: sessionID,
			Action:    "moved",
			Detail:    fmt.Sprintf("status: %s -> %s", oldStatus, status),
			Timestamp: time.Now(),
		})

		return nil
	})
}

func (e *Engine) AssignItem(ref, userRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		assigneeID, err := board.ResolveUserID(userRef)
		if err != nil {
			return err
		}

		item := board.Items[id]
		item.Assignee = assigneeID
		item.UpdatedAt = time.Now()
		item.History = append(item.History, domain.ChangeRecord{
			UserID:    userID,
			SessionID: sessionID,
			Action:    "assigned",
			Detail:    fmt.Sprintf("assigned to %s", assigneeID),
			Timestamp: time.Now(),
		})

		return nil
	})
}

func (e *Engine) BlockItem(ref, blockerRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		blockerID, err := board.ResolveID(blockerRef)
		if err != nil {
			return fmt.Errorf("invalid blocker: %w", err)
		}

		if id == blockerID {
			return fmt.Errorf("an item cannot block itself")
		}

		item := board.Items[id]
		for _, b := range item.BlockedBy {
			if b == blockerID {
				return fmt.Errorf("item #%d is already blocked by #%d", item.DisplayNum, board.Items[blockerID].DisplayNum)
			}
		}

		item.BlockedBy = append(item.BlockedBy, blockerID)
		item.UpdatedAt = time.Now()
		item.History = append(item.History, domain.ChangeRecord{
			UserID:    userID,
			SessionID: sessionID,
			Action:    "blocked",
			Detail:    fmt.Sprintf("blocked by %s", blockerID),
			Timestamp: time.Now(),
		})

		return nil
	})
}

func (e *Engine) UnblockItem(ref, blockerRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		blockerID, err := board.ResolveID(blockerRef)
		if err != nil {
			return fmt.Errorf("invalid blocker: %w", err)
		}

		item := board.Items[id]
		filtered := make([]string, 0, len(item.BlockedBy))
		found := false
		for _, b := range item.BlockedBy {
			if b == blockerID {
				found = true
				continue
			}
			filtered = append(filtered, b)
		}

		if !found {
			return fmt.Errorf("item is not blocked by %s", blockerID)
		}

		item.BlockedBy = filtered
		item.UpdatedAt = time.Now()
		item.History = append(item.History, domain.ChangeRecord{
			UserID:    userID,
			SessionID: sessionID,
			Action:    "unblocked",
			Detail:    fmt.Sprintf("unblocked from %s", blockerID),
			Timestamp: time.Now(),
		})

		return nil
	})
}

func (e *Engine) EditItem(ref, title, description, priority, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		item := board.Items[id]
		changes := []string{}

		if title != "" {
			item.Title = title
			changes = append(changes, fmt.Sprintf("title changed to %q", title))
		}
		if description != "" {
			item.Description = description
			changes = append(changes, "description updated")
		}
		if priority != "" {
			if err := domain.Priority(priority).Validate(); err != nil {
				return err
			}
			item.Priority = domain.Priority(priority)
			changes = append(changes, fmt.Sprintf("priority changed to %s", priority))
		}

		if len(changes) == 0 {
			return fmt.Errorf("no changes specified")
		}

		item.UpdatedAt = time.Now()
		for _, change := range changes {
			item.History = append(item.History, domain.ChangeRecord{
				UserID:    userID,
				SessionID: sessionID,
				Action:    "edited",
				Detail:    change,
				Timestamp: time.Now(),
			})
		}

		return nil
	})
}

func (e *Engine) DeleteItem(ref, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		// Check for children
		for _, item := range board.Items {
			if item.ParentID == id {
				return fmt.Errorf("cannot delete item #%d: it has children — delete children first", board.Items[id].DisplayNum)
			}
		}

		item := board.Items[id]
		delete(board.Items, id)
		delete(board.DisplayMap, item.DisplayNum)

		return nil
	})
}

func (e *Engine) GetItem(ref string) (*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	id, err := board.ResolveID(ref)
	if err != nil {
		return nil, err
	}

	return board.Items[id], nil
}

func (e *Engine) ListBoard() (*domain.Board, error) {
	return e.store.LoadBoard()
}

func (e *Engine) AddUser(name, identityType, provider string) error {
	if err := domain.IdentityType(identityType).Validate(); err != nil {
		return err
	}

	return e.store.Transaction(func(board *domain.Board) error {
		identity := &domain.Identity{
			ID:       domain.GenerateID(),
			Name:     name,
			Type:     domain.IdentityType(identityType),
			Provider: provider,
		}
		board.Users[identity.ID] = identity
		return nil
	})
}

func (e *Engine) RemoveUser(ref string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveUserID(ref)
		if err != nil {
			return err
		}
		delete(board.Users, id)
		return nil
	})
}

func columnNames(board *domain.Board) string {
	names := ""
	for i, c := range board.Columns {
		if i > 0 {
			names += ", "
		}
		names += c.Name
	}
	return names
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: add engine with create, move, assign, block, edit, delete operations"
```

---

## Task 6: CLI Commands — Init

**Files:**
- Create: `cmd/init.go`
- Modify: `cmd/root.go` — add persistent flags and store setup

**Step 1: Add store initialization to root command**

Modify `cmd/root.go` to add a helper that creates the store from the current working directory:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagAs      string
	flagSession string
	flagFormat  string
)

var rootCmd = &cobra.Command{
	Use:   "ob",
	Short: "Obeya — CLI Kanban board for humans and AI agents",
	Long:  "A CLI-based Kanban board manager that serves both humans (via TUI) and AI agents (via CLI commands).",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAs, "as", "", "user ID for this operation (or set OB_USER)")
	rootCmd.PersistentFlags().StringVar(&flagSession, "session", "", "session ID for audit trail (or set OB_SESSION)")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "text", "output format: text or json")
}

func getStore() store.Store {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return store.NewJSONStore(dir)
}

func getEngine() *engine.Engine {
	return engine.New(getStore())
}

func getUserID() string {
	if flagAs != "" {
		return flagAs
	}
	return os.Getenv("OB_USER")
}

func getSessionID() string {
	if flagSession != "" {
		return flagSession
	}
	return os.Getenv("OB_SESSION")
}
```

**Step 2: Create init command**

Create `cmd/init.go`:

```go
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var initColumns string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Obeya board in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		var columns []string
		if initColumns != "" {
			columns = strings.Split(initColumns, ",")
			for i := range columns {
				columns[i] = strings.TrimSpace(columns[i])
			}
		}

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		if err := s.InitBoard(boardName, columns); err != nil {
			return err
		}

		fmt.Printf("Board %q initialized in .obeya/\n", boardName)
		if len(columns) > 0 {
			fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
		} else {
			fmt.Println("Columns: backlog, todo, in-progress, review, done")
		}
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	rootCmd.AddCommand(initCmd)
}
```

**Step 3: Build and test manually**

Run: `go build -o ob . && cd /tmp && /Users/niladribose/code/obeya/ob init test-board && cat .obeya/board.json && rm -rf .obeya && cd -`
Expected: Board initialized, JSON file created with correct structure

**Step 4: Commit**

```bash
git add cmd/root.go cmd/init.go
git commit -m "feat: add init command with custom column support"
```

---

## Task 7: CLI Commands — Create

**Files:**
- Create: `cmd/create.go`

**Step 1: Implement create command**

Create `cmd/create.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	createParent   string
	createPriority string
	createAssign   string
	createTags     string
	createDesc     string
)

var createCmd = &cobra.Command{
	Use:   "create <type> <title>",
	Short: "Create an epic, story, or task",
	Long:  "Create a new item on the board. Types: epic, story, task.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		itemType := args[0]
		title := args[1]

		var tags []string
		if createTags != "" {
			tags = strings.Split(createTags, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		eng := getEngine()
		item, err := eng.CreateItem(itemType, title, createParent, createDesc, createPriority, createAssign, tags)
		if err != nil {
			return err
		}

		if flagFormat == "json" {
			data, _ := json.MarshalIndent(item, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Created %s #%d [%s]: %s\n", item.Type, item.DisplayNum, item.ID[:6], item.Title)
		if item.ParentID != "" {
			fmt.Printf("  Parent: %s\n", item.ParentID[:6])
		}
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createParent, "parent", "p", "", "parent item ID or display number")
	createCmd.Flags().StringVar(&createPriority, "priority", "medium", "priority: low, medium, high, critical")
	createCmd.Flags().StringVar(&createAssign, "assign", "", "assign to user ID")
	createCmd.Flags().StringVar(&createTags, "tag", "", "comma-separated tags")
	createCmd.Flags().StringVarP(&createDesc, "description", "d", "", "item description")
	rootCmd.AddCommand(createCmd)
}
```

**Step 2: Build and test**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add cmd/create.go
git commit -m "feat: add create command for epics, stories, and tasks"
```

---

## Task 8: CLI Commands — Move, Assign, Block, Edit, Delete

**Files:**
- Create: `cmd/move.go`
- Create: `cmd/assign.go`
- Create: `cmd/block.go`
- Create: `cmd/edit.go`
- Create: `cmd/delete.go`

**Step 1: Create move command**

Create `cmd/move.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Move an item to a different column",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.MoveItem(args[0], args[1], getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Moved #%s to %s\n", args[0], args[1])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
}
```

**Step 2: Create assign command**

Create `cmd/assign.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var assignTo string

var assignCmd = &cobra.Command{
	Use:   "assign <id>",
	Short: "Assign an item to a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if assignTo == "" {
			return fmt.Errorf("--to flag is required")
		}
		eng := getEngine()
		if err := eng.AssignItem(args[0], assignTo, getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Assigned #%s to %s\n", args[0], assignTo)
		return nil
	},
}

func init() {
	assignCmd.Flags().StringVar(&assignTo, "to", "", "user ID or name to assign to")
	rootCmd.AddCommand(assignCmd)
}
```

**Step 3: Create block command**

Create `cmd/block.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blockBy string
var unblockBy string

var blockCmd = &cobra.Command{
	Use:   "block <id>",
	Short: "Mark an item as blocked by another item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if blockBy == "" {
			return fmt.Errorf("--by flag is required")
		}
		eng := getEngine()
		if err := eng.BlockItem(args[0], blockBy, getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Marked #%s as blocked by #%s\n", args[0], blockBy)
		return nil
	},
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <id>",
	Short: "Remove a blocker from an item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if unblockBy == "" {
			return fmt.Errorf("--by flag is required")
		}
		eng := getEngine()
		if err := eng.UnblockItem(args[0], unblockBy, getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Removed blocker #%s from #%s\n", unblockBy, args[0])
		return nil
	},
}

func init() {
	blockCmd.Flags().StringVar(&blockBy, "by", "", "ID of the blocking item")
	unblockCmd.Flags().StringVar(&unblockBy, "by", "", "ID of the blocker to remove")
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(unblockCmd)
}
```

**Step 4: Create edit command**

Create `cmd/edit.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	editTitle       string
	editDescription string
	editPriority    string
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an item's title, description, or priority",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.EditItem(args[0], editTitle, editDescription, editPriority, getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Updated #%s\n", args[0])
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "new title")
	editCmd.Flags().StringVarP(&editDescription, "description", "d", "", "new description")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "new priority")
	rootCmd.AddCommand(editCmd)
}
```

**Step 5: Create delete command**

Create `cmd/delete.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an item (fails if it has children)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.DeleteItem(args[0], getUserID(), getSessionID()); err != nil {
			return err
		}
		fmt.Printf("Deleted #%s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
```

**Step 6: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 7: Commit**

```bash
git add cmd/move.go cmd/assign.go cmd/block.go cmd/edit.go cmd/delete.go
git commit -m "feat: add move, assign, block, unblock, edit, and delete commands"
```

---

## Task 9: CLI Commands — List and Show

**Files:**
- Create: `cmd/list.go`
- Create: `cmd/show.go`
- Create: `internal/engine/query.go`

**Step 1: Add query/filter support to engine**

Create `internal/engine/query.go`:

```go
package engine

import "github.com/niladribose/obeya/internal/domain"

type ListFilter struct {
	Status   string
	Assignee string
	Type     string
	Tag      string
	Blocked  bool
	Flat     bool
}

func (e *Engine) ListItems(filter ListFilter) ([]*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	var items []*domain.Item
	for _, item := range board.Items {
		if !matchesFilter(item, filter) {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (e *Engine) GetChildren(parentID string) ([]*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children, nil
}

func matchesFilter(item *domain.Item, f ListFilter) bool {
	if f.Status != "" && item.Status != f.Status {
		return false
	}
	if f.Assignee != "" && item.Assignee != f.Assignee {
		return false
	}
	if f.Type != "" && string(item.Type) != f.Type {
		return false
	}
	if f.Tag != "" {
		found := false
		for _, t := range item.Tags {
			if t == f.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if f.Blocked && len(item.BlockedBy) == 0 {
		return false
	}
	return true
}
```

**Step 2: Create list command**

Create `cmd/list.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/spf13/cobra"
)

var (
	listStatus   string
	listAssignee string
	listType     string
	listTag      string
	listBlocked  bool
	listFlat     bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items on the board",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		filter := engine.ListFilter{
			Status:   listStatus,
			Assignee: listAssignee,
			Type:     listType,
			Tag:      listTag,
			Blocked:  listBlocked,
			Flat:     listFlat,
		}

		items, err := eng.ListItems(filter)
		if err != nil {
			return err
		}

		if flagFormat == "json" {
			data, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if listFlat {
			printFlat(items)
		} else {
			printTree(items)
		}

		return nil
	},
}

func printFlat(items []*domain.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayNum < items[j].DisplayNum
	})
	for _, item := range items {
		blockedMark := ""
		if len(item.BlockedBy) > 0 {
			blockedMark = " [BLOCKED]"
		}
		fmt.Printf("#%-4d %-8s %-6s %-14s %s%s\n",
			item.DisplayNum, item.Type, item.Priority, item.Status, item.Title, blockedMark)
	}
}

func printTree(items []*domain.Item) {
	// Collect root items (no parent)
	var roots []*domain.Item
	childMap := make(map[string][]*domain.Item)

	for _, item := range items {
		if item.ParentID == "" {
			roots = append(roots, item)
		} else {
			childMap[item.ParentID] = append(childMap[item.ParentID], item)
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].DisplayNum < roots[j].DisplayNum
	})

	for _, root := range roots {
		printItem(root, 0)
		printChildren(childMap, root.ID, 1)
	}

	// Also print orphans (items whose parent was filtered out)
	if len(roots) == 0 && len(childMap) == 0 {
		for _, item := range items {
			printItem(item, 0)
		}
	}
}

func printChildren(childMap map[string][]*domain.Item, parentID string, depth int) {
	children := childMap[parentID]
	sort.Slice(children, func(i, j int) bool {
		return children[i].DisplayNum < children[j].DisplayNum
	})
	for _, child := range children {
		printItem(child, depth)
		printChildren(childMap, child.ID, depth+1)
	}
}

func printItem(item *domain.Item, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	blockedMark := ""
	if len(item.BlockedBy) > 0 {
		blockedMark = " [BLOCKED]"
	}
	fmt.Printf("%s#%-4d %-8s %-6s %-14s %s%s\n",
		indent, item.DisplayNum, item.Type, item.Priority, item.Status, item.Title, blockedMark)
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "filter by status/column")
	listCmd.Flags().StringVar(&listAssignee, "assignee", "", "filter by assignee")
	listCmd.Flags().StringVar(&listType, "type", "", "filter by type (epic, story, task)")
	listCmd.Flags().StringVar(&listTag, "tag", "", "filter by tag")
	listCmd.Flags().BoolVar(&listBlocked, "blocked", false, "show only blocked items")
	listCmd.Flags().BoolVar(&listFlat, "flat", false, "flat list without hierarchy")
	rootCmd.AddCommand(listCmd)
}
```

**Step 3: Create show command**

Create `cmd/show.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed information about an item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()

		item, err := eng.GetItem(args[0])
		if err != nil {
			return err
		}

		if flagFormat == "json" {
			data, _ := json.MarshalIndent(item, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("#%d [%s] %s\n", item.DisplayNum, item.ID[:6], item.Title)
		fmt.Printf("  Type:     %s\n", item.Type)
		fmt.Printf("  Status:   %s\n", item.Status)
		fmt.Printf("  Priority: %s\n", item.Priority)

		if item.Description != "" {
			fmt.Printf("  Desc:     %s\n", item.Description)
		}
		if item.Assignee != "" {
			fmt.Printf("  Assignee: %s\n", item.Assignee)
		}
		if item.ParentID != "" {
			fmt.Printf("  Parent:   %s\n", item.ParentID[:6])
		}
		if len(item.Tags) > 0 {
			fmt.Printf("  Tags:     %v\n", item.Tags)
		}
		if len(item.BlockedBy) > 0 {
			fmt.Printf("  Blocked by: %v\n", item.BlockedBy)
		}

		// Show children
		children, _ := eng.GetChildren(item.ID)
		if len(children) > 0 {
			fmt.Println("  Children:")
			for _, child := range children {
				fmt.Printf("    #%-4d %-8s %-14s %s\n", child.DisplayNum, child.Type, child.Status, child.Title)
			}
		}

		// Show history
		if len(item.History) > 0 {
			fmt.Println("  History:")
			for _, h := range item.History {
				sessionInfo := ""
				if h.SessionID != "" {
					sessionInfo = fmt.Sprintf(" (session: %s)", h.SessionID)
				}
				fmt.Printf("    %s: %s%s\n", h.Timestamp.Format("2006-01-02 15:04"), h.Detail, sessionInfo)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
```

**Step 4: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add cmd/list.go cmd/show.go internal/engine/query.go
git commit -m "feat: add list and show commands with filtering and tree view"
```

---

## Task 10: CLI Commands — User Management

**Files:**
- Create: `cmd/user.go`

**Step 1: Create user command with subcommands**

Create `cmd/user.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	userType     string
	userProvider string
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage board users (humans and agents)",
}

var userAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Register a new user or agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.AddUser(args[0], userType, userProvider); err != nil {
			return err
		}
		fmt.Printf("Added %s user: %s (provider: %s)\n", userType, args[0], userProvider)
		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered users",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		board, err := eng.ListBoard()
		if err != nil {
			return err
		}

		if flagFormat == "json" {
			data, _ := json.MarshalIndent(board.Users, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(board.Users) == 0 {
			fmt.Println("No users registered. Use 'ob user add' to register users.")
			return nil
		}

		for _, u := range board.Users {
			fmt.Printf("[%s] %s (%s/%s)\n", u.ID[:6], u.Name, u.Type, u.Provider)
		}
		return nil
	},
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.RemoveUser(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed user %s\n", args[0])
		return nil
	},
}

func init() {
	userAddCmd.Flags().StringVar(&userType, "type", "human", "user type: human or agent")
	userAddCmd.Flags().StringVar(&userProvider, "provider", "local", "provider: local, claude-code, opencode, codex")
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userRemoveCmd)
	rootCmd.AddCommand(userCmd)
}
```

**Step 2: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add cmd/user.go
git commit -m "feat: add user management commands (add, list, remove)"
```

---

## Task 11: CLI Commands — Board Config

**Files:**
- Create: `cmd/board.go`
- Add column management methods to engine

**Step 1: Add column management to engine**

Add to `internal/engine/engine.go`:

```go
func (e *Engine) AddColumn(name string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		if board.HasColumn(name) {
			return fmt.Errorf("column %q already exists", name)
		}
		board.Columns = append(board.Columns, domain.Column{Name: name})
		return nil
	})
}

func (e *Engine) RemoveColumn(name string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		// Check no items are in this column
		for _, item := range board.Items {
			if item.Status == name {
				return fmt.Errorf("cannot remove column %q: it contains items — move them first", name)
			}
		}

		filtered := make([]domain.Column, 0, len(board.Columns))
		found := false
		for _, c := range board.Columns {
			if c.Name == name {
				found = true
				continue
			}
			filtered = append(filtered, c)
		}
		if !found {
			return fmt.Errorf("column %q not found", name)
		}
		board.Columns = filtered
		return nil
	})
}

func (e *Engine) ReorderColumns(names []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		if len(names) != len(board.Columns) {
			return fmt.Errorf("must specify all %d columns", len(board.Columns))
		}

		colMap := make(map[string]domain.Column)
		for _, c := range board.Columns {
			colMap[c.Name] = c
		}

		reordered := make([]domain.Column, len(names))
		for i, name := range names {
			col, ok := colMap[name]
			if !ok {
				return fmt.Errorf("column %q not found", name)
			}
			reordered[i] = col
		}

		board.Columns = reordered
		return nil
	})
}
```

**Step 2: Create board command**

Create `cmd/board.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Board configuration and column management",
}

var boardConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show board configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		board, err := eng.ListBoard()
		if err != nil {
			return err
		}

		if flagFormat == "json" {
			data, _ := json.MarshalIndent(board, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Board: %s\n", board.Name)
		fmt.Printf("Version: %d\n", board.Version)
		fmt.Printf("Agent Role: %s\n", board.AgentRole)
		fmt.Printf("Columns: ")
		for i, c := range board.Columns {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(c.Name)
			if c.Limit > 0 {
				fmt.Printf("(%d)", c.Limit)
			}
		}
		fmt.Println()
		fmt.Printf("Items: %d\n", len(board.Items))
		fmt.Printf("Users: %d\n", len(board.Users))
		return nil
	},
}

var columnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "Manage board columns",
}

var columnsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new column",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.AddColumn(args[0]); err != nil {
			return err
		}
		fmt.Printf("Added column: %s\n", args[0])
		return nil
	},
}

var columnsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a column (must be empty)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		if err := eng.RemoveColumn(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed column: %s\n", args[0])
		return nil
	},
}

var columnsReorderCmd = &cobra.Command{
	Use:   "reorder <col1,col2,...>",
	Short: "Reorder columns",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		names := strings.Split(args[0], ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}

		eng := getEngine()
		if err := eng.ReorderColumns(names); err != nil {
			return err
		}
		fmt.Printf("Columns reordered: %s\n", strings.Join(names, ", "))
		return nil
	},
}

func init() {
	columnsCmd.AddCommand(columnsAddCmd)
	columnsCmd.AddCommand(columnsRemoveCmd)
	columnsCmd.AddCommand(columnsReorderCmd)
	boardCmd.AddCommand(boardConfigCmd)
	boardCmd.AddCommand(columnsCmd)
	rootCmd.AddCommand(boardCmd)
}
```

**Step 3: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/board.go internal/engine/engine.go
git commit -m "feat: add board config and column management commands"
```

---

## Task 12: CLI Commands — Skill Install

**Files:**
- Create: `cmd/skill.go`
- Create: `skill/obeya.md`

**Step 1: Write the agent skill file**

Create `skill/obeya.md`:

```markdown
# Obeya Board — Agent Skill

You have access to the `ob` CLI tool for managing a Kanban board. Use it to track, organize, and update work items.

## Setup

Before using board commands, set your identity:

```bash
export OB_USER=<your-user-id>
export OB_SESSION=<unique-session-id>
```

Or pass `--as <user-id> --session <session-id>` on each command.

Check the board state first:

```bash
ob board config --format json    # See board structure
ob list --format json            # See all items
ob user list --format json       # See registered users
```

## Permissions

Check your role: `ob board config --format json` — look at `agent_role`.
- `admin`: Full access to all operations.
- `contributor`: Only modify items assigned to you. Do not reassign, delete, or modify others' items.

## Commands

### Creating Items

```bash
ob create epic "Title"
ob create story "Title" -p <parent-id>
ob create task "Title" -p <parent-id> --priority high --tag backend
```

Types: `epic`, `story`, `task`. Use `-p` to nest under a parent.

### Moving Items

```bash
ob move <id> <status>
```

Use `ob board config` to see valid column names.

### Querying

```bash
ob list --format json                    # All items as JSON
ob list --assignee <user-id> --format json  # Your items
ob list --status in-progress --format json
ob list --blocked --format json          # Blocked items
ob show <id> --format json               # Item detail + history
```

Always use `--format json` for machine-readable output.

### Assigning

```bash
ob assign <id> --to <user-id>
```

### Dependencies

```bash
ob block <id> --by <blocker-id>     # Mark as blocked
ob unblock <id> --by <blocker-id>   # Remove blocker
```

### Editing

```bash
ob edit <id> --title "New title"
ob edit <id> --priority critical
ob edit <id> -d "Updated description"
```

### Deleting

```bash
ob delete <id>    # Fails if item has children
```

## Workflow Convention

1. **Start of session**: Run `ob list --assignee <your-id> --format json` to see assigned work.
2. **Pick a task**: Move it to in-progress: `ob move <id> in-progress`
3. **During work**: Create subtasks as needed: `ob create task "Subtask" -p <id>`
4. **Report blockers**: `ob block <id> --by <blocker-id>`
5. **Complete work**: Move to done: `ob move <id> done`
6. **Update parent**: If all children of a story are done, move the story to done.

## ID Resolution

Items can be referenced by display number (`3`) or hash prefix (`a3f`). Display numbers are easier for quick use.

## Error Handling

All commands fail fast with clear error messages. Common errors:
- "no board found — run 'ob init' first"
- "invalid status" — check valid columns with `ob board config`
- "cannot delete item: it has children"
- "an item cannot block itself"
```

**Step 2: Create skill install command**

Create `cmd/skill.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var skillProvider string

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage the Obeya agent skill",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the agent skill for detected or specified providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		skillContent, err := getSkillContent()
		if err != nil {
			return err
		}

		if skillProvider != "" {
			return installForProvider(skillProvider, skillContent)
		}

		// Auto-detect providers
		installed := 0
		providers := detectProviders()
		for _, p := range providers {
			if err := installForProvider(p, skillContent); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to install for %s: %v\n", p, err)
				continue
			}
			installed++
		}

		if installed == 0 && len(providers) == 0 {
			fmt.Println("No known agent providers detected. Use --provider to specify one.")
		}

		return nil
	},
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported providers and their install status",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		providerPaths := getProviderPaths(home)

		for name, path := range providerPaths {
			status := "not installed"
			if _, err := os.Stat(path); err == nil {
				status = "installed"
			}
			fmt.Printf("%-15s %s (%s)\n", name, status, path)
		}
		return nil
	},
}

func getSkillContent() ([]byte, error) {
	// Look for skill file relative to the executable, then in known locations
	execPath, err := os.Executable()
	if err == nil {
		skillPath := filepath.Join(filepath.Dir(execPath), "skill", "obeya.md")
		if data, err := os.ReadFile(skillPath); err == nil {
			return data, nil
		}
	}

	// Try current directory
	if data, err := os.ReadFile("skill/obeya.md"); err == nil {
		return data, nil
	}

	return nil, fmt.Errorf("skill file not found — ensure skill/obeya.md exists")
}

func getProviderPaths(home string) map[string]string {
	paths := map[string]string{
		"claude-code": filepath.Join(home, ".claude", "skills", "obeya.md"),
	}

	if runtime.GOOS == "darwin" {
		paths["opencode"] = filepath.Join(home, ".config", "opencode", "skills", "obeya.md")
	} else {
		paths["opencode"] = filepath.Join(home, ".config", "opencode", "skills", "obeya.md")
	}

	return paths
}

func detectProviders() []string {
	home, _ := os.UserHomeDir()
	var found []string

	// Claude Code
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		found = append(found, "claude-code")
	}

	return found
}

func installForProvider(provider string, content []byte) error {
	home, _ := os.UserHomeDir()
	paths := getProviderPaths(home)

	destPath, ok := paths[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q — supported: claude-code, opencode", provider)
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Printf("Installed obeya skill for %s at %s\n", provider, destPath)
	return nil
}

func init() {
	skillInstallCmd.Flags().StringVar(&skillProvider, "provider", "", "specific provider to install for")
	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillListCmd)
	rootCmd.AddCommand(skillCmd)
}
```

**Step 3: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/skill.go skill/obeya.md
git commit -m "feat: add agent skill file and skill install command"
```

---

## Task 13: Minimal TUI

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/view.go`
- Create: `cmd/tui.go`

**Step 1: Create TUI model**

Create `internal/tui/model.go`:

```go
package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
)

type Model struct {
	engine    *engine.Engine
	board     *domain.Board
	columns   []string
	cursorCol int
	cursorRow int
	err       error
}

func New(eng *engine.Engine) Model {
	return Model{engine: eng}
}

type boardLoadedMsg struct {
	board *domain.Board
}

type errMsg struct {
	err error
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		board, err := m.engine.ListBoard()
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{board}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case boardLoadedMsg:
		m.board = msg.board
		m.columns = make([]string, len(m.board.Columns))
		for i, c := range m.board.Columns {
			m.columns[i] = c.Name
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "h", "left":
			if m.cursorCol > 0 {
				m.cursorCol--
				m.cursorRow = 0
			}
		case "l", "right":
			if m.cursorCol < len(m.columns)-1 {
				m.cursorCol++
				m.cursorRow = 0
			}
		case "j", "down":
			items := m.itemsInColumn(m.columns[m.cursorCol])
			if m.cursorRow < len(items)-1 {
				m.cursorRow++
			}
		case "k", "up":
			if m.cursorRow > 0 {
				m.cursorRow--
			}
		case "r":
			// Reload board
			return m, func() tea.Msg {
				board, err := m.engine.ListBoard()
				if err != nil {
					return errMsg{err}
				}
				return boardLoadedMsg{board}
			}
		}
	}

	return m, nil
}

func (m Model) itemsInColumn(colName string) []*domain.Item {
	if m.board == nil {
		return nil
	}
	var items []*domain.Item
	for _, item := range m.board.Items {
		if item.Status == colName {
			items = append(items, item)
		}
	}
	return items
}
```

**Step 2: Create TUI view**

Create `internal/tui/view.go`:

```go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.board == nil {
		return "Loading board..."
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  Obeya Board: %s\n", m.board.Name))
	sb.WriteString(strings.Repeat("─", 80) + "\n")

	// Column headers
	colWidth := 18
	for i, col := range m.columns {
		marker := " "
		if i == m.cursorCol {
			marker = ">"
		}
		header := fmt.Sprintf("%s%-*s", marker, colWidth-1, strings.ToUpper(col))
		sb.WriteString(header)
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", colWidth*len(m.columns)) + "\n")

	// Find max items in any column
	maxRows := 0
	columnItems := make([][]*domain.Item, len(m.columns))
	for i, col := range m.columns {
		items := m.itemsInColumn(col)
		sort.Slice(items, func(a, b int) bool {
			return items[a].DisplayNum < items[b].DisplayNum
		})
		columnItems[i] = items
		if len(items) > maxRows {
			maxRows = len(items)
		}
	}

	// Render rows
	for row := 0; row < maxRows; row++ {
		for col := 0; col < len(m.columns); col++ {
			if row < len(columnItems[col]) {
				item := columnItems[col][row]
				selected := col == m.cursorCol && row == m.cursorRow
				cell := formatCell(item, selected, colWidth)
				sb.WriteString(cell)
			} else {
				sb.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString("  h/l: move columns  j/k: move rows  r: reload  q: quit\n")

	return sb.String()
}

func formatCell(item *domain.Item, selected bool, width int) string {
	prefix := " "
	if selected {
		prefix = ">"
	}

	label := fmt.Sprintf("#%d %s", item.DisplayNum, truncate(item.Title, width-6))

	if len(item.BlockedBy) > 0 {
		label += "!"
	}

	return fmt.Sprintf("%s%-*s", prefix, width-1, label)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
```

**Step 3: Create TUI command**

Create `cmd/tui.go`:

```go
package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive board TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := getEngine()
		model := tui.New(eng)

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
```

**Step 4: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add internal/tui/ cmd/tui.go
git commit -m "feat: add minimal TUI with column board view and keyboard navigation"
```

---

## Task 14: Integration Test — End-to-End CLI Flow

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write integration test**

Create `test/integration_test.go`:

```go
package test

import (
	"testing"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func TestEndToEndFlow(t *testing.T) {
	dir := t.TempDir()
	s := store.NewJSONStore(dir)

	// Init board
	err := s.InitBoard("e2e-test", nil)
	if err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}

	eng := engine.New(s)

	// Add users
	err = eng.AddUser("Alice", "human", "local")
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	err = eng.AddUser("Claude", "agent", "claude-code")
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	// Create hierarchy
	epic, err := eng.CreateItem("epic", "Build Auth System", "", "Authentication epic", "high", "", nil)
	if err != nil {
		t.Fatalf("Create epic failed: %v", err)
	}

	story, err := eng.CreateItem("story", "Login flow", epic.ID, "", "high", "", nil)
	if err != nil {
		t.Fatalf("Create story failed: %v", err)
	}

	task1, err := eng.CreateItem("task", "Create login form", story.ID, "", "medium", "", []string{"frontend"})
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	task2, err := eng.CreateItem("task", "Add JWT validation", story.ID, "", "high", "", []string{"backend"})
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}

	// Verify display numbers
	if epic.DisplayNum != 1 || story.DisplayNum != 2 || task1.DisplayNum != 3 || task2.DisplayNum != 4 {
		t.Errorf("unexpected display nums: %d, %d, %d, %d", epic.DisplayNum, story.DisplayNum, task1.DisplayNum, task2.DisplayNum)
	}

	// Move items
	err = eng.MoveItem("3", "in-progress", "", "session-1")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	// Block task2 by task1
	err = eng.BlockItem(task2.ID, task1.ID, "", "session-1")
	if err != nil {
		t.Fatalf("Block failed: %v", err)
	}

	// Verify blocked
	updated, _ := eng.GetItem(task2.ID)
	if len(updated.BlockedBy) != 1 {
		t.Errorf("expected 1 blocker, got %d", len(updated.BlockedBy))
	}

	// List with filter
	items, err := eng.ListItems(engine.ListFilter{Status: "in-progress"})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 in-progress item, got %d", len(items))
	}

	// Unblock and move to done
	err = eng.UnblockItem(task2.ID, task1.ID, "", "session-1")
	if err != nil {
		t.Fatalf("Unblock failed: %v", err)
	}

	err = eng.MoveItem("3", "done", "", "session-1")
	if err != nil {
		t.Fatalf("Move to done failed: %v", err)
	}

	// Verify history
	final, _ := eng.GetItem(task1.ID)
	if len(final.History) < 3 {
		t.Errorf("expected at least 3 history entries, got %d", len(final.History))
	}

	// Delete should fail (story has children)
	err = eng.DeleteItem(story.ID, "", "")
	if err == nil {
		t.Error("expected error deleting story with children")
	}

	// Delete leaf task
	err = eng.DeleteItem(task1.ID, "", "")
	if err != nil {
		t.Fatalf("Delete leaf task failed: %v", err)
	}

	t.Log("End-to-end flow passed")
}
```

**Step 2: Run integration test**

Run: `go test ./test/ -v`
Expected: PASS

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add test/
git commit -m "test: add end-to-end integration test for full CLI flow"
```

---

## Task 15: Final Build and Binary

**Step 1: Build final binary**

Run: `go build -o ob .`
Expected: `ob` binary created

**Step 2: Quick smoke test**

```bash
cd /tmp && mkdir ob-test && cd ob-test
ob init "my-project"
ob user add "Dev" --type human
ob user add "Claude" --type agent --provider claude-code
ob create epic "Build Feature X"
ob create story "User login" -p 1
ob create task "Design form" -p 2 --priority high --tag frontend
ob list
ob move 3 in-progress
ob show 3
ob list --format json
ob tui
# press q to exit TUI
cd - && rm -rf /tmp/ob-test
```

Expected: All commands work, TUI shows the board

**Step 3: Add .gitignore**

Create `.gitignore`:

```
ob
*.exe
.obeya/
```

**Step 4: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore for binary and local board files"
```

---

## Summary

| Task | Description | Estimated Steps |
|------|-------------|-----------------|
| 1 | Project scaffolding | 6 |
| 2 | Domain types | 5 |
| 3 | ID generation | 5 |
| 4 | Storage interface + JSON store | 6 |
| 5 | Engine — business logic | 5 |
| 6 | CLI — init command | 4 |
| 7 | CLI — create command | 3 |
| 8 | CLI — move, assign, block, edit, delete | 7 |
| 9 | CLI — list and show | 5 |
| 10 | CLI — user management | 3 |
| 11 | CLI — board config | 4 |
| 12 | Skill file + install | 4 |
| 13 | Minimal TUI | 5 |
| 14 | Integration test | 4 |
| 15 | Final build + smoke test | 4 |

**Total: 15 tasks, ~70 steps**
