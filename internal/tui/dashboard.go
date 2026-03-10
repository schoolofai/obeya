package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/metrics"
)

type dashPanel int

const (
	panelWIP dashPanel = iota
	panelVelocity
	panelCycleTime
	panelBurndown
	panelCount // sentinel
)

// DashboardModel holds data for the metrics dashboard view.
type DashboardModel struct {
	board         *domain.Board
	width, height int
	activePanel   dashPanel
	wip           []metrics.ColumnWIP
	velocity      []metrics.DayCount
	rollingAvg    []float64
	metricsR      metrics.Result
	burndown      []metrics.BurndownPoint
	epicTitle     string
	epicTotal     int
}

func newDashboardModel(board *domain.Board, w, h int) DashboardModel {
	items := metrics.BoardItems(board)
	now := time.Now()

	dm := DashboardModel{
		board:      board,
		width:      w,
		height:     h,
		wip:        metrics.WIPStatus(board),
		velocity:   metrics.DailyVelocity(items, 14, now),
		metricsR:   metrics.Compute(items, now),
	}
	dm.rollingAvg = metrics.RollingAverage(dm.velocity, 3)
	dm.selectFirstEpic(board)
	return dm
}

// SetSize updates the dashboard dimensions.
func (d *DashboardModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// NextPanel cycles the active panel forward.
func (d *DashboardModel) NextPanel() {
	d.activePanel = (d.activePanel + 1) % panelCount
}

// SelectEpic resolves an epic by display number and computes its burndown.
func (d *DashboardModel) SelectEpic(board *domain.Board, ref string) {
	id, err := board.ResolveID(ref)
	if err != nil {
		return
	}
	item, ok := board.Items[id]
	if !ok || item.Type != domain.ItemTypeEpic {
		return
	}
	children := childrenOf(board, id)
	d.epicTitle = item.Title
	d.epicTotal = len(children)
	d.burndown = metrics.EpicBurndown(item, children, time.Now())
}

func (d *DashboardModel) selectFirstEpic(board *domain.Board) {
	for _, item := range board.Items {
		if item.Type == domain.ItemTypeEpic && item.Status != "done" {
			children := childrenOf(board, item.ID)
			d.epicTitle = item.Title
			d.epicTotal = len(children)
			d.burndown = metrics.EpicBurndown(item, children, time.Now())
			return
		}
	}
}

func childrenOf(board *domain.Board, parentID string) []*domain.Item {
	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children
}

// View renders the full dashboard.
func (d DashboardModel) View() string {
	if d.width < 80 || d.height < 24 {
		return "Terminal too small (need 80x24)"
	}

	var sections []string

	// Title
	title := chartTitle.Render("Dashboard")
	titleLine := centerText(title, d.width)
	sections = append(sections, titleLine)

	// WIP bar (full width)
	wipPanel := d.renderPanel("WIP", d.renderWIPBar(), panelWIP)
	sections = append(sections, wipPanel)

	// Velocity chart (full width)
	velContent := d.renderVelocity()
	velPanel := d.renderPanel("Velocity (14d)", velContent, panelVelocity)
	sections = append(sections, velPanel)

	// Bottom panels: cycle time and burndown
	cycleContent := d.renderCycleTime(d.bottomPanelWidth())
	cyclePanel := d.renderPanel("Cycle Time", cycleContent, panelCycleTime)

	burnContent := d.renderBurndownPanel(d.bottomPanelWidth())
	burnPanel := d.renderPanel("Burndown", burnContent, panelBurndown)

	if d.width < 100 {
		sections = append(sections, cyclePanel)
		sections = append(sections, burnPanel)
	} else {
		bottom := lipgloss.JoinHorizontal(lipgloss.Top, cyclePanel, "  ", burnPanel)
		sections = append(sections, bottom)
	}

	// Help bar
	help := axisColor.Render("D: board · Esc: back · Tab: panels · E: epic · R: refresh")
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

func (d DashboardModel) bottomPanelWidth() int {
	if d.width < 100 {
		return d.width - 4
	}
	return (d.width - 6) / 2
}

func (d DashboardModel) renderPanel(title, content string, panel dashPanel) string {
	borderColor := "8"
	if d.activePanel == panel {
		borderColor = "14"
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1)

	header := chartTitle.Render(title)
	body := header + "\n" + content
	return style.Render(body)
}

func (d DashboardModel) renderWIPBar() string {
	var parts []string
	for _, w := range d.wip {
		label := fmt.Sprintf("%s:%d", w.Name, w.Count)
		if w.Limit > 0 {
			label = fmt.Sprintf("%s:%d/%d", w.Name, w.Count, w.Limit)
		}
		switch w.Level {
		case "over":
			parts = append(parts, wipOver.Render(label))
		case "warn":
			parts = append(parts, wipWarn.Render(label))
		default:
			parts = append(parts, wipOk.Render(label))
		}
	}

	line := strings.Join(parts, "  ")

	// Append cycle/lead time
	var extras []string
	if d.metricsR.CycleTime != nil {
		extras = append(extras, fmt.Sprintf("Cycle: %s", d.metricsR.CycleTime.Display))
	}
	if d.metricsR.LeadTime != nil {
		extras = append(extras, fmt.Sprintf("Lead: %s", d.metricsR.LeadTime.Display))
	}
	if len(extras) > 0 {
		line += "  " + axisColor.Render(strings.Join(extras, " · "))
	}
	return line
}

func (d DashboardModel) renderVelocity() string {
	bars := make([]BarData, len(d.velocity))
	for i, dc := range d.velocity {
		month := dc.Date.Month().String()
		label := fmt.Sprintf("%c%d", month[0], dc.Date.Day())
		bars[i] = BarData{Label: label, Value: dc.Count}
	}
	chartWidth := d.width - 4
	chartHeight := 6
	return RenderBarChart(bars, chartWidth, chartHeight)
}

func (d DashboardModel) renderCycleTime(width int) string {
	dwell := d.metricsR.Dwell
	if len(dwell) == 0 {
		return centerText("No dwell data", width)
	}
	var bars []HBarData
	for col, dw := range dwell {
		bars = append(bars, HBarData{
			Label:   col,
			Value:   dw.Average.Hours(),
			Display: metrics.FormatDuration(dw.Average),
		})
	}
	return RenderHorizontalBars(bars, width)
}

func (d DashboardModel) renderBurndownPanel(width int) string {
	if d.epicTitle == "" {
		return centerText("No epics on board", width)
	}
	if len(d.burndown) == 0 {
		return centerText("No burndown data", width)
	}

	remaining := make([]int, len(d.burndown))
	ideal := make([]float64, len(d.burndown))
	for i, bp := range d.burndown {
		remaining[i] = bp.Remaining
		ideal[i] = bp.Ideal
	}

	header := axisColor.Render(fmt.Sprintf("%s (%d items)", d.epicTitle, d.epicTotal))
	chart := RenderBurndown(remaining, ideal, 6, width)
	return header + "\n" + chart
}
