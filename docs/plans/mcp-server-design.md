# MCP Server Design — Obeya

**Date**: 2026-03-15
**Status**: Draft
**Story**: #2 from Product Backlog v1

---

## 1. Vision

Make Obeya the **universal task management backend for AI agents** by exposing every board operation via the Model Context Protocol (MCP). Any MCP-compatible host — Claude Code, Claude Desktop, Claude.ai, ChatGPT, Cursor, Windsurf, custom agents — can discover and use Obeya with zero custom integration.

```
┌──────────────────────┐      ┌──────────────────────┐
│ Claude Code          │      │ Cursor / Windsurf    │
│ Claude Desktop       │      │ ChatGPT              │
│ Claude.ai            │      │ Custom Agents        │
└──────┬───────────────┘      └──────────┬───────────┘
       │  stdio / streamable-HTTP        │
       │                                 │
       ▼                                 ▼
  ┌─────────────────────────────────────────┐
  │         ob mcp serve                    │
  │     Obeya MCP Server (Go)              │
  ├─────────────────────────────────────────┤
  │  Tools    │ Resources │ Prompts         │
  ├───────────┤───────────┤─────────────────┤
  │  Engine   │ Store     │ Metrics         │
  └───────────┴───────────┴─────────────────┘
       │                │
       ▼                ▼
  ┌──────────┐    ┌──────────────┐
  │ JSON     │    │ Cloud API    │
  │ Store    │    │ (Appwrite)   │
  └──────────┘    └──────────────┘
```

## 2. Architecture

### 2.1 Transport Strategy

| Transport | Use Case | Command |
|---|---|---|
| **stdio** (primary) | Local IDE/agent integration | `ob mcp serve` |
| **streamable-HTTP** (future) | Remote/cloud hosted boards | `ob mcp serve --http :8090` |

**Phase 1**: stdio only — covers Claude Code, Claude Desktop, Cursor, Windsurf, ChatGPT.
**Phase 2**: streamable-HTTP — enables cloud-hosted Obeya boards as remote MCP servers.

### 2.2 SDK Choice

Use the **official Go SDK**: `github.com/modelcontextprotocol/go-sdk/mcp`

Rationale:
- Maintained by Anthropic + Google
- Full spec 2025-11-25 support
- Handles JSON-RPC 2.0, capability negotiation, transport abstraction
- Type-safe tool/resource registration

### 2.3 Package Layout

```
internal/mcp/
├── server.go         # MCP server setup, tool/resource/prompt registration
├── tools.go          # All tool handlers
├── resources.go      # All resource handlers
├── prompts.go        # All prompt handlers
├── context.go        # Request context (board resolution, identity)
└── server_test.go    # Tests

cmd/mcp.go            # `ob mcp serve` command
```

### 2.4 Board Resolution

The MCP server needs to know which board to operate on. Strategy:

1. **Local mode** (default): Resolve board from CWD using existing `store.Resolve()` — same as `ob list` today
2. **Explicit path**: `ob mcp serve --board /path/to/project` — serve a specific local board
3. **Cloud mode**: `ob mcp serve --cloud` — read `cloud.json` from CWD, connect to cloud API
4. **Multi-board** (phase 2): Accept board ID as a tool parameter for cloud users with multiple boards

For phase 1, the server binds to **one board** determined at startup — matching how all other `ob` commands work.

### 2.5 Identity Resolution

Every MCP tool call needs a user identity for history tracking. Strategy:

1. Server detects the calling agent from `MCP_CLIENT_NAME` header or initialization info
2. Falls back to environment variable: `OBEYA_USER` or `OBEYA_AGENT_NAME`
3. Falls back to the first registered user on the board
4. Auto-registers the agent identity if it doesn't exist (type: `agent`, provider from MCP client info)

This enables zero-config operation — an agent connects, Obeya auto-detects who it is, and all history entries are properly attributed.

---

## 3. MCP Tools (Model-Controlled Operations)

