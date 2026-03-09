---
description: Quickly create a subtask under the current work item. Use when breaking down work during implementation.
disable-model-invocation: false
user-invocable: true
---

# Quick Create Subtask

Create a new task under the currently active work item.

## Steps

1. The title is provided via `$ARGUMENTS`
2. Find the current in-progress task from `ob list --status in-progress --format json`
3. Run `ob create task "$ARGUMENTS" -p <parent-id>`
4. Display the created task with its ID
5. If no arguments provided, ask the user for a task title
6. If no in-progress parent found, ask the user which item to create the task under
