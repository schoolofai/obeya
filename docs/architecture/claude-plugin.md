# Obeya Claude Plugin Architecture

## Overview

The Obeya Claude Code plugin is a **multi-layer integration** that injects task-tracking discipline into every Claude Code session. It uses **hooks** for runtime context injection, **skills** for board commands, and **CLAUDE.md** for persistent project rules.

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLAUDE CODE SESSION                         │
│                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────────┐  │
│  │ System Prompt │ + │  CLAUDE.md   │ + │  Hook Injections   │  │
│  │  (built-in)  │   │  (project)   │   │  (plugin context)  │  │
│  └──────────────┘   └──────────────┘   └────────────────────┘  │
│         │                  │                     │              │
│         └──────────────────┴─────────────────────┘              │
│                            │                                    │
│                    Combined Context                              │
│                    Claude sees ALL of this                       │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    13 SKILLS                              │   │
│  │  ob-status │ ob-pick │ ob-done │ ob-create │ ob-subtask  │   │
│  │  ob-edit   │ ob-show │ ob-block│ ob-unblock│ ob-plan     │   │
│  │  ob        │ ob-breakdown                                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                     Invoke `ob` CLI                              │
│                            ▼                                    │
│                  ┌──────────────────┐                            │
│                  │ .obeya/board.json│                            │
│                  └──────────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Plugin File Structure

```
obeya-plugin/
├── .claude-plugin/
│   ├── plugin.json          ← name, version, author
│   └── marketplace.json     ← marketplace metadata
├── hooks/
│   ├── hooks.json           ← declares SessionStart + PostToolUse
│   ├── run-hook.cmd         ← cross-platform dispatcher (bash/cmd polyglot)
│   ├── session-start        ← bash: checks board, injects context
│   └── post-tool-use        ← bash: detects plan files, injects reminder
└── skills/
    ├── ob/SKILL.md
    ├── ob-status/SKILL.md
    ├── ob-pick/SKILL.md
    ├── ob-done/SKILL.md
    ├── ob-create/SKILL.md
    ├── ob-subtask/SKILL.md
    ├── ob-edit/SKILL.md
    ├── ob-show/SKILL.md
    ├── ob-block/SKILL.md
    ├── ob-unblock/SKILL.md
    ├── ob-plan/SKILL.md
    └── ob-breakdown/SKILL.md
```

---

## The Two Hooks

### Hook 1: SessionStart (fires on every session start/resume/clear)

```
 User starts Claude Code session
           │
           ▼
 ┌─────────────────────────┐
 │  hooks.json triggers:   │
 │  SessionStart hook       │
 │  matcher: startup|       │
 │  resume|clear|compact    │
 │  async: false (blocks)   │
 └────────┬────────────────┘
          ▼
 ┌─────────────────────────┐     ┌──────────────────────┐
 │  session-start script   │────▶│ .obeya/board.json    │
 │  checks if board exists │     │ exists?              │
 └────────┬────────────────┘     └──────────────────────┘
          │
     ┌────┴────┐
     │         │
  EXISTS    MISSING
     │         │
     ▼         ▼
 Counts     Returns:
 in-prog    "Run ob init"
 tasks
     │
     ▼
 Returns JSON:
 ┌──────────────────────────────────────────────────────────┐
 │ { "additional_context": "...injected text..." }          │
 └──────────────────────────────────────────────────────────┘
```

**What gets injected into the prompt (verbatim):**

