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
