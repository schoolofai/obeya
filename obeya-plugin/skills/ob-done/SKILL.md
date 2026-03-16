---
description: Mark the current in-progress task as done with review context. Use when work is completed on a task, after tests pass, after a subagent reports DONE, or whenever a workflow step completes. Agents MUST use 'ob done' (not 'ob move <id> done') to include review context for human review.
disable-model-invocation: false
user-invocable: true
---

# Complete Current Task

Mark the current in-progress task as done and attach review context for human review.

## Steps

1. Run `ob list --status in-progress --format json` to find in-progress items
2. Filter to items assigned to the current user (determine identity via `--as` flag or `ob user list --format json`)
3. If multiple in-progress tasks: show them and ask the user which to complete
4. If `$ARGUMENTS` is provided, use it as the task ID
5. Gather review context:
   - **Confidence** (0-100): How confident are you the work is correct? Consider test coverage, edge cases, and risk.
   - **Purpose**: One-line summary of what changed and why
   - **Files changed**: Run `git diff --stat HEAD~1` to get changed files with line counts
   - **Tests**: List tests written/run and their pass/fail status
   - **Reproduce**: Commands to verify the work (e.g., `go test ./auth/ -run TestJWT`)
   - **Reasoning**: Why you chose this approach over alternatives
6. Complete with review context using `ob done`:

```bash
# Option A: Flags (simple cases)
ob done <id> --confidence 85 --purpose "Replace cookie sessions with JWT" \
  --files "auth/middleware.go:+82-41,auth/jwt.go:+120-0" \
  --tests "TestJWT:pass,TestRefresh:pass" \
  --reproduce "go test ./auth/ -run TestJWT" \
  --reasoning "JWT chosen for debuggability and stateless scaling"

# Option B: JSON stdin (complex cases with diffs)
cat <<'EOF' | ob done <id> --confidence 85 --context-stdin
{
  "purpose": "Replace cookie sessions with JWT",
  "files_changed": [
    {"path": "auth/middleware.go", "added": 82, "removed": 41, "diff": "...unified diff..."}
  ],
  "tests_written": [{"name": "TestJWT", "passed": true}],
  "proof": [{"check": "go vet", "status": "pass"}],
  "reasoning": "JWT chosen for debuggability",
  "reproduce": ["go test ./auth/ -run TestJWT"]
}
EOF
```

7. After completing, run `ob list --status todo --format json` to show the next available tasks
8. If all children of a parent are now done, suggest moving the parent to done too

## Confidence Guidelines

| Range | Meaning |
|-------|---------|
| 0-50  | Low confidence. Missing tests, known edge cases, risky changes. Needs careful review. |
| 51-75 | Medium confidence. Core logic tested but gaps remain. Standard review. |
| 76-100| High confidence. Well-tested, simple changes, or refactoring with full coverage. Quick review. |

## Important

- **NEVER** use `ob move <id> done` for agent work. Always use `ob done` to include review context.
- The review context enables human sponsors to efficiently review agent-completed work in the TUI.
- Items completed with `ob done` appear in the TUI's review queue column, sorted by confidence (lowest first).
