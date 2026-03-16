package engine

import (
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

// resolveActorTypeFromBoard checks if a raw user name/ID is a human or agent.
// Returns "human" (default for unknown), or "agent".
func resolveActorTypeFromBoard(board *domain.Board, rawUserID string) string {
	for _, u := range board.Users {
		if u.ID == rawUserID || strings.EqualFold(u.Name, rawUserID) {
			return string(u.Type)
		}
	}
	return "human"
}

// ResolveActorType loads the board and checks identity type.
func (e *Engine) ResolveActorType(rawUserID string) (string, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return "", err
	}
	return resolveActorTypeFromBoard(board, rawUserID), nil
}
