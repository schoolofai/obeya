# User-Level Shared Boards — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow multiple projects to share a single Obeya board at `~/.obeya/boards/<name>/`, with automatic task migration when linking.

**Architecture:** Extend `FindProjectRoot` with a new `.obeya-link` pass, add `LinkedProject`/`Project` fields to domain types, add `init --shared`, `link`, `unlink`, `boards`, and `board prune` commands.

**Tech Stack:** Go, cobra, gofrs/flock (existing deps)

---

### Task 1: Add LinkedProject and Project fields to domain types

**Files:**
- Modify: `internal/domain/types.go:90-105` (Item struct), `internal/domain/types.go:107-119` (Board struct)

**Step 1: Write the failing test**

Create: `internal/domain/types_shared_test.go`

```go
package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestLinkedProject_Fields(t *testing.T) {
	lp := domain.LinkedProject{
		Name:      "api-server",
		LocalPath: "/home/user/code/api-server",
		GitRemote: "git@github.com:user/api-server.git",
		LinkedAt:  "2026-03-09T10:00:00Z",
	}
	if lp.Name != "api-server" {
		t.Errorf("expected Name 'api-server', got %q", lp.Name)
	}
	if lp.GitRemote != "git@github.com:user/api-server.git" {
		t.Errorf("expected GitRemote set, got %q", lp.GitRemote)
	}
}

func TestItem_ProjectField(t *testing.T) {
	item := domain.Item{
		ID:      "test-1",
		Title:   "Test task",
		Project: "api-server",
	}
	if item.Project != "api-server" {
		t.Errorf("expected Project 'api-server', got %q", item.Project)
	}
}

func TestBoard_ProjectsMap(t *testing.T) {
	b := domain.NewBoard("test")
	if b.Projects == nil {
		t.Fatal("expected Projects map to be initialized")
	}
	if len(b.Projects) != 0 {
		t.Errorf("expected empty Projects map, got %d entries", len(b.Projects))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -run TestLinkedProject -v`
Expected: FAIL — `LinkedProject` type not defined

**Step 3: Write minimal implementation**

Add to `internal/domain/types.go` before the `Item` struct:

```go
type LinkedProject struct {
	Name      string `json:"name"`
	LocalPath string `json:"local_path"`
	GitRemote string `json:"git_remote"`
	LinkedAt  string `json:"linked_at"`
}
```

Add `Project` field to `Item` struct (after `Tags`):

```go
Project string `json:"project,omitempty"`
```

Add `Projects` field to `Board` struct (after `Plans`):

```go
Projects map[string]*LinkedProject `json:"projects"`
```

Initialize `Projects` in `NewBoardWithColumns` (after `Plans` init):

```go
Projects: make(map[string]*LinkedProject),
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/ -run "TestLinkedProject|TestItem_ProjectField|TestBoard_ProjectsMap" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/types.go internal/domain/types_shared_test.go
git commit -m "feat: add LinkedProject type and Project field to Item and Board"
```

---

### Task 2: Add SharedBoardDir helper and .obeya-link discovery to FindProjectRoot

**Files:**
- Modify: `internal/store/root.go`
- Modify: `internal/store/root_test.go`

**Step 1: Write the failing tests**

Append to `internal/store/root_test.go`:

```go
func TestFindProjectRoot_ObeyaLinkInCwd(t *testing.T) {
	// Setup: global board at a temp "home" dir
	homeDir := t.TempDir()
	boardDir := filepath.Join(homeDir, "boards", "myboard")
	os.MkdirAll(boardDir, 0755)
	os.WriteFile(filepath.Join(boardDir, "board.json"), []byte("{}"), 0644)

	// Setup: project dir with .obeya-link
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
	// No board created — link is stale

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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/ -run "TestFindProjectRoot_ObeyaLink|TestFindProjectRoot_StaleLink|TestSharedBoardDir" -v`
Expected: FAIL — `FindProjectRootWithHome` and `SharedBoardDir` not defined

