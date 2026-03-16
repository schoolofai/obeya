# Unified Card Hierarchy Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace redundant epic header + card rendering with unified cards that show hierarchy via type-colored left borders, breadcrumb paths, child count badges, and column-local collapse.

**Architecture:** Remove `renderGroupedCards` and cross-column epic injection. Replace with flat card rendering where each card self-describes its hierarchy (breadcrumb, left bar, badge). New `hierarchy.go` file holds pure helper functions. Collapse filter in `visibleItemsInColumn` replaces epic-specific logic.

**Tech Stack:** Go 1.26, Bubble Tea, lipgloss, teatest, Next.js/React (web)

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `internal/tui/hierarchy.go` | Pure hierarchy helpers: breadcrumb, childCount, doneCount, hasChildren, isHiddenByCollapse, orderItemsHierarchically | CREATE |
| `internal/tui/hierarchy_test.go` | Tests for all hierarchy helpers | CREATE |
| `internal/tui/card.go` | Card rendering — add breadcrumb, left bar, collapse indicator, child count badge, progress; remove renderGroupedCards, findEpicAncestor, parentBadge, isEpicGroupAtCursor, isCollapsedChild | MODIFY |
| `internal/tui/board.go` | Board rendering — replace visibleItemsInColumn collapse/ordering logic, replace renderGroupedCards call with flat card list, remove renderOrderItems | MODIFY |
| `internal/tui/styles.go` | Add breadcrumb style, child badge style, left bar colors | MODIFY |
| `internal/tui/app.go` | Update Space key handler — generic collapse for any parent item | MODIFY |
| `internal/tui/golden_test.go` | Add 3 new golden snapshots for hierarchy rendering | MODIFY |
| `web/components/board/kanban-card.tsx` | Add breadcrumb, left border CSS class, collapse indicator, child count badge, progress | MODIFY |
| `web/components/board/kanban-board.tsx` | Replace column item filtering with collapse filter + hierarchical ordering | MODIFY |
| `web/lib/hierarchy.ts` | Shared hierarchy helpers (breadcrumb, childCount, isHiddenByCollapse, orderHierarchically) | CREATE |
| `web/__tests__/lib/hierarchy.test.ts` | Tests for web hierarchy helpers | CREATE |

---

## Chunk 1: TUI Hierarchy Helpers

### Task 1: Create hierarchy helper functions with tests

**Files:**
- Create: `internal/tui/hierarchy.go`
- Create: `internal/tui/hierarchy_test.go`

- [ ] **Step 1: Write tests for breadcrumbPath**

Create `internal/tui/hierarchy_test.go`:

