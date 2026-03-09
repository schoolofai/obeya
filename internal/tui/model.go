package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
)

// Model is the Bubble Tea model for the Obeya board TUI.
type Model struct {
	engine    *engine.Engine
	board     *domain.Board
	columns   []string
	cursorCol int
	cursorRow int
	err       error
}

// New creates a new TUI model backed by the given engine.
func New(eng *engine.Engine) Model {
	return Model{engine: eng}
}

type boardLoadedMsg struct {
	board *domain.Board
}

type errMsg struct {
	err error
}

// Init loads the board on startup.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		board, err := m.engine.ListBoard()
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{board}
	}
}

// Update handles key messages and board load results.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case boardLoadedMsg:
		return m.handleBoardLoaded(msg), nil
	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleBoardLoaded(msg boardLoadedMsg) Model {
	m.board = msg.board
	m.columns = make([]string, len(m.board.Columns))
	for i, c := range m.board.Columns {
		m.columns[i] = c.Name
	}
	return m
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m = m.moveCursorLeft()
	case "l", "right":
		m = m.moveCursorRight()
	case "j", "down":
		m = m.moveCursorDown()
	case "k", "up":
		m = m.moveCursorUp()
	case "r":
		return m, m.reloadBoard()
	}
	return m, nil
}

func (m Model) moveCursorLeft() Model {
	if m.cursorCol > 0 {
		m.cursorCol--
		m.cursorRow = 0
	}
	return m
}

func (m Model) moveCursorRight() Model {
	if m.cursorCol < len(m.columns)-1 {
		m.cursorCol++
		m.cursorRow = 0
	}
	return m
}

func (m Model) moveCursorDown() Model {
	items := m.itemsInColumn(m.columns[m.cursorCol])
	if m.cursorRow < len(items)-1 {
		m.cursorRow++
	}
	return m
}

func (m Model) moveCursorUp() Model {
	if m.cursorRow > 0 {
		m.cursorRow--
	}
	return m
}

func (m Model) reloadBoard() tea.Cmd {
	return func() tea.Msg {
		board, err := m.engine.ListBoard()
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{board}
	}
}

func (m Model) itemsInColumn(colName string) []*domain.Item {
	if m.board == nil {
		return nil
	}
	var items []*domain.Item
	for _, item := range m.board.Items {
		if item.Status == colName {
			items = append(items, item)
		}
	}
	return items
}
