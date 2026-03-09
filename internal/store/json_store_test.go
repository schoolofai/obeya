package store_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func setupTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
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

func TestJSONStore_ImplementsStoreInterface(t *testing.T) {
	dir := setupTempDir(t)
	var s store.Store = store.NewJSONStore(dir)
	if s == nil {
		t.Error("JSONStore should implement Store interface")
	}
}