```go
package tui

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func testBoard() *domain.Board {
	b := domain.NewBoard("test")
	b.Items["epic-1"] = &domain.Item{ID: "epic-1", DisplayNum: 1, Title: "Auth Rewrite", Type: domain.ItemTypeEpic, Status: "backlog", ParentID: ""}
	b.Items["story-4"] = &domain.Item{ID: "story-4", DisplayNum: 4, Title: "Session Management", Type: domain.ItemTypeStory, Status: "backlog", ParentID: "epic-1"}
	b.Items["task-2"] = &domain.Item{ID: "task-2", DisplayNum: 2, Title: "Refactor middleware", Type: domain.ItemTypeTask, Status: "backlog", ParentID: "story-4"}
	b.Items["task-3"] = &domain.Item{ID: "task-3", DisplayNum: 3, Title: "JWT validation", Type: domain.ItemTypeTask, Status: "in-progress", ParentID: "story-4"}
	b.Items["task-6"] = &domain.Item{ID: "task-6", DisplayNum: 6, Title: "Update session store", Type: domain.ItemTypeTask, Status: "done", ParentID: "story-4"}
	b.Items["task-20"] = &domain.Item{ID: "task-20", DisplayNum: 20, Title: "Fix README", Type: domain.ItemTypeTask, Status: "backlog", ParentID: ""}
	b.DisplayMap[1] = "epic-1"
	b.DisplayMap[4] = "story-4"
	b.DisplayMap[2] = "task-2"
	b.DisplayMap[3] = "task-3"
	b.DisplayMap[6] = "task-6"
	b.DisplayMap[20] = "task-20"
	return b
}

func TestBreadcrumbPath(t *testing.T) {
	b := testBoard()
	// Task under story under epic
	got := breadcrumbPath(b, b.Items["task-2"], 100)
	want := "Auth Rewrite › Session Management"
	if got != want {
		t.Errorf("breadcrumbPath = %q, want %q", got, want)
	}
}

func TestBreadcrumbPath_StoryUnderEpic(t *testing.T) {
	b := testBoard()
	got := breadcrumbPath(b, b.Items["story-4"], 100)
	want := "Auth Rewrite"
	if got != want {
		t.Errorf("breadcrumbPath = %q, want %q", got, want)
	}
}

func TestBreadcrumbPath_NoParent(t *testing.T) {
	b := testBoard()
	got := breadcrumbPath(b, b.Items["epic-1"], 100)
	if got != "" {
		t.Errorf("breadcrumbPath for root = %q, want empty", got)
	}
}

func TestBreadcrumbPath_Truncation(t *testing.T) {
	b := testBoard()
	got := breadcrumbPath(b, b.Items["task-2"], 20)
	// "Auth Rewrite › Session Management" is 35 chars, should truncate from left
	if len(got) > 20 {
		t.Errorf("breadcrumbPath length = %d, want <= 20", len(got))
	}
	if got == "" {
		t.Error("breadcrumbPath should not be empty even when truncated")
	}
	// Should start with "… › "
	if len(got) > 4 && got[:len("… › ")] != "… › " {
		t.Errorf("truncated breadcrumb should start with '… › ', got %q", got)
	}
}

func TestBreadcrumbPath_CycleProtection(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", Title: "A", ParentID: "b"}
	b.Items["b"] = &domain.Item{ID: "b", Title: "B", ParentID: "a"}
	// Should not infinite loop
	got := breadcrumbPath(b, b.Items["a"], 100)
	if got == "" {
		t.Error("should produce some breadcrumb even with cycle")
	}
}

func TestChildCount(t *testing.T) {
	b := testBoard()
	// Epic has story-4 + task-2 + task-3 + task-6 = 4 descendants
	got := childCount(b, "epic-1")
	if got != 4 {
		t.Errorf("childCount(epic) = %d, want 4", got)
	}
	// Story has task-2 + task-3 + task-6 = 3 descendants
	got = childCount(b, "story-4")
	if got != 3 {
		t.Errorf("childCount(story) = %d, want 3", got)
	}
	// Leaf task has 0
	got = childCount(b, "task-2")
	if got != 0 {
		t.Errorf("childCount(leaf) = %d, want 0", got)
	}
}

func TestDoneCount(t *testing.T) {
	b := testBoard()
	// Epic: only task-6 is done = 1
	got := doneCount(b, "epic-1")
	if got != 1 {
		t.Errorf("doneCount(epic) = %d, want 1", got)
	}
}

func TestHasChildren(t *testing.T) {
	b := testBoard()
	if !hasChildItems(b, "epic-1") {
		t.Error("epic should have children")
	}
	if !hasChildItems(b, "story-4") {
		t.Error("story should have children")
	}
	if hasChildItems(b, "task-2") {
		t.Error("leaf task should not have children")
	}
}

func TestIsHiddenByCollapse_SameColumn(t *testing.T) {
	b := testBoard()
	collapsed := map[string]bool{"epic-1": true}
	// story-4 is in backlog (same as epic-1) → hidden
	if !isHiddenByCollapse(b, b.Items["story-4"], collapsed) {
		t.Error("story in same column as collapsed epic should be hidden")
	}
	// task-2 is in backlog (same as epic-1) → hidden (grandchild)
	if !isHiddenByCollapse(b, b.Items["task-2"], collapsed) {
		t.Error("task in same column as collapsed epic should be hidden")
	}
}

func TestIsHiddenByCollapse_DifferentColumn(t *testing.T) {
	b := testBoard()
	collapsed := map[string]bool{"epic-1": true}
	// task-3 is in in-progress, epic is in backlog → NOT hidden
	if isHiddenByCollapse(b, b.Items["task-3"], collapsed) {
		t.Error("task in different column than collapsed epic should NOT be hidden")
	}
}

func TestIsHiddenByCollapse_NestedCollapse(t *testing.T) {
	b := testBoard()
	collapsed := map[string]bool{"story-4": true}
	// task-2 is child of story-4, both in backlog → hidden
	if !isHiddenByCollapse(b, b.Items["task-2"], collapsed) {
		t.Error("task should be hidden when parent story is collapsed in same column")
	}
	// epic-1 is parent of story-4, NOT hidden (parent is never hidden by child collapse)
	if isHiddenByCollapse(b, b.Items["epic-1"], collapsed) {
		t.Error("parent should NOT be hidden by child's collapse")
	}
}

func TestIsHiddenByCollapse_CycleProtection(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", Status: "backlog", ParentID: "b"}
	b.Items["b"] = &domain.Item{ID: "b", Status: "backlog", ParentID: "a"}
	collapsed := map[string]bool{"a": true}
	// Should not infinite loop
	_ = isHiddenByCollapse(b, b.Items["b"], collapsed)
}

func TestOrderItemsHierarchically(t *testing.T) {
	b := testBoard()
	items := []*domain.Item{
		b.Items["task-20"], // orphan
		b.Items["task-2"],  // child of story-4
		b.Items["story-4"], // child of epic-1
		b.Items["epic-1"],  // root
	}
	ordered := orderItemsHierarchically(b, items)
	// Expected: epic-1, story-4, task-2, task-20
	wantOrder := []string{"epic-1", "story-4", "task-2", "task-20"}
	if len(ordered) != len(wantOrder) {
		t.Fatalf("got %d items, want %d", len(ordered), len(wantOrder))
	}
	for i, id := range wantOrder {
		if ordered[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, ordered[i].ID, id)
		}
	}
}

func TestOrderItemsHierarchically_MultipleRoots(t *testing.T) {
	b := testBoard()
	b.Items["epic-10"] = &domain.Item{ID: "epic-10", DisplayNum: 10, Title: "Rate Limiting", Type: domain.ItemTypeEpic, Status: "backlog"}
	b.Items["task-11"] = &domain.Item{ID: "task-11", DisplayNum: 11, Title: "Throttle", Type: domain.ItemTypeTask, Status: "backlog", ParentID: "epic-10"}

	items := []*domain.Item{
		b.Items["task-11"],
		b.Items["epic-10"],
		b.Items["task-2"],
		b.Items["story-4"],
		b.Items["epic-1"],
		b.Items["task-20"],
	}
	ordered := orderItemsHierarchically(b, items)
	// epic-1 tree first (DisplayNum 1), then epic-10 tree (DisplayNum 10), then orphan task-20
	wantOrder := []string{"epic-1", "story-4", "task-2", "epic-10", "task-11", "task-20"}
	for i, id := range wantOrder {
		if i >= len(ordered) || ordered[i].ID != id {
			got := "<nil>"
			if i < len(ordered) {
				got = ordered[i].ID
			}
			t.Errorf("position %d: got %s, want %s", i, got, id)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestBreadcrumb|TestChildCount|TestDoneCount|TestHasChildren|TestIsHidden|TestOrder" -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement hierarchy helpers**

Create `internal/tui/hierarchy.go`:

```go
package tui

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/niladribose/obeya/internal/domain"
)

