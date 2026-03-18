package tui

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

func TestBorderDebug_FullBoard(t *testing.T) {
	b := domain.NewBoard("debug")
	b.Items["epic-1"] = &domain.Item{
		ID: "epic-1", DisplayNum: 70, Type: domain.ItemTypeEpic,
		Title: "API Redesign: Migrate to Modular Versioned Architecture",
		Status: "backlog", Priority: domain.PriorityHigh,
	}
	b.Items["story-1"] = &domain.Item{
		ID: "story-1", DisplayNum: 71, Type: domain.ItemTypeStory,
		Title: "Phase 1: Route Restructuring",
		Status: "backlog", Priority: domain.PriorityHigh, ParentID: "epic-1",
	}
	b.Items["task-1"] = &domain.Item{
		ID: "task-1", DisplayNum: 74, Type: domain.ItemTypeTask,
		Title: "Create versioned router setup",
		Status: "backlog", Priority: domain.PriorityHigh, ParentID: "story-1",
	}
	b.DisplayMap[70] = "epic-1"
	b.DisplayMap[71] = "story-1"
	b.DisplayMap[74] = "task-1"

	app := App{
		board:        b,
		width:        50,
		height:       30,
		columns:      []string{"backlog"},
		collapsed:    make(map[string]bool),
		customWidths: make(map[int]int),
	}
	app.initColumnModels()

	board := app.renderBoard()
	lines := strings.Split(board, "\n")

	t.Log("\n========== FULL RENDERED BOARD ==========")
	for i, line := range lines {
		visW := lipgloss.Width(line)
		t.Logf("L%02d [vis=%2d] %s", i, visW, line)
	}

	// Now show raw hex of the story card lines (lines around card #71)
	t.Log("\n========== RAW HEX OF STORY CARD RIGHT EDGE ==========")
	for i, line := range lines {
		if i < 3 || i > 20 {
			continue
		}
		// Get last 30 bytes
		b := []byte(line)
		start := len(b) - 30
		if start < 0 {
			start = 0
		}
		tail := b[start:]
		t.Logf("L%02d hex_tail: %s", i, hex.EncodeToString(tail))
		t.Logf("L%02d str_tail: %q", i, string(tail))
	}
}
