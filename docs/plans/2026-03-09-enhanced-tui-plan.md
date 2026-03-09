# Enhanced TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rebuild `ob tui` into an interactive Trello-style Kanban board with color-coded cards, epic grouping, detail overlays, picker modals, and inline quick actions.

**Architecture:** Component-based Bubble Tea TUI with a root App model managing state transitions between Board view, Detail overlay, Picker modals, and Text Input. Each component is a separate file implementing its own Update/View cycle.

**Tech Stack:** Go, Bubble Tea (TUI framework), Lipgloss (styling), Bubbles (text input component)

**Design Doc:** `docs/plans/2026-03-09-enhanced-tui-design.md`

**Obeya Board:** Epic #1, Stories #2-#8

---

## Task 1: Styles and Key Bindings Foundation

**Obeya Story:** #2 (Refactor TUI architecture)

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/keys.go`

**Step 1: Install bubbles dependency**

Run: `go get github.com/charmbracelet/bubbles@latest`
Expected: Added to go.mod

**Step 2: Create styles.go**

Create `internal/tui/styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Card styles
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(22)

	selectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("6")). // Cyan
				Padding(0, 1).
				Width(22)

	// Type colors
	epicStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))  // Purple
	storyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))  // Blue
	taskStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))  // White

	// Priority indicators
	priCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("●●●") // Red
	priHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("●●")  // Red
	priMedium   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●●")  // Yellow
	priLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("●")   // Green

	// Status
	blockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // Red bold
	assigneeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true) // Dim cyan

	// Column headers
	activeColHeader   = lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveColHeader = lipgloss.NewStyle().Faint(true)

	// Column container
	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")). // Gray
			Padding(0, 0).
			MarginRight(1)

	activeColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("6")). // Cyan
				Padding(0, 0).
				MarginRight(1)

	// Overlay
	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(1, 2).
			Width(50)

	// Help bar
	helpStyle = lipgloss.NewStyle().Faint(true)

	// Epic group header
	epicGroupStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true)
)

func priorityIndicator(pri string) string {
	switch pri {
	case "critical":
		return priCritical
	case "high":
		return priHigh
	case "medium":
		return priMedium
	case "low":
		return priLow
	default:
		return priMedium
	}
}

func typeStyle(itemType string) lipgloss.Style {
	switch itemType {
	case "epic":
		return epicStyle
	case "story":
		return storyStyle
	default:
		return taskStyle
	}
}
```

**Step 3: Create keys.go**

Create `internal/tui/keys.go`:

```go
package tui

type viewState int

const (
	stateBoard viewState = iota
	stateDetail
	statePicker
	stateInput
	stateConfirm
)

type pickerKind int

const (
	pickerColumn pickerKind = iota
	pickerUser
	pickerItem
	pickerType
)
```

**Step 4: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 5: Commit and update board**

```bash
git add internal/tui/styles.go internal/tui/keys.go go.mod go.sum
git commit -m "feat(tui): add Lipgloss styles and state/key definitions"
```

Run: `~/bin/ob move 2 in-progress`

---

## Task 2: App Root Model — State Machine

**Obeya Story:** #2 (Refactor TUI architecture)

**Files:**
- Create: `internal/tui/app.go`
- Modify: `internal/tui/model.go` — refactor into app.go, keep model.go as data helpers
- Modify: `cmd/tui.go` — use new App model

**Step 1: Create app.go with state machine**

Create `internal/tui/app.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
)

type App struct {
	engine *engine.Engine
	board  *domain.Board

	// Board navigation
	columns    []string
	cursorCol  int
	cursorRow  int
	collapsed  map[string]bool // epicID -> collapsed

	// State machine
	state      viewState
	prevState  viewState

	// Sub-components
	detail     DetailModel
	picker     PickerModel
	input      InputModel
	confirmMsg string

	// Dimensions
	width  int
	height int

	err error
}

func NewApp(eng *engine.Engine) App {
	return App{
		engine:    eng,
		collapsed: make(map[string]bool),
		state:     stateBoard,
	}
}