These are the core operations agents invoke. Each maps directly to an existing `engine.Engine` method.

### 3.1 Item Management

#### `list_items`
List and filter board items.

```json
{
  "name": "list_items",
  "description": "List items on the Obeya board with optional filters. Returns all items matching the criteria. Use this to see what tasks exist, check status, find unassigned work, or review a specific column.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "status": {
        "type": "string",
        "description": "Filter by column status (e.g., 'backlog', 'todo', 'in-progress', 'review', 'done')"
      },
      "assignee": {
        "type": "string",
        "description": "Filter by assignee name or ID"
      },
      "type": {
        "type": "string",
        "enum": ["epic", "story", "task"],
        "description": "Filter by item type"
      },
      "tag": {
        "type": "string",
        "description": "Filter by tag"
      },
      "blocked": {
        "type": "boolean",
        "description": "If true, only show blocked items"
      }
    }
  }
}
```

**Engine call**: `engine.ListItems(ListFilter{...})`
**Response**: JSON array of items with display_num, type, title, status, priority, assignee, tags, blocked_by

---

#### `get_item`
Get full details of a single item including history.

```json
{
  "name": "get_item",
  "description": "Get detailed information about a specific board item by its number or ID. Returns full item details including description, history, blockers, and parent/child relationships.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number (e.g., '5') or full UUID"
      }
    },
    "required": ["ref"]
  }
}
```

**Engine call**: `engine.GetItem(ref)` + `engine.GetChildren(id)` + `engine.PlansForItem(id)`
**Response**: Full item with children list and linked plans

---

#### `create_item`
Create a new epic, story, or task.

```json
{
  "name": "create_item",
  "description": "Create a new item (epic, story, or task) on the board. Items start in the first column (backlog). Every item must have an assignee — use list_users to see available users.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "type": {
        "type": "string",
        "enum": ["epic", "story", "task"],
        "description": "Item type. Epics contain stories, stories contain tasks."
      },
      "title": {
        "type": "string",
        "description": "Short, descriptive title for the item"
      },
      "description": {
        "type": "string",
        "description": "Detailed description with context, acceptance criteria, and any relevant links"
      },
      "assignee": {
        "type": "string",
        "description": "Name or ID of the user/agent to assign this item to"
      },
      "priority": {
        "type": "string",
        "enum": ["low", "medium", "high", "critical"],
        "description": "Priority level. Defaults to 'medium' if not specified."
      },
      "parent": {
        "type": "string",
        "description": "Parent item reference (display number or ID) — use to nest tasks under stories, stories under epics"
      },
      "tags": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Tags for categorization (e.g., 'bug', 'frontend', 'api')"
      }
    },
    "required": ["type", "title", "assignee"]
  }
}
```

**Engine call**: `engine.CreateItem(type, title, parent, description, priority, assignee, tags)`
**Response**: Created item with display number

---

#### `move_item`
Move an item to a different column (change status).

```json
{
  "name": "move_item",
  "description": "Move an item to a different column on the board. Common workflow: backlog → todo → in-progress → review → done. The item must have an assignee before it can be moved.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number (e.g., '5') or full UUID"
      },
      "status": {
        "type": "string",
        "description": "Target column name (e.g., 'in-progress', 'review', 'done')"
      }
    },
    "required": ["ref", "status"]
  }
}
```

**Engine call**: `engine.MoveItem(ref, status, userID, sessionID)`

---

#### `edit_item`
Edit an item's title, description, or priority.

```json
{
  "name": "edit_item",
  "description": "Edit an existing item's title, description, or priority. Provide only the fields you want to change.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number or UUID"
      },
      "title": {
        "type": "string",
        "description": "New title (leave empty to keep current)"
      },
      "description": {
        "type": "string",
        "description": "New description (leave empty to keep current)"
      },
      "priority": {
        "type": "string",
        "enum": ["low", "medium", "high", "critical"],
        "description": "New priority (leave empty to keep current)"
      }
    },
    "required": ["ref"]
  }
}
```

