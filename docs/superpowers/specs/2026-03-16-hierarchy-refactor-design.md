# Unified Card Hierarchy Refactor

**Date:** 2026-03-16
**Status:** Draft

## Problem

The current board rendering has three hierarchy issues:

1. **Redundant double entity** — Epics render as both a collapsible group header AND a card underneath. The header and card show the same information.
2. **Cross-column duplication** — When an epic's children are spread across columns, the epic header duplicates in every column with a `⇠` badge. This creates visual noise.
3. **Flat hierarchy** — Only epic-level grouping exists. Stories don't group their tasks. The full epic → story → task tree is not visible.

## Design Principles

1. **One entity per item** — Every item (epic, story, task) is exactly one card. No separate group headers.
2. **Column-local collapse** — Collapsing a parent hides its children in the same column only. Children in other columns are never hidden (they represent active in-flight work).
3. **Any parent is collapsible** — If an item has children in the same column, it can be collapsed. This applies to epics, stories, and tasks with subtasks.
4. **Breadcrumbs replace duplication** — Instead of duplicating parent headers across columns, children show a faint ancestry breadcrumb above their title.
5. **Feature parity** — TUI and web implement the same hierarchy model.

## Visual Hierarchy System

### Type-colored left borders

Every card gets a left border colored by its item type:

| Type | Border Color | TUI Color |
|------|-------------|-----------|
| Epic | Magenta | Color 5 |
| Story | Blue | Color 4 |
| Task | None (default card border) | — |

This provides instant visual scanning — you can see the item type before reading the title.

### Breadcrumb path

Children show their full ancestry as a faint line above the title:

```
 Auth Rewrite › Session Mgmt
 #2 Refactor middleware
```

Breadcrumb rules:
- Walk up the `ParentID` chain to build the path
- Show parent titles only (not types or numbers) — keeps it compact
- Separate with ` › ` (thin arrow)
- Style: faint/dim text (TUI: lipgloss Faint(true), Web: opacity 0.5)
- Truncate from the left if the path exceeds the card width (show `… › Session Mgmt` instead of `Auth Rewrite › Session Mgmt › …`). TUI: compute in Go by measuring rune width and prepending `… › `. Web: use JS to measure and truncate (do NOT use CSS `direction: rtl` — it breaks with `›` separators and bidi text).
- Items with no parent show no breadcrumb line
- **Cycle protection:** All parent-chain walks (breadcrumb, collapse filter, child count) must use a visited set to guard against circular `ParentID` references, matching the pattern in the existing `findEpicAncestor` function.

### Child count badge

Items with children show a count badge on the title line:

```
 ▼ #1 Auth Rewrite                    5 items
```

Badge rules:
- Rendered as a pill: `N items` (always the word "items", not type-specific) in the parent's type color with 33% opacity background
- Only shown when the item has children (anywhere on the board, not just in the same column)
- Count includes ALL descendants (direct + transitive), not just same-column children. This is always a total count, never filtered by type.
- When collapsed: show `▶`, when expanded: show `▼`
- Items with no children show no badge and no collapse indicator
- Visual examples in this spec use abbreviated forms (`5i`) for space — the actual rendered text is `5 items`

### Progress indicator on parent cards

Parent cards show a completion fraction:

```
 epic ●●●  2/5 done
```

- Counts children whose `Status` matches the board's last column name (typically "done") vs total children. Uses `board.Columns[len(board.Columns)-1].Name` to determine the "done" column dynamically, supporting boards with custom column names.
- Faint style, appended to the type/priority line
- Only on items that have children

## Card Rendering

### Current card structure (before refactor)

```
 ╭────────────────────────╮
 │ #1 Auth Rewrite        │
 │ epic ●●●               │
 │ ▲ #10 Parent           │
 │ @claude                │
 │ ▶ description          │
 ╰────────────────────────╯
```

Plus a separate group header above: `▼ #1 Auth Rewrite`

### New card structure (after refactor)

**Epic card (has children, expanded):**
```
 ╭────────────────────────╮
 │ Auth Rewrite           │  ◄── breadcrumb (only if has parent)
 ┃ ▼ #1 Auth Rewrite  5i  │  ◄── magenta left border, collapse indicator, child count
 ┃ epic ●●●  2/5 done     │  ◄── progress
 ┃ @claude                 │
 ┃ ▶ description           │
 ╰────────────────────────╯
```

