package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestGolden_InitialRender captures the initial board state at 120x40.
func TestGolden_InitialRender(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_MoveToTodo captures the board after pressing 'l' (cursor on TODO column).
func TestGolden_MoveToTodo(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_MoveDownCard captures the board after pressing 'j' (second card selected).
func TestGolden_MoveDownCard(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DescriptionExpanded captures the board with description accordion open.
func TestGolden_DescriptionExpanded(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_DescriptionCollapsed captures the board after expanding then collapsing description.
func TestGolden_DescriptionCollapsed(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_NarrowTerminal captures the board at 80x24.
func TestGolden_NarrowTerminal(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 80, 24)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_TabCycle captures the board after tabbing through all columns back to start.
func TestGolden_TabCycle(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	for i := 0; i < 5; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(30 * time.Millisecond)
	}
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}

// TestGolden_InProgressColumn captures the board with cursor on in-progress column.
func TestGolden_InProgressColumn(t *testing.T) {
	boardFile, eng := testBoard(t)
	tm := startAndWait(t, eng, boardFile, 120, 40)
	// Move right twice: backlog -> todo -> in-progress
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(30 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	time.Sleep(50 * time.Millisecond)
	screen := getScreen(t, tm)
	teatest.RequireEqualOutput(t, []byte(screen))
}
