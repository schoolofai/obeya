package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func setupReviewTestEngine(t *testing.T) (*Engine, *domain.Board) {
	t.Helper()
	dir := t.TempDir()
	s := store.NewJSONStore(dir)
	if err := s.InitBoard("review-test", nil); err != nil {
		t.Fatal(err)
	}

	eng := New(s)
	// Set up board with users and an item via transaction
	var board *domain.Board
	if err := s.Transaction(func(b *domain.Board) error {
		aliceID := domain.GenerateID()
		b.Users[aliceID] = &domain.Identity{ID: aliceID, Name: "alice", Type: domain.IdentityHuman}
		claudeID := domain.GenerateID()
		b.Users[claudeID] = &domain.Identity{ID: claudeID, Name: "claude", Type: domain.IdentityAgent}

		item := &domain.Item{
			ID: "item-1", DisplayNum: b.NextDisplay, Title: "test task",
			Type: domain.ItemTypeTask, Status: "done",
			Assignee: claudeID, Priority: domain.PriorityMedium,
			ReviewContext: &domain.ReviewContext{Purpose: "test"},
		}
		b.Items["item-1"] = item
		b.DisplayMap[b.NextDisplay] = "item-1"
		b.NextDisplay++
		board = b
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	return eng, board
}

func TestReviewItem_Reviewed(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	if err := eng.ReviewItem("1", "reviewed", aliceID, "sess-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, _ := eng.GetItem("1")
	if item.HumanReview == nil || item.HumanReview.Status != "reviewed" {
		t.Error("expected HumanReview.Status == reviewed")
	}
	if item.HumanReview.ReviewedBy != aliceID {
		t.Errorf("ReviewedBy = %q, want %q", item.HumanReview.ReviewedBy, aliceID)
	}
}

func TestReviewItem_Hidden(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	if err := eng.ReviewItem("1", "hidden", aliceID, "sess-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, _ := eng.GetItem("1")
	if item.HumanReview == nil || item.HumanReview.Status != "hidden" {
		t.Error("expected HumanReview.Status == hidden")
	}
}

func TestReviewItem_AgentCannotReview(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	claudeID := findUserID(b, "claude")
	err := eng.ReviewItem("1", "reviewed", claudeID, "sess-1")
	if err == nil {
		t.Fatal("expected error: agents cannot review")
	}
}

func TestReviewItem_InvalidStatus(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	err := eng.ReviewItem("1", "invalid", aliceID, "sess-1")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}
