package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// ColumnModel wraps a viewport.Model to render a single Kanban column
// with a header, scrollable content area, and optional scrollbar overlay.
type ColumnModel struct {
	Name     string
	viewport viewport.Model
	active   bool
	cursor   int
}

// NewColumnModel creates a ColumnModel with the given content width and
// visible height. The viewport's own key bindings are disabled (empty KeyMap)
// so the App handles all keyboard input.
func NewColumnModel(name string, contentWidth, viewHeight int) ColumnModel {
	vp := viewport.New(contentWidth, viewHeight)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3
	vp.KeyMap = viewport.KeyMap{} // App handles keys
	return ColumnModel{
		Name:     name,
		viewport: vp,
	}
}

// SetSize updates the viewport dimensions.
func (c *ColumnModel) SetSize(contentWidth, viewHeight int) {
	c.viewport.Width = contentWidth
	c.viewport.Height = viewHeight
}

// SetContent sets the viewport's text content.
func (c *ColumnModel) SetContent(content string) {
	c.viewport.SetContent(content)
}

// ScrollToLine sets the viewport's vertical offset to the given line.
func (c *ColumnModel) ScrollToLine(line int) {
	c.viewport.SetYOffset(line)
}

// View renders the column: header (name + item count, padded to width),
// viewport content with scrollbar overlay, and column border style.
func (c *ColumnModel) View(itemCount int) string {
	w := c.viewport.Width

	header := renderColumnHeader(c.Name, itemCount, w, c.active)
	body := overlayScrollbar(c.viewport.View(), c.viewport)

	inner := header + "\n" + body

	style := columnStyle
	if c.active {
		style = activeColumnStyle
	}
	return style.Render(inner)
}

// renderColumnHeader builds the header line: column name + count, padded.
func renderColumnHeader(name string, count, width int, active bool) string {
	label := fmt.Sprintf("%s (%d)", strings.ToUpper(name), count)
	if active {
		return padToWidth(activeColHeader.Render(label), width)
	}
	return padToWidth(inactiveColHeader.Render(label), width)
}

// overlayScrollbar replaces the last character of each visible line with a
// scrollbar indicator when the viewport content overflows. The thumb is
// rendered as ┃ in cyan (color "14") and the track as │ in dim gray ("238").
// Returns vpView unchanged when content fits within the viewport.
func overlayScrollbar(vpView string, vp viewport.Model) string {
	totalLines := vp.TotalLineCount()
	visibleHeight := vp.Height
	if totalLines <= visibleHeight {
		return vpView
	}

	lines := strings.Split(vpView, "\n")

	thumbStart, thumbEnd := calcThumbRange(
		vp.YOffset, totalLines, visibleHeight, len(lines),
	)

	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	trackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	for i, line := range lines {
		var indicator string
		if i >= thumbStart && i < thumbEnd {
			indicator = thumbStyle.Render("┃")
		} else {
			indicator = trackStyle.Render("│")
		}
		lines[i] = replaceLastChar(line, indicator)
	}
	return strings.Join(lines, "\n")
}

// calcThumbRange returns the start (inclusive) and end (exclusive) line
// indices for the scrollbar thumb within the visible area.
func calcThumbRange(yOffset, totalLines, visibleHeight, renderedLines int) (int, int) {
	if renderedLines == 0 || totalLines == 0 {
		return 0, 0
	}

	thumbSize := max(1, visibleHeight*visibleHeight/totalLines)
	if thumbSize > renderedLines {
		thumbSize = renderedLines
	}

	scrollable := totalLines - visibleHeight
	if scrollable <= 0 {
		return 0, thumbSize
	}

	thumbStart := yOffset * (renderedLines - thumbSize) / scrollable
	if thumbStart+thumbSize > renderedLines {
		thumbStart = renderedLines - thumbSize
	}
	if thumbStart < 0 {
		thumbStart = 0
	}

	return thumbStart, thumbStart + thumbSize
}

// replaceLastChar replaces the last visible character of a line with the
// given replacement string. Handles ANSI-styled lines by measuring visual
// width and inserting the replacement at the correct position.
func replaceLastChar(line, replacement string) string {
	visW := lipgloss.Width(line)
	if visW == 0 {
		return replacement
	}

	// Build a line that is (visW-1) wide, then append the indicator.
	// We truncate the original line to (visW-1) visual columns and
	// pad to ensure consistent width.
	truncated := truncateVisual(line, visW-1)
	return truncated + replacement
}

// truncateVisual truncates a string (potentially containing ANSI codes)
// to at most maxVisualWidth visible columns, preserving ANSI sequences.
func truncateVisual(s string, maxVisualWidth int) string {
	if maxVisualWidth <= 0 {
		return ""
	}
	// Walk through runes, tracking visual width, preserving ANSI escapes.
	var b strings.Builder
	visW := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			b.WriteRune(r)
			continue
		}
		if inEsc {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if visW >= maxVisualWidth {
			break
		}
		b.WriteRune(r)
		visW++
	}
	// Pad if needed to reach maxVisualWidth
	for visW < maxVisualWidth {
		b.WriteByte(' ')
		visW++
	}
	return b.String()
}
