# `ob init` — How It Works

## Command Signature

```
ob init [name] --agent <agent-name> [flags]

Required:
  --agent <name>         Coding agent to configure (supported: claude-code)

Optional:
  --columns "col1,..."   Custom column names (default: backlog,todo,in-progress,review,done)
  --skip-plugin          Skip plugin installation (board + CLAUDE.md only)
  --root <path>          Custom board location (default: git repository root)
  --shared <name>        Create a shared board at ~/.obeya/boards/<name>
                         (mutually exclusive with --agent)
```

### Standalone plugin install

```
ob plugin claude-install
```

Registers the obeya marketplace and installs the plugin via `claude` CLI. Same logic as `ob init --agent claude-code` plugin step.

## Flow Diagram

```
 User runs: ob init myboard --agent claude-code
 ┌──────────────────────────────────────────────────┐
 │                                                  │
 │  Required: --agent <name>                        │
 │  Optional: --columns, --skip-plugin,             │
 │            --root, --shared                      │
 │                                                  │
 └──────────────┬───────────────────────────────────┘
                │
                ▼
 ┌──────────────────────────────────┐
 │  --shared AND --agent provided?  │
 │  YES → ERROR: mutually exclusive │
 └──────┬───────────────────────────┘
        │ NO
        ▼
 ┌──────────────────────────────┐
 │  --shared only?              │
 └──────┬───────────────┬───────┘
        │ YES           │ NO
        ▼               ▼
 ┌──────────────┐  ┌─────────────────────────────────┐
 │ SHARED PATH  │  │  --agent provided?               │
 │              │  │  NO  → ERROR: required flag       │
 │ ~/.obeya/    │  │  YES → validate agent name        │
 │  boards/     │  │        unknown → ERROR             │
 │   <name>/    │  └──────────────┬────────────────────┘
 │    .obeya/   │                 │
 │     board.   │                 ▼
 │      json    │  ┌─────────────────────────────────┐
 │              │  │  RESOLVE ROOT DIRECTORY          │
 │ If exists:   │  │                                   │
 │  ERROR       │  │  --root given?                    │
 │              │  │    YES → use that path            │
 └──────────────┘  │    NO  → walk up from cwd         │
                   │          looking for .git/        │
                   │          (FindGitRoot)             │
                   └──────────────┬────────────────────┘
                                  │
                                  ▼
                   ┌─────────────────────────────────┐
                   │  INIT BOARD AT <root>            │
                   │                                   │
                   │  1. Check if board already exists  │
                   │     .obeya/board.json present?     │
                   │     YES → print "already init"     │
                   │     NO  → continue                 │
                   │                                   │
                   │  2. Create directory:              │
                   │     <root>/.obeya/                 │
                   │                                   │
                   │  3. Create board.json with:        │
                   │     - name (arg or "obeya")        │
                   │     - columns (custom or default)  │
                   └──────────────┬────────────────────┘
                                  │
                                  ▼
                   ┌─────────────────────────────────┐
                   │  AGENT-SPECIFIC SETUP            │
                   │  (dispatched via AgentSetup)      │
                   │                                   │
                   │  agent = "claude-code":           │
                   │                                   │
                   │  1. UPDATE CLAUDE.md               │
                   │     ├─ Has <!-- obeya:start --> ?  │
                   │     │  YES → REPLACE in-place      │
                   │     ├─ Has legacy section?         │
                   │     │  YES → REPLACE old section   │
                   │     └─ Neither?                    │
                   │        APPEND new section          │
                   │                                   │
                   │  2. INSTALL PLUGIN                 │
                   │     (unless --skip-plugin)         │
                   │     ├─ claude CLI in PATH?         │
                   │     │  NO → ERROR (board+CLAUDE.md │
                   │     │  already created; run        │
                   │     │  ob plugin claude-install    │
                   │     │  later)                      │
                   │     ├─ Register marketplace:       │
                   │     │  github.com/schoolofai/obeya │
                   │     ├─ Already installed?          │
                   │     │  YES → skip                  │
                   │     └─ Install:                    │
                   │        obeya@obeya-local           │
                   └──────────────┬────────────────────┘
                                  │
                                  ▼
                   ┌─────────────────────────────────┐
                   │  SUMMARY                         │
                   │                                   │
                   │  Board "myboard" initialized      │
                   │  Columns: backlog, todo, ...       │
                   │  CLAUDE.md updated                 │
                   │  Plugin obeya@obeya-local installed│
                   └───────────────────────────────────┘
```

## What Gets Created on Disk

### Local board with claude-code agent

```
<git-root>/
├── .obeya/
│   └── board.json          ← Kanban board (name, columns, items)
├── CLAUDE.md               ← Updated with obeya instructions
│   └── <!-- obeya:start --> v5
│       ... task tracking docs ...
│       <!-- obeya:end -->
└── ... (rest of your project)
```

Plugin installed at: `~/.claude/plugins/cache/...`

### Shared board (`--shared myteam`)

```
~/.obeya/
└── boards/
    └── myteam/
        └── .obeya/
            └── board.json
```

No CLAUDE.md or plugin — shared boards are storage only.

## Key Behaviors

- **`--agent` is required** for non-shared boards. Without it, `ob init` errors with a list of supported agents.
- **`--shared` and `--agent` are mutually exclusive.** Shared boards are storage-only and don't support agent integration.
- **Idempotent for board**: If `.obeya/board.json` already exists, it prints a message but does not error — it still proceeds to agent setup.
- **Idempotent for CLAUDE.md**: Uses `<!-- obeya:start/end -->` markers to find and replace existing sections, so re-running `ob init` upgrades the instructions to the latest version (v5) without duplicating.
- **Partial success on missing claude CLI**: Board and CLAUDE.md are created even if the `claude` CLI is not found. The plugin step fails with a message to run `ob plugin claude-install` later.
- **Git-root auto-detection**: Without `--root`, it walks up the directory tree from cwd looking for `.git/` to anchor the board at the repository root.

## Agent Architecture

```
AgentSetup interface
├── Name() string
└── Setup(ctx AgentContext) error

AgentContext
├── Root       string   // project root directory
├── BoardName  string   // board name
└── SkipPlugin bool     // --skip-plugin flag

Registry: map[string]AgentSetup
├── "claude-code" → ClaudeCodeSetup
└── (future agents register here)
```

## Source Files

| File | Purpose |
|------|---------|
| `cmd/init.go` | CLI command, flag handling, delegates to agent setup |
| `cmd/plugin.go` | `sync` and `claude-install` subcommands |
| `internal/agent/agent.go` | `AgentSetup` interface, `AgentContext`, registry |
| `internal/agent/claudecode.go` | `ClaudeCodeSetup` — CLAUDE.md logic + plugin install via `claude` CLI |
| `internal/store/json_store.go` | `InitBoard()` — creates `.obeya/` dir and `board.json` |
| `internal/store/root.go` | `FindGitRoot()` — walks up directory tree for `.git/` |
| `internal/domain/board.go` | `NewBoard()` / `NewBoardWithColumns()` — board constructor |