**Engine call**: `engine.EditItem(ref, title, description, priority, userID, sessionID)`

---

#### `delete_item`
Delete an item from the board.

```json
{
  "name": "delete_item",
  "description": "Delete an item from the board. Cannot delete items that have children — delete children first. This action cannot be undone.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number or UUID"
      }
    },
    "required": ["ref"]
  }
}
```

**Engine call**: `engine.DeleteItem(ref, userID, sessionID)`

---

#### `assign_item`
Assign or reassign an item to a user/agent.

```json
{
  "name": "assign_item",
  "description": "Assign or reassign an item to a specific user or agent. Use list_users to see available assignees.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number or UUID"
      },
      "assignee": {
        "type": "string",
        "description": "Name or ID of the user/agent to assign to"
      }
    },
    "required": ["ref", "assignee"]
  }
}
```

**Engine call**: `engine.AssignItem(ref, assignee, userID, sessionID)`

---

#### `block_item`
Mark an item as blocked by another item.

```json
{
  "name": "block_item",
  "description": "Mark an item as blocked by another item. Blocked items cannot meaningfully progress until the blocker is resolved.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item to block — display number or UUID"
      },
      "blocker": {
        "type": "string",
        "description": "Blocking item — display number or UUID"
      }
    },
    "required": ["ref", "blocker"]
  }
}
```

**Engine call**: `engine.BlockItem(ref, blocker, userID, sessionID)`

---

#### `unblock_item`
Remove a blocker from an item.

```json
{
  "name": "unblock_item",
  "description": "Remove a blocker from an item, allowing it to proceed.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Blocked item — display number or UUID"
      },
      "blocker": {
        "type": "string",
        "description": "Blocker to remove — display number or UUID"
      }
    },
    "required": ["ref", "blocker"]
  }
}
```

**Engine call**: `engine.UnblockItem(ref, blocker, userID, sessionID)`

---

### 3.2 User/Identity Management

#### `list_users`
List all registered users and agents on the board.

