package tui

import (
	"fmt"
	"os/user"
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
	columns   []string
	cursorCol int
	cursorRow int
	collapsed map[string]bool
	colModels []ColumnModel

	// Description accordion
	descExpanded string // item ID whose description is expanded, "" if none
	descScrollY  int    // scroll offset within expanded description

	// Review context accordion
	reviewExpanded string // item ID whose review context is expanded
	reviewScrollY  int

	// State machine
	state     viewState
	prevState viewState

	// Sub-components
	detail      DetailModel
	picker      PickerModel
	input       InputModel
	dashboard   DashboardModel
	dag         DAGModel
	pastReviews PastReviewsModel
	confirmMsg  string

	// Dimensions
	width        int
	height       int
	customWidths map[int]int // per-column width overrides (colIdx → width), set by +/-

	// Identity for review operations
	userID    string
	sessionID string

	watcher boardWatcher
	err     error
}

// NewApp creates a new enhanced TUI app backed by the given engine.
func NewApp(eng *engine.Engine, boardPath string) App {
	uid := resolveCurrentUser()
	return App{
		engine:    eng,
		boardPath: boardPath,
		collapsed:    make(map[string]bool),
		customWidths: make(map[int]int),
		state:        stateBoard,
		userID:    uid,
		sessionID: domain.GenerateID(),
	}
}

func (a *App) initColumnModels() {
	widths := a.columnWidths()
	viewH := a.contentViewHeight()
	a.colModels = make([]ColumnModel, len(a.columns))
	for i, name := range a.columns {
		w := 22
		if i < len(widths) {
			w = widths[i]
		}
		a.colModels[i] = NewColumnModel(name, w, viewH)
	}
	if a.cursorCol >= 0 && a.cursorCol < len(a.colModels) {
		a.colModels[a.cursorCol].active = true
	}
}

// resizeColumn adjusts the custom width of a column by delta chars.
// Steals/gives space from adjacent non-active columns.
func (a *App) resizeColumn(colIdx int, delta int) {
	if colIdx < 0 || colIdx >= len(a.columns) {
		return
	}
	widths := a.columnWidths()
	current := widths[colIdx]
	newW := current + delta
	if newW < 6 {
		newW = 6
	}
	a.customWidths[colIdx] = newW
}

// rebuildColumnModels recreates column models with current widths.
func (a *App) rebuildColumnModels() {
	widths := a.columnWidths()
	viewH := a.contentViewHeight()
	for i := range a.colModels {
		w := 22
		if i < len(widths) {
			w = widths[i]
		}
		a.colModels[i].SetSize(w, viewH)
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadBoard(), a.startWatching())
}

