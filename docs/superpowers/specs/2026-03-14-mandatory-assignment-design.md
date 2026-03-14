# Mandatory Assignment & Identity Simplification

**Date:** 2026-03-14
**Status:** Draft

## Problem

Tasks on the Obeya board have no assignee. The `ob-pick` skill moves tasks to in-progress but never assigns them. The identity system has overlapping mechanisms (`OB_USER` env var, `--as` flag, `--assign` flag, OS username fallback) that create confusion and silent misattribution. In an agent-first tool, every board item must have an owner from creation.

## Design Principles

1. **Every item has an owner from birth** — no orphaned work on the board
2. **Single source of truth for ownership** — the `assignee` field, set via `--assign`
3. **No command operates on an unassigned item** — hard fail with instructions
4. **Zero env vars** — drop `OB_USER` entirely
5. **Two concerns, two flags** — `--assign` for ownership, `--as` for audit actor

## Identity Model

### Ownership (who owns this work)

| Mechanism | When | Required? | Default |
|-----------|------|-----------|---------|
| `--assign <user>` on `ob create` | Item creation | **Mandatory** | None — hard fail |
| `ob assign <id> --to <user>` | Reassignment | Explicit command | N/A |

### Actor (who ran this command)

| Mechanism | When | Required? | Default |
|-----------|------|-----------|---------|
| `--as <user>` on any command | Any `ob` operation | Optional | OS username |

### Dropped

| Mechanism | Reason |
|-----------|--------|
| `OB_USER` env var | Redundant. Caused friction with `--assign`. Single source of truth is the assignee field. |
| Auto-assign on move | Unnecessary. Assignment is explicit at creation. |

## CLI Changes

### 1. Mandatory `--assign` on all `ob create`

**File:** `cmd/create.go`

All item types (epic, story, task) require `--assign`. If missing, hard fail:

```
Error: --assign is required. Every item must have an owner.

Examples:
  ob create task "Fix bug" --assign claude
  ob create epic "Auth system" --assign niladri

If you are an agent, assign yourself:
  Claude agent:  --assign claude
  Codex agent:   --assign codex
  Cursor agent:  --assign cursor

Run 'ob user list' to see registered users.
```

The `--assign` value must resolve to a registered board user via `board.ResolveUserID()`. If the user does not exist, hard fail with an error listing registered users. This matches the existing behavior of `ob assign` and prevents storing dangling references.

**Pseudocode (cmd layer — empty check):**

```go
func runCreate(cmd *cobra.Command, args []string) {
    if createAssign == "" {
        fmt.Fprintf(os.Stderr, mandatoryAssignError)
        os.Exit(1)
    }
    // ... pass createAssign to engine.CreateItem
}
```

**Pseudocode (engine layer — user resolution inside transaction):**

```go
func (e *Engine) CreateItem(itemType, title, desc, assignee string, ...) (*domain.Item, error) {
    var created *domain.Item
    err := e.store.Transaction(func(board *domain.Board) error {
        // Resolve assignee to registered user — fails if not found
        assigneeID, err := board.ResolveUserID(assignee)
        if err != nil {
            return fmt.Errorf("unknown assignee %q: %w\nRun 'ob user list' to see registered users", assignee, err)
        }
        item := buildItem(board, itemType, title, desc, assigneeID, ...)
        board.Items[item.ID] = item
        created = item
        return nil
    })
    return created, err
}
```

### 2. Assignee guard on all state-change commands

**Location:** Centralized in `engine.go`, inside each method's `Transaction` callback.

Before any state-change operation, check that the target item has an assignee. If not, hard fail:

```
Error: item #5 has no assignee. Assign it first:

  ob assign #5 --to <user>

Examples:
  ob assign #5 --to claude
  ob assign #5 --to niladri

Run 'ob user list' to see registered users.
```

The assignee check must happen **inside** the existing `store.Transaction()` callback of each guarded method — not as a separate pre-check — to avoid TOCTOU races and double board reads.

**Pseudocode (inside a transaction callback):**

