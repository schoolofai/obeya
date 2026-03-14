# Obeya Board — Agent Skill

You have access to the `ob` CLI tool for managing a Kanban board. Use it to track, organize, and update work items during your coding sessions.

## Setup

Before using any board commands, establish your identity:

```bash
# Pass identity on each command via flags
--as <user-id>           # Your registered user ID (for audit trail — who ran the command)
--session <session-id>   # Unique identifier for this session
```

Discover your user ID with `ob user list --format json`. Use `--as` and `--session` flags on each command, or set `OB_SESSION` as an environment variable for convenience.

Discover the current board state before starting work:

```bash
ob board config --format json    # Board structure, columns, settings, agent_role
ob list --format json            # All items on the board
ob user list --format json       # Registered users and their roles
```

## Permissions

Check your role by inspecting the `agent_role` field from `ob board config --format json`.

- **admin** — Full access to all operations: create, move, assign, edit, delete any item, manage users, configure board.
- **contributor** — Only modify items assigned to you. Do not reassign, delete, or modify items belonging to others.

Always check your role at the start of a session. If you are a contributor, scope all actions to your own assignments.

## Command Reference

### Board Management

```bash
# Initialize a new board (creates .obeya/board.json)
ob init
ob init --columns "backlog,todo,in-progress,review,done"

# View board configuration
ob board config
ob board config --format json

# Manage columns
ob board columns add staging
ob board columns remove staging
ob board columns reorder "backlog,todo,in-progress,review,done"
```

### User Management

```bash
# Add users
ob user add "Alice" --type human
ob user add "Claude" --type agent --provider claude-code
ob user add "Codex" --type agent --provider codex

# List users
ob user list
ob user list --format json

# Remove a user
ob user remove b3a
```

### Creating Items

Items follow a hierarchy: **Epic > Story > Task**. Nesting is flexible — use `-p` to set a parent.

```bash
# Create an epic (top-level container) — --assign is mandatory
ob create epic "User Authentication System" --assign b3a

# Create a story under an epic
ob create story "Login Flow" -p 1 --assign b3a

# Create a task under a story
ob create task "Add JWT validation" -p 2 --assign b3a

# Create a task with all flags
ob create task "Write unit tests" -p 2 --priority high --assign b3a --tag backend -d "Cover edge cases for token expiry"
```

**Flags:**
- `--assign <user-id>` — **REQUIRED.** Every item must have an assignee. The engine rejects items without one.
- `-p <parent-id>` — Parent item (by display number or hash prefix)
- `--priority <level>` — One of: `low`, `medium`, `high`, `critical`
- `--tag <name>` — Add a freeform tag (repeatable)
- `-d <text>` — Description text

### Moving Items

```bash
ob move 3 in-progress
ob move a3f review
ob move 5 done
```

Check valid column names with `ob board config --format json` before moving.

### Assigning Items

```bash
ob assign 3 --to b3a
ob assign a3f --to c7d
```

### Editing Items

```bash
ob edit 3 --title "Updated title"
ob edit 3 --description "More detailed description"
ob edit 3 -d "Shorthand for description"
ob edit 3 --priority critical
```

### Deleting Items

```bash
ob delete 3
```

Deletion fails if the item has children. Remove or reparent children first.

### Dependencies (Blocking)

```bash
# Mark item 5 as blocked by item 3
ob block 5 --by 3

# Remove the blocker
ob unblock 5 --by 3
```

An item cannot block itself. Blocked items should not be moved to in-progress until unblocked.

### Querying Items

```bash
# List all items (tree view by default)
ob list
ob list --format json

# Flat list (no tree nesting)
ob list --flat
ob list --flat --format json

# Filter by status
ob list --status in-progress --format json

# Filter by assignee (your items)
ob list --assignee b3a --format json

# Filter by type
ob list --type epic --format json
ob list --type task --format json

# Filter by tag
ob list --tag backend --format json

# Show only blocked items
ob list --blocked --format json
```

### Plan Management

```bash
# Create an empty plan
ob plan create --title "Feature Plan"

# Import plan from file and link to items
ob plan import docs/plans/plan.md --link 1,2,3

# Update plan content
ob plan update <plan-id> docs/plans/updated.md
ob plan update <plan-id> --title "New Title"

# Show plan details
ob plan show <plan-id>
ob plan show <plan-id> --format json

# List all plans
ob plan list
ob plan list --format json

# Link/unlink items
ob plan link <plan-id> --to 4,5,6
ob plan unlink <plan-id> --from 3

# Delete a plan
ob plan delete <plan-id>
```

