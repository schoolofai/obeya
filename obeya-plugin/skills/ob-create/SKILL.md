---
description: Create a new board item (epic, story, or task). Use when adding new work items to the board, when breaking down plans into tasks, when a skill creates TodoWrite items that need board equivalents, or when starting work that has no existing board task. Every piece of work must have a board task — use this to create one.
disable-model-invocation: false
user-invocable: true
---

# Create Board Item

Create a new standalone item on the Obeya board.

## Usage

`/ob-create <type> "<title>" [flags]`

- `type`: epic, story, or task
- `title`: the item title
- **Required flag**: `--assign <user>` — every item must have an assignee. The engine rejects items without one.
- Optional flags: `--priority <low|medium|high|critical>`, `--description "<text>"`, `--tag "<tags>"`

## Steps

1. Parse `$ARGUMENTS` for type, title, and flags
2. If type is missing, ask the user: "What type? (epic / story / task)"
3. If title is missing, ask the user for a title
4. **Determine the assignee** — `--assign` is mandatory:
   - If `--assign` was provided in arguments, use it
   - Otherwise, determine the current user via `--as` flag or `ob user list --format json`
   - If still ambiguous, ask the user: "Who should this be assigned to?"
5. Run `ob create <type> "<title>" --assign <user> [flags]`
6. Display the created item with its ID and display number

## Shared Board Awareness

When the project is linked to a shared board (`.obeya-link` exists at git root), items are created on the shared board and their `project` field is automatically set to the current project name (derived from the git remote `org/repo` or the directory name as fallback). All `ob` commands resolve through the link transparently — no extra flags needed.

## Description Quality (REQUIRED)

Task descriptions must be self-contained so an agent can pick the task up cold. Before creating, ensure the description includes:

**For Epics:**
- The objective and why it matters
- Success criteria (how to know it's done)
- Scope boundaries with explicit IN and OUT sections — without an OUT section, agents drift into adjacent work:
  - IN: what this epic delivers (e.g., auth endpoints, JWT tokens, middleware)
  - OUT: what it explicitly excludes (e.g., OAuth2, RBAC, frontend changes)

**For Stories:**
- What needs to be built and why
- Acceptance criteria (testable conditions)
- Key dependencies on other stories/tasks

**For Tasks** — use these named headers so agents can scan them when picking up work:

```
## What to do
<specific implementation steps>

## Key files
<exact file paths — e.g., internal/auth/token.go, NOT just internal/auth/>

## How to verify
<executable shell command, not prose>

## Dependencies
<task IDs or titles this depends on>
```

Verification commands matter most. An agent executing this task will run your command verbatim:
- Good: `go test ./internal/auth/ -run TestTokenGenerate -v`
- Bad: "Verify that token generation works and tests pass"

If the caller provides a vague description, expand it with context from the conversation before creating. A task an agent cannot start without asking questions is incomplete.

## Plan Linking (REQUIRED — do this after every create)

You MUST check for and link plans after creating the item. Do NOT skip this section.

### Check for unimported plans first

If a plan document was written, discussed, or approved in this conversation (from plan mode, a design doc, an implementation plan, or any markdown plan file), and it has NOT yet been imported into `ob plan`:

1. Save the plan content to a temporary file if it doesn't already exist on disk
2. Run `ob plan import <path-to-plan-file> --link <new-item-id>` to import and link in one step
3. Tell the user: "Imported and linked plan: <plan-title>"

### Link to existing plans

If no new plan needs importing:

1. Run `ob plan list --format json` to get all plans with their linked item IDs
2. Check if any plan is contextually relevant (by title match against the new item)
3. If a matching plan is found:
   - Run `ob plan link <plan-id> --to <new-item-id>`
   - Tell the user: "Linked to plan: <plan-title>"
4. If no plan found at all, say "No plan linked."

