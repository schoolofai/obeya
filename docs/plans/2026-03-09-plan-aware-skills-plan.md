# Plan-Aware OB Skills Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `ob-create` and `ob-pick` skills plan-aware so tasks automatically get linked to plans and agents see plan context when picking tasks.

**Architecture:** Modify two SKILL.md files to add plan-checking steps using existing `ob plan` CLI commands. No Go code changes.

**Tech Stack:** Markdown skill files, `ob` CLI

---

### Task 1: Update ob-create skill to link new tasks to plans

**Files:**
- Modify: `obeya-plugin/skills/ob-create/SKILL.md`

**Step 1: Replace the skill file with plan-aware version**

Replace the entire content of `obeya-plugin/skills/ob-create/SKILL.md` with:

```markdown
---
description: Quickly create a subtask under the current work item. Use when breaking down work during implementation.
disable-model-invocation: false
user-invocable: true
---

# Quick Create Subtask

Create a new task under the currently active work item, and link it to any relevant plan.

## Steps

1. The title is provided via `$ARGUMENTS`
2. Find the current in-progress task from `ob list --status in-progress --format json`
3. Run `ob create task "$ARGUMENTS" -p <parent-id>`
4. Display the created task with its ID
5. If no arguments provided, ask the user for a task title
6. If no in-progress parent found, ask the user which item to create the task under

## Plan Linking

After creating the task, link it to a plan:

1. Run `ob plan list --format json` to get all plans with their linked item IDs
2. Check if the **parent item's ID** appears in any plan's `linked_items`
3. If a matching plan is found:
   - Run `ob plan link <plan-id> --to <new-task-id>`
   - Tell the user: "Linked to plan: <plan-title>"
4. If no parent match, check the current conversation for any recently written or discussed `docs/plans/*.md` file:
   - If found, import or link it: `ob plan link <plan-id> --to <new-task-id>`
   - Tell the user which plan was linked
5. If no plan found at all, continue without linking (do not error)
```

**Step 2: Verify the file reads correctly**

Run: `cat obeya-plugin/skills/ob-create/SKILL.md`
Expected: The updated content with the "Plan Linking" section

**Step 3: Commit**

```bash
git add obeya-plugin/skills/ob-create/SKILL.md
git commit -m "feat: make ob-create skill plan-aware with auto-linking"
```

---

### Task 2: Update ob-pick skill to show plan context

**Files:**
- Modify: `obeya-plugin/skills/ob-pick/SKILL.md`

**Step 1: Replace the skill file with plan-aware version**

Replace the entire content of `obeya-plugin/skills/ob-pick/SKILL.md` with:

```markdown
---
description: Pick an unassigned task from the board and start working on it. Use proactively when starting work or when asked to pick up a task.
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

## Plan Context

After picking the task, surface relevant plan context:

1. Run `ob plan list --format json` to get all plans
2. Check if the picked task's ID or its parent's ID appears in any plan's `linked_items`
3. If linked to a plan:
   - Run `ob plan show <plan-id> --format json` to get the full plan
   - Display the **plan title** as a header
   - Match the picked task's title against plan headings and step descriptions
   - Display the **most relevant section/step** from the plan (not the entire plan)
   - If no specific section matches, show the first 10 lines of the plan as summary
4. If no plan is linked, proceed normally without mentioning plans

## Environment

Set `OB_USER` to your user ID before running commands, or pass `--as <id>`.
```

**Step 2: Verify the file reads correctly**

Run: `cat obeya-plugin/skills/ob-pick/SKILL.md`
Expected: The updated content with the "Plan Context" section

**Step 3: Commit**

```bash
git add obeya-plugin/skills/ob-pick/SKILL.md
git commit -m "feat: make ob-pick skill plan-aware with context display"
```

---

### Task 3: Manual verification

**Step 1: Test ob-create with a plan-linked parent**

1. Ensure a plan exists: `ob plan list --format json`
2. Ensure an in-progress task is linked to that plan
3. Run `/ob:create "Test subtask"`
4. Verify the new task gets linked to the same plan

**Step 2: Test ob-pick with a plan-linked task**

1. Ensure a backlog/todo task exists that is linked to a plan
2. Run `/ob:pick`
3. Verify the plan context is displayed with the relevant section

**Step 3: Test both skills with no plans**

1. Use a board with no plans
2. Run `/ob:create "No plan task"` and `/ob:pick`
3. Verify both work normally without errors
