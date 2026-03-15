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
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

// testDAGBoard creates a board with hierarchical parent-child relationships
// and blocker dependencies, suitable for testing the DAG layout and rendering.
func testDAGBoard(t *testing.T) (string, *engine.Engine) {
	t.Helper()
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	board := map[string]interface{}{
		"version":      1,
		"name":         "dag-test-board",
		"next_display": 10,
		"columns": []map[string]string{
			{"name": "backlog"},
			{"name": "todo"},
			{"name": "in-progress"},
			{"name": "review"},
			{"name": "done"},
		},
		"items": map[string]interface{}{
			// Epic: Auth System
			"epic-1": map[string]interface{}{
				"id": "epic-1", "display_num": 1,
				"title":    "Auth System",
				"type":     "epic",
				"status":   "in-progress",
				"priority": "high",
			},
			// Story under Auth — independent child
			"story-2": map[string]interface{}{
				"id": "story-2", "display_num": 2,
				"title":     "OAuth Integration",
				"type":      "story",
				"status":    "todo",
				"priority":  "high",
				"parent_id": "epic-1",
			},
			// Story under Auth — independent child
			"story-3": map[string]interface{}{
				"id": "story-3", "display_num": 3,
				"title":     "DB Schema",
				"type":      "story",
				"status":    "done",
				"priority":  "medium",
				"parent_id": "epic-1",
			},
			// Story under Auth — blocked by story-2
			"story-4": map[string]interface{}{
				"id": "story-4", "display_num": 4,
				"title":      "API Endpoints",
				"type":       "story",
				"status":     "in-progress",
				"priority":   "critical",
				"parent_id":  "epic-1",
				"blocked_by": []string{"story-2"},
			},
			// Task under story-4 — blocked by story-4
			"task-5": map[string]interface{}{
				"id": "task-5", "display_num": 5,
				"title":      "Token Refresh",
				"type":       "task",
				"status":     "backlog",
				"priority":   "medium",
				"parent_id":  "story-4",
				"blocked_by": []string{"story-4"},
			},
			// Orphan task — no parent
			"task-6": map[string]interface{}{
				"id": "task-6", "display_num": 6,
				"title":    "Fix CI Pipeline",
				"type":     "task",
				"status":   "in-progress",
				"priority": "high",
			},
			// Another orphan — blocked by task-6
			"task-7": map[string]interface{}{
				"id": "task-7", "display_num": 7,
				"title":      "Deploy to Staging",
				"type":       "task",
				"status":     "todo",
				"priority":   "medium",
				"blocked_by": []string{"task-6"},
			},
		},
		"display_map": map[string]string{
			"1": "epic-1", "2": "story-2", "3": "story-3",
			"4": "story-4", "5": "task-5", "6": "task-6", "7": "task-7",
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

// makeDomainBoard creates a domain.Board from items for direct unit testing
// without needing the engine/store layer.
func makeDomainBoard(items map[string]*domain.Item) *domain.Board {
	board := domain.NewBoard("unit-test")
	board.Items = items
	displayMap := make(map[int]string)
	for _, item := range items {
		displayMap[item.DisplayNum] = item.ID
	}
	board.DisplayMap = displayMap
	return board
}

// ============================================================
// UNIT TESTS — dag_layout.go
// ============================================================

func TestBuildDAGGraph_EmptyBoard(t *testing.T) {
	board := domain.NewBoard("empty")
	g := buildDAGGraph(board)

	if len(g.nodes) != 0 {
		t.Errorf("expected 0 nodes for empty board, got %d", len(g.nodes))
	}
	if len(g.edges) != 0 {
		t.Errorf("expected 0 edges for empty board, got %d", len(g.edges))
	}
	if len(g.lanes) != 0 {
		t.Errorf("expected 0 lanes for empty board, got %d", len(g.lanes))
	}
}

func TestBuildDAGGraph_NilBoard(t *testing.T) {
	g := buildDAGGraph(nil)
	if len(g.nodes) != 0 {
		t.Errorf("expected 0 nodes for nil board, got %d", len(g.nodes))
	}
}

func TestBuildDAGGraph_SingleEpic(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic One", Type: domain.ItemTypeEpic, Status: "backlog", Priority: domain.PriorityMedium},
	})

	g := buildDAGGraph(board)

	if len(g.nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.nodes))
	}
	if g.nodes[0].item.Title != "Epic One" {
		t.Errorf("expected node title 'Epic One', got %q", g.nodes[0].item.Title)
	}
	if len(g.lanes) != 1 {
		t.Fatalf("expected 1 lane, got %d", len(g.lanes))
	}
	if g.lanes[0].label != "Epic One" {
		t.Errorf("expected lane label 'Epic One', got %q", g.lanes[0].label)
	}
}

