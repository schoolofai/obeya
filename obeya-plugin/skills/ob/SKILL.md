---
description: Show the Obeya Kanban board overview. Use when user asks about tasks, board status, or project progress.
disable-model-invocation: false
user-invocable: true
---

# Obeya Board Overview

Show the current state of the Kanban board.

## Steps

1. **Detect board type** — determine whether this project uses a local or linked/shared board:
   - Check if `.obeya-link` exists at the git root. If yes, read it to get the shared board name → **Linked board** (shared board: `<name>`)
   - Otherwise check if `.obeya/board.json` exists → **Local board**
   - If neither, tell the user to run `ob init` (local) or `ob link <name>` (shared)
2. Run `ob list --format json` to get all items
3. Run `ob board config --format json` to get column names and board settings
4. Display the results as a formatted board:
   - **Header**: show board type — e.g., `Board: local` or `Board: linked → teamboard (shared)`
   - If linked, show linked projects by running `ob boards --format json` or reading the board's Projects field
   - Group items by column/status
   - Show each item with: display number, type (epic/story/task), title, priority
   - Mark blocked items with [BLOCKED]
   - Show item counts per column
5. If the board is not initialized, tell the user to run `ob init`

## Shared Board Commands Reference

When users ask about shared/global boards, guide them with these commands:

| Command | Description |
|---|---|
| `ob init --shared <name>` | Create a new shared board at `~/.obeya/boards/<name>/` |
| `ob link <board-name>` | Link this project to an existing shared board |
| `ob link <board-name> --migrate` | Link and migrate existing local tasks to the shared board |
| `ob unlink` | Disconnect this project from its shared board |
| `ob boards` | List all shared boards with their linked projects |

### How linking works

- A shared board lives at `~/.obeya/boards/<name>/` (user-level, not inside any project)
- Running `ob link <name>` creates a `.obeya-link` file in the git root pointing to the shared board
  - If the board doesn't exist, `ob link` errors with: `board "<name>" not found — run 'ob init --shared <name>' first`
- Multiple projects can link to the same shared board, sharing tasks across repos
- All `ob` commands (list, create, move, etc.) automatically resolve through the link — no extra flags needed
- `ob list` on a shared board returns **all items from all linked projects** — use each item's `project` field to tell which project owns it
- Items created on a shared board automatically get their `project` field set to the current project name (derived from git remote `org/repo` or directory name)

### Migration details (`--migrate`)

When linking a project that has an existing local board:
- `ob link <name>` without `--migrate` errors if local tasks exist, telling you to rerun with `--migrate`
- `ob link <name> --migrate` copies all local items to the shared board, tags each with the project name, and assigns new display numbers
- The local `.obeya/` directory is renamed to `.obeya-local-backup/` (not deleted) so you can recover if needed
- Migration is additive — existing shared board items are not affected
- Item IDs change during migration (prefixed with project name), so any external references to old IDs will break

### Detecting board type

- `.obeya-link` file exists at git root → **linked to a shared board** (read the file for the board name)
- `.obeya/board.json` exists at git root → **local board** (project-specific)
- Neither exists → **no board initialized**

## Prerequisite

The `ob` CLI binary must be installed and available in PATH. Install via:
```bash
brew tap schoolofai/tap && brew install obeya
```
