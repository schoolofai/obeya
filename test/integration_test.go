package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func setupTestEngine(t *testing.T) *engine.Engine {
	t.Helper()
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)
	if err := s.InitBoard("integration-test", nil); err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}
	eng := engine.New(s)
	_ = eng.AddUser("testuser", "human", "local")
	return eng
}

func TestIntegration_FullFlow(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)

	// --- Init board ---
	if err := s.InitBoard("e2e-board", nil); err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}
	eng := engine.New(s)

	board, err := eng.ListBoard()
	if err != nil {
		t.Fatalf("ListBoard failed: %v", err)
	}
	if board.Name != "e2e-board" {
		t.Errorf("expected board name 'e2e-board', got %q", board.Name)
	}
	if len(board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(board.Columns))
	}

	// --- Add users ---
	if err := eng.AddUser("alice", "human", ""); err != nil {
		t.Fatalf("AddUser (human) failed: %v", err)
	}
	if err := eng.AddUser("bot-1", "agent", "claude"); err != nil {
		t.Fatalf("AddUser (agent) failed: %v", err)
	}

	board, _ = eng.ListBoard()
	if len(board.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(board.Users))
	}

	// --- Create hierarchy: epic -> story -> task (with tags) ---
	epic, err := eng.CreateItem("epic", "Platform Build", "", "Build the platform", "high", "alice", nil)
	if err != nil {
		t.Fatalf("CreateItem (epic) failed: %v", err)
	}
	if epic.DisplayNum != 1 {
		t.Errorf("expected epic display num 1, got %d", epic.DisplayNum)
	}

	story, err := eng.CreateItem("story", "Auth Module", fmt.Sprintf("%d", epic.DisplayNum), "Implement auth", "medium", "alice", []string{"auth", "backend"})
	if err != nil {
		t.Fatalf("CreateItem (story) failed: %v", err)
	}
	if story.DisplayNum != 2 {
		t.Errorf("expected story display num 2, got %d", story.DisplayNum)
	}
	if story.ParentID != epic.ID {
		t.Errorf("expected story parent to be epic ID")
	}

	task1, err := eng.CreateItem("task", "Login endpoint", fmt.Sprintf("%d", story.DisplayNum), "", "high", "alice", []string{"auth", "api"})
	if err != nil {
		t.Fatalf("CreateItem (task1) failed: %v", err)
	}
	if task1.DisplayNum != 3 {
		t.Errorf("expected task1 display num 3, got %d", task1.DisplayNum)
	}

	task2, err := eng.CreateItem("task", "JWT validation", fmt.Sprintf("%d", story.DisplayNum), "", "medium", "alice", []string{"auth"})
	if err != nil {
		t.Fatalf("CreateItem (task2) failed: %v", err)
	}
	if task2.DisplayNum != 4 {
		t.Errorf("expected task2 display num 4, got %d", task2.DisplayNum)
	}

	// --- Verify display numbers are sequential ---
	board, _ = eng.ListBoard()
	for i := 1; i <= 4; i++ {
		if _, ok := board.DisplayMap[i]; !ok {
			t.Errorf("display number %d missing from display map", i)
		}
	}

	// --- Move items between columns ---
	if err := eng.MoveItem(fmt.Sprintf("%d", task1.DisplayNum), "in-progress", "", "test-session"); err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	moved, err := eng.GetItem(fmt.Sprintf("%d", task1.DisplayNum))
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if moved.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", moved.Status)
	}

	// Move to invalid column should fail
	if err := eng.MoveItem(fmt.Sprintf("%d", task1.DisplayNum), "nonexistent", "", ""); err == nil {
		t.Error("expected error moving to nonexistent column")
	}

	// --- Block / unblock ---
	if err := eng.BlockItem(fmt.Sprintf("%d", task2.DisplayNum), fmt.Sprintf("%d", task1.DisplayNum), "", "test-session"); err != nil {
		t.Fatalf("BlockItem failed: %v", err)
	}

	blocked, _ := eng.GetItem(fmt.Sprintf("%d", task2.DisplayNum))
	if len(blocked.BlockedBy) != 1 {
		t.Fatalf("expected 1 blocker, got %d", len(blocked.BlockedBy))
	}
	if blocked.BlockedBy[0] != task1.ID {
		t.Errorf("expected blocker to be task1 ID")
	}

	// Duplicate block should fail
	if err := eng.BlockItem(fmt.Sprintf("%d", task2.DisplayNum), fmt.Sprintf("%d", task1.DisplayNum), "", ""); err == nil {
		t.Error("expected error on duplicate block")
	}

	// Self-block should fail
	if err := eng.BlockItem(fmt.Sprintf("%d", task1.DisplayNum), fmt.Sprintf("%d", task1.DisplayNum), "", ""); err == nil {
		t.Error("expected error on self-block")
	}

	// Unblock
	if err := eng.UnblockItem(fmt.Sprintf("%d", task2.DisplayNum), fmt.Sprintf("%d", task1.DisplayNum), "", "test-session"); err != nil {
		t.Fatalf("UnblockItem failed: %v", err)
	}

	unblocked, _ := eng.GetItem(fmt.Sprintf("%d", task2.DisplayNum))
	if len(unblocked.BlockedBy) != 0 {
		t.Errorf("expected 0 blockers after unblock, got %d", len(unblocked.BlockedBy))
	}

	// --- List with filters ---
	// Filter by status
	inProgress, err := eng.ListItems(engine.ListFilter{Status: "in-progress"})
	if err != nil {
		t.Fatalf("ListItems (status filter) failed: %v", err)
	}
	if len(inProgress) != 1 {
		t.Errorf("expected 1 in-progress item, got %d", len(inProgress))
	}

	// Filter by type
	epics, err := eng.ListItems(engine.ListFilter{Type: "epic"})
	if err != nil {
		t.Fatalf("ListItems (type filter) failed: %v", err)
	}
	if len(epics) != 1 {
		t.Errorf("expected 1 epic, got %d", len(epics))
	}

	tasks, err := eng.ListItems(engine.ListFilter{Type: "task"})
	if err != nil {
		t.Fatalf("ListItems (type=task filter) failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}

	// Filter by tag
	authTagged, err := eng.ListItems(engine.ListFilter{Tag: "auth"})
	if err != nil {
		t.Fatalf("ListItems (tag filter) failed: %v", err)
	}
	if len(authTagged) != 3 {
		t.Errorf("expected 3 items with 'auth' tag, got %d", len(authTagged))
	}

	// Re-block task2 for blocked filter test
	if err := eng.BlockItem(fmt.Sprintf("%d", task2.DisplayNum), fmt.Sprintf("%d", task1.DisplayNum), "", ""); err != nil {
		t.Fatalf("re-block failed: %v", err)
	}
	blockedItems, err := eng.ListItems(engine.ListFilter{Blocked: true})
	if err != nil {
		t.Fatalf("ListItems (blocked filter) failed: %v", err)
	}
	if len(blockedItems) != 1 {
		t.Errorf("expected 1 blocked item, got %d", len(blockedItems))
	}

	// --- Verify history entries ---
	item1, _ := eng.GetItem(fmt.Sprintf("%d", task1.DisplayNum))
	if len(item1.History) < 2 {
		t.Errorf("expected at least 2 history entries for task1 (created + moved), got %d", len(item1.History))
	}
	if item1.History[0].Action != "created" {
		t.Errorf("expected first history action 'created', got %q", item1.History[0].Action)
	}
	if item1.History[1].Action != "moved" {
		t.Errorf("expected second history action 'moved', got %q", item1.History[1].Action)
	}

	// --- Delete with children should fail ---
	if err := eng.DeleteItem(fmt.Sprintf("%d", epic.DisplayNum), "", "test-session"); err == nil {
		t.Error("expected error deleting epic with children")
	}

	if err := eng.DeleteItem(fmt.Sprintf("%d", story.DisplayNum), "", "test-session"); err == nil {
		t.Error("expected error deleting story with children")
	}

	// --- Delete leaf item should succeed ---
	if err := eng.DeleteItem(fmt.Sprintf("%d", task2.DisplayNum), "", "test-session"); err != nil {
		t.Fatalf("DeleteItem (leaf) failed: %v", err)
	}

	// Verify deleted
	if _, err := eng.GetItem(fmt.Sprintf("%d", task2.DisplayNum)); err == nil {
		t.Error("expected error getting deleted item")
	}

	// Verify remaining items
	board, _ = eng.ListBoard()
	if len(board.Items) != 3 {
		t.Errorf("expected 3 remaining items, got %d", len(board.Items))
	}
}

func TestIntegration_AssignAndEdit(t *testing.T) {
	eng := setupTestEngine(t)

	// Add a user
	if err := eng.AddUser("dev1", "human", ""); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	task, err := eng.CreateItem("task", "Test task", "", "", "low", "dev1", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	// Assign by user name
	if err := eng.AssignItem(fmt.Sprintf("%d", task.DisplayNum), "dev1", "", "session"); err != nil {
		t.Fatalf("AssignItem failed: %v", err)
	}

	assigned, _ := eng.GetItem(fmt.Sprintf("%d", task.DisplayNum))
	if assigned.Assignee == "" {
		t.Error("expected assignee to be set")
	}

	// Edit title and priority
	if err := eng.EditItem(fmt.Sprintf("%d", task.DisplayNum), "Updated title", "", "critical", "", "session"); err != nil {
		t.Fatalf("EditItem failed: %v", err)
	}

	edited, _ := eng.GetItem(fmt.Sprintf("%d", task.DisplayNum))
	if edited.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got %q", edited.Title)
	}
	if string(edited.Priority) != "critical" {
		t.Errorf("expected priority 'critical', got %q", edited.Priority)
	}

	// Edit with no changes should fail
	if err := eng.EditItem(fmt.Sprintf("%d", task.DisplayNum), "", "", "", "", ""); err == nil {
		t.Error("expected error when no changes specified")
	}
}

func TestIntegration_BoardConfig(t *testing.T) {
	eng := setupTestEngine(t)

	// Add a custom column
	if err := eng.AddColumn("qa"); err != nil {
		t.Fatalf("AddColumn failed: %v", err)
	}

	board, _ := eng.ListBoard()
	if !board.HasColumn("qa") {
		t.Error("expected board to have 'qa' column")
	}

	// Duplicate column should fail
	if err := eng.AddColumn("qa"); err == nil {
		t.Error("expected error adding duplicate column")
	}

	// Remove column
	if err := eng.RemoveColumn("qa"); err != nil {
		t.Fatalf("RemoveColumn failed: %v", err)
	}

	board, _ = eng.ListBoard()
	if board.HasColumn("qa") {
		t.Error("expected 'qa' column to be removed")
	}

	// Remove column with items should fail
	eng.CreateItem("task", "blocker", "", "", "", "testuser", nil)
	eng.MoveItem("1", "todo", "", "")
	if err := eng.RemoveColumn("todo"); err == nil {
		t.Error("expected error removing column with items")
	}
}

func TestIntegration_UserManagement(t *testing.T) {
	eng := setupTestEngine(t)

	if err := eng.AddUser("alice", "human", ""); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if err := eng.AddUser("claude", "agent", "anthropic"); err != nil {
		t.Fatalf("AddUser (agent) failed: %v", err)
	}

	// Invalid identity type
	if err := eng.AddUser("bad", "robot", ""); err == nil {
		t.Error("expected error for invalid identity type")
	}

	board, _ := eng.ListBoard()
	if len(board.Users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(board.Users))
	}

	// Remove user by name
	if err := eng.RemoveUser("alice"); err != nil {
		t.Fatalf("RemoveUser failed: %v", err)
	}

	board, _ = eng.ListBoard()
	if len(board.Users) != 2 {
		t.Errorf("expected 2 users after removal, got %d", len(board.Users))
	}
}

