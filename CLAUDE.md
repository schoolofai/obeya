
<!-- obeya:start --> v3

## Task Tracking — Obeya

This project uses Obeya (`ob`) for task tracking. The board is the single source of truth for all work.

### Mandatory: Track ALL work
Every piece of work MUST have a task on the board. Before starting any work:
1. Run `/ob:status` to check assigned tasks
2. If no task exists for this work, create one with `ob create task "Title" --description "..."`
3. Run `/ob:pick` to claim a task when implementing from the backlog
4. Run `/ob:done` when work is complete

### Creating tasks from plans
When breaking down a plan into tasks, create a full hierarchy with detailed descriptions:
- **Epics**: High-level goals. Description includes the objective, success criteria, and scope boundaries.
- **Stories**: Deliverable units. Description includes what needs to be built, why it matters, and acceptance criteria.
- **Tasks**: Atomic work items. Description includes what to do, how to verify it's done, and dependencies on other tasks.

Task descriptions must be self-contained — an agent picking one up should have everything needed to start work. Include key context inline and reference files for larger context (e.g., "See docs/plans/auth-design.md section 3 for protocol details" or "See src/auth/oauth.go for existing implementation").

### Task lifecycle
- Starting work: `ob move <id> in-progress`
- Update progress: `ob edit <id> --description "..."` — append notes as you work (discoveries, approach changes, blockers hit)
- Blocked: `ob block <id> --by <blocker-id>`
- Done: `ob move <id> done`

### Plan management
When a plan document is created, discussed, or approved:
1. Import it: `ob plan import <path-to-plan.md>`
2. Break it down into epics, stories, and tasks with full descriptions
3. Link tasks to plan: `ob plan link <plan-id> --to <task-ids>`
4. When creating subtasks under a plan-linked parent, link them too: `ob plan link <plan-id> --to <new-task-id>`

Use `ob list --format json` for full board state.

<!-- obeya:end -->
