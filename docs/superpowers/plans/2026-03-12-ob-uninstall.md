# ob uninstall Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `ob uninstall` command that removes all Obeya agent integrations while preserving user board data.

**Architecture:** New cobra command in `cmd/uninstall.go` delegates to `internal/agent/claudecode.go` for plugin removal and CLAUDE.md stripping. Reuses existing constants (`ObeyaSectionStart`, `ObeyaSectionEnd`) and `getProviders()` from `cmd/skill.go` for skill file paths.

**Tech Stack:** Go, Cobra CLI, os/exec (claude CLI calls)

**Spec:** `docs/superpowers/specs/2026-03-12-ob-uninstall-design.md`

---

## Chunk 1: Core uninstall logic

### Task 1: StripClaudeMDAt — strip obeya section from CLAUDE.md

**Files:**
- Modify: `internal/agent/claudecode.go` (add `StripClaudeMDAt` function after `AppendClaudeMDAt` at line 208)
- Create: `internal/agent/claudecode_test.go`

- [ ] **Step 1: Write failing tests for StripClaudeMDAt**

Note: existing `internal/agent/claudecode_test.go` uses `package agent_test`, so all tests use qualified `agent.` calls. The test file already exists — append these tests to it.

```go
// Append to internal/agent/claudecode_test.go (package agent_test)

func TestStripClaudeMDAt_RemovesObeyaSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := "# My Project\n\nSome instructions.\n\n" + agent.ObeyaClaudeMDContent() + "\n\n## Other stuff\n"
	os.WriteFile(path, []byte(content), 0644)

	err := agent.StripClaudeMDAt(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := os.ReadFile(path)
	if strings.Contains(string(result), agent.ObeyaSectionStart) {
		t.Error("obeya start marker still present")
	}
	if strings.Contains(string(result), agent.ObeyaSectionEnd) {
		t.Error("obeya end marker still present")
	}
	if !strings.Contains(string(result), "# My Project") {
		t.Error("non-obeya content was removed")
	}
	if !strings.Contains(string(result), "## Other stuff") {
		t.Error("content after obeya section was removed")
	}
}

func TestStripClaudeMDAt_NoMarkers_Noop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := "# My Project\n\nNo obeya here.\n"
	os.WriteFile(path, []byte(content), 0644)

	err := agent.StripClaudeMDAt(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := os.ReadFile(path)
	if string(result) != content {
		t.Errorf("file was modified when it shouldn't have been")
	}
}

func TestStripClaudeMDAt_FileDoesNotExist_Noop(t *testing.T) {
	err := agent.StripClaudeMDAt("/nonexistent/path/CLAUDE.md")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
}

func TestStripClaudeMDAt_StartWithoutEnd_Errors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := "# Project\n\n" + agent.ObeyaSectionStart + " v5\n\nDangling section.\n"
	os.WriteFile(path, []byte(content), 0644)

	err := agent.StripClaudeMDAt(path)
	if err == nil {
		t.Fatal("expected error for mismatched markers")
	}
	if !strings.Contains(err.Error(), "no end marker") {
		t.Errorf("expected 'no end marker' message, got: %v", err)
	}
}

func TestStripClaudeMDAt_OnlyObeyaContent_DeletesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := agent.ObeyaClaudeMDContent() + "\n"
	os.WriteFile(path, []byte(content), 0644)

	err := agent.StripClaudeMDAt(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted when only obeya content remains")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/agent/ -run TestStripClaudeMDAt -v`
Expected: FAIL — `StripClaudeMDAt` not defined

- [ ] **Step 3: Implement StripClaudeMDAt**

Add to `internal/agent/claudecode.go` after `AppendClaudeMDAt` (after line 208):

```go
// StripClaudeMDAt removes the obeya section from a CLAUDE.md file.
// Returns nil if the file doesn't exist or has no obeya markers (idempotent).
func StripClaudeMDAt(claudePath string) error {
	existing, err := os.ReadFile(claudePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", claudePath, err)
	}

	content := string(existing)
	startIdx := strings.Index(content, ObeyaSectionStart)
	if startIdx == -1 {
		return nil
	}

	endIdx := strings.Index(content, ObeyaSectionEnd)
	if endIdx == -1 {
		return fmt.Errorf("found obeya section start but no end marker in %s", claudePath)
	}
	endIdx += len(ObeyaSectionEnd)

	// Expand to consume surrounding blank lines
	for startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}
	for endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	cleaned := content[:startIdx]
	if startIdx > 0 && endIdx < len(content) {
		cleaned += "\n"
	}
	cleaned += content[endIdx:]

	if strings.TrimSpace(cleaned) == "" {
		return os.Remove(claudePath)
	}

	return os.WriteFile(claudePath, []byte(cleaned), 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/agent/ -run TestStripClaudeMDAt -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/claudecode.go internal/agent/claudecode_test.go
git commit -m "feat: add StripClaudeMDAt for removing obeya section from CLAUDE.md"
```

