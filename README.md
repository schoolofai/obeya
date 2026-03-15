# Obeya

> **Agent support:** Only **Claude Code** is currently supported as an AI agent. Other agents (Cursor, Windsurf, Copilot, etc.) are planned but not yet available.

CLI Kanban board for humans and AI agents.

Obeya (`ob`) is a lightweight, file-based task tracker that lives in your project directory. It works standalone from the terminal and integrates with AI coding assistants like Claude Code via the included plugin.

## Install

### macOS

```bash
# Homebrew (recommended)
brew tap schoolofai/tap
brew install obeya

# Or use the install script
curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh
```

### Linux

```bash
# Install script (recommended)
curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh

# Or install system-wide
curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh -s -- --global

# Or download .deb / .rpm from GitHub Releases
# Debian/Ubuntu:
#   sudo dpkg -i obeya_<version>_linux_amd64.deb
# RHEL/Fedora:
#   sudo rpm -i obeya_<version>_linux_amd64.rpm
```

### Windows

```powershell
# PowerShell install script (recommended)
irm https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.ps1 | iex

# Or via Scoop
scoop bucket add obeya https://github.com/schoolofai/scoop-obeya
scoop install obeya
```

### From source (all platforms)

```bash
go install github.com/niladribose/obeya@latest
```

### Binary download

Grab the latest release from [GitHub Releases](https://github.com/schoolofai/obeya/releases).

## Quick Start

### With Claude Code (recommended)

```bash
# 1. Install obeya
brew tap schoolofai/tap
brew install obeya

# 2. Install the Claude Code plugin (gives your agent ob skills + hooks)
ob plugin claude-install

# 3. Initialize a board — pick one:

# Option A: Local board (per-project, stored in .obeya/)
ob init --agent claude-code

# Option B: Shared board (global, stored in ~/.obeya/boards/)
ob init --shared myboard --agent claude-code
# Then link any project to it:
ob link myboard

# Or just start a Claude Code session — the agent will suggest running ob init
```

### Standalone (terminal only)

```bash
# Initialize a board in your project
ob init --shared myboard

# Register yourself
ob user add "Your Name" --type human

# Create work items
ob create epic "Auth System" -d "Build authentication system"
ob create story "OAuth flow" -d "Implement OAuth2" --parent <epic-id>
ob create task "Add token refresh" -d "Handle token expiry" --parent <story-id>

# Work the board
ob list                    # View all items
ob move <id> in-progress   # Start working
ob move <id> done          # Mark complete
```

## Features

- **File-based** -- board stored in `.obeya/` at your project root, committed with your code
- **Shared boards** -- link multiple projects to a single board at `~/.obeya/boards/<name>/`
- **Hierarchy** -- epics, stories, and tasks with parent-child relationships
- **Plan tracking** -- import markdown plan documents and link them to tasks
- **Agent-friendly** -- designed for AI agents to pick, work, and complete tasks autonomously

## Board Types

| What you see | Board type |
|---|---|
| `.obeya/board.json` at git root | Local board (project-specific) |
| `.obeya-link` file at git root | Linked to a shared board |

### Shared board commands

```bash
ob init --shared teamboard                    # Create a shared board (no agent setup)
ob init --shared teamboard --agent claude-code # Create shared board + agent setup
ob link teamboard                             # Link current project
ob link teamboard --migrate                   # Link and move existing tasks
ob boards                                     # List all shared boards
ob unlink                                     # Disconnect from shared board
```

## Commands

| Command | Description |
|---|---|
| `ob init` | Initialize a board |
| `ob create <type> <title>` | Create an epic, story, or task |
| `ob list` | List all items |
| `ob show <id>` | Show item details |
| `ob move <id> <status>` | Move item to a status |
| `ob edit <id>` | Edit item fields |
| `ob block <id> --by <id>` | Mark a blocker |
| `ob user add <name>` | Register a user |
| `ob plan import <file>` | Import a plan document |

## Claude Code Plugin

Obeya ships with a Claude Code plugin that gives your AI agent `/ob` slash commands for task management. The plugin installs globally (user scope) so it's available in every project.

```bash
# Install the plugin (one-time, global)
ob plugin claude-install

# Initialize a local board with agent setup
ob init --agent claude-code

# Or create a shared board with agent setup (writes to ~/.claude/CLAUDE.md)
ob init --shared myboard --agent claude-code
```

The plugin provides:
- **SessionStart hook** -- injects Obeya context into every conversation
- **PostToolUse hook** -- reminds the agent to track plans on the board
- **13 skills** -- `/ob`, `/ob-status`, `/ob-pick`, `/ob-done`, `/ob-create`, and more

See [obeya-plugin/README.md](obeya-plugin/README.md) for details.

## License

MIT