```json
{
  "name": "list_users",
  "description": "List all registered users and agents on the board. Shows name, type (human/agent), and provider. Use this to find valid assignee names before creating or assigning items.",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Engine call**: `engine.ListBoard()` → extract `board.Users`
**Response**: Array of `{id, name, type, provider}`

---

#### `add_user`
Register a new user or agent identity.

```json
{
  "name": "add_user",
  "description": "Register a new user or agent identity on the board. Required before they can be assigned items.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Display name for the user/agent"
      },
      "type": {
        "type": "string",
        "enum": ["human", "agent"],
        "description": "Whether this is a human user or an AI agent"
      },
      "provider": {
        "type": "string",
        "description": "Agent provider (e.g., 'claude-code', 'cursor', 'windsurf'). Optional for humans."
      }
    },
    "required": ["name", "type"]
  }
}
```

**Engine call**: `engine.AddUser(name, type, provider)`

---

### 3.3 Plan Management

#### `list_plans`
List all plans on the board.

```json
{
  "name": "list_plans",
  "description": "List all plans on the board. Plans are design documents linked to implementation tasks.",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Engine call**: `engine.ListPlans()`

---

#### `get_plan`
Get full plan details including content and linked items.

```json
{
  "name": "get_plan",
  "description": "Get a plan's full content and linked items by reference number.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Plan reference — display number or UUID"
      }
    },
    "required": ["ref"]
  }
}
```

**Engine call**: `engine.ShowPlan(ref)`

---

#### `create_plan`
Create a new plan.

```json
{
  "name": "create_plan",
  "description": "Create a new plan document on the board. Plans capture design decisions and can be linked to implementation tasks.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "title": {
        "type": "string",
        "description": "Plan title"
      },
      "content": {
        "type": "string",
        "description": "Plan content in markdown format"
      }
    },
    "required": ["title", "content"]
  }
}
```

**Engine call**: `engine.CreatePlan(title, content, "")`

---

#### `link_plan`
Link items to a plan.

```json
{
  "name": "link_plan",
  "description": "Link board items to a plan, establishing traceability from design to implementation.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "plan_ref": {
        "type": "string",
        "description": "Plan reference — display number or UUID"
      },
      "item_refs": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Item references to link to this plan"
      }
    },
    "required": ["plan_ref", "item_refs"]
  }
}
```

**Engine call**: `engine.LinkPlan(planRef, itemRefs)`

---

### 3.4 Board Configuration

#### `list_columns`
List all columns with WIP limits.

```json
{
  "name": "list_columns",
  "description": "List all columns on the board with their WIP limits and current item counts.",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Engine call**: `engine.ListBoard()` → extract columns + `metrics.WIPStatus(board)`

---

#### `add_column`
Add a new column to the board.

```json
{
  "name": "add_column",
  "description": "Add a new status column to the board.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Column name (e.g., 'staging', 'qa')"
      }
    },
    "required": ["name"]
  }
}
```

**Engine call**: `engine.AddColumn(name)`

---

### 3.5 Metrics & Analytics

#### `get_metrics`
Get board analytics — cycle time, lead time, throughput, WIP, dwell times.

```json
{
  "name": "get_metrics",
  "description": "Get comprehensive board analytics including cycle time, lead time, throughput, WIP status, and per-column dwell times. Use this to understand team velocity, identify bottlenecks, and track progress.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "days": {
        "type": "integer",
        "description": "Number of days for velocity history (default: 14)"
      }
    }
  }
}
```

**Engine calls**:
- `engine.ListBoard()` → `metrics.Compute(items, now)`
- `metrics.WIPStatus(board)`
- `metrics.DailyVelocity(items, days, now)`

**Response**:
```json
{
  "total_items": 42,
  "done_items": 18,
  "cycle_time": "2d 4h",
  "lead_time": "5d 12h",
  "throughput": { "this_week": 5, "last_week": 3, "per_week": 4.2 },
  "wip": [
    { "name": "in-progress", "count": 3, "limit": 5, "level": "ok" },
    { "name": "review", "count": 4, "limit": 3, "level": "over" }
  ],
  "dwell": {
    "backlog": { "avg": "3d", "count": 12 },
    "in-progress": { "avg": "1d 6h", "count": 8 }
  },
  "daily_velocity": [
    { "date": "2026-03-01", "count": 2 },
    { "date": "2026-03-02", "count": 0 }
  ]
}
```

---

#### `get_burndown`
Get burndown data for an epic.

```json
{
  "name": "get_burndown",
  "description": "Get burndown chart data for an epic, showing actual vs ideal progress over time.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "epic_ref": {
        "type": "string",
        "description": "Epic reference — display number or UUID"
      }
    },
    "required": ["epic_ref"]
  }
}
```

**Engine call**: `engine.GetItem(ref)` + `engine.GetChildren(id)` → `metrics.EpicBurndown(epic, children, now)`

---

### 3.6 Convenience / Workflow Tools

#### `pick_next_task`
Claim the next unassigned, unblocked task from a column.

```json
{
  "name": "pick_next_task",
  "description": "Pick up the next available task from the board. Finds the highest-priority unassigned, unblocked task in the specified column (default: 'todo'), assigns it to you, and moves it to 'in-progress'. Returns the task details so you can start working immediately.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "from_column": {
        "type": "string",
        "description": "Column to pick from (default: 'todo')"
      }
    }
  }
}
```

**Implementation**: Composite operation:
1. `engine.ListItems({Status: column})` → filter unassigned + unblocked
2. Sort by priority (critical > high > medium > low)
3. `engine.AssignItem(ref, selfID, ...)`
4. `engine.MoveItem(ref, "in-progress", ...)`
5. Return full item details

---

#### `complete_task`
Mark current task as done with optional notes.

```json
{
  "name": "complete_task",
  "description": "Mark a task as done, optionally adding completion notes to the description. Moves the item to the 'done' column.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Item reference — display number or UUID"
      },
      "notes": {
        "type": "string",
        "description": "Optional completion notes appended to description"
      }
    },
    "required": ["ref"]
  }
}
```

**Implementation**: Composite operation:
1. If notes provided: `engine.EditItem(ref, "", existingDesc + "\n\n---\n" + notes, "", ...)`
2. `engine.MoveItem(ref, "done", ...)`

---

#### `board_summary`
Get a high-level board summary — perfect for standup context.

```json
{
  "name": "board_summary",
  "description": "Get a concise summary of the board state: items per column, blockers, recent activity, and key metrics. Ideal for standup context or when you need a quick overview before starting work.",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Implementation**: Composite:
1. `engine.ListBoard()` → count items per column
2. `metrics.Compute(items, now)` → cycle time, throughput
3. `metrics.WIPStatus(board)` → WIP alerts
4. Filter blocked items
5. Return structured summary

---

## 4. MCP Resources (Application-Controlled Data)

Resources expose **read-only data** that hosts can inject into agent context.

### 4.1 Resource Definitions

| URI | Description | Content |
|---|---|---|
| `obeya://board/summary` | Board overview | Name, columns, item counts, WIP status |
| `obeya://board/items` | All items as JSON | Full item list (serialized board state) |
| `obeya://item/{ref}` | Single item detail | Full item with history, children, linked plans |
| `obeya://metrics` | Board analytics | Cycle time, lead time, throughput, dwell |
| `obeya://users` | User/agent list | All registered identities |
| `obeya://plans` | All plans | Plan list with titles and linked item counts |
| `obeya://plan/{ref}` | Plan content | Full plan markdown with linked items |

### 4.2 Resource Templates

```json
{
  "uriTemplate": "obeya://item/{ref}",
  "name": "Board Item",
  "description": "Get full details for a specific board item by its display number",
  "mimeType": "application/json"
}
```

### 4.3 Change Notifications

The server watches for board changes (via `fsnotify` for local, or realtime subscription for cloud) and emits `notifications/resources/updated` when the board state changes. This tells the host to re-fetch resources.

---

## 5. MCP Prompts (User-Controlled Templates)

Prompts are templates that users can invoke to get structured agent behavior.

### 5.1 Prompt Definitions

#### `daily_standup`
```json
{
  "name": "daily_standup",
  "description": "Generate a daily standup report from the board. Shows what was done yesterday, what's in progress today, and any blockers.",
  "arguments": []
}
```

**Returns messages**:
- System: "You are an engineering team member giving a standup update."
- User: "Based on the following board state, give a concise standup update..." + board summary resource

---

#### `sprint_planning`
```json
{
  "name": "sprint_planning",
  "description": "Facilitate sprint planning by analyzing the backlog and suggesting which items to pull into the next sprint based on priority and capacity.",
  "arguments": [
    {
      "name": "capacity",
      "description": "Number of items the team can complete this sprint",
      "required": false
    }
  ]
}
```

---

#### `triage_new_work`
```json
{
  "name": "triage_new_work",
  "description": "Help triage a new piece of work by creating properly structured items with the right type, priority, and parent hierarchy.",
  "arguments": [
    {
      "name": "description",
      "description": "Description of the work to be triaged",
      "required": true
    }
  ]
}
```

---

#### `retrospective`
```json
{
  "name": "retrospective",
  "description": "Generate a retrospective analysis using board metrics: what went well (fast cycle times), what needs improvement (bottlenecks, blocked items), and suggestions.",
  "arguments": []
}
```

---

## 6. Client Configuration

### 6.1 Claude Code (`~/.claude.json` or project `.mcp.json`)

```json
{
  "mcpServers": {
    "obeya": {
      "command": "ob",
      "args": ["mcp", "serve"],
      "env": {}
    }
  }
}
```

### 6.2 Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "obeya": {
      "command": "/usr/local/bin/ob",
      "args": ["mcp", "serve", "--board", "/path/to/project"],
      "env": {}
    }
  }
}
```

### 6.3 Cursor (Settings → MCP → Add Server)

```json
{
  "command": "ob",
  "args": ["mcp", "serve"]
}
```

### 6.4 ChatGPT (Apps & Connectors → MCP)

For ChatGPT, the streamable-HTTP transport is needed (phase 2):

```json
{
  "url": "https://your-obeya-instance.com/mcp",
  "transport": "streamable-http"
}
```

### 6.5 Cloud Board Configuration

```json
{
  "mcpServers": {
    "obeya": {
      "command": "ob",
      "args": ["mcp", "serve", "--cloud"],
      "env": {
        "OBEYA_TOKEN": "ob_tok_..."
      }
    }
  }
}
```

### 6.6 Auto-Configuration

`ob init --agent claude-code` will:
1. Detect if MCP config exists
2. Auto-add the `obeya` MCP server entry to `.mcp.json` (project-level) or `~/.claude.json` (global)
3. Skip if already configured

---

## 7. Implementation Plan

### Phase 1: Core MCP Server (5-7 days)

| Day | Task |
|---|---|
| 1 | Add `github.com/modelcontextprotocol/go-sdk` dependency. Create `internal/mcp/` package. Implement server setup with stdio transport. Register 3 core tools: `list_items`, `create_item`, `move_item`. |
| 2 | Implement remaining item tools: `get_item`, `edit_item`, `delete_item`, `assign_item`, `block_item`, `unblock_item`. |
| 3 | Implement user tools (`list_users`, `add_user`), plan tools (`list_plans`, `get_plan`, `create_plan`, `link_plan`), board tools (`list_columns`, `add_column`). |
| 4 | Implement metrics tool (`get_metrics`, `get_burndown`). Implement convenience tools (`pick_next_task`, `complete_task`, `board_summary`). |
| 5 | Implement resources (`obeya://board/summary`, `obeya://board/items`, `obeya://item/{ref}`, `obeya://metrics`, `obeya://users`, `obeya://plans`, `obeya://plan/{ref}`). Add change notifications via fsnotify. |
| 6 | Implement prompts (`daily_standup`, `sprint_planning`, `triage_new_work`, `retrospective`). Add `ob mcp serve` CLI command with `--board` and `--cloud` flags. |
| 7 | Tests. Auto-configuration in `ob init`. Documentation. |

