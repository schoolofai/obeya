package tui

import (
	"fmt"

	"github.com/niladribose/obeya/internal/domain"
)

// renderReviewContext renders a scrollable review context section within a card.
func (a App) renderReviewContext(rc *domain.ReviewContext, maxWidth int, scrollY int, maxLines int) []string {
	lines := reviewContextLines(rc, maxWidth)
	return renderScrollableContent(lines, maxWidth, scrollY, maxLines, "Ctrl+J/K")
}

func reviewContextLines(rc *domain.ReviewContext, maxWidth int) []string {
	var lines []string

	if rc.Purpose != "" {
		lines = append(lines, " Purpose: "+rc.Purpose)
	}

	lines = appendFilesChanged(lines, rc.FilesChanged)
	lines = appendTestsSummary(lines, rc.TestsWritten)
	lines = appendReproduceCommands(lines, rc.Reproduce)
	lines = appendProofItems(lines, rc.Proof)

	if rc.Reasoning != "" {
		lines = append(lines, "")
		wrapped := wrapText(rc.Reasoning, maxWidth-2)
		for i, w := range wrapped {
			prefix := " Reason: "
			if i > 0 {
				prefix = "         "
			}
			lines = append(lines, prefix+w)
		}
	}

	return lines
}

func appendFilesChanged(lines []string, files []domain.FileChange) []string {
	if len(files) == 0 {
		return lines
	}
	lines = append(lines, "")
	for i, f := range files {
		prefix := " Files:  "
		if i > 0 {
			prefix = "         "
		}
		lines = append(lines, fmt.Sprintf("%s%s  (+%d -%d)", prefix, f.Path, f.Added, f.Removed))
	}
	return lines
}

func appendTestsSummary(lines []string, tests []domain.TestResult) []string {
	if len(tests) == 0 {
		return lines
	}
	passed := 0
	for _, t := range tests {
		if t.Passed {
			passed++
		}
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf(" Tests:  %d total, %d pass", len(tests), passed))
	return lines
}

func appendReproduceCommands(lines []string, cmds []string) []string {
	if len(cmds) == 0 {
		return lines
	}
	lines = append(lines, "")
	lines = append(lines, " Reproduce:")
	for _, cmd := range cmds {
		lines = append(lines, "   $ "+cmd)
	}
	return lines
}

func appendProofItems(lines []string, proof []domain.ProofItem) []string {
	if len(proof) == 0 {
		return lines
	}
	lines = append(lines, "")
	lines = append(lines, " Proof:")
	for _, p := range proof {
		icon := proofIcon(p.Status)
		line := fmt.Sprintf("   %s %s", icon, p.Check)
		if p.Detail != "" {
			line += ": " + p.Detail
		}
		lines = append(lines, line)
	}
	return lines
}

func proofIcon(status string) string {
	switch status {
	case "fail":
		return "\u2717"
	case "warn":
		return "\u26a0"
	default:
		return "\u2713"
	}
}

// renderScrollableContent renders a set of lines with scroll indicators,
// similar to renderDescription but reusable for any scrollable content.
func renderScrollableContent(allLines []string, maxWidth int, scrollY int, maxLines int, scrollHint string) []string {
	totalLines := len(allLines)
	if totalLines == 0 {
		return nil
	}

	if totalLines <= maxLines {
		styled := make([]string, len(allLines))
		for i, l := range allLines {
			styled[i] = descStyle.Render(l)
		}
		return styled
	}

	maxScroll := totalLines - maxLines
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	end := scrollY + maxLines
	if end > totalLines {
		end = totalLines
	}
	visible := allLines[scrollY:end]

	styled := make([]string, 0, len(visible)+2)

	if scrollY > 0 {
		hint := fmt.Sprintf("%*s", maxWidth, "\u25b4 "+scrollHint)
		styled = append(styled, descScrollHint.Render(hint))
	}

	for _, l := range visible {
		styled = append(styled, descStyle.Render(l))
	}

	if end < totalLines {
		hint := fmt.Sprintf("%*s", maxWidth, "\u25be "+scrollHint)
		styled = append(styled, descScrollHint.Render(hint))
	}

	return styled
}