### Showing Item Details

```bash
ob show 3
ob show 3 --format json
ob show a3f --format json
```

Returns full detail including children, blocked-by list, tags, and change history.

**Always use `--format json` when parsing output programmatically.** This applies to all read commands: `list`, `show`, `board config`, `user list`.

## Workflow Conventions

Follow this workflow during every coding session:

### 1. Session Start — Check Assigned Work

```bash
ob user list --format json                    # Find your user ID
ob list --assignee <your-user-id> --format json  # List your items
```

Review your assigned items. Identify what to work on next based on priority and status.

### 2. Pick a Task — Move to In-Progress

```bash
ob move <id> in-progress
```

Only move items assigned to you (especially if you are a contributor).

### 3. Create Subtasks During Work

Break down complex work into subtasks as you discover them. `--assign` is mandatory — read the parent's assignee or use your own ID:

```bash
ob show <parent-id> --format json             # Read parent's assignee
ob create task "Extract helper function" -p <parent-id> --assign <user-id> --tag refactor
```

### 4. Report Blockers

If you encounter a dependency that prevents progress:

```bash
ob block <current-task-id> --by <blocking-item-id>
```

Move on to another available task while blocked.

### 5. Complete Work — Move to Done

```bash
ob move <id> done
```

If all children of a parent story are done, move the parent to done as well:

```bash
ob list --format json   # Check if all sibling tasks are done
ob move <parent-id> done
```

### 6. Rich Task Descriptions

When creating tasks, provide detailed descriptions using `--body-file`:

1. Write the description to a temporary file:

```bash
cat > /tmp/task-desc.md << 'TASKEOF'
## What This Task Involves

Implement JWT token validation middleware for the auth service.

## Files

- Create: `internal/auth/jwt.go`
- Modify: `internal/middleware/auth.go:45-60`
- Test: `internal/auth/jwt_test.go`

## Acceptance Criteria

- Validates token signature and expiry
- Returns 401 for invalid/expired tokens
- Extracts user ID from claims
TASKEOF
```

2. Create the task with the description file:

```bash
ob create task "Add JWT validation" -p 2 --assign b3a --body-file /tmp/task-desc.md --priority high
```

Short one-line descriptions (via `-d`) are acceptable only for trivial tasks. For any task that another agent might pick up, use `--body-file` with full context.

### 7. Plan Management

After creating a task breakdown from an implementation plan:

1. Import the plan document and link it to all related items:

```bash
ob plan import docs/plans/your-plan.md --link 1,2,3,4,5
```

2. When creating additional items related to an existing plan:

```bash
ob plan link <plan-id> --to <new-item-id>
```

3. Query plans:

```bash
ob plan list --format json
ob plan show <plan-id> --format json
```

Plans provide full context for anyone (human or agent) picking up a task. Always import your implementation plan after creating the task breakdown.

## ID Resolution

Every item has two identifiers:

- **Display number** — Auto-incrementing integer (e.g., `1`, `2`, `3`). Short and human-friendly.
- **Hash prefix** — Canonical short hash (e.g., `a3f8b2`). Globally unique.

All commands accept either form:

```bash
ob show 3       # By display number
ob show a3f     # By hash prefix (unique prefix match)
ob move 3 done  # Display number
ob move a3f done # Hash prefix
```

Use display numbers for quick interactions. Use hash prefixes when precision matters or when scripting.

## Error Handling

All commands fail fast with clear error messages. No silent failures, no fallback behavior.

Common errors and what they mean:

| Error Message | Meaning | Resolution |
|---|---|---|
| `no board found — run 'ob init' first` | No `.obeya/board.json` in the current directory tree | Run `ob init` to create a board |
| `invalid status "<name>"` | The target column does not exist | Run `ob board config` to see valid column names |
| `cannot delete item: it has children` | Item has child items that must be removed first | Delete or reparent children, then retry |
| `an item cannot block itself` | You passed the same ID for both item and blocker | Use a different blocker ID |
| `item not found: <id>` | No item matches the given display number or hash prefix | Verify the ID with `ob list --format json` |
| `user not found: <id>` | No user matches the given ID | Check users with `ob user list --format json` |
| `version conflict` | Another process modified the board concurrently | Retry the command — the board will reload |
| `permission denied` | Contributor trying to modify an unassigned item | Only modify items assigned to you |
| `WIP limit reached for column "<name>"` | Column has a work-in-progress limit | Move an item out of the column first |
