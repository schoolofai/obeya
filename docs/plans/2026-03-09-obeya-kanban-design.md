# Obeya — CLI Kanban Board Manager Design

## Overview

A CLI-based Kanban board manager serving both humans (via TUI) and AI agents (via CLI). Two editions: **Lite** (local JSON storage) and **Pro** (cloud-hosted, future). Built in Go.

## Architecture

```
┌─────────────────────────────────────────────┐
│                  CLI (ob)                    │
│  Cobra commands: create, move, list, etc.   │
├─────────────────────────────────────────────┤
│                  TUI (ob tui)               │
│  Bubble Tea: minimal board view + keys      │
├─────────────────────────────────────────────┤
│              Core Domain Layer              │
│  Board, Epic, Story, Task, Dependency       │
├─────────────────────────────────────────────┤
│           Storage Interface                 │
│  ┌──────────────┐  ┌────────────────────┐   │
│  │  Lite: JSON  │  │  Pro: Cloud API    │   │
│  │  local file  │  │  (future)          │   │
│  └──────────────┘  └────────────────────┘   │
└─────────────────────────────────────────────┘
```

### Key Principles

- **Storage interface** abstracts Lite (JSON) vs Pro (cloud API)
- **Domain layer is storage-agnostic** — all business logic lives here
- **CLI and TUI are thin layers** — parse input, call domain functions
- **Fail fast, no fallbacks** — clear errors, no silent failures

### Project Structure

```
obeya/
├── cmd/                  # Cobra command definitions
│   ├── root.go
│   ├── init.go
│   ├── create.go
│   ├── move.go
│   ├── list.go
│   ├── show.go
│   ├── assign.go
│   ├── block.go
│   └── tui.go
├── internal/
│   ├── domain/           # Core types: Board, Item, Identity, enums
│   ├── store/            # Storage interface + JSON implementation
│   ├── engine/           # Business logic (create, move, validate)
│   └── tui/              # Bubble Tea TUI components
├── skill/                # Provider-agnostic agent skill
│   └── obeya.md
├── go.mod
├── go.sum
└── main.go
```

## Data Model

### Item (unified type for epic, story, task)

| Field | Type | Description |
|---|---|---|
| ID | string | Canonical short hash (e.g. "a3f8b2") |
| DisplayNum | int | Auto-incrementing human-friendly number |
| Type | ItemType | "epic", "story", "task" |
| Title | string | Short description |
| Description | string | Optional longer detail |
| Status | string | Current column name |
| Priority | Priority | "low", "medium", "high", "critical" |
| Assignee | string | User ID (identity hash) |
| ParentID | string | Empty for top-level epics |
| BlockedBy | []string | List of item IDs (simple blockers) |
| Tags | []string | Freeform tags for filtering |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last modification timestamp |
| History | []ChangeRecord | Audit trail of changes |

### Board (root container)

| Field | Type | Description |
|---|---|---|
| Version | int | Optimistic concurrency version |
| Name | string | Board name |
| Columns | []Column | Ordered list of statuses |
| Items | map[string]*Item | Keyed by canonical ID |
| DisplayMap | map[int]string | Display number to canonical ID |
| NextDisplay | int | Next display number to assign |
| Users | map[string]*Identity | Registered users/agents |
| AgentRole | string | "admin" or "contributor" |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last modification timestamp |

### Column

| Field | Type | Description |
|---|---|---|
| Name | string | e.g. "backlog", "in-progress" |
| Limit | int | Optional WIP limit (0 = unlimited) |

### Identity

| Field | Type | Description |
|---|---|---|
| ID | string | Unique short hash |
| Name | string | Display name |
| Type | string | "human" or "agent" |
| Provider | string | For agents: "claude-code", "opencode", "codex". For humans: "local" |

### ChangeRecord (audit trail)

| Field | Type | Description |
|---|---|---|
| UserID | string | Persistent identity ID |
| SessionID | string | Ephemeral session identifier |
| Action | string | "created", "moved", "assigned", etc. |
| Detail | string | e.g. "status: todo -> in-progress" |
| Timestamp | time.Time | When the change occurred |

### Task Hierarchy

Flexible parent-child with soft conventions:
- `ob create epic` — top-level container
- `ob create story -p <epic>` — story under an epic
- `ob create task -p <story>` — task under a story
- No enforced nesting rules — flexibility when needed

### Dependencies

Simple blockers: an item declares `blocked_by` with a list of item IDs. No typed relationships for now.

## ID System

