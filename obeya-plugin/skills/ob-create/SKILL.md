---
description: Create a new board item (epic, story, or task). Use when adding new work items to the board.
disable-model-invocation: false
user-invocable: true
---

# Create Board Item

Create a new standalone item on the Obeya board.

## Usage

`/ob-create <type> "<title>" [flags]`

- `type`: epic, story, or task
- `title`: the item title
- Optional flags: `--priority <low|medium|high|critical>`, `--description "<text>"`, `--assign <user>`, `--tag "<tags>"`

## Steps

1. Parse `$ARGUMENTS` for type, title, and optional flags
2. If type is missing, ask the user: "What type? (epic / story / task)"
3. If title is missing, ask the user for a title
4. Run `ob create <type> "<title>" [flags]`
5. Display the created item with its ID and display number

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
2. Check if any plan is contextually relevant (by title match against the new item)
3. If a matching plan is found:
   - Run `ob plan link <plan-id> --to <new-item-id>`
   - Tell the user: "Linked to plan: <plan-title>"
4. If no plan found at all, say "No plan linked."