func TestBuildDAGGraph_EpicWithChildren(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Story A", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
		"s2": {ID: "s2", DisplayNum: 3, Title: "Story B", Type: domain.ItemTypeStory, Status: "done", Priority: domain.PriorityMedium, ParentID: "e1"},
	})

	g := buildDAGGraph(board)

	if len(g.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(g.nodes))
	}

	// Epic should be at gridCol 0
	epicNode := findNodeByID(g, "e1")
	if epicNode == nil {
		t.Fatal("epic node not found")
	}
	if epicNode.gridCol != 0 {
		t.Errorf("epic gridCol expected 0, got %d", epicNode.gridCol)
	}

	// Children should be at gridCol 1 (independent siblings)
	s1 := findNodeByID(g, "s1")
	s2 := findNodeByID(g, "s2")
	if s1 == nil || s2 == nil {
		t.Fatal("child nodes not found")
	}
	if s1.gridCol != 1 {
		t.Errorf("story A gridCol expected 1, got %d", s1.gridCol)
	}
	if s2.gridCol != 1 {
		t.Errorf("story B gridCol expected 1, got %d", s2.gridCol)
	}

	// Independent children should have different gridRows
	if s1.gridRow == s2.gridRow {
		t.Errorf("independent siblings should have different gridRows, both got %d", s1.gridRow)
	}

	// Should have 2 parent edges (epic -> s1, epic -> s2)
	parentEdges := 0
	for _, e := range g.edges {
		if e.edgeKind == "parent" {
			parentEdges++
		}
	}
	if parentEdges != 2 {
		t.Errorf("expected 2 parent edges, got %d", parentEdges)
	}
}

func TestBuildDAGGraph_BlockerDependency(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "First", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
		"s2": {ID: "s2", DisplayNum: 3, Title: "Second", Type: domain.ItemTypeStory, Status: "backlog", Priority: domain.PriorityMedium, ParentID: "e1", BlockedBy: []string{"s1"}},
	})

	g := buildDAGGraph(board)

	s1 := findNodeByID(g, "s1")
	s2 := findNodeByID(g, "s2")
	if s1 == nil || s2 == nil {
		t.Fatal("nodes not found")
	}

	// s2 is blocked by s1, so s2 should be at a higher gridCol
	if s2.gridCol <= s1.gridCol {
		t.Errorf("blocked node should be at higher gridCol: s1=%d, s2=%d", s1.gridCol, s2.gridCol)
	}

	// Should have a blocker edge
	hasBlockerEdge := false
	for _, e := range g.edges {
		if e.edgeKind == "blocker" {
			from := g.nodes[e.fromIdx]
			to := g.nodes[e.toIdx]
			if from.id == "s1" && to.id == "s2" {
				hasBlockerEdge = true
			}
		}
	}
	if !hasBlockerEdge {
		t.Error("expected blocker edge from s1 to s2")
	}
}

