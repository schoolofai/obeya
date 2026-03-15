# Obeya MCP Server

**Command**: `ob mcp serve`
**Transport**: stdio (JSON-RPC 2.0)
**SDK**: [mcp-go](https://github.com/mark3labs/mcp-go) v0.45.0

The MCP server exposes every Obeya board operation to any MCP-compatible host — Claude Code, Claude Desktop, Cursor, Windsurf, ChatGPT, and custom agents.

```
┌──────────────────────┐      ┌──────────────────────┐
│ Claude Code          │      │ Cursor / Windsurf    │
│ Claude Desktop       │      │ ChatGPT / Custom     │
└──────┬───────────────┘      └──────────┬───────────┘
       │  stdio (JSON-RPC 2.0)          │
       └────────────┬───────────────────┘
                    ▼
           ┌──────────────────┐
           │  ob mcp serve    │
           ├──────────────────┤
           │ 21 Tools         │
           │  5 Resources     │
           │  4 Prompts       │
           ├──────────────────┤
           │ Engine → Store   │
           └────┬────────┬────┘
                │        │
         ┌──────┘        └──────┐
         ▼                      ▼
    ┌──────────┐         ┌───────────┐
    │ JSON     │         │ Cloud API │
    │ Store    │         │ (Appwrite)│
    └──────────┘         └───────────┘
```

---

## Quick Start

### 1. Install

```bash
brew tap schoolofai/tap
brew install obeya
```

### 2. Initialize a board

```bash
cd your-project
ob init --agent claude-code
```

### 3. Configure your MCP host

**Claude Code** — add to `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "obeya": {
      "command": "ob",
      "args": ["mcp", "serve"]
    }
  }
}
```

**Claude Desktop** — add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "obeya": {
      "command": "/usr/local/bin/ob",
      "args": ["mcp", "serve", "--board", "/path/to/project"]
    }
  }
}
```

**Cursor** — Settings → MCP → Add Server:

```json
{
  "command": "ob",
  "args": ["mcp", "serve"]
}
```

The server starts, discovers the board from the working directory, and blocks on stdin until the host disconnects.

---

## Board Resolution

The MCP server resolves which board to operate on using the same logic as all `ob` commands. It checks three locations in priority order:

| Priority | Source | How it works |
|---|---|---|
| 1st | **Shared board** | Finds `.obeya-link` in cwd (or ancestors), follows it to `~/.obeya/boards/<name>/` |
| 2nd | **Local project board** | Finds `.obeya/board.json` in cwd (or ancestors) |
| 3rd | **Git root fallback** | Finds `.git` directory, uses that root |

After resolving the root directory, `NewStore()` checks for `.obeya/cloud.json`:
- **Present** (with `-tags cloud` build): creates a `CloudStore` connecting to the Appwrite API
- **Absent**: creates a `JSONStore` using the local `board.json` file

### Override with `--board`

```bash
ob mcp serve --board /path/to/project
ob mcp serve --board ~/.obeya/boards/my-shared-board
```

This changes the working directory before resolution, so it works with all board modes.

### All three modes work transparently

- **Local project**: Agent runs from project directory → `.obeya/board.json` found
- **Shared/global**: Agent runs from linked project → `.obeya-link` followed → `~/.obeya/boards/<name>/`
- **Cloud**: Board root found → `.obeya/cloud.json` detected → `CloudStore` created

No special flags or configuration needed — the server auto-detects the board mode.

---

## Identity Resolution

Every tool call needs a user identity for the audit trail. The server resolves identity at startup:

1. Check `OBEYA_USER` environment variable → resolve to board user ID
2. Check `OBEYA_AGENT_NAME` environment variable → resolve to board user ID
3. Fall back to first registered agent on the board
4. Fall back to first registered user of any type

Session IDs for history records come from:
1. `OB_SESSION` environment variable (if set)
2. `mcp-<pid>` (auto-generated from process ID)

---

## Tools Reference

### Item Management (9 tools)

#### `list_items`

List and filter board items.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `status` | string | no | Filter by column (e.g., `backlog`, `in-progress`, `done`) |
| `assignee` | string | no | Filter by assignee name or ID |
| `type` | string | no | Filter by type: `epic`, `story`, `task` |
| `tag` | string | no | Filter by tag |
| `blocked` | boolean | no | If true, only show blocked items |

**Returns**: JSON array of items with `display_num`, `type`, `title`, `status`, `priority`, `assignee`, `tags`, `blocked_by_count`.

---

#### `get_item`

Get full details of a single item including description, history, children, and linked plans.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Display number (e.g., `5`) or UUID |

**Returns**: Full item detail with `id`, `description`, `parent_id`, `children[]`, `linked_plans[]`, `history[]`, `created_at`, `updated_at`.

---

#### `create_item`

Create a new epic, story, or task. Items start in the first column (backlog).

| Parameter | Type | Required | Description |
|---|---|---|---|
| `type` | string | **yes** | `epic`, `story`, or `task` |
| `title` | string | **yes** | Short, descriptive title |
| `assignee` | string | **yes** | Name or ID of the assignee |
| `description` | string | no | Detailed description with context |
| `priority` | string | no | `low`, `medium`, `high`, or `critical` |
| `parent` | string | no | Parent item reference for nesting |

**Returns**: `{ created: true, display_num, id, title, type, status, assignee }`

---

#### `move_item`

Move an item to a different column.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item reference |
| `status` | string | **yes** | Target column name (e.g., `in-progress`, `done`) |

**Returns**: `"Moved #5 to in-progress"`

---

#### `edit_item`

Edit an item's title, description, or priority. Provide only fields to change.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item reference |
| `title` | string | no | New title |
| `description` | string | no | New description |
| `priority` | string | no | `low`, `medium`, `high`, or `critical` |

---

#### `delete_item`

Delete an item. Cannot delete items with children.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item reference |

---

#### `assign_item`

Assign or reassign an item to a user or agent.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item reference |
| `assignee` | string | **yes** | Name or ID of the assignee |

---

#### `block_item`

Mark an item as blocked by another item.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item to block |
| `blocker` | string | **yes** | Blocking item reference |

---

#### `unblock_item`

Remove a blocker from an item.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Blocked item |
| `blocker` | string | **yes** | Blocker to remove |

---

### User Management (2 tools)

#### `list_users`

List all registered users and agents on the board. No parameters.

**Returns**: Array of `{ id, name, type, provider }`.

---

#### `add_user`

Register a new user or agent identity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | **yes** | Display name |
| `type` | string | **yes** | `human` or `agent` |
| `provider` | string | no | Agent provider (e.g., `claude-code`, `cursor`) |

---

### Plan Management (4 tools)

#### `list_plans`

List all plans. No parameters.

**Returns**: Array of `{ display_num, title, linked_items }`.

---

#### `get_plan`

Get full plan details.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Plan reference |

**Returns**: `{ display_num, title, content, source_file, linked_items, created_at, updated_at }`

---

#### `create_plan`

Create a new plan document.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `title` | string | **yes** | Plan title |
| `content` | string | **yes** | Markdown content |

---

#### `link_plan`

Link board items to a plan for traceability.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `plan_ref` | string | **yes** | Plan reference |
| `item_refs` | string | **yes** | Comma-separated item references |

---

### Board Configuration (2 tools)

#### `list_columns`

List all columns with WIP limits and current counts. No parameters.

**Returns**: Array of `{ name, count, limit, level }` where `level` is `ok`, `warn`, or `over`.

---

#### `add_column`

Add a new status column.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | **yes** | Column name (e.g., `staging`, `qa`) |

---

### Metrics & Analytics (2 tools)

#### `get_metrics`

Get comprehensive board analytics.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `days` | number | no | Days for velocity history (default: 14) |

**Returns**:
```json
{
  "total_items": 42,
  "done_items": 18,
  "cycle_time": "2d 4h",
  "lead_time": "5d 12h",
  "throughput": { "this_week": 5, "last_week": 3, "per_week": 4.2, "total": 18 },
  "wip": [{ "name": "in-progress", "count": 3, "limit": 5, "level": "ok" }],
  "dwell": { "backlog": { "avg": "3d", "count": 12 } },
  "daily_velocity": [{ "date": "2026-03-01", "count": 2 }]
}
```

---

#### `get_burndown`

Get burndown chart data for an epic showing actual vs ideal progress.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `epic_ref` | string | **yes** | Epic reference |

**Returns**: Array of `{ date, remaining, ideal }` data points.

---

### Workflow Tools (3 tools)

These are composite operations that combine multiple engine calls for common agent workflows.

#### `pick_next_task`

Find the highest-priority unassigned, unblocked task, assign it to you, and move to in-progress.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `from_column` | string | no | Column to pick from (default: `todo`) |

**Logic**:
1. List items in the target column
2. Filter to unassigned and unblocked
3. Sort by priority (critical > high > medium > low)
4. Assign to the server's resolved identity
5. Move to `in-progress`

**Returns**: The picked item's details, or a message if no items are available.

---

#### `complete_task`

Mark a task as done with optional completion notes.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ref` | string | **yes** | Item reference |
| `notes` | string | no | Completion notes appended to description |

