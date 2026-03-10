# Git-Root Board Discovery — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `ob` discover the board by walking up directories to find `.obeya/` or `.git/`, so it works from any subdirectory of a project.

**Architecture:** Add a `FindProjectRoot` function in the store package that does a two-pass upward directory walk (first for `.obeya/`, then for `.git/`). All command entry points (`getStore`, `getEngine`, `newEngine`, `ob init`) use this instead of `os.Getwd()` directly.

**Tech Stack:** Go stdlib (`os`, `path/filepath`), existing cobra CLI

---

### Task 1: Create `FindProjectRoot` with tests

**Files:**
- Create: `internal/store/root.go`
- Create: `internal/store/root_test.go`

**Step 1: Write the failing tests**

```go
// internal/store/root_test.go
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
	// .obeya in a child dir, .git in parent — .obeya wins
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
	// Use a temp dir with no .obeya or .git anywhere
	dir := t.TempDir()
	child := filepath.Join(dir, "isolated")
	os.MkdirAll(child, 0755)

	_, err := store.FindProjectRoot(child)
	if err == nil {
		t.Fatal("expected error when no .obeya or .git found")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/ -run TestFindProjectRoot -v`
Expected: FAIL — `FindProjectRoot` doesn't exist yet

**Step 3: Write the implementation**

```go
// internal/store/root.go
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
	dir := abs
	for {
		boardFile := filepath.Join(dir, ".obeya", "board.json")
		if _, err := os.Stat(boardFile); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Pass 2: walk up looking for .git
	dir = abs
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no git repository found — use 'ob init --root <path>' to specify a board location")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run TestFindProjectRoot -v`
Expected: All 5 tests PASS

**Step 5: Commit**

```bash
git add internal/store/root.go internal/store/root_test.go
git commit -m "feat: add FindProjectRoot for upward directory discovery"
```

---

### Task 2: Wire `FindProjectRoot` into `cmd/helpers.go`

**Files:**
- Modify: `cmd/helpers.go:12-31`

**Step 1: Update `getStore` and `getEngine`**

Replace both functions to use `FindProjectRoot`:

```go
func getStore() store.Store {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return store.NewJSONStore(root)
}

func getEngine() (*engine.Engine, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		return nil, err
	}
	s := store.NewJSONStore(root)
	if !s.BoardExists() {
		return nil, fmt.Errorf("no board found — run 'ob init' first")
	}
	return engine.New(s), nil
}
```

**Step 2: Fix `cmd/list.go:68-74` — remove duplicate `newEngine`**

The `newEngine` function in `list.go` hardcodes `"."`. Replace it to use the shared `getEngine`:

```go
// Delete the newEngine function entirely (lines 68-74 of cmd/list.go).
// Replace all calls to newEngine() with getEngine().
```

Find all calls to `newEngine()` in `cmd/list.go` and replace with `getEngine()`.

**Step 3: Build and run existing tests**

Run: `go build ./... && go test ./...`
Expected: All pass, no regressions

**Step 4: Manual smoke test from subdirectory**

Run: `mkdir -p /tmp/test-sub && cd /tmp/test-sub && /path/to/ob list`
Expected: Error about no git repo (not a "cannot find board in current dir" error)

Run from project subdirectory: `cd internal/store && ../../ob list`
Expected: Shows the board (finds `.obeya/` in project root)

**Step 5: Commit**

```bash
git add cmd/helpers.go cmd/list.go
git commit -m "feat: use FindProjectRoot in all command entry points"
```

---

### Task 3: Update `ob init` to target git root + add `--root` flag

**Files:**
- Modify: `cmd/init.go:14-49`

**Step 1: Add `--root` flag and git-root targeting**

```go
var initColumns string
var initClaudeMD bool
var initRoot string

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board. Defaults to the git repository root. Use --root to specify a custom location.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := resolveInitRoot()
		if err != nil {
			return err
		}

		s := store.NewJSONStore(root)
		columns := parseColumns(initColumns)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		if err := s.InitBoard(boardName, columns); err != nil {
			return err
		}

		fmt.Printf("Board %q initialized in %s/.obeya/\n", boardName, root)
		if len(columns) > 0 {
			fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
		} else {
			fmt.Println("Columns: backlog, todo, in-progress, review, done")
		}

		if initClaudeMD {
			claudePath := filepath.Join(root, "CLAUDE.md")
			if err := appendClaudeMDAt(claudePath); err != nil {
				return fmt.Errorf("could not update CLAUDE.md: %w", err)
			}
			fmt.Println("Updated CLAUDE.md with Obeya board instructions")
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().BoolVar(&initClaudeMD, "claude-md", true, "append Obeya instructions to project CLAUDE.md")
	initCmd.Flags().StringVar(&initRoot, "root", "", "directory to create .obeya in (default: git repository root)")
	rootCmd.AddCommand(initCmd)
}
```

**Step 2: Add `resolveInitRoot` helper**

```go
func resolveInitRoot() (string, error) {
	if initRoot != "" {
		abs, err := filepath.Abs(initRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve --root path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("--root path does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("--root path is not a directory: %s", abs)
		}
		return abs, nil
	}

	// Default: find git root by walking up
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no git repository found — use 'ob init --root <path>' to specify a board location")
}
```

**Step 3: Update `appendClaudeMD` to accept a path**

Rename `appendClaudeMD()` to `appendClaudeMDAt(claudePath string)` and update it to use the passed path instead of hardcoded `"CLAUDE.md"`. Remove the old `printInitConfirmation` function (inlined above).

**Step 4: Add `"path/filepath"` to imports**

Ensure `cmd/init.go` imports `"path/filepath"` and `"github.com/niladribose/obeya/internal/store"` (store is needed indirectly — actually no, we use `store.NewJSONStore` directly, so yes import it).

Wait — `init.go` currently calls `getStore()` which hides the store import. Now we call `store.NewJSONStore(root)` directly, so add the import.

**Step 5: Build and test**

Run: `go build ./... && go test ./...`
Expected: All pass

**Step 6: Commit**

```bash
git add cmd/init.go
git commit -m "feat: ob init targets git root, add --root flag for non-git projects"
```

---

### Task 4: Final verification and cleanup

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All pass

**Step 2: Build binary**

Run: `go build -o ob .`

**Step 3: Integration smoke tests**

```bash
# Test 1: ob works from subdirectory
cd internal/store && ../../ob list && cd ../..

# Test 2: ob init from subdirectory targets git root
cd /tmp && mkdir test-git && cd test-git && git init
ob init test-from-sub
ls .obeya/board.json  # should exist
cd .. && rm -rf test-git

# Test 3: ob init --root works
cd /tmp && mkdir custom-root && ob init --root /tmp/custom-root my-board
ls /tmp/custom-root/.obeya/board.json  # should exist
rm -rf /tmp/custom-root

# Test 4: error when no git repo
cd /tmp && mkdir no-git && cd no-git
ob list  # should error: "no git repository found..."
cd .. && rm -rf no-git
```

**Step 4: Update task on board**

Run: `ob move 16 done`

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: verify git-root board discovery"
```