**Step 3: Write minimal implementation**

Add to `internal/store/root.go`:

```go
// SharedBoardDir returns the path for a named shared board under the obeya home.
// obeyaHome is typically ~/.obeya.
func SharedBoardDir(obeyaHome, boardName string) string {
	return filepath.Join(obeyaHome, "boards", boardName)
}

// ObeyaHome returns the default obeya home directory (~/.obeya).
func ObeyaHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".obeya"), nil
}

// FindProjectRootWithHome is like FindProjectRoot but accepts an explicit obeya home
// directory for testing. Production code should use FindProjectRoot.
func FindProjectRootWithHome(startDir, obeyaHome string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Pass 0: walk up looking for .obeya-link
	if linkDir, found := walkUpFor(abs, obeyaLinkExists); found {
		return resolveLink(linkDir, obeyaHome)
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

func obeyaLinkExists(dir string) bool {
	linkFile := filepath.Join(dir, ".obeya-link")
	_, err := os.Stat(linkFile)
	return err == nil
}

func resolveLink(dir, obeyaHome string) (string, error) {
	linkFile := filepath.Join(dir, ".obeya-link")
	data, err := os.ReadFile(linkFile)
	if err != nil {
		return "", fmt.Errorf("failed to read .obeya-link: %w", err)
	}
	boardName := strings.TrimSpace(string(data))
	if boardName == "" {
		return "", fmt.Errorf(".obeya-link file is empty")
	}

	boardDir := SharedBoardDir(obeyaHome, boardName)
	boardFile := filepath.Join(boardDir, "board.json")
	if _, err := os.Stat(boardFile); err != nil {
		return "", fmt.Errorf("linked board %q not found at %s — run 'ob unlink' to remove the stale link", boardName, boardDir)
	}
	return boardDir, nil
}
```

Add `"strings"` to the imports in `root.go`.

Update `FindProjectRoot` to delegate:

```go
func FindProjectRoot(startDir string) (string, error) {
	obeyaHome, err := ObeyaHome()
	if err != nil {
		return "", err
	}
	return FindProjectRootWithHome(startDir, obeyaHome)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run "TestFindProjectRoot|TestSharedBoardDir" -v`
Expected: ALL PASS (including existing tests)

**Step 5: Commit**

```bash
git add internal/store/root.go internal/store/root_test.go
git commit -m "feat: add .obeya-link discovery and SharedBoardDir to FindProjectRoot"
```

---

### Task 3: Add `ob init --shared` command

**Files:**
- Modify: `cmd/init.go`

**Step 1: Write the failing test**

Create: `cmd/init_shared_test.go`

```go
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
	boardFile := filepath.Join(boardDir, "board.json")

	// The board dir should not exist yet
	if _, err := os.Stat(boardFile); err == nil {
		t.Fatal("board should not exist before init")
	}

	// Create the board directory and init
	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, nil); err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}

	// Verify board.json was created
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

	// Second init should fail
	s2 := store.NewJSONStore(boardDir)
	err := s2.InitBoard(boardName, nil)
	if err == nil {
		t.Fatal("expected error when board already exists")
	}
}
```

**Step 2: Run test to verify it fails/passes**

Run: `go test ./cmd/ -run "TestInitShared" -v`
Expected: PASS (this tests existing JSONStore behavior; the real new code is the CLI wiring)

**Step 3: Write the command changes**

Modify `cmd/init.go` — add `--shared` flag:

```go
var initShared string
```

In `init()` function, add:

```go
initCmd.Flags().StringVar(&initShared, "shared", "", "create a shared board at ~/.obeya/boards/<name>")
```

In the `RunE` function, add at the top (before `resolveInitRoot`):

```go
if initShared != "" {
    return initSharedBoard(initShared, columns)
}
```

Add `initSharedBoard` function:

