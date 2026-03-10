# Agent-Aware `ob init` Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `ob init` agent-aware with a mandatory `--agent` flag, and add `ob plugin claude-install` for standalone plugin installation via the `claude` CLI.

**Architecture:** New `internal/agent` package with an `AgentSetup` interface. Claude Code implementation handles CLAUDE.md updates (moved from `cmd/init.go`) and plugin installation via `claude` CLI. `cmd/init.go` validates flags and delegates to agent setup after board creation.

**Tech Stack:** Go, cobra CLI, os/exec for shelling out to `claude` CLI

**Spec:** `docs/superpowers/specs/2026-03-10-agent-aware-init-design.md`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/agent/agent.go` | `AgentContext` struct, `AgentSetup` interface, agent registry |
| `internal/agent/claudecode.go` | `ClaudeCodeSetup` — CLAUDE.md logic (moved from `cmd/init.go`) + plugin install via `claude` CLI |
| `internal/agent/claudecode_test.go` | Tests for CLAUDE.md generation, CLI detection, version check |
| `cmd/init.go` | Modified — `--agent` required, `--claude-md` removed, delegates to agent setup |
| `cmd/init_test.go` | New — tests for flag validation, `--shared`+`--agent` exclusivity |
| `cmd/plugin.go` | Modified — add `claude-install` subcommand |
| `docs/architecture/init/ob-init.md` | Updated flow diagram |

---

## Chunk 1: Agent Package Foundation

### Task 1: Create `AgentSetup` interface and registry

**Files:**
- Create: `internal/agent/agent.go`
- Test: `internal/agent/agent_test.go`

- [ ] **Step 1: Write the failing test for agent registry**

```go
// internal/agent/agent_test.go
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
```

Note: `TestGetAgent_ClaudeCode` and `TestSupportedAgents` are deferred to Task 2 when the Claude Code agent is registered. This keeps tests green at every commit.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/niladribose/code/obeya && go test ./internal/agent/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write the implementation**

```go
// internal/agent/agent.go
package agent

import (
	"fmt"
	"sort"
	"strings"
)

// AgentContext carries metadata needed by agent setup.
type AgentContext struct {
	Root       string // project root directory
	BoardName  string // board name for summary output
	SkipPlugin bool   // --skip-plugin flag
}

// AgentSetup defines the interface for agent-specific initialization.
type AgentSetup interface {
	Name() string
	Setup(ctx AgentContext) error
}

var registry = map[string]AgentSetup{}

func register(a AgentSetup) {
	registry[a.Name()] = a
}

// Get returns the AgentSetup for the given name.
func Get(name string) (AgentSetup, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported agent %q. Supported: %s", name, strings.Join(SupportedNames(), ", "))
	}
	return a, nil
}

// SupportedNames returns sorted list of registered agent names.
func SupportedNames() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/niladribose/code/obeya && go test ./internal/agent/ -v`
Expected: `TestGetAgent_Unknown` PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/agent.go internal/agent/agent_test.go
git commit -m "feat: add agent package with AgentSetup interface and registry"
```

---

### Task 2: Create `ClaudeCodeSetup` with CLAUDE.md logic

**Files:**
- Create: `internal/agent/claudecode.go`
- Create: `internal/agent/claudecode_test.go`
- Modify: `cmd/init.go` (remove CLAUDE.md functions — they move to the new file)

- [ ] **Step 1: Write the failing tests for CLAUDE.md generation, registration, and skip-plugin**

```go
// internal/agent/claudecode_test.go
package agent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/agent"
)

// --- Tests deferred from Task 1 (need ClaudeCodeSetup registered) ---

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

// --- CLAUDE.md tests ---

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

// --- Setup with SkipPlugin ---

func TestClaudeCodeSetup_SkipPlugin(t *testing.T) {
	dir := t.TempDir()
	// Create a .git dir so it looks like a project root
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

	// CLAUDE.md should still be created
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Error("CLAUDE.md should exist even with skip-plugin")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/niladribose/code/obeya && go test ./internal/agent/ -v -run TestClaudeCode`
Expected: FAIL — `ClaudeCodeSetup` and `AppendClaudeMDAt` not defined

