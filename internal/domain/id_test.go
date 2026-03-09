package domain_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestGenerateID(t *testing.T) {
	id := domain.GenerateID()
	if len(id) != 8 {
		t.Errorf("expected ID length 8, got %d: %q", len(id), id)
	}
	// IDs should be unique
	id2 := domain.GenerateID()
	if id == id2 {
		t.Errorf("two generated IDs should not be equal: %q", id)
	}
}

func TestResolveID_ByHash(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("abc12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_ByHashPrefix(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("abc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_ByDisplayNum(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.DisplayMap[1] = "abc12345"

	resolved, err := board.ResolveID("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "abc12345" {
		t.Errorf("expected 'abc12345', got %q", resolved)
	}
}

func TestResolveID_NotFound(t *testing.T) {
	board := domain.NewBoard("test")
	_, err := board.ResolveID("xyz")
	if err == nil {
		t.Error("expected error for unknown ID, got nil")
	}
}

func TestResolveID_AmbiguousPrefix(t *testing.T) {
	board := domain.NewBoard("test")
	board.Items["abc12345"] = &domain.Item{ID: "abc12345", DisplayNum: 1}
	board.Items["abc12999"] = &domain.Item{ID: "abc12999", DisplayNum: 2}
	board.DisplayMap[1] = "abc12345"
	board.DisplayMap[2] = "abc12999"

	_, err := board.ResolveID("abc1")
	if err == nil {
		t.Error("expected ambiguous error, got nil")
	}
}

func TestBoard_ResolvePlanID(t *testing.T) {
	board := domain.NewBoard("test")
	plan := &domain.Plan{
		ID:         "plan1234",
		DisplayNum: 1,
		Title:      "Test Plan",
	}
	board.Plans["plan1234"] = plan
	board.DisplayMap[1] = "plan1234"

	// By display number
	id, err := board.ResolvePlanID("1")
	if err != nil {
		t.Fatalf("ResolvePlanID by display num failed: %v", err)
	}
	if id != "plan1234" {
		t.Errorf("expected plan1234, got %s", id)
	}

	// By exact ID
	id, err = board.ResolvePlanID("plan1234")
	if err != nil {
		t.Fatalf("ResolvePlanID by exact ID failed: %v", err)
	}
	if id != "plan1234" {
		t.Errorf("expected plan1234, got %s", id)
	}

	// By prefix
	id, err = board.ResolvePlanID("plan")
	if err != nil {
		t.Fatalf("ResolvePlanID by prefix failed: %v", err)
	}
	if id != "plan1234" {
		t.Errorf("expected plan1234, got %s", id)
	}

	// Not found
	_, err = board.ResolvePlanID("999")
	if err == nil {
		t.Error("expected error for non-existent plan")
	}
}

func TestResolveUserID_ByExactID(t *testing.T) {
	board := domain.NewBoard("test")
	board.Users["user123"] = &domain.Identity{ID: "user123", Name: "Alice"}

	resolved, err := board.ResolveUserID("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "user123" {
		t.Errorf("expected 'user123', got %q", resolved)
	}
}

func TestResolveUserID_ByName(t *testing.T) {
	board := domain.NewBoard("test")
	board.Users["user123"] = &domain.Identity{ID: "user123", Name: "Alice"}

	resolved, err := board.ResolveUserID("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "user123" {
		t.Errorf("expected 'user123', got %q", resolved)
	}
}

func TestResolveUserID_NotFound(t *testing.T) {
	board := domain.NewBoard("test")
	_, err := board.ResolveUserID("nobody")
	if err == nil {
		t.Error("expected error for unknown user, got nil")
	}
}
