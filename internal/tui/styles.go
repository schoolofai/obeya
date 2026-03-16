package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

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
	blockedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	assigneeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)
	unassignedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)

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

	// Description accordion
	descIndicatorStyle = lipgloss.NewStyle().Faint(true)
	descStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	descScrollHint     = lipgloss.NewStyle().Faint(true)

	// Agent badge
	agentBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Bold(true)

	// Confidence indicators
	confLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	confMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	confHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	// Sponsor
	sponsorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)

	// Review queue column (amber instead of cyan)
	reviewQueueColumnStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 0).
		MarginRight(1)

	activeReviewQueueColumnStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(0, 0).
		MarginRight(1)

	// Reviewed card
	reviewedCardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Faint(true).
		Padding(0, 1)

	// Review context accordion
	reviewContextIndicatorStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("3"))

	// Downstream impact
	downstreamStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	// Hierarchy: left bar colors
	epicBarColor  = lipgloss.Color("5")  // magenta
	storyBarColor = lipgloss.Color("4")  // blue

	// Breadcrumb
	breadcrumbStyle = lipgloss.NewStyle().Faint(true)

	// Child count badge
	epicBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Background(lipgloss.Color("53")) // dark magenta bg

	storyBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Background(lipgloss.Color("17")) // dark blue bg

	// Progress fraction
	progressStyle = lipgloss.NewStyle().Faint(true)
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

func confidenceIndicator(confidence *int) string {
	if confidence == nil {
		return ""
	}
	c := *confidence
	switch {
	case c <= 50:
		return confLow.Render(fmt.Sprintf("%d%% \u26a0 LOW", c))
	case c <= 75:
		return confMedium.Render(fmt.Sprintf("%d%%", c))
	default:
		return confHigh.Render(fmt.Sprintf("%d%% \u2713", c))
	}
}

func leftBarStyle(itemType domain.ItemType) (lipgloss.Color, bool) {
	switch itemType {
	case domain.ItemTypeEpic:
		return epicBarColor, true
	case domain.ItemTypeStory:
		return storyBarColor, true
	default:
		return "", false
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
