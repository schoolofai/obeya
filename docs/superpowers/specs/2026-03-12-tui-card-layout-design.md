# TUI Card Layout: Full Title Wrapping + Accordion Description

**Date:** 2026-03-12
**Status:** Approved

## Problem

Card titles in the TUI Kanban board are truncated with ellipsis (`...`) when they exceed the column width. Users cannot tell what an item is about without opening the detail view. Descriptions are only visible in the detail view, requiring a context switch.

## Solution

Two changes to the card rendering in `internal/tui/board.go`:

1. **Full title wrapping** — titles wrap across multiple lines instead of being truncated. Cards grow taller to accommodate.
2. **Accordion description** — a toggleable inline description panel on the selected card, capped at 5 lines with internal scrolling.

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Title truncation | No truncation — full wrap | Title is the most important information for board scanning |
| Title line cap | Unlimited | Users need to see the complete title at a glance |
| Description trigger | `v` keybind (mnemonic: "view") | `d` is already bound to delete; `v` is unused in board state |
| Description scrolling | `J`/`K` (shift+j/k) | Keeps `j`/`k` always for card navigation — no behavioral asymmetry |
| Accordion indicator | `▶`/`▼` on selected card only | Reduces visual noise; unselected cards stay clean |
| Description height cap | 5 lines of content | Scroll indicators render outside the 5-line viewport (don't consume content lines) |
| Long description overflow | Scroll within 5-line viewport | User can scroll with `J`/`K` when description is expanded |
| Simultaneous expansions | One at a time | Moving cursor auto-collapses previous; prevents column overflow |
| Description text wrapping | Word-wrapped to card width | Same `wrapText` function as titles; long lines never overflow the card border |

## Card Anatomy

### Current

```
╭──────────────────────╮
│ #12 Add token refr...│
│ task ●●              │
│ @claude              │
╰──────────────────────╯
```

### Proposed — collapsed (default)

```
╭──────────────────────╮
│ #12 Add token refresh│
│     handling for     │
│     expired sessions │
│ task ●●              │
│ @claude              │
│ ▶ description        │  ← only on selected card; hidden if no description
╰──────────────────────╯
```

### Proposed — expanded (press `v`)

```
╭──────────────────────╮
│ #12 Add token refresh│
│     handling for     │
│     expired sessions │
│ task ●●              │
│ @claude              │
│ ▼ description        │
│ ─────────────────────│
│ Handle JWT token     │  ← 5 lines of content
│ expiry by auto       │
│ refresh. When a 401  │
│ is received, attempt │
│ to refresh before    │
│              ▾ J/K   │  ← scroll indicator (outside the 5-line viewport)
╰──────────────────────╯
```

## Keyboard Controls

| Key | Context | Action |
|---|---|---|
| `v` | Board view, card selected | Toggle description expand/collapse |
| `J` (shift+j) | Description expanded | Scroll description down |
| `K` (shift+k) | Description expanded | Scroll description up |
| `esc` | Description expanded | Collapse description (consumed; does not propagate) |
| `j`/`k`/`↑`/`↓` | Board view | Navigate cards (unchanged); auto-collapses any open description |
| `h`/`l`/`←`/`→`/`tab` | Board view | Navigate columns (unchanged); auto-collapses any open description |
| `space` | Board view | Toggle epic collapse (unchanged) |
| `d` | Board view | Delete item (unchanged) |

## Behavior Rules

1. **Only one description expanded at a time.** Moving the cursor to a different card (any direction, any key) auto-collapses the previously expanded description.
2. **Accordion indicator only on selected card.** Unselected cards show no `▶` hint.
3. **Description scrolls within a fixed 5-line content viewport.** Scroll indicators (`▾`/`▴`) render outside the viewport and do not consume content lines. When total description lines <= 5, no scroll indicators are shown.
4. **Title wraps fully — never truncated.** Card height adapts to title length. Short titles produce compact cards identical in height to today's cards.
5. **Empty descriptions show nothing.** No `▶` indicator if the item has no description.
6. **Help bar updated.** Add `v:desc` to the bottom help bar alongside existing shortcuts.
7. **Description text is word-wrapped** to fit within the card width, using the same `wrapText` function as titles. Long lines never overflow the card border.
8. **Card width is set dynamically** from `columnWidth()` rather than the hardcoded `Width(22)` in styles. Both `cardStyle` and `selectedCardStyle` are rendered with the current column width.
9. **Epic group headers remain truncated.** Only card titles get full wrapping. Epic group header truncation in `renderGroupedCards` is unchanged — these are section dividers, not primary content.

## Architecture

### Files Modified

| File | Change |
|---|---|
| `internal/tui/board.go` | Replace `truncate()` usage in `renderCard()` with `wrapText()`; add description accordion rendering; add `wrapText()` and `renderDescription()` functions |
| `internal/tui/app.go` | Add `descExpanded string` and `descScrollY int` state fields; handle `v`, `J`, `K` key events; auto-collapse on all cursor movement |
| `internal/tui/styles.go` | Add `descStyle` and `descIndicatorStyle`; remove hardcoded `Width(22)` from card styles (width set at render time) |

### New Functions

- `wrapText(s string, maxWidth int) []string` — word-wraps a string into lines that fit within `maxWidth` characters. Breaks on word boundaries when possible, mid-word only when a single word exceeds `maxWidth`. Handles empty strings (returns `[]string{""}`).
- `renderDescription(desc string, maxWidth int, scrollY int, maxLines int) []string` — word-wraps the description, applies the scroll offset, takes `maxLines` lines, and appends scroll indicators if content exists above or below the viewport. Returns a slice of styled lines for appending to the card's line list.

### State Changes to `App` struct

```go
type App struct {
    // ... existing fields ...
    descExpanded string  // item ID whose description is expanded, "" if none
    descScrollY  int     // scroll offset within the expanded description
}
```

### Key Handling Pseudocode

```
on key "v":
    if selected item has no description: ignore
    if descExpanded == selected item ID:
        descExpanded = ""  // collapse
    else:
        descExpanded = selected item ID  // expand (auto-collapses previous)
        descScrollY = 0

on key "J" (shift+j, when descExpanded != ""):
    descScrollY++ (clamped to totalLines - maxLines)

on key "K" (shift+k, when descExpanded != ""):
    descScrollY-- (clamped to 0)

on any cursor move (j/k/up/down/h/l/left/right/tab):
    descExpanded = ""  // auto-collapse
    descScrollY = 0
    // then proceed with normal navigation

on key "esc":
    if descExpanded != "":
        descExpanded = ""  // collapse
        descScrollY = 0
        return  // consume the esc, don't propagate
```

### Rendering Pseudocode

```
func renderCard(item, selected):
    w := columnWidth()

    // Title: wrap instead of truncate
    prefix := fmt.Sprintf("#%d ", item.DisplayNum)
    firstLineMax := w - 4 - len(prefix)  // 4 = border(2) + padding(2)
    restLineMax := w - 4                  // continuation lines use full width
    titleLines := wrapTextWithPrefix(item.Title, firstLineMax, restLineMax)
    line1 := prefix + titleLines[0]
    for remaining titleLines:
        append line with leading indent (len(prefix) spaces)

    // Type + priority (unchanged)
    // Parent badge (unchanged)
    // Assignee + blocked (unchanged)

    // Accordion indicator (only on selected card with non-empty description)
    if selected && item.Description != "":
        if descExpanded == item.ID:
            append "▼ description"
            append separator line "─" * (w - 4)
            append renderDescription(item.Description, w - 4, descScrollY, 5)
        else:
            append "▶ description"

    // Apply card style with dynamic width
    style := cardStyle.Width(w - 2)  // -2 for border
    if selected:
        style = selectedCardStyle.Width(w - 2)
    return style.Render(content)
```

## Testing

### Unit tests for `wrapText` (`board_test.go`)

- Empty string returns `[""]`
- String shorter than maxWidth returns single-element slice
- String exactly at maxWidth returns single-element slice
- Long string wraps at word boundaries
- Single word longer than maxWidth breaks mid-word
- Unicode/multi-byte characters handled correctly
- Multiple spaces between words collapsed

### Unit tests for `renderDescription` (`board_test.go`)

- Description shorter than 5 lines — no scroll indicators
- Description exactly 5 lines — no scroll indicators
- Description longer than 5 lines at scrollY=0 — shows `▾` below
- Description scrolled to middle — shows `▴` above and `▾` below
- Description scrolled to end — shows `▴` above only
- Empty description returns empty string

### Integration tests (manual or via `cmd/` test)

- Short titles render identically to current behavior (no visual regression)
- Long titles wrap correctly at word boundaries
- `v` toggles description on selected card
- `J`/`K` scrolls within expanded description
- `j`/`k`/arrow cursor movement auto-collapses expanded description
- Column navigation (`h`/`l`) auto-collapses expanded description
- No `▶` indicator on cards with empty descriptions
- Only one description expanded at a time
- `esc` collapses description before propagating
- `d` still triggers delete (no conflict)
- Card widths adapt to terminal/column width changes