func TestIntegration_GetChildren(t *testing.T) {
	eng := setupTestEngine(t)

	parent, err := eng.CreateItem("epic", "Parent", "", "", "", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem (parent) failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := eng.CreateItem("story", fmt.Sprintf("Child %d", i), fmt.Sprintf("%d", parent.DisplayNum), "", "", "testuser", nil)
		if err != nil {
			t.Fatalf("CreateItem (child %d) failed: %v", i, err)
		}
	}

	children, err := eng.GetChildren(parent.ID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 3 {
		t.Errorf("expected 3 children, got %d", len(children))
	}
}

func TestIntegration_InitBoardAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)
	if err := s.InitBoard("first", nil); err != nil {
		t.Fatalf("first InitBoard failed: %v", err)
	}

	// Second init should fail
	if err := s.InitBoard("second", nil); err == nil {
		t.Error("expected error on duplicate init")
	}
}

func TestIntegration_CustomColumns(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)
	if err := s.InitBoard("custom", []string{"new", "active", "closed"}); err != nil {
		t.Fatalf("InitBoard (custom cols) failed: %v", err)
	}

	eng := engine.New(s)
	_ = eng.AddUser("testuser", "human", "local")
	board, _ := eng.ListBoard()
	if len(board.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(board.Columns))
	}

	// Items should start in first custom column
	item, err := eng.CreateItem("task", "Test", "", "", "", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
	if item.Status != "new" {
		t.Errorf("expected initial status 'new', got %q", item.Status)
	}
}

