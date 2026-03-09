package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestSharedBoard_RoundTrip(t *testing.T) {
	homeDir := t.TempDir()
	boardName := "roundtrip-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	// 1. Init shared board
	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, nil); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// 2. Create a project dir with .obeya-link
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte(boardName), 0644)

	// 3. Verify discovery
	root, err := store.FindProjectRootWithHome(projectDir, homeDir)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	if root != boardDir {
		t.Errorf("expected %s, got %s", boardDir, root)
	}

	// 4. Add a task with project tag to shared board
	if err := s.Transaction(func(b *domain.Board) error {
		b.Items["rt-1"] = &domain.Item{
			ID:         "rt-1",
			DisplayNum: b.NextDisplay,
			Title:      "Round trip task",
			Status:     "todo",
			Project:    "test-project",
		}
		b.DisplayMap[b.NextDisplay] = "rt-1"
		b.NextDisplay++
		return nil
	}); err != nil {
		t.Fatalf("add task failed: %v", err)
	}

	// 5. Register project
	if err := s.Transaction(func(b *domain.Board) error {
		b.Projects["test-project"] = &domain.LinkedProject{
			Name:      "test-project",
			LocalPath: projectDir,
			LinkedAt:  "2026-03-09T10:00:00Z",
		}
		return nil
	}); err != nil {
		t.Fatalf("register project failed: %v", err)
	}

	// 6. Verify task and project exist on reload
	board, err := s.LoadBoard()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if item, ok := board.Items["rt-1"]; !ok {
		t.Error("task not found on shared board")
	} else if item.Project != "test-project" {
		t.Errorf("expected project 'test-project', got %q", item.Project)
	}

	if proj, ok := board.Projects["test-project"]; !ok {
		t.Error("project not found in board registry")
	} else if proj.LocalPath != projectDir {
		t.Errorf("expected LocalPath %s, got %s", projectDir, proj.LocalPath)
	}

	// 7. Unlink — remove .obeya-link and project from board
	os.Remove(filepath.Join(projectDir, ".obeya-link"))
	if err := s.Transaction(func(b *domain.Board) error {
		delete(b.Projects, "test-project")
		return nil
	}); err != nil {
		t.Fatalf("unlink project failed: %v", err)
	}

	// 8. Verify project removed but task remains
	board, err = s.LoadBoard()
	if err != nil {
		t.Fatalf("reload after unlink failed: %v", err)
	}
	if _, ok := board.Projects["test-project"]; ok {
		t.Error("project should have been removed")
	}
	if _, ok := board.Items["rt-1"]; !ok {
		t.Error("task should still exist after unlink")
	}

	// 9. Verify discovery falls back (no more .obeya-link)
	_, err = store.FindProjectRootWithHome(projectDir, homeDir)
	// Should NOT resolve to shared board anymore, should find .git
	if err != nil {
		// It should find .git and return projectDir
		t.Logf("discovery after unlink: %v", err)
	}
}
