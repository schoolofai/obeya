# TUI Card Layout Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace truncated card titles with full word-wrapping and add an accordion-style expandable description panel on the selected card.

**Architecture:** Two new functions (`wrapText`, `renderDescription`) in `board.go`, two new state fields (`descExpanded`, `descScrollY`) in `app.go`, and dynamic card width in `styles.go`. The existing `renderCard` function is modified to use wrapping and conditionally render the description accordion.

**Tech Stack:** Go, Bubble Tea, Lipgloss

**Spec:** `docs/superpowers/specs/2026-03-12-tui-card-layout-design.md`

---

## Chunk 1: Word Wrapping

### Task 1: Add `wrapText` function with tests

**Files:**
- Modify: `internal/tui/board.go` — add `wrapText` function after `truncate` (end of file)
- Create: `internal/tui/board_test.go` — unit tests for `wrapText`

- [ ] **Step 1: Write the failing tests for `wrapText`**

```go
// internal/tui/board_test.go
package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWrapText_ShortString(t *testing.T) {
	lines := wrapText("hello", 20)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [\"hello\"], got %v", lines)
	}
}

func TestWrapText_ExactWidth(t *testing.T) {
	lines := wrapText("hello", 5)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [\"hello\"], got %v", lines)
	}
}

func TestWrapText_WordBoundary(t *testing.T) {
	lines := wrapText("hello world foo", 11)
	if len(lines) != 2 || lines[0] != "hello world" || lines[1] != "foo" {
		t.Errorf("expected [\"hello world\", \"foo\"], got %v", lines)
	}
}

func TestWrapText_LongWord(t *testing.T) {
	lines := wrapText("abcdefghij", 5)
	if len(lines) != 2 || lines[0] != "abcde" || lines[1] != "fghij" {
		t.Errorf("expected [\"abcde\", \"fghij\"], got %v", lines)
	}
}

func TestWrapText_MultipleWraps(t *testing.T) {
	lines := wrapText("one two three four five", 9)
	// "one two" (7), "three" (5), "four five" (9)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}

func TestWrapText_Empty(t *testing.T) {
	lines := wrapText("", 20)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("expected [\"\"], got %v", lines)
	}
}

func TestWrapText_SingleSpaces(t *testing.T) {
	// Multiple spaces between words should not produce empty tokens
	lines := wrapText("a  b  c", 10)
	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "a") || !strings.Contains(joined, "b") || !strings.Contains(joined, "c") {
		t.Errorf("expected all words present, got %v", lines)
	}
}

func TestWrapText_Unicode(t *testing.T) {
	// Multi-byte characters should wrap at rune boundaries, not byte boundaries
	lines := wrapText("héllo wörld", 7)
	for i, line := range lines {
		if utf8.RuneCountInString(line) > 7 {
			t.Errorf("line %d exceeds maxWidth in runes: %q (%d runes)", i, line, utf8.RuneCountInString(line))
		}
	}
	joined := strings.Join(lines, " ")
	if joined != "héllo wörld" {
		t.Errorf("expected original text reconstructed, got %q", joined)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestWrapText -v`
Expected: FAIL — `wrapText` undefined

- [ ] **Step 3: Implement `wrapText` in `board.go`**

Add `"unicode/utf8"` to the imports in `board.go`, then add after the `truncate` function (end of file):

```go
// wrapText word-wraps s into lines of at most maxWidth runes.
// Breaks on word boundaries when possible; breaks mid-word only when
// a single word exceeds maxWidth. Uses rune count for correct Unicode handling.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	if utf8.RuneCountInString(s) <= maxWidth {
		return []string{s}
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	curLen := 0
	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if cur == "" {
			// First word on the line — may need mid-word break
			runes := []rune(word)
			for len(runes) > maxWidth {
				lines = append(lines, string(runes[:maxWidth]))
				runes = runes[maxWidth:]
			}
			cur = string(runes)
			curLen = len(runes)
			continue
		}
		if curLen+1+wordLen <= maxWidth {
			cur += " " + word
			curLen += 1 + wordLen
		} else {
			lines = append(lines, cur)
			// Start new line — may need mid-word break
			runes := []rune(word)
			for len(runes) > maxWidth {
				lines = append(lines, string(runes[:maxWidth]))
				runes = runes[maxWidth:]
			}
			cur = string(runes)
			curLen = len(runes)
		}
	}
	lines = append(lines, cur)
	return lines
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestWrapText -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/board.go internal/tui/board_test.go
git commit -m "feat(tui): add wrapText function for card title wrapping"
```

