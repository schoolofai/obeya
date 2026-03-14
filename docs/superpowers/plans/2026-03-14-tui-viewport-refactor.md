# TUI Viewport Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual line-slicing scroll logic with per-column `viewport.Model` components, split oversized files by responsibility, and restore the scrollbar — producing a maintainable TUI architecture that scales.

**Architecture:** Each Kanban column becomes a `ColumnModel` that owns a `viewport.Model` for scrolling/clipping. Card rendering is extracted to its own file. The root `App` delegates key events to the active column. All rendering uses pre-padded content (no `lipgloss.Width()` on multi-line content) to avoid the blank-line bug permanently.

**Tech Stack:** Go, Bubble Tea v1.3, Bubbles viewport v1.0, lipgloss v1.1, teatest

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `internal/tui/app.go` | Root model: state machine, view routing, top-level key dispatch | Modify — remove column rendering/scrolling logic, delegate to ColumnModel |
| `internal/tui/column.go` | **NEW** — ColumnModel wrapping viewport.Model, per-column cursor, card rendering orchestration | Create |
| `internal/tui/card.go` | **NEW** — renderCard, renderGroupedCards, wrapText, padToWidth, renderDescription | Create (extract from board.go) |
| `internal/tui/board.go` | Board-level layout: renderBoard joins columns, help bar, header | Modify — slim down to layout-only |
| `internal/tui/styles.go` | Style definitions | No change |
| `internal/tui/column_test.go` | **NEW** — ColumnModel unit tests | Create |
| `internal/tui/card_test.go` | **NEW** — card rendering tests (moved from board_test.go) | Create (rename) |
| `internal/tui/app_test.go` | Integration tests with teatest | Modify — update for new architecture |

## Chunk 1: Extract card rendering to `card.go`

Pure extraction — no behavior change. Move card rendering functions out of `board.go` into their own file.

### Task 1: Extract card rendering functions

**Files:**
- Create: `internal/tui/card.go`
- Create: `internal/tui/card_test.go`
- Modify: `internal/tui/board.go` — remove extracted functions
- Modify: `internal/tui/board_test.go` — remove card-specific tests

**Functions to move to `card.go`:**
- `renderCard` (method on App)
- `renderGroupedCards` (method on App)
- `renderDescription` (method on App)
- `wrapText`
- `padToWidth`
- `truncate`
- `parentBadge` (method on App)
- `isItemAtCursor` (method on App)
- `isEpicGroupAtCursor` (method on App)
- `findEpicAncestor`
- `isCollapsedChild` (method on App)

- [ ] **Step 1: Create `card.go` with all card rendering functions**

Move the functions listed above from `board.go` to `card.go`. Keep the same package (`tui`), same receiver types, same signatures. No logic changes.

```go
// internal/tui/card.go
package tui

// All functions moved verbatim from board.go:
// renderCard, renderGroupedCards, renderDescription,
// wrapText, padToWidth, truncate, parentBadge,
// isItemAtCursor, isEpicGroupAtCursor, findEpicAncestor,
// isCollapsedChild
```

- [ ] **Step 2: Create `card_test.go` with card-specific tests**

Move `wrapText` tests from `board_test.go` to `card_test.go`. Add the `TestTUI_RenderCard_Isolation` test from `app_test.go` (or keep it there — it uses the full App).

- [ ] **Step 3: Remove moved functions from `board.go` and `board_test.go`**

Delete only the functions that now live in `card.go`. Keep in `board.go`: `renderBoard`, `renderColumn`, `buildScrollbar`, `columnWidth`, `cyclePriority`, `userNames`, `itemPickerLabels`, `resolveUserName`, `visibleItemsInColumn`, `renderOrderItems`, `renderBoardWithOverlay`.

- [ ] **Step 4: Run tests to verify no regression**

Run: `go test ./internal/tui/ -v -timeout 60s`
Expected: All tests PASS. No behavior change.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/card.go internal/tui/card_test.go internal/tui/board.go internal/tui/board_test.go
git commit -m "refactor(tui): extract card rendering to card.go"
```

## Chunk 2: Create ColumnModel with viewport.Model

The core architectural change. Each column becomes a model that owns a `viewport.Model`.

### Task 2: Define ColumnModel struct and constructor

**Files:**
- Create: `internal/tui/column.go`
- Create: `internal/tui/column_test.go`

- [ ] **Step 1: Write failing test for ColumnModel construction**

```go
// internal/tui/column_test.go
package tui

