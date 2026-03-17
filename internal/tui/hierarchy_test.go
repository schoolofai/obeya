package tui

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func testHierarchyBoard() *domain.Board {
	b := domain.NewBoard("test")
	b.Items["epic-1"] = &domain.Item{ID: "epic-1", DisplayNum: 1, Title: "Auth Rewrite", Type: domain.ItemTypeEpic, Status: "backlog", ParentID: ""}
	b.Items["story-4"] = &domain.Item{ID: "story-4", DisplayNum: 4, Title: "Session Management", Type: domain.ItemTypeStory, Status: "backlog", ParentID: "epic-1"}
	b.Items["task-2"] = &domain.Item{ID: "task-2", DisplayNum: 2, Title: "Refactor middleware", Type: domain.ItemTypeTask, Status: "backlog", ParentID: "story-4"}
	b.Items["task-3"] = &domain.Item{ID: "task-3", DisplayNum: 3, Title: "JWT validation", Type: domain.ItemTypeTask, Status: "in-progress", ParentID: "story-4"}
	b.Items["task-6"] = &domain.Item{ID: "task-6", DisplayNum: 6, Title: "Update session store", Type: domain.ItemTypeTask, Status: "done", ParentID: "story-4"}
	b.Items["task-20"] = &domain.Item{ID: "task-20", DisplayNum: 20, Title: "Fix README", Type: domain.ItemTypeTask, Status: "backlog", ParentID: ""}
	b.DisplayMap[1] = "epic-1"
	b.DisplayMap[4] = "story-4"
	b.DisplayMap[2] = "task-2"
	b.DisplayMap[3] = "task-3"
	b.DisplayMap[6] = "task-6"
	b.DisplayMap[20] = "task-20"
	return b
}

func TestBreadcrumbPath(t *testing.T) {
	b := testHierarchyBoard()
	got := breadcrumbPath(b, b.Items["task-2"], 100)
	want := "#1 › #4"
	if got != want {
		t.Errorf("breadcrumbPath = %q, want %q", got, want)
	}
}

func TestBreadcrumbPath_StoryUnderEpic(t *testing.T) {
	b := testHierarchyBoard()
	got := breadcrumbPath(b, b.Items["story-4"], 100)
	want := "#1"
	if got != want {
		t.Errorf("breadcrumbPath = %q, want %q", got, want)
	}
}

func TestBreadcrumbPath_NoParent(t *testing.T) {
	b := testHierarchyBoard()
	got := breadcrumbPath(b, b.Items["epic-1"], 100)
	if got != "" {
		t.Errorf("breadcrumbPath for root = %q, want empty", got)
	}
}

func TestBreadcrumbPath_CycleProtection(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", DisplayNum: 10, Title: "A", ParentID: "b"}
	b.Items["b"] = &domain.Item{ID: "b", DisplayNum: 20, Title: "B", ParentID: "a"}
	got := breadcrumbPath(b, b.Items["a"], 100)
	if got == "" {
		t.Error("should produce some breadcrumb even with cycle")
	}
}

func TestChildCount(t *testing.T) {
	b := testHierarchyBoard()
	// Epic has story-4 + task-2 + task-3 + task-6 = 4 descendants
	got := childCount(b, "epic-1")
	if got != 4 {
		t.Errorf("childCount(epic) = %d, want 4", got)
	}
	// Story has task-2 + task-3 + task-6 = 3 descendants
	got = childCount(b, "story-4")
	if got != 3 {
		t.Errorf("childCount(story) = %d, want 3", got)
	}
	// Leaf task has 0
	got = childCount(b, "task-2")
	if got != 0 {
		t.Errorf("childCount(leaf) = %d, want 0", got)
	}
}

func TestDoneCount(t *testing.T) {
	b := testHierarchyBoard()
	// Epic: only task-6 is done = 1
	got := doneCount(b, "epic-1")
	if got != 1 {
		t.Errorf("doneCount(epic) = %d, want 1", got)
	}
}

func TestHasChildren(t *testing.T) {
	b := testHierarchyBoard()
	if !hasChildItems(b, "epic-1") {
		t.Error("epic should have children")
	}
	if !hasChildItems(b, "story-4") {
		t.Error("story should have children")
	}
	if hasChildItems(b, "task-2") {
		t.Error("leaf task should not have children")
	}
}