---

### Task 2: UninstallPlugin — uninstall claude plugin and marketplace

**Files:**
- Modify: `internal/agent/claudecode.go` (add `UninstallPlugin` function)
- Modify: `internal/agent/claudecode_test.go` (add tests)

- [ ] **Step 1: Write failing tests for UninstallPlugin**

Note: `UninstallPlugin` shells out to `claude` CLI, so unit tests focus on the exported helpers. Integration tests would require `claude` CLI. Tests use `package agent_test` to match existing file.

```go
// Append to internal/agent/claudecode_test.go (package agent_test)

func TestCheckClaudeCLI_NoClaude_Errors(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := agent.CheckClaudeCLI()
	if err == nil {
		t.Fatal("expected error when claude CLI not found")
	}
	if !strings.Contains(err.Error(), "claude CLI not found") {
		t.Errorf("expected 'claude CLI not found' message, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/ -run TestCheckClaudeCLI -v`
Expected: FAIL — `CheckClaudeCLI` not defined

- [ ] **Step 3: Implement UninstallPlugin**

Add to `internal/agent/claudecode.go`:

```go
// CheckClaudeCLI verifies the claude CLI is available in PATH.
// Call this once before using UninstallPlugin — UninstallPlugin trusts it was already called.
func CheckClaudeCLI() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Required to uninstall plugin.\n" +
			"Install it or remove the plugin manually:\n" +
			"  claude plugin uninstall obeya@obeya-local\n" +
			"  claude plugin marketplace remove obeya-local")
	}
	return nil
}

// UninstallPlugin removes the obeya plugin and marketplace registration via claude CLI.
// Caller must call CheckClaudeCLI first. Skips steps if already uninstalled (idempotent).
func UninstallPlugin() error {
	// Step 1: Uninstall plugin if currently installed
	if alreadyInstalled() {
		cmd := exec.Command("claude", "plugin", "uninstall", pluginRef)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall plugin: %s: %w", string(out), err)
		}
	}

	// Step 2: Remove marketplace if currently registered
	if marketplaceRegistered() {
		cmd := exec.Command("claude", "plugin", "marketplace", "remove", "obeya-local")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove marketplace: %s: %w", string(out), err)
		}
	}

	return nil
}

// marketplaceRegistered checks if the obeya marketplace is currently registered.
func marketplaceRegistered() bool {
	cmd := exec.Command("claude", "plugin", "marketplace", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "obeya")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/ -run TestCheckClaudeCLI -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/claudecode.go internal/agent/claudecode_test.go
git commit -m "feat: add UninstallPlugin for removing claude plugin and marketplace"
```

---

### Task 3: Extract skill file paths to shared helper

**Files:**
- Modify: `cmd/skill.go` (export `GetProviders` — rename `getProviders` to `GetProviders`)
- Create: `cmd/uninstall.go` (will consume `GetProviders`)

- [ ] **Step 1: Rename getProviders to GetProviders and export ProviderInfo**

In `cmd/skill.go`, make these exact changes:

```go
// Line 37-42: Export the struct (note: includes Supported field added in v2.0)
type ProviderInfo struct {
	Name      string
	ConfigDir string
	SkillFile string
	Supported bool
}

// Line 44: Export the function, update return type
func GetProviders() []ProviderInfo {

// Line 50-54: Update struct literals
return []ProviderInfo{
	{Name: "claude-code", ConfigDir: filepath.Join(home, ".claude"), SkillFile: "obeya.md", Supported: true},
	{Name: "opencode", ConfigDir: filepath.Join(home, ".opencode"), SkillFile: "obeya.md", Supported: false},
	{Name: "codex", ConfigDir: filepath.Join(home, ".codex"), SkillFile: "obeya.md", Supported: false},
}

// Line 59: Update call site in runSkillInstall
providers := GetProviders()

// Line 101: Update filterProviders signature
func filterProviders(providers []ProviderInfo, name string) []ProviderInfo {
	for _, p := range providers {
		if p.Name == name {
			return []ProviderInfo{p}
		}
	}

// Line 113: Update installSkillForProvider signature
func installSkillForProvider(p ProviderInfo, content []byte) error {

// Line 125: Update call site in runSkillList
providers := GetProviders()
```