**Logic**:
1. If notes provided, append to item description with `---` separator
2. Move item to `done`

---

#### `board_summary`

Get a concise board overview — ideal for standup context. No parameters.

**Returns**:
```json
{
  "board_name": "my-project",
  "total_items": 42,
  "done_items": 18,
  "columns": { "backlog": 10, "todo": 5, "in-progress": 3, "review": 2, "done": 18 },
  "blocked_items": [{ "num": 7, "title": "Auth flow", "blocked_by": 1 }],
  "alerts": ["'review' is over WIP limit (4/3)"],
  "throughput": { "this_week": 5, "last_week": 3 },
  "cycle_time": "2d 4h"
}
```

---

## Resources

Resources expose read-only board data that hosts can inject into agent context windows. All return `application/json`.

| URI | Description |
|---|---|
| `obeya://board/summary` | Board name, total/done item counts, items per column |
| `obeya://board/items` | All items as a sorted JSON array |
| `obeya://metrics` | Cycle time, lead time, throughput |
| `obeya://users` | All registered users and agents |
| `obeya://plans` | All plans with linked item counts |

Resources complement tools: hosts can pre-populate agent context with board state (via resources), then agents call tools to take actions.

---

## Prompts

Prompts are templates that users invoke to get structured agent behavior with board context.

