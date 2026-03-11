# Real-time TUI Board Updates via fsnotify — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-refresh the TUI board when `board.json` changes on disk, eliminating the need for manual `r` refresh.

**Architecture:** Add an `fsnotify` file watcher that monitors the `.obeya/` directory (not the file directly, since `writeBoard()` uses atomic rename which replaces the inode). Filter events for `board.json` filename, debounce, and send a Bubble Tea message to trigger the existing `loadBoard()` path. The board directory path is passed from `cmd/tui.go` into `tui.NewApp()`.

**Tech Stack:** Go, fsnotify, Bubble Tea

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/tui/watcher.go` | Create | Directory watcher, filename filter, debounce, message types |
| `internal/tui/watcher_test.go` | Create | Unit tests for watcher behavior |
| `internal/tui/app.go` | Modify | Accept board path, wire watcher into Init/Update, cleanup on quit |
| `internal/store/store.go` | Modify | Add `BoardFilePath()` to Store interface |
| `internal/store/json_store.go` | Modify | Implement `BoardFilePath()` |
| `internal/engine/engine.go` | Modify | Expose `BoardFilePath()` |
| `cmd/tui.go` | Modify | Pass board file path to `tui.NewApp()` |
| `go.mod` / `go.sum` | Modify | Add fsnotify dependency |

---

## Task 1: Add fsnotify dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add fsnotify**

```bash
cd /Users/niladribose/code/obeya && go get github.com/fsnotify/fsnotify
```

- [ ] **Step 2: Verify it resolved**

```bash
grep fsnotify go.mod
```
Expected: `github.com/fsnotify/fsnotify vX.Y.Z`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add fsnotify for real-time TUI file watching"
```

---

## Task 2: Create the file watcher

**Files:**
- Create: `internal/tui/watcher.go`
- Create: `internal/tui/watcher_test.go`

**Important:** The watcher monitors the **directory** containing `board.json`, not the file itself. This is because `json_store.go:writeBoard()` uses atomic rename (`board.json.tmp` → `board.json`), which replaces the inode. Watching the file directly would cause the watcher to silently stop after the first write.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/watcher_test.go`:

```go
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSendsMessageOnFileChange(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond) // let watcher settle

	// Modify via atomic rename (same as json_store.go writeBoard)
	tmpFile := boardFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpFile, boardFile); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected file change notification, got timeout")
	}
}

func TestWatcherDebounceCoalescesRapidWrites(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond)

	// Write 5 times rapidly via atomic rename (same pattern as writeBoard)
	for i := 0; i < 5; i++ {
		tmpFile := boardFile + ".tmp"
		os.WriteFile(tmpFile, []byte(fmt.Sprintf(`{"version":%d}`, i)), 0644)
		os.Rename(tmpFile, boardFile)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire, then drain
	time.Sleep(200 * time.Millisecond)

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 1 {
		t.Fatalf("expected 1 debounced notification, got %d", count)
	}
}

func TestWatcherCloseStopsWatching(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	os.WriteFile(boardFile, []byte(`{}`), 0644)

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}

	ch := w.events()
	w.close()

	// Write after close — should NOT get notification
	time.Sleep(50 * time.Millisecond)
	os.WriteFile(boardFile, []byte(`{"version":99}`), 0644)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("should not receive events after close")
		}
		// channel closed — expected
	case <-time.After(300 * time.Millisecond):
		// success — no event received
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/niladribose/code/obeya && go test ./internal/tui/ -run TestWatcher -v
```
Expected: compilation errors — `newBoardWatcher` doesn't exist yet.

- [ ] **Step 3: Implement the watcher**

Create `internal/tui/watcher.go`:

```go
package tui

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// boardFileChangedMsg signals that the board file was modified on disk.
type boardFileChangedMsg struct{}

// watcherStartedMsg carries the initialized watcher (or nil on failure).
type watcherStartedMsg struct {
	watcher *boardWatcher
	err     error
}

const debounceInterval = 100 * time.Millisecond

// boardWatcher watches a directory for changes to a specific file.
// It watches the directory (not the file) because writeBoard() uses
// atomic rename (tmp → board.json), which replaces the inode.
type boardWatcher struct {
	watcher  *fsnotify.Watcher
	eventCh  chan struct{}
	done     chan struct{}
	closeOnce sync.Once
	fileName string // just the base name, e.g. "board.json"
}

func newBoardWatcher(boardFilePath string) (*boardWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(boardFilePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	bw := &boardWatcher{
		watcher:  w,
		eventCh:  make(chan struct{}, 1),
		done:     make(chan struct{}),
		fileName: filepath.Base(boardFilePath),
	}

	go bw.loop()
	return bw, nil
}

func (bw *boardWatcher) loop() {
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
		close(bw.eventCh)
	}()

	for {
		select {
		case event, ok := <-bw.watcher.Events:
			if !ok {
				return
			}
			// Only react to events on our target file
			if filepath.Base(event.Name) != bw.fileName {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					select {
					case bw.eventCh <- struct{}{}:
					default:
						// channel full, notification already pending
					}
				})
			}
		case _, ok := <-bw.watcher.Errors:
			if !ok {
				return
			}
		case <-bw.done:
			return
		}
	}
}

func (bw *boardWatcher) events() <-chan struct{} {
	return bw.eventCh
}