func TestIntegration_PlanWorkflow(t *testing.T) {
	eng := setupTestEngine(t)

	// Create items
	epic, err := eng.CreateItem("epic", "Auth System", "", "", "high", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem epic failed: %v", err)
	}
	task, err := eng.CreateItem("task", "JWT Validation", fmt.Sprintf("%d", epic.DisplayNum), "", "medium", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem task failed: %v", err)
	}

	// Import plan
	content := "# Auth Implementation Plan\n\nBuild JWT auth with middleware."
	plan, err := eng.ImportPlan(content, "docs/plans/auth-plan.md", []string{
		fmt.Sprintf("%d", epic.DisplayNum),
		fmt.Sprintf("%d", task.DisplayNum),
	})
	if err != nil {
		t.Fatalf("ImportPlan failed: %v", err)
	}
	if plan.Title != "Auth Implementation Plan" {
		t.Errorf("expected title from markdown heading, got %q", plan.Title)
	}
	if len(plan.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(plan.LinkedItems))
	}

	// Show plan
	shown, err := eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if err != nil {
		t.Fatalf("ShowPlan failed: %v", err)
	}
	if shown.Content != content {
		t.Error("plan content mismatch")
	}

	// List plans
	plans, err := eng.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}

	// Link additional item
	task2, _ := eng.CreateItem("task", "Middleware", fmt.Sprintf("%d", epic.DisplayNum), "", "medium", "testuser", nil)
	if err := eng.LinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", task2.DisplayNum)}); err != nil {
		t.Fatalf("LinkPlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(shown.LinkedItems) != 3 {
		t.Errorf("expected 3 linked items after link, got %d", len(shown.LinkedItems))
	}

	// Unlink
	if err := eng.UnlinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", task.DisplayNum)}); err != nil {
		t.Fatalf("UnlinkPlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(shown.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items after unlink, got %d", len(shown.LinkedItems))
	}

	// PlansForItem
	itemPlans, err := eng.PlansForItem(epic.ID)
	if err != nil {
		t.Fatalf("PlansForItem failed: %v", err)
	}
	if len(itemPlans) != 1 {
		t.Errorf("expected 1 plan for epic, got %d", len(itemPlans))
	}

	// Update plan
	if err := eng.UpdatePlan(fmt.Sprintf("%d", plan.DisplayNum), "Updated Auth Plan", "# Updated\n\nNew content."); err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if shown.Title != "Updated Auth Plan" {
		t.Errorf("expected updated title, got %q", shown.Title)
	}

	// Delete plan
	if err := eng.DeletePlan(fmt.Sprintf("%d", plan.DisplayNum)); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}
	_, err = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if err == nil {
		t.Error("expected error showing deleted plan")
	}
}

