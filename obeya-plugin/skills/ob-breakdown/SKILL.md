---
description: Break down a plan into an epic/story/task hierarchy on the Obeya board. Use after a plan document is created or approved (e.g., from superpowers:writing-plans), when the user says "break this down", "create tasks from plan", or when a plan exists but has no linked board tasks. This is the bridge between planning and execution.
disable-model-invocation: false
user-invocable: true
---

# Break Down Plan into Board Tasks

Import a plan and create a full epic/story/task hierarchy on the Obeya board with self-contained descriptions.

## When to Use

- After `superpowers:writing-plans` creates a plan document
- When a plan file exists but has no linked board tasks
- When the user says "break this down" or "create tasks"
- When `/ob-plan` shows a plan with zero linked items

## Steps

### 1. Find and import the plan

If `$ARGUMENTS` is a file path, use it. Otherwise:

1. Check if a plan was written in this conversation (look for recent plan files in `docs/superpowers/plans/` or similar)
2. Run `ob plan list --format json` — check for plans with zero linked items
3. If no plan found, ask the user for the plan file path

Import if not already imported:
```
ob plan import <path-to-plan.md> --title "<extracted-title>"
```

### 2. Read the plan and identify structure

Read the plan content and identify:
- The **overall goal** → becomes the Epic
- **Major deliverables/milestones** → become Stories
- **Atomic implementation steps** → become Tasks

### 3. Create the Epic

```
ob create epic "<goal title>" --description "<description>"
```

The epic description must include:
- Objective: what we're building and why
- Success criteria: how to know it's done
- Scope: what's in and out
- Reference: "See <plan-file-path> for full plan"

Link to plan: `ob plan link <plan-id> --to <epic-id>`

### 4. Create Stories under the Epic

For each major deliverable:

```
ob create story "<deliverable title>" -p <epic-id> --description "<description>"
```

Story descriptions must include:
- What needs to be built
- Why it matters (user value or technical necessity)
- Acceptance criteria (testable conditions)
- Dependencies on other stories

Link to plan: `ob plan link <plan-id> --to <story-id>`

### 5. Create Tasks under each Story

For each atomic implementation step:

```
ob create task "<step title>" -p <story-id> --description "<description>"
```

Task descriptions MUST be self-contained. Include:
- **What**: specific implementation steps
- **Where**: key file paths to modify/create (e.g., "Modify cmd/show.go lines 45-60, add new flag in init()")
- **How to verify**: test command or expected behavior (e.g., "Run `go test ./cmd/ -run TestShowVerbose` — should pass")
- **Dependencies**: which other tasks must complete first
- **Context**: any architectural decisions or constraints from the plan

Link to plan: `ob plan link <plan-id> --to <task-id>`

### 6. Set up dependencies

If tasks have ordering dependencies:
```
ob block <dependent-task-id> --by <prerequisite-task-id>
```

### 7. Display the result

Show the complete hierarchy:
```
ob list --format json
```

Display as a tree with item numbers, types, and titles. Confirm with the user that the breakdown looks correct.

## Quality Checklist

Before finishing, verify:
- [ ] Every plan step has a corresponding board task
- [ ] All items are linked to the plan (`ob plan show <plan-id>` shows all linked items)
- [ ] Task descriptions include file paths and verification commands
- [ ] Dependencies between tasks are set via `ob block`
- [ ] The hierarchy is epic → stories → tasks (not a flat list)

## Example Output

```
Epic #1: Add metrics command to ob CLI
├── Story #2: Core metrics data model
│   ├── Task #3: Add duration tracking to board items
│   └── Task #4: Create metrics aggregation functions
├── Story #5: CLI command implementation
│   ├── Task #6: Implement `ob metrics` command scaffold
│   ├── Task #7: Add column time calculations
│   └── Task #8: Add throughput stats
└── Story #9: Output formatting
    ├── Task #10: Text table output
    └── Task #11: JSON output format

Plan: "Metrics Command Design" linked to all 11 items
```
