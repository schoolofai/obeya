# ob uninstall — Design Spec

**Date:** 2026-03-12

## Summary

Add an `ob uninstall` command that cleanly removes all Obeya agent integrations (Claude Code plugin, skill files, CLAUDE.md injections) while preserving user board data. The command adapts based on whether it's run inside a project directory.

## Command Interface

```
ob uninstall
```

No flags, no arguments. Behavior adapts to context.

## Prerequisites

The command requires the `claude` CLI to be in PATH. If not found, it errors immediately:

```
Error: claude CLI not found. Required to uninstall plugin.
Install it or remove the plugin manually:
  claude plugin uninstall obeya@obeya-local
  claude plugin marketplace remove obeya-local
```

## Context Detection

On startup, the command checks:

1. Is the current directory inside a git repository?
2. Does that repo's `CLAUDE.md` contain `<!-- obeya:start -->` markers?

If both are true, project-level cleanup is included. Otherwise, only global cleanup runs.

## Dry-Run Preview

Before executing, the command displays exactly what will be changed:

```
ob uninstall — the following changes will be made:

  CLAUDE CODE PLUGIN
  ├── Uninstall plugin: obeya@obeya-local
  └── Remove marketplace: obeya-local

  SKILL FILES
  ├── Remove ~/.claude/obeya.md
  ├── Remove ~/.opencode/obeya.md
  └── Remove ~/.codex/obeya.md

  CLAUDE.md
  ├── Strip obeya section from ~/.claude/CLAUDE.md
  └── Strip obeya section from ./CLAUDE.md

  PRESERVED (not touched)
  ├── .obeya/          (board data)
  ├── ~/.obeya/        (shared boards)
  └── .obeya-link      (board links)

Proceed? [y/N]
```

**Display rules:**
- Skill file lines only appear if the file exists on disk
- Project CLAUDE.md line only appears when run inside a project with obeya markers
- "PRESERVED" section always shown so user knows their data is safe

## Execution Steps

Each step runs sequentially. If any step fails, execution stops immediately with an error (no fallbacks).

### Step 1: Plugin Removal (via claude CLI)

```
claude plugin uninstall obeya@obeya-local
claude plugin marketplace remove obeya-local
```

This removes the plugin registration, cache directory, and hook registrations — all managed by Claude Code.

### Step 2: Skill File Removal

Remove agent skill files if they exist:

- `~/.claude/obeya.md`
- `~/.opencode/obeya.md`
- `~/.codex/obeya.md`

### Step 3a: Global CLAUDE.md Cleanup

Strip the obeya section from `~/.claude/CLAUDE.md`:

- Remove everything between `<!-- obeya:start -->` and `<!-- obeya:end -->` markers, inclusive
- Clean up surrounding blank lines
- If no markers found, skip silently (idempotent)

### Step 3b: Project CLAUDE.md Cleanup (conditional)

Only runs when inside a project with obeya markers in its CLAUDE.md:

- Same stripping logic as 3a, applied to `<git-root>/CLAUDE.md`
- If no markers found, skip silently (idempotent)

### Step 4: Post-Uninstall Banner

```
┌───────────────────────────────────────────────────────┐
│                                                       │
│  Obeya agent integrations removed successfully.       │
│                                                       │
│  To fully remove ob from your system:                 │
│                                                       │
│    brew uninstall obeya                               │
│                                                       │
│  Board data in .obeya/ directories was preserved.     │
│  Delete them manually if no longer needed.            │
│                                                       │
└───────────────────────────────────────────────────────┘
```

## stripObeyaSection Logic

```
func stripObeyaSection(path string):
    content = readFile(path)
    cleaned = regex.replace(content,
        `(?ms)\n*<!-- obeya:start.*?-->.*?<!-- obeya:end -->\n*`,
        "\n")
    writeFile(path, cleaned)
```

The regex matches the versioned marker (`<!-- obeya:start --> v5` or any future version) through the end marker.

## Error Handling

| Failure | Behavior |
|---------|----------|
| `claude` CLI not in PATH | Error with message, exit immediately |
| `claude plugin uninstall` fails | Print stderr output, exit immediately |
| `claude plugin marketplace remove` fails | Print stderr output, exit immediately |
| File removal fails (permission, etc.) | Print path and OS error, exit immediately |
| CLAUDE.md has no obeya markers | Skip silently (idempotent) |
| User answers N to confirmation | Exit 0, no changes made |

## What Is NOT Removed

The following are explicitly preserved — they are user data:

- `.obeya/` directories (local board data)
- `~/.obeya/` directory (shared boards)
- `.obeya-link` files (board links)
- `.obeya-local-backup/` directories (migration backups)

## Idempotent Behavior

Running `ob uninstall` multiple times is safe:

- Plugin already uninstalled → `claude plugin uninstall` reports no-op or error (handle gracefully)
- Skill files already deleted → skip
- CLAUDE.md already clean → skip
- First run from home dir (global only), second run from project dir → only project CLAUDE.md cleanup runs

## File Placement

The command implementation goes in:

- `cmd/uninstall.go` — cobra command definition, confirmation prompt, banner output
- `internal/agent/claudecode.go` — add `UninstallPlugin()` method alongside existing `InstallPlugin()`
- `internal/claudemd/` — reuse or extend existing CLAUDE.md manipulation logic for stripping sections
