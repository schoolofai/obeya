package engine_test

import (
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func setupEngine(t *testing.T) (*engine.Engine, store.Store) {
	t.Helper()
	dir := t.TempDir()
	s := store.NewJSONStore(dir)
	_ = s.InitBoard("test", nil)
	eng := engine.New(s)
	_, _ = eng.AddUser("testuser", "human", "local")
	return eng, s
}

func createUnassignedItem(t *testing.T, s store.Store, title string) *domain.Item {
	t.Helper()
	var created *domain.Item
	err := s.Transaction(func(board *domain.Board) error {
		item := &domain.Item{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Type:        "task",
			Title:       title,
			Description: "test",
			Status:      board.Columns[0].Name,
			Priority:    "medium",
			Assignee:    "",
		}
		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++
		created = item
		return nil
	})
	if err != nil {
		t.Fatalf("createUnassignedItem failed: %v", err)
	}
	return created
}

func TestCreateItem(t *testing.T) {
	eng, _ := setupEngine(t)

	item, err := eng.CreateItem("epic", "Build auth system", "", "", "medium", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if item.Type != "epic" {
		t.Errorf("expected type 'epic', got %q", item.Type)
	}
	if item.Title != "Build auth system" {
		t.Errorf("expected title 'Build auth system', got %q", item.Title)
	}
	if item.Status != "backlog" {
		t.Errorf("expected default status 'backlog', got %q", item.Status)
	}
	if item.DisplayNum != 1 {
		t.Errorf("expected display num 1, got %d", item.DisplayNum)
	}
}

func TestCreateItem_WithParent(t *testing.T) {
	eng, _ := setupEngine(t)

	epic, _ := eng.CreateItem("epic", "Epic", "", "", "medium", "testuser", nil)
	story, err := eng.CreateItem("story", "Story", epic.ID, "", "medium", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem with parent failed: %v", err)
	}
	if story.ParentID != epic.ID {
		t.Errorf("expected parent %q, got %q", epic.ID, story.ParentID)
	}
}

func TestCreateItem_InvalidParent(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.CreateItem("story", "Story", "nonexistent", "", "medium", "testuser", nil)
	if err == nil {
		t.Error("expected error for invalid parent")
	}
}

func TestCreateItem_InvalidType(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.CreateItem("bug", "Bug report", "", "", "medium", "testuser", nil)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestCreateItem_EmptyTitle(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.CreateItem("task", "", "", "", "medium", "testuser", nil)
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateItem_InvalidPriority(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.CreateItem("task", "Task", "", "", "urgent", "testuser", nil)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestCreateItem_DefaultPriority(t *testing.T) {
	eng, _ := setupEngine(t)

	item, err := eng.CreateItem("task", "Task", "", "", "", "testuser", nil)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
	if item.Priority != "medium" {
		t.Errorf("expected default priority 'medium', got %q", item.Priority)
	}
}

func TestMoveItem(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Fix bug", "", "", "medium", "testuser", nil)

	err := eng.MoveItem(item.ID, "in-progress", "", "")
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	updated, _ := eng.GetItem(item.ID)
	if updated.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %q", updated.Status)
	}
}

func TestMoveItem_InvalidStatus(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Fix bug", "", "", "medium", "testuser", nil)

	err := eng.MoveItem(item.ID, "invalid-column", "", "")
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestBlockItem(t *testing.T) {
	eng, _ := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "testuser", nil)

	err := eng.BlockItem(task2.ID, task1.ID, "", "")
	if err != nil {
		t.Fatalf("BlockItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task2.ID)
	if len(updated.BlockedBy) != 1 || updated.BlockedBy[0] != task1.ID {
		t.Errorf("expected BlockedBy [%s], got %v", task1.ID, updated.BlockedBy)
	}
}

func TestBlockItem_SelfBlock(t *testing.T) {
	eng, _ := setupEngine(t)

	task, _ := eng.CreateItem("task", "Task", "", "", "medium", "testuser", nil)

	err := eng.BlockItem(task.ID, task.ID, "", "")
	if err == nil {
		t.Error("expected error for self-blocking")
	}
}

func TestBlockItem_Duplicate(t *testing.T) {
	eng, _ := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "testuser", nil)

	_ = eng.BlockItem(task2.ID, task1.ID, "", "")
	err := eng.BlockItem(task2.ID, task1.ID, "", "")
	if err == nil {
		t.Error("expected error for duplicate block")
	}
}

func TestUnblockItem(t *testing.T) {
	eng, _ := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "testuser", nil)

	_ = eng.BlockItem(task2.ID, task1.ID, "", "")
	err := eng.UnblockItem(task2.ID, task1.ID, "", "")
	if err != nil {
		t.Fatalf("UnblockItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task2.ID)
	if len(updated.BlockedBy) != 0 {
		t.Errorf("expected empty BlockedBy, got %v", updated.BlockedBy)
	}
}

func TestUnblockItem_NotBlocked(t *testing.T) {
	eng, _ := setupEngine(t)

	task1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", nil)
	task2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "testuser", nil)

	err := eng.UnblockItem(task2.ID, task1.ID, "", "")
	if err == nil {
		t.Error("expected error when unblocking item that is not blocked")
	}
}

func TestAssignItem(t *testing.T) {
	eng, _ := setupEngine(t)

	task, _ := eng.CreateItem("task", "Task", "", "", "medium", "testuser", nil)
	_, _ = eng.AddUser("Dev", "human", "local")

	board, _ := eng.ListBoard()
	var userID string
	for id := range board.Users {
		userID = id
	}

	err := eng.AssignItem(task.ID, userID, "", "")
	if err != nil {
		t.Fatalf("AssignItem failed: %v", err)
	}

	updated, _ := eng.GetItem(task.ID)
	if updated.Assignee != userID {
		t.Errorf("expected assignee %q, got %q", userID, updated.Assignee)
	}
}

func TestEditItem(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Old Title", "", "", "medium", "testuser", nil)

	err := eng.EditItem(item.ID, "New Title", "New desc", "high", "", "")
	if err != nil {
		t.Fatalf("EditItem failed: %v", err)
	}

	updated, _ := eng.GetItem(item.ID)
	if updated.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", updated.Title)
	}
	if updated.Description != "New desc" {
		t.Errorf("expected description 'New desc', got %q", updated.Description)
	}
	if updated.Priority != "high" {
		t.Errorf("expected priority 'high', got %q", updated.Priority)
	}
}

func TestEditItem_NoChanges(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Title", "", "", "medium", "testuser", nil)

	err := eng.EditItem(item.ID, "", "", "", "", "")
	if err == nil {
		t.Error("expected error when no changes specified")
	}
}

func TestDeleteItem(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Task", "", "", "medium", "testuser", nil)

	err := eng.DeleteItem(item.ID, "", "")
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	_, err = eng.GetItem(item.ID)
	if err == nil {
		t.Error("expected error getting deleted item")
	}
}

func TestDeleteItem_WithChildren(t *testing.T) {
	eng, _ := setupEngine(t)

	epic, _ := eng.CreateItem("epic", "Epic", "", "", "medium", "testuser", nil)
	_, _ = eng.CreateItem("story", "Story", epic.ID, "", "medium", "testuser", nil)

	err := eng.DeleteItem(epic.ID, "", "")
	if err == nil {
		t.Error("expected error when deleting item with children")
	}
}

func TestAddUser(t *testing.T) {
	eng, _ := setupEngine(t)

	added, err := eng.AddUser("Claude Agent", "agent", "claude-code")
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if !added {
		t.Error("expected added=true for new user")
	}

	board, _ := eng.ListBoard()
	if len(board.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(board.Users))
	}

	found := false
	for _, u := range board.Users {
		if u.Name == "Claude Agent" {
			found = true
			if u.Provider != "claude-code" {
				t.Errorf("expected provider 'claude-code', got %q", u.Provider)
			}
		}
	}
	if !found {
		t.Error("expected to find user 'Claude Agent'")
	}
}

func TestAddUser_DuplicateIsIdempotent(t *testing.T) {
	eng, _ := setupEngine(t)

	added1, err := eng.AddUser("Claude", "agent", "claude-code")
	if err != nil {
		t.Fatalf("first AddUser failed: %v", err)
	}
	if !added1 {
		t.Error("expected added=true for first add")
	}

	// Exact duplicate
	added2, err := eng.AddUser("Claude", "agent", "claude-code")
	if err != nil {
		t.Fatalf("duplicate AddUser failed: %v", err)
	}
	if added2 {
		t.Error("expected added=false for duplicate")
	}

	// Case-insensitive duplicate
	added3, err := eng.AddUser("claude", "agent", "claude-code")
	if err != nil {
		t.Fatalf("case-insensitive AddUser failed: %v", err)
	}
	if added3 {
		t.Error("expected added=false for case-insensitive duplicate")
	}

	board, _ := eng.ListBoard()
	count := 0
	for _, u := range board.Users {
		if strings.EqualFold(u.Name, "claude") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 'Claude' user, got %d", count)
	}
}

func TestAddUser_InvalidType(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.AddUser("Bot", "robot", "local")
	if err == nil {
		t.Error("expected error for invalid identity type")
	}
}

func TestRemoveUser(t *testing.T) {
	eng, _ := setupEngine(t)

	_, _ = eng.AddUser("Dev", "human", "local")

	board, _ := eng.ListBoard()
	var devID string
	for id, u := range board.Users {
		if u.Name == "Dev" {
			devID = id
		}
	}

	err := eng.RemoveUser(devID)
	if err != nil {
		t.Fatalf("RemoveUser failed: %v", err)
	}

	board, _ = eng.ListBoard()
	if len(board.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(board.Users))
	}
}

func TestListItems_WithFilter(t *testing.T) {
	eng, _ := setupEngine(t)

	_, _ = eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", []string{"frontend"})
	item2, _ := eng.CreateItem("task", "Task 2", "", "", "high", "testuser", []string{"backend"})
	_ = eng.MoveItem(item2.ID, "in-progress", "", "")

	items, err := eng.ListItems(engine.ListFilter{Status: "in-progress"})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	items, err = eng.ListItems(engine.ListFilter{Tag: "frontend"})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item with tag 'frontend', got %d", len(items))
	}
}

func TestCreatePlan(t *testing.T) {
	eng, _ := setupEngine(t)

	plan, err := eng.CreatePlan("Design Doc", "Some content", "design.md")
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if plan.Title != "Design Doc" {
		t.Errorf("expected title 'Design Doc', got %q", plan.Title)
	}
	if plan.DisplayNum != 1 {
		t.Errorf("expected display num 1, got %d", plan.DisplayNum)
	}
	if plan.Content != "Some content" {
		t.Errorf("expected content 'Some content', got %q", plan.Content)
	}
	if plan.SourceFile != "design.md" {
		t.Errorf("expected source file 'design.md', got %q", plan.SourceFile)
	}
}

func TestCreatePlan_EmptyTitle(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.CreatePlan("", "content", "")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestImportPlan(t *testing.T) {
	eng, _ := setupEngine(t)

	item1, _ := eng.CreateItem("task", "Task 1", "", "", "medium", "testuser", nil)
	item2, _ := eng.CreateItem("task", "Task 2", "", "", "medium", "testuser", nil)

	content := "# My Plan\n\nThis is the plan content."
	plan, err := eng.ImportPlan(content, "plan.md", []string{item1.ID, item2.ID})
	if err != nil {
		t.Fatalf("ImportPlan failed: %v", err)
	}

	if plan.Title != "My Plan" {
		t.Errorf("expected title 'My Plan', got %q", plan.Title)
	}
	if len(plan.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(plan.LinkedItems))
	}
}

func TestImportPlan_NoHeading(t *testing.T) {
	eng, _ := setupEngine(t)

	_, err := eng.ImportPlan("no heading here", "", nil)
	if err == nil {
		t.Error("expected error when no markdown heading found")
	}
}

func TestLinkUnlinkPlan(t *testing.T) {
	eng, _ := setupEngine(t)

	plan, _ := eng.CreatePlan("Plan", "content", "")
	item, _ := eng.CreateItem("task", "Task", "", "", "medium", "testuser", nil)

	err := eng.LinkPlan(plan.ID, []string{item.ID})
	if err != nil {
		t.Fatalf("LinkPlan failed: %v", err)
	}

	updated, _ := eng.ShowPlan(plan.ID)
	if len(updated.LinkedItems) != 1 {
		t.Errorf("expected 1 linked item, got %d", len(updated.LinkedItems))
	}

	err = eng.UnlinkPlan(plan.ID, []string{item.ID})
	if err != nil {
		t.Fatalf("UnlinkPlan failed: %v", err)
	}

	updated, _ = eng.ShowPlan(plan.ID)
	if len(updated.LinkedItems) != 0 {
		t.Errorf("expected 0 linked items, got %d", len(updated.LinkedItems))
	}
}

func TestDeletePlan(t *testing.T) {
	eng, _ := setupEngine(t)

	plan, _ := eng.CreatePlan("Plan", "content", "")

	err := eng.DeletePlan(plan.ID)
	if err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}

	_, err = eng.ShowPlan(plan.ID)
	if err == nil {
		t.Error("expected error showing deleted plan")
	}
}

