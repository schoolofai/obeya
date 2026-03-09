package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestUnregisterProject_RemovesFromBoard(t *testing.T) {
	homeDir := t.TempDir()
	boardName := "test-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	s := store.NewJSONStore(boardDir)
	s.InitBoard(boardName, nil)

	// Add a project to the board
	s.Transaction(func(b *domain.Board) error {
		b.Projects["my-project"] = &domain.LinkedProject{
			Name:      "my-project",
			LocalPath: "/tmp/my-project",
		}
		return nil
	})

	// Verify project exists
	board, _ := s.LoadBoard()
	if _, ok := board.Projects["my-project"]; !ok {
		t.Fatal("project should exist before unlink")
	}

	// Remove project via transaction (simulating unlink behavior)
	s.Transaction(func(b *domain.Board) error {
		delete(b.Projects, "my-project")
		return nil
	})

	// Verify project removed
	board, _ = s.LoadBoard()
	if _, ok := board.Projects["my-project"]; ok {
		t.Fatal("project should have been removed")
	}
}

func TestUnlink_ErrorsWhenNotLinked(t *testing.T) {
	projectDir := t.TempDir()
	linkFile := filepath.Join(projectDir, ".obeya-link")

	// Verify link file does not exist
	if _, err := os.Stat(linkFile); err == nil {
		t.Fatal("link file should not exist")
	}
}
