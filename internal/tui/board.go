package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

const humanReviewColName = "human-review"

func isHumanReviewColumn(columns []string, colIdx int) bool {
	return colIdx >= 0 && colIdx < len(columns) && columns[colIdx] == humanReviewColName
}

func (a App) renderBoard() string {
	var cols []string
	widths := a.columnWidths()
	for i, colName := range a.columns {
		w := widths[i]
		items := a.visibleItemsInColumn(i)
		nativeCount := 0
		for _, it := range items {
			if it.Status == colName {
				nativeCount++
			}
		}
		var cardViews []string
		for _, item := range items {
			indent := 0
			if a.isFirstLevelChild(item, colName) {
				indent = 2
			}
			card := a.renderCardWithWidth(item, a.isItemAtCursor(item), w-indent)
			if indent > 0 {
				card = indentCard(card, indent)
			}
			cardViews = append(cardViews, card)
		}
		cardContent := ""
		if len(cardViews) > 0 {
			allCards := strings.Join(cardViews, "\n")
			cardLines := strings.Split(allCards, "\n")
			for j, line := range cardLines {
				cardLines[j] = padToWidth(line, w)
			}
			cardContent = strings.Join(cardLines, "\n")
		}
		if i < len(a.colModels) {
			a.colModels[i].SetContent(cardContent)
			cols = append(cols, a.colModels[i].View(nativeCount))
		}
	}
	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	// Clamp each board line to terminal width to prevent overflow
	board = a.clampToTerminalWidth(board)

	header := fmt.Sprintf("  Obeya Board: %s", a.board.Name)
	help := helpStyle.Render(
		"  hjkl:nav  v:desc  V:review  m:move  a:assign  c:create  d:del  p:pri  R:reviewed  x:hide  P:past  Enter:detail  G:dag  D:dash  q:quit",
	)
	result := header + "\n" + board + "\n" + help
	return a.clampToTerminalWidth(result)
}

func (a App) visibleItemsInColumn(colIdx int) []*domain.Item {
	if a.board == nil || colIdx < 0 || colIdx >= len(a.columns) {
		return nil
	}
	colName := a.columns[colIdx]

	// Virtual human-review column: show items awaiting review.
	if colName == humanReviewColName {
		return a.humanReviewItems()
	}

	// Collect items in this column
	var colItems []*domain.Item
	for _, item := range a.board.Items {
		if item.Status == colName {
			colItems = append(colItems, item)
		}
	}

	// Order hierarchically: parents before children, depth-first by DisplayNum
	return orderItemsHierarchically(a.board, colItems)
}

func (a App) renderBoardWithOverlay(overlay string) string {
	board := a.renderBoard()
	overlayRendered := overlayStyle.Render(overlay)
	return board + "\n\n" + overlayRendered
}

// columnWidth returns the content width for the current cursor column.
// Delegates to columnWidthFor for per-column proportional sizing.
func (a App) columnWidth() int {
	return a.columnWidthFor(a.cursorCol)
}

// columnWidths computes per-column widths proportionally.
// Empty columns shrink to a narrow strip; populated columns share the rest.
func (a App) columnWidths() []int {
	n := len(a.columns)
	if n == 0 || a.width == 0 {
		widths := make([]int, n)
		for i := range widths {
			widths[i] = 22
		}
		return widths
	}

	const minEmpty = 6  // just enough for column header abbreviation
	const minFull = 16
	// Each column: border(2) + content(w) + marginRight(1) = w + 3
	// All columns including last get marginRight(1)
	// Total: sum(w_i) + n*3 <= terminal_width
	// => sum(w_i) <= terminal_width - 3*n

	itemCounts := a.itemCountsPerColumn()
	available := a.width - 3*n

	var emptyCount, popCount int
	for _, c := range itemCounts {
		if c == 0 {
			emptyCount++
		} else {
			popCount++
		}
	}

	widths := make([]int, n)

	if popCount == 0 {
		each := available / n
		for i := range widths {
			widths[i] = each
		}
		return widths
	}

	// Give empty columns minimum, populated columns share the rest equally — no cap
	emptyTotal := emptyCount * minEmpty
	popAvail := available - emptyTotal
	perPop := popAvail / popCount
	if perPop < minFull {
		perPop = minFull
	}

	for i, c := range itemCounts {
		if c == 0 {
			widths[i] = minEmpty
		} else {
			widths[i] = perPop
		}
	}
	return widths
}

