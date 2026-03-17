package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

// TestFullBoardRender_RightBordersVisible renders a full board with epics,
// stories, and tasks across multiple columns and verifies every card has
// its right border visible.
func TestFullBoardRender_RightBordersVisible(t *testing.T) {
	b := domain.NewBoard("border-test")
	b.Users["agent-1"] = &domain.Identity{ID: "agent-1", Name: "Claude Opus", Type: domain.IdentityAgent}
	b.Users["human-1"] = &domain.Identity{ID: "human-1", Name: "Niladri", Type: domain.IdentityHuman}

	// Epic in backlog
	b.Items["epic-70"] = &domain.Item{
		ID: "epic-70", DisplayNum: 70, Type: domain.ItemTypeEpic,
		Title: "API Redesign: Migrate to Modular Versioned Architecture",
		Status: "backlog", Priority: domain.PriorityHigh,
	}
	b.DisplayMap[70] = "epic-70"

	// Story under epic in backlog
	b.Items["story-71"] = &domain.Item{
		ID: "story-71", DisplayNum: 71, Type: domain.ItemTypeStory,
		Title: "Phase 1: Route Restructuring",
		Status: "backlog", Priority: domain.PriorityHigh, ParentID: "epic-70",
	}
	b.DisplayMap[71] = "story-71"

	// Tasks under story in backlog
	for i, title := range []string{
		"Create versioned router setup",
		"Create user domain handlers",
		"Create project domain handlers",
	} {
		num := 74 + i
		id := fmt.Sprintf("task-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeTask,
			Title: title, Status: "backlog", Priority: domain.PriorityHigh,
			ParentID: "story-71",
		}
		b.DisplayMap[num] = id
	}

	// Epic in done with stories
	b.Items["epic-1"] = &domain.Item{
		ID: "epic-1", DisplayNum: 1, Type: domain.ItemTypeEpic,
		Title: "Enhanced TUI — Trello-style board with quick actions",
		Status: "done", Priority: domain.PriorityHigh,
	}
	b.DisplayMap[1] = "epic-1"

	for i, title := range []string{
		"Refactor TUI architecture into components",
		"Trello-style column board with styled cards",
		"Epic grouping with collapse/expand",
		"Detail overlay panel",
		"Picker and input modals",
	} {
		num := 2 + i
		id := fmt.Sprintf("done-story-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeStory,
			Title: title, Status: "done", Priority: domain.PriorityHigh,
			ParentID: "epic-1",
		}
		b.DisplayMap[num] = id
	}

	// Task in in-progress
	b.Items["task-168"] = &domain.Item{
		ID: "task-168", DisplayNum: 168, Type: domain.ItemTypeTask,
		Title: "Final integration verification",
		Status: "in-progress", Priority: domain.PriorityHigh,
		ParentID: "story-71",
	}
	b.DisplayMap[168] = "task-168"

	// Agent item in review queue
	conf := 85
	b.Items["review-177"] = &domain.Item{
		ID: "review-177", DisplayNum: 177, Type: domain.ItemTypeTask,
		Title: "Sample review item", Status: "done", Priority: domain.PriorityMedium,
		Assignee: "agent-1", Sponsor: "human-1",
		Confidence: &conf,
		ReviewContext: &domain.ReviewContext{Purpose: "test"},
		HumanReview:   &domain.HumanReview{Status: "pending"},
	}
	b.DisplayMap[177] = "review-177"

	for _, termWidth := range []int{120, 160, 200} {
		t.Run(fmt.Sprintf("width_%d", termWidth), func(t *testing.T) {
			app := App{
				board:        b,
				width:        termWidth,
				height:       40,
				columns:      []string{"backlog", "todo", "in-progress", "review", "done", humanReviewColName},
				collapsed:    make(map[string]bool),
				customWidths: make(map[int]int),
			}
			app.initColumnModels()

			board := app.renderBoard()
			boardLines := strings.Split(board, "\n")

			// Print full board for visual inspection
			t.Logf("\n=== FULL BOARD at %d chars wide ===", termWidth)
			for i, line := range boardLines {
				t.Logf("L%02d [w=%3d] %s", i, lipgloss.Width(line), line)
			}

			// Check that card border characters exist on card lines
			// Card top: ╭ and ╮ (or ┏ and ┓ for selected)
			// Card middle: │ on both sides (or ┃ for selected)
			// Card bottom: ╰ and ╯ (or ┗ and ┛ for selected)
			rightBorderChars := "╮│╯┓┃┛"

			// Scan each column's cards for right borders
			widths := app.columnWidths()
			for colIdx, colName := range app.columns {
				w := widths[colIdx]
				items := app.visibleItemsInColumn(colIdx)
				for _, item := range items {
					card := app.renderCardWithWidth(item, false, w)
					cardLines := strings.Split(card, "\n")
					rightBorderFound := false
					for _, line := range cardLines {
						// Strip trailing whitespace and check last visible char
						stripped := strings.TrimRight(line, " \t")
						if len(stripped) == 0 {
							continue
						}
						runes := []rune(stripped)
						lastChar := string(runes[len(runes)-1])
						if strings.ContainsAny(lastChar, rightBorderChars) {
							rightBorderFound = true
						}
					}
					if !rightBorderFound {
						t.Errorf("MISSING RIGHT BORDER: col=%s #%d %q (type=%s) at width=%d",
							colName, item.DisplayNum, item.Title, item.Type, termWidth)
						// Print the card for debugging
						for i, line := range cardLines {
							t.Logf("  card L%d [w=%d]: %s", i, lipgloss.Width(line), line)
						}
					}
				}
			}

			// Also check no board line exceeds terminal width
			for i, line := range boardLines {
				visW := lipgloss.Width(line)
				if visW > termWidth {
					t.Errorf("Board line %d exceeds terminal: vis=%d max=%d", i, visW, termWidth)
				}
			}
		})
	}
}
