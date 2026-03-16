package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestResolveDownstream_NoBlockers(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a"}
	got := ResolveDownstream("a", b)
	if len(got) != 0 {
		t.Errorf("got %d downstream, want 0", len(got))
	}
}

func TestResolveDownstream_FindsBlockedItems(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", DisplayNum: 1}
	b.Items["b"] = &domain.Item{ID: "b", DisplayNum: 2, BlockedBy: []string{"a"}}
	b.Items["c"] = &domain.Item{ID: "c", DisplayNum: 3, BlockedBy: []string{"a"}}
	b.Items["d"] = &domain.Item{ID: "d", DisplayNum: 4, BlockedBy: []string{"b"}}

	got := ResolveDownstream("a", b)
	if len(got) != 2 {
		t.Fatalf("got %d downstream, want 2", len(got))
	}
	// Should contain b and c but not d (d is blocked by b, not a)
	ids := map[string]bool{}
	for _, id := range got {
		ids[id] = true
	}
	if !ids["b"] || !ids["c"] {
		t.Errorf("expected b and c, got %v", got)
	}
}