### `daily_standup`

Generates a standup report from current board state. No arguments.

Injects: in-progress items, items in review, blocked items, throughput metrics.

### `sprint_planning`

Analyzes the backlog and suggests items for the next sprint.

| Argument | Required | Description |
|---|---|---|
| `capacity` | no | Number of items the team can complete |

### `triage_new_work`

Helps structure new work into properly typed board items with priorities and hierarchy.

| Argument | Required | Description |
|---|---|---|
| `description` | **yes** | Description of the work to triage |

### `retrospective`

Generates a retrospective analysis from board metrics — what went well, bottlenecks, and suggestions.

---

## Architecture

### Package Layout

```
internal/mcp/
├── server.go         # Server struct, tool/resource/prompt registration and handlers
└── server_test.go    # 18 tests covering all tool categories + mcptest integration

cmd/mcp.go            # `ob mcp serve` command with --board flag
```

### How It Works

```
ob mcp serve
    │
    ├── getEngine()
    │   ├── FindProjectRoot(cwd)    ← resolves local, shared, or cloud board
    │   └── NewStore(root, "")      ← JSONStore or CloudStore
    │
    ├── mcp.New(engine)
    │   ├── resolveIdentity()       ← finds agent user for audit trail
    │   ├── registerTools()         ← 21 tools
    │   ├── registerResources()     ← 5 resources
    │   └── registerPrompts()       ← 4 prompts
    │
    └── mcpserver.ServeStdio(srv)   ← blocks, handles JSON-RPC on stdin/stdout
```