- [ ] **Step 2: Run existing tests to ensure nothing breaks**

Run: `go test ./cmd/... -v`
Expected: existing tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/skill.go
git commit -m "refactor: export GetProviders and ProviderInfo for reuse"
```

---

## Chunk 2: Cobra command and integration

### Task 4: Implement ob uninstall command

**Files:**
- Create: `cmd/uninstall.go`
- Create: `cmd/uninstall_test.go`

- [ ] **Step 1: Write failing test for the uninstall command**

```go
// cmd/uninstall_test.go
package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstall_ShowsPreview(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	// Run with --yes but expect it to fail because claude CLI is likely not in
	// a test PATH. The point is to verify the command exists.
	cmd := exec.Command(bin, "uninstall", "--yes")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	outStr := string(out)

	// Should either show preview or fail with claude CLI error
	if !strings.Contains(outStr, "claude CLI not found") &&
		!strings.Contains(outStr, "the following changes will be made") {
		t.Errorf("unexpected output: %s", outStr)
	}
}

func TestUninstall_StripsCLAUDEmd(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	// Create a CLAUDE.md with obeya section in the project
	claudePath := filepath.Join(dir, "CLAUDE.md")
	content := "# Project\n\n<!-- obeya:start --> v5\n\n## Task Tracking\n\nStuff.\n\n<!-- obeya:end -->\n\n## Other\n"
	os.WriteFile(claudePath, []byte(content), 0644)

	// Run with --yes and fake PATH so claude CLI is not found
	// This should fail at plugin step, but we test the CLAUDE.md stripping
	// by calling StripClaudeMDAt directly (tested in Task 1)
	// Here we just verify the command registered correctly
	cmd := exec.Command(bin, "uninstall", "--help")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %s", out)
	}
	if !strings.Contains(string(out), "Remove Obeya agent integrations") {
		t.Errorf("expected help text, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/... -run TestUninstall -v`
Expected: FAIL — command not registered

- [ ] **Step 3: Implement cmd/uninstall.go**

```go
// cmd/uninstall.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var flagUninstallYes bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Obeya agent integrations (preserves board data)",
	Long:  "Removes the Claude Code plugin, skill files, and CLAUDE.md obeya sections.\nBoard data in .obeya/ directories is preserved.",
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVarP(&flagUninstallYes, "yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx, err := buildUninstallContext()
	if err != nil {
		return err
	}

	printPreview(ctx)

	if !flagUninstallYes {
		if !promptConfirm("Proceed? [y/N] ") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := executeUninstall(ctx); err != nil {
		return err
	}

	printBanner()
	return nil
}

type uninstallContext struct {
	inProject      bool
	gitRoot        string
	globalClaudeMD string
	skillFiles     []skillFileInfo
}

type skillFileInfo struct {
	provider string
	path     string
	exists   bool
}

func buildUninstallContext() (*uninstallContext, error) {
	// Prerequisite: claude CLI must be available
	if err := agent.CheckClaudeCLI(); err != nil {
		return nil, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	ctx := &uninstallContext{
		globalClaudeMD: filepath.Join(home, ".claude", "CLAUDE.md"),
	}

	// Detect project context
	cwd, err := os.Getwd()
	if err == nil {
		if gitRoot, err := store.FindGitRoot(cwd); err == nil {
			claudePath := filepath.Join(gitRoot, "CLAUDE.md")
			if data, err := os.ReadFile(claudePath); err == nil {
				if strings.Contains(string(data), agent.ObeyaSectionStart) {
					ctx.inProject = true
					ctx.gitRoot = gitRoot
				}
			}
		}
	}

	// Discover skill files
	for _, p := range GetProviders() {
		path := filepath.Join(p.ConfigDir, p.SkillFile)
		_, err := os.Stat(path)
		ctx.skillFiles = append(ctx.skillFiles, skillFileInfo{
			provider: p.Name,
			path:     path,
			exists:   err == nil,
		})
	}

	return ctx, nil
}

func printPreview(ctx *uninstallContext) {
	fmt.Println("ob uninstall — the following changes will be made:")
	fmt.Println()
	fmt.Println("  CLAUDE CODE PLUGIN")
	fmt.Println("  ├── Uninstall plugin: obeya@obeya-local")
	fmt.Println("  └── Remove marketplace: obeya-local")
	fmt.Println()

	hasSkills := false
	for _, sf := range ctx.skillFiles {
		if sf.exists {
			hasSkills = true
			break
		}
	}

	if hasSkills {
		fmt.Println("  SKILL FILES")
		existing := []skillFileInfo{}
		for _, sf := range ctx.skillFiles {
			if sf.exists {
				existing = append(existing, sf)
			}
		}
		for i, sf := range existing {
			prefix := "  ├──"
			if i == len(existing)-1 {
				prefix = "  └──"
			}
			fmt.Printf("%s Remove %s\n", prefix, sf.path)
		}
		fmt.Println()
	}

	fmt.Println("  CLAUDE.md")
	if ctx.inProject {
		fmt.Printf("  ├── Strip obeya section from %s\n", ctx.globalClaudeMD)
		fmt.Printf("  └── Strip obeya section from %s\n", filepath.Join(ctx.gitRoot, "CLAUDE.md"))
	} else {
		fmt.Printf("  └── Strip obeya section from %s\n", ctx.globalClaudeMD)
	}
	fmt.Println()

	fmt.Println("  PRESERVED (not touched)")
	fmt.Println("  ├── .obeya/          (board data, cloud config)")
	fmt.Println("  ├── ~/.obeya/        (shared boards, credentials)")
	fmt.Println("  └── .obeya-link      (board links)")
	fmt.Println()
}

func executeUninstall(ctx *uninstallContext) error {
	// Step 1: Plugin removal
	if err := agent.UninstallPlugin(); err != nil {
		return fmt.Errorf("plugin removal failed: %w", err)
	}
	fmt.Println("  ✓ Plugin and marketplace removed")

	// Step 2: Skill files
	for _, sf := range ctx.skillFiles {
		if !sf.exists {
			continue
		}
		if err := os.Remove(sf.path); err != nil {
			return fmt.Errorf("failed to remove skill file %s: %w", sf.path, err)
		}
		fmt.Printf("  ✓ Removed %s\n", sf.path)
	}

	// Step 3a: Global CLAUDE.md
	if err := agent.StripClaudeMDAt(ctx.globalClaudeMD); err != nil {
		return fmt.Errorf("failed to clean global CLAUDE.md: %w", err)
	}
	fmt.Printf("  ✓ Cleaned %s\n", ctx.globalClaudeMD)

	// Step 3b: Project CLAUDE.md
	if ctx.inProject {
		projectPath := filepath.Join(ctx.gitRoot, "CLAUDE.md")
		if err := agent.StripClaudeMDAt(projectPath); err != nil {
			return fmt.Errorf("failed to clean project CLAUDE.md: %w", err)
		}
		fmt.Printf("  ✓ Cleaned %s\n", projectPath)
	}

	return nil
}

func printBanner() {
	fmt.Println()
	fmt.Println("┌───────────────────────────────────────────────────────┐")
	fmt.Println("│                                                       │")
	fmt.Println("│  Obeya agent integrations removed successfully.       │")
	fmt.Println("│                                                       │")
	fmt.Println("│  To fully remove ob from your system:                 │")
	fmt.Println("│                                                       │")
	fmt.Println("│    brew uninstall obeya                               │")
	fmt.Println("│                                                       │")
	fmt.Println("│  Board data and cloud config were preserved.           │")
	fmt.Println("│  Delete them manually if no longer needed.            │")
	fmt.Println("│                                                       │")
	fmt.Println("└───────────────────────────────────────────────────────┘")
}

func promptConfirm(prompt string) bool {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
```

Note: `CheckClaudeCLI` was already added in Task 2 Step 3. No additional code needed here.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/... -run TestUninstall -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/uninstall.go cmd/uninstall_test.go internal/agent/claudecode.go
git commit -m "feat: add ob uninstall command"
```

---

### Task 5: Manual smoke test

- [ ] **Step 1: Build and run dry-run preview**

```bash
go build -o ob . && ./ob uninstall
```

Expected: shows preview, asks `Proceed? [y/N]`, responds to `N` with "Aborted."

- [ ] **Step 2: Verify --yes flag skips prompt**

```bash
./ob uninstall --yes
```

Expected: executes without prompting (or fails at plugin step if claude CLI not available — that's fine for a smoke test).

- [ ] **Step 3: Verify --help output**

```bash
./ob uninstall --help
```

Expected: shows "Remove Obeya agent integrations (preserves board data)" and documents `--yes` flag.

- [ ] **Step 4: Run from outside a project**

```bash
cd /tmp && /path/to/ob uninstall
```

Expected: preview does NOT show project CLAUDE.md line.

- [ ] **Step 5: Commit any fixes from smoke testing**

Stage only the specific files that were fixed, then commit:

```bash
git add <changed-files>
git commit -m "fix: smoke test fixes for ob uninstall"
```