```go
// Example: inside MoveItem's Transaction callback
func (e *Engine) MoveItem(ref, status, userID, sessionID string) error {
    return e.store.Transaction(func(board *domain.Board) error {
        id, err := board.ResolveID(ref)
        if err != nil {
            return err
        }
        item := board.Items[id]

        // Assignee guard — inline, inside the transaction
        if item.Assignee == "" {
            return fmt.Errorf("item #%d has no assignee. Run: ob assign %d --to <user>",
                item.DisplayNum, item.DisplayNum)
        }

        // ... existing move logic
    })
}
```

A shared helper `checkAssignee(item *domain.Item) error` can be extracted for reuse across methods, but it must be called from within the transaction, not before it.

Commands guarded:

| Command | Guard |
|---------|-------|
| `ob move <id> <status>` | Assignee must be set |
| `ob edit <id>` | Assignee must be set. Note: `ob edit` does NOT gain an `--assign` flag — use `ob assign` for reassignment. |
| `ob block <id>` | Assignee must be set |
| `ob unblock <id>` | Assignee must be set |
| `ob delete <id>` | Assignee must be set (prevents silent cleanup of orphaned items — assign first, then delete intentionally) |

Note: there is no `cmd/done.go` — "done" is achieved via `ob move <id> done`, which is covered by the `MoveItem` guard above.

Commands NOT guarded (read-only or assignment itself):

| Command | Reason |
|---------|--------|
| `ob list` | Read-only |
| `ob show <id>` | Read-only |
| `ob assign <id> --to <user>` | This IS the fix for the guard |
| `ob user list/add/remove` | User management, not item ops |

### 3. Drop `OB_USER` env var

**File:** `cmd/helpers.go`

Remove `OB_USER` from `getUserID()`. The function becomes:

```go
func getUserID() string {
    if flagAs != "" {
        return flagAs
    }
    u, err := user.Current()
    if err != nil {
        return "unknown"
    }
    return u.Username
}
```

This function is now only used for the `--as` actor in history records, not for ownership.

### 4. `OB_USER` deprecation warning

If `OB_USER` is set in the environment, print a one-time warning to stderr:

```
Warning: OB_USER is deprecated and ignored. Use --assign for ownership, --as for audit.
```

**Pseudocode:**

```go
func init() {
    if os.Getenv("OB_USER") != "" {
        fmt.Fprintln(os.Stderr, "Warning: OB_USER is deprecated and ignored. "+
            "Use --assign for ownership, --as for audit.")
    }
}
```

## TUI Changes

### 5. Render `@unassigned` for items without assignee

**File:** `internal/tui/card.go` (lines 43-53, metadata section)

Current behavior: if `item.Assignee == ""`, nothing is rendered.

New behavior: render `@unassigned` in red, faint style.

**Pseudocode:**

```go
// In renderCard(), metadata section:
if item.Assignee != "" {
    name := resolveUserName(a.board, item.Assignee)
    meta = append(meta, assigneeStyle.Render("@"+name))
} else {
    meta = append(meta, unassignedStyle.Render("@unassigned"))
}
```

**New style in `styles.go`:**

```go
unassignedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
// Color 1 = red, faint for subtlety
```

**Visual result:**

```
┌──────────────────────┐    ┌──────────────────────┐
| #5 Fix login redirect|    | #8 Add OAuth flow    |
| task @@              |    | story @@@            |
| @Claude              |    | @unassigned          |
└──────────────────────┘    └──────────────────────┘
```

## Skill Changes

### 6. Update all 6 skills

Each skill needs updates to work with mandatory assignment and dropped `OB_USER`.

#### `ob-create` skill

- Remove reference to `--assign` being optional
- Instruct: always include `--assign` — if you are an agent, assign yourself
- Remove `OB_USER` / env var references

#### `ob-subtask` skill

- Same as `ob-create` — `--assign` is mandatory on the underlying `ob create` call
- Instruct: read the parent item's assignee (via `ob show <parent> --format json`) and explicitly pass it as `--assign <parent-assignee>` in the generated `ob create` command, unless the user specifies a different assignee
- There is no automatic CLI-level inheritance — the skill must always pass `--assign` explicitly

#### `ob-pick` skill

- Current: finds unassigned tasks, runs `ob move <id> in-progress`
- New flow:
  1. Find tasks (any assignment status, filtered by priority/status)
  2. If unassigned: run `ob assign <id> --to <self>` first
  3. Then run `ob move <id> in-progress`
