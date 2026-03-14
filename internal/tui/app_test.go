package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

// testBoard creates a board.json with realistic data including long titles.
func testBoard(t *testing.T) (string, *engine.Engine) {
	t.Helper()
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	board := map[string]interface{}{
		"version":      1,
		"name":         "test-board",
		"next_display": 10,
		"columns": []map[string]string{
			{"name": "backlog"},
			{"name": "todo"},
			{"name": "in-progress"},
			{"name": "review"},
			{"name": "done"},
		},
		"items": map[string]interface{}{
			"item-1": map[string]interface{}{
				"id": "item-1", "display_num": 1,
				"title":    "Short task",
				"type":     "task",
				"status":   "backlog",
				"priority": "medium",
			},
			"item-2": map[string]interface{}{
				"id": "item-2", "display_num": 2,
				"title":       "Add token refresh handling for expired sessions with automatic retry logic",
				"description": "Handle JWT token expiry by auto refresh. When a 401 is received, attempt to refresh before failing.",
				"type":        "story",
				"status":      "backlog",
				"priority":    "high",
			},
			"item-3": map[string]interface{}{
				"id": "item-3", "display_num": 3,
				"title":    "Another task in todo column",
				"type":     "task",
				"status":   "todo",
				"priority": "low",
			},
			"item-4": map[string]interface{}{
				"id": "item-4", "display_num": 4,
				"title":    "Implement the full end-to-end integration test suite for cloud synchronization with real-time WebSocket events",
				"type":     "task",
				"status":   "in-progress",
				"priority": "critical",
			},
			"item-5": map[string]interface{}{
				"id": "item-5", "display_num": 5,
				"title":    "Fix bug in review",
				"type":     "task",
				"status":   "review",
				"priority": "medium",
			},
			"item-6": map[string]interface{}{
				"id": "item-6", "display_num": 6,
				"title":    "Done task",
				"type":     "task",
				"status":   "done",
				"priority": "low",
			},
		},
		"display_map": map[string]string{
			"1": "item-1", "2": "item-2", "3": "item-3",
			"4": "item-4", "5": "item-5", "6": "item-6",
		},
		"users":    map[string]interface{}{},
		"plans":    map[string]interface{}{},
		"projects": map[string]interface{}{},
	}

	data, _ := json.MarshalIndent(board, "", "  ")
	boardFile := filepath.Join(obeyaDir, "board.json")
	os.WriteFile(boardFile, data, 0644)

	s := store.NewJSONStore(dir)
	eng := engine.New(s)
	return boardFile, eng
}

// getScreen sends a quit command, gets the final model, and calls View()
// to get the rendered screen as a string.
func getScreen(t *testing.T, tm *teatest.TestModel) string {
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	model := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	return model.(App).View()
}

// startAndWait creates a test model, waits for the board to load, returns the model.
func startAndWait(t *testing.T, eng *engine.Engine, boardFile string, width, height int) *teatest.TestModel {
	t.Helper()
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(width, height))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG")) ||
			bytes.Contains(bts, []byte("Loading board"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))

	// Give time for the board to fully load and render
	time.Sleep(100 * time.Millisecond)

	return tm
}

func TestTUI_FullScreenRender(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)

	screen := getScreen(t, tm)
	t.Logf("\n=== FULL SCREEN RENDER (120x40) ===\n%s", screen)

	for _, col := range []string{"BACKLOG", "TODO", "IN-PROGRESS", "REVIEW", "DONE"} {
		if !bytes.Contains([]byte(screen), []byte(col)) {
			t.Errorf("column header %q missing", col)
		}
	}

	if !bytes.Contains([]byte(screen), []byte("h/l:columns")) {
		t.Error("help bar missing")
	}
}

func TestTUI_ColumnNavigation(t *testing.T) {
	boardFile, eng := testBoard(t)

	t.Run("initial_state_backlog_active", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)
		screen := getScreen(t, tm)
		t.Logf("\n=== INITIAL STATE ===\n%s", screen)

		// Item #2 (highest display_num in backlog) should be selected
		if !bytes.Contains([]byte(screen), []byte("#2")) {
			t.Error("expected item #2 to be visible in initial state")
		}
	})

	t.Run("press_l_moves_to_todo", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 'l' (expect TODO active) ===\n%s", screen)

		// Item #3 is in TODO — should be selected after moving right
		if !bytes.Contains([]byte(screen), []byte("#3")) {
			t.Error("expected item #3 in TODO to be visible")
		}
	})

	t.Run("press_l_twice_moves_to_in_progress", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 'l' x2 (expect IN-PROGRESS active) ===\n%s", screen)
	})

	t.Run("press_h_moves_left", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
		time.Sleep(50 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 'l' then 'h' (expect BACKLOG active) ===\n%s", screen)
	})

	t.Run("tab_cycles_columns", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		// Tab through all 5 columns and back to start
		for i := 0; i < 5; i++ {
			tm.Send(tea.KeyMsg{Type: tea.KeyTab})
			time.Sleep(50 * time.Millisecond)
		}

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 5 tabs (should be back to BACKLOG) ===\n%s", screen)
	})
}

func TestTUI_CardNavigation(t *testing.T) {
	boardFile, eng := testBoard(t)

	t.Run("press_j_moves_down", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		// Backlog has items #2 (first, newest) and #1. Press j to move to #1
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 'j' (move down in backlog) ===\n%s", screen)
	})

	t.Run("press_k_moves_up", func(t *testing.T) {
		tm := startAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(50 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== AFTER 'j' then 'k' (back to top) ===\n%s", screen)
	})
}

