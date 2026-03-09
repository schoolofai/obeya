package tui

import "github.com/niladribose/obeya/internal/domain"

type boardLoadedMsg struct {
	board *domain.Board
}

type errMsg struct {
	err error
}
