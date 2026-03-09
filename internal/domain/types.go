package domain

import (
	"fmt"
	"time"
)

type ItemType string

const (
	ItemTypeEpic  ItemType = "epic"
	ItemTypeStory ItemType = "story"
	ItemTypeTask  ItemType = "task"
)

func (t ItemType) Validate() error {
	switch t {
	case ItemTypeEpic, ItemTypeStory, ItemTypeTask:
		return nil
	default:
		return fmt.Errorf("invalid item type: %q (must be epic, story, or task)", string(t))
	}
}

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

func (p Priority) Validate() error {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return nil
	default:
		return fmt.Errorf("invalid priority: %q (must be low, medium, high, or critical)", string(p))
	}
}

type IdentityType string

const (
	IdentityHuman IdentityType = "human"
	IdentityAgent IdentityType = "agent"
)

func (t IdentityType) Validate() error {
	switch t {
	case IdentityHuman, IdentityAgent:
		return nil
	default:
		return fmt.Errorf("invalid identity type: %q (must be human or agent)", string(t))
	}
}

type Column struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
}

type ChangeRecord struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

type Identity struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Type     IdentityType `json:"type"`
	Provider string       `json:"provider"`
}

type Item struct {
	ID          string         `json:"id"`
	DisplayNum  int            `json:"display_num"`
	Type        ItemType       `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Priority    Priority       `json:"priority"`
	Assignee    string         `json:"assignee,omitempty"`
	ParentID    string         `json:"parent_id,omitempty"`
	BlockedBy   []string       `json:"blocked_by,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	History     []ChangeRecord `json:"history,omitempty"`
}

type Board struct {
	Version     int                  `json:"version"`
	Name        string               `json:"name"`
	Columns     []Column             `json:"columns"`
	Items       map[string]*Item     `json:"items"`
	DisplayMap  map[int]string       `json:"display_map"`
	NextDisplay int                  `json:"next_display"`
	Users       map[string]*Identity `json:"users"`
	AgentRole   string               `json:"agent_role"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

var defaultColumns = []string{"backlog", "todo", "in-progress", "review", "done"}

func NewBoard(name string) *Board {
	return NewBoardWithColumns(name, defaultColumns)
}

func NewBoardWithColumns(name string, columnNames []string) *Board {
	cols := make([]Column, len(columnNames))
	for i, n := range columnNames {
		cols[i] = Column{Name: n}
	}
	now := time.Now()
	return &Board{
		Version:     1,
		Name:        name,
		Columns:     cols,
		Items:       make(map[string]*Item),
		DisplayMap:  make(map[int]string),
		NextDisplay: 1,
		Users:       make(map[string]*Identity),
		AgentRole:   "admin",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// HasColumn checks if a column name exists on the board.
func (b *Board) HasColumn(name string) bool {
	for _, c := range b.Columns {
		if c.Name == name {
			return true
		}
	}
	return false
}
