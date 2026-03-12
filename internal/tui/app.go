package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
)

// App is the enhanced Bubble Tea model for the Obeya board TUI.
type App struct {
	engine    *engine.Engine
	board     *domain.Board
	boardPath string // path to board.json for file watching

	// Board navigation
	columns    []string
	cursorCol  int
	cursorRow  int
	collapsed  map[string]bool
	colScrollY map[int]int // per-column scroll offsets

	// Description accordion
	descExpanded string // item ID whose description is expanded, "" if none
	descScrollY  int    // scroll offset within expanded description

	// State machine
	state     viewState
	prevState viewState

	// Sub-components
	detail     DetailModel
	picker     PickerModel
	input      InputModel
	dashboard  DashboardModel
	confirmMsg string

	// Dimensions
	width  int
	height int

	watcher *boardWatcher
	err     error
}

// NewApp creates a new enhanced TUI app backed by the given engine.
func NewApp(eng *engine.Engine, boardPath string) App {
	return App{
		engine:     eng,
		boardPath:  boardPath,
		collapsed:  make(map[string]bool),
		colScrollY: make(map[int]int),
		state:      stateBoard,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadBoard(), a.startWatching())
}

func (a App) startWatching() tea.Cmd {
	return func() tea.Msg {
		w, err := newBoardWatcher(a.boardPath)
		if err != nil {
			return watcherStartedMsg{watcher: nil, err: err}
		}
		return watcherStartedMsg{watcher: w}
	}
}

func (a App) waitForFileChange() tea.Cmd {
	return func() tea.Msg {
		if a.watcher == nil {
			return nil
		}
		select {
		case _, ok := <-a.watcher.events():
			if !ok {
				return nil
			}
			return boardFileChangedMsg{}
		case err, ok := <-a.watcher.errors():
			if !ok {
				return nil
			}
			return errMsg{err}
		}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case boardLoadedMsg:
		a.board = msg.board
		a.columns = extractColumns(msg.board)
		a.clampCursor()
		if a.state == stateDashboard {
			a.dashboard = newDashboardModel(a.board, a.width, a.height)
		}
		return a, nil
	case errMsg:
		a.err = msg.err
		return a, nil
	case watcherStartedMsg:
		a.watcher = msg.watcher
		if msg.err != nil {
			a.err = fmt.Errorf("file watcher failed: %w (press r to refresh manually)", msg.err)
			return a, nil
		}
		return a, a.waitForFileChange()
	case boardFileChangedMsg:
		return a, tea.Batch(a.loadBoard(), a.waitForFileChange())
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a App) View() string {
	if a.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit, r to retry.", a.err)
	}
	if a.board == nil {
		return "Loading board..."
	}

	switch a.state {
	case stateDetail:
		a.detail.SetSize(a.width, a.height)
		return a.detail.View()
	case statePicker:
		return a.renderBoardWithOverlay(a.picker.View())
	case stateInput:
		return a.renderBoardWithOverlay(a.input.View())
	case stateConfirm:
		return a.renderBoardWithOverlay(a.renderConfirm())
	case stateDashboard:
		a.dashboard.SetSize(a.width, a.height)
		return a.dashboard.View()
	default:
		return a.renderBoard()
	}
}


func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.state {
	case stateBoard:
		return a.handleBoardKey(msg)
	case stateDetail:
		return a.handleDetailKey(msg)
	case statePicker:
		return a.handlePickerKey(msg)
	case stateInput:
		return a.handleInputKey(msg)
	case stateConfirm:
		return a.handleConfirmKey(msg)
	case stateDashboard:
		return a.handleDashboardKey(msg)
	}
	return a, nil
}

func (a App) loadBoard() tea.Cmd {
	return func() tea.Msg {
		board, err := a.engine.ListBoard()
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{board}
	}
}

func (a App) selectedItem() *domain.Item {
	if a.board == nil {
		return nil
	}
	items := a.visibleItemsInColumn(a.cursorCol)
	if a.cursorRow >= len(items) {
		return nil
	}
	return items[a.cursorRow]
}