---

### Task 2: Replace title truncation with wrapping in `renderCard`

**Files:**
- Modify: `internal/tui/board.go:140-175` — rewrite `renderCard` to use `wrapText` (title wrapping only, no accordion yet)
- Modify: `internal/tui/styles.go:7-17` — remove hardcoded `Width(22)` from card styles

- [ ] **Step 1: Write a test for wrapped card rendering**

Add to `internal/tui/board_test.go`:

```go
func TestWrapText_TitleLikeString(t *testing.T) {
	// Simulates a card title that would be truncated at width 16
	title := "Implement automatic token refresh handling"
	lines := wrapText(title, 16)
	// Full title must be present across all lines
	joined := strings.Join(lines, " ")
	if joined != title {
		t.Errorf("wrapped lines should reconstruct title\nexpected: %q\ngot:      %q", title, joined)
	}
	// No line should exceed maxWidth
	for i, line := range lines {
		if len(line) > 16 {
			t.Errorf("line %d exceeds maxWidth: %q (%d chars)", i, line, len(line))
		}
	}
}
```

- [ ] **Step 2: Run test to verify it passes** (this tests `wrapText` which already works)

Run: `go test ./internal/tui/ -run TestWrapText_TitleLikeString -v`
Expected: PASS

- [ ] **Step 3: Remove hardcoded Width from card styles**

In `internal/tui/styles.go`, change lines 7-17 from:

```go
cardStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(0, 1).
    Width(22)

selectedCardStyle = lipgloss.NewStyle().
    Border(lipgloss.ThickBorder()).
    BorderForeground(lipgloss.Color("14")).
    Bold(true).
    Padding(0, 1).
    Width(22)
```

To:

```go
cardStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(0, 1)

selectedCardStyle = lipgloss.NewStyle().
    Border(lipgloss.ThickBorder()).
    BorderForeground(lipgloss.Color("14")).
    Bold(true).
    Padding(0, 1)
```

- [ ] **Step 4: Rewrite `renderCard` with title wrapping and dynamic width (no accordion yet)**

Replace `renderCard` in `board.go` (lines 140-175) with:

```go
func (a App) renderCard(item *domain.Item, selected bool) string {
	w := a.columnWidth()
	contentW := w - 4 // border(2) + padding(2)
	if contentW < 10 {
		contentW = 10
	}

	// Title: wrap instead of truncate
	prefix := fmt.Sprintf("#%d ", item.DisplayNum)
	titleMax := contentW - len(prefix)
	if titleMax < 4 {
		titleMax = 4
	}
	// Wrap entire title at titleMax — this equals the available text width for
	// both the first line (contentW - prefix) and continuation lines
	// (contentW - indent), since indent == len(prefix) spaces.
	titleLines := wrapText(item.Title, titleMax)
	line1 := prefix + titleLines[0]

	lines := []string{line1}
	indent := strings.Repeat(" ", len(prefix))
	for _, tl := range titleLines[1:] {
		lines = append(lines, indent+tl)
	}

	typLabel := typeStyle(string(item.Type)).Render(string(item.Type))
	line2 := fmt.Sprintf("%s %s", typLabel, priorityIndicator(string(item.Priority)))
	lines = append(lines, line2)

	badge := a.parentBadge(item)
	if badge != "" {
		lines = append(lines, badge)
	}

	var metaParts []string
	if item.Assignee != "" {
		name := resolveUserName(a.board, item.Assignee)
		metaParts = append(metaParts, assigneeStyle.Render("@"+name))
	}
	if len(item.BlockedBy) > 0 {
		metaParts = append(metaParts, blockedStyle.Render("[!]"))
	}
	if len(metaParts) > 0 {
		lines = append(lines, strings.Join(metaParts, " "))
	}

	content := strings.Join(lines, "\n")
	if selected {
		return selectedCardStyle.Width(w - 2).Render(content)
	}
	return cardStyle.Width(w - 2).Render(content)
}
```

