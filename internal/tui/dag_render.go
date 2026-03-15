package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

// DAG-specific styles
var (
	dagNodeStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1).
			Width(dagNodeW - 2) // subtract border width

	dagNodeSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("14")).
				Bold(true).
				Padding(0, 1).
				Width(dagNodeW - 2)

	dagNodeDoneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("2")).
				Foreground(lipgloss.Color("2")).
				Padding(0, 1).
				Width(dagNodeW - 2)

	// Glowing node: bright border for in-progress items (tick=0)
	dagNodeGlowBright = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("14")).
				Bold(true).
				Padding(0, 1).
				Width(dagNodeW - 2)

	// Glowing node: dim border for in-progress items (tick=1)
	dagNodeGlowDim = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(0, 1).
			Width(dagNodeW - 2)

	dagLaneLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true).
			Faint(true)

	dagHelpStyle = lipgloss.NewStyle().Faint(true)
)

// renderDAGNode renders a single DAG node box as a multi-line string.
func renderDAGNode(n dagNode, selected bool, tick int) string {
	item := n.item
	contentW := dagNodeW - 6 // borders + padding
	if contentW < 8 {
		contentW = 8
	}

	// Title line: #N Title (truncated)
	title := fmt.Sprintf("#%d %s", item.DisplayNum, truncate(item.Title, contentW-4))

	// Type + priority line
	typLabel := typeStyle(string(item.Type)).Render(string(item.Type))
	priLabel := priorityIndicator(string(item.Priority))
	meta := fmt.Sprintf("%s %s", typLabel, priLabel)

	// Status line
	statusLine := renderStatusBar(item, contentW)

	content := title + "\n" + meta + "\n" + statusLine

	// Choose style based on state
	isInProgress := item.Status == "in-progress"
	isDone := item.Status == "done"

	if selected {
		return dagNodeSelectedStyle.Render(content)
	}
	if isInProgress {
		if tick%2 == 0 {
			return dagNodeGlowBright.Render(content)
		}
		return dagNodeGlowDim.Render(content)
	}
	if isDone {
		return dagNodeDoneStyle.Render(content)
	}
	return dagNodeStyle.Render(content)
}

// renderStatusBar renders a progress/status indicator for the node.
func renderStatusBar(item *domain.Item, width int) string {
	if width < 4 {
		width = 4
	}

	switch item.Status {
	case "done":
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(
			strings.Repeat("█", width) + " ✓",
		)
		return bar
	case "in-progress":
		// Show a partial progress bar with fire indicator
		filled := width * 2 / 3
		if filled < 1 {
			filled = 1
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(
			strings.Repeat("█", filled),
		) + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
			strings.Repeat("░", width-filled),
		) + " 🔥"
		return bar
	case "review":
		filled := width * 4 / 5
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(
			strings.Repeat("█", filled),
		) + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
			strings.Repeat("░", width-filled),
		)
		return bar
	case "todo":
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
			strings.Repeat("░", width),
		)
		return bar
	default: // backlog
		return lipgloss.NewStyle().Faint(true).Render(item.Status)
	}
}

// renderDAGCanvas renders the full DAG as a string, composing nodes and edges.
func renderDAGCanvas(g dagGraph, selectedNode int, tick int) string {
	if len(g.nodes) == 0 {
		return centerText("No items on board", 40)
	}

	// Render each node into its box string
	nodeBoxes := make([]string, len(g.nodes))
	for i, n := range g.nodes {
		nodeBoxes[i] = renderDAGNode(n, i == selectedNode, tick)
	}

	// Create canvas lines — each lane rendered separately
	var allLines []string

	for _, lane := range g.lanes {
		// Lane header
		label := dagLaneLabel.Render("── " + lane.label + " ")
		labelLine := label + dagLaneLabel.Render(strings.Repeat("─", max(0, g.width-lipgloss.Width(label))))
		allLines = append(allLines, labelLine)

		// Calculate lane canvas dimensions
		laneHeight := lane.height * (dagNodeH + dagGapY)
		if laneHeight < dagNodeH+2 {
			laneHeight = dagNodeH + 2
		}

		// Initialize a 2D char canvas for this lane
		canvas := make([][]rune, laneHeight)
		for r := range canvas {
			canvas[r] = make([]rune, g.width+dagNodeW)
			for c := range canvas[r] {
				canvas[r][c] = ' '
			}
		}

		// Draw edges first (so nodes overlay them)
		for _, e := range g.edges {
			fromInLane := false
			toInLane := false
			for _, ni := range lane.nodeIdxs {
				if ni == e.fromIdx {
					fromInLane = true
				}
				if ni == e.toIdx {
					toInLane = true
				}
			}
			if !fromInLane && !toInLane {
				continue
			}

			from := g.nodes[e.fromIdx]
			to := g.nodes[e.toIdx]

			// Convert to lane-relative y
			fromY := from.y - lane.yStart + dagNodeH/2
			toY := to.y - lane.yStart + dagNodeH/2
			fromX := from.x + dagNodeW
			toX := to.x

			if fromX < 0 || toX < 0 {
				continue
			}

			drawEdge(canvas, fromX, fromY, toX, toY, e.edgeKind)
		}

		// Compose using lipgloss positioning instead of raw canvas
		// Build edge layer as string
		edgeLines := make([]string, len(canvas))
		for r, row := range canvas {
			edgeLines[r] = string(row)
		}
		edgeLayer := strings.Join(edgeLines, "\n")

		// Place each node box on the edge layer using lipgloss
		result := edgeLayer
		for _, ni := range lane.nodeIdxs {
			n := g.nodes[ni]
			box := nodeBoxes[ni]
			ny := n.y - lane.yStart
			nx := n.x
			result = placeBox(result, box, nx, ny)
		}

		allLines = append(allLines, result)
	}

	return strings.Join(allLines, "\n")
}