func (a App) Init() tea.Cmd {
	return a.loadBoard()
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
		return a, nil
	case errMsg:
		a.err = msg.err
		return a, nil
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
		return a.renderBoardWithOverlay(a.detail.View())
	case statePicker:
		return a.renderBoardWithOverlay(a.picker.View())
	case stateInput:
		return a.renderBoardWithOverlay(a.input.View())
	case stateConfirm:
		return a.renderBoardWithOverlay(a.renderConfirm())
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

func extractColumns(board *domain.Board) []string {
	cols := make([]string, len(board.Columns))
	for i, c := range board.Columns {
		cols[i] = c.Name
	}
	return cols
}
```

**Step 2: Update cmd/tui.go to use App**

Modify `cmd/tui.go` — change `tui.New(eng)` to `tui.NewApp(eng)`.

**Step 3: Build and verify**

Run: `go build -o ob .`
Expected: May have compilation errors from missing methods — that's OK, we'll add them in subsequent tasks.

**Step 4: Commit**

```bash
git add internal/tui/app.go cmd/tui.go
git commit -m "feat(tui): add App root model with state machine"
```

---

## Task 3: Board View — Trello-style Columns with Styled Cards

**Obeya Story:** #3 (Trello-style column board)

**Files:**
- Create: `internal/tui/board.go`
- Remove old logic from: `internal/tui/view.go` (replace with thin wrapper)

**Step 1: Create board.go with column rendering**

Create `internal/tui/board.go`:

```go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

func (a App) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "h", "left":
		a.cursorCol = max(0, a.cursorCol-1)
		a.cursorRow = 0
	case "l", "right":
		a.cursorCol = min(len(a.columns)-1, a.cursorCol+1)
		a.cursorRow = 0
	case "j", "down":
		items := a.visibleItemsInColumn(a.cursorCol)
		a.cursorRow = min(len(items)-1, a.cursorRow+1)
	case "k", "up":
		a.cursorRow = max(0, a.cursorRow-1)
	case "tab":
		a.cursorCol = (a.cursorCol + 1) % len(a.columns)
		a.cursorRow = 0
	case "enter":
		if item := a.selectedItem(); item != nil {
			a.detail = NewDetailModel(item, a.board)
			a.state = stateDetail
		}
	case "m":
		if item := a.selectedItem(); item != nil {
			a.picker = NewPickerModel(pickerColumn, a.columns, fmt.Sprintf("Move #%d to:", item.DisplayNum))
			a.state = statePicker
			a.prevState = stateBoard
		}
	case "a":
		if item := a.selectedItem(); item != nil {
			users := userNames(a.board)
			a.picker = NewPickerModel(pickerUser, users, fmt.Sprintf("Assign #%d to:", item.DisplayNum))
			a.state = statePicker
			a.prevState = stateBoard
		}
	case "c":
		types := []string{"epic", "story", "task"}
		a.picker = NewPickerModel(pickerType, types, "Create item type:")
		a.state = statePicker
		a.prevState = stateBoard
	case "d":
		if item := a.selectedItem(); item != nil {
			a.confirmMsg = fmt.Sprintf("Delete #%d %q? (y/n)", item.DisplayNum, item.Title)
			a.state = stateConfirm
		}
	case "b":
		if item := a.selectedItem(); item != nil {
			itemLabels := itemPickerLabels(a.board, item.ID)
			a.picker = NewPickerModel(pickerItem, itemLabels, fmt.Sprintf("Block #%d by:", item.DisplayNum))
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
		if item := a.selectedItem(); item != nil && item.Type == "epic" {
			a.collapsed[item.ID] = !a.collapsed[item.ID]
		}
	case "/":
		a.input = NewInputModel("Search:", "")
		a.state = stateInput
	case "r":
		a.err = nil
		return a, a.loadBoard()
	case "?":
		// Could show help overlay — for now just a no-op
	}
	return a, nil
}

func (a App) renderBoard() string {
	var sb strings.Builder

	// Header
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("  Obeya Board: %s", a.board.Name))
	sb.WriteString(title + "\n\n")

	// Render columns side by side
	colViews := make([]string, len(a.columns))
	for i, colName := range a.columns {
		colViews[i] = a.renderColumn(i, colName)
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, colViews...))

	// Help bar
	sb.WriteString("\n")
	help := "h/l:columns  j/k:items  m:move  a:assign  c:create  d:delete  p:priority  Enter:detail  Space:collapse  /:search  r:reload  q:quit"
	sb.WriteString(helpStyle.Render(help))

	return sb.String()
}