**Story card (has children, under an epic):**
```
 ╭────────────────────────╮
 │ Auth Rewrite           │  ◄── breadcrumb: parent epic title
 ┃ ▼ #4 Session Mgmt  2t  │  ◄── blue left border, collapse indicator, task count
 ┃ story ●●               │
 ┃ @claude                │
 ╰────────────────────────╯
```

**Task card (leaf, under a story under an epic):**
```
 ╭────────────────────────╮
 │ Auth Rewrite › Sess Mg │  ◄── breadcrumb: epic › story
 │ #2 Refactor middleware │  ◄── no left border accent, no collapse indicator
 │ task ●●                │
 │ @claude                │
 ╰────────────────────────╯
```

**Task card in a different column (no parent duplication):**
```
 ╭────────────────────────╮
 │ Auth Rewrite › Token V │  ◄── breadcrumb shows ancestry
 │ #3 JWT validation      │  ◄── no ghost parent header, no ⇠ badge
 │ task ●                 │
 │ @claude                │
 ╰────────────────────────╯
```

## Collapse Behavior

### Column-local collapse (Pattern 3)

When a user presses Space on any card:

1. Check if the selected item has children on the board (`hasChildren(board, item.ID)`)
2. If **no children**: Space does nothing (no-op). No visual feedback needed.
3. If **has children**: Toggle the item's collapsed state in the `collapsed` map
4. **Same column:** Hide all descendants whose `Status` matches the current column. If those descendants are themselves parents with expanded descendants in this column, hide those too (recursive via `isHiddenByCollapse`).
5. **Other columns:** Do nothing. Children in other columns remain visible with their breadcrumbs.
6. Update the collapse indicator: `▼` → `▶`
7. The item card itself always remains visible

This replaces the current epic-specific collapse logic. The current Space handler in `app.go` calls `findEpicAncestor` to find the epic and toggles that — the new handler simply toggles `collapsed[selectedItem.ID]` directly, regardless of item type.

### Collapse state storage

```go
// collapsed maps item ID → bool (true = collapsed)
// This is the existing field on App, unchanged
collapsed map[string]bool
```

The collapse state is per-item, not per-item-per-column. If an epic is collapsed, its children are hidden in whichever column the epic lives in. Children in other columns are unaffected because the collapse filter only applies when rendering items in the same column as their collapsed ancestor.

### Collapse filter logic

When building the visible items for a column:

```
for each item in column:
    walk up ParentID chain
    if any ancestor is:
        (a) collapsed AND
        (b) also in this same column (ancestor.Status == column name)
    then: hide this item
    else: show this item
```

This ensures:
- Collapsing an epic in Backlog hides its children in Backlog
- The same epic's children in In-Progress remain visible
- Collapsing a story hides its tasks in the same column
- Nested collapse works: collapsing an epic hides its stories, which hides their tasks

## Removed Concepts

### Cross-column epic headers — REMOVED

The current `renderGroupedCards` function creates cross-column epic group headers (the `⇠` badge entries). These are entirely removed. Children in other columns show their ancestry via breadcrumbs instead.

### Separate group header — REMOVED

The current epic group rendering has two visual elements: a header line (`▼ #1 Auth Rewrite`) and an epic card below it. The header is removed. The epic card itself gains the collapse indicator (`▼`/`▶`) and child count badge. One entity per item.

### `findEpicAncestor` grouping — REPLACED

The current `renderGroupedCards` groups items by epic ancestor and renders them with epic headers. This is replaced by a simpler rendering model:

1. All items in a column are rendered as cards in display order
2. Parent cards have collapse indicators and child count badges
3. Collapsed parents hide their same-column children (via the collapse filter)
4. No special grouping logic — just flat rendering with visual hierarchy cues

## Ordering Within Columns

Items within a column are ordered to preserve visual hierarchy:

The ordering algorithm is a depth-first tree traversal:

1. Build a tree of items in this column using `ParentID` relationships
2. Root items (no parent, or parent not in this column) are sorted by `DisplayNum` ascending
3. For each root, recursively emit: root card, then children sorted by `DisplayNum`, then each child's children, etc.
4. Items from different parent trees interleave by their root's `DisplayNum` — if epic #1 has `DisplayNum` 1 and epic #10 has `DisplayNum` 10, all of epic #1's tree appears before epic #10's tree
5. Orphan items (no parent at all) are treated as roots and sorted among other roots by `DisplayNum`

