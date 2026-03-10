# Design: `/ob-create` and `/ob-subtask` Skills

Date: 2026-03-09

## Problem

The existing `/ob-create` skill is misleading — it creates subtasks but shares a name with the CLI's `ob create` which creates standalone items. Users need both capabilities as slash commands, with clear naming.

## Design

### `/ob-create` — Create a new standalone board item

**Usage:** `/ob-create <type> "title" [flags]`

**Supported types:** epic, story, task

**Examples:**
```
/ob-create epic "Authentication system"
/ob-create story "User login flow" --priority high
/ob-create task "Fix redirect bug" --description "After OAuth callback" --assign niladribose
```

**Steps:**
1. Parse arguments: `<type>` (epic/story/task), `"title"`, optional flags
2. If missing type or title, ask the user
3. Run `ob create <type> "<title>" [flags]`
4. Display created item with ID
5. Plan linking

### `/ob-subtask` — Create a child item under a parent

**Usage:** `/ob-subtask [parent] [type] "title" [flags]`

- `parent`: optional display number of the parent item
- `type`: optional item type, defaults to `task`

**Examples:**
```
/ob-subtask 15 "Fix the login redirect"            → task under #15
/ob-subtask 15 story "User auth flow"               → story under #15
/ob-subtask "Quick bugfix"                          → task under current in-progress item
/ob-subtask story "Design the API" --priority high  → story under current in-progress item
```

**Parent resolution (when parent omitted):**
1. Run `ob list --status in-progress --format json`
2. If exactly one in-progress item → use it
3. If multiple → list them and ask user to pick
4. If none → ask user for a parent ID

**Steps:**
1. Parse arguments
2. Resolve parent (explicit or via in-progress lookup)
3. Run `ob create <type> "<title>" -p <parent-id> [flags]`
4. Display created item with ID and parent relationship
5. Plan linking

### Plan Linking (shared, runs after both skills)

1. Check if a plan was discussed/created in conversation but not yet imported → import and link
2. Otherwise check `ob plan list` for plans linked to the parent → link new item to same plan
3. If no parent match, check for contextually relevant plans by title
4. If nothing found, say "No plan linked."
