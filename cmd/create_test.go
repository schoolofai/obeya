package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupBoardDir(t *testing.T, bin string) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %s\n%s", err, out)
	}

	// Register a test user so --assign can resolve
	cmd = exec.Command(bin, "user", "add", "testbot", "--type", "agent", "--provider", "claude-code")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("user add failed: %s\n%s", err, out)
	}
	return dir
}

func TestCreate_MissingAssignFails(t *testing.T) {
	bin := buildOb(t)
	dir := setupBoardDir(t, bin)

	cmd := exec.Command(bin, "create", "task", "Test task", "-d", "test description")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when --assign is missing")
	}
	output := string(out)
	if !strings.Contains(output, "--assign is required") {
		t.Errorf("expected '--assign is required' in error, got:\n%s", output)
	}
	if !strings.Contains(output, "ob user list") {
		t.Errorf("expected 'ob user list' hint in error, got:\n%s", output)
	}
}

func TestCreate_WithAssignSucceeds(t *testing.T) {
	bin := buildOb(t)
	dir := setupBoardDir(t, bin)

	cmd := exec.Command(bin, "create", "task", "Test task", "--assign", "testbot", "-d", "test description")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success with --assign, got error: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "Created task") {
		t.Errorf("expected 'Created task' output, got:\n%s", out)
	}
}

func TestCreate_UnknownAssigneeFails(t *testing.T) {
	bin := buildOb(t)
	dir := setupBoardDir(t, bin)

	cmd := exec.Command(bin, "create", "task", "Test task", "--assign", "nonexistent-user", "-d", "test description")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for unknown assignee")
	}
	output := string(out)
	if !strings.Contains(output, "unknown assignee") {
		t.Errorf("expected 'unknown assignee' in error, got:\n%s", output)
	}
}

func TestCreate_OBUserDeprecationWarning(t *testing.T) {
	bin := buildOb(t)
	dir := setupBoardDir(t, bin)

	cmd := exec.Command(bin, "list")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "OB_USER=some-old-value")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list command failed: %s\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "OB_USER is deprecated") {
		t.Errorf("expected deprecation warning for OB_USER, got:\n%s", output)
	}
}