func (a App) startWatching() tea.Cmd {
	return func() tea.Msg {
		w, err := newLocalBoardWatcher(a.boardPath)
		if err != nil {
			return watcherStartedMsg{err: err}
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
		widths := a.columnWidths()
		viewH := a.contentViewHeight()
		for i := range a.colModels {
			w := 22
			if i < len(widths) {
				w = widths[i]
			}
			a.colModels[i].SetSize(w, viewH)
		}
		if a.state == stateDAG {
			a.dag.SetSize(msg.Width, msg.Height)
		}
		return a, nil
	case boardLoadedMsg:
		a.board = msg.board
		a.columns = extractColumns(msg.board)
		// Always show human-review column — signals humans are part of the process.
		a.columns = append(a.columns, humanReviewColName)
		a.initColumnModels()
		a.clampCursor()
		if a.state == stateDashboard {
			a.dashboard = newDashboardModel(a.board, a.width, a.height)
		}
		if a.state == stateDAG {
			a.dag = newDAGModel(a.board, a.width, a.height)
		}
		return a, nil
	case dagTickMsg:
		if a.state == stateDAG {
			a.dag.tick++
			a.dag.updateViewport()
			return a, dagTickCmd()
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
	case stateDAG:
		a.dag.SetSize(a.width, a.height)
		return a.dag.View()
	case statePastReviews:
		a.pastReviews.SetSize(a.width, a.height)
		return a.pastReviews.View()
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
	case statePastReviews:
		return a.handlePastReviewsKey(msg)
	case stateDAG:
		return a.handleDAGKey(msg)
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
	if a.height <= 0 || a.board == nil || a.cursorCol >= len(a.colModels) {
		return
	}
	item := a.selectedItem()
	if item == nil {
		a.colModels[a.cursorCol].ScrollToLine(0)
		return
	}

	items := a.visibleItemsInColumn(a.cursorCol)
	var cardViews []string
	for _, it := range items {
		cardViews = append(cardViews, a.renderCard(it, a.isItemAtCursor(it)))
	}
	if len(cardViews) == 0 {
		a.colModels[a.cursorCol].ScrollToLine(0)
		return
	}
	cardContent := strings.Join(cardViews, "\n")
	cardLines := strings.Split(cardContent, "\n")

	viewH := a.contentViewHeight()
	if viewH <= 0 || len(cardLines) <= viewH {
		a.colModels[a.cursorCol].ScrollToLine(0)
		return
	}

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

	offset := a.colModels[a.cursorCol].viewport.YOffset

	// Scroll up if item is above viewport (with 2-line margin)
	if cardTop < offset+2 {
		offset = cardTop - 2
	}
	// Scroll down if item is below viewport
	if cardTop > offset+viewH-6 {
		offset = cardTop - viewH + 6
	}

	if offset < 0 {
		offset = 0
	}
	a.colModels[a.cursorCol].ScrollToLine(offset)
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


func (a *App) collapseDescription() {
	a.descExpanded = ""
	a.descScrollY = 0
}

func (a *App) collapseReviewContext() {
	a.reviewExpanded = ""
	a.reviewScrollY = 0
}

// clampDescScroll ensures descScrollY doesn't exceed the maximum scroll offset
// for the currently expanded description.
func (a *App) clampDescScroll(maxLines int) {
	if a.descExpanded == "" {
		return
	}
	item := a.selectedItem()
	if item == nil || item.Description == "" {
		return
	}
	w := a.columnWidth()
	contentW := w - 4
	if contentW < 10 {
		contentW = 10
	}
	// Count total wrapped lines (same logic as renderDescription)
	paragraphs := strings.Split(item.Description, "\n")
	totalLines := 0
	for _, p := range paragraphs {
		if p == "" {
			totalLines++
			continue
		}
		totalLines += len(wrapText(p, contentW))
	}
	maxScroll := totalLines - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.descScrollY > maxScroll {
		a.descScrollY = maxScroll
	}
}

// clampReviewScroll ensures reviewScrollY doesn't exceed the maximum scroll offset
// for the currently expanded review context.
func (a *App) clampReviewScroll(maxLines int) {
	if a.reviewExpanded == "" {
		return
	}
	item := a.selectedItem()
	if item == nil || item.ReviewContext == nil {
		return
	}
	w := a.columnWidth()
	contentW := w - 4
	if contentW < 10 {
		contentW = 10
	}
	totalLines := len(reviewContextLines(item.ReviewContext, contentW))
	maxScroll := totalLines - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.reviewScrollY > maxScroll {
		a.reviewScrollY = maxScroll
	}
}

func (a App) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a.quit()
	case "esc":
		if a.descExpanded != "" {
			a.collapseDescription()
			return a, nil // consume esc, don't propagate
		}
	case "h", "left":
		if a.cursorCol > 0 {
			a.collapseDescription()
			a.collapseReviewContext()
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = false
			}
			a.cursorCol--
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = true
			}
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "l", "right":
		if a.cursorCol < len(a.columns)-1 {
			a.collapseDescription()
			a.collapseReviewContext()
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = false
			}
			a.cursorCol++
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = true
			}
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "tab":
		a.collapseDescription()
		a.collapseReviewContext()
		if len(a.columns) > 0 {
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = false
			}
			a.cursorCol = (a.cursorCol + 1) % len(a.columns)
			if len(a.colModels) > 0 {
				a.colModels[a.cursorCol].active = true
			}
			a.cursorRow = 0
			a.clampCursor()
			a.scrollToSelected()
		}
	case "j", "down":
		a.collapseDescription()
		a.collapseReviewContext()
		items := a.visibleItemsInColumn(a.cursorCol)
		if a.cursorRow < len(items)-1 {
			a.cursorRow++
			a.scrollToSelected()
		}
	case "k", "up":
		a.collapseDescription()
		a.collapseReviewContext()
		if a.cursorRow > 0 {
			a.cursorRow--
			a.scrollToSelected()
		}
	case "v":
		if item := a.selectedItem(); item != nil && item.Description != "" {
			if a.descExpanded == item.ID {
				a.descExpanded = ""
				a.descScrollY = 0
			} else {
				a.descExpanded = item.ID
				a.descScrollY = 0
			}
		}
	case "J":
		if a.descExpanded != "" {
			a.descScrollY++
			a.clampDescScroll(5)
		}
	case "K":
		if a.descExpanded != "" {
			if a.descScrollY > 0 {
				a.descScrollY--
			}
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
		// No-op: hierarchy shown via indentation, no collapse
	case "+", "=":
		a.resizeColumn(a.cursorCol, 4)
		a.rebuildColumnModels()
	case "-", "_":
		a.resizeColumn(a.cursorCol, -4)
		a.rebuildColumnModels()
	case "0":
		// Reset all custom widths
		a.customWidths = make(map[int]int)
		a.rebuildColumnModels()
	case "/":
		a.input = newInputModel("Search:")
		a.state = stateInput
		a.prevState = stateBoard
	case "D":
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
		a.prevState = stateBoard
		a.state = stateDashboard
	case "G":
		a.dag = newDAGModel(a.board, a.width, a.height)
		a.prevState = stateBoard
		a.state = stateDAG
		return a, dagTickCmd()
	case "V":
		sel := a.selectedItem()
		if sel != nil && sel.ReviewContext != nil {
			if a.reviewExpanded == sel.ID {
				a.reviewExpanded = ""
				a.reviewScrollY = 0
			} else {
				a.reviewExpanded = sel.ID
				a.reviewScrollY = 0
			}
		}
	case "ctrl+j":
		if a.reviewExpanded != "" {
			a.reviewScrollY++
			a.clampReviewScroll(5)
		}
	case "R":
		if isHumanReviewColumn(a.columns, a.cursorCol) {
			sel := a.selectedItem()
			if sel != nil {
				_ = a.engine.ReviewItem(
					fmt.Sprint(sel.DisplayNum), "reviewed", a.userID, a.sessionID,
				)
				return a, a.loadBoard()
			}
		}
	case "x":
		if isHumanReviewColumn(a.columns, a.cursorCol) {
			sel := a.selectedItem()
			if sel != nil {
				_ = a.engine.ReviewItem(
					fmt.Sprint(sel.DisplayNum), "hidden", a.userID, a.sessionID,
				)
				return a, a.loadBoard()
			}
		}
	case "ctrl+k":
		if a.reviewExpanded != "" && a.reviewScrollY > 0 {
			a.reviewScrollY--
		}
	case "P":
		a.pastReviews = newPastReviewsModel(a.board)
		a.pastReviews.SetSize(a.width, a.height)
		a.state = statePastReviews
		return a, nil
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
		a.state = a.prevState
		if a.state == stateDAG {
			return a, dagTickCmd()
		}
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
	// When picker was invoked from DAG, use the DAG's selected item
	if a.prevState == stateDAG {
		item = a.dag.SelectedItem()
	}

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

	returnState := a.prevState
	if returnState == 0 {
		returnState = stateBoard
	}
	a.state = returnState
	if returnState == stateDAG {
		return a, tea.Batch(a.loadBoard(), dagTickCmd())
	}
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
		assignee := a.firstRegisteredUser()
		if assignee == "" {
			a.err = fmt.Errorf("no registered users — register one with 'ob user add' then retry, or use CLI: ob create %s %q --assign <user>", ctx, value)
			a.state = stateBoard
			return a, nil
		}
		parentRef := ""
		if ctx != "epic" {
			if item := a.selectedItem(); item != nil {
				parentRef = item.ID
			}
		}
		if _, err := a.engine.CreateItem(ctx, value, parentRef, "", "medium", assignee, nil, ""); err != nil {
			a.err = fmt.Errorf("create failed: %w", err)
			a.state = stateBoard
			return a, nil
		}
	}
	a.state = stateBoard
	return a, a.loadBoard()
}

