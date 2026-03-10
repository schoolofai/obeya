---
description: Create a subtask under a parent item. Use when breaking down work during implementation.
disable-model-invocation: false
user-invocable: true
---

# Create Subtask

Create a child item under a parent on the Obeya board.

## Usage

`/ob-subtask [parent] [type] "<title>" [flags]`

- `parent`: optional display number of the parent item (e.g. `15`)
- `type`: optional item type — epic, story, or task (defaults to `task`)
- `title`: the item title
- Optional flags: `--priority <low|medium|high|critical>`, `--description "<text>"`, `--assign <user>`, `--tag "<tags>"`

**Examples:**
- `/ob-subtask 15 "Fix login redirect"` → task under #15
- `/ob-subtask 15 story "User auth flow"` → story under #15
- `/ob-subtask "Quick bugfix"` → task under current in-progress item
- `/ob-subtask story "Design the API"` → story under current in-progress item

## Steps

1. Parse `$ARGUMENTS` for optional parent number, optional type, title, and optional flags
   - A leading number is the parent display number
   - A word matching `epic`, `story`, or `task` (before the title) is the type
   - Everything else in quotes is the title
   - If no type specified, default to `task`
2. If title is missing, ask the user for a title

### Resolve parent

3. If parent display number was provided, use it directly
4. If parent was NOT provided:
   a. Run `ob list --status in-progress --format json`
   b. If exactly **one** in-progress item → use it as parent
   c. If **multiple** in-progress items → list them with display numbers and ask the user to pick:
      ```
      Multiple items in progress:
      #12 - Build auth system (epic)
      #15 - Implement login page (task)
      #18 - Design API schema (story)
      Which item should be the parent?
      ```
   d. If **no** in-progress items → ask the user for a parent display number

### Create the item

5. Run `ob create <type> "<title>" -p <parent-id> [flags]`
6. Display the created item with its ID, display number, and parent relationship

## Plan Linking (REQUIRED — do this after every create)

You MUST check for and link plans after creating the item. Do NOT skip this section.

### Check for unimported plans first

If a plan document was written, discussed, or approved in this conversation (from plan mode, a design doc, an implementation plan, or any markdown plan file), and it has NOT yet been imported into `ob plan`:

1. Save the plan content to a temporary file if it doesn't already exist on disk
2. Run `ob plan import <path-to-plan-file> --link <new-item-id>` to import and link in one step
3. Tell the user: "Imported and linked plan: <plan-title>"

### Link to existing plans

If no new plan needs importing:

1. Run `ob plan list --format json` to get all plans with their linked item IDs
2. Check if the **parent item's ID** appears in any plan's `linked_items`
3. If a matching plan is found:
   - Run `ob plan link <plan-id> --to <new-item-id>`
   - Tell the user: "Linked to plan: <plan-title>"
4. If no parent match, check if any plan is contextually relevant (by title match against the item or its parent)
   - If found, run `ob plan link <plan-id> --to <new-item-id>`
   - Tell the user which plan was linked
5. If no plan found at all, say "No plan linked."
