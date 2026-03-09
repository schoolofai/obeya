# Obeya Distribution & Claude Code Plugin — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up binary distribution via GoReleaser + Homebrew tap, create a Claude Code skills-only plugin, and add CLAUDE.md project hook to `ob init`.

**Architecture:** GoReleaser builds cross-platform binaries on tag push via GitHub Actions, auto-updates a Homebrew tap repo. A separate plugin directory contains skill files that wrap `ob` CLI commands for Claude Code slash commands. The `ob init` command is extended to inject board instructions into project CLAUDE.md.

**Tech Stack:** GoReleaser, GitHub Actions, GitHub CLI (`gh`), Claude Code plugin format

**Design Doc:** `docs/plans/2026-03-09-obeya-distribution-design.md`

**NOTE:** GitHub account is `schoolofai` (not `niladribose`). All repo references use `schoolofai`.

---

## Task 1: Create Homebrew Tap Repository

**Files:**
- Create: remote repo `schoolofai/homebrew-tap` on GitHub

**Step 1: Create the homebrew-tap repo on GitHub**

Run:
```bash
gh repo create schoolofai/homebrew-tap --public --description "Homebrew tap for Obeya CLI" --clone=false
```
Expected: Repository created at `https://github.com/schoolofai/homebrew-tap`

**Step 2: Initialize the repo with a README**

Run:
```bash
cd /tmp && mkdir homebrew-tap && cd homebrew-tap && git init
```

Create `README.md`:
```markdown
# Homebrew Tap

Homebrew formulae for [Obeya](https://github.com/schoolofai/obeya) — CLI Kanban board for humans and AI agents.

## Install

```bash
brew tap schoolofai/tap
brew install obeya
```
```

Run:
```bash
git add README.md && git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/schoolofai/homebrew-tap.git
git push -u origin main
cd /Users/niladribose/code/obeya && rm -rf /tmp/homebrew-tap
```
Expected: Remote repo has README.md on main branch

**Step 3: Commit**

No commit needed in obeya repo — this was external setup.

---

## Task 2: Create GoReleaser Configuration

**Files:**
- Create: `.goreleaser.yml`

**Step 1: Create GoReleaser config**

Create `.goreleaser.yml`:

```yaml
version: 2

project_name: obeya

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - main: .
    binary: ob
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

brews:
  - repository:
      owner: schoolofai
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    name: obeya
    homepage: "https://github.com/schoolofai/obeya"
    description: "CLI Kanban board for humans and AI agents"
    license: "MIT"
    install: |
      bin.install "ob"
    test: |
      system "#{bin}/ob", "--help"
```

**Step 2: Verify GoReleaser config syntax**

Run:
```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser check
```
Expected: No errors in config

**Step 3: Commit**

```bash
git add .goreleaser.yml
git commit -m "feat: add GoReleaser config for cross-platform builds and Homebrew tap"
```

---

## Task 3: Create GitHub Actions Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Create the workflow directory**

Run:
```bash
mkdir -p .github/workflows
```

**Step 2: Create the release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

**Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow for GoReleaser"
```

---

## Task 4: Add Version Info to Binary

**Files:**
- Modify: `main.go`
- Modify: `cmd/root.go`

**Step 1: Add version variables to main.go**

Update `main.go` to include linker-injected version variables:

```go
package main

import "github.com/niladribose/obeya/cmd"

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}
```

**Step 2: Add version command to root.go**

Read `cmd/root.go` first, then add a `SetVersionInfo` function and wire up the `--version` flag. Add to the file:

```go
var (
	appVersion = "dev"
	appCommit  = "none"
)

