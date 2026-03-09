# Obeya — Distribution & Claude Code Integration Design

## Overview

Three-part plan to distribute the `ob` binary and integrate it into Claude Code sessions.

## Part 1: Distribution — GoReleaser + Homebrew

### Pipeline

```
git tag v0.1.0 && git push --tags
         │
         ▼
  GitHub Actions CI
         │
         ▼
    GoReleaser
    ├── Build: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
    ├── Create GitHub Release with checksums
    └── Update Homebrew tap (niladribose/homebrew-tap)
```

### Install Methods

```bash
# Homebrew (macOS/Linux)
brew tap niladribose/tap
brew install obeya

# Direct download from GitHub Releases

# Go developers (free)
go install github.com/niladribose/obeya@latest
```

### Files Needed

- `.goreleaser.yml` — build targets, archive formats, Homebrew tap config
- `.github/workflows/release.yml` — triggered on tag push, runs GoReleaser
- Separate repo: `niladribose/homebrew-tap` — GoReleaser auto-updates the formula

### Homebrew Tap Setup

1. Create GitHub repo `niladribose/homebrew-tap`
2. Add GitHub personal access token as `HOMEBREW_TAP_TOKEN` secret in obeya repo
3. GoReleaser config references the tap repo:

```yaml
brews:
  - repository:
      owner: niladribose
      name: homebrew-tap
    name: obeya
    homepage: "https://github.com/niladribose/obeya"
    description: "CLI Kanban board for humans and AI agents"
    install: |
      bin.install "ob"
```

## Part 2: Claude Code Plugin (Skills-Only)

A skills-only plugin — no MCP server needed. Each skill wraps `ob` CLI commands via bash.

### Plugin Structure

```
obeya-plugin/
├── .claude-plugin/
│   └── plugin.json
├── skills/
│   ├── ob/
│   │   └── SKILL.md         # /ob — board overview
│   ├── ob-pick/
│   │   └── SKILL.md         # /ob:pick — claim a task
│   ├── ob-done/
│   │   └── SKILL.md         # /ob:done — complete current task
│   ├── ob-status/
│   │   └── SKILL.md         # /ob:status — show assigned items
│   ├── ob-block/
│   │   └── SKILL.md         # /ob:block — report blocker
│   └── ob-create/
│       └── SKILL.md         # /ob:create — quick subtask
└── README.md
```

### plugin.json

```json
{
  "name": "obeya",
  "description": "Claude Code integration for the Obeya Kanban board CLI",
  "version": "1.0.0",
  "author": {
    "name": "Niladri Bose"
  },
  "homepage": "https://github.com/niladribose/obeya",
  "repository": "https://github.com/niladribose/obeya",
  "license": "MIT",
  "keywords": ["kanban", "tasks", "cli", "productivity"]
}
```

### Skill Commands

| Command | Action |
|---|---|
| `/ob` | Run `ob list --format json`, show board overview as formatted table |
| `/ob:pick` | Run `ob list --format json`, find unassigned tasks, pick lowest ID, run `ob move <id> in-progress` |
| `/ob:done` | Find current in-progress task for this agent, run `ob move <id> done`, show next available |
| `/ob:status` | Run `ob list --assignee $OB_USER --format json`, show assigned items |
| `/ob:block` | Accept `$ARGUMENTS` as blocker ID, run `ob block <current-task> --by <id>` |
| `/ob:create` | Accept `$ARGUMENTS` as title, run `ob create task "$ARGUMENTS" -p <current-parent>` |

### Skill Frontmatter

Each SKILL.md uses:

```yaml
---
description: <what this command does>
disable-model-invocation: false
user-invocable: true
---
```

`disable-model-invocation: false` allows Claude to invoke proactively based on context.

### Installation

```bash
# Local testing
claude --plugin-dir ./obeya-plugin

# Public distribution (via marketplace or git)
claude plugin install obeya --scope user
```

### Prerequisite

The `ob` binary must be installed separately (via Homebrew or GitHub Releases). The plugin only teaches Claude how to use it.

## Part 3: CLAUDE.md Project Hook

When `ob init` runs, optionally append to the project's `CLAUDE.md`:

```markdown
## Task Tracking — Obeya

This project uses Obeya (`ob`) for task tracking. Before starting work:
1. Run `/ob:status` to check assigned tasks
2. Run `/ob:pick` to claim a task if none assigned
3. Run `/ob:done` when work is complete

Use `ob list --format json` for full board state.
```

This is a lightweight hint — not mandatory. The plugin's skill descriptions handle the heavy lifting.

## Design Decisions Summary

| Decision | Choice |
|---|---|
| Binary distribution | GoReleaser + GitHub Actions |
| Package manager | Homebrew tap (niladribose/homebrew-tap) |
| Cross-platform | darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64 |
| Claude Code integration | Skills-only plugin (no MCP server) |
| Slash commands | /ob, /ob:pick, /ob:done, /ob:status, /ob:block, /ob:create |
| Plugin distribution | Git repo, installable via claude plugin install |
| Project awareness | CLAUDE.md injection via ob init |
