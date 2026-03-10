package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestPrintItemChildrenVerbose(t *testing.T) {
	children := []*domain.Item{
		{
			DisplayNum:  3,
			Type:        domain.ItemTypeTask,
			Status:      "in-progress",
			Title:       "Fix login bug",
			Priority:    domain.PriorityHigh,
			Assignee:    "niladri",
			BlockedBy:   []string{"#5"},
			Description: "This is a short description",
		},
		{
			DisplayNum:  4,
			Type:        domain.ItemTypeTask,
			Status:      "backlog",
			Title:       "Add tests",
			Priority:    domain.PriorityMedium,
			Assignee:    "",
			BlockedBy:   nil,
			Description: "Write unit tests for the auth module that cover all edge cases and ensure full coverage of the login flow",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printChildrenVerbose(children)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check child #3 has priority, assignee, and blocked-by on same line
	if !strings.Contains(output, "Priority: high | Assignee: niladri | Blocked by: #5") {
		t.Errorf("expected pipe-delimited detail line with blocked-by, got:\n%s", output)
	}

	// Check child #4 has dash for missing assignee, no blocked-by
	if !strings.Contains(output, "Priority: medium | Assignee: —") {
		t.Errorf("expected 'Priority: medium | Assignee: —' for unset assignee, got:\n%s", output)
	}

	// Check description truncation (80 chars + ...)
	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated description with '...', got:\n%s", output)
	}

	// Check short description is NOT truncated
	if !strings.Contains(output, "This is a short description") {
		t.Errorf("expected full short description, got:\n%s", output)
	}
}
