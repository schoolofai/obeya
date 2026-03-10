# Design: `ob show --verbose`

**Date:** 2026-03-10

## Summary

Add a `--verbose` / `-v` boolean flag to `ob show`. When set, children are displayed with richer detail (priority, assignee, blocked-by, description snippet).

## Default children output (unchanged)

```
Children:
  #3 [task] in-progress — Fix login bug
  #4 [task] backlog — Add tests
```

## Verbose children output

```
Children:
  #3 [task] in-progress — Fix login bug
      Priority: high | Assignee: niladri | Blocked by: #5
      Desc: First 80 chars of description text...

  #4 [task] backlog — Add tests
      Priority: medium | Assignee: —
      Desc: Write unit tests for the auth module...
```

## Rules

- Description truncated to 80 chars with `...` if longer
- Assignee shows `—` if unset
- Blocked-by line only appears if item has blockers
- Priority always shown
- No change to `--format json` output (already includes everything)
- Flag is `--verbose` with `-v` shorthand, default `false`

## Files to change

1. `cmd/show.go` — add `--verbose` flag, pass to `printItemChildren`
2. `obeya-plugin/skills/ob-show/SKILL.md` — document the flag