// drawEdge draws an edge on the rune canvas using box-drawing characters.
func drawEdge(canvas [][]rune, x1, y1, x2, y2 int, kind string) {
	if len(canvas) == 0 {
		return
	}
	maxY := len(canvas) - 1
	maxX := len(canvas[0]) - 1

	clampX := func(x int) int {
		if x < 0 {
			return 0
		}
		if x > maxX {
			return maxX
		}
		return x
	}
	clampY := func(y int) int {
		if y < 0 {
			return 0
		}
		if y > maxY {
			return maxY
		}
		return y
	}

	hChar := '─'
	vChar := '│'
	arrowChar := '▶'
	if kind == "blocker" {
		hChar = '╌'
		vChar = '╎'
	}

	if y1 == y2 {
		// Straight horizontal
		y := clampY(y1)
		startX := clampX(x1)
		endX := clampX(x2 - 1)
		for x := startX; x <= endX; x++ {
			canvas[y][x] = hChar
		}
		if x2 <= maxX && x2 >= 0 {
			canvas[y][clampX(x2)] = arrowChar
		}
		return
	}

	// L-shaped: horizontal then vertical then horizontal
	midX := clampX(x1 + (x2-x1)/2)
	cy1 := clampY(y1)
	cy2 := clampY(y2)

	// Horizontal from start to midX
	for x := clampX(x1); x <= midX; x++ {
		canvas[cy1][x] = hChar
	}

	// Vertical from y1 to y2
	startY, endY := cy1, cy2
	if startY > endY {
		startY, endY = endY, startY
	}
	for y := startY; y <= endY; y++ {
		canvas[y][midX] = vChar
	}

	// Corners
	if y2 > y1 {
		canvas[cy1][midX] = '┐'
		canvas[cy2][midX] = '└'
	} else {
		canvas[cy1][midX] = '┘'
		canvas[cy2][midX] = '┌'
	}

	// Horizontal from midX to end
	for x := midX + 1; x < clampX(x2); x++ {
		canvas[cy2][x] = hChar
	}
	if x2 >= 0 && x2 <= maxX {
		canvas[cy2][clampX(x2)] = arrowChar
	}
}

// placeBox overlays a multi-line ANSI box string onto a base string at position (x, y).
func placeBox(base, box string, x, y int) string {
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")

	for i, bLine := range boxLines {
		row := y + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLine := baseLines[row]
		baseRunes := []rune(baseLine)

		// Pad base line if needed
		neededWidth := x + lipgloss.Width(bLine)
		for len(baseRunes) < neededWidth {
			baseRunes = append(baseRunes, ' ')
		}

		// Build new line: prefix + box line + suffix
		prefix := string(baseRunes[:x])
		suffix := ""
		boxW := lipgloss.Width(bLine)
		if x+boxW < len(baseRunes) {
			suffix = string(baseRunes[x+boxW:])
		}
		baseLines[row] = prefix + bLine + suffix
	}

	return strings.Join(baseLines, "\n")
}

// renderDAGHelp returns the help bar for DAG view.
func renderDAGHelp() string {
	return dagHelpStyle.Render(
		"  h/l:scroll  j/k:nodes  Enter:detail  G:board  D:dashboard  0:focus in-progress  q:quit",
	)
}