```
┌──────────────────── <system-reminder> ─────────────────────────┐
│                                                                 │
│  SessionStart hook additional context:                          │
│                                                                 │
│  <IMPORTANT>                                                    │
│  Obeya Task Tracking is ACTIVE for this project.                │
│                                                                 │
│  You MUST invoke the obeya:ob-status skill BEFORE invoking      │
│  any other skill — including superpowers skills.                 │
│                                                                 │
│  The sequence is: ob-status FIRST, then other skills.           │
│  If ob-status shows no task, invoke obeya:ob-create.            │
│                                                                 │
│  When ANY workflow completes work, invoke obeya:ob-done.        │
│                                                                 │
│  CLAUDE.md instructions have highest priority.                  │
│  </IMPORTANT>                                                   │
│                                                                 │
│  ACTIVE WORK: N task(s) in-progress.                            │
│  Invoke obeya:ob-status to see them.                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key detail:** `async: false` means this hook **blocks** the session from starting until the context is injected. Claude cannot act before it sees these instructions.

### Hook 2: PostToolUse (fires after every Write/Edit)

```
 Claude writes/edits a file
           │
           ▼
 ┌─────────────────────────┐
 │  hooks.json triggers:   │
 │  PostToolUse hook        │
 │  timeout: 10s            │
 │  async: true             │
 └────────┬────────────────┘
          ▼
 ┌─────────────────────────────┐
 │  post-tool-use script       │
 │  reads stdin (tool_input)   │
 │  checks tool_name == Write  │
 │         or Edit             │
 └────────┬────────────────────┘
          │
     ┌────┴────────┐
     │             │
  file matches   no match
  plan|spec|     │
  design|prd|    ▼
  brief.md      Returns {}
     │          (no injection)
     ▼
 Returns:
 ┌──────────────────────────────────────────────────────────┐
 │ "A plan document was just written. You MUST now:         │
 │  (1) Import it with ob plan import                       │
 │  (2) Break it down into epic/story/task hierarchy        │
 │  (3) Link all items to the plan.                         │
 │  Do NOT proceed to implementation without board tasks."  │
 └──────────────────────────────────────────────────────────┘
```

**Key detail:** `async: true` means this hook runs **non-blocking**. It's a reminder, not a gate — Claude continues working but receives the context as a follow-up system reminder.

**File pattern matched:** `(plan|spec|design|prd|brief)\.(md|txt)$`

---

## How Context Layers Stack in the Prompt

```
Priority (highest to lowest):

 ┌─────────────────────────────────────────────────────────────┐
 │ 1. User's explicit instructions (CLAUDE.md)                 │  <-- Persistent
 │    ┌─────────────────────────────────────────────────────┐  │      in repo
 │    │ <!-- obeya:start --> v5                             │  │
 │    │ "Every piece of work MUST have a task on the board" │  │
 │    │ "ob-status -> ob-pick -> work -> ob-done"           │  │
 │    │ "Board is authoritative over TodoWrite"             │  │
 │    │ <!-- obeya:end -->                                  │  │
 │    └─────────────────────────────────────────────────────┘  │
 ├─────────────────────────────────────────────────────────────┤
 │ 2. SessionStart hook injection                              │  <-- Per-session
 │    "Obeya Task Tracking is ACTIVE"                          │      runtime
 │    "ob-status FIRST, then other skills"                     │      injection
 │    "ACTIVE WORK: N tasks in-progress"                       │
 ├─────────────────────────────────────────────────────────────┤
 │ 3. Skill descriptions (from SKILL.md files)                 │  <-- Registered
 │    "obeya:ob-status - Show tasks assigned to you"           │      at install
 │    "obeya:ob-done - Mark current task as done"              │
 │    ... 13 skills listed in system-reminder ...              │
 ├─────────────────────────────────────────────────────────────┤
 │ 4. PostToolUse hook injection (conditional)                 │  <-- Reactive,
 │    Only fires when plan files are created/edited            │      on specific
 │    "Break down plan into board tasks"                       │      tool use
 └─────────────────────────────────────────────────────────────┘
```

**Double reinforcement:** CLAUDE.md provides persistent rules that survive across sessions. The SessionStart hook provides a per-session reminder with live task counts. Together, they make the instructions very difficult for the model to ignore.

---

## The 13 Skills — Grouped by Purpose

### Lifecycle

```
/ob-status ──▶ /ob-pick ──▶ [work] ──▶ /ob-done
"what am I      "claim a      ...       "mark it
 working on?"    task"                   complete"
