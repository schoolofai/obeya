# ob uninstall — Design Spec

**Date:** 2026-03-12

## Summary

Add an `ob uninstall` command that cleanly removes all Obeya agent integrations (Claude Code plugin, skill files, CLAUDE.md injections) while preserving user board data. The command adapts based on whether it's run inside a project directory.

## Command Interface

```
ob uninstall
```

No flags, no arguments. Behavior adapts to context.

A `--yes` / `-y` flag is supported to skip the confirmation prompt (for CI/scripting, consistent with `scripts/release.sh`).

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

Before running removal commands, check if the plugin is currently installed (mirror the `alreadyInstalled()` check in `internal/agent/claudecode.go`). Skip each sub-step if already absent.

```
if pluginInstalled("obeya@obeya-local"):
    claude plugin uninstall obeya@obeya-local

if marketplaceRegistered("obeya-local"):
    claude plugin marketplace remove obeya-local
```

Detection uses `~/.claude/plugins/installed_plugins.json` — check for the plugin ref string. For marketplace, check the marketplace config. This avoids the contradiction between fail-fast error handling and idempotent re-runs.

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

Use a string-index approach to mirror `AppendClaudeMDAt` in `internal/agent/claudecode.go`, reusing the same `ObeyaSectionStart` and `ObeyaSectionEnd` constants. This guarantees symmetry between install and uninstall.

```
func stripObeyaSection(path string):
    if !fileExists(path):
        return  // file doesn't exist, skip silently

    content = readFile(path)
    startIdx = strings.Index(content, ObeyaSectionStart)  // "<!-- obeya:start -->"
    if startIdx == -1:
        return  // no markers, skip silently (idempotent)

    endIdx = strings.Index(content, ObeyaSectionEnd)       // "<!-- obeya:end -->"
    if endIdx == -1:
        throw "found obeya start marker but no end marker in " + path

    // Include the end marker itself
    endIdx = endIdx + len(ObeyaSectionEnd)

    // Expand to consume surrounding blank lines
    for startIdx > 0 && content[startIdx-1] == '\n':
        startIdx--
    for endIdx < len(content) && content[endIdx] == '\n':
        endIdx++

    cleaned = content[:startIdx] + "\n" + content[endIdx:]

    // If file is now empty/whitespace-only, delete it
    if strings.TrimSpace(cleaned) == "":
        removeFile(path)
        return

    writeFile(path, cleaned)
```

This handles:
- The actual marker format: `<!-- obeya:start --> v5` (version string after closing `-->`)
- Surrounding blank line cleanup consistent with `AppendClaudeMDAt`
- Missing file (skip silently)
- File becomes empty after stripping (delete the file)
- Mismatched markers (start without end → error, fail fast)

## Error Handling

| Failure | Behavior |
|---------|----------|
| `claude` CLI not in PATH | Error with message, exit immediately |
| `claude plugin uninstall` fails (plugin present) | Print stderr output, exit immediately |
| `claude plugin marketplace remove` fails (marketplace present) | Print stderr output, exit immediately |
| Plugin/marketplace already absent | Skip silently (idempotent) |
| CLAUDE.md file does not exist | Skip silently |
| CLAUDE.md has start marker but no end marker | Error with path, exit immediately |
| CLAUDE.md becomes empty after stripping | Delete the file |
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

- Plugin already uninstalled → detected via `installed_plugins.json` check, skip
- Marketplace already removed → detected via marketplace config check, skip
- Skill files already deleted → `fileExists` check, skip
- CLAUDE.md already clean → no markers found, skip
- CLAUDE.md doesn't exist → skip
- First run from home dir (global only), second run from project dir → only project CLAUDE.md cleanup runs

## File Placement

The command implementation goes in:

- `cmd/uninstall.go` — cobra command definition, confirmation prompt, banner output
- `internal/agent/claudecode.go` — add `UninstallPlugin()` method alongside existing `InstallPlugin()`, add `StripClaudeMDAt()` alongside existing `AppendClaudeMDAt()`, reuse `ObeyaSectionStart`/`ObeyaSectionEnd` constants

Skill file paths should be sourced from `getProviders()` in `cmd/skill.go` (or extracted to a shared location) rather than hardcoded, to stay in sync with the install paths.
