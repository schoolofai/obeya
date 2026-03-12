package store

import (
	"fmt"

	"github.com/niladribose/obeya/internal/domain"
)

// CloudStore implements the Store interface using the Obeya Cloud API.
// It uses a diff-and-sync strategy for transactions: fetch the full board,
// apply mutations in-memory, detect changes, and send targeted API calls.
type CloudStore struct {
	client  *CloudClient
	boardID string
}

// NewCloudStore creates a new CloudStore for the given API URL, token, and board ID.
func NewCloudStore(apiURL, token, boardID string) *CloudStore {
	return &CloudStore{
		client:  NewCloudClient(apiURL, token),
		boardID: boardID,
	}
}

// Transaction performs a read-modify-write cycle against the cloud API.
// 1. Fetch the current board state via export endpoint
// 2. Snapshot the board for later diffing
// 3. Run the mutation function on the board
// 4. If fn returns error, abort without sending any API calls
// 5. Diff the snapshot vs mutated board
// 6. Send granular API calls for each detected change
func (cs *CloudStore) Transaction(fn func(board *domain.Board) error) error {
	board, err := cs.client.ExportBoard(cs.boardID)
	if err != nil {
		return fmt.Errorf("cloud transaction: failed to fetch board: %w", err)
	}

	snapshot := SnapshotBoard(board)

	if err := fn(board); err != nil {
		return err
	}

	diff := DiffBoard(snapshot, board)

	if err := cs.applyDiff(diff); err != nil {
		return fmt.Errorf("cloud transaction: failed to apply changes: %w", err)
	}

	return nil
}

// LoadBoard fetches a read-only snapshot of the board from the cloud.
func (cs *CloudStore) LoadBoard() (*domain.Board, error) {
	board, err := cs.client.ExportBoard(cs.boardID)
	if err != nil {
		return nil, fmt.Errorf("cloud load board failed: %w", err)
	}
	return board, nil
}

// InitBoard is not supported in cloud mode — boards are created via ob init --cloud.
func (cs *CloudStore) InitBoard(name string, columns []string) error {
	return fmt.Errorf("InitBoard is not supported in cloud mode — use 'ob init --cloud' to create a cloud board")
}

// BoardExists always returns true for cloud mode — if cloud.json exists, the board exists.
func (cs *CloudStore) BoardExists() bool {
	return true
}

// BoardFilePath returns an empty string in cloud mode.
// The TUI uses WebSocket instead of fsnotify for cloud boards.
func (cs *CloudStore) BoardFilePath() string {
	return ""
}

// applyDiff sends the detected changes to the cloud API as individual operations.
func (cs *CloudStore) applyDiff(diff *BoardDiff) error {
	for _, item := range diff.CreatedItems {
		if err := cs.client.CreateItem(cs.boardID, item); err != nil {
			return fmt.Errorf("failed to create item %s: %w", item.ID, err)
		}
	}

	for _, moved := range diff.MovedItems {
		if err := cs.client.MoveItem(moved.ItemID, moved.NewStatus); err != nil {
			return fmt.Errorf("failed to move item %s: %w", moved.ItemID, err)
		}
	}

	for _, item := range diff.UpdatedItems {
		if err := cs.client.UpdateItem(item); err != nil {
			return fmt.Errorf("failed to update item %s: %w", item.ID, err)
		}
	}

	for _, id := range diff.DeletedItemIDs {
		if err := cs.client.DeleteItem(id); err != nil {
			return fmt.Errorf("failed to delete item %s: %w", id, err)
		}
	}

	return nil
}