```go
func initSharedBoard(boardName string, columns []string) error {
	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}

	boardDir := store.SharedBoardDir(obeyaHome, boardName)
	boardFile := filepath.Join(boardDir, "board.json")

	if _, err := os.Stat(boardFile); err == nil {
		return fmt.Errorf("board %q already exists — use 'ob link %s' to connect this project", boardName, boardName)
	}

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Shared board %q initialized at %s\n", boardName, boardDir)
	return nil
}
```

**Step 4: Run all init tests**

Run: `go test ./cmd/ -run "TestInit" -v && go build ./...`
Expected: PASS, builds clean

**Step 5: Commit**

```bash
git add cmd/init.go cmd/init_shared_test.go
git commit -m "feat: add ob init --shared to create user-level boards"
```

---

### Task 4: Add `ob link` command with migration

**Files:**
- Create: `cmd/link.go`
- Create: `cmd/link_test.go`
- Create: `internal/store/migrate.go`
- Create: `internal/store/migrate_test.go`

**Step 1: Write the failing test for migration logic**

Create `internal/store/migrate_test.go`:

```go
package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestMigrateLocalToShared(t *testing.T) {
	// Setup local board with tasks
	localDir := t.TempDir()
	localStore := store.NewJSONStore(localDir)
	localStore.InitBoard("local", nil)
	localStore.Transaction(func(b *domain.Board) error {
		b.Items["t1"] = &domain.Item{
			ID: "t1", DisplayNum: 1, Title: "Task one", Status: "todo",
		}
		b.DisplayMap[1] = "t1"
		b.NextDisplay = 2
		return nil
	})

	// Setup shared board
	sharedDir := t.TempDir()
	sharedStore := store.NewJSONStore(sharedDir)
	sharedStore.InitBoard("shared", nil)

	// Migrate
	count, err := store.MigrateLocalToShared(localDir, sharedDir, "my-project")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migrated task, got %d", count)
	}

	// Verify task on shared board
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

	// Verify local .obeya renamed to .obeya-local-backup
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
	localStore.InitBoard("local", nil)

	sharedDir := t.TempDir()
	sharedStore := store.NewJSONStore(sharedDir)
	sharedStore.InitBoard("shared", nil)

	count, err := store.MigrateLocalToShared(localDir, sharedDir, "my-project")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 migrated tasks, got %d", count)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestMigrateLocal -v`
Expected: FAIL — `MigrateLocalToShared` not defined

**Step 3: Write migration implementation**

Create `internal/store/migrate.go`:

```go
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

// MigrateLocalToShared copies all items from a local board to a shared board,
// tagging each with the project name. Renames local .obeya/ to .obeya-local-backup/.
// Returns the number of migrated items.
func MigrateLocalToShared(localRoot, sharedBoardDir, projectName string) (int, error) {
	localStore := NewJSONStore(localRoot)
	localBoard, err := localStore.LoadBoard()
	if err != nil {
		return 0, fmt.Errorf("failed to load local board: %w", err)
	}

	itemCount := len(localBoard.Items)
	if itemCount == 0 {
		if err := backupLocalObeya(localRoot); err != nil {
			return 0, err
		}
		return 0, nil
	}

	sharedStore := NewJSONStore(sharedBoardDir)
	err = sharedStore.Transaction(func(shared *domain.Board) error {
		for _, item := range localBoard.Items {
			newID := fmt.Sprintf("%s-%s", projectName, item.ID)
			migrated := *item
			migrated.ID = newID
			migrated.Project = projectName
			migrated.DisplayNum = shared.NextDisplay
			shared.Items[newID] = &migrated
			shared.DisplayMap[shared.NextDisplay] = newID
			shared.NextDisplay++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to write migrated items to shared board: %w", err)
	}

	if err := backupLocalObeya(localRoot); err != nil {
		return 0, err
	}

	return itemCount, nil
}

func backupLocalObeya(root string) error {
	src := filepath.Join(root, ".obeya")
	dst := filepath.Join(root, ".obeya-local-backup")
	if _, err := os.Stat(dst); err == nil {
		dst = fmt.Sprintf("%s-%d", dst, time.Now().Unix())
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to rename .obeya to backup: %w", err)
	}
	return nil
}
```

