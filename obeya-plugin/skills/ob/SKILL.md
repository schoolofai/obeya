---
description: Show the Obeya Kanban board overview. Use when user asks about tasks, board status, or project progress.
disable-model-invocation: false
user-invocable: true
---

# Obeya Board Overview

Show the current state of the Kanban board.

## Steps

1. Run `ob list --format json` to get all items
2. Run `ob board config --format json` to get column names and board settings
3. Display the results as a formatted board:
   - Group items by column/status
   - Show each item with: display number, type (epic/story/task), title, priority
   - Mark blocked items with [BLOCKED]
   - Show item counts per column
4. If the board is not initialized, tell the user to run `ob init`

## Prerequisite

The `ob` CLI binary must be installed and available in PATH. Install via:
```bash
brew tap schoolofai/tap && brew install obeya
```
