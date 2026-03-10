# Design: Git-root anchored board discovery

**Date**: 2026-03-09
**Status**: Approved

## Problem

Running `ob` from a subdirectory fails because the store only looks for `.obeya/` in `cwd`. It should walk up to find the project root, like `.git` discovery. One code project (git repo) = one board.

## Discovery algorithm

```
findProjectRoot(startDir):
    dir = startDir
    while dir != "/":
        if exists(dir/.obeya/board.json):
            return dir
        dir = parent(dir)

    dir = startDir
    while dir != "/":
        if exists(dir/.git):
            return dir
        dir = parent(dir)

    error("no git repository found — use 'ob init --root <path>' to specify a board location")
```

Two-pass walk: first for `.obeya/`, then for `.git/`. This way if someone manually placed `.obeya/` in a subdirectory (unusual but possible), it still works.

## `ob init` behavior

- **Default**: resolves git root, creates `.obeya/` there
- **`--root <path>` flag**: creates `.obeya/` at the specified path (for non-git projects)
- **No git root + no `--root`**: error with message suggesting the flag

## Changes

| File | Change |
|------|--------|
| `internal/store/root.go` (new) | `FindProjectRoot(startDir) (string, error)` — two-pass walk |
| `internal/store/root_test.go` (new) | Tests: git root, `.obeya` found, neither found, nested |
| `cmd/helpers.go` | `getStore()`/`getEngine()` use `FindProjectRoot` instead of `os.Getwd()` |
| `cmd/init.go` | Add `--root` flag; default resolves git root for placement |

## Edge cases

- **No git, no `.obeya/`**: hard error with `--root` suggestion
- **`.obeya/` exists without `.git/`**: works (first pass finds it)
- **Nested git repos**: nearest `.git` wins
- **`--root` with existing board**: existing "already initialized" error still applies

## What stays the same

- `JSONStore` interface unchanged — it still takes a `rootDir` string
- Board file format unchanged
- All commands work identically once the root is resolved
