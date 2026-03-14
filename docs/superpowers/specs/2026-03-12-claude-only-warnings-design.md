# Claude-Only Agent Support Warnings

**Date:** 2026-03-12
**Status:** Approved

## Problem

Obeya's architecture is extensible for multiple coding agents, but only Claude Code is currently implemented. Provider placeholders (opencode, codex) in CLI output and help text mislead users into thinking multi-agent support works. Users of other coding agents discover the limitation only after hitting cryptic errors.

## Design

Explicit, blocking warnings at every user-facing surface. Tone: hard fails with clear messaging. No ambiguity.

### Changes

| Surface | File(s) | Change |
|---|---|---|
| `ob init --agent X` error | `internal/agent/agent.go` | Expanded error message with "only Claude Code supported" |
| `ob init --help` | `cmd/init.go` | `--agent` flag description notes Claude-only |
| `ob user add` validation | `cmd/user.go` | Hard fail on unsupported providers with clear message |
| `ob user add --help` | `cmd/user.go` | Provider flag description notes Claude-only |
| `ob skill list` output | `cmd/skill.go` | Add `Supported` field, show "not yet supported" status |
| `ob skill install` validation | `cmd/skill.go` | Hard fail on unsupported providers |
| Provider struct | `cmd/skill.go` | Add `Supported bool` field to provider type |
| README banner | `README.md` | Prominent notice at top |

### Error Messages

**`ob init --agent <unsupported>`:**
```
Error: unsupported agent 'X'.

Only Claude Code is currently supported as a first-class agent.
Supported agents: claude-code

Other agents (cursor, windsurf, copilot, etc.) are not yet supported.
```

**`ob user add --provider <unsupported>`:**
```
Error: unsupported provider 'X'.

Only Claude Code is currently supported as an agent provider.
Supported providers: local (human), claude-code (agent)

Other agent providers are planned but not yet available.
```

**`ob skill install --provider <unsupported>`:**
```
Error: provider 'X' is not yet supported.

Only 'claude-code' is currently supported. Run: ob skill install --provider claude-code
```

### Not Changed (by design)

- Plugin skills — already Claude-specific by nature
- `internal/domain/types.go` — identity types stay extensible
- `claudecode.go` — already Claude-only
- Test files — updated to match new error messages