func SetVersionInfo(version, commit string) {
	appVersion = version
	appCommit = commit
	rootCmd.Version = version + " (" + commit + ")"
}
```

**Step 3: Build and verify**

Run:
```bash
go build -o ob . && ./ob --version
```
Expected: Output shows `ob version dev (none)`

**Step 4: Commit**

```bash
git add main.go cmd/root.go
git commit -m "feat: add version info with linker flags for GoReleaser"
```

---

## Task 5: Create Claude Code Plugin Structure

**Files:**
- Create: `obeya-plugin/.claude-plugin/plugin.json`
- Create: `obeya-plugin/skills/ob/SKILL.md`
- Create: `obeya-plugin/skills/ob-pick/SKILL.md`
- Create: `obeya-plugin/skills/ob-done/SKILL.md`
- Create: `obeya-plugin/skills/ob-status/SKILL.md`
- Create: `obeya-plugin/skills/ob-block/SKILL.md`
- Create: `obeya-plugin/skills/ob-create/SKILL.md`

**Step 1: Create plugin.json**

Create `obeya-plugin/.claude-plugin/plugin.json`:

```json
{
  "name": "obeya",
  "description": "Claude Code integration for the Obeya Kanban board CLI",
  "version": "1.0.0",
  "author": {
    "name": "Niladri Bose"
  },
  "homepage": "https://github.com/schoolofai/obeya",
  "repository": "https://github.com/schoolofai/obeya",
  "license": "MIT",
  "keywords": ["kanban", "tasks", "cli", "productivity", "agents"]
}
```

**Step 2: Create /ob skill (board overview)**

Create `obeya-plugin/skills/ob/SKILL.md`:

```markdown
---
description: Show the Obeya Kanban board overview. Use when user asks about tasks, board status, or project progress.
disable-model-invocation: false
user-invocable: true
---

# Obeya Board Overview

Show the current state of the Kanban board.

## Steps

1. Run `ob list --format json` to get all items
2. Run `ob board config --format json` to get column names and board settings
3. Display the results as a formatted board:
   - Group items by column/status
   - Show each item with: display number, type (epic/story/task), title, priority
   - Mark blocked items with [BLOCKED]
   - Show item counts per column
4. If the board is not initialized, tell the user to run `ob init`

## Prerequisite

The `ob` CLI binary must be installed and available in PATH. Install via:
```bash
brew tap schoolofai/tap && brew install obeya
```
```

**Step 3: Create /ob:pick skill**

Create `obeya-plugin/skills/ob-pick/SKILL.md`:

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

## Environment

Set `OB_USER` to your user ID before running commands, or pass `--as <id>`.
```

**Step 4: Create /ob:done skill**

Create `obeya-plugin/skills/ob-done/SKILL.md`:

```markdown
---
description: Mark the current in-progress task as done. Use when work is completed on a task.
disable-model-invocation: false
user-invocable: true
---

# Complete Current Task

Mark the current in-progress task as done and show what's next.

## Steps

1. Run `ob list --status in-progress --format json` to find in-progress items
2. If using `OB_USER`, filter to items assigned to the current user
3. If exactly one in-progress task: run `ob move <id> done`
4. If multiple in-progress tasks: show them and ask the user which to complete
5. If `$ARGUMENTS` is provided, use it as the task ID: run `ob move $ARGUMENTS done`
6. After completing, run `ob list --status todo --format json` to show the next available tasks
7. If all children of a parent are now done, suggest moving the parent to done too
```

**Step 5: Create /ob:status skill**

Create `obeya-plugin/skills/ob-status/SKILL.md`:

```markdown
---
description: Show tasks assigned to you or the current agent. Use to check what you're working on.
disable-model-invocation: false
user-invocable: true
---

# Show My Status

Display all items assigned to the current user/agent.

## Steps

1. Determine the current user:
   - Check `OB_USER` environment variable
   - If not set, run `ob user list --format json` and identify the likely current user
2. Run `ob list --format json` and filter to items where `assignee` matches the current user
3. Group by status and display:
   - In-progress items first (what you're actively working on)
   - Todo items next (what's queued)
   - Blocked items highlighted with their blockers
4. Show a summary: X in-progress, Y todo, Z blocked
```

**Step 6: Create /ob:block skill**

Create `obeya-plugin/skills/ob-block/SKILL.md`:

```markdown
---
description: Report a blocker on the current task. Use when work cannot proceed due to a dependency.
disable-model-invocation: false
user-invocable: true
---

# Report a Blocker

Mark the current task as blocked by another item.

## Steps

1. If `$ARGUMENTS` is provided, parse it as `<blocker-id>`:
   - Find the current in-progress task (from `ob list --status in-progress --format json`)
   - Run `ob block <current-task-id> --by $ARGUMENTS`
2. If no arguments:
   - Show current in-progress tasks
   - Run `ob list --format json` and display available items that could be blockers
   - Ask the user which item is blocking
3. After blocking, suggest moving to another available task
```

**Step 7: Create /ob:create skill**