func TestIntegration_CLISmokeTest(t *testing.T) {
	// Verify the binary exists (built by Task 13)
	if _, err := os.Stat("../ob"); err != nil {
		t.Skip("ob binary not found — skipping CLI smoke test (run 'go build -o ob .' first)")
	}
}

// TestIntegration_AssignThenMoveFlow tests the full mandatory assignment lifecycle:
// unassigned item → move blocked → assign → move succeeds
func TestIntegration_AssignThenMoveFlow(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)
	if err := s.InitBoard("assign-flow", nil); err != nil {
		t.Fatalf("InitBoard failed: %v", err)
	}
	eng := engine.New(s)

	// Register a user
	if err := eng.AddUser("agent-1", "agent", "claude-code"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	// Create an assigned item
	task, err := eng.CreateItem("task", "Normal task", "", "has owner", "medium", "agent-1", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	// Move should succeed on assigned item
	if err := eng.MoveItem(fmt.Sprintf("%d", task.DisplayNum), "in-progress", "", ""); err != nil {
		t.Fatalf("MoveItem on assigned item should succeed: %v", err)
	}

	// Create a legacy unassigned item directly via store (simulates pre-migration data)
	var unassignedNum int
	err = s.Transaction(func(board *domain.Board) error {
		item := &domain.Item{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Type:        "task",
			Title:       "Legacy unassigned",
			Description: "no owner",
			Status:      board.Columns[0].Name,
			Priority:    "medium",
			Assignee:    "",
		}
		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++
		unassignedNum = item.DisplayNum
		return nil
	})
	if err != nil {
		t.Fatalf("create unassigned item failed: %v", err)
	}

	// Move should FAIL on unassigned item
	err = eng.MoveItem(fmt.Sprintf("%d", unassignedNum), "in-progress", "", "")
	if err == nil {
		t.Fatal("MoveItem on unassigned item should fail")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' error, got: %v", err)
	}

	// Edit should FAIL on unassigned item
	err = eng.EditItem(fmt.Sprintf("%d", unassignedNum), "new title", "", "", "", "")
	if err == nil {
		t.Fatal("EditItem on unassigned item should fail")
	}

	// Delete should FAIL on unassigned item
	err = eng.DeleteItem(fmt.Sprintf("%d", unassignedNum), "", "")
	if err == nil {
		t.Fatal("DeleteItem on unassigned item should fail")
	}

	// Assign should SUCCEED (this is the fix path)
	err = eng.AssignItem(fmt.Sprintf("%d", unassignedNum), "agent-1", "", "")
	if err != nil {
		t.Fatalf("AssignItem on unassigned item should succeed: %v", err)
	}

	// Now move should SUCCEED
	err = eng.MoveItem(fmt.Sprintf("%d", unassignedNum), "in-progress", "", "")
	if err != nil {
		t.Fatalf("MoveItem after assign should succeed: %v", err)
	}

	// Verify final state
	item, _ := eng.GetItem(fmt.Sprintf("%d", unassignedNum))
	if item.Assignee == "" {
		t.Error("expected assignee to be set after assign")
	}
	if item.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", item.Status)
	}
}