func TestTUI_NarrowTerminal(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 80, 24)

	screen := getScreen(t, tm)
	t.Logf("\n=== NARROW TERMINAL (80x24) ===\n%s", screen)

	// Should still render without panicking
	for _, col := range []string{"BACKLOG", "TODO", "IN-PROGRESS", "REVIEW", "DONE"} {
		if !bytes.Contains([]byte(screen), []byte(col)) {
			t.Errorf("column header %q missing in narrow terminal", col)
		}
	}
}

func TestTUI_RealBoard(t *testing.T) {
	boardFile := "/Users/niladribose/code/obeya/.obeya/board.json"
	if _, err := os.Stat(boardFile); err != nil {
		t.Skip("real board.json not found, skipping")
	}

	gitRoot := "/Users/niladribose/code/obeya"
	s := store.NewJSONStore(gitRoot)
	eng := engine.New(s)
	app := NewApp(eng, boardFile)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Give time for async board load
	time.Sleep(500 * time.Millisecond)

	screen := getScreen(t, tm)
	t.Logf("\n=== REAL BOARD (120x40) ===\n%s", screen)

	lines := bytes.Split([]byte(screen), []byte("\n"))
	t.Logf("\n=== Screen lines: %d ===", len(lines))
}

func TestTUI_RealBoard_Navigation(t *testing.T) {
	boardFile := "/Users/niladribose/code/obeya/.obeya/board.json"
	if _, err := os.Stat(boardFile); err != nil {
		t.Skip("real board.json not found, skipping")
	}

	gitRoot := "/Users/niladribose/code/obeya"
	s := store.NewJSONStore(gitRoot)
	eng := engine.New(s)

	t.Run("press_l_moves_right", func(t *testing.T) {
		app := NewApp(eng, boardFile)
		tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
		time.Sleep(500 * time.Millisecond)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(100 * time.Millisecond)

		screen := getScreen(t, tm)
		t.Logf("\n=== REAL BOARD AFTER 'l' ===\n%s", screen)
	})

	t.Run("press_j_x5", func(t *testing.T) {
		app := NewApp(eng, boardFile)
		tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
		time.Sleep(500 * time.Millisecond)

		for i := 0; i < 5; i++ {
			tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
			time.Sleep(50 * time.Millisecond)
		}

		screen := getScreen(t, tm)
		t.Logf("\n=== REAL BOARD AFTER 5x 'j' ===\n%s", screen)
	})
}

func TestTUI_ScrollbarVisible(t *testing.T) {
	boardFile := "/Users/niladribose/code/obeya/.obeya/board.json"
	if _, err := os.Stat(boardFile); err != nil {
		t.Skip("real board.json not found, skipping")
	}

	gitRoot := "/Users/niladribose/code/obeya"
	s := store.NewJSONStore(gitRoot)
	eng := engine.New(s)
	app := NewApp(eng, boardFile)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Give time for async board load
	time.Sleep(500 * time.Millisecond)

	screen := getScreen(t, tm)
	t.Logf("\n=== SCROLLBAR TEST (120x40) ===\n%s", screen)

	if !strings.Contains(screen, "┃") {
		t.Error("expected scrollbar thumb character ┃ to be visible in rendered output")
	}
}

func TestTUI_DescriptionAccordion(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)

	// Press 'v' to expand description on the selected item (item-2 in backlog)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(100 * time.Millisecond)

	screen := getScreen(t, tm)
	t.Logf("\n=== DESCRIPTION ACCORDION (after 'v') ===\n%s", screen)

	if !strings.Contains(screen, "▼ description") {
		t.Error("expected expanded description indicator '▼ description' in output")
	}
}

func TestTUI_Resize(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)

	// Send resize message
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	time.Sleep(100 * time.Millisecond)

	screen := getScreen(t, tm)
	t.Logf("\n=== RESIZE TEST (80x24) ===\n%s", screen)

	lines := strings.Split(screen, "\n")
	t.Logf("Line count after resize: %d", len(lines))

	if len(lines) > 26 {
		t.Errorf("expected <= 26 lines after resize to 80x24, got %d", len(lines))
	}
}

// TestTUI_RenderCard_Isolation tests renderCard directly to check for blank lines.
func TestTUI_RenderCard_Isolation(t *testing.T) {
	boardFile := "/Users/niladribose/code/obeya/.obeya/board.json"
	if _, err := os.Stat(boardFile); err != nil {
		t.Skip("real board.json not found, skipping")
	}

	gitRoot := "/Users/niladribose/code/obeya"
	s := store.NewJSONStore(gitRoot)
	eng := engine.New(s)
	board, err := eng.ListBoard()
	if err != nil {
		t.Fatal(err)
	}

	app := App{
		board:     board,
		columns:   extractColumns(board),
		collapsed: make(map[string]bool),
		width:     120,
		height:    40,
	}
	app.initColumnModels()

	// Render a few cards with long titles
	for _, item := range board.Items {
		if len(item.Title) > 30 {
			rendered := app.renderCard(item, false)
			lines := strings.Split(rendered, "\n")
			t.Logf("\n=== Card #%d (%d chars title, %d rendered lines) ===\n%s",
				item.DisplayNum, len(item.Title), len(lines), rendered)

			// Check for blank lines inside the card
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				// Skip first/last lines (borders)
				if i > 0 && i < len(lines)-1 && trimmed == "│" {
					t.Errorf("Card #%d has blank interior line at position %d", item.DisplayNum, i)
				}
			}
			break // just test one
		}
	}
}
