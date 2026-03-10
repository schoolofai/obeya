package agent_test

import (
	"testing"

	"github.com/niladribose/obeya/internal/agent"
)

func TestGetAgent_Unknown(t *testing.T) {
	_, err := agent.Get("unknown-agent")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}