func TestIsHiddenByCollapse_SameColumn(t *testing.T) {
	b := testHierarchyBoard()
	collapsed := map[string]bool{"epic-1": true}
	// story-4 is in backlog (same as epic-1) → hidden
	if !isHiddenByCollapse(b, b.Items["story-4"], collapsed) {
		t.Error("story in same column as collapsed epic should be hidden")
	}
	// task-2 is in backlog (same as epic-1) → hidden (grandchild)
	if !isHiddenByCollapse(b, b.Items["task-2"], collapsed) {
		t.Error("task in same column as collapsed epic should be hidden")
	}
}

func TestIsHiddenByCollapse_DifferentColumn(t *testing.T) {
	b := testHierarchyBoard()
	collapsed := map[string]bool{"epic-1": true}
	// task-3 is in in-progress, epic is in backlog → NOT hidden
	if isHiddenByCollapse(b, b.Items["task-3"], collapsed) {
		t.Error("task in different column than collapsed epic should NOT be hidden")
	}
}

func TestIsHiddenByCollapse_NestedCollapse(t *testing.T) {
	b := testHierarchyBoard()
	collapsed := map[string]bool{"story-4": true}
	// task-2 is child of story-4, both in backlog → hidden
	if !isHiddenByCollapse(b, b.Items["task-2"], collapsed) {
		t.Error("task should be hidden when parent story is collapsed in same column")
	}
	// epic-1 is parent of story-4, NOT hidden (parent is never hidden by child collapse)
	if isHiddenByCollapse(b, b.Items["epic-1"], collapsed) {
		t.Error("parent should NOT be hidden by child's collapse")
	}
}

func TestIsHiddenByCollapse_CycleProtection(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", Status: "backlog", ParentID: "b"}
	b.Items["b"] = &domain.Item{ID: "b", Status: "backlog", ParentID: "a"}
	collapsed := map[string]bool{"a": true}
	// Should not infinite loop
	_ = isHiddenByCollapse(b, b.Items["b"], collapsed)
}

func TestOrderItemsHierarchically(t *testing.T) {
	b := testHierarchyBoard()
	items := []*domain.Item{
		b.Items["task-20"], // orphan
		b.Items["task-2"],  // child of story-4
		b.Items["story-4"], // child of epic-1
		b.Items["epic-1"],  // root
	}
	ordered := orderItemsHierarchically(b, items)
	// Expected: epic-1, story-4, task-2, task-20
	wantOrder := []string{"epic-1", "story-4", "task-2", "task-20"}
	if len(ordered) != len(wantOrder) {
		t.Fatalf("got %d items, want %d", len(ordered), len(wantOrder))
	}
	for i, id := range wantOrder {
		if ordered[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, ordered[i].ID, id)
		}
	}
}

func TestOrderItemsHierarchically_MultipleRoots(t *testing.T) {
	b := testHierarchyBoard()
	b.Items["epic-10"] = &domain.Item{ID: "epic-10", DisplayNum: 10, Title: "Rate Limiting", Type: domain.ItemTypeEpic, Status: "backlog"}
	b.Items["task-11"] = &domain.Item{ID: "task-11", DisplayNum: 11, Title: "Throttle", Type: domain.ItemTypeTask, Status: "backlog", ParentID: "epic-10"}

	items := []*domain.Item{
		b.Items["task-11"],
		b.Items["epic-10"],
		b.Items["task-2"],
		b.Items["story-4"],
		b.Items["epic-1"],
		b.Items["task-20"],
	}
	ordered := orderItemsHierarchically(b, items)
	// epic-1 tree first (DisplayNum 1), then epic-10 tree (DisplayNum 10), then orphan task-20
	wantOrder := []string{"epic-1", "story-4", "task-2", "epic-10", "task-11", "task-20"}
	for i, id := range wantOrder {
		if i >= len(ordered) || ordered[i].ID != id {
			got := "<nil>"
			if i < len(ordered) {
				got = ordered[i].ID
			}
			t.Errorf("position %d: got %s, want %s", i, got, id)
		}
	}
}
