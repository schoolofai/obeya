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

### Option 1: Local development install (recommended for contributors)

Register the plugin directory as a local marketplace, then install from it:

```bash
claude plugin marketplace add /path/to/obeya/obeya-plugin
claude plugin install obeya@obeya-local
```

Restart Claude Code. The `/ob` slash commands will be available.

### Option 2: Quick test (single session)

```bash
claude --plugin-dir ./obeya-plugin
```

### Option 3: Install from marketplace (when published)

```bash
claude plugin install obeya --scope user
```

### Uninstall

```bash
claude plugin uninstall obeya@obeya-local
claude plugin marketplace remove obeya-local
```

## Commands

| Command | Description |
|---|---|
| `/ob` | Show board overview |
| `/ob:pick` | Pick an unassigned task and start working (shows linked plan context) |
| `/ob:done` | Mark current task as done |
| `/ob:status` | Show your assigned tasks |
| `/ob:block` | Report a blocker on current task |
| `/ob:create <title>` | Create a subtask under current work (auto-links to parent's plan) |
| `/ob:plan` | Manage plan documents — import, link, show |

## Setup

1. Initialize a board in your project: `ob init`
2. Register yourself: `ob user add "Your Name" --type human`
3. Register agents: `ob user add "Claude" --type agent --provider claude-code`
4. Create work items: `ob create epic "My Feature"`
5. Use `/ob:pick` in Claude Code to start working

## Updating

When you update the `ob` CLI, re-run `ob init` to refresh the CLAUDE.md instructions in your project. The init command uses versioned markers to replace outdated instructions without affecting the rest of your CLAUDE.md.

```bash
ob init
```

This is safe to run on existing boards — it will skip board creation and only update the CLAUDE.md section.

## How It Works

Each slash command wraps `ob` CLI commands. The plugin is skills-only — no MCP server required. Claude executes `ob` commands via bash and presents the results.

Plan-aware features:
- `/ob:pick` checks if the picked task is linked to a plan and displays the relevant section
- `/ob:create` auto-links new subtasks to their parent's plan
- Both commands will import unimported plan documents from the current conversation
