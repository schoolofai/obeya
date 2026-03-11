
<!-- obeya:start --> v5

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

### Obeya board is authoritative over session tools
`TodoWrite` and `TaskCreate` are ephemeral session aids. The Obeya board persists across sessions and is the source of truth. When any skill or workflow uses session tools (e.g., TodoWrite), the corresponding work MUST also exist on the Obeya board. Specifically:
- Before creating a TodoWrite checklist, ensure equivalent tasks exist on the board via `ob create`
- When marking a TodoWrite item complete, also run `ob move <id> done`
- When a skill workflow says "mark task complete in TodoWrite", ALSO run `ob move <id> done`

### Integration with other skills and workflows
When using skills that dispatch subagents or manage work (e.g., superpowers, subagent-driven-development, executing-plans):
- The **controller/orchestrator** (not the subagent) is responsible for obeya board updates
- Before dispatching work: ensure the task exists on the board and is in-progress
- After subagent completes: run `ob move <id> done` on the corresponding board task
- When a plan is broken into subtasks by any skill: create those subtasks on the board too
- This applies regardless of which skill is orchestrating the work

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

## Releasing to Homebrew

The `ob` CLI is distributed via Homebrew through `schoolofai/homebrew-tap`. The release pipeline is fully automated — pushing a git tag triggers everything.

### How it works

```
git tag v0.x.0 → push tag → GitHub Actions → GoReleaser → builds binaries + creates GitHub Release + pushes Formula/obeya.rb to schoolofai/homebrew-tap
```

Key files:
- `.goreleaser.yml` — build config, archive format, Homebrew formula template
- `.github/workflows/release.yml` — GitHub Actions workflow triggered by `v*` tags
- `scripts/release.sh` — release script with preflight checks

### How to release

```bash
./scripts/release.sh 0.1.0                          # with default message
./scripts/release.sh 0.2.0 "feat: shared boards"    # with custom message
```

The script runs preflight checks (clean tree, on main, synced with remote, tests pass), then creates and pushes the tag.

### Secrets required

- `HOMEBREW_TAP_TOKEN` — a GitHub PAT with write access to `schoolofai/homebrew-tap`. Stored in GitHub repo settings > Secrets > Actions. GoReleaser uses it to push the formula file.

### End-user install

```bash
brew tap schoolofai/tap
brew install obeya
```

### Troubleshooting releases

- **Build fails**: check Go version in `.github/workflows/release.yml` matches `go.mod`
- **Formula push fails**: verify `HOMEBREW_TAP_TOKEN` PAT has `repo` or `public_repo` scope on `schoolofai/homebrew-tap`
- **Tests fail in pipeline**: GoReleaser runs `go test ./...` as a pre-hook — fix tests locally first