- [ ] **Step 5: Run all tests to verify nothing is broken**

Run: `go test ./internal/tui/ -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/board.go internal/tui/board_test.go internal/tui/styles.go
git commit -m "feat(tui): replace title truncation with full word-wrapping"
```

---

## Chunk 2: Description Accordion

### Task 3: Add accordion rendering, state fields, `renderDescription`, and styles

This task adds all the pieces needed for the accordion to compile as one atomic unit: state fields in `App`, the `renderDescription` method, new styles, and the accordion section in `renderCard`.

**Files:**
- Modify: `internal/tui/app.go:13-42` — add `descExpanded` and `descScrollY` fields to `App` struct
- Modify: `internal/tui/board.go` — add `renderDescription` method after `wrapText`; add accordion rendering to `renderCard`
- Modify: `internal/tui/styles.go` — add `descIndicatorStyle`, `descStyle`, `descScrollHint`
- Modify: `internal/tui/board_test.go` — unit tests for `renderDescription`

- [ ] **Step 1: Add state fields to `App` struct**

In `app.go`, add after `colScrollY` (line 23):

```go
	// Description accordion
	descExpanded string // item ID whose description is expanded, "" if none
	descScrollY  int    // scroll offset within expanded description
```

- [ ] **Step 2: Add description styles to `styles.go`**

Add after the `crossColBadge` line (line 72):

```go
// Description accordion
descIndicatorStyle = lipgloss.NewStyle().Faint(true)
descStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
descScrollHint     = lipgloss.NewStyle().Faint(true)
```

- [ ] **Step 3: Write failing tests for `renderDescription`**

Add to `internal/tui/board_test.go`:

```go
func TestRenderDescription_Short(t *testing.T) {
	a := App{}
	lines := a.renderDescription("Short desc", 20, 0, 5)
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %v", len(lines), lines)
	}
}

func TestRenderDescription_ExactlyMaxLines(t *testing.T) {
	a := App{}
	desc := "line one\nline two\nline three\nline four\nline five"
	lines := a.renderDescription(desc, 40, 0, 5)
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d: %v", len(lines), lines)
	}
}

func TestRenderDescription_OverMaxLines_ShowsScrollDown(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven"
	lines := a.renderDescription(desc, 40, 0, 5)
	// 5 content lines + 1 scroll indicator
	if len(lines) != 6 {
		t.Errorf("expected 6 lines (5 content + 1 indicator), got %d: %v", len(lines), lines)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "\u25be") { // ▾
		t.Errorf("expected down scroll indicator, got: %q", last)
	}
}

func TestRenderDescription_ScrolledMiddle_ShowsBothIndicators(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight"
	lines := a.renderDescription(desc, 40, 2, 5)
	// 1 up indicator + 5 content + 1 down indicator
	if len(lines) != 7 {
		t.Errorf("expected 7 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "\u25b4") { // ▴
		t.Errorf("expected up scroll indicator on first line, got: %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "\u25be") { // ▾
		t.Errorf("expected down scroll indicator on last line, got: %q", lines[len(lines)-1])
	}
}

func TestRenderDescription_ScrolledToEnd_ShowsUpOnly(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven"
	// 7 lines total, scrollY=2 means lines 2-6 visible (indices 2,3,4,5,6)
	lines := a.renderDescription(desc, 40, 2, 5)
	if !strings.Contains(lines[0], "\u25b4") { // ▴
		t.Errorf("expected up indicator, got: %q", lines[0])
	}
	last := lines[len(lines)-1]
	if strings.Contains(last, "\u25be") { // ▾
		t.Errorf("should NOT have down indicator when scrolled to end, got: %q", last)
	}
}

func TestRenderDescription_Empty(t *testing.T) {
	a := App{}
	lines := a.renderDescription("", 20, 0, 5)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty desc, got %d", len(lines))
	}
}

func TestRenderDescription_WrapsLongLines(t *testing.T) {
	a := App{}
	desc := "This is a very long description line that should be wrapped to fit"
	lines := a.renderDescription(desc, 15, 0, 5)
	for i, line := range lines {
		// Strip ANSI escape codes for length check (lipgloss adds them)
		plain := stripAnsi(line)
		if len(plain) > 15 {
			t.Errorf("line %d exceeds maxWidth: %q (%d chars)", i, plain, len(plain))
		}
	}
}

// stripAnsi removes ANSI escape sequences for length testing.
func stripAnsi(s string) string {
	result := ""
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result += string(r)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestRenderDescription -v`
Expected: FAIL — `renderDescription` undefined

