package store_test

import (
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func TestDiffBoard_NoChanges(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Task",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	board.DisplayMap[1] = "item1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)
	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 0 {
		t.Errorf("expected 0 created items, got %d", len(diff.CreatedItems))
	}
	if len(diff.UpdatedItems) != 0 {
		t.Errorf("expected 0 updated items, got %d", len(diff.UpdatedItems))
	}
	if len(diff.DeletedItemIDs) != 0 {
		t.Errorf("expected 0 deleted items, got %d", len(diff.DeletedItemIDs))
	}
	if len(diff.MovedItems) != 0 {
		t.Errorf("expected 0 moved items, got %d", len(diff.MovedItems))
	}
}

func TestDiffBoard_ItemCreated(t *testing.T) {
	board := domain.NewBoard("test")
	snapshot := store.SnapshotBoard(board)

	board.Items["new1"] = &domain.Item{
		ID: "new1", DisplayNum: 1, Title: "New Task",
		Type: domain.ItemTypeTask, Status: "backlog", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "new1"
	board.NextDisplay = 2

	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 1 {
		t.Fatalf("expected 1 created item, got %d", len(diff.CreatedItems))
	}
	if diff.CreatedItems[0].ID != "new1" {
		t.Errorf("created item ID: got %q, want 'new1'", diff.CreatedItems[0].ID)
	}
}

func TestDiffBoard_ItemDeleted(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["del1"] = &domain.Item{
		ID: "del1", DisplayNum: 1, Title: "Delete Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "del1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	delete(board.Items, "del1")
	delete(board.DisplayMap, 1)

	diff := store.DiffBoard(snapshot, board)

	if len(diff.DeletedItemIDs) != 1 {
		t.Fatalf("expected 1 deleted item, got %d", len(diff.DeletedItemIDs))
	}
	if diff.DeletedItemIDs[0] != "del1" {
		t.Errorf("deleted item ID: got %q, want 'del1'", diff.DeletedItemIDs[0])
	}
}

func TestDiffBoard_ItemMoved(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["mv1"] = &domain.Item{
		ID: "mv1", DisplayNum: 1, Title: "Move Me",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "mv1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["mv1"].Status = "done"
	board.Items["mv1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.MovedItems) != 1 {
		t.Fatalf("expected 1 moved item, got %d", len(diff.MovedItems))
	}
	if diff.MovedItems[0].ItemID != "mv1" {
		t.Errorf("moved item ID: got %q, want 'mv1'", diff.MovedItems[0].ItemID)
	}
	if diff.MovedItems[0].NewStatus != "done" {
		t.Errorf("new status: got %q, want 'done'", diff.MovedItems[0].NewStatus)
	}
}

func TestDiffBoard_ItemUpdated(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["upd1"] = &domain.Item{
		ID: "upd1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		Description: "old desc", CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "upd1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["upd1"].Title = "Updated"
	board.Items["upd1"].Description = "new desc"
	board.Items["upd1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.UpdatedItems) != 1 {
		t.Fatalf("expected 1 updated item, got %d", len(diff.UpdatedItems))
	}
	if diff.UpdatedItems[0].ID != "upd1" {
		t.Errorf("updated item ID: got %q, want 'upd1'", diff.UpdatedItems[0].ID)
	}
}

func TestDiffBoard_MoveAndUpdateSeparated(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["both1"] = &domain.Item{
		ID: "both1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
		CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "both1"
	board.NextDisplay = 2

	snapshot := store.SnapshotBoard(board)

	board.Items["both1"].Status = "done"
	board.Items["both1"].Title = "Changed"
	board.Items["both1"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.MovedItems) != 1 {
		t.Errorf("expected 1 moved item, got %d", len(diff.MovedItems))
	}
	if len(diff.UpdatedItems) != 1 {
		t.Errorf("expected 1 updated item, got %d", len(diff.UpdatedItems))
	}
}

func TestDiffBoard_MultipleChanges(t *testing.T) {
	now := time.Now()
	board := domain.NewBoard("test")
	board.Items["keep"] = &domain.Item{
		ID: "keep", DisplayNum: 1, Title: "Keep", Type: domain.ItemTypeTask,
		Status: "todo", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now,
	}
	board.Items["remove"] = &domain.Item{
		ID: "remove", DisplayNum: 2, Title: "Remove", Type: domain.ItemTypeTask,
		Status: "todo", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now,
	}
	board.DisplayMap[1] = "keep"
	board.DisplayMap[2] = "remove"
	board.NextDisplay = 3

	snapshot := store.SnapshotBoard(board)

	// Delete one
	delete(board.Items, "remove")
	delete(board.DisplayMap, 2)

	// Create one
	board.Items["added"] = &domain.Item{
		ID: "added", DisplayNum: 3, Title: "Added", Type: domain.ItemTypeTask,
		Status: "backlog", Priority: domain.PriorityLow,
	}
	board.DisplayMap[3] = "added"
	board.NextDisplay = 4

	// Move one
	board.Items["keep"].Status = "done"
	board.Items["keep"].UpdatedAt = time.Now()

	diff := store.DiffBoard(snapshot, board)

	if len(diff.CreatedItems) != 1 {
		t.Errorf("created: got %d, want 1", len(diff.CreatedItems))
	}
	if len(diff.DeletedItemIDs) != 1 {
		t.Errorf("deleted: got %d, want 1", len(diff.DeletedItemIDs))
	}
	if len(diff.MovedItems) != 1 {
		t.Errorf("moved: got %d, want 1", len(diff.MovedItems))
	}
}

func TestSnapshotBoard_DeepCopy(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["item1"] = &domain.Item{
		ID: "item1", DisplayNum: 1, Title: "Original",
		Type: domain.ItemTypeTask, Status: "todo", Priority: domain.PriorityMedium,
	}
	board.DisplayMap[1] = "item1"

	snapshot := store.SnapshotBoard(board)

	// Mutate original
	board.Items["item1"].Title = "Modified"
	board.Items["item2"] = &domain.Item{ID: "item2"}

	// Snapshot should be unchanged
	if snapshot.Items["item1"].Title != "Original" {
		t.Errorf("snapshot mutated: title is %q, want 'Original'", snapshot.Items["item1"].Title)
	}
	if _, exists := snapshot.Items["item2"]; exists {
		t.Error("snapshot should not contain item2")
	}
}