func TestBuildDAGGraph_OrphanLane(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Orphan A", Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium},
		"t2": {ID: "t2", DisplayNum: 2, Title: "Orphan B", Type: domain.ItemTypeTask, Status: "in-progress", Priority: domain.PriorityHigh},
	})

	g := buildDAGGraph(board)

	if len(g.nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.nodes))
	}

	// Should create an orphan lane
	hasOrphanLane := false
	for _, lane := range g.lanes {
		if lane.epicID == "" {
			hasOrphanLane = true
			if lane.label != "Independent Tasks" {
				t.Errorf("orphan lane label expected 'Independent Tasks', got %q", lane.label)
			}
		}
	}
	if !hasOrphanLane {
		t.Error("expected an orphan lane")
	}
}

func TestBuildDAGGraph_MixedEpicAndOrphan(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Story", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
		"t1": {ID: "t1", DisplayNum: 3, Title: "Orphan", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityLow},
	})

	g := buildDAGGraph(board)

	if len(g.lanes) != 2 {
		t.Fatalf("expected 2 lanes (epic + orphan), got %d", len(g.lanes))
	}
	if g.lanes[0].epicID != "e1" {
		t.Errorf("first lane should be for epic e1, got %q", g.lanes[0].epicID)
	}
	if g.lanes[1].epicID != "" {
		t.Errorf("second lane should be orphan lane (empty epicID), got %q", g.lanes[1].epicID)
	}
}

func TestBuildDAGGraph_FirstInProgressNode(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Backlog", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium},
		"t2": {ID: "t2", DisplayNum: 2, Title: "In Progress", Type: domain.ItemTypeTask, Status: "in-progress", Priority: domain.PriorityHigh},
		"t3": {ID: "t3", DisplayNum: 3, Title: "Done", Type: domain.ItemTypeTask, Status: "done", Priority: domain.PriorityLow},
	})

	g := buildDAGGraph(board)
	idx := g.firstInProgressNode()

	if idx < 0 {
		t.Fatal("expected to find an in-progress node")
	}
	if g.nodes[idx].item.Status != "in-progress" {
		t.Errorf("expected in-progress node, got status %q", g.nodes[idx].item.Status)
	}
}

func TestBuildDAGGraph_NoInProgressNode(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Done", Type: domain.ItemTypeTask, Status: "done", Priority: domain.PriorityMedium},
	})

	g := buildDAGGraph(board)
	idx := g.firstInProgressNode()

	if idx != -1 {
		t.Errorf("expected -1 for no in-progress nodes, got %d", idx)
	}
}

func TestBuildDAGGraph_CanvasPositions(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Story", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
	})

	g := buildDAGGraph(board)

	epic := findNodeByID(g, "e1")
	story := findNodeByID(g, "s1")
	if epic == nil || story == nil {
		t.Fatal("nodes not found")
	}

	// Epic at gridCol 0 should be at x=0
	if epic.x != 0 {
		t.Errorf("epic x expected 0, got %d", epic.x)
	}

	// Story at gridCol 1 should be offset by dagNodeW + dagGapX
	expectedX := 1 * (dagNodeW + dagGapX)
	if story.x != expectedX {
		t.Errorf("story x expected %d, got %d", expectedX, story.x)
	}

	// Both should have non-negative y
	if epic.y < 0 || story.y < 0 {
		t.Errorf("expected non-negative y positions, got epic.y=%d, story.y=%d", epic.y, story.y)
	}

	// Canvas should have positive dimensions
	if g.width <= 0 || g.height <= 0 {
		t.Errorf("expected positive canvas dimensions, got %dx%d", g.width, g.height)
	}
}

func TestBuildDAGGraph_DeepHierarchy(t *testing.T) {
	// Epic -> Story -> Task (grandchild should appear in the DAG)
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "backlog", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Story", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
		"t1": {ID: "t1", DisplayNum: 3, Title: "Task", Type: domain.ItemTypeTask, Status: "in-progress", Priority: domain.PriorityLow, ParentID: "s1"},
	})

	g := buildDAGGraph(board)

	if len(g.nodes) != 3 {
		t.Fatalf("expected 3 nodes (epic + story + task), got %d", len(g.nodes))
	}

	// All should be in one lane
	if len(g.lanes) != 1 {
		t.Errorf("expected 1 lane, got %d", len(g.lanes))
	}
}