// breadcrumbPath returns the ancestry path for an item.
// Example: "Auth Rewrite › Session Mgmt"
// Truncates from the left with "… › " prefix if the path exceeds maxWidth.
func breadcrumbPath(board *domain.Board, item *domain.Item, maxWidth int) string {
	if item.ParentID == "" {
		return ""
	}

	var titles []string
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if visited[cur.ParentID] {
			break
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			break
		}
		titles = append([]string{parent.Title}, titles...)
		cur = parent
	}

	if len(titles) == 0 {
		return ""
	}

	path := strings.Join(titles, " › ")
	if utf8.RuneCountInString(path) <= maxWidth {
		return path
	}

	// Truncate from the left: remove leading segments until it fits
	for len(titles) > 1 {
		titles = titles[1:]
		path = "… › " + strings.Join(titles, " › ")
		if utf8.RuneCountInString(path) <= maxWidth {
			return path
		}
	}
	// Still too long: truncate the last remaining title
	path = "… › " + titles[0]
	runes := []rune(path)
	if len(runes) > maxWidth {
		return string(runes[:maxWidth-3]) + "..."
	}
	return path
}

// childCount returns the total number of descendants of an item.
func childCount(board *domain.Board, itemID string) int {
	count := 0
	for _, item := range board.Items {
		if isDescendantOf(board, item, itemID) {
			count++
		}
	}
	return count
}

// doneCount returns the number of descendants in the board's last column.
func doneCount(board *domain.Board, itemID string) int {
	doneCol := "done"
	if len(board.Columns) > 0 {
		doneCol = board.Columns[len(board.Columns)-1].Name
	}
	count := 0
	for _, item := range board.Items {
		if item.Status == doneCol && isDescendantOf(board, item, itemID) {
			count++
		}
	}
	return count
}

// hasChildItems returns true if any item on the board has itemID as its ParentID.
func hasChildItems(board *domain.Board, itemID string) bool {
	for _, item := range board.Items {
		if item.ParentID == itemID {
			return true
		}
	}
	return false
}

