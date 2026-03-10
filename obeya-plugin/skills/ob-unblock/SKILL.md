---
description: Remove a blocker from a task. Use when a blocking dependency has been resolved, when a blocked task can now proceed, or when a blocker was set in error. Counterpart to ob-block.
disable-model-invocation: false
user-invocable: true
---

# Unblock a Task

Remove a blocker from a board item so it can proceed.

## Usage

`/ob-unblock [task-id] [blocker-id]`

## Steps

1. If `$ARGUMENTS` provides both task ID and blocker ID, run `ob unblock <task-id> --by <blocker-id>`
2. If only one argument is provided:
   - Find blocked in-progress tasks: `ob list --status in-progress --format json`
   - Filter to items with non-empty `blocked_by`
   - If exactly one blocked task, use the argument as the blocker ID
   - Otherwise, ask the user to clarify which task and which blocker
3. If no arguments:
   - Run `ob list --format json` and filter to items with non-empty `blocked_by`
   - Display all blocked items with their blockers:
     ```
     Blocked items:
     #12 - Build auth system — blocked by #8 (Design DB schema)
     #15 - Deploy to staging — blocked by #12 (Build auth system)
     ```
   - Ask the user which blocker to remove
4. After unblocking, display the updated item with `ob show <id> --format json`
5. If the unblocked item was in backlog or todo, suggest moving it to in-progress
