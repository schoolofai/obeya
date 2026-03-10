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
		"  h/l:columns  j/k:items  m:move  a:assign  c:create  d:delete  " +
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

	// Card content with per-column scrolling
	cardViews := a.renderGroupedCards(items, colIdx)
	viewH := a.contentViewHeight()

	var cardContent string
	if len(cardViews) == 0 {
		// Empty column — pad to viewport height
		if viewH > 0 {
			emptyLines := make([]string, viewH)
			cardContent = strings.Join(emptyLines, "\n")
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

			visible := make([]string, end-offset)
			copy(visible, cardLines[offset:end])

			// Per-column scrollbar
			scrollbar := buildScrollbar(viewH, offset, len(cardLines))
			for i := range visible {
				if i < len(scrollbar) {
					visible[i] = visible[i] + " " + scrollbar[i]
				}
			}
			cardContent = strings.Join(visible, "\n")
		} else if viewH > 0 && len(cardLines) < viewH {
			// Pad shorter columns to uniform height
			for len(cardLines) < viewH {
				cardLines = append(cardLines, "")
			}
			cardContent = strings.Join(cardLines, "\n")
		} else {
			cardContent = allCards
		}
	}

	content := header + "\n" + cardContent
	if isActive {
		return activeColumnStyle.Width(w).Render(content)
	}
	return columnStyle.Width(w).Render(content)
}

func (a App) renderCard(item *domain.Item, selected bool) string {
	w := a.columnWidth()
	titleMax := w - 6
	if titleMax < 10 {
		titleMax = 10
	}

	line1 := fmt.Sprintf("#%d %s", item.DisplayNum, truncate(item.Title, titleMax))
	typLabel := typeStyle(string(item.Type)).Render(string(item.Type))
	line2 := fmt.Sprintf("%s %s", typLabel, priorityIndicator(string(item.Priority)))

	lines := []string{line1, line2}

	badge := a.parentBadge(item)
	if badge != "" {
		lines = append(lines, badge)
	}

	var line4Parts []string
	if item.Assignee != "" {
		name := resolveUserName(a.board, item.Assignee)
		line4Parts = append(line4Parts, assigneeStyle.Render("@"+name))
	}
	if len(item.BlockedBy) > 0 {
		line4Parts = append(line4Parts, blockedStyle.Render("[!]"))
	}
	if len(line4Parts) > 0 {
		lines = append(lines, strings.Join(line4Parts, " "))
	}

	content := strings.Join(lines, "\n")
	if selected {
		return selectedCardStyle.Render(content)
	}
	return cardStyle.Render(content)
}

func (a App) renderGroupedCards(items []*domain.Item, colIdx int) []string {
	colName := a.columns[colIdx]

	type epicGroup struct {
		epicID    string
		epicItem  *domain.Item
		epicInCol bool // true when the epic card itself is navigable here
		children  []*domain.Item
	}

	groupOrder := []string{}
	groups := map[string]*epicGroup{}
	var orphans []*domain.Item

	for _, item := range items {
		epicID := findEpicAncestor(a.board, item)
		if epicID == "" {
			if item.Type == domain.ItemTypeEpic {
				if _, ok := groups[item.ID]; !ok {
					groups[item.ID] = &epicGroup{
						epicID:    item.ID,
						epicItem:  item,
						epicInCol: true,
					}
					groupOrder = append(groupOrder, item.ID)
				}
			} else {
				orphans = append(orphans, item)
			}
			continue
		}
		g, ok := groups[epicID]
		if !ok {
			g = &epicGroup{epicID: epicID}
			groups[epicID] = g
			groupOrder = append(groupOrder, epicID)
			if epic, exists := a.board.Items[epicID]; exists {
				g.epicItem = epic
			}
		}
		if item.ID == epicID {
			g.epicItem = item
			g.epicInCol = true
		} else {
			g.children = append(g.children, item)
		}
	}

	var views []string

	for _, eid := range groupOrder {
		g := groups[eid]
		epic := g.epicItem
		collapsed := a.collapsed[eid]

		if epic != nil {
			epicNum := epic.DisplayNum
			epicTitle := truncate(epic.Title, a.columnWidth()-10)
			groupSelected := a.isEpicGroupAtCursor(eid)
			style := epicGroupStyle
			if groupSelected {
				style = selectedEpicGroupStyle
			}
			isCrossCol := epic.Status != colName
			if collapsed {
				total := len(g.children)
				if g.epicInCol {
					total++
				}
				hdr := fmt.Sprintf("\u25b6 #%d %s (%d items)", epicNum, epicTitle, total)
				if isCrossCol {
					hdr += crossColBadge
				}
				views = append(views, style.Render(hdr))
				if g.epicInCol {
					views = append(views, a.renderCard(epic, a.isItemAtCursor(epic)))
				}
			} else {
				hdr := fmt.Sprintf("\u25bc #%d %s", epicNum, epicTitle)
				if isCrossCol {
					hdr += crossColBadge
				}
				views = append(views, style.Render(hdr))
				if g.epicInCol {
					views = append(views, a.renderCard(epic, a.isItemAtCursor(epic)))
				}
				for _, child := range g.children {
					views = append(views, a.renderCard(child, a.isItemAtCursor(child)))
				}
			}
		} else {
			for _, child := range g.children {
				views = append(views, a.renderCard(child, a.isItemAtCursor(child)))
			}
		}
	}

	for _, item := range orphans {
		views = append(views, a.renderCard(item, a.isItemAtCursor(item)))
	}

	return views
}

func (a App) isItemAtCursor(item *domain.Item) bool {
	sel := a.selectedItem()
	return sel != nil && sel.ID == item.ID
}

func (a App) isEpicGroupAtCursor(epicID string) bool {
	sel := a.selectedItem()
	if sel == nil {
		return false
	}
	if sel.ID == epicID {
		return true
	}
	ancestor := findEpicAncestor(a.board, sel)
	return ancestor == epicID
}

func (a App) parentBadge(item *domain.Item) string {
	if item.ParentID == "" {
		return ""
	}
	parent, ok := a.board.Items[item.ParentID]
	if !ok {
		return ""
	}
	if parent.Type == domain.ItemTypeEpic || parent.Status != item.Status {
		return lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf("\u2191 #%d %s", parent.DisplayNum, truncate(parent.Title, a.columnWidth()-10)),
		)
	}
	return ""
}

func (a App) isCollapsedChild(item *domain.Item) bool {
	epicID := findEpicAncestor(a.board, item)
	if epicID == "" || epicID == item.ID {
		return false
	}
	return a.collapsed[epicID]
}

func findEpicAncestor(board *domain.Board, item *domain.Item) string {
	if item.Type == domain.ItemTypeEpic {
		return item.ID
	}
	visited := map[string]bool{}
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
		if parent.Type == domain.ItemTypeEpic {
			return parent.ID
		}
		cur = parent
	}
	return ""
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
		return colItems[i].DisplayNum < colItems[j].DisplayNum
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
		return colItems[i].DisplayNum < colItems[j].DisplayNum
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

func (a App) columnWidth() int {
	if a.width == 0 || len(a.columns) == 0 {
		return 24
	}
	w := (a.width - 2) / len(a.columns)
	if w < 20 {
		return 20
	}
	if w > 30 {
		return 30
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
