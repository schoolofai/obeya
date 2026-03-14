package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

func (a App) renderBoard() string {
	var cols []string
	for i, colName := range a.columns {
		cols = append(cols, a.renderColumn(i, colName))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	header := fmt.Sprintf("  Obeya Board: %s", a.board.Name)
	help := helpStyle.Render(
		"  h/l:columns  j/k:items  v:desc  m:move  a:assign  c:create  d:delete  " +
			"p:priority  Enter:detail  Space:collapse  /:search  r:reload  q:quit",
	)

	return header + "\n" + board + "\n" + help
}

func buildScrollbar(trackH, offset, totalLines int) []string {
	track := make([]string, trackH)

	thumbH := (trackH * trackH) / totalLines
	if thumbH < 1 {
		thumbH = 1
	}

	maxOffset := totalLines - trackH
	thumbPos := 0
	if maxOffset > 0 {
		thumbPos = (offset * (trackH - thumbH)) / maxOffset
	}
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos+thumbH > trackH {
		thumbPos = trackH - thumbH
	}

	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	trackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	for i := 0; i < trackH; i++ {
		if i >= thumbPos && i < thumbPos+thumbH {
			track[i] = thumbStyle.Render("┃")
		} else {
			track[i] = trackStyle.Render("│")
		}
	}
	return track
}

func (a App) renderColumn(colIdx int, colName string) string {
	items := a.visibleItemsInColumn(colIdx)
	isActive := colIdx == a.cursorCol
	w := a.columnWidth()

	// Column header — count only items native to this column.
	nativeCount := 0
	for _, it := range items {
		if it.Status == colName {
			nativeCount++
		}
	}
	count := fmt.Sprintf(" (%d)", nativeCount)
	var header string
	if isActive {
		header = activeColHeader.Render(strings.ToUpper(colName) + count)
	} else {
		header = inactiveColHeader.Render(strings.ToUpper(colName) + count)
	}
	header = padToWidth(header, w)

	// Card content with per-column scrolling
	cardViews := a.renderGroupedCards(items, colIdx)
	viewH := a.contentViewHeight()

	var contentLines []string
	if len(cardViews) == 0 {
		// Empty column — fill with blank lines
		for i := 0; i < viewH; i++ {
			contentLines = append(contentLines, strings.Repeat(" ", w))
		}
	} else {
		allCards := strings.Join(cardViews, "\n")
		cardLines := strings.Split(allCards, "\n")

		if viewH > 0 && len(cardLines) > viewH {
			// Scroll: clip to viewport using this column's offset
			offset := a.colScrollY[colIdx]
			maxOffset := len(cardLines) - viewH
			if offset > maxOffset {
				offset = maxOffset
			}
			if offset < 0 {
				offset = 0
			}
			end := offset + viewH
			if end > len(cardLines) {
				end = len(cardLines)
			}
			contentLines = make([]string, end-offset)
			copy(contentLines, cardLines[offset:end])
		} else {
			contentLines = cardLines
		}
	}

	// Pad all lines to uniform width and pad to viewH
	for i, line := range contentLines {
		contentLines[i] = padToWidth(line, w)
	}
	for len(contentLines) < viewH {
		contentLines = append(contentLines, strings.Repeat(" ", w))
	}

	content := header + "\n" + strings.Join(contentLines, "\n")
	if isActive {
		return activeColumnStyle.Render(content)
	}
	return columnStyle.Render(content)
}

func (a App) visibleItemsInColumn(colIdx int) []*domain.Item {
	if a.board == nil || colIdx < 0 || colIdx >= len(a.columns) {
		return nil
	}
	colName := a.columns[colIdx]

	// Collect all items in this column.
	var colItems []*domain.Item
	for _, item := range a.board.Items {
		if item.Status == colName {
			colItems = append(colItems, item)
		}
	}
	sort.Slice(colItems, func(i, j int) bool {
		return colItems[i].DisplayNum > colItems[j].DisplayNum
	})

	// Include cross-column parent epics so they are navigable.
	crossColEpic := map[string]bool{}
	for _, item := range colItems {
		epicID := findEpicAncestor(a.board, item)
		if epicID == "" || epicID == item.ID || crossColEpic[epicID] {
			continue
		}
		epic, exists := a.board.Items[epicID]
		if exists && epic.Status != colName {
			colItems = append(colItems, epic)
			crossColEpic[epicID] = true
		}
	}

	// Re-sort after adding cross-column epics.
	sort.Slice(colItems, func(i, j int) bool {
		return colItems[i].DisplayNum > colItems[j].DisplayNum
	})

	// Filter collapsed children.
	var filtered []*domain.Item
	for _, item := range colItems {
		epicID := findEpicAncestor(a.board, item)
		if epicID == "" || epicID == item.ID {
			filtered = append(filtered, item)
			continue
		}
		if !a.collapsed[epicID] {
			filtered = append(filtered, item)
			continue
		}
		// Collapsed child — epic (in-column or cross-column) represents the group.
		// Filter out all children.
	}

	// Reorder to match visual render order: epic groups first, then orphans.
	return a.renderOrderItems(filtered, colIdx)
}

// renderOrderItems reorders items to match the visual layout produced by
// renderGroupedCards: epic groups (epic card first, then children) followed
// by orphan items. This keeps cursor navigation consistent with display.
func (a App) renderOrderItems(items []*domain.Item, colIdx int) []*domain.Item {
	type group struct {
		epicID   string
		epic     *domain.Item
		children []*domain.Item
	}

	groupOrder := []string{}
	groups := map[string]*group{}
	var orphans []*domain.Item

	for _, item := range items {
		epicID := findEpicAncestor(a.board, item)
		if epicID == "" {
			if item.Type == domain.ItemTypeEpic {
				if _, ok := groups[item.ID]; !ok {
					groups[item.ID] = &group{epicID: item.ID, epic: item}
					groupOrder = append(groupOrder, item.ID)
				}
			} else {
				orphans = append(orphans, item)
			}
			continue
		}
		g, ok := groups[epicID]
		if !ok {
			g = &group{epicID: epicID}
			groups[epicID] = g
			groupOrder = append(groupOrder, epicID)
		}
		if item.ID == epicID {
			g.epic = item
		} else {
			g.children = append(g.children, item)
		}
	}

	var ordered []*domain.Item
	for _, eid := range groupOrder {
		g := groups[eid]
		if g.epic != nil {
			ordered = append(ordered, g.epic)
		}
		ordered = append(ordered, g.children...)
	}
	ordered = append(ordered, orphans...)
	return ordered
}

func (a App) renderBoardWithOverlay(overlay string) string {
	board := a.renderBoard()
	overlayRendered := overlayStyle.Render(overlay)
	return board + "\n\n" + overlayRendered
}

// columnWidth returns the content width for each column.
// Layout per column: border(2) + content(w) + marginRight(1).
// Total: n*(w+3) - 1 <= terminal_width  =>  w = (terminal_width + 1 - 3*n) / n
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