func (a *App) scrollToSelected() {
	if a.height <= 0 || a.board == nil {
		return
	}
	item := a.selectedItem()
	if item == nil {
		a.colScrollY[a.cursorCol] = 0
		return
	}

	// Render card content for the current column only
	items := a.visibleItemsInColumn(a.cursorCol)
	cardViews := a.renderGroupedCards(items, a.cursorCol)
	if len(cardViews) == 0 {
		a.colScrollY[a.cursorCol] = 0
		return
	}
	cardContent := strings.Join(cardViews, "\n")
	cardLines := strings.Split(cardContent, "\n")

	viewH := a.contentViewHeight()
	if viewH <= 0 || len(cardLines) <= viewH {
		a.colScrollY[a.cursorCol] = 0
		return
	}

	maxOffset := len(cardLines) - viewH

	// Find the selected item's marker line
	marker := fmt.Sprintf("#%d ", item.DisplayNum)
	cardTop := -1
	for i, line := range cardLines {
		if strings.Contains(line, marker) {
			cardTop = i
			break
		}
	}
	if cardTop < 0 {
		return
	}

	offset := a.colScrollY[a.cursorCol]

	// Scroll up if item is above viewport (with 2-line margin)
	if cardTop < offset+2 {
		offset = cardTop - 2
	}
	// Scroll down if item is below viewport
	if cardTop > offset+viewH-6 {
		offset = cardTop - viewH + 6
	}

	// Clamp
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	a.colScrollY[a.cursorCol] = offset
}

// contentViewHeight returns the available height for card content inside a column.
func (a App) contentViewHeight() int {
	// a.height minus: board header(1) + help bar(1) + column border(2) + column header(1)
	h := a.height - 5
	if h < 1 {
		return 0
	}
	return h
}

func (a *App) clampCursor() {
	if a.cursorCol >= len(a.columns) {
		a.cursorCol = len(a.columns) - 1
	}
	if a.cursorCol < 0 {
		a.cursorCol = 0
	}
	items := a.visibleItemsInColumn(a.cursorCol)
	if a.cursorRow >= len(items) {
		a.cursorRow = len(items) - 1
	}
	if a.cursorRow < 0 {
		a.cursorRow = 0
	}
}

func (a App) quit() (tea.Model, tea.Cmd) {
	if a.watcher != nil {
		a.watcher.close()
	}
	return a, tea.Quit
}

func extractColumns(board *domain.Board) []string {
	cols := make([]string, len(board.Columns))
	for i, c := range board.Columns {
		cols[i] = c.Name
	}
	return cols
}


func (a App) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a.quit()
	case "h", "left":
		if a.cursorCol > 0 {
			a.cursorCol--
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "l", "right":
		if a.cursorCol < len(a.columns)-1 {
			a.cursorCol++
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "tab":
		if len(a.columns) > 0 {
			a.cursorCol = (a.cursorCol + 1) % len(a.columns)
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "j", "down":
		items := a.visibleItemsInColumn(a.cursorCol)
		if a.cursorRow < len(items)-1 {
			a.cursorRow++
			a.scrollToSelected()
		}
	case "k", "up":
		if a.cursorRow > 0 {
			a.cursorRow--
			a.scrollToSelected()
		}
	case "enter":
		if item := a.selectedItem(); item != nil {
			a.detail = newDetailModel(item, a.board)
			a.prevState = stateBoard
			a.state = stateDetail
		}
	case "m":
		if item := a.selectedItem(); item != nil {
			a.picker = newPickerModel(
				fmt.Sprintf("Move #%d to:", item.DisplayNum),
				pickerColumn, a.columns,
			)
			a.state = statePicker
			a.prevState = stateBoard
		}
	case "a":
		if item := a.selectedItem(); item != nil {
			users := userNames(a.board)
			a.picker = newPickerModel(
				fmt.Sprintf("Assign #%d to:", item.DisplayNum),
				pickerUser, users,
			)
			a.state = statePicker
			a.prevState = stateBoard
		}
	case "c":
		types := []string{"epic", "story", "task"}
		a.picker = newPickerModel("Create item type:", pickerType, types)
		a.state = statePicker
		a.prevState = stateBoard
	case "d":
		if item := a.selectedItem(); item != nil {
			a.confirmMsg = fmt.Sprintf("Delete #%d %q? (y/n)", item.DisplayNum, item.Title)
			a.state = stateConfirm
		}
	case "b":
		if item := a.selectedItem(); item != nil {
			labels := itemPickerLabels(a.board, item.ID)
			a.picker = newPickerModel(
				fmt.Sprintf("Block #%d by:", item.DisplayNum),
				pickerItem, labels,
			)
			a.state = statePicker
			a.prevState = stateBoard
		}
	case "p":
		if item := a.selectedItem(); item != nil {
			nextPri := cyclePriority(string(item.Priority))
			_ = a.engine.EditItem(item.ID, "", "", nextPri, "", "")
			return a, a.loadBoard()
		}
	case " ":
		if item := a.selectedItem(); item != nil {
			epicID := findEpicAncestor(a.board, item)
			if epicID != "" {
				a.collapsed[epicID] = !a.collapsed[epicID]
				a.clampCursor()
			}
		}
	case "/":
		a.input = newInputModel("Search:")
		a.state = stateInput
		a.prevState = stateBoard
	case "D":
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
		a.prevState = stateBoard
		a.state = stateDashboard
	case "r":
		a.err = nil
		return a, a.loadBoard()
	}
	return a, nil
}

func (a App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	item := a.detail.item
	switch msg.String() {
	case "esc":
		a.state = stateBoard
	case "q":
		return a.quit()
	case "tab":
		a.detail.NextTab()
	case "shift+tab":
		a.detail.PrevTab()
	case "j", "down":
		a.detail.ScrollDown()
	case "k", "up":
		a.detail.ScrollUp()
	case "m":
		if item != nil {
			a.picker = newPickerModel(
				fmt.Sprintf("Move #%d to:", item.DisplayNum),
				pickerColumn, a.columns,
			)
			a.state = statePicker
			a.prevState = stateDetail
		}
	case "a":
		if item != nil {
			users := userNames(a.board)
			a.picker = newPickerModel(
				fmt.Sprintf("Assign #%d to:", item.DisplayNum),
				pickerUser, users,
			)
			a.state = statePicker
			a.prevState = stateDetail
		}
	case "p":
		if item != nil {
			nextPri := cyclePriority(string(item.Priority))
			_ = a.engine.EditItem(item.ID, "", "", nextPri, "", "")
			return a, a.loadBoard()
		}
	}
	return a, nil
}

func (a App) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = a.prevState
	case "j", "down":
		a.picker.MoveDown()
	case "k", "up":
		a.picker.MoveUp()
	case "enter":
		return a.executePickerSelection()
	case "q":
		return a.quit()
	}
	return a, nil
}