- [ ] **Step 3: Write the implementation**

Move the CLAUDE.md logic from `cmd/init.go` to `internal/agent/claudecode.go`. The `ClaudeCodeSetup.Setup()` method calls `AppendClaudeMDAt` and then attempts plugin install.

```go
// internal/agent/claudecode.go
package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ObeyaSectionStart = "<!-- obeya:start -->"
const ObeyaSectionEnd = "<!-- obeya:end -->"
const ObeyaSectionVersion = "v5"

const marketplaceURL = "https://github.com/schoolofai/obeya.git"
const pluginRef = "obeya@obeya-local"

func init() {
	register(&ClaudeCodeSetup{})
}

// ClaudeCodeSetup implements AgentSetup for Claude Code.
type ClaudeCodeSetup struct{}

func (c *ClaudeCodeSetup) Name() string { return "claude-code" }

func (c *ClaudeCodeSetup) Setup(ctx AgentContext) error {
	// 1. Update CLAUDE.md
	claudePath := filepath.Join(ctx.Root, "CLAUDE.md")
	if err := AppendClaudeMDAt(claudePath); err != nil {
		return fmt.Errorf("could not update CLAUDE.md: %w", err)
	}
	fmt.Println("Updated CLAUDE.md with Obeya board instructions")

	// 2. Install plugin (unless skipped)
	if ctx.SkipPlugin {
		return nil
	}

	return installClaudePlugin()
}

func installClaudePlugin() error {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Install Claude Code from https://docs.anthropic.com/en/docs/claude-code, then run: ob plugin claude-install")
	}

	// Register marketplace
	cmd := exec.Command(claudeBin, "plugin", "marketplace", "add", marketplaceURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register obeya marketplace: %w", err)
	}

	// Check if already installed at current version
	if alreadyInstalled(claudeBin) {
		fmt.Printf("Plugin %s is already installed and up to date\n", pluginRef)
		return nil
	}

	// Install plugin
	cmd = exec.Command(claudeBin, "plugin", "install", pluginRef, "--scope", "user")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install obeya plugin: %w", err)
	}

	fmt.Printf("Plugin %s installed\n", pluginRef)
	return nil
}

// alreadyInstalled checks if the obeya plugin is present in claude's plugin list.
// NOTE: Deliberately simplified for v1 — checks presence only, not version comparison.
// The `claude plugin install` command itself handles upgrades, so this is a
// performance optimization to skip the install call when unnecessary.
func alreadyInstalled(claudeBin string) bool {
	out, err := exec.Command(claudeBin, "plugin", "list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "obeya@obeya-local")
}

// InstallPlugin is the public entry point for `ob plugin claude-install`.
func InstallPlugin() error {
	return installClaudePlugin()
}

func ObeyaClaudeMDContent() string {
	return ObeyaSectionStart + " " + ObeyaSectionVersion + `

## Task Tracking — Obeya

This project uses Obeya (` + "`ob`" + `) for task tracking. The board is the single source of truth for all work.

