# Design: Enhanced ob init — Detailed Task Descriptions & Mandatory Board Usage

**Date:** 2026-03-10
**Status:** Approved

## Problem

1. Tasks created from plans lack detailed descriptions — agents creating tasks only set titles, leaving tasks without enough context for another agent to pick up and complete.
2. The CLAUDE.md template injected by `ob init` doesn't instruct agents to use the board for ALL work, so ad-hoc tasks go untracked.

## Solution

Two changes:

### 1. `ob create` requires `--description`

**File:** `cmd/create.go`

Add validation after body-file resolution (line ~41) that fails if `createDesc` is empty:

```go
// After body-file resolution, before parseTags
if strings.TrimSpace(createDesc) == "" {
    return fmt.Errorf("--description (-d) or --body-file is required. Provide enough context so any agent can complete this task")
}
```

This is a fast-fail — no fallback, no default description.

### 2. Enhanced CLAUDE.md template

**File:** `cmd/init.go` — replace `obeyaClaudeMDContent()` body

Bump version marker to `v3` to trigger re-injection on existing projects.

New template content:

```markdown
<!-- obeya:start --> v3

## Task Tracking — Obeya

This project uses Obeya (`ob`) for task tracking. The board is the single source of truth for all work.

### Mandatory: Track ALL work
Every piece of work MUST have a task on the board. Before starting any work:
1. Run `/ob:status` to check assigned tasks
2. If no task exists for this work, create one with `ob create task "Title" --description "..."`
3. Run `/ob:pick` to claim a task when implementing from the backlog
4. Run `/ob:done` when work is complete

### Creating tasks from plans
When breaking down a plan into tasks, create a full hierarchy with detailed descriptions:
- **Epics**: High-level goals. Description includes the objective, success criteria, and scope boundaries.
- **Stories**: Deliverable units. Description includes what needs to be built, why it matters, and acceptance criteria.
- **Tasks**: Atomic work items. Description includes what to do, how to verify it's done, and dependencies on other tasks.

Task descriptions must be self-contained — an agent picking one up should have everything needed to start work. Include key context inline and reference files for larger context (e.g., "See docs/plans/auth-design.md section 3 for protocol details" or "See src/auth/oauth.go for existing implementation").

### Task lifecycle
- Starting work: `ob move <id> in-progress`
- Update progress: `ob edit <id> --description "..."` — append notes as you work (discoveries, approach changes, blockers hit)
- Blocked: `ob block <id> --by <blocker-id>`
- Done: `ob move <id> done`

### Plan management
When a plan document is created, discussed, or approved:
1. Import it: `ob plan import <path-to-plan.md>`
2. Break it down into epics, stories, and tasks with full descriptions
3. Link tasks to plan: `ob plan link <plan-id> --to <task-ids>`
4. When creating subtasks under a plan-linked parent, link them too: `ob plan link <plan-id> --to <new-task-id>`

Use `ob list --format json` for full board state.

<!-- obeya:end -->
```

### 3. Update current project CLAUDE.md

Run `ob init` (or manually replace the obeya section) on the current codebase so the new template takes effect immediately.

## Files Changed

| File | Change |
|------|--------|
| `cmd/create.go` | Add description-required validation |
| `cmd/init.go` | New CLAUDE.md template, bump version to v3 |
| `CLAUDE.md` (current project) | Re-inject with new template |

## Not Changed

- No new commands or flags
- No changes to domain types or engine
- No changes to plan import logic
- Task hierarchy parsing stays manual (agent-driven)
