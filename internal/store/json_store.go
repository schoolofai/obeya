package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/niladribose/obeya/internal/domain"
)

type JSONStore struct {
	rootDir   string
	obeyaDir  string
	boardFile string
	lockFile  string
}

func NewJSONStore(rootDir string) *JSONStore {
	obeyaDir := filepath.Join(rootDir, ".obeya")
	return &JSONStore{
		rootDir:   rootDir,
		obeyaDir:  obeyaDir,
		boardFile: filepath.Join(obeyaDir, "board.json"),
		lockFile:  filepath.Join(obeyaDir, "board.lock"),
	}
}

func (s *JSONStore) BoardExists() bool {
	_, err := os.Stat(s.boardFile)
	return err == nil
}

func (s *JSONStore) InitBoard(name string, columns []string) error {
	if s.BoardExists() {
		return fmt.Errorf("board already initialized in %s", s.obeyaDir)
	}

	if err := os.MkdirAll(s.obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .obeya directory: %w", err)
	}

	var board *domain.Board
	if len(columns) > 0 {
		board = domain.NewBoardWithColumns(name, columns)
	} else {
		board = domain.NewBoard(name)
	}

	return s.writeBoard(board)
}

func (s *JSONStore) LoadBoard() (*domain.Board, error) {
	if !s.BoardExists() {
		return nil, fmt.Errorf("no board found — run 'ob init' first")
	}

	data, err := os.ReadFile(s.boardFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read board file: %w", err)
	}

	var board domain.Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to parse board file: %w", err)
	}

	if board.Plans == nil {
		board.Plans = make(map[string]*domain.Plan)
	}

	if board.Projects == nil {
		board.Projects = make(map[string]*domain.LinkedProject)
	}

	return &board, nil
}

func (s *JSONStore) Transaction(fn func(board *domain.Board) error) error {
	fileLock := flock.New(s.lockFile)

	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire board lock: %w", err)
	}
	defer fileLock.Unlock()

	board, err := s.LoadBoard()
	if err != nil {
		return err
	}

	savedVersion := board.Version

	if err := fn(board); err != nil {
		return err
	}

	if board.Version != savedVersion {
		return fmt.Errorf("concurrent modification detected: board version changed during transaction")
	}

	board.Version++
	board.UpdatedAt = time.Now()

	return s.writeBoard(board)
}

func (s *JSONStore) writeBoard(board *domain.Board) error {
	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}

	tmpFile := s.boardFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp board file: %w", err)
	}

	if err := os.Rename(tmpFile, s.boardFile); err != nil {
		return fmt.Errorf("failed to atomically replace board file: %w", err)
	}

	return nil
}
