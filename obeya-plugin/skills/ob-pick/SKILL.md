---
description: Pick an unassigned task from the board and start working on it. Use proactively when starting work, when asked to pick up a task, or when the current task is done and more work remains. This skill should be used whenever an agent needs to begin new work — including after plan creation, after completing a previous task, or when the user says to start implementing.
disable-model-invocation: false
user-invocable: true
---

# Pick a Task

Claim an unassigned task and move it to in-progress.

## Steps

1. Run `ob list --format json` to get all items
2. Find tasks that are:
   - Status: `backlog` or `todo`
   - Not blocked (empty `blocked_by`)
   - Not assigned (empty `assignee`), OR assigned to the current user
3. Pick the task with the lowest display number (highest priority first if equal)
4. Run `ob move <id> in-progress` to claim it
5. Display the picked task details to the user
6. If no unassigned tasks are available, tell the user

## Plan Context (REQUIRED — do this after every pick)

You MUST check for plan context after picking the task. Do NOT skip this section.

### Check for unimported plans first

If a plan document was written, discussed, or approved in this conversation (from plan mode, a design doc, an implementation plan, or any markdown plan file), and it has NOT yet been imported into `ob plan`:

1. Save the plan content to a temporary file if it doesn't already exist on disk
2. Run `ob plan import <path-to-plan-file> --link <picked-task-id>` to import and link
3. Then continue to show the plan context below

### Show plan context

1. Run `ob plan list --format json` to get all plans
2. Check if the picked task's ID or its parent's ID appears in any plan's `linked_items`
3. If linked to a plan:
   - Run `ob plan show <plan-id> --format json` to get the full plan
   - Display a "**Plan:**" header with the plan title
   - Match the picked task's title against plan headings and step descriptions
   - Display the **most relevant section/step** from the plan (not the entire plan)
   - If no specific section matches, show the first 10 lines of the plan as summary
4. If no plan is linked, say "No plan linked to this task."

## Environment

Set `OB_USER` to your user ID before running commands, or pass `--as <id>`.