// ============================================================
// UNIT TESTS — dag_render.go
// ============================================================

func TestRenderDAGNode_Styles(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		selected bool
		tick     int
		wantSub  string // substring that should appear
	}{
		{"backlog_unselected", "backlog", false, 0, "#1"},
		{"selected", "backlog", true, 0, "#1"},
		{"in_progress_bright", "in-progress", false, 0, "🔥"},
		{"in_progress_dim", "in-progress", false, 1, "🔥"},
		{"done", "done", false, 0, "✓"},
		{"todo", "todo", false, 0, "#1"},
		{"review", "review", false, 0, "#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := dagNode{
				item: &domain.Item{
					ID:         "t1",
					DisplayNum: 1,
					Title:      "Test Node",
					Type:       domain.ItemTypeTask,
					Status:     tt.status,
					Priority:   domain.PriorityMedium,
				},
				w: dagNodeW,
				h: dagNodeH,
			}
			result := renderDAGNode(node, tt.selected, tt.tick)
			if !strings.Contains(result, tt.wantSub) {
				t.Errorf("renderDAGNode(%s) missing substring %q in:\n%s", tt.name, tt.wantSub, result)
			}
		})
	}
}

func TestRenderDAGNode_TitleTruncation(t *testing.T) {
	node := dagNode{
		item: &domain.Item{
			ID:         "t1",
			DisplayNum: 1,
			Title:      "This is a very long title that should definitely be truncated to fit in the node box",
			Type:       domain.ItemTypeTask,
			Status:     "backlog",
			Priority:   domain.PriorityMedium,
		},
		w: dagNodeW,
		h: dagNodeH,
	}
	result := renderDAGNode(node, false, 0)

	// Should not contain the full title
	if strings.Contains(result, "should definitely be truncated") {
		t.Error("expected title to be truncated, but full title appeared")
	}
	// Should contain the display number
	if !strings.Contains(result, "#1") {
		t.Error("expected display number #1 in output")
	}
}

func TestRenderStatusBar(t *testing.T) {
	tests := []struct {
		status  string
		wantSub string
	}{
		{"done", "✓"},
		{"in-progress", "🔥"},
		{"review", "█"},
		{"todo", "░"},
		{"backlog", "backlog"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			item := &domain.Item{Status: tt.status}
			result := renderStatusBar(item, 12)
			if !strings.Contains(result, tt.wantSub) {
				t.Errorf("renderStatusBar(%s) missing %q in: %s", tt.status, tt.wantSub, result)
			}
		})
	}
}

func TestRenderDAGCanvas_Empty(t *testing.T) {
	g := dagGraph{}
	result := renderDAGCanvas(g, -1, 0)
	if !strings.Contains(result, "No items") {
		t.Errorf("expected 'No items' for empty graph, got: %s", result)
	}
}

func TestRenderDAGCanvas_WithNodes(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Story", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
	})

	g := buildDAGGraph(board)
	canvas := renderDAGCanvas(g, 0, 0)

	// Should contain the lane header
	if !strings.Contains(canvas, "Epic") {
		t.Error("expected lane label 'Epic' in canvas")
	}
	// Should contain the node titles
	if !strings.Contains(canvas, "#1") {
		t.Error("expected node #1 in canvas")
	}
	if !strings.Contains(canvas, "#2") {
		t.Error("expected node #2 in canvas")
	}
}

