package cmd_test

import (
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

func TestInit_SharedAndAgentMutuallyExclusive(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--shared", "test", "--agent", "claude-code")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when --shared and --agent both provided")
	}
	if !strings.Contains(string(out), "mutually exclusive") {
		t.Errorf("expected mutual exclusivity message, got: %s", out)
	}
}
