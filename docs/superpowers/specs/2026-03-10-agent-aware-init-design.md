# Agent-Aware `ob init` with Plugin Installation

**Date:** 2026-03-10
**Status:** Approved

## Summary

Make `ob init` agent-aware with a mandatory `--agent` flag. When `--agent claude-code` is specified, init creates the board, updates CLAUDE.md, and installs the obeya Claude Code plugin via the official `claude` CLI. A standalone `ob plugin claude-install` command allows plugin installation independently of init.

## Command Signatures

### `ob init` (modified)

```bash
ob init [name] --agent <agent-name> [flags]

# Required:
  --agent <name>         Coding agent to configure (currently: claude-code)

# Existing flags (unchanged):
  --columns "col1,..."   Custom column names
  --root <path>          Custom board location
  --shared <name>        Shared board at ~/.obeya/boards/<name>

# New flags:
  --skip-plugin          Skip plugin installation (board + CLAUDE.md only)

# Removed:
  --claude-md            No longer independent — controlled by --agent
```

**Mutual exclusivity:** `--shared` and `--agent` cannot be used together. Shared boards are storage-only — agent setup (CLAUDE.md, plugin) only makes sense in a project context. If both are provided, error: `"--shared and --agent are mutually exclusive. Shared boards do not support agent integration."`

### `ob plugin claude-install` (new)

```bash
ob plugin claude-install
```

No flags. Registers the obeya GitHub marketplace and installs the plugin via `claude` CLI. This is the same logic called by `ob init --agent claude-code` during the plugin step.

**Prerequisites:** Requires `claude` CLI in PATH. Does NOT require an existing board — the plugin can be installed independently.

## Flow

```
ob init myboard --agent claude-code
│
├─ 1. VALIDATE
│     --shared + --agent both provided?
│     YES → error: "--shared and --agent are mutually exclusive"
│
│     --shared provided (without --agent)?
│     YES → create shared board (existing logic, unchanged) → exit
│
│     --agent flag provided?
│     NO  → error: "required flag --agent not provided. Supported: claude-code"
│     YES → is it a supported agent?
│           NO  → error: "unsupported agent 'X'. Supported: claude-code"
│
├─ 2. RESOLVE ROOT (unchanged)
│     --root given? → use it
│     else          → walk up for .git/ (FindGitRoot)
│
├─ 3. CREATE BOARD (unchanged)
│     .obeya/board.json with name + columns
│
├─ 4. AGENT-SPECIFIC SETUP (dispatched via AgentSetup interface)
│     ┌─ agent = "claude-code" ─────────────────────────┐
│     │                                                  │
│     │  4a. Update CLAUDE.md                            │
│     │      Same versioned section logic as today.      │
│     │      obeyaClaudeMDContent() and                  │
│     │      appendClaudeMDAt() move to                  │
│     │      internal/agent/claudecode.go                │
│     │                                                  │
│     │  4b. Install plugin (unless --skip-plugin)       │
│     │      ├─ Check: is `claude` in PATH?              │
│     │      │  NO → error on plugin step only.          │
│     │      │  Board + CLAUDE.md already created.       │
│     │      │  Exit non-zero with message:              │
│     │      │  "claude CLI not found. Install Claude    │
│     │      │   Code, then run ob plugin claude-install │
│     │      │   to complete setup."                     │
│     │      │                                           │
│     │      ├─ Register marketplace:                    │
│     │      │  claude plugin marketplace add \           │
│     │      │    https://github.com/schoolofai/obeya.git│
│     │      │                                           │
│     │      ├─ Version check:                           │
│     │      │  Run: claude plugin list --json            │
│     │      │  (or parse text output for                │
│     │      │   "obeya@obeya-local" + version)          │
│     │      │  Same version? → skip with message        │
│     │      │  Older version? → proceed to install      │
│     │      │  Not installed? → proceed to install      │
│     │      │                                           │
│     │      └─ Install plugin:                          │
│     │         claude plugin install \                   │
│     │           obeya@obeya-local --scope user         │
│     │                                                  │
│     └──────────────────────────────────────────────────┘
│
└─ 5. SUMMARY
      Board "myboard" initialized in /path/.obeya/
      Columns: backlog, todo, in-progress, review, done
      CLAUDE.md updated with Obeya instructions
      Plugin obeya@obeya-local installed (v1.0.0)
```

