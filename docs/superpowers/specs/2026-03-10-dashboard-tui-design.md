# Dashboard TUI — Design Spec

**Date:** 2026-03-10
**Status:** Approved

## Overview

Add a real-time dashboard view to the existing Obeya TUI, accessible by pressing `D` from the board view. The dashboard displays four metrics panels in a stacked layout: WIP status bar, velocity chart, cycle time distribution, and epic burndown.

## Architecture

### Integration

The dashboard is a new `stateDashboard` view state in the existing Bubble Tea app (`internal/tui/app.go`). It reuses the existing file watcher for real-time updates — when `board.json` changes, metrics recompute automatically.

**State machine:**
```
stateBoard ←→ stateDashboard (toggle with shift+D key)
         ↘ stateDetail, statePicker, stateInput (unchanged)
```

**Implementation notes:**
- Add `stateDashboard` to the `viewState` iota const block in `internal/tui/keys.go`
- Add `stateDashboard` cases to both `View()` and `handleKey()` in `app.go`
- The `D` keybinding uses shift+D (`"D"` in Bubble Tea key msg). Lowercase `d` is already bound to delete in the board view — no conflict.
- When `E` opens the epic picker (`statePicker`), the picker's own key handlers take over. `Tab` and other dashboard keys are inactive until the picker closes.
- `Esc` also returns to board view (consistent with other states)

### New Files

| File | Purpose |
|------|---------|
| `internal/metrics/metrics.go` | Shared metrics computation (extracted from `cmd/metrics.go`) |
| `internal/tui/dashboard.go` | Dashboard model, update handler, and view rendering |
| `internal/tui/charts.go` | Terminal chart rendering helpers (bar chart, horizontal bars, burndown) |

### Data Flow

1. Board loads via existing file watcher
2. On entering dashboard state, convert `board.Items` (map) to `[]*domain.Item` slice, then compute all metrics
3. `WIPStatus` takes `*domain.Board` directly (needs `Board.Columns` for limits and `Board.Items` for counts)
4. On `boardFileChangedMsg`, recompute metrics (real-time for free)
5. Rendering uses lipgloss for layout and colors — no external charting libraries

## Layout: Stacked with Status Bar

```
┌─────────────────────────────────────────────────────────────────┐
│ WIP  backlog 8/20  todo 4/5  in-prog 6/3  review 1/5 │ cycle: 3.2d · lead: 7.5d │
├─────────────────────────────────────────────────────────────────┤
│ VELOCITY (14d) — avg 2.3/day                                    │
│ ▇                                                               │
│ █  ▅        ▇                                                   │
│ █  █  ▃  █  █  ▅  ▂  █  ▆  ▃  ▇  ▄  █  ▅   ─── 3d avg        │
│ F27 F28 M01 M02 M03 M04 M05 M06 M07 M08 M09 M10              │
├────────────────────────────────┬────────────────────────────────┤
│ CYCLE TIME                     │ BURNDOWN: Epic #3 — 5/12 left │
│ backlog     ██████ 2.1d        │ █                              │
│ todo        ████ 1.4d          │ █ █                            │
│ in-progress ████████ 3.2d      │ █ █ █ ·                       │
│ review      ██ 0.8d            │ █ █ █ █ · █                   │
│                                │ █ █ █ █ █ █ ·                 │
├────────────────────────────────┴────────────────────────────────┤
│ D: board · Tab: panels · E: epic · R: refresh                  │
└─────────────────────────────────────────────────────────────────┘
```

## Panel Specifications

### WIP Status Bar (top, full width)

Single horizontal line showing per-column item counts vs limits.

- Format: `column: count/limit` for each non-done column
- Colors: green (<80% of limit), yellow (80–100%), red (>100%)
- Columns with `limit=0`: show count only, no color warning
- Right side: cycle time and lead time averages

### Velocity Chart (middle, full width)

14-day bar chart of completed items per day.

