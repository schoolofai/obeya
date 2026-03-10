---
description: Edit a board item's title, description, or priority. Use when updating task details, appending progress notes, recording discoveries, or refining task descriptions during implementation. Should be used proactively to keep task descriptions current as work progresses.
disable-model-invocation: false
user-invocable: true
---

# Edit Board Item

Update an item's title, description, or priority on the Obeya board.

## Usage

`/ob-edit <id> [flags]`

- `id`: the item's display number or ID
- Flags:
  - `--title "<new title>"` — change the item's title
  - `--description "<text>"` — replace the item's description
  - `--body-file <path>` — read new description from a file
  - `--priority <low|medium|high|critical>` — change priority

## Steps

1. If `$ARGUMENTS` is provided, parse it for the item ID and flags
2. If no ID is provided:
   - Run `ob list --status in-progress --format json` to find current work
   - If exactly one in-progress task, use it
   - If multiple, show them and ask which to edit
3. Run `ob edit <id> [flags]`
4. Run `ob show <id> --format json` and display the updated item to confirm

## Appending Progress Notes

When updating a task description to add progress notes (discoveries, approach changes, blockers hit), read the current description first with `ob show <id> --format json`, then append the new notes to the existing description rather than replacing it:

```
ob edit <id> --description "<existing description>

---
Progress: <new notes>"
```
