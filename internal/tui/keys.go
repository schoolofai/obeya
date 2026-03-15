package tui

type viewState int

const (
	stateBoard viewState = iota
	stateDetail
	statePicker
	stateInput
	stateConfirm
	stateDashboard
	stateDAG
)

type pickerKind int

const (
	pickerColumn pickerKind = iota
	pickerUser
	pickerItem
	pickerType
	pickerEpic
)

type detailTab int

const (
	tabFields detailTab = iota
	tabPlan
	tabHistory
)