- [ ] **Step 5: Implement `renderDescription`**

Add to `board.go` after `wrapText`:

```go
// renderDescription word-wraps and renders a description within a scrollable
// viewport of maxLines content lines. Returns the rendered lines including
// scroll indicators (outside the viewport — they don't consume content lines).
func (a App) renderDescription(desc string, maxWidth int, scrollY int, maxLines int) []string {
	if desc == "" {
		return nil
	}

	// Split on newlines first, then wrap each paragraph
	paragraphs := strings.Split(desc, "\n")
	var allLines []string
	for _, p := range paragraphs {
		if p == "" {
			allLines = append(allLines, "")
			continue
		}
		wrapped := wrapText(p, maxWidth)
		allLines = append(allLines, wrapped...)
	}

	totalLines := len(allLines)
	if totalLines == 0 {
		return nil
	}

	// No scrolling needed
	if totalLines <= maxLines {
		styled := make([]string, len(allLines))
		for i, l := range allLines {
			styled[i] = descStyle.Render(l)
		}
		return styled
	}

	// Clamp scrollY
	maxScroll := totalLines - maxLines
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	// Slice the viewport
	end := scrollY + maxLines
	if end > totalLines {
		end = totalLines
	}
	visible := allLines[scrollY:end]

	styled := make([]string, 0, len(visible)+2)

	// Up indicator (outside viewport)
	if scrollY > 0 {
		hint := fmt.Sprintf("%*s", maxWidth, "\u25b4 J/K")
		styled = append(styled, descScrollHint.Render(hint))
	}

	for _, l := range visible {
		styled = append(styled, descStyle.Render(l))
	}

	// Down indicator (outside viewport)
	if end < totalLines {
		hint := fmt.Sprintf("%*s", maxWidth, "\u25be J/K")
		styled = append(styled, descScrollHint.Render(hint))
	}

	return styled
}
```

- [ ] **Step 6: Add accordion rendering to `renderCard`**

In the `renderCard` function (modified in Task 2), add the accordion section before the final `content := strings.Join(lines, "\n")` line. Insert this block after the `metaParts` section:

```go
	// Accordion indicator — only on selected card with non-empty description
	if selected && item.Description != "" {
		if a.descExpanded == item.ID {
			lines = append(lines, descIndicatorStyle.Render("\u25bc description"))
			sep := strings.Repeat("\u2500", contentW)
			lines = append(lines, lipgloss.NewStyle().Faint(true).Render(sep))
			lines = append(lines, a.renderDescription(item.Description, contentW, a.descScrollY, 5)...)
		} else {
			lines = append(lines, descIndicatorStyle.Render("\u25b6 description"))
		}
	}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestRenderDescription -v`
Expected: PASS

Run: `go test ./internal/tui/ -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go internal/tui/board_test.go internal/tui/styles.go
git commit -m "feat(tui): add description accordion rendering with scrollable viewport"
```

---

### Task 4: Add keyboard handling in `app.go`

**Files:**
- Modify: `internal/tui/app.go:291-399` — add `v`, `J`, `K`, `esc` handlers and auto-collapse on cursor move
- Modify: `internal/tui/board.go` — update help bar text

- [ ] **Step 1: Add `v` key handler for description toggle**

In `handleBoardKey` (app.go), add a new case after the `"D"` case (line 393):

```go
	case "v":
		if item := a.selectedItem(); item != nil && item.Description != "" {
			if a.descExpanded == item.ID {
				a.descExpanded = ""
				a.descScrollY = 0
			} else {
				a.descExpanded = item.ID
				a.descScrollY = 0
			}
		}
```