func TestDrawEdge_StraightHorizontal(t *testing.T) {
	canvas := make([][]rune, 5)
	for r := range canvas {
		canvas[r] = make([]rune, 30)
		for c := range canvas[r] {
			canvas[r][c] = ' '
		}
	}

	drawEdge(canvas, 5, 2, 15, 2, "parent")

	// Check that horizontal line chars were drawn
	hasArrow := false
	hasLine := false
	for c := 5; c <= 15; c++ {
		if canvas[2][c] == '▶' {
			hasArrow = true
		}
		if canvas[2][c] == '─' {
			hasLine = true
		}
	}
	if !hasArrow {
		t.Error("expected arrow char '▶' in horizontal edge")
	}
	if !hasLine {
		t.Error("expected line char '─' in horizontal edge")
	}
}

func TestDrawEdge_LShapedVertical(t *testing.T) {
	canvas := make([][]rune, 10)
	for r := range canvas {
		canvas[r] = make([]rune, 30)
		for c := range canvas[r] {
			canvas[r][c] = ' '
		}
	}

	drawEdge(canvas, 5, 2, 20, 6, "parent")

	// Should have corners
	hasCorner := false
	for r := range canvas {
		for c := range canvas[r] {
			ch := canvas[r][c]
			if ch == '┐' || ch == '└' || ch == '┘' || ch == '┌' {
				hasCorner = true
			}
		}
	}
	if !hasCorner {
		t.Error("expected corner chars in L-shaped edge")
	}
}

func TestDrawEdge_BlockerStyle(t *testing.T) {
	canvas := make([][]rune, 5)
	for r := range canvas {
		canvas[r] = make([]rune, 30)
		for c := range canvas[r] {
			canvas[r][c] = ' '
		}
	}

	drawEdge(canvas, 5, 2, 15, 2, "blocker")

	// Blocker edges use dashed lines
	hasDash := false
	for c := 5; c < 15; c++ {
		if canvas[2][c] == '╌' {
			hasDash = true
		}
	}
	if !hasDash {
		t.Error("expected dashed char '╌' in blocker edge")
	}
}

func TestPlaceBox(t *testing.T) {
	base := ".....\n.....\n....."
	box := "AB\nCD"
	result := placeBox(base, box, 1, 1)

	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Line 1 should contain AB at position 1
	if !strings.Contains(lines[1], "AB") {
		t.Errorf("expected 'AB' in line 1, got: %q", lines[1])
	}
	// Line 2 should contain CD at position 1
	if !strings.Contains(lines[2], "CD") {
		t.Errorf("expected 'CD' in line 2, got: %q", lines[2])
	}
}

func TestRenderDAGHelp(t *testing.T) {
	help := renderDAGHelp()
	if !strings.Contains(help, "h/l:scroll") {
		t.Error("help bar missing 'h/l:scroll'")
	}
	if !strings.Contains(help, "G:board") {
		t.Error("help bar missing 'G:board'")
	}
	if !strings.Contains(help, "Enter:detail") {
		t.Error("help bar missing 'Enter:detail'")
	}
}

// ============================================================
// UNIT TESTS — dag.go (model)
// ============================================================

func TestDAGModel_New(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Task", Type: domain.ItemTypeTask, Status: "in-progress", Priority: domain.PriorityMedium},
	})

	m := newDAGModel(board, 120, 40)

	if !m.ready {
		t.Error("expected model to be ready")
	}
	if m.board != board {
		t.Error("expected model to reference the board")
	}
	if len(m.graph.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(m.graph.nodes))
	}
}

func TestDAGModel_AutoScrollToInProgress(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Backlog", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium},
		"t2": {ID: "t2", DisplayNum: 2, Title: "In Progress", Type: domain.ItemTypeTask, Status: "in-progress", Priority: domain.PriorityHigh},
	})

	m := newDAGModel(board, 120, 40)

	// Cursor should be on the in-progress node
	if m.cursorNode < 0 || m.cursorNode >= len(m.graph.nodes) {
		t.Fatal("cursor node out of range")
	}
	if m.graph.nodes[m.cursorNode].item.Status != "in-progress" {
		t.Errorf("expected cursor on in-progress node, got %q", m.graph.nodes[m.cursorNode].item.Status)
	}
}

