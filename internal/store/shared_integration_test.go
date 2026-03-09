package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestSharedBoard_EndToEnd(t *testing.T) {
	// 1. Create shared board
	homeDir := t.TempDir()
	boardName := "e2e-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, nil); err != nil {
		t.Fatalf("init shared board failed: %v", err)
	}

	// 2. Setup project with local board + tasks
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)

	localStore := store.NewJSONStore(projectDir)
	localStore.InitBoard("local", nil)
	localStore.Transaction(func(b *domain.Board) error {
		b.Items["task-1"] = &domain.Item{
			ID: "task-1", DisplayNum: 1, Title: "Local task", Status: "todo",
		}
		b.DisplayMap[1] = "task-1"
		b.NextDisplay = 2
		return nil
	})

	// 3. Migrate local to shared
	count, err := store.MigrateLocalToShared(projectDir, boardDir, "my-project")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migrated, got %d", count)
	}

	// 4. Write .obeya-link
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte(boardName), 0644)

	// 5. Verify discovery resolves to shared board
	root, err := store.FindProjectRootWithHome(projectDir, homeDir)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	if root != boardDir {
		t.Errorf("expected discovery to resolve to %s, got %s", boardDir, root)
	}

	// 6. Verify migrated task on shared board
	board, _ := s.LoadBoard()
	found := false
	for _, item := range board.Items {
		if item.Project == "my-project" && item.Title == "Local task" {
			found = true
		}
	}
	if !found {
		t.Error("migrated task not found on shared board")
	}

	// 7. Verify local backup exists
	if _, err := os.Stat(filepath.Join(projectDir, ".obeya-local-backup")); err != nil {
		t.Error("expected .obeya-local-backup to exist")
	}
}