func TestUpdatePlan(t *testing.T) {
	eng, _ := setupEngine(t)

	plan, _ := eng.CreatePlan("Old Title", "old content", "")

	err := eng.UpdatePlan(plan.ID, "New Title", "new content")
	if err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}

	updated, _ := eng.ShowPlan(plan.ID)
	if updated.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", updated.Title)
	}
	if updated.Content != "new content" {
		t.Errorf("expected content 'new content', got %q", updated.Content)
	}
}

func TestUpdatePlan_NoChanges(t *testing.T) {
	eng, _ := setupEngine(t)

	plan, _ := eng.CreatePlan("Title", "content", "")

	err := eng.UpdatePlan(plan.ID, "", "")
	if err == nil {
		t.Error("expected error when no changes specified")
	}
}

func TestListPlans(t *testing.T) {
	eng, _ := setupEngine(t)

	_, _ = eng.CreatePlan("Plan A", "content a", "")
	_, _ = eng.CreatePlan("Plan B", "content b", "")

	plans, err := eng.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestPlansForItem(t *testing.T) {
	eng, _ := setupEngine(t)

	item, _ := eng.CreateItem("task", "Task", "", "", "medium", "testuser", nil)
	plan, _ := eng.CreatePlan("Plan", "content", "")
	_ = eng.LinkPlan(plan.ID, []string{item.ID})

	plans, err := eng.PlansForItem(item.ID)
	if err != nil {
		t.Fatalf("PlansForItem failed: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}
}

func TestGetChildren(t *testing.T) {
	eng, _ := setupEngine(t)

	epic, _ := eng.CreateItem("epic", "Epic", "", "", "medium", "testuser", nil)
	_, _ = eng.CreateItem("story", "Story 1", epic.ID, "", "medium", "testuser", nil)
	_, _ = eng.CreateItem("story", "Story 2", epic.ID, "", "medium", "testuser", nil)

	children, err := eng.GetChildren(epic.ID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestCheckAssignee_Unassigned(t *testing.T) {
	item := &domain.Item{DisplayNum: 5, Assignee: ""}
	err := engine.CheckAssignee(item)
	if err == nil {
		t.Fatal("expected error for unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ob assign 5") {
		t.Errorf("expected fix instructions in error, got: %v", err)
	}
}

func TestCheckAssignee_Assigned(t *testing.T) {
	item := &domain.Item{DisplayNum: 5, Assignee: "user123"}
	err := engine.CheckAssignee(item)
	if err != nil {
		t.Fatalf("expected no error for assigned item, got: %v", err)
	}
}

func TestMoveItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Unassigned task")
	err := eng.MoveItem(item.ID, "in-progress", "", "")
	if err == nil {
		t.Fatal("expected error moving unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' error, got: %v", err)
	}
}

func TestEditItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Test")
	err := eng.EditItem(item.ID, "New title", "", "", "", "")
	if err == nil {
		t.Fatal("expected error editing unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' error, got: %v", err)
	}
}

func TestBlockItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item1, _ := eng.CreateItem("task", "Task1", "", "desc", "medium", "testuser", nil)
	item2 := createUnassignedItem(t, s, "Task2")
	err := eng.BlockItem(item2.ID, item1.ID, "", "")
	if err == nil {
		t.Fatal("expected error blocking unassigned item")
	}
}

func TestUnblockItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item1, _ := eng.CreateItem("task", "Task1", "", "desc", "medium", "testuser", nil)
	item2 := createUnassignedItem(t, s, "Task2")
	err := eng.UnblockItem(item2.ID, item1.ID, "", "")
	if err == nil {
		t.Fatal("expected error unblocking unassigned item")
	}
}

func TestDeleteItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Test")
	err := eng.DeleteItem(item.ID, "", "")
	if err == nil {
		t.Fatal("expected error deleting unassigned item")
	}
}

func TestCreateItem_EmptyAssigneeFails(t *testing.T) {
	eng, _ := setupEngine(t)
	_, err := eng.CreateItem("task", "Test", "", "desc", "medium", "", nil)
	if err == nil {
		t.Fatal("expected error for empty assignee")
	}
	if !strings.Contains(err.Error(), "assignee is required") {
		t.Errorf("expected 'assignee is required' error, got: %v", err)
	}
}

func TestCreateItem_UnknownAssigneeFails(t *testing.T) {
	eng, _ := setupEngine(t)
	_, err := eng.CreateItem("task", "Test", "", "desc", "medium", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown assignee")
	}
	if !strings.Contains(err.Error(), "ob user list") {
		t.Errorf("expected 'ob user list' in error, got: %v", err)
	}
}

func TestCreateItem_AssigneeResolved(t *testing.T) {
	eng, _ := setupEngine(t)
	item, err := eng.CreateItem("task", "Test", "", "desc", "medium", "testuser", nil)
	if err != nil {
		t.Fatalf("expected success with valid assignee, got: %v", err)
	}
	if item.Assignee == "" {
		t.Error("expected assignee to be set")
	}
}

