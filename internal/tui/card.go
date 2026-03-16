package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
)

func (a App) renderCard(item *domain.Item, selected bool) string {
	w := a.columnWidth()
	barColor, hasBar := leftBarStyle(item.Type)
	contentW := w - 4 // border(2) + padding(2)
	if hasBar {
		contentW-- // account for ┃ character
	}
	if contentW < 10 {
		contentW = 10
	}

	lines := a.buildCardLines(item, selected, contentW)

	for i, line := range lines {
		lines[i] = padToWidth(line, contentW)
	}

	content := strings.Join(lines, "\n")

	// Prepend left bar for epics/stories
	if hasBar {
		content = prependLeftBar(content, barColor)
	}

	return a.applyCardStyle(item, selected, content)
}

func (a App) buildCardLines(item *domain.Item, selected bool, contentW int) []string {
	var lines []string

	// Breadcrumb (above title, faint)
	bc := breadcrumbPath(a.board, item, contentW)
	if bc != "" {
		lines = append(lines, breadcrumbStyle.Render(bc))
	}

	// Title with optional collapse indicator and child count badge
	lines = append(lines, a.buildHierarchyTitleLines(item, contentW)...)

	// Type + Priority + optional progress
	lines = append(lines, a.buildTypePriorityLine(item)...)

	// Meta line (assignee, blocked, sponsor, downstream)
	lines = a.appendMetaLine(lines, item)

	// Accordions
	lines = a.appendDescAccordion(lines, item, selected, contentW)
	lines = a.appendReviewAccordion(lines, item, selected, contentW)

	return lines
}

func (a App) buildHierarchyTitleLines(item *domain.Item, contentW int) []string {
	prefix := ""

	// Collapse indicator for items with children
	if hasChildItems(a.board, item.ID) {
		if a.collapsed[item.ID] {
			prefix += "\u25b6 "
		} else {
			prefix += "\u25bc "
		}
	}

	// Agent badge
	if u, ok := a.board.Users[item.Assignee]; ok && u.Type == domain.IdentityAgent {
		prefix += agentBadgeStyle.Render("AGENT") + " "
	}

	prefix += fmt.Sprintf("#%d ", item.DisplayNum)

	// Child count badge (rendered after title)
	badge := ""
	if count := childCount(a.board, item.ID); count > 0 {
		badgeText := fmt.Sprintf("%d items", count)
		switch item.Type {
		case domain.ItemTypeEpic:
			badge = " " + epicBadgeStyle.Render(badgeText)
		case domain.ItemTypeStory:
			badge = " " + storyBadgeStyle.Render(badgeText)
		default:
			badge = " " + progressStyle.Render(badgeText)
		}
	}

	badgeWidth := lipgloss.Width(badge)
	titleMax := contentW - utf8.RuneCountInString(prefix) - badgeWidth
	if titleMax < 4 {
		titleMax = 4
	}
	titleLines := wrapText(item.Title, titleMax)
	lines := []string{prefix + titleLines[0] + badge}
	indent := strings.Repeat(" ", utf8.RuneCountInString(prefix))
	for _, tl := range titleLines[1:] {
		lines = append(lines, indent+tl)
	}
	return lines
}

func (a App) buildTypePriorityLine(item *domain.Item) []string {
	typLabel := typeStyle(string(item.Type)).Render(string(item.Type))
	line := fmt.Sprintf("%s %s", typLabel, priorityIndicator(string(item.Priority)))

	confStr := confidenceIndicator(item.Confidence)
	if confStr != "" {
		line += "  " + confStr
	}

	// Progress for parents
	if total := childCount(a.board, item.ID); total > 0 {
		done := doneCount(a.board, item.ID)
		line += "  " + progressStyle.Render(fmt.Sprintf("%d/%d done", done, total))
	}

	return []string{line}
}

func prependLeftBar(content string, color lipgloss.Color) string {
	bar := lipgloss.NewStyle().Foreground(color).Render("\u2503")
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = bar + line
	}
	return strings.Join(lines, "\n")
}

