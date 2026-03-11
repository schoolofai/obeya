# Obeya

CLI Kanban board for humans and AI agents.

Obeya (`ob`) is a lightweight, file-based task tracker that lives in your project directory. It works standalone from the terminal and integrates with AI coding assistants like Claude Code via the included plugin.

## Install

### Homebrew (macOS / Linux)

```bash
brew tap schoolofai/tap
brew install obeya
```

### From source

```bash
go install github.com/niladribose/obeya@latest
```

### Binary download

Grab the latest release from [GitHub Releases](https://github.com/schoolofai/obeya/releases).

## Quick Start

```bash
# Initialize a board in your project
ob init

# Register yourself
ob user add "Your Name" --type human

# Create work items
ob create epic "Auth System"
ob create story "OAuth flow" --parent <epic-id>
ob create task "Add token refresh" --parent <story-id>

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
ob init --shared teamboard       # Create a shared board
ob link teamboard                # Link current project
ob link teamboard --migrate      # Link and move existing tasks
ob boards                        # List all shared boards
ob unlink                        # Disconnect from shared board
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

Obeya ships with a Claude Code plugin that gives your AI agent `/ob` slash commands for task management.

See [obeya-plugin/README.md](obeya-plugin/README.md) for installation and usage.

## License

MIT
