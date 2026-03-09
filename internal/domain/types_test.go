package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestItemTypeValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"epic", true},
		{"story", true},
		{"task", true},
		{"bug", false},
		{"", false},
	}
	for _, tt := range tests {
		err := domain.ItemType(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("ItemType(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ItemType(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestPriorityValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"critical", true},
		{"urgent", false},
		{"", false},
	}
	for _, tt := range tests {
		err := domain.Priority(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("Priority(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("Priority(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestIdentityTypeValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"human", true},
		{"agent", true},
		{"bot", false},
	}
	for _, tt := range tests {
		err := domain.IdentityType(tt.input).Validate()
		if tt.valid && err != nil {
			t.Errorf("IdentityType(%q) should be valid, got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("IdentityType(%q) should be invalid, got nil error", tt.input)
		}
	}
}

func TestNewBoardDefaults(t *testing.T) {
	board := domain.NewBoard("test-board")
	if board.Name != "test-board" {
		t.Errorf("expected name 'test-board', got %q", board.Name)
	}
	if len(board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(board.Columns))
	}
	expectedCols := []string{"backlog", "todo", "in-progress", "review", "done"}
	for i, col := range board.Columns {
		if col.Name != expectedCols[i] {
			t.Errorf("column %d: expected %q, got %q", i, expectedCols[i], col.Name)
		}
	}
	if board.Version != 1 {
		t.Errorf("expected version 1, got %d", board.Version)
	}
	if board.NextDisplay != 1 {
		t.Errorf("expected NextDisplay 1, got %d", board.NextDisplay)
	}
	if board.AgentRole != "admin" {
		t.Errorf("expected AgentRole 'admin', got %q", board.AgentRole)
	}
}

func TestNewBoardCustomColumns(t *testing.T) {
	board := domain.NewBoardWithColumns("custom", []string{"todo", "doing", "done"})
	if len(board.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(board.Columns))
	}
}

func TestBoardHasColumn(t *testing.T) {
	board := domain.NewBoard("test")
	if !board.HasColumn("backlog") {
		t.Error("expected HasColumn('backlog') to be true")
	}
	if board.HasColumn("nonexistent") {
		t.Error("expected HasColumn('nonexistent') to be false")
	}
}