func (a App) appendMetaLine(lines []string, item *domain.Item) []string {
	var metaParts []string
	if item.Assignee != "" {
		name := resolveUserName(a.board, item.Assignee)
		metaParts = append(metaParts, assigneeStyle.Render("@"+name))
	} else {
		metaParts = append(metaParts, unassignedStyle.Render("@unassigned"))
	}
	if len(item.BlockedBy) > 0 {
		metaParts = append(metaParts, blockedStyle.Render("[!]"))
	}
	if item.Sponsor != "" {
		sponsorName := resolveUserName(a.board, item.Sponsor)
		metaParts = append(metaParts, sponsorStyle.Render("sponsor: @"+sponsorName))
	}
	lines = append(lines, strings.Join(metaParts, " "))

	// Downstream impact
	downstream := engine.ResolveDownstream(item.ID, a.board)
	if len(downstream) > 0 {
		lines = append(lines, downstreamStyle.Render(
			fmt.Sprintf("\u26a1 unblocks %d tasks", len(downstream)),
		))
	}

	return lines
}

func (a App) appendDescAccordion(lines []string, item *domain.Item, selected bool, contentW int) []string {
	if !selected || item.Description == "" {
		return lines
	}
	if a.descExpanded == item.ID {
		lines = append(lines, descIndicatorStyle.Render("\u25bc description"))
		sep := strings.Repeat("\u2500", contentW)
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(sep))
		lines = append(lines, a.renderDescription(item.Description, contentW, a.descScrollY, 5)...)
	} else {
		lines = append(lines, descIndicatorStyle.Render("\u25b6 description"))
	}
	return lines
}

func (a App) appendReviewAccordion(lines []string, item *domain.Item, selected bool, contentW int) []string {
	if !selected || item.ReviewContext == nil {
		return lines
	}
	if a.reviewExpanded == item.ID {
		lines = append(lines, reviewContextIndicatorStyle.Render("\u25bc review context"))
		sep := strings.Repeat("\u2504", contentW)
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(sep))
		lines = append(lines, a.renderReviewContext(item.ReviewContext, contentW, a.reviewScrollY, 5)...)
	} else {
		lines = append(lines, reviewContextIndicatorStyle.Render("\u25b6 review context"))
	}
	return lines
}

func (a App) applyCardStyle(item *domain.Item, selected bool, content string) string {
	if selected {
		return selectedCardStyle.Render(content)
	}
	if item.HumanReview != nil && item.HumanReview.Status == "reviewed" {
		return reviewedCardStyle.Render(content)
	}
	return cardStyle.Render(content)
}

func (a App) isItemAtCursor(item *domain.Item) bool {
	sel := a.selectedItem()
	return sel != nil && sel.ID == item.ID
}

// padToWidth pads a string (which may contain ANSI codes) with spaces
// so its visual width equals targetW. Uses lipgloss.Width for ANSI-aware measurement.
func padToWidth(s string, targetW int) string {
	visW := lipgloss.Width(s)
	if visW >= targetW {
		return s
	}
	return s + strings.Repeat(" ", targetW-visW)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// wrapText word-wraps s into lines of at most maxWidth runes.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	if utf8.RuneCountInString(s) <= maxWidth {
		return []string{s}
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	curLen := 0
	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if cur == "" {
			runes := []rune(word)
			for len(runes) > maxWidth {
				lines = append(lines, string(runes[:maxWidth]))
				runes = runes[maxWidth:]
			}
			cur = string(runes)
			curLen = len(runes)
			continue
		}
		if curLen+1+wordLen <= maxWidth {
			cur += " " + word
			curLen += 1 + wordLen
		} else {
			lines = append(lines, cur)
			runes := []rune(word)
			for len(runes) > maxWidth {
				lines = append(lines, string(runes[:maxWidth]))
				runes = runes[maxWidth:]
			}
			cur = string(runes)
			curLen = len(runes)
		}
	}
	lines = append(lines, cur)
	return lines
}

// renderDescription word-wraps and renders a description within a scrollable
// viewport of maxLines content lines.
func (a App) renderDescription(desc string, maxWidth int, scrollY int, maxLines int) []string {
	if desc == "" {
		return nil
	}

	paragraphs := strings.Split(desc, "\n")
	var allLines []string
	for _, p := range paragraphs {
		if p == "" {
			allLines = append(allLines, "")
			continue
		}
		wrapped := wrapText(p, maxWidth)
		allLines = append(allLines, wrapped...)
	}

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
		hint := fmt.Sprintf("%*s", maxWidth, "\u25b4 J/K")
		styled = append(styled, descScrollHint.Render(hint))
	}

	for _, l := range visible {
		styled = append(styled, descStyle.Render(l))
	}

	if end < totalLines {
		hint := fmt.Sprintf("%*s", maxWidth, "\u25be J/K")
		styled = append(styled, descScrollHint.Render(hint))
	}

	return styled
}