```

### Creation

| Skill | Purpose |
|---|---|
| `/ob-create` | Create top-level epic, story, or task |
| `/ob-subtask` | Create child under a parent item |
| `/ob-breakdown` | Plan document -> full epic/story/task hierarchy |

### Management

| Skill | Purpose |
|---|---|
| `/ob-edit` | Update title, description, priority |
| `/ob-show` | View full details of an item |
| `/ob-block` | Report a dependency blocker |
| `/ob-unblock` | Remove a blocker |

### Planning & Overview

| Skill | Purpose |
|---|---|
| `/ob-plan` | Import, show, or link plan documents |
| `/ob` | Show the full Kanban board overview |

Each skill is a `SKILL.md` file that wraps `ob` CLI commands. Skills have two key flags:
- `user-invocable: true` — user can invoke manually via `/ob-done`
- `disable-model-invocation: false` — Claude can invoke proactively based on context

---

## End-to-End Flow: What Claude Sees

```
┌─ SESSION START ──────────────────────────────────────────────────────┐
│                                                                       │
│  1. Claude Code loads CLAUDE.md ──▶ "Obeya rules: track all work"    │
│                                                                       │
│  2. SessionStart hook fires  ──▶ "ob-status MUST come first"         │
│                                   "ACTIVE WORK: 2 tasks"             │
│                                                                       │
│  3. Skills registered        ──▶ 13 skills appear in system-reminder │
│                                                                       │
├─ USER SAYS: "Add auth to the app" ──────────────────────────────────┤
│                                                                       │
│  4. Claude sees hook context: "invoke ob-status FIRST"               │
│     -> invokes /ob-status                                            │
│     -> sees: task #7 "Add auth" already exists, in-progress          │
│                                                                       │
│  5. Claude works, writes plan file: auth-design.md                   │
│                                                                       │
│  6. PostToolUse hook fires   ──▶ "Plan written! Import + breakdown"  │
│                                                                       │
│  7. Claude invokes /ob-plan import, /ob-breakdown                    │
│     -> creates epic, stories, tasks on board                         │
│                                                                       │
│  8. Claude implements, then invokes /ob-done                         │
│     -> task #7 moves to done column                                  │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Installation & Plugin Sync

### Installation Methods

```
# Local development (from source)
claude plugin marketplace add /path/to/obeya/obeya-plugin
claude plugin install obeya@obeya-local

# Quick test (no install)
claude --plugin-dir ./obeya-plugin

# Public distribution (when published)
claude plugin install obeya --scope user
```

### Live Development with Plugin Sync

```
ob plugin sync
```

This command:
1. Finds the `obeya-plugin/` directory in the git root
2. Reads `~/.claude/plugins/installed_plugins.json` to locate the installed copy
3. Replaces the cached copy with a **symlink** pointing to source
4. Changes take effect after Claude Code session restart

```
~/.claude/plugins/cache/obeya/
         │
         │  (symlink replaces copy)
         ▼
/path/to/obeya/obeya-plugin/
```

This means editing hook scripts or skill files in the source directory immediately takes effect on next session — no reinstall needed.

---

## CLAUDE.md Management

The `ob init` command injects an Obeya section into the project's `CLAUDE.md` using versioned markers:

```markdown
<!-- obeya:start --> v5
## Task Tracking — Obeya
... rules and workflows ...
<!-- obeya:end -->
```

Running `ob init` again safely replaces only the content between the markers, preserving any custom sections the user has added above or below. The version number (`v5`) tracks the instruction format.

---

## Data Flow Summary

```
hooks.json
    │
    ├──▶ SessionStart ──▶ session-start (bash)
    │                         │
    │                    reads .obeya/board.json
    │                         │
    │                    returns { additional_context: "..." }
    │                         │
    │                    Claude sees it as <system-reminder>
    │
    └──▶ PostToolUse ──▶ post-tool-use (bash)
                              │
                         reads stdin (tool_input JSON)
                              │
                         matches file pattern?
                              │
                         yes: returns { additional_context: "..." }
                         no:  returns {}
```

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| **Hooks are bash scripts** | Portable, no compilation needed, fast execution |
| **SessionStart is sync** | Must block session — context must be present before Claude acts |
| **PostToolUse is async** | Non-blocking — just a reminder, doesn't gate tool execution |
| **Skills are `.md` files, not MCP** | No server needed, simpler distribution, Claude reads markdown natively |
| **CLAUDE.md uses versioned markers** | `ob init` can safely re-run without overwriting user's custom sections |
| **Plugin sync uses symlinks** | Live development — edit source, restart session, changes appear |
| **Double reinforcement** | CLAUDE.md (persistent rules) + hook injection (session reminder) = hard to ignore |