func (a App) executePickerSelection() (tea.Model, tea.Cmd) {
	selected := a.picker.Selected()
	item := a.selectedItem()

	switch a.picker.kind {
	case pickerColumn:
		if item != nil && selected != "" {
			_ = a.engine.MoveItem(item.ID, selected, "", "")
		}
	case pickerUser:
		if item != nil && selected != "" {
			_ = a.engine.AssignItem(item.ID, selected, "", "")
		}
	case pickerType:
		a.input = newInputModel(fmt.Sprintf("New %s title:", selected), selected)
		a.state = stateInput
		a.prevState = stateBoard
		return a, nil
	case pickerItem:
		if item != nil && selected != "" {
			blockerNum := extractItemNum(selected)
			_ = a.engine.BlockItem(item.ID, blockerNum, "", "")
		}
	case pickerEpic:
		epicNum := extractItemNum(selected)
		a.dashboard.SelectEpic(a.board, epicNum)
		a.state = stateDashboard
		return a, nil
	}

	a.state = stateBoard
	return a, a.loadBoard()
}

func extractItemNum(display string) string {
	parts := strings.SplitN(display, " ", 2)
	if len(parts) > 0 {
		return strings.TrimPrefix(parts[0], "#")
	}
	return display
}

func (a App) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = a.prevState
		return a, nil
	case "enter":
		value := a.input.Value()
		if value == "" {
			return a, nil
		}
		return a.executeInput(value)
	}
	// Forward to textinput
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return a, cmd
}

func (a App) executeInput(value string) (tea.Model, tea.Cmd) {
	ctx := a.input.context
	switch ctx {
	case "epic", "story", "task":
		parentRef := ""
		if ctx != "epic" {
			if item := a.selectedItem(); item != nil {
				parentRef = item.ID
			}
		}
		_, _ = a.engine.CreateItem(ctx, value, parentRef, "", "medium", "", nil)
	}
	a.state = stateBoard
	return a, a.loadBoard()
}

func (a App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if item := a.selectedItem(); item != nil {
			_ = a.engine.DeleteItem(item.ID, "", "")
		}
		a.state = stateBoard
		return a, a.loadBoard()
	case "n", "N", "esc":
		a.state = stateBoard
	}
	return a, nil
}

func (a App) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "D", "esc":
		a.state = stateBoard
	case "q", "ctrl+c":
		return a.quit()
	case "tab":
		a.dashboard.NextPanel()
	case "E":
		epics := epicPickerLabels(a.board)
		if len(epics) > 0 {
			a.picker = newPickerModel("Select epic for burndown:", pickerEpic, epics)
			a.state = statePicker
			a.prevState = stateDashboard
		}
	case "R", "r":
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
	}
	return a, nil
}

func epicPickerLabels(board *domain.Board) []string {
	var labels []string
	for _, item := range board.Items {
		if item.Type == domain.ItemTypeEpic {
			labels = append(labels, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
		}
	}
	return labels
}

func (a App) renderConfirm() string {
	return a.confirmMsg + "\n\ny: confirm  n/esc: cancel"
}
