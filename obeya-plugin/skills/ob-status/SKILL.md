---
description: Show tasks assigned to you or the current agent. Use this BEFORE starting any work to check what you're working on. Must be invoked at the start of any implementation session, before picking new tasks, and whenever the user asks about task status or progress. Use proactively — don't wait to be asked.
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
