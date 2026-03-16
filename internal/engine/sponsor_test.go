package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func boardWithUsers(humans []string, agents []string) *domain.Board {
	b := domain.NewBoard("test")
	for _, name := range humans {
		id := domain.GenerateID()
		b.Users[id] = &domain.Identity{ID: id, Name: name, Type: domain.IdentityHuman}
	}
	for _, name := range agents {
		id := domain.GenerateID()
		b.Users[id] = &domain.Identity{ID: id, Name: name, Type: domain.IdentityAgent}
	}
	return b
}

func findUserID(b *domain.Board, name string) string {
	for _, u := range b.Users {
		if u.Name == name {
			return u.ID
		}
	}
	return ""
}

func TestResolveSponsor_HumanCreatesItem(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	humanID := findUserID(b, "alice")
	got, err := resolveSponsor(b, humanID, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("human sponsor = %q, want empty", got)
	}
}

func TestResolveSponsor_AutoAssignSingleHuman(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	got, err := resolveSponsor(b, agentID, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	aliceID := findUserID(b, "alice")
	if got != aliceID {
		t.Errorf("sponsor = %q, want %q (alice)", got, aliceID)
	}
}

func TestResolveSponsor_Explicit(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	bobID := findUserID(b, "bob")
	got, err := resolveSponsor(b, agentID, bobID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bobID {
		t.Errorf("sponsor = %q, want %q (bob)", got, bobID)
	}
}

func TestResolveSponsor_CopiedFromParent(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	aliceID := findUserID(b, "alice")
	agentID := findUserID(b, "claude")

	// Create parent epic with sponsor
	epic := &domain.Item{ID: "epic-1", Sponsor: aliceID}
	b.Items["epic-1"] = epic
	b.DisplayMap[1] = "epic-1"

	got, err := resolveSponsor(b, agentID, "", "epic-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != aliceID {
		t.Errorf("sponsor = %q, want %q (alice, from parent)", got, aliceID)
	}
}

func TestResolveSponsor_MultipleHumansNoSponsor(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	_, err := resolveSponsor(b, agentID, "", "")
	if err == nil {
		t.Fatal("expected error for multiple humans with no sponsor")
	}
}

func TestResolveSponsor_ExplicitMustBeHuman(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude", "codex"})
	claudeID := findUserID(b, "claude")
	codexID := findUserID(b, "codex")
	_, err := resolveSponsor(b, claudeID, codexID, "")
	if err == nil {
		t.Fatal("expected error: sponsor must be human")
	}
}

func TestResolveSponsor_UnknownExplicitSponsor(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	_, err := resolveSponsor(b, agentID, "nonexistent-id", "")
	if err == nil {
		t.Fatal("expected error for unknown sponsor")
	}
}

func TestResolveSponsor_ParentExistsButNoSponsor(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	b.Items["epic-1"] = &domain.Item{ID: "epic-1", Sponsor: ""}
	_, err := resolveSponsor(b, agentID, "", "epic-1")
	if err == nil {
		t.Fatal("expected error: parent has no sponsor, multiple humans")
	}
}