**Step 4: Run migration tests**

Run: `go test ./internal/store/ -run TestMigrateLocal -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/migrate.go internal/store/migrate_test.go
git commit -m "feat: add MigrateLocalToShared for task migration on link"
```

**Step 6: Write the failing test for link command**

Create `cmd/link_test.go`:

```go
package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestLink_WritesLinkFile(t *testing.T) {
	homeDir := t.TempDir()
	boardName := "test-board"
	boardDir := store.SharedBoardDir(homeDir, boardName)

	s := store.NewJSONStore(boardDir)
	s.InitBoard(boardName, nil)

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)

	linkFile := filepath.Join(projectDir, ".obeya-link")
	os.WriteFile(linkFile, []byte(boardName), 0644)

	data, err := os.ReadFile(linkFile)
	if err != nil {
		t.Fatalf("failed to read link file: %v", err)
	}
	if string(data) != boardName {
		t.Errorf("expected %q, got %q", boardName, string(data))
	}
}
```

**Step 7: Write link command**

Create `cmd/link.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var linkMigrate bool

var linkCmd = &cobra.Command{
	Use:   "link <board-name>",
	Short: "Link this project to a shared board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardName := args[0]

		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardDir := store.SharedBoardDir(obeyaHome, boardName)
		boardFile := filepath.Join(boardDir, "board.json")
		if _, err := os.Stat(boardFile); err != nil {
			return fmt.Errorf("board %q not found — run 'ob init --shared %s' first", boardName, boardName)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		gitRoot, err := store.FindGitRoot(cwd)
		if err != nil {
			return err
		}

		linkFile := filepath.Join(gitRoot, ".obeya-link")
		if _, err := os.Stat(linkFile); err == nil {
			existing, _ := os.ReadFile(linkFile)
			return fmt.Errorf("this project is already linked to board %q", strings.TrimSpace(string(existing)))
		}

		// Check for local board and handle migration
		localBoard := filepath.Join(gitRoot, ".obeya", "board.json")
		if _, err := os.Stat(localBoard); err == nil {
			localStore := store.NewJSONStore(gitRoot)
			board, loadErr := localStore.LoadBoard()
			if loadErr != nil {
				return fmt.Errorf("failed to read local board: %w", loadErr)
			}

			taskCount := len(board.Items)
			if taskCount > 0 && !linkMigrate {
				return fmt.Errorf(
					"this project has %d tasks on a local board — rerun with --migrate to move them to %q",
					taskCount, boardName,
				)
			}

			migrated, migErr := store.MigrateLocalToShared(gitRoot, boardDir, resolveProjectName(gitRoot))
			if migErr != nil {
				return fmt.Errorf("migration failed: %w", migErr)
			}
			if migrated > 0 {
				fmt.Printf("Migrated %d tasks to shared board %q\n", migrated, boardName)
			}
		}

		// Write .obeya-link
		if err := os.WriteFile(linkFile, []byte(boardName), 0644); err != nil {
			return fmt.Errorf("failed to write .obeya-link: %w", err)
		}

		// Register project in shared board
		projectName := resolveProjectName(gitRoot)
		gitRemote := resolveGitRemote(gitRoot)

		sharedStore := store.NewJSONStore(boardDir)
		err = sharedStore.Transaction(func(b *domain.Board) error {
			if b.Projects == nil {
				b.Projects = make(map[string]*domain.LinkedProject)
			}
			b.Projects[projectName] = &domain.LinkedProject{
				Name:      projectName,
				LocalPath: gitRoot,
				GitRemote: gitRemote,
				LinkedAt:  time.Now().Format(time.RFC3339),
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to register project in shared board: %w", err)
		}

		fmt.Printf("Linked project %q to shared board %q\n", projectName, boardName)
		return nil
	},
}

func init() {
	linkCmd.Flags().BoolVar(&linkMigrate, "migrate", false, "migrate local board tasks to the shared board")
	rootCmd.AddCommand(linkCmd)
}

func resolveProjectName(gitRoot string) string {
	remote := resolveGitRemote(gitRoot)
	if remote != "" {
		// Extract org/repo from remote URL
		remote = strings.TrimSuffix(remote, ".git")
		parts := strings.Split(remote, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	}
	return filepath.Base(gitRoot)
}

func resolveGitRemote(gitRoot string) string {
	cmd := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

**Step 8: Run tests and build**

Run: `go test ./cmd/ -run TestLink -v && go build ./...`
Expected: PASS, builds clean

**Step 9: Commit**

```bash
git add cmd/link.go cmd/link_test.go
git commit -m "feat: add ob link command with task migration"
```

---

### Task 5: Add `ob unlink` command

**Files:**
- Create: `cmd/unlink.go`

**Step 1: Write the failing test**

Create `cmd/unlink_test.go`:

```go
package cmd_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnlink_RemovesLinkFile(t *testing.T) {
	projectDir := t.TempDir()
	linkFile := filepath.Join(projectDir, ".obeya-link")
	os.WriteFile(linkFile, []byte("test-board"), 0644)

	// Verify file exists
	if _, err := os.Stat(linkFile); err != nil {
		t.Fatalf("link file should exist: %v", err)
	}

	// Remove it (simulating unlink)
	os.Remove(linkFile)

	if _, err := os.Stat(linkFile); err == nil {
		t.Fatal("link file should have been removed")
	}
}
```

**Step 2: Write unlink command**

Create `cmd/unlink.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink this project from its shared board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		gitRoot, err := store.FindGitRoot(cwd)
		if err != nil {
			return err
		}

		linkFile := filepath.Join(gitRoot, ".obeya-link")
		data, err := os.ReadFile(linkFile)
		if err != nil {
			return fmt.Errorf("this project is not linked to any shared board")
		}

		boardName := strings.TrimSpace(string(data))
		projectName := resolveProjectName(gitRoot)

		// Remove project from shared board's registry
		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardDir := store.SharedBoardDir(obeyaHome, boardName)
		if _, err := os.Stat(filepath.Join(boardDir, "board.json")); err == nil {
			sharedStore := store.NewJSONStore(boardDir)
			txErr := sharedStore.Transaction(func(b *domain.Board) error {
				delete(b.Projects, projectName)
				return nil
			})
			if txErr != nil {
				return fmt.Errorf("failed to unregister project from shared board: %w", txErr)
			}
		}

		if err := os.Remove(linkFile); err != nil {
			return fmt.Errorf("failed to remove .obeya-link: %w", err)
		}

		fmt.Printf("Unlinked project %q from shared board %q\n", projectName, boardName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}
