package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

const colWidth = 18

// View renders the board as a text-based column layout.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}
	if m.board == nil {
		return "Loading board..."
	}

	var sb strings.Builder
	m.renderHeader(&sb)
	m.renderColumnHeaders(&sb)
	m.renderItems(&sb)
	m.renderHelpBar(&sb)
	return sb.String()
}

func (m Model) renderHeader(sb *strings.Builder) {
	sb.WriteString(fmt.Sprintf("  Obeya Board: %s\n", m.board.Name))
	sb.WriteString(strings.Repeat("\u2500", 80) + "\n")
}

func (m Model) renderColumnHeaders(sb *strings.Builder) {
	for i, col := range m.columns {
		marker := " "
		if i == m.cursorCol {
			marker = ">"
		}
		header := fmt.Sprintf("%s%-*s", marker, colWidth-1, strings.ToUpper(col))
		sb.WriteString(header)
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("\u2500", colWidth*len(m.columns)) + "\n")
}

func (m Model) renderItems(sb *strings.Builder) {
	columnItems := m.sortedColumnItems()
	maxRows := maxItemCount(columnItems)

	for row := 0; row < maxRows; row++ {
		for col := 0; col < len(m.columns); col++ {
			if row < len(columnItems[col]) {
				item := columnItems[col][row]
				selected := col == m.cursorCol && row == m.cursorRow
				sb.WriteString(formatCell(item, selected, colWidth))
			} else {
				sb.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		sb.WriteString("\n")
	}
}

func (m Model) renderHelpBar(sb *strings.Builder) {
	sb.WriteString("\n")
	sb.WriteString("  h/l: move columns  j/k: move rows  r: reload  q: quit\n")
}

func (m Model) sortedColumnItems() [][]*domain.Item {
	columnItems := make([][]*domain.Item, len(m.columns))
	for i, col := range m.columns {
		items := m.itemsInColumn(col)
		sort.Slice(items, func(a, b int) bool {
			return items[a].DisplayNum < items[b].DisplayNum
		})
		columnItems[i] = items
	}
	return columnItems
}

func maxItemCount(columnItems [][]*domain.Item) int {
	max := 0
	for _, items := range columnItems {
		if len(items) > max {
			max = len(items)
		}
	}
	return max
}

func formatCell(item *domain.Item, selected bool, width int) string {
	prefix := " "
	if selected {
		prefix = ">"
	}
	label := fmt.Sprintf("#%d %s", item.DisplayNum, truncate(item.Title, width-6))
	if len(item.BlockedBy) > 0 {
		label += "!"
	}
	return fmt.Sprintf("%s%-*s", prefix, width-1, label)
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
