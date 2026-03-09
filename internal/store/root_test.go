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