## Agent Registry (extensibility)

Agents implement a simple interface:

```go
// AgentContext carries board metadata needed by agent setup.
type AgentContext struct {
    Root       string   // project root directory
    BoardName  string   // board name for summary output
    SkipPlugin bool     // --skip-plugin flag
}

type AgentSetup interface {
    Name() string
    Setup(ctx AgentContext) error
}
```

Claude Code implementation (`internal/agent/claudecode.go`):
- Writes/updates CLAUDE.md with versioned obeya section (moved from `cmd/init.go`)
- Shells out to `claude` CLI for marketplace registration and plugin install

Future agents implement this interface with their own setup logic. The init command delegates to the agent-specific setup after board creation.

Currently supported agents:
- `claude-code` — Claude Code with obeya plugin

## Plugin Distribution

The obeya plugin is distributed via GitHub as a self-hosted marketplace:
- Repository: `https://github.com/schoolofai/obeya.git`
- Marketplace name: `obeya-local` (from `obeya-plugin/.claude-plugin/marketplace.json`)
- Plugin name: `obeya`
- Users register the marketplace once; updates handled by `claude plugin update`

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `--agent` missing (no `--shared`) | Hard error, list supported agents |
| `--shared` + `--agent` both provided | Hard error: mutually exclusive |
| `--shared` without `--agent` | Existing shared board logic, unchanged |
| Unknown agent name | Hard error, list supported agents |
| `claude` not in PATH | Board + CLAUDE.md created. Plugin step fails with non-zero exit + message to run `ob plugin claude-install` later |
| Marketplace registration fails | Hard error on plugin step, show stderr from `claude` CLI |
| Plugin install fails | Hard error on plugin step, show stderr from `claude` CLI |
| Plugin already installed (same version) | Skip with message, not an error |
| Board already exists | Message + continue to agent setup (unchanged) |
| `claude plugin list` output unparseable | Proceed with install (assume not installed) |

No fallback mechanisms. Fast fail with clear error messages.

## Command Tree After Changes

```
ob init [name] --agent <agent>    # modified: --agent required, --claude-md removed
ob plugin
├── sync                          # unchanged: dev symlink workflow
└── claude-install                # new: standalone plugin install via claude CLI
```

## Breaking Changes

This is a pre-1.0 project with a small user base. No deprecation period — clean break.

- `--agent` flag is now **required** on `ob init` (unless `--shared` is used). Existing `ob init` calls without it will error with a clear message listing supported agents.
- `--claude-md` flag removed. When `--agent claude-code` is specified, CLAUDE.md is always created/updated. There is no opt-out — CLAUDE.md is essential for the plugin to function correctly.
- Exit code changes: `ob init` without `--agent` now exits non-zero (previously exited 0). Scripts that call `ob init` must be updated.

## Files to Modify

| File | Change |
|------|--------|
| `cmd/init.go` | Add `--agent` (required), `--skip-plugin` flags. Remove `--claude-md` flag. Add `--shared` + `--agent` mutual exclusivity check. Move CLAUDE.md logic to `internal/agent/`. Dispatch to `AgentSetup` after board creation. |
| `cmd/plugin.go` | Add `claude-install` subcommand calling shared install logic from `internal/agent/claudecode.go`. |
| `internal/agent/agent.go` | New file. `AgentContext` struct, `AgentSetup` interface, agent registry (map of name → setup). |
| `internal/agent/claudecode.go` | New file. `ClaudeCodeSetup` implementing `AgentSetup`. Contains: CLAUDE.md logic (moved from `cmd/init.go`), `claude` CLI detection, marketplace registration, version check, plugin install. |
| `internal/agent/claudecode_test.go` | New file. Tests for CLAUDE.md generation, CLI detection, version comparison. |
| `cmd/init_test.go` | Update existing tests for `--agent` requirement, `--shared` + `--agent` mutual exclusivity. |
| `docs/architecture/init/ob-init.md` | Update flow diagram to reflect new agent-aware flow. |
