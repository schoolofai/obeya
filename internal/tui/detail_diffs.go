package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

var (
	diffAddStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	diffDelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	diffHunkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)
	diffFileStyle = lipgloss.NewStyle().Bold(true)
)

func renderDiffsTab(rc *domain.ReviewContext, width int) string {
	if rc == nil {
		return "  No review context."
	}
	var sections []string
	for _, fc := range rc.FilesChanged {
		if fc.Diff == "" {
			continue
		}
		header := diffFileStyle.Render(
			fmt.Sprintf("  %s  (+%d -%d)", fc.Path, fc.Added, fc.Removed),
		)
		sep := "  " + strings.Repeat("\u2500", width-8)
		diffLines := colorizeDiff(fc.Diff)
		sections = append(sections, header+"\n"+sep+"\n"+diffLines)
	}
	if len(sections) == 0 {
		return "  No diffs available."
	}
	return strings.Join(sections, "\n\n")
}

func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	styled := make([]string, 0, len(lines))
	for _, line := range lines {
		styled = append(styled, colorizeDiffLine(line))
	}
	return strings.Join(styled, "\n")
}

func colorizeDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "@@"):
		return "  " + diffHunkStyle.Render(line)
	case strings.HasPrefix(line, "+"):
		return "  " + diffAddStyle.Render(line)
	case strings.HasPrefix(line, "-"):
		return "  " + diffDelStyle.Render(line)
	default:
		return "  " + line
	}
}

// hasAnyDiff returns true if any FileChange in the ReviewContext has diff content.
func hasAnyDiff(rc *domain.ReviewContext) bool {
	for _, f := range rc.FilesChanged {
		if f.Diff != "" {
			return true
		}
	}
	return false
}