func (a App) renderColumn(colIdx int, colName string) string {
	isActive := colIdx == a.cursorCol
	items := a.visibleItemsInColumn(colIdx)

	// Header
	var header string
	if isActive {
		header = activeColHeader.Render(fmt.Sprintf(" %s ", strings.ToUpper(colName)))
	} else {
		header = inactiveColHeader.Render(fmt.Sprintf(" %s ", strings.ToUpper(colName)))
	}

	// Cards
	var cards []string
	cards = append(cards, header)
	cards = append(cards, "")

	for rowIdx, item := range items {
		selected := isActive && rowIdx == a.cursorRow
		card := a.renderCard(item, selected)
		cards = append(cards, card)
	}

	// Item count
	count := helpStyle.Render(fmt.Sprintf(" %d items", len(items)))
	cards = append(cards, "", count)

	content := strings.Join(cards, "\n")

	colW := a.columnWidth()
	if isActive {
		return activeColumnStyle.Width(colW).Render(content)
	}
	return columnStyle.Width(colW).Render(content)
}

func (a App) renderCard(item *domain.Item, selected bool) string {
	// Title line
	titleLine := fmt.Sprintf("#%d %s", item.DisplayNum, truncate(item.Title, 18))

	// Type + priority line
	typePri := fmt.Sprintf("%s %s", typeStyle(string(item.Type)).Render(string(item.Type)), priorityIndicator(string(item.Priority)))

	// Parent badge (if item's parent is in a different column)
	parentBadge := a.parentBadge(item)

	// Assignee + blocked
	var bottomLine string
	if item.Assignee != "" {
		name := resolveUserName(a.board, item.Assignee)
		bottomLine = assigneeStyle.Render("@" + name)
	}
	if len(item.BlockedBy) > 0 {
		blocked := blockedStyle.Render("[!]")
		if bottomLine != "" {
			bottomLine += "  " + blocked
		} else {
			bottomLine = blocked
		}
	}

	var lines []string
	lines = append(lines, titleLine, typePri)
	if parentBadge != "" {
		lines = append(lines, parentBadge)
	}
	if bottomLine != "" {
		lines = append(lines, bottomLine)
	}

	content := strings.Join(lines, "\n")
	if selected {
		return selectedCardStyle.Render(content)
	}
	return cardStyle.Render(content)
}

func (a App) renderBoardWithOverlay(overlay string) string {
	board := a.renderBoard()
	return lipgloss.Place(a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
	_ = board // overlay replaces board for now
}

func (a App) renderConfirm() string {
	return overlayStyle.Render(a.confirmMsg)
}

func (a App) visibleItemsInColumn(colIdx int) []*domain.Item {
	if a.board == nil || colIdx >= len(a.columns) {
		return nil
	}
	colName := a.columns[colIdx]
	var items []*domain.Item
	for _, item := range a.board.Items {
		if item.Status != colName {
			continue
		}
		// Skip children of collapsed epics
		if a.isCollapsedChild(item) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayNum < items[j].DisplayNum
	})
	return items
}

func (a App) isCollapsedChild(item *domain.Item) bool {
	if item.ParentID == "" {
		return false
	}
	// Walk up parent chain
	parent, ok := a.board.Items[item.ParentID]
	if !ok {
		return false
	}
	if parent.Type == "epic" && a.collapsed[parent.ID] {
		return true
	}
	// Check grandparent (task -> story -> epic)
	if parent.ParentID != "" {
		grandparent, ok := a.board.Items[parent.ParentID]
		if ok && grandparent.Type == "epic" && a.collapsed[grandparent.ID] {
			return true
		}
	}
	return false
}

func (a App) parentBadge(item *domain.Item) string {
	if item.ParentID == "" {
		return ""
	}
	parent, ok := a.board.Items[item.ParentID]
	if !ok {
		return ""
	}
	// Show badge if parent is in different column or is an epic in same column
	if parent.Status != item.Status || parent.Type == "epic" {
		badge := fmt.Sprintf("↑ #%d %s", parent.DisplayNum, truncate(parent.Title, 12))
		return lipgloss.NewStyle().Faint(true).Render(badge)
	}
	return ""
}

func (a App) columnWidth() int {
	if a.width == 0 {
		return 24
	}
	w := (a.width - 2) / len(a.columns)
	if w < 20 {
		return 20
	}
	if w > 30 {
		return 30
	}
	return w
}

func cyclePriority(current string) string {
	switch current {
	case "low":
		return "medium"
	case "medium":
		return "high"
	case "high":
		return "critical"
	case "critical":
		return "low"
	default:
		return "medium"
	}
}