// columnWidthFor returns the content width for a specific column.
func (a App) columnWidthFor(colIdx int) int {
	widths := a.columnWidths()
	if colIdx >= 0 && colIdx < len(widths) {
		return widths[colIdx]
	}
	return 22
}

// itemCountsPerColumn returns the item count for each column.
func (a App) itemCountsPerColumn() []int {
	counts := make([]int, len(a.columns))
	for i, colName := range a.columns {
		if colName == humanReviewColName {
			counts[i] = len(a.humanReviewItems())
		} else if a.board != nil {
			for _, item := range a.board.Items {
				if item.Status == colName {
					counts[i]++
				}
			}
		}
	}
	return counts
}

func cyclePriority(current string) string {
	order := []string{"low", "medium", "high", "critical"}
	for i, p := range order {
		if p == current {
			return order[(i+1)%len(order)]
		}
	}
	return "medium"
}

func userNames(board *domain.Board) []string {
	var names []string
	for _, u := range board.Users {
		names = append(names, u.Name)
	}
	sort.Strings(names)
	return names
}

func itemPickerLabels(board *domain.Board, excludeID string) []string {
	var items []*domain.Item
	for _, item := range board.Items {
		if item.ID != excludeID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayNum < items[j].DisplayNum
	})
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = fmt.Sprintf("#%d %s", item.DisplayNum, item.Title)
	}
	return labels
}

func resolveUserName(board *domain.Board, userID string) string {
	if u, ok := board.Users[userID]; ok {
		return u.Name
	}
	return userID
}

// clampToTerminalWidth truncates each line to terminal width using lipgloss
// for ANSI-aware and unicode-aware width measurement.
func (a App) clampToTerminalWidth(content string) string {
	if a.width <= 0 {
		return content
	}
	maxW := a.width
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for lipgloss.Width(lines[i]) > maxW {
			// Remove one rune at a time from the end until it fits
			runes := []rune(lines[i])
			if len(runes) == 0 {
				break
			}
			lines[i] = string(runes[:len(runes)-1])
		}
		_ = line
	}
	return strings.Join(lines, "\n")
}

// isFirstLevelChild returns true if the item's direct parent is a root in
// this column (parent is in this column but parent's parent is NOT).
// Only this first level gets indentation — deeper children use breadcrumbs.
func (a App) isFirstLevelChild(item *domain.Item, colName string) bool {
	if item.ParentID == "" {
		return false
	}
	parent, ok := a.board.Items[item.ParentID]
	if !ok {
		return false
	}
	if parent.Status != colName {
		return false
	}
	// Parent is in this column — check if parent is a root (its own parent NOT in this column)
	if parent.ParentID == "" {
		return true // parent has no parent → it's a root → indent this item
	}
	grandparent, ok := a.board.Items[parent.ParentID]
	if !ok {
		return true // grandparent missing → parent is effectively a root
	}
	return grandparent.Status != colName // indent only if grandparent is in a different column
}

// indentCard prepends spaces to each line of a rendered card.
func indentCard(card string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(card, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// humanReviewItems returns items that need human review, sorted by confidence ascending.
func (a App) humanReviewItems() []*domain.Item {
	var items []*domain.Item
	for _, item := range a.board.Items {
		if item.Status != "done" || item.ReviewContext == nil {
			continue
		}
		if item.HumanReview != nil && item.HumanReview.Status == "hidden" {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		ci := confidenceValue(items[i].Confidence)
		cj := confidenceValue(items[j].Confidence)
		if ci != cj {
			return ci < cj
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return items
}

func confidenceValue(c *int) int {
	if c == nil {
		return -1
	}
	return *c
}

// hasReviewableItems returns true if any items need human review.
func (a App) hasReviewableItems() bool {
	if a.board == nil {
		return false
	}
	for _, item := range a.board.Items {
		if item.Status == "done" && item.ReviewContext != nil {
			if item.HumanReview == nil || item.HumanReview.Status != "hidden" {
				return true
			}
		}
	}
	return false
}
