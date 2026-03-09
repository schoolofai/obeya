package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestMigrateLocalToShared(t *testing.T) {
	localDir := t.TempDir()
	localStore := store.NewJSONStore(localDir)
	if err := localStore.InitBoard("local", nil); err != nil {
		t.Fatalf("init local board: %v", err)
	}
	if err := localStore.Transaction(func(b *domain.Board) error {
		b.Items["t1"] = &domain.Item{
			ID: "t1", DisplayNum: 1, Title: "Task one", Status: "todo",
		}
		b.DisplayMap[1] = "t1"
		b.NextDisplay = 2
		return nil
	}); err != nil {
		t.Fatalf("add item: %v", err)
	}

	sharedDir := t.TempDir()
	sharedStore := store.NewJSONStore(sharedDir)
	if err := sharedStore.InitBoard("shared", nil); err != nil {
		t.Fatalf("init shared board: %v", err)
	}

	count, err := store.MigrateLocalToShared(localDir, sharedDir, "my-project")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migrated task, got %d", count)
	}

	board, err := sharedStore.LoadBoard()
	if err != nil {
		t.Fatalf("failed to load shared board: %v", err)
	}

	found := false
	for _, item := range board.Items {
		if item.Title == "Task one" && item.Project == "my-project" {
			found = true
		}
	}
	if !found {
		t.Error("migrated task not found on shared board")
	}

	if _, err := os.Stat(filepath.Join(localDir, ".obeya")); err == nil {
		t.Error("expected .obeya to be renamed")
	}
	if _, err := os.Stat(filepath.Join(localDir, ".obeya-local-backup")); err != nil {
		t.Error("expected .obeya-local-backup to exist")
	}
}

func TestMigrateLocalToShared_EmptyBoard(t *testing.T) {
	localDir := t.TempDir()
	localStore := store.NewJSONStore(localDir)
	if err := localStore.InitBoard("local", nil); err != nil {
		t.Fatalf("init local board: %v", err)
	}

	sharedDir := t.TempDir()
	sharedStore := store.NewJSONStore(sharedDir)
	if err := sharedStore.InitBoard("shared", nil); err != nil {
		t.Fatalf("init shared board: %v", err)
	}

	count, err := store.MigrateLocalToShared(localDir, sharedDir, "my-project")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 migrated tasks, got %d", count)
	}

	// Verify local .obeya was still backed up
	if _, err := os.Stat(filepath.Join(localDir, ".obeya")); err == nil {
		t.Error("expected .obeya to be renamed even for empty board")
	}
	if _, err := os.Stat(filepath.Join(localDir, ".obeya-local-backup")); err != nil {
		t.Error("expected .obeya-local-backup to exist")
	}
}
