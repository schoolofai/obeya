---
description: Show full details of a board item including description, history, children, and blocked status. Use when you need to understand a task's context, check its history, see its subtasks, or review its full description before starting work.
disable-model-invocation: false
user-invocable: true
---

# Show Item Details

Display complete details of a board item.

## Usage

`/ob-show <id>` — show item details
`/ob-show <id> --verbose` — show item details with rich children info

## Steps

1. If `$ARGUMENTS` is provided, use it as the item ID
2. If no arguments:
   - Run `ob list --status in-progress --format json` to find current work
   - If exactly one in-progress task, show it
   - If multiple, show them and ask which to display
   - If none, ask the user for an item ID or display number
3. Run `ob show <id> --format json`
4. Display the item details in a readable format:
   - **Header**: `#<display_num> — <title>` with type badge (epic/story/task)
   - **Status**: current column, priority, assignee (display `Assignee: @unassigned` when the assignee field is empty)
   - **Project**: if on a shared board, show the item's `project` field (which project owns this item)
   - **Description**: full text
   - **Parent**: if this is a child item, show the parent's title and ID
   - **Children**: list any child items with their status
   - **Children (verbose)**: if `--verbose`, show each child's priority, assignee, blocked-by, and description snippet (80 char max)
   - **Blocked by**: show blocking items if any
   - **Tags**: display all tags
   - **History**: show recent changes (last 5 entries) with timestamps
