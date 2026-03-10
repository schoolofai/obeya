package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Dashboard-specific styles kept co-located with chart rendering code.
var (
	chartTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	barColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	avgLineColor = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	axisColor    = lipgloss.NewStyle().Faint(true)
	wipOk        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	wipWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	wipOver      = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	burnColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	idealColor   = lipgloss.NewStyle().Faint(true)
)

// BarData represents a single bar in a vertical bar chart.
type BarData struct {
	Label string
	Value int
}

// HBarData represents a single bar in a horizontal bar chart.
type HBarData struct {
	Label   string
	Value   float64
	Display string
}

// RenderBarChart renders a vertical bar chart using Unicode block characters.
func RenderBarChart(data []BarData, width, height int) string {
	if len(data) == 0 {
		return centerText("No data", width)
	}

	if allBarValuesZero(data) {
		return centerText("No completed items in last 14 days", width)
	}

	maxVal := maxBarValue(data)
	barWidth := 3
	gap := 1

	rows := buildBarRows(data, maxVal, height, barWidth, gap)
	labelRow := buildBarLabels(data, barWidth, gap)

	var sb strings.Builder
	for _, row := range rows {
		sb.WriteString(row)
		sb.WriteString("\n")
	}
	sb.WriteString(labelRow)
	return sb.String()
}

// RenderHorizontalBars renders a horizontal bar chart.
func RenderHorizontalBars(data []HBarData, width int) string {
	if len(data) == 0 {
		return centerText("No data", width)
	}

	maxLabel := longestLabel(data)
	barSpace := calcBarSpace(width, maxLabel)
	maxVal := maxHBarValue(data)

	var sb strings.Builder
	for i, d := range data {
		label := padRight(d.Label, maxLabel)
		barLen := scaledBarLen(d.Value, maxVal, barSpace)
		bar := barColor.Render(strings.Repeat("█", barLen))
		line := fmt.Sprintf("%s %s %s", axisColor.Render(label), bar, d.Display)
		sb.WriteString(line)
		if i < len(data)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// RenderBurndown renders a vertical burndown chart with an ideal line overlay.
func RenderBurndown(remaining []int, ideal []float64, height, width int) string {
	if len(remaining) == 0 {
		return centerText("No data", width)
	}

	if allRemainingZero(remaining) {
		return centerText("All done!", width)
	}

	maxVal := maxRemainingValue(remaining)
	barWidth := 2
	gap := 1

	rows := buildBurndownRows(remaining, ideal, maxVal, height, barWidth, gap)

	var sb strings.Builder
	for _, row := range rows {
		sb.WriteString(row)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// --- helpers -----------------------------------------------------------------

func centerText(text string, width int) string {
	if width <= len(text) {
		return text
	}
	pad := (width - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}

func allBarValuesZero(data []BarData) bool {
	for _, d := range data {
		if d.Value != 0 {
			return false
		}
	}
	return true
}

func maxBarValue(data []BarData) int {
	m := 0
	for _, d := range data {
		if d.Value > m {
			m = d.Value
		}
	}
	return m
}

func buildBarRows(data []BarData, maxVal, height, barWidth, gap int) []string {
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		rowFromBottom := height - 1 - row
		var line strings.Builder
		for i, d := range data {
			cell := barCell(d.Value, maxVal, height, rowFromBottom, barWidth)
			line.WriteString(cell)
			if i < len(data)-1 {
				line.WriteString(strings.Repeat(" ", gap))
			}
		}
		rows[row] = line.String()
	}
	return rows
}

func barCell(value, maxVal, height, rowFromBottom, barWidth int) string {
	scaledHeight := (float64(value) / float64(maxVal)) * float64(height)
	fullRows := int(scaledHeight)
	fraction := scaledHeight - float64(fullRows)

	if rowFromBottom < fullRows {
		return barColor.Render(strings.Repeat("█", barWidth))
	}
	if rowFromBottom == fullRows && fraction > 0 {
		idx := int(fraction * float64(len(blockChars)))
		if idx >= len(blockChars) {
			idx = len(blockChars) - 1
		}
		ch := string(blockChars[idx])
		return barColor.Render(strings.Repeat(ch, barWidth))
	}
	return strings.Repeat(" ", barWidth)
}

func buildBarLabels(data []BarData, barWidth, gap int) string {
	var sb strings.Builder
	for i, d := range data {
		label := d.Label
		if len(label) > barWidth {
			label = label[:barWidth]
		}
		label = padRight(label, barWidth)
		sb.WriteString(axisColor.Render(label))
		if i < len(data)-1 {
			sb.WriteString(strings.Repeat(" ", gap))
		}
	}
	return sb.String()
}

func longestLabel(data []HBarData) int {
	m := 0
	for _, d := range data {
		if len(d.Label) > m {
			m = len(d.Label)
		}
	}
	return m
}

func calcBarSpace(width, maxLabel int) int {
	space := width - maxLabel - 2 - 6
	if space < 5 {
		space = 5
	}
	return space
}

func maxHBarValue(data []HBarData) float64 {
	m := 0.0
	for _, d := range data {
		if d.Value > m {
			m = d.Value
		}
	}
	return m
}

func scaledBarLen(value, maxVal float64, barSpace int) int {
	if maxVal == 0 || value == 0 {
		return 0
	}
	l := int(math.Round((value / maxVal) * float64(barSpace)))
	if l < 1 {
		l = 1
	}
	return l
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func allRemainingZero(remaining []int) bool {
	for _, v := range remaining {
		if v != 0 {
			return false
		}
	}
	return true
}

func maxRemainingValue(remaining []int) int {
	m := 0
	for _, v := range remaining {
		if v > m {
			m = v
		}
	}
	return m
}

func buildBurndownRows(remaining []int, ideal []float64, maxVal, height, barWidth, gap int) []string {
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		rowFromBottom := height - 1 - row
		var line strings.Builder
		for i, v := range remaining {
			cell := burndownCell(v, ideal, i, maxVal, height, rowFromBottom, barWidth)
			line.WriteString(cell)
			if i < len(remaining)-1 {
				line.WriteString(strings.Repeat(" ", gap))
			}
		}
		rows[row] = line.String()
	}
	return rows
}

func burndownCell(value int, ideal []float64, idx, maxVal, height, rowFromBottom, barWidth int) string {
	scaledHeight := (float64(value) / float64(maxVal)) * float64(height)
	fullRows := int(scaledHeight)
	fraction := scaledHeight - float64(fullRows)

	isBar := false
	barStr := ""

	if rowFromBottom < fullRows {
		isBar = true
		barStr = strings.Repeat("█", barWidth)
	} else if rowFromBottom == fullRows && fraction > 0 {
		isBar = true
		ci := int(fraction * float64(len(blockChars)))
		if ci >= len(blockChars) {
			ci = len(blockChars) - 1
		}
		barStr = strings.Repeat(string(blockChars[ci]), barWidth)
	}

	if isBar {
		return burnColor.Render(barStr)
	}

	if isIdealRow(ideal, idx, maxVal, height, rowFromBottom) {
		return idealColor.Render(strings.Repeat("·", barWidth))
	}

	return strings.Repeat(" ", barWidth)
}

func isIdealRow(ideal []float64, idx, maxVal, height, rowFromBottom int) bool {
	if idx >= len(ideal) || maxVal == 0 {
		return false
	}
	idealScaled := (ideal[idx] / float64(maxVal)) * float64(height)
	idealRow := int(math.Round(idealScaled))
	return rowFromBottom == idealRow
}
