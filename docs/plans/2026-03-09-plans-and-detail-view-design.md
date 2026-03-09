# Plans & Full-Screen Detail View — Design

## Overview

Add plans as linkable documents to Obeya items, enrich task descriptions via `--body-file`, and replace the TUI detail overlay with a full-screen tabbed view.

## Data Model

### Plan Struct

```go
type Plan struct {
    ID          string    `json:"id"`
    DisplayNum  int       `json:"display_num"`
    Title       string    `json:"title"`
    Content     string    `json:"content"`
    SourceFile  string    `json:"source_file,omitempty"`
    LinkedItems []string  `json:"linked_items"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Board Changes

```go
type Board struct {
    // ... existing fields ...
    Plans map[string]*Plan `json:"plans"`
}
```

- Plans share the board's `NextDisplay` counter — no ID collisions with items.
- Linkage lives on the Plan side (`LinkedItems`). Items have no new field.
- Reverse lookup (item → plans) is computed by scanning `Board.Plans`.
- Backwards compatible: existing boards get an empty `plans` map on first load.

## CLI Commands

### `ob plan` Subcommand Group

| Command | Description |
|---------|-------------|
| `ob plan create --title "Title"` | Create an empty plan |
| `ob plan import <file> --link 1,2,3` | Import markdown file, link to items |
| `ob plan update <id> <file>` | Replace plan content from file |
| `ob plan update <id> --title "New"` | Update plan title only |
| `ob plan show <id>` | Show plan metadata + linked items |
| `ob plan show <id> --format json` | JSON output |
| `ob plan list` | List all plans |
| `ob plan list --format json` | JSON output |
| `ob plan link <id> --to 4,5,6` | Link additional items to plan |
| `ob plan unlink <id> --from 3` | Remove item link from plan |
| `ob plan delete <id>` | Delete plan (items unaffected) |

**Behaviors:**
- `import` reads file, uses first `# Heading` as title (or `--title` override)
- `link/unlink` are additive/subtractive; linking an already-linked item is a no-op
- All commands accept display number or hash prefix for plan ID

### `--body-file` Flag

Added to `ob create` and `ob edit`:

```bash
ob create task "Add JWT" -p 2 --body-file /tmp/desc.md
ob edit 3 --body-file /tmp/updated.md
```

Reads file content as description. Mutually exclusive with `-d` — error if both provided.

## TUI — Full-Screen Tabbed Detail View

Enter on any item opens a full-screen view replacing the board (not an overlay).

### Layout

```
┌─ #3 Design Form ──────────────────────────────────────────────┐
│ [Fields]  [Plan]  [History]                                    │
│                                                                │
│  Type:       task                                              │
│  Status:     in-progress                                       │
│  Priority:   ●● medium                                        │
│  Assignee:   @claude                                           │
│  Parent:     #2 Login Flow                                     │
│  Tags:       frontend                                          │
│  Blocked:    —                                                 │
│  Description:                                                  │
│    Implement the login form with email/password fields...      │
│                                                                │
│  Children:                                                     │
│    #4   task     todo         Implement JWT                    │
│                                                                │
│ Tab/Shift-Tab:switch  m:move  a:assign  p:priority  Esc:back   │
└────────────────────────────────────────────────────────────────┘
```

### Tabs

| Tab | Content |
|-----|---------|
| Fields | Item metadata, description (line-wrapped), children list |
| Plan | Linked plan markdown content, scrollable. Multiple plans stacked with headers. "No plan linked." if none. |
| History | Full changelog, scrollable |

### Key Bindings

| Key | Action |
|-----|--------|
| Tab / Shift-Tab | Switch between tabs |
| j/k | Scroll content within active tab |
| m | Move item (opens picker) |
| a | Assign item (opens picker) |
| p | Cycle priority |
| Esc | Return to board view |

## Skill Updates

### Verbose Descriptions

Instruct agents to write rich descriptions using `--body-file`:
- What the task involves (2-3 sentences)
- Files to create or modify (exact paths)
- Acceptance criteria
- Relevant code snippets or pseudocode
- Dependencies or prerequisites

### Plan Import Workflow

After creating a task breakdown, agents import the plan:

```bash
ob plan import docs/plans/your-plan.md --link 1,2,3,4,5
```

### Plugin

New `/ob:plan` slash command for plan operations.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Plan storage | In `board.json` `plans` map | Self-contained, agents can read via CLI |
| Plan linkage direction | Plan → items (not item → plan) | Single source of truth, no field on Item |
| Plan types | Type-agnostic | Any item can be linked — epic, story, task |
| Display numbers | Shared counter with items | Uniform `#N` references, no collisions |
| Detail view | Full-screen tabbed | Clean separation of fields/plan/history |
| Description input | `--body-file` flag | Handles multi-line markdown without shell escaping |
| Content scrolling | j/k within active tab | Consistent with board navigation |
