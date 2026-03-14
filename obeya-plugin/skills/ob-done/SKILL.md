---
description: Mark the current in-progress task as done. Use when work is completed on a task, after tests pass, after a subagent reports DONE, or whenever a workflow step completes. If any skill or workflow marks work complete via TodoWrite or TaskCreate, this skill MUST also be invoked to keep the Obeya board in sync.
disable-model-invocation: false
user-invocable: true
---

# Complete Current Task

Mark the current in-progress task as done and show what's next.

## Steps

1. Run `ob list --status in-progress --format json` to find in-progress items
2. Filter to items assigned to the current user (determine identity via `--as` flag or `ob user list --format json`)
3. If exactly one in-progress task: run `ob move <id> done`
4. If multiple in-progress tasks: show them and ask the user which to complete
5. If `$ARGUMENTS` is provided, use it as the task ID: run `ob move $ARGUMENTS done`
6. After completing, run `ob list --status todo --format json` to show the next available tasks
7. If all children of a parent are now done, suggest moving the parent to done too
