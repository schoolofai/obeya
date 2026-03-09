package store_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestSharedBoard_TwoProjects_FullE2E(t *testing.T) {
	// ===== SETUP: Global board =====
	homeDir := t.TempDir()
	boardName := "team-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	globalStore := store.NewJSONStore(boardDir)
	if err := globalStore.InitBoard(boardName, nil); err != nil {
		t.Fatalf("init global board failed: %v", err)
	}

	// ===== PROJECT A: api-server =====
	projectA := t.TempDir()
	initGitRepo(t, projectA, "project-a")

	// Init local board and add tasks
	storeA := store.NewJSONStore(projectA)
	if err := storeA.InitBoard("api-server", nil); err != nil {
		t.Fatalf("init project A board failed: %v", err)
	}
	if err := storeA.Transaction(func(b *domain.Board) error {
		b.Items["a1"] = &domain.Item{ID: "a1", DisplayNum: 1, Title: "Setup API routes", Status: "todo", Type: "task"}
		b.Items["a2"] = &domain.Item{ID: "a2", DisplayNum: 2, Title: "Add auth middleware", Status: "in-progress", Type: "task"}
		b.Items["a3"] = &domain.Item{ID: "a3", DisplayNum: 3, Title: "Write API tests", Status: "backlog", Type: "task"}
		b.DisplayMap[1] = "a1"
		b.DisplayMap[2] = "a2"
		b.DisplayMap[3] = "a3"
		b.NextDisplay = 4
		return nil
	}); err != nil {
		t.Fatalf("add project A tasks failed: %v", err)
	}

	// Verify local board has 3 tasks
	boardA, err := storeA.LoadBoard()
	if err != nil {
		t.Fatalf("load project A board failed: %v", err)
	}
	if len(boardA.Items) != 3 {
		t.Fatalf("project A should have 3 tasks, got %d", len(boardA.Items))
	}

	// ===== PROJECT B: web-app =====
	projectB := t.TempDir()
	initGitRepo(t, projectB, "project-b")

	storeB := store.NewJSONStore(projectB)
	if err := storeB.InitBoard("web-app", nil); err != nil {
		t.Fatalf("init project B board failed: %v", err)
	}
	if err := storeB.Transaction(func(b *domain.Board) error {
		b.Items["b1"] = &domain.Item{ID: "b1", DisplayNum: 1, Title: "Create React components", Status: "todo", Type: "task"}
		b.Items["b2"] = &domain.Item{ID: "b2", DisplayNum: 2, Title: "Style landing page", Status: "done", Type: "task"}
		b.DisplayMap[1] = "b1"
		b.DisplayMap[2] = "b2"
		b.NextDisplay = 3
		return nil
	}); err != nil {
		t.Fatalf("add project B tasks failed: %v", err)
	}

	boardB, err := storeB.LoadBoard()
	if err != nil {
		t.Fatalf("load project B board failed: %v", err)
	}
	if len(boardB.Items) != 2 {
		t.Fatalf("project B should have 2 tasks, got %d", len(boardB.Items))
	}

	// ===== MIGRATE PROJECT A =====
	projectAName := filepath.Base(projectA)
	countA, err := store.MigrateLocalToShared(projectA, boardDir, projectAName)
	if err != nil {
		t.Fatalf("migration A failed: %v", err)
	}
	if countA != 3 {
		t.Errorf("expected 3 migrated from A, got %d", countA)
	}

	// Write .obeya-link for project A
	if err := os.WriteFile(filepath.Join(projectA, ".obeya-link"), []byte(boardName), 0644); err != nil {
		t.Fatalf("write .obeya-link for A failed: %v", err)
	}

	// Register project A
	if err := globalStore.Transaction(func(b *domain.Board) error {
		b.Projects[projectAName] = &domain.LinkedProject{
			Name:      projectAName,
			LocalPath: projectA,
			LinkedAt:  "2026-03-09T10:00:00Z",
		}
		return nil
	}); err != nil {
		t.Fatalf("register project A failed: %v", err)
	}

	// ===== MIGRATE PROJECT B =====
	projectBName := filepath.Base(projectB)
	countB, err := store.MigrateLocalToShared(projectB, boardDir, projectBName)
	if err != nil {
		t.Fatalf("migration B failed: %v", err)
	}
	if countB != 2 {
		t.Errorf("expected 2 migrated from B, got %d", countB)
	}

	// Write .obeya-link for project B
	if err := os.WriteFile(filepath.Join(projectB, ".obeya-link"), []byte(boardName), 0644); err != nil {
		t.Fatalf("write .obeya-link for B failed: %v", err)
	}

	// Register project B
	if err := globalStore.Transaction(func(b *domain.Board) error {
		b.Projects[projectBName] = &domain.LinkedProject{
			Name:      projectBName,
			LocalPath: projectB,
			LinkedAt:  "2026-03-09T11:00:00Z",
		}
		return nil
	}); err != nil {
		t.Fatalf("register project B failed: %v", err)
	}

	// ===== VERIFY GLOBAL BOARD =====
	globalBoard, err := globalStore.LoadBoard()
	if err != nil {
		t.Fatalf("failed to load global board: %v", err)
	}

	// Should have 5 total tasks (3 from A + 2 from B)
	if len(globalBoard.Items) != 5 {
		t.Errorf("expected 5 total tasks on global board, got %d", len(globalBoard.Items))
	}

	// Should have 2 projects registered
	if len(globalBoard.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(globalBoard.Projects))
	}

	// Verify all tasks from project A are tagged correctly
	aCount := 0
	for _, item := range globalBoard.Items {
		if item.Project == projectAName {
			aCount++
		}
	}
	if aCount != 3 {
		t.Errorf("expected 3 tasks tagged with project A, got %d", aCount)
	}

	// Verify all tasks from project B are tagged correctly
	bCount := 0
	for _, item := range globalBoard.Items {
		if item.Project == projectBName {
			bCount++
		}
	}
	if bCount != 2 {
		t.Errorf("expected 2 tasks tagged with project B, got %d", bCount)
	}

	// Verify specific task titles exist with correct project tags
	taskTitles := map[string]string{} // title -> project
	for _, item := range globalBoard.Items {
		taskTitles[item.Title] = item.Project
	}

	expectedTasks := map[string]string{
		"Setup API routes":         projectAName,
		"Add auth middleware":      projectAName,
		"Write API tests":         projectAName,
		"Create React components": projectBName,
		"Style landing page":      projectBName,
	}

	for title, expectedProject := range expectedTasks {
		if gotProject, ok := taskTitles[title]; !ok {
			t.Errorf("task %q not found on global board", title)
		} else if gotProject != expectedProject {
			t.Errorf("task %q: expected project %q, got %q", title, expectedProject, gotProject)
		}
	}

	// Verify task statuses were preserved during migration
	for _, item := range globalBoard.Items {
		if item.Title == "Add auth middleware" && item.Status != "in-progress" {
			t.Errorf("expected 'Add auth middleware' status 'in-progress', got %q", item.Status)
		}
		if item.Title == "Style landing page" && item.Status != "done" {
			t.Errorf("expected 'Style landing page' status 'done', got %q", item.Status)
		}
	}

	// Verify display numbers don't collide
	displayNums := map[int]string{}
	for _, item := range globalBoard.Items {
		if existing, ok := displayNums[item.DisplayNum]; ok {
			t.Errorf("display number %d collision: %q and %q", item.DisplayNum, existing, item.Title)
		}
		displayNums[item.DisplayNum] = item.Title
	}

	// ===== VERIFY LOCAL BACKUPS =====
	if _, err := os.Stat(filepath.Join(projectA, ".obeya")); err == nil {
		t.Error("project A .obeya should have been renamed to backup")
	}
	if _, err := os.Stat(filepath.Join(projectA, ".obeya-local-backup")); err != nil {
		t.Error("project A .obeya-local-backup should exist")
	}

	if _, err := os.Stat(filepath.Join(projectB, ".obeya")); err == nil {
		t.Error("project B .obeya should have been renamed to backup")
	}
	if _, err := os.Stat(filepath.Join(projectB, ".obeya-local-backup")); err != nil {
		t.Error("project B .obeya-local-backup should exist")
	}

	// ===== VERIFY DISCOVERY FROM BOTH PROJECTS =====
	rootA, err := store.FindProjectRootWithHome(projectA, homeDir)
	if err != nil {
		t.Fatalf("discovery from project A failed: %v", err)
	}
	if rootA != boardDir {
		t.Errorf("project A should resolve to global board %s, got %s", boardDir, rootA)
	}

	rootB, err := store.FindProjectRootWithHome(projectB, homeDir)
	if err != nil {
		t.Fatalf("discovery from project B failed: %v", err)
	}
	if rootB != boardDir {
		t.Errorf("project B should resolve to global board %s, got %s", boardDir, rootB)
	}

	// ===== VERIFY DISCOVERY FROM SUBDIRECTORIES =====
	subDirA := filepath.Join(projectA, "src", "handlers")
	if err := os.MkdirAll(subDirA, 0755); err != nil {
		t.Fatalf("create subdirectory failed: %v", err)
	}
	rootFromSub, err := store.FindProjectRootWithHome(subDirA, homeDir)
	if err != nil {
		t.Fatalf("discovery from subdirectory failed: %v", err)
	}
	if rootFromSub != boardDir {
		t.Errorf("subdirectory should resolve to global board %s, got %s", boardDir, rootFromSub)
	}
}

func initGitRepo(t *testing.T, dir, name string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed for %s: %v", name, err)
	}
	// Configure git user for the test repo
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
}
