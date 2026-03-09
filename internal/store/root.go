package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from startDir looking for the project root.
// First pass: looks for .obeya/board.json (existing board).
// Second pass: looks for .git (git repository root).
// Returns an error if neither is found.
func FindProjectRoot(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Pass 1: walk up looking for .obeya/board.json
	if root, found := walkUpFor(abs, obeyaBoardExists); found {
		return root, nil
	}

	// Pass 2: walk up looking for .git
	if root, found := walkUpFor(abs, gitDirExists); found {
		return root, nil
	}

	return "", fmt.Errorf("no git repository found — use 'ob init --root <path>' to specify a board location")
}

// walkUpFor walks from dir toward the filesystem root, calling check at each level.
// Returns the directory where check returned true and a boolean indicating success.
func walkUpFor(dir string, check func(string) bool) (string, bool) {
	for {
		if check(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func obeyaBoardExists(dir string) bool {
	boardFile := filepath.Join(dir, ".obeya", "board.json")
	_, err := os.Stat(boardFile)
	return err == nil
}

func gitDirExists(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// FindGitRoot walks up from startDir looking for a .git directory.
// Returns the directory containing .git, or an error if none is found.
func FindGitRoot(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if root, found := walkUpFor(abs, gitDirExists); found {
		return root, nil
	}

	return "", fmt.Errorf("no git repository found — use 'ob init --root <path>' to specify a board location")
}