// isDescendantOf checks if item is a descendant of ancestorID (with cycle protection).
func isDescendantOf(board *domain.Board, item *domain.Item, ancestorID string) bool {
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if cur.ParentID == ancestorID {
			return true
		}
		if visited[cur.ParentID] {
			return false
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			return false
		}
		cur = parent
	}
	return false
}

// isHiddenByCollapse checks if an item should be hidden because
// a collapsed ancestor is in the same column.
func isHiddenByCollapse(board *domain.Board, item *domain.Item, collapsed map[string]bool) bool {
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if visited[cur.ParentID] {
			return false
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			return false
		}
		if collapsed[parent.ID] && parent.Status == item.Status {
			return true
		}
		cur = parent
	}
	return false
}

// orderItemsHierarchically sorts items so parents appear before children,
// using depth-first traversal ordered by DisplayNum.
func orderItemsHierarchically(board *domain.Board, items []*domain.Item) []*domain.Item {
	itemSet := map[string]bool{}
	for _, it := range items {
		itemSet[it.ID] = true
	}

	// Find roots: items whose parent is not in this column's item set
	var roots []*domain.Item
	childrenOf := map[string][]*domain.Item{}

	for _, it := range items {
		if it.ParentID == "" || !itemSet[it.ParentID] {
			roots = append(roots, it)
		} else {
			childrenOf[it.ParentID] = append(childrenOf[it.ParentID], it)
		}
	}

	// Sort roots and children by DisplayNum ascending
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].DisplayNum < roots[j].DisplayNum
	})
	for k := range childrenOf {
		children := childrenOf[k]
		sort.Slice(children, func(i, j int) bool {
			return children[i].DisplayNum < children[j].DisplayNum
		})
	}

	// Depth-first emit
	var ordered []*domain.Item
	var walk func(item *domain.Item)
	walk = func(item *domain.Item) {
		ordered = append(ordered, item)
		for _, child := range childrenOf[item.ID] {
			walk(child)
		}
	}
	for _, root := range roots {
		walk(root)
	}

	return ordered
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestBreadcrumb|TestChildCount|TestDoneCount|TestHasChildren|TestIsHidden|TestOrder" -v`
Expected: All 12 PASS

- [ ] **Step 5: Run full test suite**

Run: `./scripts/test.sh`
Expected: All pass (new file, no existing code changed yet)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/hierarchy.go internal/tui/hierarchy_test.go
git commit -m "feat: add hierarchy helpers — breadcrumb, childCount, collapse filter, ordering"
```

## Chunk 2: TUI Card & Board Rendering Refactor

### Task 2: Add hierarchy styles

**Files:**
- Modify: `internal/tui/styles.go`

- [ ] **Step 1: Add new styles**

Append to `internal/tui/styles.go`:

```go
	// Hierarchy: left bar colors
	epicBarColor  = lipgloss.Color("5")  // magenta
	storyBarColor = lipgloss.Color("4")  // blue

	// Breadcrumb
	breadcrumbStyle = lipgloss.NewStyle().Faint(true)

	// Child count badge
	epicBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Background(lipgloss.Color("53")) // dark magenta bg

	storyBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Background(lipgloss.Color("17")) // dark blue bg

	// Progress fraction
	progressStyle = lipgloss.NewStyle().Faint(true)
