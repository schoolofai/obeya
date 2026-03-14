---
description: Pick an unassigned task from the board and start working on it. Use proactively when starting work, when asked to pick up a task, or when the current task is done and more work remains. This skill should be used whenever an agent needs to begin new work — including after plan creation, after completing a previous task, or when the user says to start implementing.
disable-model-invocation: false
user-invocable: true
---

# Pick a Task

Claim an unassigned task and move it to in-progress.

## Steps

1. **Detect board type** — check if `.obeya-link` exists at git root:
   - If linked: note the shared board name, and when displaying tasks show their `project` field so the user knows which project each task belongs to
   - If local: proceed normally
2. Run `ob list --format json` to get all items
   - On a shared board, `ob list` returns items from **all linked projects** — use the `project` field on each item to tell them apart
3. Find tasks that are: status `backlog` or `todo`, not blocked, not assigned (or assigned to current user)
   - On a shared board with multiple projects, prefer tasks whose `project` field matches the current project unless the user requests otherwise
   - The current project name is derived from the git remote (`org/repo` format) or the directory name as fallback — check items' `project` field against this
4. Pick the highest-priority eligible task
5. **Assign-then-move** — ensure the item has an assignee before moving:
   - If the item is unassigned, run `ob assign <id> --to <self>` first (determine your identity via `--as` flag or `ob user list --format json`)
   - Then run `ob move <id> in-progress`
6. Display the picked task details to the user (include `project` field on shared boards)
7. If no eligible tasks, tell the user

## Board Awareness (REQUIRED — scan the full board)

Before diving into your task's details, summarize the board state for the user. This prevents tunnel vision on your own work:

1. Note any **blocking relationships** — which tasks are blocked and by what (e.g., "#4 is blocked by #3, so #4 can't start until #3 completes")
2. Note which tasks are **eligible to pick** vs blocked vs already in-progress
3. If completing your current task would **unblock** other tasks, call that out — it helps the user prioritize

## Parent Context (REQUIRED after pick)

If the picked task has a parent story or epic, retrieve it so you understand the broader goal:

1. Run `ob show <parent-id> --format json`
2. Display the parent's title, description summary, and acceptance criteria
3. This gives the agent context for why this task exists — without it, agents implement tasks in isolation without understanding the story-level goal they contribute to

## Plan Context (REQUIRED after pick)

Surface any plan linked to this task so the agent has design context:

1. If a plan was written in this conversation but not yet imported, import it first: `ob plan import <path> --link <task-id>`
2. Run `ob plan list --format json`
3. Check if the task's ID or parent's ID appears in any plan's `linked_items`
4. If linked:
   - Run `ob plan show <plan-id> --format json`
   - Match the task title against plan headings and show the **most relevant section**
5. If not linked, say "No plan linked to this task" and explain what was checked (this reasoning helps the user understand)

## Identity

Determine your identity via the `--as <id>` flag or by running `ob user list --format json`. The `--as` flag is used for audit trail (who ran the command), not for ownership — ownership is set via `ob assign`.