func userNames(board *domain.Board) []string {
	var names []string
	for _, u := range board.Users {
		names = append(names, fmt.Sprintf("%s (%s)", u.Name, u.ID[:6]))
	}
	return names
}

func itemPickerLabels(board *domain.Board, excludeID string) []string {
	var labels []string
	for _, item := range board.Items {
		if item.ID == excludeID {
			continue
		}
		labels = append(labels, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
	}
	return labels
}

func resolveUserName(board *domain.Board, userID string) string {
	if u, ok := board.Users[userID]; ok {
		return u.Name
	}
	if len(userID) > 6 {
		return userID[:6]
	}
	return userID
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 2: Replace view.go with thin wrapper**

Replace `internal/tui/view.go` contents — remove all old rendering logic, keep only `truncate` (now used by board.go) and remove the old `View()` method since App.View() handles it.

Actually, since App has its own View(), we should delete the old Model's View() entirely. The old `model.go` and `view.go` will be replaced by the new component files. Keep `model.go` only for the message types (`boardLoadedMsg`, `errMsg`).

Rewrite `internal/tui/model.go` to only contain shared message types:

```go
package tui

import "github.com/niladribose/obeya/internal/domain"

type boardLoadedMsg struct {
	board *domain.Board
}

type errMsg struct {
	err error
}
```

Delete `internal/tui/view.go` or empty it — all rendering is now in `board.go`.

**Step 3: Build and verify**

Run: `go build -o ob .`
Expected: Builds (some methods may be stubs — fill in next tasks)

**Step 4: Commit and update board**

```bash
git add internal/tui/
git commit -m "feat(tui): add Trello-style column board with styled cards"
```

Run: `~/bin/ob move 3 in-progress`

---

## Task 4: Detail Overlay Panel

**Obeya Story:** #5 (Detail overlay panel)

**Files:**
- Create: `internal/tui/detail.go`

**Step 1: Create detail.go**

Create `internal/tui/detail.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

type DetailModel struct {
	item  *domain.Item
	board *domain.Board
}

func NewDetailModel(item *domain.Item, board *domain.Board) DetailModel {
	return DetailModel{item: item, board: board}
}

func (d DetailModel) View() string {
	item := d.item
	var sb strings.Builder

	// Title
	title := fmt.Sprintf("#%d %s", item.DisplayNum, item.Title)
	sb.WriteString(typeStyle(string(item.Type)).Bold(true).Render(title) + "\n\n")

	// Fields
	sb.WriteString(fmt.Sprintf("  Type:     %s\n", item.Type))
	sb.WriteString(fmt.Sprintf("  Status:   %s\n", item.Status))
	sb.WriteString(fmt.Sprintf("  Priority: %s %s\n", priorityIndicator(string(item.Priority)), item.Priority))

	if item.Assignee != "" {
		name := resolveUserName(d.board, item.Assignee)
		sb.WriteString(fmt.Sprintf("  Assignee: %s\n", assigneeStyle.Render("@"+name)))
	}

	if item.ParentID != "" {
		if parent, ok := d.board.Items[item.ParentID]; ok {
			sb.WriteString(fmt.Sprintf("  Parent:   #%d %s\n", parent.DisplayNum, parent.Title))
		}
	}

	if item.Description != "" {
		sb.WriteString(fmt.Sprintf("  Desc:     %s\n", item.Description))
	}

	if len(item.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("  Tags:     %s\n", strings.Join(item.Tags, ", ")))
	}

	if len(item.BlockedBy) > 0 {
		var blockers []string
		for _, bid := range item.BlockedBy {
			if bi, ok := d.board.Items[bid]; ok {
				blockers = append(blockers, fmt.Sprintf("#%d", bi.DisplayNum))
			}
		}
		sb.WriteString(fmt.Sprintf("  Blocked:  %s\n", blockedStyle.Render(strings.Join(blockers, ", "))))
	}

	// Children
	children := findChildren(d.board, item.ID)
	if len(children) > 0 {
		sb.WriteString("\n  Children:\n")
		for _, ch := range children {
			sb.WriteString(fmt.Sprintf("    #%-4d %-8s %-12s %s\n", ch.DisplayNum, ch.Type, ch.Status, ch.Title))
		}
	}

	// History (last 5)
	if len(item.History) > 0 {
		sb.WriteString("\n  History:\n")
		start := len(item.History) - 5
		if start < 0 {
			start = 0
		}
		for _, h := range item.History[start:] {
			sb.WriteString(fmt.Sprintf("    %s  %s\n", h.Timestamp.Format("15:04"), h.Detail))
		}
	}

	// Action hints
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  [m]ove  [a]ssign  [p]riority  [Esc]close"))

	return overlayStyle.Render(sb.String())
}

func findChildren(board *domain.Board, parentID string) []*domain.Item {
	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children
}
```

**Step 2: Add detail key handling to app.go**

Add `handleDetailKey` method to `internal/tui/app.go`:

```go
func (a App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = stateBoard
	case "m":
		item := a.detail.item
		a.picker = NewPickerModel(pickerColumn, a.columns, fmt.Sprintf("Move #%d to:", item.DisplayNum))
		a.state = statePicker
		a.prevState = stateDetail
	case "a":
		item := a.detail.item
		users := userNames(a.board)
		a.picker = NewPickerModel(pickerUser, users, fmt.Sprintf("Assign #%d to:", item.DisplayNum))
		a.state = statePicker
		a.prevState = stateDetail
	case "p":
		item := a.detail.item
		nextPri := cyclePriority(string(item.Priority))
		_ = a.engine.EditItem(item.ID, "", "", nextPri, "", "")
		return a, a.loadBoard()
	case "q":
		return a, tea.Quit
	}
	return a, nil
}
```

**Step 3: Build and verify**

Run: `go build -o ob .`
Expected: Builds successfully

**Step 4: Commit and update board**

```bash
git add internal/tui/detail.go internal/tui/app.go
git commit -m "feat(tui): add detail overlay panel with item info and history"
```

Run: `~/bin/ob move 5 in-progress`

---

## Task 5: Picker Modal

**Obeya Story:** #6 (Picker and input modals)

**Files:**
- Create: `internal/tui/picker.go`

**Step 1: Create picker.go**

Create `internal/tui/picker.go`:

```go
package tui

import (
	"fmt"
	"strings"
)

type PickerModel struct {
	kind    pickerKind
	title   string
	options []string
	cursor  int
}

func NewPickerModel(kind pickerKind, options []string, title string) PickerModel {
	return PickerModel{kind: kind, options: options, title: title}
}

func (p PickerModel) View() string {
	var sb strings.Builder
	sb.WriteString(p.title + "\n\n")

	for i, opt := range p.options {
		cursor := "  "
		if i == p.cursor {
			cursor = "▸ "
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("Enter:select  Esc:cancel"))

	return overlayStyle.Render(sb.String())
}

func (p *PickerModel) MoveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *PickerModel) MoveDown() {
	if p.cursor < len(p.options)-1 {
		p.cursor++
	}
}

func (p PickerModel) Selected() string {
	if p.cursor < len(p.options) {
		return p.options[p.cursor]
	}
	return ""
}
```

**Step 2: Add picker key handling to app.go**

Add `handlePickerKey` method:

```go
func (a App) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = stateBoard
		return a, nil
	case "j", "down":
		a.picker.MoveDown()
	case "k", "up":
		a.picker.MoveUp()
	case "enter":
		return a.executePickerSelection()
	case "q":
		return a, tea.Quit
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
			// Extract user ID from "Name (abc123)" format
			userID := extractUserID(selected)
			_ = a.engine.AssignItem(item.ID, userID, "", "")
		}
	case pickerType:
		// Transition to text input for title
		a.input = NewInputModel(fmt.Sprintf("New %s title:", selected), selected)
		a.state = stateInput
		return a, nil
	case pickerItem:
		if item != nil && selected != "" {
			blockerNum := extractItemNum(selected)
			_ = a.engine.BlockItem(item.ID, blockerNum, "", "")
		}
	}

	a.state = stateBoard
	return a, a.loadBoard()
}

func extractUserID(display string) string {
	// Format: "Name (abc123)"
	idx := strings.LastIndex(display, "(")
	if idx < 0 {
		return display
	}
	return strings.TrimSuffix(display[idx+1:], ")")
}

func extractItemNum(display string) string {
	// Format: "#3 Some Title"
	parts := strings.SplitN(display, " ", 2)
	if len(parts) > 0 {
		return strings.TrimPrefix(parts[0], "#")
	}
	return display
}
```

**Step 3: Build and verify**

Run: `go build -o ob .`

**Step 4: Commit and update board**

```bash
git add internal/tui/picker.go internal/tui/app.go
git commit -m "feat(tui): add reusable picker modal for columns, users, items, types"
```

Run: `~/bin/ob move 6 in-progress`

---

## Task 6: Text Input Modal

**Obeya Story:** #6 (Picker and input modals)

**Files:**
- Create: `internal/tui/input.go`

**Step 1: Create input.go**

Create `internal/tui/input.go`:

```go
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
)

type InputModel struct {
	title    string
	context  string // e.g. item type for create
	input    textinput.Model
}

func NewInputModel(title, context string) InputModel {
	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	return InputModel{title: title, context: context, input: ti}
}

func (m InputModel) View() string {
	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		m.title,
		m.input.View(),
		helpStyle.Render("Enter:confirm  Esc:cancel"),
	)
	return overlayStyle.Render(content)
}

func (m InputModel) Value() string {
	return m.input.Value()
}
```

**Step 2: Add input key handling to app.go**

Add `handleInputKey` method and `handleConfirmKey`:

```go
func (a App) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = stateBoard
		return a, nil
	case "enter":
		value := a.input.Value()
		if value == "" {
			return a, nil
		}
		return a.executeInput(value)
	default:
		// Forward to text input
		var cmd tea.Cmd
		a.input.input, cmd = a.input.input.Update(msg)
		return a, cmd
	}
}

func (a App) executeInput(value string) (tea.Model, tea.Cmd) {
	context := a.input.context

	switch {
	case context == "epic" || context == "story" || context == "task":
		// Create item
		parentRef := ""
		if context != "epic" {
			if item := a.selectedItem(); item != nil {
				parentRef = item.ID
			}
		}
		_, _ = a.engine.CreateItem(context, value, parentRef, "", "medium", "", nil)
	default:
		// Search filter — for now just reload (search implementation is Task 7)
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
```

**Step 3: Build and verify**

Run: `go build -o ob .`

**Step 4: Commit and update board**

```bash
git add internal/tui/input.go internal/tui/app.go
git commit -m "feat(tui): add text input modal and confirm dialog"
```

---

## Task 7: Epic Grouping with Collapse/Expand

**Obeya Story:** #4 (Epic grouping)

**Files:**
- Modify: `internal/tui/board.go` — add epic group headers to renderColumn

**Step 1: Add epic group rendering**

Add to `board.go` a method that groups items by epic before rendering in `renderColumn`. Between the column header and the card list, insert epic group headers:

```go
func (a App) renderGroupedCards(items []*domain.Item, colIdx int) []string {
	var cards []string

	// Separate epics and non-epic items
	epicItems := make(map[string][]*domain.Item) // epicID -> children in this column
	var orphans []*domain.Item
	var epicsInCol []*domain.Item

	for _, item := range items {
		if item.Type == "epic" {
			epicsInCol = append(epicsInCol, item)
			continue
		}
		epicID := findEpicAncestor(a.board, item)
		if epicID != "" {
			epicItems[epicID] = append(epicItems[epicID], item)
		} else {
			orphans = append(orphans, item)
		}
	}

	// Render each epic group
	for _, epic := range epicsInCol {
		collapsed := a.collapsed[epic.ID]
		children := epicItems[epic.ID]
		childCount := len(children)

		if collapsed {
			header := fmt.Sprintf("▶ #%d %s (%d items)", epic.DisplayNum, truncate(epic.Title, 12), childCount)
			cards = append(cards, epicGroupStyle.Render(header))
		} else {
			header := fmt.Sprintf("▼ #%d %s", epic.DisplayNum, truncate(epic.Title, 14))
			cards = append(cards, epicGroupStyle.Render(header))
			// Render epic card itself
			selected := colIdx == a.cursorCol && a.isItemAtCursor(epic)
			cards = append(cards, a.renderCard(epic, selected))
			// Render children
			for _, child := range children {
				selected := colIdx == a.cursorCol && a.isItemAtCursor(child)
				cards = append(cards, a.renderCard(child, selected))
			}
		}
	}

	// Render items belonging to epics in other columns (with parent badge)
	for epicID, children := range epicItems {
		alreadyRendered := false
		for _, e := range epicsInCol {
			if e.ID == epicID {
				alreadyRendered = true
				break
			}
		}
		if !alreadyRendered {
			for _, child := range children {
				selected := colIdx == a.cursorCol && a.isItemAtCursor(child)
				cards = append(cards, a.renderCard(child, selected))
			}
		}
	}

	// Render orphans
	for _, item := range orphans {
		selected := colIdx == a.cursorCol && a.isItemAtCursor(item)
		cards = append(cards, a.renderCard(item, selected))
	}

	return cards
}

func (a App) isItemAtCursor(item *domain.Item) bool {
	items := a.visibleItemsInColumn(a.cursorCol)
	if a.cursorRow < len(items) {
		return items[a.cursorRow].ID == item.ID
	}
	return false
}

func findEpicAncestor(board *domain.Board, item *domain.Item) string {
	current := item
	for current.ParentID != "" {
		parent, ok := board.Items[current.ParentID]
		if !ok {
			break
		}
		if parent.Type == "epic" {
			return parent.ID
		}
		current = parent
	}
	return ""
}
```

Update `renderColumn` to use `renderGroupedCards` instead of iterating items directly.

**Step 2: Build and verify**

Run: `go build -o ob .`

**Step 3: Commit and update board**

```bash
git add internal/tui/board.go
git commit -m "feat(tui): add epic grouping with collapse/expand"
```

Run: `~/bin/ob move 4 in-progress`

---

## Task 8: Wire Everything Together and Test

**Obeya Story:** #7 (Quick action key bindings)

**Files:**
- Modify: `internal/tui/app.go` — ensure all state transitions compile
- Delete: `internal/tui/model.go` (old Model) — if not needed
- Delete: `internal/tui/view.go` (old View) — replaced by board.go

**Step 1: Clean up old files**

Remove the old `Model` struct from `model.go`. Keep only the message types. If `view.go` still exists with old code, delete it or replace with a comment redirecting to board.go.

**Step 2: Ensure all imports resolve**

Run: `go build -o ob .`
Fix any compilation errors — missing imports, undefined methods, type mismatches.

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All existing tests pass (TUI has no tests — it's visual)

**Step 4: Manual TUI test**

```bash
cd /tmp && rm -rf tui-test && mkdir tui-test && cd tui-test
~/bin/ob init test-board
~/bin/ob user add "Dev" --type human
~/bin/ob create epic "Auth System" --priority high
~/bin/ob create story "Login" -p 1
~/bin/ob create task "Form" -p 2 --priority medium --tag frontend
~/bin/ob create task "JWT" -p 2 --priority high --tag backend
~/bin/ob move 3 in-progress
~/bin/ob tui
```

In the TUI, verify:
- Columns render with borders and colors
- Cards show type, priority, parent badge
- h/l moves between columns, j/k between items
- Enter shows detail overlay
- m opens column picker, select moves item
- c opens type picker → title input → creates item
- Space collapses/expands epics
- p cycles priority
- d shows confirm, y deletes
- Esc closes overlays
- q quits

**Step 5: Commit and update board**

```bash
cd /Users/niladribose/code/obeya
git add internal/tui/
git commit -m "feat(tui): wire all components, clean up old model"
```

Run:
```bash
~/bin/ob move 2 done
~/bin/ob move 3 done
~/bin/ob move 4 done
~/bin/ob move 5 done
~/bin/ob move 6 done
~/bin/ob move 7 done
~/bin/ob move 1 done
```

---

## Summary

| Task | Story | Description | Files |
|------|-------|-------------|-------|
| 1 | #2 | Styles + key definitions | styles.go, keys.go |
| 2 | #2 | App root model + state machine | app.go, cmd/tui.go |
| 3 | #3 | Board view — Trello columns + cards | board.go, model.go rewrite |
| 4 | #5 | Detail overlay panel | detail.go |
| 5 | #6 | Picker modal | picker.go |
| 6 | #6 | Text input modal + confirm | input.go |
| 7 | #4 | Epic grouping + collapse | board.go update |
| 8 | #7 | Wire together + test | cleanup + manual test |

**Total: 8 tasks**

**Parallelizable groups:**
- Tasks 1-2 (foundation) — sequential
- Tasks 3-6 (components) — can be parallel after Task 2, but share app.go
- Tasks 7-8 (integration) — sequential, after all above

**Note:** Story #8 (search/filter) is intentionally left out — it's low priority and can be a follow-up. The `/` key binding is wired but the filter logic is a stub.
