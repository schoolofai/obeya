package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

func TestCardWidth_NoClipping(t *testing.T) {
	b := domain.NewBoard("clip-test")
	b.Users["agent-1"] = &domain.Identity{ID: "agent-1", Name: "Claude Opus", Type: domain.IdentityAgent}
	b.Users["human-1"] = &domain.Identity{ID: "human-1", Name: "Niladri", Type: domain.IdentityHuman}

	epic := &domain.Item{
		ID: "epic-70", DisplayNum: 70, Type: domain.ItemTypeEpic,
		Title: "API Redesign: Migrate to Modular Versioned Architecture",
		Status: "backlog", Priority: domain.PriorityHigh,
	}
	b.Items[epic.ID] = epic
	b.DisplayMap[70] = epic.ID

	story := &domain.Item{
		ID: "story-71", DisplayNum: 71, Type: domain.ItemTypeStory,
		Title: "Phase 1: Route Restructuring",
		Status: "backlog", Priority: domain.PriorityHigh, ParentID: epic.ID,
	}
	b.Items[story.ID] = story
	b.DisplayMap[71] = story.ID

	taskTitles := []string{
		"Create versioned router setup",
		"Create user domain handlers",
		"Create project domain handlers",
		"Create billing domain handlers",
	}
	for i, title := range taskTitles {
		num := 74 + i
		id := fmt.Sprintf("task-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeTask,
			Title: title, Status: "backlog", Priority: domain.PriorityHigh,
			ParentID: story.ID,
		}
		b.DisplayMap[num] = id
	}

	doneTitles := []string{
		"Enhanced TUI — Trello-style board with quick actions",
		"Refactor TUI architecture into components",
		"Trello-style column board with styled cards",
		"Epic grouping with collapse/expand",
		"Detail overlay panel",
		"Picker and input modals",
	}
	for i, title := range doneTitles {
		num := 1 + i
		id := fmt.Sprintf("done-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeStory,
			Title: title, Status: "done", Priority: domain.PriorityHigh,
			ParentID: epic.ID,
		}
		b.DisplayMap[num] = id
	}

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

	for _, termWidth := range []int{120, 140, 160, 200} {
		t.Run(fmt.Sprintf("width_%d", termWidth), func(t *testing.T) {
			app := App{
				board:     b,
				width:     termWidth,
				height:    40,
				columns:   []string{"backlog", "todo", "in-progress", "review", "done", humanReviewColName},
				collapsed: make(map[string]bool),
			}
			app.initColumnModels()

			widths := app.columnWidths()
			t.Logf("Terminal=%d  Widths: %v", termWidth, widths)

			for i, colName := range app.columns {
				w := widths[i]
				items := app.visibleItemsInColumn(i)
				for _, item := range items {
					indent := 0
					if app.isFirstLevelChild(item, colName) {
						indent = 2
					}
					card := app.renderCardWithWidth(item, false, w-indent)
					if indent > 0 {
						card = indentCard(card, indent)
					}
					lines := strings.Split(card, "\n")
					for lineNum, line := range lines {
						visW := lipgloss.Width(line)
						if visW > w {
							t.Errorf("CLIP col=%s #%d line=%d vis=%d max=%d excess=%d",
								colName, item.DisplayNum, lineNum, visW, w, visW-w)
						}
					}
				}
			}
		})
	}
}

func TestBoardRender_NoClipping(t *testing.T) {
	b := domain.NewBoard("clip-test")
	b.Users["agent-1"] = &domain.Identity{ID: "agent-1", Name: "Claude Opus", Type: domain.IdentityAgent}
	b.Users["human-1"] = &domain.Identity{ID: "human-1", Name: "Niladri", Type: domain.IdentityHuman}

	epic := &domain.Item{
		ID: "epic-70", DisplayNum: 70, Type: domain.ItemTypeEpic,
		Title: "API Redesign: Migrate to Modular Versioned Architecture",
		Status: "backlog", Priority: domain.PriorityHigh,
	}
	b.Items[epic.ID] = epic
	b.DisplayMap[70] = epic.ID

	story := &domain.Item{
		ID: "story-71", DisplayNum: 71, Type: domain.ItemTypeStory,
		Title: "Phase 1: Route Restructuring",
		Status: "backlog", Priority: domain.PriorityHigh, ParentID: epic.ID,
	}
	b.Items[story.ID] = story
	b.DisplayMap[71] = story.ID

	for i, title := range []string{"Create versioned router setup", "Create user domain handlers"} {
		num := 74 + i
		id := fmt.Sprintf("task-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeTask,
			Title: title, Status: "backlog", Priority: domain.PriorityHigh,
			ParentID: story.ID,
		}
		b.DisplayMap[num] = id
	}

	for i, title := range []string{
		"Enhanced TUI — Trello-style board with quick actions",
		"Trello-style column board with styled cards",
	} {
		num := 1 + i
		id := fmt.Sprintf("done-%d", num)
		b.Items[id] = &domain.Item{
			ID: id, DisplayNum: num, Type: domain.ItemTypeStory,
			Title: title, Status: "done", Priority: domain.PriorityHigh,
			ParentID: epic.ID,
		}
		b.DisplayMap[num] = id
	}

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

	app := App{
		board:     b,
		width:     160,
		height:    40,
		columns:   []string{"backlog", "todo", "in-progress", "review", "done", humanReviewColName},
		collapsed: make(map[string]bool),
	}
	app.initColumnModels()

	rendered := app.renderBoard()
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		visW := lipgloss.Width(line)
		if visW > app.width {
			t.Errorf("Board line %d exceeds terminal width: vis=%d terminal=%d", i, visW, app.width)
		}
	}
	// Print the first 15 lines for visual inspection
	for i, line := range lines {
		if i >= 15 { break }
		t.Logf("L%02d [w=%3d] %s", i, lipgloss.Width(line), line)
	}
}

