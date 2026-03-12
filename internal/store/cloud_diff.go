package store

import (
	"encoding/json"

	"github.com/niladribose/obeya/internal/domain"
)

// BoardDiff describes the changes between two board states.
type BoardDiff struct {
	CreatedItems   []*domain.Item
	UpdatedItems   []*domain.Item
	DeletedItemIDs []string
	MovedItems     []MovedItem
}

// MovedItem records a status change for a single item.
type MovedItem struct {
	ItemID    string
	NewStatus string
}

// BoardSnapshot holds a deep-copied board state for diffing.
type BoardSnapshot struct {
	Items map[string]*domain.Item
}

// SnapshotBoard creates a deep copy of the board's items for later diffing.
func SnapshotBoard(board *domain.Board) *BoardSnapshot {
	snapshot := &BoardSnapshot{
		Items: make(map[string]*domain.Item, len(board.Items)),
	}

	for id, item := range board.Items {
		snapshot.Items[id] = deepCopyItem(item)
	}

	return snapshot
}

// DiffBoard compares a snapshot (before) with the current board (after) and
// returns all detected changes. Moves are detected separately from field updates.
func DiffBoard(before *BoardSnapshot, after *domain.Board) *BoardDiff {
	diff := &BoardDiff{}

	for id, item := range after.Items {
		if _, existed := before.Items[id]; !existed {
			diff.CreatedItems = append(diff.CreatedItems, item)
		}
	}

	for id := range before.Items {
		if _, exists := after.Items[id]; !exists {
			diff.DeletedItemIDs = append(diff.DeletedItemIDs, id)
		}
	}

	for id, afterItem := range after.Items {
		beforeItem, existed := before.Items[id]
		if !existed {
			continue
		}

		statusChanged := beforeItem.Status != afterItem.Status
		fieldsChanged := itemFieldsChanged(beforeItem, afterItem)

		if statusChanged {
			diff.MovedItems = append(diff.MovedItems, MovedItem{
				ItemID:    id,
				NewStatus: afterItem.Status,
			})
		}

		if fieldsChanged {
			diff.UpdatedItems = append(diff.UpdatedItems, afterItem)
		}
	}

	return diff
}

// itemFieldsChanged checks if any non-status fields changed between two item versions.
func itemFieldsChanged(before, after *domain.Item) bool {
	if before.Title != after.Title {
		return true
	}
	if before.Description != after.Description {
		return true
	}
	if before.Priority != after.Priority {
		return true
	}
	if before.Assignee != after.Assignee {
		return true
	}
	if before.ParentID != after.ParentID {
		return true
	}
	if before.Project != after.Project {
		return true
	}
	if !stringSlicesEqual(before.Tags, after.Tags) {
		return true
	}
	if !stringSlicesEqual(before.BlockedBy, after.BlockedBy) {
		return true
	}
	return false
}

// stringSlicesEqual returns true if two slices have identical contents.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// deepCopyItem creates a deep copy of an Item using JSON round-trip.
func deepCopyItem(item *domain.Item) *domain.Item {
	data, err := json.Marshal(item)
	if err != nil {
		panic("failed to marshal item for deep copy: " + err.Error())
	}
	var copy domain.Item
	if err := json.Unmarshal(data, &copy); err != nil {
		panic("failed to unmarshal item for deep copy: " + err.Error())
	}
	return &copy
}
