package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

// TreeNode represents a node in the past-reviews hierarchy.
type TreeNode struct {
	Item       *domain.Item
	Children   []*TreeNode
	IsReviewed bool // false = structural-only ancestor
}

// PastReviewsModel renders a hierarchical tree of all reviewed items.
type PastReviewsModel struct {
	nodes     []*TreeNode
	board     *domain.Board
	cursor    int
	flatList  []*domain.Item
	collapsed map[string]bool
	scrollY   int
	width     int
	height    int
}

func newPastReviewsModel(board *domain.Board) PastReviewsModel {
	nodes := BuildReviewTree(board)
	collapsed := map[string]bool{}
	flat := flattenTreeVisible(nodes, collapsed)
	return PastReviewsModel{
		nodes:     nodes,
		board:     board,
		flatList:  flat,
		collapsed: collapsed,
	}
}

// BuildReviewTree creates a hierarchical tree of reviewed items with their ancestors.
func BuildReviewTree(board *domain.Board) []*TreeNode {
	reviewed := map[string]bool{}
	for _, item := range board.Items {
		if item.HumanReview != nil {
			reviewed[item.ID] = true
		}
	}
	if len(reviewed) == 0 {
		return nil
	}

	nodeMap := map[string]*TreeNode{}
	var roots []*TreeNode

	var ensureNode func(id string) *TreeNode
	ensureNode = func(id string) *TreeNode {
		if n, ok := nodeMap[id]; ok {
			return n
		}
		item, ok := board.Items[id]
		if !ok {
			return nil
		}
		n := &TreeNode{Item: item, IsReviewed: reviewed[id]}
		nodeMap[id] = n

		if item.ParentID != "" {
			parent := ensureNode(item.ParentID)
			if parent != nil {
				parent.Children = append(parent.Children, n)
				return n
			}
		}
		roots = append(roots, n)
		return n
	}

	for id := range reviewed {
		ensureNode(id)
	}

	sortNodes(roots)
	return roots
}

func sortNodes(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Item.DisplayNum < nodes[j].Item.DisplayNum
	})
	for _, n := range nodes {
		sortNodes(n.Children)
	}
}

func flattenTreeVisible(nodes []*TreeNode, collapsed map[string]bool) []*domain.Item {
	var flat []*domain.Item
	var walk func([]*TreeNode)
	walk = func(ns []*TreeNode) {
		for _, n := range ns {
			flat = append(flat, n.Item)
			if !collapsed[n.Item.ID] {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return flat
}

func (m PastReviewsModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Past Reviews") +
		lipgloss.NewStyle().Faint(true).Render("  [Esc close  Space collapse/expand]")

	if len(m.nodes) == 0 {
		content := lipgloss.NewStyle().Faint(true).Render("  No reviewed items yet.")
		return m.renderBorder(title + "\n\n" + content)
	}

	lines := m.renderAllNodes()
	content := strings.Join(lines, "\n")
	return m.renderBorder(title + "\n\n" + content)
}

func (m PastReviewsModel) renderAllNodes() []string {
	var lines []string
	for _, n := range m.nodes {
		lines = append(lines, m.renderRootNode(n))
		lines = append(lines, m.renderChildren(n.Children, "  ")...)
	}
	return lines
}

func (m PastReviewsModel) renderRootNode(n *TreeNode) string {
	check, style := reviewNodeStyle(n)
	typeLabel := strings.ToTitle(string(n.Item.Type)[:1]) + string(n.Item.Type)[1:]
	collapseIndicator := m.collapseIndicator(n)
	label := fmt.Sprintf("%s%s%s #%d %s", collapseIndicator, check, typeLabel, n.Item.DisplayNum, n.Item.Title)
	if m.isAtCursor(n.Item.ID) {
		return lipgloss.NewStyle().Bold(true).Reverse(true).Render(label)
	}
	return style.Render(label)
}

func (m PastReviewsModel) renderChildren(nodes []*TreeNode, indent string) []string {
	var lines []string
	for i, n := range nodes {
		if m.collapsed[n.Item.ID] && len(n.Children) > 0 {
			continue // skip collapsed children handled by flattenTreeVisible
		}
		prefix, childIndent := treeConnectors(i, len(nodes), indent)
		check, style := reviewNodeStyle(n)
		collapseInd := m.collapseIndicator(n)
		label := fmt.Sprintf("%s%s%s%s#%d %s",
			indent, prefix, collapseInd, check,
			n.Item.DisplayNum, n.Item.Title)
		if m.isAtCursor(n.Item.ID) {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Reverse(true).Render(label))
		} else {
			lines = append(lines, style.Render(label))
		}
		if !m.collapsed[n.Item.ID] {
			lines = append(lines, m.renderChildren(n.Children, childIndent)...)
		}
	}
	return lines
}

func treeConnectors(index, total int, indent string) (string, string) {
	if index == total-1 {
		return "\u2514\u2500\u2500 ", indent + "    "
	}
	return "\u251c\u2500\u2500 ", indent + "\u2502   "
}

func reviewNodeStyle(n *TreeNode) (string, lipgloss.Style) {
	if n.IsReviewed {
		return "\u2713 ", lipgloss.NewStyle()
	}
	return "", lipgloss.NewStyle().Faint(true)
}

func (m PastReviewsModel) isAtCursor(id string) bool {
	sel := m.flatList[m.cursor]
	return sel != nil && sel.ID == id
}

func (m PastReviewsModel) collapseIndicator(n *TreeNode) string {
	if len(n.Children) == 0 {
		return ""
	}
	if m.collapsed[n.Item.ID] {
		return "\u25b6 "
	}
	return "\u25bc "
}

func (m PastReviewsModel) renderBorder(content string) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2)
	if m.width > 0 {
		border = border.Width(m.width - 4)
	}
	return border.Render(content)
}

func (m *PastReviewsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *PastReviewsModel) CursorDown() {
	if m.cursor < len(m.flatList)-1 {
		m.cursor++
	}
}

func (m *PastReviewsModel) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *PastReviewsModel) SelectedItem() *domain.Item {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		return m.flatList[m.cursor]
	}
	return nil
}

func (m *PastReviewsModel) ToggleCollapse() {
	sel := m.SelectedItem()
	if sel == nil {
		return
	}
	// Only toggle nodes that have children
	node := m.findNode(sel.ID, m.nodes)
	if node == nil || len(node.Children) == 0 {
		return
	}
	m.collapsed[sel.ID] = !m.collapsed[sel.ID]
	m.rebuildFlat()
}

func (m *PastReviewsModel) rebuildFlat() {
	m.flatList = flattenTreeVisible(m.nodes, m.collapsed)
	if m.cursor >= len(m.flatList) {
		m.cursor = len(m.flatList) - 1
	}
}

func (m *PastReviewsModel) findNode(id string, nodes []*TreeNode) *TreeNode {
	for _, n := range nodes {
		if n.Item.ID == id {
			return n
		}
		if found := m.findNode(id, n.Children); found != nil {
			return found
		}
	}
	return nil
}