- One bar per day using Unicode block characters (`▁▂▃▄▅▆▇█`)
- X-axis: date labels (e.g., `M01`, `M08`)
- 3-day rolling average rendered as `─` line at appropriate height
- Title includes daily average: `VELOCITY (14d) — avg 2.3/day`
- Days with 0 completions: empty space (no bar)
- Counts all item types (epics, stories, tasks)

### Cycle Time Distribution (bottom left)

Horizontal bar chart of average dwell time per column.

- One row per column: backlog, todo, in-progress, review
- Bars use `█` characters, scaled proportionally to longest bar
- Value shown at end: e.g., `2.1d`

### Burndown Chart (bottom right)

Vertical bar chart of remaining children for a selected epic.

- Bars show remaining item count over time
- Ideal line rendered with `·` characters descending diagonally
- Title: `BURNDOWN: Epic #3 — 5/12 left`
- User selects epic via `E` key (opens existing PickerModel)
- No epics: panel shows "No epics on board"

## Keybindings

| Key | Action |
|-----|--------|
| `D` (shift+D) | Toggle back to board view |
| `Esc` | Return to board view (consistent with other states) |
| `Tab` | Cycle focus between panels (visual highlight) |
| `E` | Open epic picker for burndown chart |
| `R` | Force refresh metrics |

## Metrics Computation

### Extracted to `internal/metrics/metrics.go`

Existing computations moved from `cmd/metrics.go`:
- `Compute(items, now)` — dwell time, cycle time, lead time, throughput
- Helper functions: `processDwells`, `computeCycleTime`, `buildThroughput`, `avgDuration`, `formatDuration`

`cmd/metrics.go` becomes a thin CLI wrapper calling `metrics.Compute()`.

### New Computations

```go
// Per-day completion counts over last N days
func DailyVelocity(items []*domain.Item, days int, now time.Time) []DayCount
// DayCount: {Date time.Time, Count int}

// Computed from DailyVelocity output
func RollingAverage(days []DayCount, window int) []float64

// Remaining children of an epic over time
func EpicBurndown(epic *domain.Item, children []*domain.Item, now time.Time) []BurndownPoint
// BurndownPoint: {Date time.Time, Remaining int, Ideal float64}
//
// Algorithm: Start with total children count. Scan each child's History
// for "moved to done" events, collect timestamps, sort chronologically.
// At each done timestamp, decrement remaining count to produce a data point.
// Ideal line: linear from total to 0 over the span (epic created → now).

// Current count vs limit per column
func WIPStatus(board *domain.Board) []ColumnWIP
// ColumnWIP: {Name string, Count int, Limit int, Level string}
// Level: "ok" (<80%), "warn" (80-100%), "over" (>100%)
```

All computed from existing `Item.History` records and `Column.Limit` fields. No schema changes.

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| No completed items in 14 days | Velocity: "No completed items in last 14 days" |
| No epics on board | Burndown: "No epics on board", `E` key disabled |
| All column limits are 0 | WIP: show counts only, no color coding |
| Terminal < 80x24 | Show "Terminal too small" message |
| Terminal < 100 cols wide | Stack bottom panels vertically: cycle time on top, burndown below. Velocity chart height reduced proportionally to make room. |
| Item moved to done multiple times | `DailyVelocity` scans history independently — collects ALL "moved to done" timestamps, counting each as a separate velocity event. This is separate from the existing `processDwells` which only tracks the last done timestamp for cycle/lead time. |
| Epic with no children | Burndown: "0/0 — no children" |
| History gaps (items with no events in window) | Excluded from velocity; cycle time uses available data |

## Performance

- Metrics computed on board load and each file change (debounced 100ms by existing watcher)
- Complexity: O(items × history_length) — fine for boards up to thousands of items
- No caching needed for v1

## Testing

- Unit tests for `internal/metrics/` package: `Compute`, `DailyVelocity`, `RollingAverage`, `EpicBurndown`, `WIPStatus`
- Test edge cases: empty board, no done items, items with no history, multiple done transitions
- Verify `cmd/metrics.go` CLI output is unchanged after extraction (regression test)
- Chart rendering helpers tested with known inputs and expected string output
