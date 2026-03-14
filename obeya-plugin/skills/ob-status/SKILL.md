---
description: Show tasks assigned to you or the current agent. Use this BEFORE starting any work to check what you're working on. Must be invoked at the start of any implementation session, before picking new tasks, and whenever the user asks about task status or progress. Use proactively — don't wait to be asked.
disable-model-invocation: false
user-invocable: true
---

# Show My Status

Display all items assigned to the current user/agent.

## Steps

1. **Detect board type** — show whether this project uses a local or linked/shared board:
   - Check if `.obeya-link` exists at the git root → read it for the shared board name
   - Otherwise check for `.obeya/board.json` → local board
   - Display: `Board: local` or `Board: linked → <name> (shared)`
2. Determine the current user:
   - Use the `--as <id>` flag if provided
   - Otherwise, run `ob user list --format json` and identify the likely current user
   - If ambiguous, ask the user: "Which user are you?"
3. Run `ob list --format json` and filter to items where `assignee` matches the current user
4. Group by status and display:
   - In-progress items first (what you're actively working on)
   - Todo items next (what's queued)
   - Blocked items highlighted with their blockers
5. Show a summary: X in-progress, Y todo, Z blocked
