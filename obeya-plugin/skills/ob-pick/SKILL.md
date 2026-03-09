---
description: Pick an unassigned task from the board and start working on it. Use proactively when starting work or when asked to pick up a task.
disable-model-invocation: false
user-invocable: true
---

# Pick a Task

Claim an unassigned task and move it to in-progress.

## Steps

1. Run `ob list --format json` to get all items
2. Find tasks that are:
   - Status: `backlog` or `todo`
   - Not blocked (empty `blocked_by`)
   - Not assigned (empty `assignee`), OR assigned to the current user
3. Pick the task with the lowest display number (highest priority first if equal)
4. Run `ob move <id> in-progress` to claim it
5. Display the picked task details to the user
6. If no unassigned tasks are available, tell the user

## Environment

Set `OB_USER` to your user ID before running commands, or pass `--as <id>`.
