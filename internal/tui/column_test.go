package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
)

func TestColumnModel_New(t *testing.T) {
	col := NewColumnModel("backlog", 30, 10)

	if col.Name != "backlog" {
		t.Errorf("expected Name=%q, got %q", "backlog", col.Name)
	}
	if col.viewport.Width != 30 {
		t.Errorf("expected viewport Width=30, got %d", col.viewport.Width)
	}
	if col.viewport.Height != 10 {
		t.Errorf("expected viewport Height=10, got %d", col.viewport.Height)
	}
}

func TestColumnModel_View(t *testing.T) {
	col := NewColumnModel("in-progress", 30, 10)
	col.active = true
	col.SetContent("card one\ncard two\ncard three")

	view := col.View(3)

	if !strings.Contains(view, "IN-PROGRESS") {
		t.Error("expected View to contain column name 'IN-PROGRESS'")
	}
	if !strings.Contains(view, "3") {
		t.Error("expected View to contain item count '3'")
	}
	if !strings.Contains(view, "card one") {
		t.Error("expected View to contain content 'card one'")
	}
	if !strings.Contains(view, "card two") {
		t.Error("expected View to contain content 'card two'")
	}
}

func TestColumnModel_ScrollbarAppears(t *testing.T) {
	// Test overlayScrollbar directly to avoid border character confusion.
	vp := viewport.New(30, 5)
	vp.KeyMap = viewport.KeyMap{}

	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "item line padded to width ----"
	}
	vp.SetContent(strings.Join(lines, "\n"))

	result := overlayScrollbar(vp.View(), vp)

	hasThumb := strings.Contains(result, "┃")
	hasTrack := strings.Contains(result, "│")
	if !hasThumb && !hasTrack {
		t.Error("expected scrollbar characters (┃ or │) when content overflows viewport")
	}
	if !hasThumb {
		t.Error("expected scrollbar thumb (┃) to appear for overflow content")
	}
}

func TestColumnModel_NoScrollbarWhenFits(t *testing.T) {
	// Test overlayScrollbar directly to avoid border character confusion.
	vp := viewport.New(30, 10)
	vp.KeyMap = viewport.KeyMap{}
	vp.SetContent("line one\nline two\nline three")

	result := overlayScrollbar(vp.View(), vp)

	if strings.Contains(result, "┃") {
		t.Error("expected no scrollbar thumb (┃) when content fits in viewport")
	}
	// │ should not appear because overlayScrollbar returns input unchanged
	// when content fits. The raw viewport output has no │ characters.
	if strings.Contains(result, "│") {
		t.Error("expected no scrollbar track (│) when content fits in viewport")
	}
}
