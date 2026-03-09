---
description: Report a blocker on the current task. Use when work cannot proceed due to a dependency.
disable-model-invocation: false
user-invocable: true
---

# Report a Blocker

Mark the current task as blocked by another item.

## Steps

1. If `$ARGUMENTS` is provided, parse it as `<blocker-id>`:
   - Find the current in-progress task (from `ob list --status in-progress --format json`)
   - Run `ob block <current-task-id> --by $ARGUMENTS`
2. If no arguments:
   - Show current in-progress tasks
   - Run `ob list --format json` and display available items that could be blockers
   - Ask the user which item is blocking
3. After blocking, suggest moving to another available task
