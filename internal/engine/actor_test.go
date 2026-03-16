package engine

import (
	"testing"
)

func TestResolveActorType_Agent(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	got := resolveActorTypeFromBoard(b, "claude")
	if got != "agent" {
		t.Errorf("got %q, want agent", got)
	}
}

func TestResolveActorType_Human(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	got := resolveActorTypeFromBoard(b, "alice")
	if got != "human" {
		t.Errorf("got %q, want human", got)
	}
}

func TestResolveActorType_Unknown(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	got := resolveActorTypeFromBoard(b, "unknown-user")
	if got != "human" {
		t.Errorf("got %q, want human (default for unknown)", got)
	}
}