### Phase 2: Cloud + HTTP Transport (3-4 days)

- Streamable-HTTP transport for remote access
- Cloud board resolution via `CloudStore`
- WebSocket-based resource change notifications for cloud boards
- Multi-board support (board ID as tool parameter)
- Auth token forwarding

### Phase 3: Advanced Features (2-3 days)

- Tool annotations (read-only vs destructive) per MCP spec
- Progress notifications for long-running operations
- Sampling support (server-initiated LLM calls for auto-triage)
- Icon/branding for the MCP server listing

---

## 8. Tool Summary Matrix

| Tool | Engine Method | Mutates | Category |
|---|---|---|---|
| `list_items` | `ListItems()` | No | Items |
| `get_item` | `GetItem()` + `GetChildren()` + `PlansForItem()` | No | Items |
| `create_item` | `CreateItem()` | Yes | Items |
| `move_item` | `MoveItem()` | Yes | Items |
| `edit_item` | `EditItem()` | Yes | Items |
| `delete_item` | `DeleteItem()` | Yes | Items |
| `assign_item` | `AssignItem()` | Yes | Items |
| `block_item` | `BlockItem()` | Yes | Items |
| `unblock_item` | `UnblockItem()` | Yes | Items |
| `list_users` | `ListBoard()` | No | Users |
| `add_user` | `AddUser()` | Yes | Users |
| `list_plans` | `ListPlans()` | No | Plans |
| `get_plan` | `ShowPlan()` | No | Plans |
| `create_plan` | `CreatePlan()` | Yes | Plans |
| `link_plan` | `LinkPlan()` | Yes | Plans |
| `list_columns` | `ListBoard()` + `WIPStatus()` | No | Board |
| `add_column` | `AddColumn()` | Yes | Board |
| `get_metrics` | `Compute()` + `WIPStatus()` + `DailyVelocity()` | No | Metrics |
| `get_burndown` | `EpicBurndown()` | No | Metrics |
| `pick_next_task` | Composite | Yes | Workflow |
| `complete_task` | Composite | Yes | Workflow |
| `board_summary` | Composite | No | Workflow |