// TestIntegration_CreateRejectsEmptyAssignee verifies the engine itself
// rejects items with no assignee (not just the CLI layer).
func TestIntegration_CreateRejectsEmptyAssignee(t *testing.T) {
	eng := setupTestEngine(t)

	_, err := eng.CreateItem("task", "No owner", "", "desc", "medium", "", nil)
	if err == nil {
		t.Fatal("CreateItem with empty assignee should fail")
	}
	if !strings.Contains(err.Error(), "assignee is required") {
		t.Errorf("expected 'assignee is required', got: %v", err)
	}
}

// TestIntegration_AllGuardsBlockUnassigned verifies every guarded operation
// fails on unassigned items with the correct error.
func TestIntegration_AllGuardsBlockUnassigned(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewJSONStore(tmpDir)
	_ = s.InitBoard("guard-test", nil)
	eng := engine.New(s)
	_ = eng.AddUser("user1", "human", "local")

	// Create a blocker item (assigned)
	blocker, _ := eng.CreateItem("task", "Blocker", "", "desc", "medium", "user1", nil)

	// Create unassigned item via store
	var itemID string
	var itemNum int
	_ = s.Transaction(func(board *domain.Board) error {
		item := &domain.Item{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Type:        "task",
			Title:       "Unassigned",
			Description: "no owner",
			Status:      board.Columns[0].Name,
			Priority:    "medium",
		}
		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++
		itemID = item.ID
		itemNum = item.DisplayNum
		return nil
	})

	ref := fmt.Sprintf("%d", itemNum)
	blockerRef := fmt.Sprintf("%d", blocker.DisplayNum)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"MoveItem", func() error { return eng.MoveItem(ref, "in-progress", "", "") }},
		{"EditItem", func() error { return eng.EditItem(ref, "x", "", "", "", "") }},
		{"BlockItem", func() error { return eng.BlockItem(ref, blockerRef, "", "") }},
		{"UnblockItem", func() error { return eng.UnblockItem(ref, blockerRef, "", "") }},
		{"DeleteItem", func() error { return eng.DeleteItem(ref, "", "") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Fatalf("%s should fail on unassigned item", tc.name)
			}
			if !strings.Contains(err.Error(), "no assignee") {
				t.Errorf("expected 'no assignee' error from %s, got: %v", tc.name, err)
			}
		})
	}

	// AssignItem should NOT be guarded
	err := eng.AssignItem(ref, "user1", "", "")
	if err != nil {
		t.Fatalf("AssignItem should succeed on unassigned item: %v", err)
	}

	_ = itemID // used for ref
}
