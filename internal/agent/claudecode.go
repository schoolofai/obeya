package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ObeyaSectionStart   = "<!-- obeya:start -->"
	ObeyaSectionEnd     = "<!-- obeya:end -->"
	ObeyaSectionVersion = "v5"

	marketplaceURL = "https://github.com/schoolofai/obeya.git"
	pluginRef      = "obeya@obeya-local"
)

func init() {
	register(&ClaudeCodeSetup{})
}

// ClaudeCodeSetup implements AgentSetup for the Claude Code agent.
type ClaudeCodeSetup struct{}

func (c *ClaudeCodeSetup) Name() string { return "claude-code" }

// Setup updates CLAUDE.md and optionally installs the Claude plugin.
func (c *ClaudeCodeSetup) Setup(ctx AgentContext) error {
	claudePath := filepath.Join(ctx.Root, "CLAUDE.md")
	if ctx.Shared {
		// Shared board: write to global ~/.claude/CLAUDE.md
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine home directory: %w", err)
		}
		claudePath = filepath.Join(home, ".claude", "CLAUDE.md")
	}

	if err := AppendClaudeMDAt(claudePath); err != nil {
		return fmt.Errorf("could not update CLAUDE.md: %w", err)
	}
	fmt.Printf("Updated CLAUDE.md at %s\n", claudePath)

	if !ctx.SkipPlugin {
		if err := installClaudePlugin(); err != nil {
			return fmt.Errorf("could not install claude plugin: %w", err)
		}
	}

	return nil
}

// InstallPlugin is a public entry point for standalone plugin installation.
func InstallPlugin() error {
	return installClaudePlugin()
}

func installClaudePlugin() error {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH: %w", err)
	}
	_ = claudeBin

	// Register marketplace
	regCmd := exec.Command("claude", "plugin", "marketplace", "add", marketplaceURL)
	if out, err := regCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to register marketplace: %s: %w", string(out), err)
	}

	if alreadyInstalled() {
		return nil
	}

	installCmd := exec.Command("claude", "plugin", "install", pluginRef)
	if out, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install plugin: %s: %w", string(out), err)
	}

	return nil
}

func alreadyInstalled() bool {
	cmd := exec.Command("claude", "plugin", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), pluginRef)
}

// ObeyaClaudeMDContent returns the full obeya section for CLAUDE.md.
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
	if alreadyInstalled() {
		cmd := exec.Command("claude", "plugin", "uninstall", pluginRef)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall plugin: %s: %w", string(out), err)
		}
	}

	if marketplaceRegistered() {
		cmd := exec.Command("claude", "plugin", "marketplace", "remove", "obeya-local")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove marketplace: %s: %w", string(out), err)
		}
	}

	return nil
}

func marketplaceRegistered() bool {
	cmd := exec.Command("claude", "plugin", "marketplace", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "obeya")
}

// AppendClaudeMDAt writes or updates the obeya section in a CLAUDE.md file.
func AppendClaudeMDAt(claudePath string) error {
	content := ObeyaClaudeMDContent()

	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	existingStr := string(existing)

	// Case 1: Replace existing section (version upgrade)
	if startIdx := strings.Index(existingStr, ObeyaSectionStart); startIdx != -1 {
		endIdx := strings.Index(existingStr, ObeyaSectionEnd)
		if endIdx == -1 {
			return fmt.Errorf("found obeya section start but no end marker in CLAUDE.md")
		}
		endIdx += len(ObeyaSectionEnd)
		// Skip trailing newline if present
		if endIdx < len(existingStr) && existingStr[endIdx] == '\n' {
			endIdx++
		}
		updated := existingStr[:startIdx] + content + existingStr[endIdx:]
		return os.WriteFile(claudePath, []byte(updated), 0644)
	}

	// Case 2: Legacy section without markers
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

	// Case 3: Fresh append
	if err := os.MkdirAll(filepath.Dir(claudePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for CLAUDE.md: %w", err)
	}
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