func (a App) firstRegisteredUser() string {
	if a.board == nil || len(a.board.Users) == 0 {
		return ""
	}
	for id := range a.board.Users {
		return id
	}
	return ""
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

func (a App) handleDAGKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "G", "esc":
		a.state = stateBoard
		return a, nil
	case "q", "ctrl+c":
		return a.quit()
	case "j", "down":
		a.dag.moveToNextNode()
	case "k", "up":
		a.dag.moveToPrevNode()
	case "h", "left":
		a.dag.scrollLeft()
	case "l", "right":
		a.dag.scrollRight()
	case "enter":
		if item := a.dag.SelectedItem(); item != nil {
			a.detail = newDetailModel(item, a.board)
			a.prevState = stateDAG
			a.state = stateDetail
		}
	case "D":
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
		a.prevState = stateDAG
		a.state = stateDashboard
	case "0":
		a.dag.autoScrollToInProgress()
		a.dag.updateViewport()
	case "m":
		if item := a.dag.SelectedItem(); item != nil {
			a.picker = newPickerModel(
				fmt.Sprintf("Move #%d to:", item.DisplayNum),
				pickerColumn, a.columns,
			)
			a.state = statePicker
			a.prevState = stateDAG
		}
	case "r":
		return a, a.loadBoard()
	}
	return a, nil
}

func (a App) handlePastReviewsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		a.pastReviews.CursorDown()
	case "k", "up":
		a.pastReviews.CursorUp()
	case "enter":
		sel := a.pastReviews.SelectedItem()
		if sel != nil {
			a.detail = newDetailModel(sel, a.board)
			a.detail.SetSize(a.width, a.height)
			a.prevState = statePastReviews
			a.state = stateDetail
		}
	case " ":
		a.pastReviews.ToggleCollapse()
	case "esc", "P":
		a.state = stateBoard
	case "q", "ctrl+c":
		return a.quit()
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

func resolveCurrentUser() string {
	u, err := user.Current()
	if err != nil || u == nil {
		return "unknown"
	}
	return u.Username
}
