---
description: Create a subtask under a parent item. Use when breaking down work during implementation, when a plan is decomposed into tasks, or when any workflow creates sub-items that need board tracking. Preferred over ob-create when the work has a clear parent epic or story.
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
3. If parent not provided, use the current in-progress item. If ambiguous, ask the user.
4. Run `ob create <type> "<title>" -p <parent-id> [flags]`
5. Display the created item with its ID, display number, and parent relationship

## Description Quality (REQUIRED)

Follow the same description quality standards as `/ob-create` — self-contained descriptions with named headers (What to do, Key files, How to verify, Dependencies). Verification must be an executable command, not prose. If no `--description` flag was provided, generate one from conversation context.

## Plan Linking (REQUIRED — do this after every create)

Follow the same plan linking steps as `/ob-create`: check for unimported plans first, then link to existing plans. The parent item's plan link is inherited — if the parent is linked to a plan, link this subtask to the same plan.
