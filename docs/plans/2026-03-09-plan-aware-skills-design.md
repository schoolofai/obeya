# Plan-Aware OB Skills — Design

## Problem

The `ob-create` and `ob-pick` skills have no awareness of plans. When creating or picking tasks, the agent never checks if a plan exists or links tasks to plans — even though the `ob plan` CLI commands fully support this.

## Solution

Modify the skill prompts (no CLI changes) to make `ob-create` and `ob-pick` plan-aware using existing `ob plan` commands.

## ob-create Changes

After creating a task:

1. Run `ob plan list --format json` to get all plans with their linked items
2. Check if the parent item's ID appears in any plan's linked items
3. If match found — run `ob plan link <plan-id> --to <new-task-id>` to inherit the plan
4. If no parent match — check conversation context for recently written/discussed `docs/plans/*.md` files, and if one exists, link it
5. Confirm to the user which plan was linked (or note that none was found)

## ob-pick Changes

After picking a task:

1. Run `ob plan list --format json` to get all plans
2. Check if the picked task's ID (or its parent's ID) appears in any plan's linked items
3. If linked to a plan — run `ob plan show <plan-id> --format json`
4. Display: plan title + the section/step most relevant to the picked task (match by task title against plan headings/steps)
5. If no plan linked — proceed as before, no change

## Scope

- Only `ob-create/SKILL.md` and `ob-pick/SKILL.md` are modified
- No CLI code changes
- No new commands
- All other skills unchanged