### Mandatory: Track ALL work
Every piece of work MUST have a task on the board. Before starting any work:
1. Run ` + "`/ob:status`" + ` to check assigned tasks
2. If no task exists for this work, create one with ` + "`ob create task \"Title\" --description \"...\"`" + `
3. Run ` + "`/ob:pick`" + ` to claim a task when implementing from the backlog
4. Run ` + "`/ob:done`" + ` when work is complete

### Creating tasks from plans
When breaking down a plan into tasks, create a full hierarchy with detailed descriptions:
- **Epics**: High-level goals. Description includes the objective, success criteria, and scope boundaries.
- **Stories**: Deliverable units. Description includes what needs to be built, why it matters, and acceptance criteria.
- **Tasks**: Atomic work items. Description includes what to do, how to verify it's done, and dependencies on other tasks.

Task descriptions must be self-contained — an agent picking one up should have everything needed to start work. Include key context inline and reference files for larger context (e.g., "See docs/plans/auth-design.md section 3 for protocol details" or "See src/auth/oauth.go for existing implementation").

### Obeya board is authoritative over session tools
` + "`TodoWrite`" + ` and ` + "`TaskCreate`" + ` are ephemeral session aids. The Obeya board persists across sessions and is the source of truth. When any skill or workflow uses session tools (e.g., TodoWrite), the corresponding work MUST also exist on the Obeya board. Specifically:
- Before creating a TodoWrite checklist, ensure equivalent tasks exist on the board via ` + "`ob create`" + `
- When marking a TodoWrite item complete, also run ` + "`ob move <id> done`" + `
- When a skill workflow says "mark task complete in TodoWrite", ALSO run ` + "`ob move <id> done`" + `

### Integration with other skills and workflows
When using skills that dispatch subagents or manage work (e.g., superpowers, subagent-driven-development, executing-plans):
- The **controller/orchestrator** (not the subagent) is responsible for obeya board updates
- Before dispatching work: ensure the task exists on the board and is in-progress
- After subagent completes: run ` + "`ob move <id> done`" + ` on the corresponding board task
- When a plan is broken into subtasks by any skill: create those subtasks on the board too
- This applies regardless of which skill is orchestrating the work

### Task lifecycle
- Starting work: ` + "`ob move <id> in-progress`" + `
- Update progress: ` + "`ob edit <id> --description \"...\"`" + ` — append notes as you work (discoveries, approach changes, blockers hit)
- Blocked: ` + "`ob block <id> --by <blocker-id>`" + `
- Done: ` + "`ob move <id> done`" + `

### Plan management
When a plan document is created, discussed, or approved:
1. Import it: ` + "`ob plan import <path-to-plan.md>`" + `
2. Break it down into epics, stories, and tasks with full descriptions
3. Link tasks to plan: ` + "`ob plan link <plan-id> --to <task-ids>`" + `
4. When creating subtasks under a plan-linked parent, link them too: ` + "`ob plan link <plan-id> --to <new-task-id>`" + `

Use ` + "`ob list --format json`" + ` for full board state.

` + ObeyaSectionEnd + `
`
}