### Tool → Engine Mapping

Every tool handler delegates to `engine.Engine` methods. The engine handles all business logic: validation, parent resolution, user resolution, history tracking, and transactional persistence. The MCP layer is a thin adapter that:

1. Extracts parameters from `CallToolRequest`
2. Calls one or more engine methods
3. Formats the result as JSON
4. Returns a `CallToolResult`

Errors from the engine are returned as MCP tool errors (`IsError: true`), not protocol errors — this lets the agent see the error message and retry.

### Identity Flow

```
Environment variable (OBEYA_USER / OBEYA_AGENT_NAME)
         │
         ▼ resolve against board users
    ┌────────────┐
    │  board     │ ← first agent, then first user
    │  .Users    │
    └────────────┘
         │
         ▼
    s.userID    ← used for all MoveItem, EditItem, etc. calls
    s.sessionID ← OB_SESSION env var or "mcp-<pid>"
```

---

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `OBEYA_USER` | Resolve agent identity by name | (none) |
| `OBEYA_AGENT_NAME` | Alternative identity lookup | (none) |
| `OB_SESSION` | Session ID for history records | `mcp-<pid>` |

---

## Testing

The MCP server has 18 tests in `internal/mcp/server_test.go`:

| Test | What it covers |
|---|---|
| `TestCreateAndListItems` | Create a task, list it back |
| `TestMoveItem` | Move item between columns |
| `TestEditItem` | Edit title and priority |
| `TestDeleteItem` | Delete and verify removal |
| `TestAssignItem` | Reassign to different user |
| `TestBlockUnblock` | Block and unblock flow |
| `TestListUsers` | List registered users |
| `TestPlanOperations` | Create and list plans |
| `TestListColumns` | Column listing with WIP |
| `TestGetMetrics` | Metrics after completing items |
| `TestBoardSummary` | Summary output structure |
| `TestPickNextTask` | Empty column behavior |
| `TestCompleteTask` | Complete with notes |
| `TestFilterByStatus` | Status-based item filtering |
| `TestMCPServerRegistration` | Server creation |
| `TestAddColumn` | Column addition |
| `TestCreateItemError` | Error handling for missing fields |
| `TestMCPTestIntegration` | End-to-end via mcptest (stdio pipes) |

Run them:

```bash
go test ./internal/mcp/ -v
```

Or as part of the full suite:

```bash
./scripts/test.sh
```

---

## Typical Agent Workflow

An agent using the MCP server follows this loop:

```
1. board_summary          ← understand current state
2. pick_next_task          ← claim work
3. get_item ref            ← read full description
4. (agent does the work — writes code, runs tests)
5. complete_task ref       ← mark done with notes
6. pick_next_task          ← next item
```

For planning work:

```
1. create_plan             ← document the design
2. create_item (epic)      ← high-level goal
3. create_item (story)     ← deliverable under epic
4. create_item (task) ×N   ← atomic work items under story
5. link_plan               ← connect tasks to plan
6. pick_next_task          ← start executing
```

---

## Tool Summary

| Tool | Mutates | Category |
|---|---|---|
| `list_items` | No | Items |
| `get_item` | No | Items |
| `create_item` | Yes | Items |
| `move_item` | Yes | Items |
| `edit_item` | Yes | Items |
| `delete_item` | Yes | Items |
| `assign_item` | Yes | Items |
| `block_item` | Yes | Items |
| `unblock_item` | Yes | Items |
| `list_users` | No | Users |
| `add_user` | Yes | Users |
| `list_plans` | No | Plans |
| `get_plan` | No | Plans |
| `create_plan` | Yes | Plans |
| `link_plan` | Yes | Plans |
| `list_columns` | No | Board |
| `add_column` | Yes | Board |
| `get_metrics` | No | Metrics |
| `get_burndown` | No | Metrics |
| `pick_next_task` | Yes | Workflow |
| `complete_task` | Yes | Workflow |
| `board_summary` | No | Workflow |