func TestDAGModel_Navigation(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "A", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium},
		"t2": {ID: "t2", DisplayNum: 2, Title: "B", Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium},
		"t3": {ID: "t3", DisplayNum: 3, Title: "C", Type: domain.ItemTypeTask, Status: "done", Priority: domain.PriorityMedium},
	})

	m := newDAGModel(board, 120, 40)
	initial := m.cursorNode

	m.moveToNextNode()
	after1 := m.cursorNode
	if after1 == initial {
		t.Error("moveToNextNode did not change cursor")
	}

	m.moveToNextNode()
	after2 := m.cursorNode
	if after2 == after1 {
		t.Error("second moveToNextNode did not change cursor")
	}

	// Wrap around
	m.moveToNextNode()
	after3 := m.cursorNode
	if after3 != initial {
		t.Errorf("expected wrap to %d, got %d", initial, after3)
	}
}

func TestDAGModel_MovePrev(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "A", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium},
		"t2": {ID: "t2", DisplayNum: 2, Title: "B", Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium},
	})

	m := newDAGModel(board, 120, 40)
	m.cursorNode = 0

	m.moveToPrevNode()
	// Should wrap to last node
	if m.cursorNode != len(m.graph.nodes)-1 {
		t.Errorf("expected wrap to last node (%d), got %d", len(m.graph.nodes)-1, m.cursorNode)
	}
}

func TestDAGModel_Scrolling(t *testing.T) {
	// Create a wide graph (epic + chained children) that exceeds viewport
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
		"s1": {ID: "s1", DisplayNum: 2, Title: "Step 1", Type: domain.ItemTypeStory, Status: "todo", Priority: domain.PriorityMedium, ParentID: "e1"},
		"s2": {ID: "s2", DisplayNum: 3, Title: "Step 2", Type: domain.ItemTypeStory, Status: "backlog", Priority: domain.PriorityMedium, ParentID: "e1", BlockedBy: []string{"s1"}},
		"s3": {ID: "s3", DisplayNum: 4, Title: "Step 3", Type: domain.ItemTypeStory, Status: "backlog", Priority: domain.PriorityMedium, ParentID: "e1", BlockedBy: []string{"s2"}},
	})

	m := newDAGModel(board, 40, 20) // narrow viewport to force scrolling

	m.scrollRight()
	if m.scrollX <= 0 {
		t.Errorf("scrollRight should increase scrollX (graph width=%d, viewport=%d), got scrollX=%d", m.graph.width, m.width, m.scrollX)
	}

	savedX := m.scrollX
	m.scrollLeft()
	if m.scrollX >= savedX {
		t.Error("scrollLeft should decrease scrollX")
	}

	// Scroll left past 0 should clamp
	for i := 0; i < 20; i++ {
		m.scrollLeft()
	}
	if m.scrollX < 0 {
		t.Errorf("scrollX should not go negative, got %d", m.scrollX)
	}
}

func TestDAGModel_SelectedItem(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"t1": {ID: "t1", DisplayNum: 1, Title: "Task", Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium},
	})

	m := newDAGModel(board, 120, 40)
	m.cursorNode = 0

	item := m.SelectedItem()
	if item == nil {
		t.Fatal("expected selected item")
	}
	if item.ID != "t1" {
		t.Errorf("expected item t1, got %s", item.ID)
	}

	// Out of range
	m.cursorNode = -1
	if m.SelectedItem() != nil {
		t.Error("expected nil for out-of-range cursor")
	}
}

func TestDAGModel_View(t *testing.T) {
	board := makeDomainBoard(map[string]*domain.Item{
		"e1": {ID: "e1", DisplayNum: 1, Title: "My Epic", Type: domain.ItemTypeEpic, Status: "in-progress", Priority: domain.PriorityHigh},
	})
	board.Name = "test-board"

	m := newDAGModel(board, 120, 40)
	view := m.View()

	if !strings.Contains(view, "DAG View") {
		t.Error("expected 'DAG View' in header")
	}
	if !strings.Contains(view, "test-board") {
		t.Error("expected board name in header")
	}
	if !strings.Contains(view, "h/l:scroll") {
		t.Error("expected help bar")
	}
}

