package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

// DetailModel renders a full-screen tabbed detail view for a single item.
type DetailModel struct {
	item      *domain.Item
	board     *domain.Board
	plans     []*domain.Plan
	activeTab detailTab
	scrollY   int
	width     int
	height    int
}

func newDetailModel(item *domain.Item, board ...*domain.Board) DetailModel {
	d := DetailModel{item: item}
	if len(board) > 0 {
		d.board = board[0]
		d.plans = plansForItem(d.board, item.ID)
	}
	return d
}

func (d *DetailModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DetailModel) NextTab() {
	d.activeTab = (d.activeTab + 1) % 3
	d.scrollY = 0
}

func (d *DetailModel) PrevTab() {
	d.activeTab = (d.activeTab + 2) % 3
	d.scrollY = 0
}

func (d *DetailModel) ScrollDown() {
	d.scrollY++
}

func (d *DetailModel) ScrollUp() {
	if d.scrollY > 0 {
		d.scrollY--
	}
}

func (d DetailModel) View() string {
	if d.item == nil {
		return "No item selected"
	}

	innerWidth := d.width - 8
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder
	d.renderHeader(&b)
	d.renderTabBar(&b)
	content := d.renderTabContent()

	lines := strings.Split(content, "\n")
	viewH := d.height - 8
	if viewH < 1 {
		viewH = 1
	}
	if d.scrollY > len(lines)-viewH {
		d.scrollY = len(lines) - viewH
	}
	if d.scrollY < 0 {
		d.scrollY = 0
	}
	end := d.scrollY + viewH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[d.scrollY:end]
	b.WriteString(strings.Join(visible, "\n"))
	b.WriteString("\n\n")
	d.renderHelpBar(&b)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2).
		Width(d.width - 4).
		Height(d.height - 2)

	return borderStyle.Render(b.String())
}

func (d DetailModel) renderHeader(b *strings.Builder) {
	header := fmt.Sprintf(" #%d %s", d.item.DisplayNum, d.item.Title)
	b.WriteString(typeStyle(string(d.item.Type)).Bold(true).Render(header))
	b.WriteString("\n")
}

func (d DetailModel) renderTabBar(b *strings.Builder) {
	tabs := []struct {
		label string
		tab   detailTab
	}{
		{"Fields", tabFields},
		{"Plan", tabPlan},
		{"History", tabHistory},
	}

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true).
		Underline(true)

	for i, t := range tabs {
		if i > 0 {
			b.WriteString("  ")
		}
		if d.activeTab == t.tab {
			b.WriteString(activeStyle.Render("[" + t.label + "]"))
		} else {
			b.WriteString(helpStyle.Render("[" + t.label + "]"))
		}
	}
	b.WriteString("\n\n")
}

func (d DetailModel) renderTabContent() string {
	switch d.activeTab {
	case tabFields:
		return d.renderFieldsTab()
	case tabPlan:
		return d.renderPlanTab()
	case tabHistory:
		return d.renderHistoryTab()
	}
	return ""
}

func (d DetailModel) renderFieldsTab() string {
	var b strings.Builder
	writeField(&b, "Type", string(d.item.Type))
	writeField(&b, "Status", d.item.Status)
	writeField(&b, "Priority", priorityIndicator(string(d.item.Priority))+" "+string(d.item.Priority))

	if d.item.Assignee != "" {
		name := d.resolveUserName(d.item.Assignee)
		writeField(&b, "Assignee", assigneeStyle.Render("@"+name))
	}

	d.writeParentField(&b)
	d.writeTagsField(&b)
	d.writeBlockedField(&b)
	d.writeDescriptionField(&b)
	d.writeChildrenField(&b)

	return b.String()
}

func (d DetailModel) writeParentField(b *strings.Builder) {
	if d.item.ParentID == "" || d.board == nil {
		return
	}
	parent, ok := d.board.Items[d.item.ParentID]
	if !ok {
		return
	}
	writeField(b, "Parent", fmt.Sprintf("#%d %s", parent.DisplayNum, parent.Title))
}

func (d DetailModel) writeTagsField(b *strings.Builder) {
	if len(d.item.Tags) == 0 {
		return
	}
	writeField(b, "Tags", strings.Join(d.item.Tags, ", "))
}

func (d DetailModel) writeBlockedField(b *strings.Builder) {
	if len(d.item.BlockedBy) == 0 {
		writeField(b, "Blocked", "—")
		return
	}
	refs := formatBlockedRefs(d.board, d.item.BlockedBy)
	writeField(b, "Blocked", blockedStyle.Render(refs))
}

func (d DetailModel) writeDescriptionField(b *strings.Builder) {
	if d.item.Description == "" {
		return
	}
	b.WriteString("\n  Description:\n")
	for _, line := range strings.Split(d.item.Description, "\n") {
		b.WriteString("    " + line + "\n")
	}
}

func (d DetailModel) writeChildrenField(b *strings.Builder) {
	if d.board == nil {
		return
	}
	children := findChildren(d.board, d.item.ID)
	if len(children) == 0 {
		return
	}
	b.WriteString("\n  Children:\n")
	for _, c := range children {
		line := fmt.Sprintf("    #%-4d %-8s %-12s %s\n",
			c.DisplayNum, string(c.Type), c.Status, c.Title)
		b.WriteString(line)
	}
}

func (d DetailModel) renderPlanTab() string {
	if len(d.plans) == 0 {
		return "  No plan linked."
	}
	var b strings.Builder
	for i, p := range d.plans {
		if i > 0 {
			b.WriteString("\n  ────────────────────\n\n")
		}
		header := fmt.Sprintf("  Plan #%d: %s", p.DisplayNum, p.Title)
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
		b.WriteString("\n\n")
		for _, line := range strings.Split(p.Content, "\n") {
			b.WriteString("    " + line + "\n")
		}
	}
	return b.String()
}

func (d DetailModel) renderHistoryTab() string {
	if len(d.item.History) == 0 {
		return "  No history."
	}
	var b strings.Builder
	for _, h := range d.item.History {
		ts := h.Timestamp.Format("2006-01-02 15:04")
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n", ts, h.Action, h.Detail))
	}
	return b.String()
}

func (d DetailModel) renderHelpBar(b *strings.Builder) {
	b.WriteString(helpStyle.Render("Tab/Shift-Tab:switch  j/k:scroll  m:move  a:assign  p:priority  Esc:back"))
}

func (d DetailModel) resolveUserName(userID string) string {
	if d.board != nil && d.board.Users != nil {
		if u, ok := d.board.Users[userID]; ok {
			return u.Name
		}
	}
	return userID
}

func writeField(b *strings.Builder, label, value string) {
	b.WriteString(fmt.Sprintf("  %-10s %s\n", label+":", value))
}

func formatBlockedRefs(board *domain.Board, ids []string) string {
	refs := make([]string, 0, len(ids))
	for _, id := range ids {
		if board != nil {
			if item, ok := board.Items[id]; ok {
				refs = append(refs, "#"+strconv.Itoa(item.DisplayNum))
				continue
			}
		}
		refs = append(refs, id)
	}
	return strings.Join(refs, ", ")
}

func findChildren(board *domain.Board, parentID string) []*domain.Item {
	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children
}

func plansForItem(board *domain.Board, itemID string) []*domain.Plan {
	if board == nil || board.Plans == nil {
		return nil
	}
	var result []*domain.Plan
	for _, p := range board.Plans {
		for _, linked := range p.LinkedItems {
			if linked == itemID {
				result = append(result, p)
				break
			}
		}
	}
	return result
}
