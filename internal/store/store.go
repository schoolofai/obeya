package store

import "github.com/niladribose/obeya/internal/domain"

// Store abstracts board persistence. Lite uses JSON files; Pro will use a cloud API.
type Store interface {
	// Transaction performs an atomic read-modify-write on the board.
	// The provided function receives the current board state and may mutate it.
	// Changes are persisted atomically when the function returns nil.
	Transaction(fn func(board *domain.Board) error) error

	// LoadBoard returns a read-only snapshot of the board (no lock).
	LoadBoard() (*domain.Board, error)

	// InitBoard creates a new board with the given config.
	InitBoard(name string, columns []string) error

	// BoardExists checks if a board has been initialized in the current directory.
	BoardExists() bool
}