func (bw *boardWatcher) close() {
	bw.closeOnce.Do(func() {
		close(bw.done)
	})
	bw.watcher.Close()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/niladribose/code/obeya && go test ./internal/tui/ -run TestWatcher -v -count=1
```
Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/watcher.go internal/tui/watcher_test.go
git commit -m "feat: add directory-level file watcher with debounce for board.json changes"
```

---

## Task 3: Wire the watcher into the TUI App

**Files:**
- Modify: `internal/store/store.go` — add `BoardFilePath()` to interface
- Modify: `internal/store/json_store.go` — implement `BoardFilePath()`
- Modify: `internal/engine/engine.go` — expose `BoardFilePath()`
- Modify: `internal/tui/app.go:13-53` (struct + NewApp + Init)
- Modify: `internal/tui/app.go:55-73` (Update)
- Modify: `internal/tui/app.go:232-236` (quit handler)
- Modify: `cmd/tui.go:14-26` (pass board path)

- [ ] **Step 1: Add `BoardFilePath()` to Store interface and JSONStore**

In `internal/store/store.go`, add to the interface:

```go
// BoardFilePath returns the path to the board file on disk.
BoardFilePath() string
```

In `internal/store/json_store.go`, add after `BoardExists()` (after line 34):

```go
func (s *JSONStore) BoardFilePath() string {
	return s.boardFile
}
```

- [ ] **Step 2: Expose board file path from Engine**

In `internal/engine/engine.go`, add after `New()` (after line 17):

```go
// BoardFilePath returns the store's board file path (for file watching).
func (e *Engine) BoardFilePath() string {
	return e.store.BoardFilePath()
}
```

- [ ] **Step 3: Build to verify Store interface is satisfied**

```bash
cd /Users/niladribose/code/obeya && go build ./...
```
Expected: clean build. JSONStore is the only Store implementation (verified — no mocks exist).

- [ ] **Step 4: Update `App` struct and `NewApp` to accept board path**

In `internal/tui/app.go`, add fields to `App` struct:

```go
type App struct {
	engine    *engine.Engine
	board     *domain.Board
	boardPath string // path to board.json for file watching

	// Board navigation
	columns    []string
	cursorCol  int
	cursorRow  int
	collapsed  map[string]bool
	colScrollY map[int]int

	// State machine
	state     viewState
	prevState viewState

	// Sub-components
	detail     DetailModel
	picker     PickerModel
	input      InputModel
	confirmMsg string

	// Dimensions
	width  int
	height int

	watcher *boardWatcher
	err     error
}
```

Update `NewApp` signature:

```go
func NewApp(eng *engine.Engine, boardPath string) App {
	return App{
		engine:     eng,
		boardPath:  boardPath,
		collapsed:  make(map[string]bool),
		colScrollY: make(map[int]int),
		state:      stateBoard,
	}
}
```

- [ ] **Step 5: Update `Init()` to start the watcher**

Replace `Init()`:

```go
func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadBoard(), a.startWatching())
}

func (a App) startWatching() tea.Cmd {
	return func() tea.Msg {
		w, err := newBoardWatcher(a.boardPath)
		if err != nil {
			return watcherStartedMsg{watcher: nil, err: err}
		}
		return watcherStartedMsg{watcher: w}
	}
}
```

- [ ] **Step 6: Update `Update()` to handle watcher messages**

Add these cases to the `switch msg := msg.(type)` in `Update()`, after the `errMsg` case:

```go
case watcherStartedMsg:
	a.watcher = msg.watcher
	if msg.err != nil {
		a.err = fmt.Errorf("file watcher failed: %w (press r to refresh manually)", msg.err)
		return a, nil
	}
	return a, a.waitForFileChange()
case boardFileChangedMsg:
	return a, tea.Batch(a.loadBoard(), a.waitForFileChange())
```

Add the helper method (in `watcher.go`, not `app.go`, for cohesion):

```go
func (a App) waitForFileChange() tea.Cmd {
	return func() tea.Msg {
		if a.watcher == nil {
			return nil
		}
		_, ok := <-a.watcher.events()
		if !ok {
			return nil
		}
		return boardFileChangedMsg{}
	}
}
```

Note: `waitForFileChange` needs to be in `app.go` since it's a method on `App`. But the types (`boardFileChangedMsg`, `watcherStartedMsg`) stay in `watcher.go`.

- [ ] **Step 7: Close the watcher on quit**

Update the quit case in `handleBoardKey` (line 234):

```go
case "q", "ctrl+c":
	if a.watcher != nil {
		a.watcher.close()
	}
	return a, tea.Quit
```

Update the quit case in `handleDetailKey` (line 344):

```go
case "q":
	if a.watcher != nil {
		a.watcher.close()
	}
	return a, tea.Quit
```

Update the quit case in `handlePickerKey` (line 393):

```go
case "q":
	if a.watcher != nil {
		a.watcher.close()
	}
	return a, tea.Quit
```

- [ ] **Step 8: Update `cmd/tui.go` to pass board path**

```go
RunE: func(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}
	model := tui.NewApp(eng, eng.BoardFilePath())

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
},
```

- [ ] **Step 9: Build and verify compilation**

```bash
cd /Users/niladribose/code/obeya && go build ./...
```
Expected: clean build, no errors.

- [ ] **Step 10: Run all tests**

```bash
cd /Users/niladribose/code/obeya && go test ./... -count=1
```
Expected: all tests pass.

- [ ] **Step 11: Manual smoke test**

1. Run `ob tui` in one terminal
2. In another terminal, run `ob create task "test auto-refresh"` and `ob move <item> <column>`
3. Verify the TUI updates within ~200ms without pressing `r`
4. Verify pressing `r` still works for manual refresh
5. Verify TUI-initiated actions (press `m` to move) don't cause visible flicker (self-triggered reload is idempotent)
6. Verify `q` exits cleanly (no hung process)

- [ ] **Step 12: Commit**

```bash
git add internal/store/store.go internal/store/json_store.go internal/engine/engine.go internal/tui/app.go internal/tui/watcher.go cmd/tui.go
git commit -m "feat: wire fsnotify watcher into TUI for real-time board updates"
```
