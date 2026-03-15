package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildOb(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "ob")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = mustGitRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return bin
}

func mustGitRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestInit_RequiresAgentFlag(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when --agent not provided")
	}
	if !strings.Contains(string(out), "required flag --agent") {
		t.Errorf("expected agent requirement message, got: %s", out)
	}
}

func TestInit_RejectsUnknownAgent(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--agent", "vim")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(string(out), "unsupported agent") {
		t.Errorf("expected unsupported agent message, got: %s", out)
	}
}

func TestInit_RegistersBothUsers(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %s", out)
	}

	output := string(out)
	if !strings.Contains(output, "Registered agent user: Claude") {
		t.Errorf("expected agent registration message, got: %s", output)
	}
	if !strings.Contains(output, "Registered human user:") {
		t.Errorf("expected human registration message, got: %s", output)
	}

	// Verify users via ob user list --format json
	listCmd := exec.Command(bin, "user", "list", "--format", "json")
	listCmd.Dir = dir
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("user list failed: %s", listOut)
	}

	var users map[string]struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(listOut, &users); err != nil {
		t.Fatalf("failed to parse user list JSON: %v\n%s", err, listOut)
	}

	var hasAgent, hasHuman bool
	for _, u := range users {
		if u.Name == "Claude" && u.Type == "agent" && u.Provider == "claude-code" {
			hasAgent = true
		}
		if u.Type == "human" {
			hasHuman = true
		}
	}
	if !hasAgent {
		t.Errorf("expected agent user 'Claude' with type=agent, provider=claude-code in user list: %s", listOut)
	}
	if !hasHuman {
		t.Errorf("expected a human user in user list: %s", listOut)
	}
}

func TestInit_IdempotentUsers(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	// First init
	cmd := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("first init failed: %s", out)
	}

	// Second init (board already exists, but users should not duplicate)
	cmd2 := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd2.Dir = dir
	if out, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("second init failed: %s", out)
	}

	// Count users — should be exactly 2 (not 4)
	listCmd := exec.Command(bin, "user", "list", "--format", "json")
	listCmd.Dir = dir
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("user list failed: %s", listOut)
	}

	var users map[string]json.RawMessage
	if err := json.Unmarshal(listOut, &users); err != nil {
		t.Fatalf("failed to parse user list JSON: %v\n%s", err, listOut)
	}
	if len(users) != 2 {
		t.Errorf("expected exactly 2 users after double init, got %d: %s", len(users), listOut)
	}
}

func TestInit_SharedRequiresAgent(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--shared", "test-board")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when --shared without --agent")
	}
	if !strings.Contains(string(out), "--shared requires --agent") {
		t.Errorf("expected '--shared requires --agent' message, got: %s", out)
	}
}

func TestInit_SharedAndAgentCreatesSharedBoardWithSetup(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	boardName := "test-shared-" + t.Name()

	// Use --skip-plugin to avoid needing claude CLI
	cmd := exec.Command(bin, "init", "--shared", boardName, "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success for --shared + --agent, got error: %s", out)
	}

	// Verify shared board was created
	home, _ := os.UserHomeDir()
	boardFile := filepath.Join(home, ".obeya", "boards", boardName, ".obeya", "board.json")
	if _, err := os.Stat(boardFile); err != nil {
		t.Errorf("shared board not created at %s", boardFile)
	}

	// Verify global CLAUDE.md was updated
	claudeMD := filepath.Join(home, ".claude", "CLAUDE.md")
	claudeData, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatalf("could not read global CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claudeData), "Task Tracking — Obeya") {
		t.Error("global CLAUDE.md does not contain Obeya section")
	}

	// Cleanup
	os.RemoveAll(filepath.Join(home, ".obeya", "boards", boardName))
}
