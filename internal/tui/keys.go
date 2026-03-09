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

type detailTab int

const (
	tabFields detailTab = iota
	tabPlan
	tabHistory
)
