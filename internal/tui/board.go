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
	w := a.columnWidth()
	for i, colName := range a.columns {
		items := a.visibleItemsInColumn(i)
		nativeCount := 0
		for _, it := range items {
			if it.Status == colName {
				nativeCount++
			}
		}
		var cardViews []string
		for _, item := range items {
			card := a.renderCard(item, a.isItemAtCursor(item))
			if a.isFirstLevelChild(item, colName) {
				card = indentCard(card, 2)
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
	header := fmt.Sprintf("  Obeya Board: %s", a.board.Name)
	help := helpStyle.Render(
		"  h/l:columns  j/k:items  v:desc  V:review  m:move  a:assign  c:create  d:delete  " +
			"p:priority  R:reviewed  x:hide  P:past-reviews  Enter:detail  G:dag  D:dash  q:quit",
	)
	return header + "\n" + board + "\n" + help
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

// columnWidth returns the content width for each column.
func (a App) columnWidth() int {
	if a.width == 0 || len(a.columns) == 0 {
		return 22
	}
	n := len(a.columns)
	w := (a.width + 1 - 3*n) / n
	if w < 18 {
		return 18
	}
	if w > 28 {
		return 28
	}
	return w
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

// isFirstLevelChild returns true if the item's direct parent is in the same column.
// Only the first level of nesting gets indentation — deeper children use breadcrumbs.
func (a App) isFirstLevelChild(item *domain.Item, colName string) bool {
	if item.ParentID == "" {
		return false
	}
	parent, ok := a.board.Items[item.ParentID]
	if !ok {
		return false
	}
	return parent.Status == colName
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