Create `obeya-plugin/skills/ob-create/SKILL.md`:

```markdown
---
description: Quickly create a subtask under the current work item. Use when breaking down work during implementation.
disable-model-invocation: false
user-invocable: true
---

# Quick Create Subtask

Create a new task under the currently active work item.

## Steps

1. The title is provided via `$ARGUMENTS`
2. Find the current in-progress task from `ob list --status in-progress --format json`
3. Run `ob create task "$ARGUMENTS" -p <parent-id>`
4. Display the created task with its ID
5. If no arguments provided, ask the user for a task title
6. If no in-progress parent found, ask the user which item to create the task under
```

**Step 8: Commit**

```bash
git add obeya-plugin/
git commit -m "feat: add Claude Code skills-only plugin with /ob slash commands"
```

---

## Task 6: Add CLAUDE.md Injection to `ob init`

**Files:**
- Modify: `cmd/init.go`

**Step 1: Read current init.go**

Read `cmd/init.go` to understand current structure.

**Step 2: Add CLAUDE.md injection**

Add a `--claude-md` flag (default: true) to `ob init`. After creating the board, append Obeya instructions to the project's `CLAUDE.md`:

```go
var initClaudeMD bool

// In init() function:
initCmd.Flags().BoolVar(&initClaudeMD, "claude-md", true, "append Obeya instructions to project CLAUDE.md")

// After board creation in RunE:
if initClaudeMD {
    if err := appendClaudeMD(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not update CLAUDE.md: %v\n", err)
    } else {
        fmt.Println("Updated CLAUDE.md with Obeya board instructions")
    }
}
```

The `appendClaudeMD` function:

```go
func appendClaudeMD() error {
    claudeMDPath := "CLAUDE.md"

    content := `
## Task Tracking — Obeya

This project uses Obeya (` + "`ob`" + `) for task tracking. Before starting work:
1. Run ` + "`/ob:status`" + ` to check assigned tasks
2. Run ` + "`/ob:pick`" + ` to claim a task if none assigned
3. Run ` + "`/ob:done`" + ` when work is complete

Use ` + "`ob list --format json`" + ` for full board state.
`

    existing, err := os.ReadFile(claudeMDPath)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to read CLAUDE.md: %w", err)
    }

    // Don't duplicate if already present
    if strings.Contains(string(existing), "Task Tracking — Obeya") {
        return nil
    }

    f, err := os.OpenFile(claudeMDPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("failed to open CLAUDE.md: %w", err)
    }
    defer f.Close()

    if _, err := f.WriteString(content); err != nil {
        return fmt.Errorf("failed to write to CLAUDE.md: %w", err)
    }

    return nil
}
```

**Step 3: Build and test**

Run:
```bash
go build -o ob . && cd /tmp && mkdir ob-claude-test && cd ob-claude-test
/Users/niladribose/code/obeya/ob init test
cat CLAUDE.md
/Users/niladribose/code/obeya/ob init test2 --claude-md=false  # should fail (board exists) but won't write CLAUDE.md
cd /Users/niladribose/code/obeya && rm -rf /tmp/ob-claude-test
```
Expected: CLAUDE.md created with Obeya section

**Step 4: Commit**

```bash
git add cmd/init.go
git commit -m "feat: inject Obeya instructions into CLAUDE.md on board init"
```

---

## Task 7: Create Plugin README

**Files:**
- Create: `obeya-plugin/README.md`

**Step 1: Create README**

Create `obeya-plugin/README.md`:

```markdown
# Obeya — Claude Code Plugin

Claude Code integration for the [Obeya](https://github.com/schoolofai/obeya) CLI Kanban board.

## Prerequisites

Install the `ob` binary first:

```bash
brew tap schoolofai/tap
brew install obeya
```

Or download from [GitHub Releases](https://github.com/schoolofai/obeya/releases).

## Installation

### Local testing

```bash
claude --plugin-dir ./obeya-plugin
```

### Permanent install

```bash
claude plugin install obeya --scope user
```

## Commands

| Command | Description |
|---|---|
| `/ob` | Show board overview |
| `/ob:pick` | Pick an unassigned task and start working |
| `/ob:done` | Mark current task as done |
| `/ob:status` | Show your assigned tasks |
| `/ob:block` | Report a blocker on current task |
| `/ob:create <title>` | Create a subtask under current work |

## Setup

1. Initialize a board in your project: `ob init`
2. Register yourself: `ob user add "Your Name" --type human`
3. Register agents: `ob user add "Claude" --type agent --provider claude-code`
4. Create work items: `ob create epic "My Feature"`
5. Use `/ob:pick` in Claude Code to start working

## How It Works

Each slash command wraps `ob` CLI commands. The plugin is skills-only — no MCP server required. Claude executes `ob` commands via bash and presents the results.
```