This creates a natural tree-like visual flow within each column without indentation:

```
 ┌── BACKLOG ──────────────┐
 │                          │
 │ ▼ #1 Auth Rewrite  [5i] │  ◄── epic (depth 0)
 │ ▼ #4 Session Mgmt  [2t] │  ◄── story (depth 1, child of #1)
 │   #2 Refactor middle    │  ◄── task (depth 2, child of #4)
 │   #6 Update session     │  ◄── task (depth 2, child of #4)
 │ ▼ #5 Token Valid.  [2t] │  ◄── story (depth 1, child of #1)
 │   #8 Parse JWT          │  ◄── task (depth 2, child of #5)
 │                          │
 │ #20 Fix README           │  ◄── orphan task (depth 0)
 │                          │
 └──────────────────────────┘
```

## TUI Implementation

### Files to modify

| File | Changes |
|------|---------|
| `internal/tui/card.go` | Remove `renderGroupedCards`. Add breadcrumb rendering, left border, child count badge, progress indicator. Collapse indicator on parent cards. |
| `internal/tui/board.go` | Update `visibleItemsInColumn` with collapse filter. Update `renderBoard` to render flat card list (no grouping). Add ordering logic. Remove cross-column epic injection. |
| `internal/tui/styles.go` | Add left border styles for epic (magenta) and story (blue). Add breadcrumb style. Add child count badge style. |
| `internal/tui/app.go` | Update Space key handler — collapse any item with children (not just epics). Remove epic-specific collapse logic. |

### Removed functions

- `renderGroupedCards` — replaced by flat card rendering with collapse filter
- `findEpicAncestor` — DELETE. All callers replaced by generic parent-chain walking. The past reviews pane (`past_reviews.go`) uses its own `BuildReviewTree` which walks parents directly.
- `isEpicGroupAtCursor` — DELETE. No longer needed.
- `isCollapsedChild` — DELETE. Replaced by generic `isHiddenByCollapse`.
- `parentBadge` / `appendParentBadge` — DELETE. Replaced by breadcrumbs which serve the same purpose but richer (full ancestry path vs single parent reference).

### New helper functions

```go
// breadcrumbPath returns the ancestry path for an item as a string.
// Example: "Auth Rewrite › Session Mgmt"
func breadcrumbPath(board *domain.Board, item *domain.Item, maxWidth int) string

// childCount returns the total number of descendants of an item.
func childCount(board *domain.Board, itemID string) int

// doneCount returns the number of descendants with Status == "done".
func doneCount(board *domain.Board, itemID string) int

// hasChildren returns true if any item on the board has this item as ParentID.
func hasChildren(board *domain.Board, itemID string) bool

// isHiddenByCollapse checks if an item should be hidden because
// a collapsed ancestor is in the same column.
func isHiddenByCollapse(board *domain.Board, item *domain.Item, collapsed map[string]bool) bool

// orderItemsHierarchically sorts items within a column so parents
// appear before their children, preserving tree structure.
func orderItemsHierarchically(board *domain.Board, items []*domain.Item) []*domain.Item
```

### Card left border (lipgloss)

lipgloss does not support per-side border colors. Use a colored `┃` character as the first character of each content line inside the card. This is the same technique used for the review queue amber styling.

```go
// Prepend a colored bar to each line of card content
func prependLeftBar(content string, color lipgloss.Color) string {
    bar := lipgloss.NewStyle().Foreground(color).Render("┃")
    lines := strings.Split(content, "\n")
    for i, line := range lines {
        lines[i] = bar + line
    }
    return strings.Join(lines, "\n")
}
```

This approach reduces the usable content width by 1 character (the bar). The `contentW` calculation in `renderCard` must account for this: `contentW = w - 4 - 1` for epics/stories (border + padding + bar), `contentW = w - 4` for tasks (no bar).

Applied to:
- Epics: `┃` in Color 5 (magenta)
- Stories: `┃` in Color 4 (blue)
- Tasks: no bar

## Web Implementation

### Component changes