func TestColumnStyleOverhead(t *testing.T) {
	// Measure exact overhead lipgloss adds to content
	content := "hello"  // 5 chars
	rendered := columnStyle.Render(content)
	visW := lipgloss.Width(rendered)
	t.Logf("Content=%d  Rendered=%d  Overhead=%d", 5, visW, visW-5)
	t.Logf("Rendered: %q", rendered)

	// With active style
	rendered2 := activeColumnStyle.Render(content)
	visW2 := lipgloss.Width(rendered2)
	t.Logf("Active Content=%d  Rendered=%d  Overhead=%d", 5, visW2, visW2-5)

	// Review queue
	rendered3 := reviewQueueColumnStyle.Render(content)
	visW3 := lipgloss.Width(rendered3)
	t.Logf("ReviewQueue Content=%d  Rendered=%d  Overhead=%d", 5, visW3, visW3-5)
}

func TestSingleColumnWidth(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["item-1"] = &domain.Item{
		ID: "item-1", DisplayNum: 1, Type: domain.ItemTypeTask,
		Title: "Short task", Status: "backlog", Priority: domain.PriorityMedium,
	}
	b.DisplayMap[1] = "item-1"

	app := App{board: b, width: 100, height: 40, columns: []string{"backlog"}, collapsed: make(map[string]bool)}
	app.initColumnModels()
	widths := app.columnWidths()
	w := widths[0]
	t.Logf("Content width: %d", w)

	items := app.visibleItemsInColumn(0)
	var cardViews []string
	for _, item := range items {
		cardViews = append(cardViews, app.renderCardWithWidth(item, false, w))
	}
	for _, cv := range cardViews {
		for i, line := range strings.Split(cv, "\n") {
			lw := lipgloss.Width(line)
			t.Logf("Card line %d: vis=%d contentW=%d %q", i, lw, w, line)
		}
	}

	// Now render through the column model
	app.colModels[0].SetContent(strings.Join(cardViews, "\n"))
	colView := app.colModels[0].View(1)
	for i, line := range strings.Split(colView, "\n") {
		lw := lipgloss.Width(line)
		t.Logf("Col  line %d: vis=%d %q", i, lw, line)
	}
}

func TestMeasureColumnWidths(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["item-1"] = &domain.Item{ID: "item-1", DisplayNum: 1, Type: domain.ItemTypeTask, Title: "Test", Status: "backlog", Priority: domain.PriorityMedium}
	b.Items["item-2"] = &domain.Item{ID: "item-2", DisplayNum: 2, Type: domain.ItemTypeTask, Title: "Test2", Status: "done", Priority: domain.PriorityMedium}
	b.DisplayMap[1] = "item-1"
	b.DisplayMap[2] = "item-2"

	app := App{board: b, width: 100, height: 40, columns: []string{"backlog", "todo", "done"}, collapsed: make(map[string]bool)}
	app.initColumnModels()
	widths := app.columnWidths()
	t.Logf("Widths: %v  sum=%d", widths, widths[0]+widths[1]+widths[2])
	t.Logf("Available should be: %d - 3*3 = %d", 100, 100-9)
	t.Logf("Expected total: sum + 3*3 = %d", widths[0]+widths[1]+widths[2]+9)

	board := app.renderBoard()
	for i, line := range strings.Split(board, "\n") {
		if i > 5 { break }
		t.Logf("L%d [w=%d]", i, lipgloss.Width(line))
	}
}