func TestHorizontalScroll(t *testing.T) {
	content := "0123456789\nabcdefghij"

	scrolled := horizontalScroll(content, 3, 5)
	lines := strings.Split(scrolled, "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "34567" {
		t.Errorf("expected '34567', got %q", lines[0])
	}
	if lines[1] != "defgh" {
		t.Errorf("expected 'defgh', got %q", lines[1])
	}
}

func TestHorizontalScroll_BeyondContent(t *testing.T) {
	content := "abc\ndef"
	scrolled := horizontalScroll(content, 10, 5)
	lines := strings.Split(scrolled, "\n")

	for _, line := range lines {
		if line != "" {
			t.Errorf("expected empty line for scroll past content, got %q", line)
		}
	}
}

// ============================================================
// INTEGRATION TESTS — teatest E2E through App
// ============================================================

// startDAGAndWait starts the app, waits for board load, switches to DAG view.
func startDAGAndWait(t *testing.T, eng *engine.Engine, boardFile string, width, height int) *teatest.TestModel {
	t.Helper()
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(width, height))

	// Wait for board to load
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG")) ||
			bytes.Contains(bts, []byte("Loading board"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	// Press G to enter DAG view
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})

	// Wait for DAG view to render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("DAG View"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	return tm
}

// getDAGScreen quits and captures the final rendered screen.
func getDAGScreen(t *testing.T, tm *teatest.TestModel) string {
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	model := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	return model.(App).View()
}

func TestTUI_DAG_EnterAndExit(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG VIEW (120x40) ===\n%s", screen)

	if !strings.Contains(screen, "DAG View") {
		t.Error("expected 'DAG View' header")
	}
	if !strings.Contains(screen, "dag-test-board") {
		t.Error("expected board name 'dag-test-board'")
	}
}

func TestTUI_DAG_ShowsLanes(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG LANES ===\n%s", screen)

	// Should show the epic lane
	if !strings.Contains(screen, "Auth System") {
		t.Error("expected 'Auth System' epic lane")
	}
}

func TestTUI_DAG_ShowsNodes(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	screen := getDAGScreen(t, tm)

	// Check that node display numbers appear
	if !strings.Contains(screen, "#1") {
		t.Error("expected node #1 in DAG")
	}
}

func TestTUI_DAG_HelpBar(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	screen := getDAGScreen(t, tm)

	if !strings.Contains(screen, "h/l:scroll") {
		t.Error("expected help bar with scroll instructions")
	}
	if !strings.Contains(screen, "G:board") {
		t.Error("expected 'G:board' in help bar")
	}
}

func TestTUI_DAG_NodeNavigation(t *testing.T) {
	boardFile, eng := testDAGBoard(t)

	t.Run("j_moves_to_next_node", func(t *testing.T) {
		tm := startDAGAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(100 * time.Millisecond)

		screen := getDAGScreen(t, tm)
		t.Logf("\n=== DAG AFTER 'j' ===\n%s", screen)
		// Just verify it didn't crash and still shows DAG view
		if !strings.Contains(screen, "DAG View") {
			t.Error("expected DAG view after j navigation")
		}
	})

	t.Run("k_moves_to_prev_node", func(t *testing.T) {
		tm := startDAGAndWait(t, eng, boardFile, 120, 40)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(100 * time.Millisecond)

		screen := getDAGScreen(t, tm)
		if !strings.Contains(screen, "DAG View") {
			t.Error("expected DAG view after k navigation")
		}
	})
}