- Remove `OB_USER` reference from environment section
- Add `--as <agent-name>` for audit trail

#### `ob-status` skill

- Remove `OB_USER` detection logic
- Use `--as` flag or ask the user who they are
- Filter by assignee field (unchanged query, different identity source)

#### `ob-done` skill

- Remove `OB_USER` filtering reference
- Use `--as` for history, rely on assignee field for "my tasks" filtering

#### `ob-show` skill

- Display assignee prominently (already does this)
- Show `unassigned` in output when assignee is empty

## Error Message Catalog

All error messages follow the same pattern: state the problem, show the fix, give agent-specific examples.

### Missing `--assign` on create

```
Error: --assign is required. Every item must have an owner.

Examples:
  ob create task "Fix bug" --assign claude
  ob create epic "Auth system" --assign niladri

If you are an agent, assign yourself:
  Claude agent:  --assign claude
  Codex agent:   --assign codex
  Cursor agent:  --assign cursor

Run 'ob user list' to see registered users.
```

### Unassigned item on state-change

```
Error: item #5 has no assignee. Assign it first:

  ob assign #5 --to <user>

Examples:
  ob assign #5 --to claude
  ob assign #5 --to niladri

Run 'ob user list' to see registered users.
```

### Deprecated `OB_USER`

```
Warning: OB_USER is deprecated and ignored. Use --assign for ownership, --as for audit.
```

## Migration

Existing boards will have items without assignees. These are handled by:

1. **TUI:** renders `@unassigned` in red — makes gaps visible
2. **CLI guards:** any state-change on unassigned items fails with instructions to assign first
3. **No automatic migration** — items are assigned on-demand as they're touched

## Testing

### CLI tests

- `ob create task "x"` without `--assign` → exit code 1, error message
- `ob create task "x" --assign claude` → success, assignee set and resolved via `ResolveUserID`
- `ob create task "x" --assign nonexistent` → exit code 1, error: user not found
- `ob move #5 in-progress` on unassigned item → exit code 1, error message
- `ob move #5 in-progress` on assigned item → success
- `ob edit #5 --description "x"` on unassigned item → exit code 1, error message
- `ob block #5 --by #3` on unassigned item → exit code 1, error message
- `ob unblock #5` on unassigned item → exit code 1, error message
- `ob delete #5` on unassigned item → exit code 1, error message
- `ob assign #5 --to claude` on unassigned item → success (this IS the fix)
- `OB_USER` set → deprecation warning on stderr

### TUI tests (teatest)

- Card with assignee renders `@<name>` in cyan
- Card without assignee renders `@unassigned` in red
- Golden file updates for new rendering

### Skill tests

- Manual verification that each skill's instructions produce correct `ob` commands

## Files Changed

| File | Change |
|------|--------|
| `cmd/create.go` | Mandatory `--assign` validation + user resolution via `board.ResolveUserID()` |
| `cmd/helpers.go` | Drop `OB_USER` from `getUserID()`, add deprecation warning |
| `cmd/root.go` | Remove `OB_USER` reference from `--as` flag help text |
| `internal/engine/engine.go` | Inline assignee guard in `MoveItem`, `EditItem`, `BlockItem`, `UnblockItem`, `DeleteItem` transaction callbacks. Add `ResolveUserID` call in `CreateItem`. |
| `internal/tui/card.go` | `@unassigned` rendering |
| `internal/tui/styles.go` | `unassignedStyle` definition |
| `internal/tui/card_test.go` | Tests for unassigned rendering |
| `skill/obeya.md` | Remove all `OB_USER` / `$OB_USER` references |
| `obeya-plugin/skills/ob-create/SKILL.md` | Mandatory `--assign` instructions, remove `OB_USER` |
| `obeya-plugin/skills/ob-subtask/SKILL.md` | Mandatory `--assign` with explicit parent-assignee lookup |
| `obeya-plugin/skills/ob-pick/SKILL.md` | Assign-then-move flow, remove `OB_USER` |
| `obeya-plugin/skills/ob-status/SKILL.md` | Drop `OB_USER` detection, use `--as` for identity |
| `obeya-plugin/skills/ob-done/SKILL.md` | Drop `OB_USER` reference |
| `obeya-plugin/skills/ob-show/SKILL.md` | Display `unassigned` label |
