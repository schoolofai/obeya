package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstall_ShowsPreviewOrClaudeError(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "uninstall", "--yes")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	outStr := string(out)

	if !strings.Contains(outStr, "claude CLI not found") &&
		!strings.Contains(outStr, "the following changes will be made") {
		t.Errorf("unexpected output: %s", outStr)
	}
}

func TestUninstall_HelpOutput(t *testing.T) {
	bin := buildOb(t)

	cmd := exec.Command(bin, "uninstall", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %s", out)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "Removes the Claude Code plugin") {
		t.Errorf("expected help text, got: %s", outStr)
	}
	if !strings.Contains(outStr, "--yes") {
		t.Errorf("expected --yes flag in help, got: %s", outStr)
	}
}