import (
    "testing"
)

func TestColumnModel_New(t *testing.T) {
    col := NewColumnModel("backlog", 21, 35)
    if col.Name != "backlog" {
        t.Errorf("expected name 'backlog', got %q", col.Name)
    }
    if col.viewport.Width != 21 {
        t.Errorf("expected viewport width 21, got %d", col.viewport.Width)
    }
    if col.viewport.Height != 35 {
        t.Errorf("expected viewport height 35, got %d", col.viewport.Height)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestColumnModel_New -v`
Expected: FAIL — `NewColumnModel` undefined.

- [ ] **Step 3: Implement ColumnModel**

```go
// internal/tui/column.go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/lipgloss"
)

// ColumnModel represents a single Kanban column with a scrollable viewport.
type ColumnModel struct {
    Name     string
    viewport viewport.Model
    cursor   int   // selected item index within this column
    active   bool  // true when this column has focus
}

// NewColumnModel creates a column with a viewport sized to contentWidth x viewHeight.
func NewColumnModel(name string, contentWidth, viewHeight int) ColumnModel {
    vp := viewport.New(contentWidth, viewHeight)
    vp.MouseWheelEnabled = true
    vp.MouseWheelDelta = 3
    // Disable viewport's own key handling — App handles keys
    vp.KeyMap = viewport.KeyMap{}
    return ColumnModel{
        Name:     name,
        viewport: vp,
    }
}

// SetSize updates the viewport dimensions (called on terminal resize).
func (c *ColumnModel) SetSize(contentWidth, viewHeight int) {
    c.viewport.Width = contentWidth
    c.viewport.Height = viewHeight
}

// SetContent sets the rendered card content for this column.
// The viewport handles clipping and scroll offset.
func (c *ColumnModel) SetContent(content string) {
    c.viewport.SetContent(content)
}

// ScrollToLine sets the viewport's Y offset to show a specific line.
// Clamps to valid range automatically.
func (c *ColumnModel) ScrollToLine(line int) {
    c.viewport.SetYOffset(line)
}

// View renders the column: header + viewport + styled border.
func (c ColumnModel) View(itemCount int) string {
    // Header
    count := fmt.Sprintf(" (%d)", itemCount)
    var header string
    if c.active {
        header = activeColHeader.Render(strings.ToUpper(c.Name) + count)
    } else {
        header = inactiveColHeader.Render(strings.ToUpper(c.Name) + count)
    }
    header = padToWidth(header, c.viewport.Width)

    // Viewport content (already clipped to viewHeight)
    vpView := c.viewport.View()

    // Scrollbar overlay on the right edge of viewport content
    vpView = c.overlayScrollbar(vpView)

    content := header + "\n" + vpView
    if c.active {
        return activeColumnStyle.Render(content)
    }
    return columnStyle.Render(content)
}

// overlayScrollbar replaces the last character of each viewport line with
// a scrollbar indicator, keeping the column width stable.
func (c ColumnModel) overlayScrollbar(vpView string) string {
    total := c.viewport.TotalLineCount()
    viewH := c.viewport.Height
    if total <= viewH || viewH <= 0 {
        return vpView // no scroll needed
    }

    lines := strings.Split(vpView, "\n")
    offset := c.viewport.YOffset

    thumbH := (viewH * viewH) / total
    if thumbH < 1 {
        thumbH = 1
    }
    maxOffset := total - viewH
    thumbPos := 0
    if maxOffset > 0 {
        thumbPos = (offset * (viewH - thumbH)) / maxOffset
    }
    if thumbPos < 0 {
        thumbPos = 0
    }
    if thumbPos+thumbH > viewH {
        thumbPos = viewH - thumbH
    }

    thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
    trackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

    for i := range lines {
        if i >= viewH {
            break
        }
        var indicator string
        if i >= thumbPos && i < thumbPos+thumbH {
            indicator = thumbStyle.Render("┃")
        } else {
            indicator = trackStyle.Render("│")
        }
        // Replace trailing space with scrollbar char
        line := lines[i]
        runes := []rune(line)
        if len(runes) > 0 {
            runes = runes[:len(runes)-1]
        }
        lines[i] = string(runes) + indicator
    }
    return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestColumnModel -v`
Expected: PASS

- [ ] **Step 5: Add test for SetContent + View**

```go
func TestColumnModel_View(t *testing.T) {
    col := NewColumnModel("todo", 20, 5)
    col.active = true
    col.SetContent("line 1\nline 2\nline 3")

    view := col.View(3)
    if !strings.Contains(view, "TODO") {
        t.Error("expected TODO header in view")
    }
    if !strings.Contains(view, "line 1") {
        t.Error("expected content in view")
    }
}

func TestColumnModel_ScrollbarAppears(t *testing.T) {
    col := NewColumnModel("backlog", 20, 3)
    // Content longer than viewport
    lines := make([]string, 10)
    for i := range lines {
        lines[i] = fmt.Sprintf("card %d content here", i)
    }
    col.SetContent(strings.Join(lines, "\n"))

    view := col.View(10)
    // Scrollbar uses ┃ and │ characters
    if !strings.Contains(view, "┃") && !strings.Contains(view, "│") {
        t.Error("expected scrollbar indicators in view when content overflows")
    }
}
```

- [ ] **Step 6: Run tests and commit**

Run: `go test ./internal/tui/ -v -timeout 60s`
Expected: All tests PASS

```bash
git add internal/tui/column.go internal/tui/column_test.go
git commit -m "feat(tui): add ColumnModel with viewport and scrollbar"
```

## Chunk 3: Wire ColumnModel into App

Replace `colScrollY` with `[]ColumnModel`. Update `renderBoard` to delegate to column views. Update key handlers.

### Task 3: Replace App's column state with ColumnModel slice

**Files:**
- Modify: `internal/tui/app.go` — replace `colScrollY`, update init/resize/key handling
- Modify: `internal/tui/board.go` — simplify `renderBoard`, remove `renderColumn`, `buildScrollbar`, `contentViewHeight`

- [ ] **Step 1: Update App struct**

In `app.go`, replace:
```go
colScrollY map[int]int // per-column scroll offsets
```
with:
```go
colModels []ColumnModel // per-column viewport models
```

Remove the `colScrollY: make(map[int]int)` from `NewApp`.

- [ ] **Step 2: Initialize columns on boardLoadedMsg**

In the `boardLoadedMsg` handler in `Update()`, after setting `a.columns`, create column models:

```go
case boardLoadedMsg:
    a.board = msg.board
    a.columns = extractColumns(msg.board)
    a.initColumnModels()
    a.clampCursor()
    // ... existing dashboard logic
```

Add helper:
```go
func (a *App) initColumnModels() {
    w := a.columnWidth()
    viewH := a.contentViewHeight()
    a.colModels = make([]ColumnModel, len(a.columns))
    for i, name := range a.columns {
        a.colModels[i] = NewColumnModel(name, w, viewH)
    }
    if a.cursorCol < len(a.colModels) {
        a.colModels[a.cursorCol].active = true
    }
}
```

- [ ] **Step 3: Handle WindowSizeMsg — resize all column viewports**

Update the existing handler:
```go
case tea.WindowSizeMsg:
    a.width = msg.Width
    a.height = msg.Height
    w := a.columnWidth()
    viewH := a.contentViewHeight()
    for i := range a.colModels {
        a.colModels[i].SetSize(w, viewH)
    }
    return a, nil
```

- [ ] **Step 4: Update renderBoard to use ColumnModel.View()**

Simplify `renderBoard` in `board.go`:

```go
func (a App) renderBoard() string {
    var cols []string
    for i, colName := range a.columns {
        items := a.visibleItemsInColumn(i)

        // Count native items for the header
        nativeCount := 0
        for _, it := range items {
            if it.Status == colName {
                nativeCount++
            }
        }

        // Render cards as content string
        cardViews := a.renderGroupedCards(items, i)
        cardContent := ""
        if len(cardViews) > 0 {
            allCards := strings.Join(cardViews, "\n")
            // Pad each line to column width
            w := a.columnWidth()
            cardLines := strings.Split(allCards, "\n")
            for j, line := range cardLines {
                cardLines[j] = padToWidth(line, w)
            }
            cardContent = strings.Join(cardLines, "\n")
        }

        // Set content on the column's viewport
        if i < len(a.colModels) {
            a.colModels[i].SetContent(cardContent)
            cols = append(cols, a.colModels[i].View(nativeCount))
        }
    }

    board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

    header := fmt.Sprintf("  Obeya Board: %s", a.board.Name)
    help := helpStyle.Render(
        "  h/l:columns  j/k:items  v:desc  m:move  a:assign  c:create  d:delete  " +
            "p:priority  Enter:detail  Space:collapse  /:search  r:reload  q:quit",
    )

    return header + "\n" + board + "\n" + help
}
```

- [ ] **Step 5: Update scrollToSelected to use ColumnModel.ScrollToLine()**

Replace `a.colScrollY[a.cursorCol] = offset` with `a.colModels[a.cursorCol].ScrollToLine(offset)`. Remove all references to `colScrollY`.

```go
func (a *App) scrollToSelected() {
    if a.height <= 0 || a.board == nil {
        return
    }
    if a.cursorCol >= len(a.colModels) {
        return
    }

    item := a.selectedItem()
    if item == nil {
        a.colModels[a.cursorCol].ScrollToLine(0)
        return
    }

    items := a.visibleItemsInColumn(a.cursorCol)
    cardViews := a.renderGroupedCards(items, a.cursorCol)
    if len(cardViews) == 0 {
        a.colModels[a.cursorCol].ScrollToLine(0)
        return
    }
    cardContent := strings.Join(cardViews, "\n")
    cardLines := strings.Split(cardContent, "\n")

    viewH := a.contentViewHeight()
    if viewH <= 0 || len(cardLines) <= viewH {
        a.colModels[a.cursorCol].ScrollToLine(0)
        return
    }

    // Find the selected item's marker line
    marker := fmt.Sprintf("#%d ", item.DisplayNum)
    cardTop := -1
    for i, line := range cardLines {
        if strings.Contains(line, marker) {
            cardTop = i
            break
        }
    }
    if cardTop < 0 {
        return
    }

    offset := a.colModels[a.cursorCol].viewport.YOffset

    if cardTop < offset+2 {
        offset = cardTop - 2
    }
    if cardTop > offset+viewH-6 {
        offset = cardTop - viewH + 6
    }
    if offset < 0 {
        offset = 0
    }

    a.colModels[a.cursorCol].ScrollToLine(offset)
}
```

- [ ] **Step 6: Update key handlers — set active column on navigation**

In `handleBoardKey`, when moving columns (h/l/tab), update `active` flag:

```go
case "h", "left":
    if a.cursorCol > 0 {
        a.collapseDescription()
        a.colModels[a.cursorCol].active = false
        a.cursorCol--
        a.colModels[a.cursorCol].active = true
        a.cursorRow = 0
        a.clampCursor()
        a.scrollToSelected()
    }
case "l", "right":
    if a.cursorCol < len(a.columns)-1 {
        a.collapseDescription()
        a.colModels[a.cursorCol].active = false
        a.cursorCol++
        a.colModels[a.cursorCol].active = true
        a.cursorRow = 0
        a.clampCursor()
        a.scrollToSelected()
    }
case "tab":
    a.collapseDescription()
    if len(a.columns) > 0 {
        a.colModels[a.cursorCol].active = false
        a.cursorCol = (a.cursorCol + 1) % len(a.columns)
        a.colModels[a.cursorCol].active = true
        a.cursorRow = 0
        a.clampCursor()
        a.scrollToSelected()
    }
```

- [ ] **Step 7: Remove old code from board.go**

Delete from `board.go`:
- `renderColumn` function (replaced by ColumnModel.View)
- `buildScrollbar` function (replaced by ColumnModel.overlayScrollbar)
- Old `renderBoard` (replaced in step 4)

Keep in `board.go`:
- `columnWidth`
- `visibleItemsInColumn`, `renderOrderItems`
- `renderBoardWithOverlay`
- `cyclePriority`, `userNames`, `itemPickerLabels`, `resolveUserName`

- [ ] **Step 8: Remove contentViewHeight from app.go if it moved**

`contentViewHeight` stays in `app.go` since it depends on `a.height`.

- [ ] **Step 9: Run all tests**

Run: `go test ./internal/tui/ -v -timeout 60s`
Expected: All tests PASS

- [ ] **Step 10: Run real board teatest**

Run: `go test -v -run TestTUI_RealBoard ./internal/tui/ -timeout 30s`
Expected: PASS with 40-line output, all columns aligned, scrollbar visible on overflow columns.

- [ ] **Step 11: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go internal/tui/column.go
git commit -m "refactor(tui): wire ColumnModel with viewport into App"
```

## Chunk 4: Update and expand teatest coverage

Ensure all existing features work with the new architecture and add regression tests.

### Task 4: Update and add teatest tests

**Files:**
- Modify: `internal/tui/app_test.go`

- [ ] **Step 1: Verify existing tests still pass**

Run: `go test -v -run TestTUI ./internal/tui/ -timeout 60s`
All existing `TestTUI_*` tests should pass without modification.

- [ ] **Step 2: Add scrollbar visibility test**

```go
func TestTUI_ScrollbarVisible(t *testing.T) {
    boardFile := "/Users/niladribose/code/obeya/.obeya/board.json"
    if _, err := os.Stat(boardFile); err != nil {
        t.Skip("real board.json not found")
    }
    s := store.NewJSONStore("/Users/niladribose/code/obeya")
    eng := engine.New(s)
    app := NewApp(eng, boardFile)
    tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
    time.Sleep(500 * time.Millisecond)

    screen := getScreen(t, tm)
    // Backlog has 18 items — should show scrollbar
    if !strings.Contains(screen, "┃") {
        t.Error("expected scrollbar indicator in overflow column")
    }
}
```

- [ ] **Step 3: Add description accordion test**

```go
func TestTUI_DescriptionAccordion(t *testing.T) {
    boardFile, eng := testBoard(t)
    // Add a description to item-2
    // ... (modify testBoard to include descriptions)
    tm := startAndWait(t, eng, boardFile, 120, 40)

    // Press 'v' to expand description
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
    time.Sleep(50 * time.Millisecond)

    screen := getScreen(t, tm)
    t.Logf("\n=== DESCRIPTION EXPANDED ===\n%s", screen)
    if !strings.Contains(screen, "▼ description") {
        t.Error("expected expanded description indicator")
    }
}
```

- [ ] **Step 4: Add terminal resize test**

```go
func TestTUI_Resize(t *testing.T) {
    boardFile, eng := testBoard(t)
    app := NewApp(eng, boardFile)
    tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
    time.Sleep(200 * time.Millisecond)

    // Resize to narrow terminal
    tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
    time.Sleep(100 * time.Millisecond)

    screen := getScreen(t, tm)
    lines := strings.Split(screen, "\n")
    if len(lines) > 26 { // 24 + small margin
        t.Errorf("expected ~24 lines after resize to 80x24, got %d", len(lines))
    }
}
```

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -timeout 120s`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app_test.go
git commit -m "test(tui): add viewport scrollbar, accordion, and resize tests"
```

## Chunk 5: Clean up and verify

### Task 5: Final cleanup and verification

**Files:**
- All TUI files — verify line counts, remove dead code

- [ ] **Step 1: Verify file sizes are reasonable**

Target line counts after refactor:
- `app.go`: ~400 lines (down from 658)
- `board.go`: ~200 lines (down from 685)
- `column.go`: ~150 lines (new)
- `card.go`: ~350 lines (extracted from board.go)

Run: `wc -l internal/tui/*.go | sort -rn`

- [ ] **Step 2: Search for dead code**

Run: `grep -n 'colScrollY\|buildScrollbar' internal/tui/*.go`
Expected: No matches (all references removed)

- [ ] **Step 3: Run full test suite one final time**

Run: `go test ./... -timeout 120s`
Expected: All tests PASS

- [ ] **Step 4: Run the TUI manually to visual-verify**

Run: `go run . tui`
Verify:
- All 5 columns visible and aligned
- h/l moves between columns
- j/k scrolls within columns
- Scrollbar appears on overflow columns
- Full titles wrap correctly
- `v` toggles description accordion
- `q` quits cleanly

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "refactor(tui): complete viewport architecture cleanup"
```
