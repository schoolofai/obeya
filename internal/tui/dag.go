package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

// dagTickMsg drives the glow animation for in-progress nodes.
type dagTickMsg time.Time

// DAGModel is the Bubble Tea sub-model for DAG visualization.
type DAGModel struct {
	board      *domain.Board
	graph      dagGraph
	viewport   viewport.Model
	cursorNode int  // index into graph.nodes
	tick       int  // animation cycle counter
	width      int
	height     int
	scrollX    int  // horizontal scroll offset
	ready      bool // true after first resize
}

// newDAGModel creates a new DAG model from a board.
func newDAGModel(board *domain.Board, w, h int) DAGModel {
	g := buildDAGGraph(board)

	vp := viewport.New(w, h-3) // -3 for header + help bar + padding
	vp.KeyMap = viewport.KeyMap{}
	vp.MouseWheelEnabled = true

	m := DAGModel{
		board:    board,
		graph:    g,
		viewport: vp,
		width:    w,
		height:   h,
		ready:    true,
	}

	// Auto-scroll to first in-progress node
	m.autoScrollToInProgress()
	m.updateViewport()

	return m
}

// SetSize updates the viewport dimensions.
func (m *DAGModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentH := h - 3
	if contentH < 1 {
		contentH = 1
	}
	m.viewport.Width = w
	m.viewport.Height = contentH
	m.updateViewport()
}

// SelectedItem returns the currently selected item, or nil.
func (m *DAGModel) SelectedItem() *domain.Item {
	if m.cursorNode < 0 || m.cursorNode >= len(m.graph.nodes) {
		return nil
	}
	return m.graph.nodes[m.cursorNode].item
}

// updateViewport re-renders the DAG canvas and sets viewport content.
func (m *DAGModel) updateViewport() {
	content := renderDAGCanvas(m.graph, m.cursorNode, m.tick)

	// Apply horizontal scroll by trimming/shifting lines
	if m.scrollX > 0 {
		content = horizontalScroll(content, m.scrollX, m.width)
	}

	m.viewport.SetContent(content)
}

// horizontalScroll shifts all lines left by offset characters, showing at most width chars.
func horizontalScroll(content string, offset, width int) string {
	lines := splitLines(content)
	for i, line := range lines {
		runes := []rune(line)
		if offset >= len(runes) {
			lines[i] = ""
			continue
		}
		end := offset + width
		if end > len(runes) {
			end = len(runes)
		}
		lines[i] = string(runes[offset:end])
	}
	return joinLines(lines)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, l := range lines[1:] {
		result += "\n" + l
	}
	return result
}

// autoScrollToInProgress scrolls to center the first in-progress node.
func (m *DAGModel) autoScrollToInProgress() {
	idx := m.graph.firstInProgressNode()
	if idx < 0 {
		return
	}
	n := m.graph.nodes[idx]
	m.cursorNode = idx

	// Center horizontally
	targetX := n.x - m.width/2 + dagNodeW/2
	if targetX < 0 {
		targetX = 0
	}
	m.scrollX = targetX

	// Center vertically
	targetY := n.y - m.viewport.Height/2 + dagNodeH/2
	if targetY < 0 {
		targetY = 0
	}
	m.viewport.SetYOffset(targetY)
}

// moveToNextNode moves the cursor to the next node in the graph.
func (m *DAGModel) moveToNextNode() {
	if len(m.graph.nodes) == 0 {
		return
	}
	m.cursorNode = (m.cursorNode + 1) % len(m.graph.nodes)
	m.ensureNodeVisible()
	m.updateViewport()
}

// moveToPrevNode moves the cursor to the previous node.
func (m *DAGModel) moveToPrevNode() {
	if len(m.graph.nodes) == 0 {
		return
	}
	m.cursorNode--
	if m.cursorNode < 0 {
		m.cursorNode = len(m.graph.nodes) - 1
	}
	m.ensureNodeVisible()
	m.updateViewport()
}

// ensureNodeVisible adjusts scroll to keep the selected node visible.
func (m *DAGModel) ensureNodeVisible() {
	if m.cursorNode < 0 || m.cursorNode >= len(m.graph.nodes) {
		return
	}
	n := m.graph.nodes[m.cursorNode]

	// Horizontal scroll
	if n.x < m.scrollX {
		m.scrollX = n.x - 2
	}
	if n.x+dagNodeW > m.scrollX+m.width {
		m.scrollX = n.x + dagNodeW - m.width + 2
	}
	if m.scrollX < 0 {
		m.scrollX = 0
	}

	// Vertical scroll — let viewport handle it, but nudge if needed
	if n.y < m.viewport.YOffset {
		m.viewport.SetYOffset(n.y - 1)
	}
	if n.y+dagNodeH > m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(n.y + dagNodeH - m.viewport.Height + 1)
	}
}

// scrollLeft scrolls the viewport left.
func (m *DAGModel) scrollLeft() {
	m.scrollX -= 8
	if m.scrollX < 0 {
		m.scrollX = 0
	}
	m.updateViewport()
}

// scrollRight scrolls the viewport right.
func (m *DAGModel) scrollRight() {
	maxScroll := m.graph.width - m.width + dagNodeW
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollX += 8
	if m.scrollX > maxScroll {
		m.scrollX = maxScroll
	}
	m.updateViewport()
}

// View renders the DAG view.
func (m DAGModel) View() string {
	header := fmt.Sprintf("  DAG View: %s", m.board.Name)
	if m.cursorNode >= 0 && m.cursorNode < len(m.graph.nodes) {
		n := m.graph.nodes[m.cursorNode]
		header += lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf("  │  #%d %s [%s]", n.item.DisplayNum, n.item.Title, n.item.Status),
		)
	}

	scrollInfo := ""
	if m.graph.width > m.width {
		pct := 0
		maxScroll := m.graph.width - m.width + dagNodeW
		if maxScroll > 0 {
			pct = m.scrollX * 100 / maxScroll
		}
		scrollInfo = lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf("  [scroll: %d%%]", pct),
		)
	}

	help := renderDAGHelp()
	return header + scrollInfo + "\n" + m.viewport.View() + "\n" + help
}

// dagTickCmd returns a tick command for glow animation.
func dagTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return dagTickMsg(t)
	})
}