- **Canonical ID**: Short hash generated on creation (e.g. "a3f8b2")
- **Display number**: Auto-incrementing integer alias (e.g. 1, 2, 3)
- **CLI accepts either**: `ob show 3` or `ob show a3f`
- **Pro scoping**: Display numbers scoped per-board on the server

## Concurrency

### Lite (local JSON)

- **File-level locking**: `flock` advisory lock on `.obeya/board.lock`
- **Optimistic versioning**: Board carries a `version` field, checked on write
- **Atomic writes**: Write to `board.json.tmp`, then `os.Rename()` to `board.json`
- **Short lock duration**: Lock held only during read-modify-write, not during user interaction

### Pro (cloud, future)

- **ETag-based optimistic concurrency**: `If-Match` header on PUT requests
- **409 Conflict on stale writes**: Fail fast, no silent retry

### Storage Interface

```go
type Store interface {
    Transaction(fn func(board *Board) error) error
    LoadBoard() (*Board, error)
    InitBoard(config BoardConfig) error
    BoardExists() bool
}
```

## Identity & Sessions

- **Persistent identity per provider**: Agent registers once (e.g. "Claude Code" = user `b3a`)
- **Session tracking**: Each command logs a `session_id` in the change record
- **Self-identification**: Via `OB_USER` and `OB_SESSION` env vars, or `--as` / `--session` flags

## CLI Command Reference

```bash
# Board management
ob init                                  # Create .obeya/board.json
ob init --columns "todo,doing,done"      # Init with custom columns
ob board config                          # Show/edit board settings
ob board columns add <name>
ob board columns remove <name>
ob board columns reorder <n1,n2,n3>

# User management
ob user add "Name" --type human
ob user add "Name" --type agent --provider claude-code
ob user list
ob user remove <id>

# Item CRUD
ob create epic "Title"
ob create story "Title" -p <parent>
ob create task "Title" -p <parent>
ob create task "Title" -p <parent> --priority high --assign <user> --tag backend

# Item operations
ob move <id> <status>
ob assign <id> --to <user>
ob edit <id> --title "New title"
ob edit <id> --description "Details"
ob edit <id> --priority critical
ob delete <id>                           # Fails if item has children

# Dependencies
ob block <id> --by <id>
ob unblock <id> --by <id>

# Querying
ob list                                  # Tree view
ob list --flat                           # Flat list
ob list --status in-progress
ob list --assignee <user>
ob list --type epic
ob list --tag backend
ob list --blocked
ob show <id>                             # Full detail + children + history

# Output formats
ob list --format json
ob show <id> --format json

# TUI
ob tui

# Skill management
ob skill install                         # Auto-detect providers
ob skill install --provider claude-code
ob skill install --list
```

### Global flags on mutating commands

- `--as <user-id>` — identity (or `OB_USER` env var)
- `--session <session-id>` — session tracking (or `OB_SESSION` env var)

## Agent Skill

Provider-agnostic markdown file installed per provider:
- Claude Code: `~/.claude/skills/obeya.md`
- OpenCode: loaded via system prompt/config
- Codex: included in instruction context

### Skill content covers

1. **Setup** — env vars, board state discovery
2. **Command reference** — all commands with examples
3. **Permissions** — respect `AgentRole` config (admin vs contributor)
4. **Workflow conventions**:
   - On session start: check assigned work, pick a task, move to in-progress
   - During work: create subtasks, report blockers
   - On completion: move to done, update parent if applicable

### Installation

`ob skill install` detects available providers and copies the skill to the correct location for each.

## TUI (Minimal v1)

- Basic board view: columns with task titles
- Keyboard navigation for moving between columns/tasks
- Simple key commands for create/move/assign
- Will iterate toward full interactivity (lazygit-style) in future versions

## Design Decisions Summary

| Decision | Choice |
|---|---|
| Language | Go (Cobra + Bubble Tea) |
| Storage (Lite) | Local JSON file (.obeya/board.json) |
| Storage (Pro) | Cloud API (future, same Store interface) |
| Columns | Customizable, default 5-column (backlog, todo, in-progress, review, done) |
| Task hierarchy | Flexible parent-child with soft conventions (Epic -> Story -> Task) |
| Dependencies | Simple blocked_by list |
| IDs | Canonical short hash + display number alias |
| Concurrency | File lock + optimistic versioning (Lite), ETag-based (Pro) |
| Identity | Persistent per-provider + session tracking |
| CLI command | ob |
| TUI | Minimal for now |
| Agent skill | Provider-agnostic markdown, auto-install per provider |
| Agent permissions | Configurable: admin or contributor |
| Error handling | Fail fast, no fallbacks |