| Component | Changes |
|-----------|---------|
| `KanbanCard` | Add breadcrumb line above title. Add type-colored left border via CSS `border-left`. Add child count badge pill. Add collapse indicator (▼/▶) on parent cards. Add progress fraction. |
| `KanbanBoard` | Update column rendering — flat card list with collapse filter. Remove epic grouping/duplication logic. Add hierarchical ordering. Collapse filter and ordering applied in `KanbanBoard` before passing items to `KanbanColumn` — the column component receives pre-filtered, pre-ordered items and needs no changes. |

### CSS

```css
.card--epic { border-left: 3px solid #d946ef; }
.card--story { border-left: 3px solid #3b82f6; }
.card--task { border-left: none; }

.card__breadcrumb {
    font-size: 0.7rem;
    opacity: 0.4;
    margin-bottom: 2px;
    white-space: nowrap;
    overflow: hidden;
    /* Left-truncation done in JS — compute the string with "… › " prefix
       when it exceeds container width. Do NOT use direction: rtl as it
       breaks › separators and bidi text rendering. */
}

.card__child-badge {
    font-size: 0.65rem;
    padding: 1px 6px;
    border-radius: 10px;
    opacity: 0.8;
}
.card__child-badge--epic { background: #d946ef33; color: #d946ef; }
.card__child-badge--story { background: #3b82f633; color: #3b82f6; }
```

### Collapse interaction

- Click the ▼/▶ indicator on a parent card to toggle collapse
- Keyboard: Space when card is focused
- Collapse state stored in component state (local, not persisted)
- Column-local filter applied during render

## Testing

### TUI helper tests

**File:** `internal/tui/hierarchy_test.go`

- `TestBreadcrumbPath` — builds correct ancestry string
- `TestBreadcrumbPath_Truncation` — truncates from left with `… ›` prefix when too wide
- `TestBreadcrumbPath_CycleProtection` — circular ParentID doesn't infinite loop
- `TestChildCount` — counts all descendants (transitive)
- `TestDoneCount` — counts done descendants only
- `TestHasChildren` — true for parents, false for leaves
- `TestIsHiddenByCollapse_SameColumn` — collapsed parent hides child in same column
- `TestIsHiddenByCollapse_DifferentColumn` — collapsed parent does NOT hide child in different column
- `TestIsHiddenByCollapse_NestedCollapse` — grandchild hidden when grandparent collapsed
- `TestIsHiddenByCollapse_CycleProtection` — circular ParentID doesn't infinite loop
- `TestOrderItemsHierarchically` — parent before children, sorted by DisplayNum
- `TestOrderItemsHierarchically_MultipleRoots` — items from different parent trees ordered by root DisplayNum

### TUI tests

- `TestUnifiedCard_EpicWithBorder` — magenta left border on epic card
- `TestUnifiedCard_StoryWithBorder` — blue left border on story card
- `TestUnifiedCard_Breadcrumb` — breadcrumb path renders above title
- `TestUnifiedCard_ChildCountBadge` — badge shows on parent cards
- `TestUnifiedCard_CollapseIndicator` — ▼/▶ on parent, none on leaf
- `TestUnifiedCard_NoCrossColumnDuplication` — no ⇠ badge, no ghost headers
- `TestColumnLocalCollapse` — Space on epic hides children in same column only
- `TestCollapseStory` — stories are collapsible too
- `TestCollapseTaskWithSubtasks` — tasks with subtasks are collapsible

### Golden file tests

- `testdata/unified-hierarchy-expanded.golden` — board with epic → story → task, all expanded
- `testdata/unified-hierarchy-collapsed.golden` — epic collapsed, children hidden in same column
- `testdata/unified-hierarchy-cross-column.golden` — children in other columns visible with breadcrumbs

## Migration

No data model changes. All changes are rendering-only. The `collapsed` map on the App struct already stores per-item collapse state — the only change is that it now applies to any item with children, not just epics.

Existing boards render correctly because:
- New breadcrumb logic handles items with no parent (no breadcrumb shown)
- New child count logic handles items with no children (no badge shown)
- Collapse behavior defaults to expanded (unchanged)

## Out of Scope

- Drag-to-reparent (changing hierarchy by dragging cards)
- Indentation/tree connectors (Option A from brainstorming — decided against due to terminal width)
- Swimlane layout (Option B from brainstorming — decided against due to paradigm shift)
- Persisting collapse state to board.json (collapse is ephemeral, per-session)