```

**Step 3: Run tests and build**

Run: `go test ./cmd/ -run TestUnlink -v && go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/unlink.go cmd/unlink_test.go
git commit -m "feat: add ob unlink command"
```

---

### Task 6: Add `ob boards` command

**Files:**
- Create: `cmd/boards.go`

**Step 1: Write the command**

Create `cmd/boards.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var boardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "List all shared boards",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardsDir := filepath.Join(obeyaHome, "boards")
		entries, err := os.ReadDir(boardsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No shared boards found. Run 'ob init --shared <name>' to create one.")
				return nil
			}
			return fmt.Errorf("failed to read boards directory: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No shared boards found. Run 'ob init --shared <name>' to create one.")
			return nil
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			boardFile := filepath.Join(boardsDir, entry.Name(), "board.json")
			if _, err := os.Stat(boardFile); err != nil {
				continue
			}

			s := store.NewJSONStore(filepath.Join(boardsDir, entry.Name()))
			board, err := s.LoadBoard()
			if err != nil {
				fmt.Printf("%-20s  (error: %v)\n", entry.Name(), err)
				continue
			}

			projectCount := len(board.Projects)
			noun := "projects"
			if projectCount == 1 {
				noun = "project"
			}
			fmt.Printf("%-20s  %d %s\n", entry.Name(), projectCount, noun)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(boardsCmd)
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: builds clean

