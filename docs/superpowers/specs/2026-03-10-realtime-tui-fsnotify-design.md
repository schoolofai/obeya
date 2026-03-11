# Real-time TUI Board Updates via fsnotify

**Date:** 2026-03-10
**Status:** Approved

## Problem

The Obeya TUI board requires pressing `r` to refresh. When agents update tasks via the `ob` CLI, the TUI doesn't reflect changes until manually refreshed.

## Solution

Use `fsnotify` to watch `.obeya/board.json` for write events. On change, trigger the existing `loadBoard()` reload path automatically.

## Data Flow

```
ob move T-3 done
  → Engine.MoveItem()
    → Store.Transaction()
      → writes board.json
        → fsnotify fires WRITE event
          → debounce (100ms)
            → boardFileChangedMsg sent to Bubble Tea
              → loadBoard() → boardLoadedMsg → View() re-renders
```

## Components

### New: `internal/tui/watcher.go`

- `boardFileChangedMsg` type
- `startWatcher(path string) tea.Cmd` — starts fsnotify watcher, returns Bubble Tea command that listens for events
- Debounce logic: 100ms window to coalesce rapid writes from a single transaction

### Modified: `internal/tui/app.go`

- Store watcher reference on `App` struct
- Wire `startWatcher()` into `Init()` as a `tea.Batch` alongside `loadBoard()`
- Handle `boardFileChangedMsg` in `Update()` — calls `loadBoard()` (same as pressing `r`)
- Close watcher on quit

### Modified: `go.mod`

- Add `github.com/fsnotify/fsnotify` dependency

## Edge Cases

- **Watcher fails to start**: manual `r` refresh still works, log warning
- **File deleted then recreated**: re-establish watch on error
- **Self-triggered reload** (TUI writes board.json): debounce handles gracefully, reload is idempotent

## What Stays the Same

- `r` key still works for manual refresh
- No changes to store, engine, or CLI commands
- No changes to board rendering logic
