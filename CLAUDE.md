
## CRITICAL: Validation Before Any Commit

**Every agent MUST run the test suite before committing or claiming work is done.** This is non-negotiable.

```bash
./scripts/test.sh
```

This runs 4 checks in order — all must pass:
1. **Build** — `go build ./...` (compile check)
2. **Vet** — `go vet ./...` (static analysis)
3. **Tests** — `go test ./...` (257 tests across 10 packages)
4. **Golden files** — `go test ./internal/tui/ -run TestGolden` (8 TUI visual snapshots)

If golden file tests fail after intentional TUI changes, regenerate with:
```bash
./scripts/test.sh --update
```

Quick options for faster feedback during development:
```bash
./scripts/test.sh --tui       # TUI tests only
./scripts/test.sh --golden    # golden file snapshots only
```

**Never skip this.** A failing test suite means the work is not done.

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

## TUI Testing — teatest

All TUI features MUST have automated tests using Charm's `teatest` library. The TUI is the primary user interface and regressions are unacceptable.

### Setup

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/x/exp/teatest"
)
```

### Core Pattern

```go
func TestFeature(t *testing.T) {
    // 1. Create model with fixed terminal size
    m := NewTestApp(boardPath)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

    // 2. Send key presses to navigate
    tm.Send(tea.KeyMsg{Type: tea.KeyRight})  // arrow keys
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})  // letter keys

    // 3. Assert on rendered output
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("expected column header"))
    }, teatest.WithDuration(2*time.Second))

    // 4. Clean up
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

### Best Practices

1. **Always set a fixed terminal size** — use `WithInitialTermSize(120, 40)` to ensure deterministic rendering. Never rely on the host terminal dimensions.
2. **Test navigation paths, not pixel positions** — assert that column headers, card titles, or status text appear in the output after key sequences, not that they appear at specific coordinates.
3. **Use `WaitFor` for async assertions** — the TUI renders asynchronously. Never read output immediately after sending keys; use `WaitFor` with a condition function.
4. **Golden file tests for visual regressions** — use `teatest.RequireEqualOutput(t, out)` for layout-sensitive features. Run `go test -update` to regenerate golden files after intentional changes.
5. **One test per navigation path** — test column scrolling (h/l), card navigation (j/k), description accordion (v/J/K), and view switching independently.
6. **Test at boundary sizes** — test with narrow terminals (80x24), wide terminals (200x50), and the default (120x40) to catch overflow and wrapping bugs.
7. **Every TUI PR must include teatest coverage** — no TUI changes are considered complete without corresponding automated tests that exercise the changed rendering or key handling.

### Key Message Reference

```go
// Arrow keys
tea.KeyMsg{Type: tea.KeyUp}
tea.KeyMsg{Type: tea.KeyDown}
tea.KeyMsg{Type: tea.KeyLeft}
tea.KeyMsg{Type: tea.KeyRight}

// Letter keys
tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}

// Special keys
tea.KeyMsg{Type: tea.KeyEnter}
tea.KeyMsg{Type: tea.KeyEsc}
tea.KeyMsg{Type: tea.KeyTab}
tea.KeyMsg{Type: tea.KeySpace}

// Type a string (sends individual key events)
tm.Type("search text")
```

### Test File Location

TUI tests go in `internal/tui/app_test.go` (integration/navigation tests) and `internal/tui/board_test.go` (unit tests for rendering functions).

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
./scripts/release.sh 0.1.0                          # interactive, with default message
./scripts/release.sh 0.2.0 "feat: shared boards"    # interactive, with custom message
./scripts/release.sh --yes 0.1.0                     # non-interactive (for agents/CI)
./scripts/release.sh -y 0.2.0 "feat: shared boards"  # non-interactive with message
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
