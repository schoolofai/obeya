package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

// TestGolden_InitialRender captures the initial board state at 120x40.
func TestGolden_InitialRender(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_MoveToTodo captures the board after pressing 'l' (cursor on TODO column).
func TestGolden_MoveToTodo(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_MoveDownCard captures the board after pressing 'j' (second card selected).
func TestGolden_MoveDownCard(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DescriptionExpanded captures the board with description accordion open.
func TestGolden_DescriptionExpanded(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DescriptionCollapsed captures the board after expanding then collapsing description.
func TestGolden_DescriptionCollapsed(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_NarrowTerminal captures the board at 80x24.
func TestGolden_NarrowTerminal(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 80, 24)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_TabCycle captures the board after tabbing through all columns back to start.
func TestGolden_TabCycle(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	for i := 0; i < 5; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(30 * time.Millisecond)
	}
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_InProgressColumn captures the board with cursor on in-progress column.
func TestGolden_InProgressColumn(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	// Move right twice: backlog -> todo -> in-progress
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(30 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DAGView captures the DAG view at 120x40 with hierarchy board.
func TestGolden_DAGView(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	// Enter DAG view
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("DAG View"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DAGView_Navigate captures the DAG after moving to next node.
func TestGolden_DAGView_Navigate(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("DAG View"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	// Navigate to next node
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// testReviewBoard creates a board with agent-completed items that have ReviewContext.
func testReviewBoard(t *testing.T) (string, *engine.Engine) {
	t.Helper()
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	conf45 := 45
	conf85 := 85
	board := map[string]interface{}{
		"version":      1,
		"name":         "review-test-board",
		"next_display": 7,
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
				"title": "Setup auth module", "type": "task",
				"status": "backlog", "priority": "medium",
				"assignee": "agent-1",
			},
			"item-2": map[string]interface{}{
				"id": "item-2", "display_num": 2,
				"title": "Refactor JWT middleware", "type": "task",
				"status": "done", "priority": "high",
				"assignee": "agent-1", "sponsor": "human-1",
				"confidence": &conf45,
				"review_context": map[string]interface{}{
					"purpose": "Replace cookie sessions with JWT",
					"files_changed": []map[string]interface{}{
						{"path": "auth/middleware.go", "added": 82, "removed": 41},
					},
					"tests_written": []map[string]interface{}{{"name": "TestJWT", "passed": true}},
					"reproduce":     []string{"go test ./auth/ -run TestJWT"},
				},
				"human_review": map[string]interface{}{"status": "pending"},
			},
			"item-3": map[string]interface{}{
				"id": "item-3", "display_num": 3,
				"title": "Add rate limiting", "type": "task",
				"status": "done", "priority": "medium",
				"assignee": "agent-1", "sponsor": "human-1",
				"confidence": &conf85,
				"review_context": map[string]interface{}{
					"purpose": "Prevent API abuse",
					"files_changed": []map[string]interface{}{
						{"path": "api/ratelimit.go", "added": 45, "removed": 0},
					},
					"tests_written": []map[string]interface{}{
						{"name": "TestRateLimit", "passed": true},
						{"name": "TestRateLimitBurst", "passed": true},
					},
				},
				"human_review": map[string]interface{}{"status": "pending"},
			},
			"item-4": map[string]interface{}{
				"id": "item-4", "display_num": 4,
				"title": "Fix login bug", "type": "task",
				"status": "done", "priority": "critical",
				"assignee": "human-1",
			},
			"item-5": map[string]interface{}{
				"id": "item-5", "display_num": 5,
				"title": "Reviewed task", "type": "task",
				"status": "done", "priority": "low",
				"assignee": "agent-1",
				"review_context": map[string]interface{}{"purpose": "Already reviewed"},
				"human_review":   map[string]interface{}{"status": "reviewed", "reviewed_by": "human-1"},
			},
		},
		"display_map": map[string]string{
			"1": "item-1", "2": "item-2", "3": "item-3",
			"4": "item-4", "5": "item-5",
		},
		"users": map[string]interface{}{
			"agent-1": map[string]interface{}{
				"id": "agent-1", "name": "claude", "type": "agent",
			},
			"human-1": map[string]interface{}{
				"id": "human-1", "name": "niladri", "type": "human",
			},
		},
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

func startAndWaitReview(t *testing.T, eng *engine.Engine, boardFile string) *teatest.TestModel {
	t.Helper()
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG")) ||
			bytes.Contains(bts, []byte("REVIEW QUEUE"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	return tm
}

// TestGolden_HumanReviewColumn captures the board with the virtual review queue column.
func TestGolden_HumanReviewColumn(t *testing.T) {
	boardFile, eng := testReviewBoard(t)
	tm := startAndWaitReview(t, eng, boardFile)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_ReviewColumnSelected captures the review queue column when selected.
func TestGolden_ReviewColumnSelected(t *testing.T) {
	boardFile, eng := testReviewBoard(t)
	tm := startAndWaitReview(t, eng, boardFile)
	// Navigate to the review queue column (last column, 6th including virtual)
	for i := 0; i < 5; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_PastReviewsTree captures the past reviews pane with hierarchical tree.
func TestGolden_PastReviewsTree(t *testing.T) {
	boardFile, eng := testReviewBoard(t)
	tm := startAndWaitReview(t, eng, boardFile)
	// Open past reviews pane
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Past Reviews"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// testHierarchyGoldenBoard creates a board with epic → story → task hierarchy for golden tests.
func testHierarchyGoldenBoard(t *testing.T) (string, *engine.Engine) {
	t.Helper()
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	board := map[string]interface{}{
		"version":      1,
		"name":         "hierarchy-board",
		"next_display": 8,
		"columns": []map[string]string{
			{"name": "backlog"},
			{"name": "in-progress"},
			{"name": "done"},
		},
		"items": map[string]interface{}{
			"epic-1": map[string]interface{}{
				"id": "epic-1", "display_num": 1,
				"title": "Auth Rewrite", "type": "epic",
				"status": "backlog", "priority": "high",
			},
			"story-2": map[string]interface{}{
				"id": "story-2", "display_num": 2,
				"title": "Session Mgmt", "type": "story",
				"status": "backlog", "priority": "medium",
				"parent_id": "epic-1",
			},
			"task-3": map[string]interface{}{
				"id": "task-3", "display_num": 3,
				"title": "Refactor middleware", "type": "task",
				"status": "backlog", "priority": "medium",
				"parent_id": "story-2",
			},
			"task-4": map[string]interface{}{
				"id": "task-4", "display_num": 4,
				"title": "JWT validation", "type": "task",
				"status": "in-progress", "priority": "high",
				"parent_id": "story-2",
			},
			"task-5": map[string]interface{}{
				"id": "task-5", "display_num": 5,
				"title": "Update session store", "type": "task",
				"status": "done", "priority": "low",
				"parent_id": "story-2",
			},
			"task-6": map[string]interface{}{
				"id": "task-6", "display_num": 6,
				"title": "Fix README typo", "type": "task",
				"status": "backlog", "priority": "low",
			},
		},
		"display_map": map[string]string{
			"1": "epic-1", "2": "story-2", "3": "task-3",
			"4": "task-4", "5": "task-5", "6": "task-6",
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

// TestGolden_UnifiedHierarchyExpanded captures hierarchy board with all items expanded.
func TestGolden_UnifiedHierarchyExpanded(t *testing.T) {
	boardFile, eng := testHierarchyGoldenBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_UnifiedHierarchyCollapsed captures hierarchy board with epic collapsed in backlog.
func TestGolden_UnifiedHierarchyCollapsed(t *testing.T) {
	boardFile, eng := testHierarchyGoldenBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	// Press Space to collapse the epic (#1 is first selected item)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_UnifiedHierarchyCrossColumn captures hierarchy with tasks in different columns showing breadcrumbs.
func TestGolden_UnifiedHierarchyCrossColumn(t *testing.T) {
	boardFile, eng := testHierarchyGoldenBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	// Move to in-progress column to see cross-column task with breadcrumb
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_AgentCardExpanded captures an agent card with review context expanded.
func TestGolden_AgentCardExpanded(t *testing.T) {
	boardFile, eng := testReviewBoard(t)
	tm := startAndWaitReview(t, eng, boardFile)
	// Navigate to the review queue column
	for i := 0; i < 5; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	// Expand review context on selected card
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}