**Step 3: Commit**

```bash
git add cmd/boards.go
git commit -m "feat: add ob boards command to list shared boards"
```

---

### Task 7: Add `ob board prune` subcommand

**Files:**
- Modify: `cmd/board.go` (add prune subcommand)

**Step 1: Read current board.go to understand structure**

Check: `cmd/board.go` for existing subcommands

**Step 2: Add prune subcommand**

Add to `cmd/board.go` (or create `cmd/board_prune.go`):

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var boardPruneCmd = &cobra.Command{
	Use:   "prune <board-name>",
	Short: "Remove dead project entries from a shared board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardName := args[0]

		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardDir := store.SharedBoardDir(obeyaHome, boardName)
		s := store.NewJSONStore(boardDir)
		if !s.BoardExists() {
			return fmt.Errorf("board %q not found", boardName)
		}

		pruned := 0
		err = s.Transaction(func(b *domain.Board) error {
			for name, proj := range b.Projects {
				if _, err := os.Stat(proj.LocalPath); os.IsNotExist(err) {
					delete(b.Projects, name)
					pruned++
					fmt.Printf("Removed dead project: %s (%s)\n", name, proj.LocalPath)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		if pruned == 0 {
			fmt.Println("No dead projects found.")
		} else {
			fmt.Printf("Pruned %d dead project(s).\n", pruned)
		}
		return nil
	},
}
```

Register under the board command's `init()`.

**Step 3: Build and verify**

Run: `go build ./...`
Expected: builds clean

**Step 4: Commit**

```bash
git add cmd/board_prune.go
git commit -m "feat: add ob board prune to remove dead project entries"
```

---

### Task 8: Auto-tag tasks with project name on creation

**Files:**
- Modify: `cmd/helpers.go` — add `getProjectName()` helper
- Modify: `cmd/create.go` — set `Item.Project` on task creation

**Step 1: Add getProjectName to helpers.go**

```go
func getProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	gitRoot, err := store.FindGitRoot(cwd)
	if err != nil {
		return ""
	}
	linkFile := filepath.Join(gitRoot, ".obeya-link")
	if _, err := os.Stat(linkFile); err != nil {
		return "" // not linked, no project tag
	}
	return resolveProjectName(gitRoot)
}
```

**Step 2: Modify create command to set project**

In `cmd/create.go`, after the `Item` is constructed, add:

```go
item.Project = getProjectName()
```

**Step 3: Run existing tests**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add cmd/helpers.go cmd/create.go
git commit -m "feat: auto-tag tasks with project name when linked to shared board"
```

---

### Task 9: Integration smoke test

**Files:**
- Create: `internal/store/shared_integration_test.go`

**Step 1: Write end-to-end test**

```go
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
```

**Step 2: Run test**

Run: `go test ./internal/store/ -run TestSharedBoard_EndToEnd -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/store/shared_integration_test.go
git commit -m "test: add end-to-end integration test for shared boards"
```

---

### Task 10: Update LoadBoard to initialize Projects map (nil safety)

**Files:**
- Modify: `internal/store/json_store.go:55-75`

**Step 1: Add nil-safety for Projects map**

In `LoadBoard()`, after the `Plans` nil check, add:

```go
if board.Projects == nil {
    board.Projects = make(map[string]*domain.LinkedProject)
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add internal/store/json_store.go
git commit -m "fix: initialize Projects map on LoadBoard for backwards compatibility"
```