**Step 2: Commit**

```bash
git add obeya-plugin/README.md
git commit -m "docs: add plugin README with install and usage instructions"
```

---

## Task 8: Create GitHub Repo and Push

**Files:**
- No new files — push existing repo to GitHub

**Step 1: Create GitHub repository**

Run:
```bash
gh repo create schoolofai/obeya --public --description "CLI Kanban board for humans and AI agents" --source . --push
```
Expected: Repo created and all commits pushed

**Step 2: Verify on GitHub**

Run:
```bash
gh repo view schoolofai/obeya --web
```
Expected: Opens browser showing the repo with all files

**Step 3: Add HOMEBREW_TAP_TOKEN secret**

This is a manual step. Create a GitHub personal access token with `repo` scope for `schoolofai/homebrew-tap`, then:

Run:
```bash
gh secret set HOMEBREW_TAP_TOKEN --repo schoolofai/obeya
```
Expected: Prompts for token value, stores as repo secret

---

## Task 9: Test Release Pipeline

**Step 1: Do a dry run of GoReleaser**

Run:
```bash
goreleaser release --snapshot --clean
```
Expected: Builds binaries in `dist/` for all platforms without publishing

**Step 2: Verify built binaries**

Run:
```bash
ls dist/
./dist/obeya_darwin_arm64_v8.0/ob --help
```
Expected: Binary runs and shows help

**Step 3: Tag and release**

Run:
```bash
git tag v0.1.0
git push origin v0.1.0
```
Expected: GitHub Actions triggers, GoReleaser creates release, Homebrew tap updates

**Step 4: Verify release**

Run:
```bash
gh release view v0.1.0 --repo schoolofai/obeya
```
Expected: Release with binaries for all 5 platform/arch combinations

**Step 5: Verify Homebrew tap**

Run:
```bash
gh repo view schoolofai/homebrew-tap --json content
```
Expected: `Formula/obeya.rb` file exists in the tap repo

---

## Task 10: Test Plugin Locally

**Step 1: Test plugin loads in Claude Code**

Run:
```bash
claude --plugin-dir /Users/niladribose/code/obeya/obeya-plugin
```

Then in the Claude Code session:
```
/ob
```
Expected: Claude runs `ob list --format json` and shows the board

**Step 2: Test each slash command**

In a test directory with an initialized board:
```bash
mkdir /tmp/ob-plugin-test && cd /tmp/ob-plugin-test
ob init test-project
ob user add "Tester" --type human
ob create epic "Test Feature"
ob create task "Do something" -p 1
```

Then in Claude Code with the plugin:
```
/ob:status
/ob:pick
/ob:done
/ob:create "Subtask from plugin"
```
Expected: Each command works, items move correctly on the board

**Step 3: Clean up**

```bash
rm -rf /tmp/ob-plugin-test
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Create Homebrew tap repo | GitHub repo: schoolofai/homebrew-tap |
| 2 | GoReleaser config | `.goreleaser.yml` |
| 3 | GitHub Actions release workflow | `.github/workflows/release.yml` |
| 4 | Version info in binary | `main.go`, `cmd/root.go` |
| 5 | Claude Code plugin (6 skills) | `obeya-plugin/` directory |
| 6 | CLAUDE.md injection in `ob init` | `cmd/init.go` |
| 7 | Plugin README | `obeya-plugin/README.md` |
| 8 | Push to GitHub | Remote repo setup |
| 9 | Test release pipeline | Tag + GoReleaser dry run |
| 10 | Test plugin locally | Manual Claude Code testing |

**Total: 10 tasks**

**Parallelizable groups:**
- Tasks 1-4 (release pipeline) — sequential, dependency chain
- Task 5-7 (plugin) — can run in parallel with Tasks 2-4
- Tasks 8-10 (integration testing) — sequential, after all above