// AppendClaudeMDAt writes or updates the obeya section in a CLAUDE.md file.
func AppendClaudeMDAt(claudePath string) error {
	content := ObeyaClaudeMDContent()

	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	existingStr := string(existing)

	// Replace existing section if present (handles version upgrades)
	if startIdx := strings.Index(existingStr, ObeyaSectionStart); startIdx != -1 {
		endIdx := strings.Index(existingStr, ObeyaSectionEnd)
		if endIdx == -1 {
			return fmt.Errorf("found obeya section start but no end marker in CLAUDE.md")
		}
		endIdx += len(ObeyaSectionEnd)
		if endIdx < len(existingStr) && existingStr[endIdx] == '\n' {
			endIdx++
		}
		updated := existingStr[:startIdx] + content + existingStr[endIdx:]
		return os.WriteFile(claudePath, []byte(updated), 0644)
	}

	// Legacy check: replace old section without markers
	if strings.Contains(existingStr, "Task Tracking — Obeya") {
		legacyStart := strings.Index(existingStr, "## Task Tracking — Obeya")
		if legacyStart > 0 {
			rest := existingStr[legacyStart+1:]
			nextHeading := strings.Index(rest, "\n## ")
			var legacyEnd int
			if nextHeading != -1 {
				legacyEnd = legacyStart + 1 + nextHeading + 1
			} else {
				legacyEnd = len(existingStr)
			}
			updated := existingStr[:legacyStart] + content + existingStr[legacyEnd:]
			return os.WriteFile(claudePath, []byte(updated), 0644)
		}
	}

	// Fresh append
	f, err := os.OpenFile(claudePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CLAUDE.md: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to CLAUDE.md: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/niladribose/code/obeya && go test ./internal/agent/ -v`
Expected: All tests PASS (including `TestGetAgent_ClaudeCode` from Task 1 now that `init()` registers it)

- [ ] **Step 5: Commit**

```bash
git add internal/agent/claudecode.go internal/agent/claudecode_test.go
git commit -m "feat: add ClaudeCodeSetup with CLAUDE.md logic and plugin install"
```

---

## Chunk 2: Modify `cmd/init.go` and `cmd/plugin.go`

### Task 3: Update `cmd/init.go` with `--agent` flag

**Files:**
- Modify: `cmd/init.go`

- [ ] **Step 1: Write the failing test for init flag validation**

```go
// cmd/init_test.go
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
	cmd.Dir = filepath.Join(mustGitRoot(t))
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

func TestInit_SharedWithoutAgentWorks(t *testing.T) {
	bin := buildOb(t)

	// Use a custom home to avoid touching real ~/.obeya
	homeDir := t.TempDir()
	os.Setenv("OBEYA_HOME", homeDir)
	defer os.Unsetenv("OBEYA_HOME")

	cmd := exec.Command(bin, "init", "--shared", "test-board")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shared init should work without --agent: %s\n%s", err, out)
	}
}
```

Note: `TestInit_SharedWithoutAgentWorks` depends on `OBEYA_HOME` env var support in `store.ObeyaHome()`. If that env var isn't supported, this test should use a different isolation approach — check `internal/store/root.go` during implementation.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/niladribose/code/obeya && go test ./cmd/ -v -run TestInit_`
Expected: FAIL — current `cmd/init.go` doesn't have `--agent` flag

- [ ] **Step 3: Modify `cmd/init.go`**

Replace the entire file. Key changes:
- Add `initAgent` string var and `--agent` flag (required)
- Add `initSkipPlugin` bool var and `--skip-plugin` flag
- Remove `initClaudeMD` var and `--claude-md` flag
- Add `--shared` + `--agent` mutual exclusivity check
- After board creation, dispatch to `agent.Get(initAgent).Setup()`
- Remove `obeyaClaudeMDContent()`, `appendClaudeMDAt()`, and related constants (moved to `internal/agent/claudecode.go`)

Updated `cmd/init.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initColumns string
var initAgent string
var initSkipPlugin bool
var initRoot string
var initShared string

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board with agent integration. Requires --agent flag.\nUse --shared for storage-only boards (no agent integration).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		columns := parseColumns(initColumns)

		// --shared and --agent are mutually exclusive
		if initShared != "" && initAgent != "" {
			return fmt.Errorf("--shared and --agent are mutually exclusive. Shared boards do not support agent integration")
		}

		// Shared board path (no agent needed)
		if initShared != "" {
			return initSharedBoard(initShared, columns)
		}

		// --agent is required for non-shared boards
		if initAgent == "" {
			return fmt.Errorf("required flag --agent not provided. Supported: %s", strings.Join(agent.SupportedNames(), ", "))
		}

		// Validate agent name
		agentSetup, err := agent.Get(initAgent)
		if err != nil {
			return err
		}

		root, err := resolveInitRoot()
		if err != nil {
			return err
		}

		s := store.NewJSONStore(root)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		err = s.InitBoard(boardName, columns)
		if err != nil {
			if !strings.Contains(err.Error(), "already initialized") {
				return err
			}
			fmt.Printf("Board already initialized in %s/.obeya/\n", root)
		} else {
			fmt.Printf("Board %q initialized in %s/.obeya/\n", boardName, root)
			if len(columns) > 0 {
				fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
			} else {
				fmt.Println("Columns: backlog, todo, in-progress, review, done")
			}
		}

		// Delegate to agent-specific setup
		ctx := agent.AgentContext{
			Root:       root,
			BoardName:  boardName,
			SkipPlugin: initSkipPlugin,
		}
		if err := agentSetup.Setup(ctx); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().StringVar(&initAgent, "agent", "", "coding agent to configure (supported: claude-code)")
	initCmd.Flags().BoolVar(&initSkipPlugin, "skip-plugin", false, "skip plugin installation")
	initCmd.Flags().StringVar(&initRoot, "root", "", "directory to create .obeya in (default: git repository root)")
	initCmd.Flags().StringVar(&initShared, "shared", "", "create a shared board at ~/.obeya/boards/<name>")
	rootCmd.AddCommand(initCmd)
}

func initSharedBoard(boardName string, columns []string) error {
	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}

	boardDir := store.SharedBoardDir(obeyaHome, boardName)
	boardFile := filepath.Join(boardDir, ".obeya", "board.json")

	if _, err := os.Stat(boardFile); err == nil {
		return fmt.Errorf("board %q already exists — use 'ob link %s' to connect this project", boardName, boardName)
	}

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Shared board %q initialized at %s\n", boardName, boardDir)
	return nil
}

func parseColumns(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func resolveInitRoot() (string, error) {
	if initRoot != "" {
		abs, err := filepath.Abs(initRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve --root path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("--root path does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("--root path is not a directory: %s", abs)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return store.FindGitRoot(cwd)
}
```

- [ ] **Step 4: Verify the project compiles**

Run: `cd /Users/niladribose/code/obeya && go build ./...`
Expected: No errors. Check that no other files in `cmd/` import the removed constants/functions (`obeyaSectionStart`, `obeyaSectionEnd`, `obeyaSectionVersion`, `obeyaClaudeMDContent`, `appendClaudeMDAt`).

- [ ] **Step 5: Run all existing tests**

Run: `cd /Users/niladribose/code/obeya && go test ./... -v -count=1`
Expected: All tests pass. If any test referenced the removed functions/constants, update them to use `agent.ObeyaSectionStart` etc.

- [ ] **Step 6: Commit**

```bash
git add cmd/init.go cmd/init_test.go
git commit -m "feat: make --agent flag required on ob init, remove --claude-md"
```

---

### Task 4: Add `ob plugin claude-install` subcommand

**Files:**
- Modify: `cmd/plugin.go`

- [ ] **Step 1: Add the `claude-install` subcommand to `cmd/plugin.go`**

Add after the existing `pluginSyncCmd` definition:

```go
var pluginClaudeInstallCmd = &cobra.Command{
	Use:   "claude-install",
	Short: "Install the obeya plugin into Claude Code via the claude CLI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.InstallPlugin()
	},
}
```

Update the `init()` function to register it:

```go
func init() {
	pluginCmd.AddCommand(pluginSyncCmd)
	pluginCmd.AddCommand(pluginClaudeInstallCmd)
	rootCmd.AddCommand(pluginCmd)
}
```

Add the import for `"github.com/niladribose/obeya/internal/agent"`.

- [ ] **Step 2: Verify it compiles and the help output is correct**

Run: `cd /Users/niladribose/code/obeya && go build -o ob . && ./ob plugin --help`
Expected output includes both `sync` and `claude-install` subcommands.

- [ ] **Step 3: Commit**

```bash
git add cmd/plugin.go
git commit -m "feat: add ob plugin claude-install subcommand"
```

---

## Chunk 3: Update Documentation

### Task 5: Update `docs/architecture/init/ob-init.md`

**Files:**
- Modify: `docs/architecture/init/ob-init.md`

- [ ] **Step 1: Update the flow diagram and command signature**

Update the document to reflect:
- `--agent` flag is required (unless `--shared`)
- `--claude-md` flag removed
- `--skip-plugin` flag added
- `--shared` + `--agent` mutually exclusive
- Agent-specific setup step after board creation
- New `ob plugin claude-install` command documented

- [ ] **Step 2: Commit**

```bash
git add docs/architecture/init/ob-init.md
git commit -m "docs: update init architecture doc for agent-aware flow"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/niladribose/code/obeya && go test ./... -v -count=1`
Expected: All tests pass

- [ ] **Step 2: Run go vet**

Run: `cd /Users/niladribose/code/obeya && go vet ./...`
Expected: No issues

- [ ] **Step 3: Manual smoke test — init without --agent**

Run: `cd /tmp && mkdir test-init && cd test-init && git init && /Users/niladribose/code/obeya/ob init`
Expected: Error message listing supported agents

- [ ] **Step 4: Manual smoke test — init with --agent and --skip-plugin**

Run: `cd /tmp/test-init && /Users/niladribose/code/obeya/ob init --agent claude-code --skip-plugin`
Expected: Board created, CLAUDE.md updated, no plugin install attempted

- [ ] **Step 5: Manual smoke test — shared + agent mutual exclusivity**

Run: `cd /tmp/test-init && /Users/niladribose/code/obeya/ob init --shared test --agent claude-code`
Expected: Error about mutual exclusivity

- [ ] **Step 6: Cleanup and final commit if needed**

```bash
rm -rf /tmp/test-init
```
