package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestFindProjectRoot_ObeyaDirInCwd(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".obeya"), 0755)
	os.WriteFile(filepath.Join(dir, ".obeya", "board.json"), []byte("{}"), 0644)

	root, err := store.FindProjectRoot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFindProjectRoot_ObeyaDirInParent(t *testing.T) {
	parent := t.TempDir()
	os.MkdirAll(filepath.Join(parent, ".obeya"), 0755)
	os.WriteFile(filepath.Join(parent, ".obeya", "board.json"), []byte("{}"), 0644)

	child := filepath.Join(parent, "src", "pkg")
	os.MkdirAll(child, 0755)

	root, err := store.FindProjectRoot(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != parent {
		t.Errorf("expected %s, got %s", parent, root)
	}
}

func TestFindProjectRoot_GitRootFallback(t *testing.T) {
	parent := t.TempDir()
	os.MkdirAll(filepath.Join(parent, ".git"), 0755)

	child := filepath.Join(parent, "src", "pkg")
	os.MkdirAll(child, 0755)

	root, err := store.FindProjectRoot(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != parent {
		t.Errorf("expected %s, got %s", parent, root)
	}
}

func TestFindProjectRoot_ObeyaTakesPrecedenceOverGit(t *testing.T) {
	grandparent := t.TempDir()
	os.MkdirAll(filepath.Join(grandparent, ".git"), 0755)

	parent := filepath.Join(grandparent, "sub")
	os.MkdirAll(filepath.Join(parent, ".obeya"), 0755)
	os.WriteFile(filepath.Join(parent, ".obeya", "board.json"), []byte("{}"), 0644)

	child := filepath.Join(parent, "deep")
	os.MkdirAll(child, 0755)

	root, err := store.FindProjectRoot(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != parent {
		t.Errorf("expected %s, got %s", parent, root)
	}
}

func TestFindProjectRoot_NothingFound(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "isolated")
	os.MkdirAll(child, 0755)

	_, err := store.FindProjectRoot(child)
	if err == nil {
		t.Fatal("expected error when no .obeya or .git found")
	}
}

func TestFindGitRoot_Found(t *testing.T) {
	parent := t.TempDir()
	os.MkdirAll(filepath.Join(parent, ".git"), 0755)

	child := filepath.Join(parent, "src", "pkg")
	os.MkdirAll(child, 0755)

	root, err := store.FindGitRoot(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != parent {
		t.Errorf("expected %s, got %s", parent, root)
	}
}

func TestFindGitRoot_InCwd(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	root, err := store.FindGitRoot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFindGitRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "no-git")
	os.MkdirAll(child, 0755)

	_, err := store.FindGitRoot(child)
	if err == nil {
		t.Fatal("expected error when no .git found")
	}
}

func TestFindProjectRoot_ObeyaLinkInCwd(t *testing.T) {
	homeDir := t.TempDir()
	boardDir := filepath.Join(homeDir, "boards", "myboard")
	os.MkdirAll(boardDir, 0755)
	os.WriteFile(filepath.Join(boardDir, "board.json"), []byte("{}"), 0644)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte("myboard"), 0644)

	root, err := store.FindProjectRootWithHome(projectDir, homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != boardDir {
		t.Errorf("expected %s, got %s", boardDir, root)
	}
}

func TestFindProjectRoot_ObeyaLinkInParent(t *testing.T) {
	homeDir := t.TempDir()
	boardDir := filepath.Join(homeDir, "boards", "myboard")
	os.MkdirAll(boardDir, 0755)
	os.WriteFile(filepath.Join(boardDir, "board.json"), []byte("{}"), 0644)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte("myboard"), 0644)

	child := filepath.Join(projectDir, "src", "pkg")
	os.MkdirAll(child, 0755)

	root, err := store.FindProjectRootWithHome(child, homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != boardDir {
		t.Errorf("expected %s, got %s", boardDir, root)
	}
}

func TestFindProjectRoot_ObeyaLinkTakesPrecedenceOverLocal(t *testing.T) {
	homeDir := t.TempDir()
	boardDir := filepath.Join(homeDir, "boards", "myboard")
	os.MkdirAll(boardDir, 0755)
	os.WriteFile(filepath.Join(boardDir, "board.json"), []byte("{}"), 0644)

	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte("myboard"), 0644)
	os.MkdirAll(filepath.Join(projectDir, ".obeya"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".obeya", "board.json"), []byte("{}"), 0644)

	root, err := store.FindProjectRootWithHome(projectDir, homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != boardDir {
		t.Errorf("expected linked board %s, got %s", boardDir, root)
	}
}

func TestFindProjectRoot_StaleLink(t *testing.T) {
	homeDir := t.TempDir()

	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, ".obeya-link"), []byte("ghost-board"), 0644)

	_, err := store.FindProjectRootWithHome(projectDir, homeDir)
	if err == nil {
		t.Fatal("expected error for stale .obeya-link")
	}
}

func TestSharedBoardDir(t *testing.T) {
	home := "/home/testuser"
	result := store.SharedBoardDir(home, "client-work")
	expected := "/home/testuser/boards/client-work"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