**Total: 21 tools, 7 resources, 4 prompts**

---

## 9. Design Decisions

### Why expose ALL operations?

Agents need full autonomy. A "read-only" MCP server would be useful for context but wouldn't let agents actually manage work. By exposing create, move, assign, block, and delete, we enable fully autonomous agent workflows:

1. Agent picks up task → `pick_next_task`
2. Agent works on it (writes code)
3. Agent completes it → `complete_task`
4. Agent picks next → `pick_next_task`

This is the autonomous loop that makes Obeya valuable.

### Why composite "convenience" tools?

`pick_next_task` and `complete_task` could be done by chaining `list_items` → `assign_item` → `move_item`. But:
- Fewer LLM tool calls = faster execution
- Atomic operations = less chance of partial state
- Opinionated workflows = agents do the right thing by default

### Why resources AND tools?

Resources are for **context injection** — hosts can automatically include board state in the agent's context window without the agent asking. Tools are for **actions** — the agent decides when to call them. Both are needed:
- Resource: Host injects `obeya://board/summary` so agent always knows board state
- Tool: Agent calls `move_item` when it finishes work

### Why not just wrap the web API?

The CLI's `engine.Engine` is the single source of truth for business logic — it handles validation, parent resolution, user resolution, history tracking, and transactional persistence. Wrapping the engine directly ensures the MCP server has identical behavior to the CLI. The web API is a separate concern (cloud multi-tenancy, Appwrite persistence).

