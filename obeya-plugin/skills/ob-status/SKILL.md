---
description: Show tasks assigned to you or the current agent. Use to check what you're working on.
disable-model-invocation: false
user-invocable: true
---

# Show My Status

Display all items assigned to the current user/agent.

## Steps

1. Determine the current user:
   - Check `OB_USER` environment variable
   - If not set, run `ob user list --format json` and identify the likely current user
2. Run `ob list --format json` and filter to items where `assignee` matches the current user
3. Group by status and display:
   - In-progress items first (what you're actively working on)
   - Todo items next (what's queued)
   - Blocked items highlighted with their blockers
4. Show a summary: X in-progress, Y todo, Z blocked
