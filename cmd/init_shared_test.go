package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestInitShared_CreatesBoard(t *testing.T) {
	homeDir := t.TempDir()
	boardName := "test-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)
	boardFile := filepath.Join(boardDir, ".obeya", "board.json")

	if _, err := os.Stat(boardFile); err == nil {
		t.Fatal("board should not exist before init")
	}

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, nil); err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}

	if _, err := os.Stat(boardFile); err != nil {
		t.Fatalf("board.json not created: %v", err)
	}
}

func TestInitShared_ErrorsIfBoardExists(t *testing.T) {
	homeDir := t.TempDir()
	boardName := "existing-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, nil); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	s2 := store.NewJSONStore(boardDir)
	err := s2.InitBoard(boardName, nil)
	if err == nil {
		t.Fatal("expected error when board already exists")
	}
}