func TestTUI_DAG_HorizontalScroll(t *testing.T) {
	boardFile, eng := testDAGBoard(t)

	t.Run("l_scrolls_right", func(t *testing.T) {
		tm := startDAGAndWait(t, eng, boardFile, 80, 30)

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(100 * time.Millisecond)

		screen := getDAGScreen(t, tm)
		t.Logf("\n=== DAG AFTER scroll right ===\n%s", screen)
		if !strings.Contains(screen, "DAG View") {
			t.Error("expected DAG view after scroll")
		}
	})

	t.Run("h_scrolls_left", func(t *testing.T) {
		tm := startDAGAndWait(t, eng, boardFile, 80, 30)

		// First scroll right, then left
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
		time.Sleep(100 * time.Millisecond)

		screen := getDAGScreen(t, tm)
		if !strings.Contains(screen, "DAG View") {
			t.Error("expected DAG view after scroll left")
		}
	})
}

func TestTUI_DAG_ReturnToBoard(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Press G to go back to board
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== AFTER G (back to board) ===\n%s", screen)

	// Should be back in board view
	if !strings.Contains(screen, "BACKLOG") {
		t.Error("expected board view with BACKLOG column after pressing G")
	}
}

func TestTUI_DAG_EscReturnToBoard(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)

	if !strings.Contains(screen, "BACKLOG") {
		t.Error("expected board view after Esc from DAG")
	}
}

func TestTUI_DAG_EnterOpensDetail(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Press Enter to open detail view
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG ENTER -> DETAIL ===\n%s", screen)

	// Detail view shows "Fields" tab
	if !strings.Contains(screen, "Fields") && !strings.Contains(screen, "Type") {
		t.Error("expected detail view after Enter from DAG")
	}
}

func TestTUI_DAG_FocusInProgress(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Press 0 to focus on in-progress
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG AFTER '0' (focus in-progress) ===\n%s", screen)

	if !strings.Contains(screen, "DAG View") {
		t.Error("expected DAG view after focus")
	}
}

func TestTUI_DAG_BoardHelpShowsDAGKeybinding(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	app := NewApp(eng, boardFile)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("BACKLOG"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)

	screen := getScreen(t, tm)
	if !strings.Contains(screen, "G:dag") {
		t.Error("expected 'G:dag' in board help bar")
	}
}

func TestTUI_DAG_MovePicker(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Press m to open move picker
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	time.Sleep(100 * time.Millisecond)

	// Should show picker overlay — press Esc to cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	// Should be back in DAG view after Esc from picker
	if !strings.Contains(screen, "DAG View") {
		t.Error("expected DAG view after picker cancel")
	}
}

func TestTUI_DAG_NarrowTerminal(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 80, 24)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG NARROW (80x24) ===\n%s", screen)

	if !strings.Contains(screen, "DAG View") {
		t.Error("expected DAG view in narrow terminal")
	}
}

func TestTUI_DAG_WideTerminal(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 200, 50)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG WIDE (200x50) ===\n%s", screen)

	if !strings.Contains(screen, "DAG View") {
		t.Error("expected DAG view in wide terminal")
	}
}

func TestTUI_DAG_Resize(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Resize to smaller
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	if !strings.Contains(screen, "DAG View") {
		t.Error("expected DAG view after resize")
	}
}

func TestTUI_DAG_DashboardFromDAG(t *testing.T) {
	boardFile, eng := testDAGBoard(t)
	tm := startDAGAndWait(t, eng, boardFile, 120, 40)

	// Press D to go to dashboard from DAG
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	time.Sleep(100 * time.Millisecond)

	screen := getDAGScreen(t, tm)
	t.Logf("\n=== DAG -> DASHBOARD ===\n%s", screen)

	if !strings.Contains(screen, "Dashboard") {
		t.Error("expected Dashboard view after pressing D from DAG")
	}
}

// ============================================================
// Helpers
// ============================================================

func findNodeByID(g dagGraph, id string) *dagNode {
	for i := range g.nodes {
		if g.nodes[i].id == id {
			return &g.nodes[i]
		}
	}
	return nil
}