- [ ] **Step 2: Add helper methods to `app.go`**

```go
func (a *App) collapseDescription() {
	a.descExpanded = ""
	a.descScrollY = 0
}

// clampDescScroll ensures descScrollY doesn't exceed the maximum scroll offset
// for the currently expanded description. Call after incrementing descScrollY.
func (a *App) clampDescScroll(maxLines int) {
	if a.descExpanded == "" {
		return
	}
	item := a.selectedItem()
	if item == nil || item.Description == "" {
		return
	}
	w := a.columnWidth()
	contentW := w - 4
	if contentW < 10 {
		contentW = 10
	}
	// Count total wrapped lines (same logic as renderDescription)
	paragraphs := strings.Split(item.Description, "\n")
	totalLines := 0
	for _, p := range paragraphs {
		if p == "" {
			totalLines++
			continue
		}
		totalLines += len(wrapText(p, contentW))
	}
	maxScroll := totalLines - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.descScrollY > maxScroll {
		a.descScrollY = maxScroll
	}
}
```

- [ ] **Step 3: Add `J`/`K` handlers for description scrolling**

In `handleBoardKey`, add after the `"v"` case:

```go
	case "J":
		if a.descExpanded != "" {
			a.descScrollY++
			a.clampDescScroll(5)
		}
	case "K":
		if a.descExpanded != "" {
			if a.descScrollY > 0 {
				a.descScrollY--
			}
		}
```

- [ ] **Step 4: Add auto-collapse on cursor movement**

Then in `handleBoardKey`, add `a.collapseDescription()` at the start of each navigation case. Modify the existing cases:

For `"h", "left"` (line 295): add `a.collapseDescription()` as first line inside the `if`.

For `"l", "right"` (line 302): add `a.collapseDescription()` as first line inside the `if`.

For `"tab"` (line 309): add `a.collapseDescription()` before the `if`.

For `"j", "down"` (line 316): add `a.collapseDescription()` before the items lookup.

For `"k", "up"` (line 322): add `a.collapseDescription()` before the `if`.

- [ ] **Step 5: Add `esc` handling for description collapse**

In `handleBoardKey`, add a new case before the `"q"` case:

```go
	case "esc":
		if a.descExpanded != "" {
			a.collapseDescription()
			return a, nil // consume esc, don't propagate
		}
```

- [ ] **Step 6: Update help bar**

In `renderBoard` (board.go, line 22), update the help text:

```go
help := helpStyle.Render(
    "  h/l:columns  j/k:items  v:desc  m:move  a:assign  c:create  d:delete  " +
        "p:priority  Enter:detail  Space:collapse  /:search  r:reload  q:quit",
)
```

- [ ] **Step 7: Run all tests**

Run: `go test ./internal/tui/ -v`
Expected: PASS

Run: `go test ./...`
Expected: PASS (all packages)

- [ ] **Step 8: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go
git commit -m "feat(tui): add description accordion keybindings (v/J/K) and auto-collapse"
```

---

## Chunk 3: Manual Verification

### Task 5: Smoke test the TUI

**Files:** None (manual testing only)

- [ ] **Step 1: Build and launch TUI**

```bash
go build -o ob . && ./ob tui
```

- [ ] **Step 2: Verify title wrapping**

Navigate to an item with a long title. Confirm the full title is visible across multiple lines. Confirm short titles render compactly (same height as before).

- [ ] **Step 3: Verify accordion**

Select a card with a description. Press `v` — confirm `▼ description` appears with the description text. Press `v` again — confirm it collapses. Navigate away — confirm it auto-collapses.

- [ ] **Step 4: Verify description scrolling**

Find/create an item with a long description (>5 lines). Press `v` to expand. Press `J` to scroll down — confirm `▴` appears at top. Press `K` to scroll up. Confirm `j`/`k` still navigate cards normally.

- [ ] **Step 5: Verify no regressions**

Confirm `d` still deletes. Confirm `D` still opens dashboard. Confirm `space` still collapses epic groups. Confirm `esc` collapses description before exiting overlays.

- [ ] **Step 6: Final commit if any fixes needed**

```bash
git add -A && git commit -m "fix(tui): address smoke test findings"
```