### Transport: stdio first, HTTP later

stdio covers 95% of use cases (local IDE agents). It's simpler, requires no auth, and works immediately. Streamable-HTTP is only needed for remote/cloud scenarios (ChatGPT, cloud-hosted boards) and can be added in phase 2 without changing the tool definitions.

---

## 10. Testing Strategy

### Unit Tests

```go
func TestListItemsTool(t *testing.T) {
    // Create board with test items
    s := store.NewJSONStore(testBoardPath)
    eng := engine.New(s)
    eng.CreateItem("task", "Test task", "", "desc", "high", "agent-1", nil)

    // Create MCP server
    srv := mcp.NewServer(eng)

    // Call tool
    result, err := srv.CallTool("list_items", map[string]interface{}{
        "status": "backlog",
    })
    assert.NoError(t, err)
    assert.Contains(t, result, "Test task")
}
```

### Integration Tests

1. Start MCP server in subprocess
2. Send JSON-RPC messages over stdin
3. Assert responses on stdout
4. Test full workflow: create → assign → move → complete → metrics

### Golden File Tests

Capture tool response JSON for regression detection:
```go
func TestToolResponses(t *testing.T) {
    // ... setup ...
    result := srv.CallTool("board_summary", nil)
    golden.RequireEqual(t, result)
}
```

---

## 11. Success Criteria

1. `ob mcp serve` starts and responds to `initialize` handshake
2. All 21 tools callable from Claude Code with `.mcp.json` config
3. All 7 resources discoverable and readable
4. All 4 prompts invokable
5. Full workflow test passes: pick → work → complete → metrics
6. Works with both local JSON store and cloud store
7. Zero-config identity detection for Claude Code agents
8. `ob init --agent claude-code` auto-configures MCP
9. Resource change notifications fire on board mutations
10. All existing `./scripts/test.sh` tests continue to pass
