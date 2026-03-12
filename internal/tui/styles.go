package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Card styles
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	selectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("14")). // Bright cyan
				Bold(true).
				Padding(0, 1)

	// Type colors
	epicStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	storyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	taskStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	// Priority indicators
	priCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("\u25cf\u25cf\u25cf")
	priHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("\u25cf\u25cf")
	priMedium   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("\u25cf\u25cf")
	priLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("\u25cf")

	// Status
	blockedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	assigneeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)

	// Column headers
	activeColHeader   = lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveColHeader = lipgloss.NewStyle().Faint(true)

	// Column container
	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 0).
			MarginRight(1)

	activeColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("6")).
				Padding(0, 0).
				MarginRight(1)

	// Overlay
	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(1, 2).
			Width(50)

	// Help bar
	helpStyle = lipgloss.NewStyle().Faint(true)

	// Epic group header
	epicGroupStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true)

	selectedEpicGroupStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Bold(true).
				Underline(true)

	// Cross-column badge appended to epic headers when epic lives elsewhere.
	crossColBadge = lipgloss.NewStyle().Faint(true).Render(" ⇠")

	// Description accordion
	descIndicatorStyle = lipgloss.NewStyle().Faint(true)
	descStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	descScrollHint     = lipgloss.NewStyle().Faint(true)
)

func priorityIndicator(pri string) string {
	switch pri {
	case "critical":
		return priCritical
	case "high":
		return priHigh
	case "medium":
		return priMedium
	case "low":
		return priLow
	default:
		return priMedium
	}
}

func typeStyle(itemType string) lipgloss.Style {
	switch itemType {
	case "epic":
		return epicStyle
	case "story":
		return storyStyle
	default:
		return taskStyle
	}
}
