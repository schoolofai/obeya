package agent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/agent"
)

func TestGetAgent_ClaudeCode(t *testing.T) {
	a, err := agent.Get("claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name() != "claude-code" {
		t.Errorf("expected name claude-code, got %s", a.Name())
	}
}

func TestSupportedAgents_IncludesClaudeCode(t *testing.T) {
	names := agent.SupportedNames()
	found := false
	for _, n := range names {
		if n == "claude-code" {
			found = true
		}
	}
	if !found {
		t.Error("expected claude-code in supported agents")
	}
}

func TestAppendClaudeMD_FreshFile(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")

	err := agent.AppendClaudeMDAt(claudePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- obeya:start -->") {
		t.Error("missing obeya start marker")
	}
	if !strings.Contains(content, "<!-- obeya:end -->") {
		t.Error("missing obeya end marker")
	}
	if !strings.Contains(content, "Task Tracking") {
		t.Error("missing task tracking section")
	}
}

func TestAppendClaudeMD_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")

	initial := "# My Project\n\n<!-- obeya:start --> v1\nold content\n<!-- obeya:end -->\n\n## Other stuff\n"
	os.WriteFile(claudePath, []byte(initial), 0644)

	err := agent.AppendClaudeMDAt(claudePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(claudePath)
	content := string(data)

	if strings.Contains(content, "old content") {
		t.Error("old content should have been replaced")
	}
	if !strings.Contains(content, "Other stuff") {
		t.Error("non-obeya content should be preserved")
	}
	if !strings.Contains(content, agent.ObeyaSectionVersion) {
		t.Error("should contain current version")
	}
}

func TestAppendClaudeMD_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")

	os.WriteFile(claudePath, []byte("# Existing project\n\nSome content.\n"), 0644)

	err := agent.AppendClaudeMDAt(claudePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(claudePath)
	content := string(data)

	if !strings.Contains(content, "Existing project") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, "<!-- obeya:start -->") {
		t.Error("obeya section should be appended")
	}
}

func TestAppendClaudeMD_BrokenEndMarker(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")

	broken := "# Project\n\n<!-- obeya:start --> v1\ncontent without end marker\n"
	os.WriteFile(claudePath, []byte(broken), 0644)

	err := agent.AppendClaudeMDAt(claudePath)
	if err == nil {
		t.Fatal("expected error when end marker is missing")
	}
}

func TestClaudeCodeSetup_SkipPlugin(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	a, _ := agent.Get("claude-code")
	ctx := agent.AgentContext{
		Root:       dir,
		BoardName:  "test",
		SkipPlugin: true,
	}

	err := a.Setup(ctx)
	if err != nil {
		t.Fatalf("unexpected error with skip-plugin: %v", err)
	}

	claudePath := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Error("CLAUDE.md should exist even with skip-plugin")
	}
}