```

Add helper:

```go
func leftBarStyle(itemType domain.ItemType) (lipgloss.Color, bool) {
	switch itemType {
	case domain.ItemTypeEpic:
		return epicBarColor, true
	case domain.ItemTypeStory:
		return storyBarColor, true
	default:
		return "", false
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: Compiles

- [ ] **Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add hierarchy styles — left bar, breadcrumb, badge, progress"
```

### Task 3: Refactor card rendering with hierarchy features

**Files:**
- Modify: `internal/tui/card.go`

- [ ] **Step 1: Update buildCardLines to add breadcrumb, collapse indicator, child badge, progress, left bar**

Replace `buildCardLines`:

```go
func (a App) buildCardLines(item *domain.Item, selected bool, contentW int) []string {
	var lines []string

	// Breadcrumb (above title, faint)
	bc := breadcrumbPath(a.board, item, contentW)
	if bc != "" {
		lines = append(lines, breadcrumbStyle.Render(bc))
	}

	// Title with optional collapse indicator and child count badge
	lines = append(lines, a.buildHierarchyTitleLines(item, contentW)...)

	// Type + Priority + optional progress
	lines = append(lines, a.buildTypePriorityLine(item)...)

	// Meta line (assignee, blocked, sponsor, downstream)
	lines = a.appendMetaLine(lines, item)

	// Accordions
	lines = a.appendDescAccordion(lines, item, selected, contentW)
	lines = a.appendReviewAccordion(lines, item, selected, contentW)

	return lines
}
```

- [ ] **Step 2: Add buildHierarchyTitleLines**

```go
func (a App) buildHierarchyTitleLines(item *domain.Item, contentW int) []string {
	prefix := ""

	// Collapse indicator for items with children
	if hasChildItems(a.board, item.ID) {
		if a.collapsed[item.ID] {
			prefix += "▶ "
		} else {
			prefix += "▼ "
		}
	}

	// Agent badge
	if u, ok := a.board.Users[item.Assignee]; ok && u.Type == domain.IdentityAgent {
		prefix += agentBadgeStyle.Render("AGENT") + " "
	}

	prefix += fmt.Sprintf("#%d ", item.DisplayNum)

	// Child count badge (rendered after title)
	badge := ""
	if count := childCount(a.board, item.ID); count > 0 {
		badgeText := fmt.Sprintf("%d items", count)
		switch item.Type {
		case domain.ItemTypeEpic:
			badge = " " + epicBadgeStyle.Render(badgeText)
		case domain.ItemTypeStory:
			badge = " " + storyBadgeStyle.Render(badgeText)
		default:
			badge = " " + progressStyle.Render(badgeText)
		}
	}

	badgeWidth := lipgloss.Width(badge)
	titleMax := contentW - utf8.RuneCountInString(prefix) - badgeWidth
	if titleMax < 4 {
		titleMax = 4
	}
	titleLines := wrapText(item.Title, titleMax)
	lines := []string{prefix + titleLines[0] + badge}
	indent := strings.Repeat(" ", utf8.RuneCountInString(prefix))
	for _, tl := range titleLines[1:] {
		lines = append(lines, indent+tl)
	}
	return lines
}
```

- [ ] **Step 3: Update buildTypePriorityLine with progress**

```go
func (a App) buildTypePriorityLine(item *domain.Item) []string {
	typLabel := typeStyle(string(item.Type)).Render(string(item.Type))
	line := fmt.Sprintf("%s %s", typLabel, priorityIndicator(string(item.Priority)))

	confStr := confidenceIndicator(item.Confidence)
	if confStr != "" {
		line += "  " + confStr
	}

	// Progress for parents
	if total := childCount(a.board, item.ID); total > 0 {
		done := doneCount(a.board, item.ID)
		line += "  " + progressStyle.Render(fmt.Sprintf("%d/%d done", done, total))
	}

	return []string{line}
}
```

- [ ] **Step 4: Add left bar to renderCard**

Update `renderCard` to apply left bar for epics/stories:

```go
func (a App) renderCard(item *domain.Item, selected bool) string {
	w := a.columnWidth()
	barColor, hasBar := leftBarStyle(item.Type)
	contentW := w - 4 // border(2) + padding(2)
	if hasBar {
		contentW-- // account for ┃ character
	}
	if contentW < 10 {
		contentW = 10
	}

	lines := a.buildCardLines(item, selected, contentW)

	for i, line := range lines {
		lines[i] = padToWidth(line, contentW)
	}

	content := strings.Join(lines, "\n")

	// Prepend left bar for epics/stories
	if hasBar {
		content = prependLeftBar(content, barColor)
	}

	return a.applyCardStyle(item, selected, content)
}
```

Add `prependLeftBar`:

```go
func prependLeftBar(content string, color lipgloss.Color) string {
	bar := lipgloss.NewStyle().Foreground(color).Render("┃")
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = bar + line
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 5: Delete removed functions**

Delete from `card.go`:
- `renderGroupedCards` (entire function, lines 150-252)
- `isEpicGroupAtCursor` (lines 259-269)
- `parentBadge` (lines 271-285)
- `appendParentBadge` (lines 74-80)
- `isCollapsedChild` (lines 287-293)
- `findEpicAncestor` (lines 295-316)

Remove `appendParentBadge` call from `buildCardLines`.

- [ ] **Step 6: Verify build**

Run: `go build ./internal/tui/`
Expected: Compile errors from board.go (still references deleted functions) — this is expected, fixed in Task 4.

- [ ] **Step 7: Commit (WIP)**

```bash
git add internal/tui/card.go
git commit -m "WIP: refactor card.go — unified cards with hierarchy features"
```

### Task 4: Refactor board rendering

**Files:**
- Modify: `internal/tui/board.go`

- [ ] **Step 1: Replace renderGroupedCards call with flat card list**

In `renderBoard`, replace line 29 (`cardViews := a.renderGroupedCards(items, i)`) with:

```go
		var cardViews []string
		for _, item := range items {
			cardViews = append(cardViews, a.renderCard(item, a.isItemAtCursor(item)))
		}
```

- [ ] **Step 2: Replace visibleItemsInColumn with new logic**

Replace the entire `visibleItemsInColumn` function body (after the human-review check) with:

```go
	// Collect items in this column
	var colItems []*domain.Item
	for _, item := range a.board.Items {
		if item.Status == colName {
			colItems = append(colItems, item)
		}
	}

	// Filter out items hidden by collapsed ancestors
	var visible []*domain.Item
	for _, item := range colItems {
		if !isHiddenByCollapse(a.board, item, a.collapsed) {
			visible = append(visible, item)
		}
	}

	// Order hierarchically: parents before children, depth-first by DisplayNum
	return orderItemsHierarchically(a.board, visible)
```

- [ ] **Step 3: Delete renderOrderItems**

Delete the entire `renderOrderItems` function (lines 114-164) — replaced by `orderItemsHierarchically`.

- [ ] **Step 4: Remove cross-column epic injection**

The old code (lines 75-92) injected cross-column epics into column items. This entire block is now removed — children in other columns show breadcrumbs instead.

- [ ] **Step 5: Verify build compiles**

Run: `go build ./internal/tui/`
Expected: Compiles cleanly

- [ ] **Step 6: Run full test suite**

Run: `./scripts/test.sh`
Expected: Build passes, some golden tests may fail (expected — rendering changed)

- [ ] **Step 7: Commit**

```bash
git add internal/tui/board.go
git commit -m "feat: refactor board.go — flat rendering with collapse filter and hierarchical ordering"
```

### Task 5: Update Space key handler

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Replace epic-specific collapse with generic collapse**

Replace the Space key handler (lines 556-563):

```go
	case " ":
		if item := a.selectedItem(); item != nil {
			if hasChildItems(a.board, item.ID) {
				a.collapsed[item.ID] = !a.collapsed[item.ID]
				a.clampCursor()
			}
		}
```

This replaces the old logic that called `findEpicAncestor` and only collapsed epics.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: Compiles

- [ ] **Step 3: Run tests**

Run: `./scripts/test.sh`
Expected: Pass (may need golden update)

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: generic collapse — any item with children is collapsible"
```

### Task 6: Update golden files and add new snapshots

**Files:**
- Modify: `internal/tui/golden_test.go`

- [ ] **Step 1: Regenerate existing golden files**

Run: `./scripts/test.sh --update`
Expected: Golden files regenerated with new card rendering

- [ ] **Step 2: Add new golden test cases**

Add to `golden_test.go` test cases for:
- `TestGolden_UnifiedHierarchyExpanded` — board with epic → story → task, all expanded, showing breadcrumbs and left bars
- `TestGolden_UnifiedHierarchyCollapsed` — epic collapsed in backlog, children hidden there but visible in other columns
- `TestGolden_UnifiedHierarchyCrossColumn` — tasks in different columns with breadcrumbs, no ghost headers

Each test creates a board with the `testBoard()` fixture from hierarchy_test.go, renders at 120x40.

- [ ] **Step 3: Generate new golden files**

Run: `./scripts/test.sh --update`

- [ ] **Step 4: Verify all golden files**

Run: `./scripts/test.sh --golden`
Expected: All golden tests pass

- [ ] **Step 5: Run full test suite**

Run: `./scripts/test.sh`
Expected: All checks pass

- [ ] **Step 6: Commit**

```bash
git add internal/tui/golden_test.go internal/tui/testdata/
git commit -m "test: update golden files for unified card hierarchy rendering"
```

## Chunk 3: Web Implementation

### Task 7: Create web hierarchy helpers

**Files:**
- Create: `web/lib/hierarchy.ts`
- Create: `web/__tests__/lib/hierarchy.test.ts`

- [ ] **Step 1: Write tests**

Create `web/__tests__/lib/hierarchy.test.ts`:

```typescript
import { breadcrumbPath, childCount, doneCount, hasChildren, isHiddenByCollapse, orderItemsHierarchically } from '@/lib/hierarchy'
import type { Item } from '@/lib/types'

function makeItem(overrides: Partial<Item>): Item {
  return {
    $id: overrides.$id || 'id',
    display_num: overrides.display_num || 1,
    title: overrides.title || 'test',
    type: overrides.type || 'task',
    status: overrides.status || 'backlog',
    priority: overrides.priority || 'medium',
    parent_id: overrides.parent_id || '',
    assignee: '',
    blocked_by: [],
    tags: [],
    created_at: '',
    updated_at: '',
    ...overrides,
  } as Item
}

const testItems: Record<string, Item> = {
  'epic-1': makeItem({ $id: 'epic-1', display_num: 1, title: 'Auth Rewrite', type: 'epic', status: 'backlog' }),
  'story-4': makeItem({ $id: 'story-4', display_num: 4, title: 'Session Mgmt', type: 'story', status: 'backlog', parent_id: 'epic-1' }),
  'task-2': makeItem({ $id: 'task-2', display_num: 2, title: 'Refactor', type: 'task', status: 'backlog', parent_id: 'story-4' }),
  'task-3': makeItem({ $id: 'task-3', display_num: 3, title: 'JWT', type: 'task', status: 'in-progress', parent_id: 'story-4' }),
  'task-6': makeItem({ $id: 'task-6', display_num: 6, title: 'Update store', type: 'task', status: 'done', parent_id: 'story-4' }),
}

describe('breadcrumbPath', () => {
  it('builds ancestry path', () => {
    expect(breadcrumbPath(testItems, testItems['task-2'])).toBe('Auth Rewrite › Session Mgmt')
  })
  it('returns empty for root', () => {
    expect(breadcrumbPath(testItems, testItems['epic-1'])).toBe('')
  })
})

describe('childCount', () => {
  it('counts all descendants', () => {
    expect(childCount(testItems, 'epic-1')).toBe(4)
  })
  it('returns 0 for leaf', () => {
    expect(childCount(testItems, 'task-2')).toBe(0)
  })
})

describe('isHiddenByCollapse', () => {
  it('hides same-column child', () => {
    expect(isHiddenByCollapse(testItems, testItems['story-4'], { 'epic-1': true })).toBe(true)
  })
  it('does NOT hide cross-column child', () => {
    expect(isHiddenByCollapse(testItems, testItems['task-3'], { 'epic-1': true })).toBe(false)
  })
})

describe('orderItemsHierarchically', () => {
  it('parents before children', () => {
    const items = [testItems['task-2'], testItems['story-4'], testItems['epic-1']]
    const ordered = orderItemsHierarchically(testItems, items)
    expect(ordered.map(i => i.$id)).toEqual(['epic-1', 'story-4', 'task-2'])
  })
})
```

- [ ] **Step 2: Implement web hierarchy helpers**

Create `web/lib/hierarchy.ts`:

```typescript
import type { Item } from './types'

export function breadcrumbPath(allItems: Record<string, Item>, item: Item): string {
  if (!item.parent_id) return ''
  const titles: string[] = []
  const visited = new Set([item.$id])
  let cur = item
  while (cur.parent_id) {
    if (visited.has(cur.parent_id)) break
    visited.add(cur.parent_id)
    const parent = allItems[cur.parent_id]
    if (!parent) break
    titles.unshift(parent.title)
    cur = parent
  }
  return titles.join(' › ')
}

export function childCount(allItems: Record<string, Item>, itemId: string): number {
  let count = 0
  for (const item of Object.values(allItems)) {
    if (isDescendantOf(allItems, item, itemId)) count++
  }
  return count
}

export function doneCount(allItems: Record<string, Item>, itemId: string, doneStatus = 'done'): number {
  let count = 0
  for (const item of Object.values(allItems)) {
    if (item.status === doneStatus && isDescendantOf(allItems, item, itemId)) count++
  }
  return count
}

export function hasChildren(allItems: Record<string, Item>, itemId: string): boolean {
  return Object.values(allItems).some(item => item.parent_id === itemId)
}

function isDescendantOf(allItems: Record<string, Item>, item: Item, ancestorId: string): boolean {
  const visited = new Set([item.$id])
  let cur = item
  while (cur.parent_id) {
    if (cur.parent_id === ancestorId) return true
    if (visited.has(cur.parent_id)) return false
    visited.add(cur.parent_id)
    const parent = allItems[cur.parent_id]
    if (!parent) return false
    cur = parent
  }
  return false
}

export function isHiddenByCollapse(
  allItems: Record<string, Item>,
  item: Item,
  collapsed: Record<string, boolean>
): boolean {
  const visited = new Set([item.$id])
  let cur = item
  while (cur.parent_id) {
    if (visited.has(cur.parent_id)) return false
    visited.add(cur.parent_id)
    const parent = allItems[cur.parent_id]
    if (!parent) return false
    if (collapsed[parent.$id] && parent.status === item.status) return true
    cur = parent
  }
  return false
}

export function orderItemsHierarchically(
  allItems: Record<string, Item>,
  columnItems: Item[]
): Item[] {
  const idSet = new Set(columnItems.map(i => i.$id))
  const roots: Item[] = []
  const childrenOf: Record<string, Item[]> = {}

  for (const item of columnItems) {
    if (!item.parent_id || !idSet.has(item.parent_id)) {
      roots.push(item)
    } else {
      (childrenOf[item.parent_id] ||= []).push(item)
    }
  }

  const byNum = (a: Item, b: Item) => a.display_num - b.display_num
  roots.sort(byNum)
  for (const children of Object.values(childrenOf)) children.sort(byNum)

  const ordered: Item[] = []
  const walk = (item: Item) => {
    ordered.push(item)
    for (const child of childrenOf[item.$id] || []) walk(child)
  }
  for (const root of roots) walk(root)
  return ordered
}
```

- [ ] **Step 3: Run tests**

Run: `cd web && npx vitest run __tests__/lib/hierarchy.test.ts`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add web/lib/hierarchy.ts web/__tests__/lib/hierarchy.test.ts
git commit -m "feat: add web hierarchy helpers — breadcrumb, collapse, ordering"
```

### Task 8: Update KanbanCard with hierarchy features

**Files:**
- Modify: `web/components/board/kanban-card.tsx`

- [ ] **Step 1: Add breadcrumb, left border, collapse indicator, child badge, progress**

Update the card component to:
- Accept `allItems`, `collapsed`, `onToggleCollapse` props
- Render breadcrumb above title via `breadcrumbPath()`
- Apply CSS class `card--epic` / `card--story` / `card--task` for left border
- Show ▼/▶ collapse indicator and child count badge when `hasChildren()`
- Show `N/M done` progress line when item has children

- [ ] **Step 2: Add CSS classes**

Add to the card's Tailwind/CSS:
- `border-l-3 border-fuchsia-500` for epics
- `border-l-3 border-blue-500` for stories
- Breadcrumb: `text-xs opacity-40`
- Badge: inline pill with type-colored background at 20% opacity

- [ ] **Step 3: Commit**

```bash
git add web/components/board/kanban-card.tsx
git commit -m "feat: KanbanCard — breadcrumb, left border, collapse indicator, child badge"
```

### Task 9: Update KanbanBoard with hierarchy filtering and ordering

**Files:**
- Modify: `web/components/board/kanban-board.tsx`

- [ ] **Step 1: Add collapse state and filter items per column**

Add `useState` for collapse map. Before passing items to each column, apply:
1. `isHiddenByCollapse` filter
2. `orderItemsHierarchically` ordering

Remove any existing epic grouping/duplication logic.

- [ ] **Step 2: Pass hierarchy props to KanbanCard**

Pass `allItems`, `collapsed`, `onToggleCollapse` to each card.

- [ ] **Step 3: Run web tests**

Run: `cd web && npx vitest run`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add web/components/board/kanban-board.tsx
git commit -m "feat: KanbanBoard — hierarchy filtering, ordering, collapse state"
```

### Task 10: Final integration verification

- [ ] **Step 1: Run Go full test suite**

Run: `./scripts/test.sh`
Expected: All checks pass

- [ ] **Step 2: Run web tests**

Run: `cd web && npx vitest run`
Expected: All pass

- [ ] **Step 3: Manual TUI smoke test**

```bash
go build -o ob .
./ob init --agent claude
./ob create epic "Auth Rewrite" --assign claude -d "Rewrite auth"
./ob create story "Session Mgmt" --assign claude -d "Sessions" --parent 1
./ob create task "Refactor middleware" --assign claude -d "Refactor" --parent 2
./ob create task "JWT validation" --assign claude -d "JWT" --parent 2
./ob move 4 in-progress
./ob tui
# Verify:
#   - Epic shows magenta left bar, ▼ indicator, "4 items" badge, "0/4 done" progress
#   - Story shows blue left bar, ▼ indicator, "2 items" badge
#   - Tasks show breadcrumb "Auth Rewrite › Session Mgmt"
#   - Task #4 in in-progress column shows breadcrumb, NO ghost epic header
#   - Space on epic collapses children in backlog only
#   - Task #4 in in-progress remains visible
```

- [ ] **Step 4: Commit any fixes**

---

## Summary

| Chunk | Tasks | Key Files |
|-------|-------|-----------|
| 1: Hierarchy Helpers | 1 | `hierarchy.go`, `hierarchy_test.go` |
| 2: TUI Rendering | 2-6 | `card.go`, `board.go`, `styles.go`, `app.go`, golden files |
| 3: Web Implementation | 7-10 | `hierarchy.ts`, `kanban-card.tsx`, `kanban-board.tsx` |

**Total: 10 tasks, ~50 steps**

Each task is independently testable and committable. TDD throughout.
