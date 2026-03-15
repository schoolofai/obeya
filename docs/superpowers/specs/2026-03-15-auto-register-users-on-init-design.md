# Auto-Register Users During `ob init`

**Date:** 2026-03-15
**Status:** Draft

## Problem

After `ob init --agent claude-code`, the board has zero registered users. With mandatory `--assign` on all `ob create` commands (v0.2.0), both agents and humans hit a dead end: they can't create tasks because there's no one to assign to, and `ob user list` returns nothing.

## Design

### What happens during `ob init`

The `--agent` flag is mandatory on ALL init paths (including shared boards). This ensures every board always has both users registered from the start.

User registration is extracted into a shared helper `registerInitUsers(eng, agentProvider)` and called from all init paths:

1. **Main path** (`ob init --agent claude-code`): registers both agent + human
2. **Shared + agent** (`ob init --shared myboard --agent claude-code`): registers both agent + human on the shared board

The previous shared-only path (`ob init --shared myboard` without `--agent`) now returns an error requiring `--agent`.

Two users are always registered:

1. **The agent** — name and provider derived from the `--agent` flag
2. **The human** — name resolved from the environment

### Agent name resolution

Static mapping from provider name:

```
claude-code → name: "Claude", type: agent, provider: claude-code
```

Future providers would add entries to this map. The name is short and recognizable for use with `--assign`.

### Human name resolution

Resolution chain (first non-empty wins):

```
1. git config user.name     → best quality, user-chosen
2. os/user.Current().Name   → OS display name, always available
3. os/user.Current().Username → OS login, last resort
```

All three are checked at init time. In non-git projects, source 1 is skipped.

### Init output

After registration, init prints the registered users so the agent knows what names to use:

```
Board "obeya" initialized in /path/.obeya/
Columns: backlog, todo, in-progress, review, done

Users registered:
  Claude    (agent, claude-code)
  Niladri   (human, local)

Ready to use: ob create task "title" --assign Claude -d "description"
```

### Idempotency

If `ob init` is run on an already-initialized board, it currently prints "Board already initialized" and continues to agent setup. The user registration should also be idempotent: skip registration if users with the same name already exist. No error, no duplicate.

### Pseudocode

```go
// Shared helper called from all three init paths.
// agentProvider is "" when init is shared-only (no agent).
func registerInitUsers(eng *engine.Engine, agentProvider string) error {
    if agentProvider != "" {
        agentName, err := agentDisplayName(agentProvider)
        if err != nil {
            return err
        }
        if err := eng.AddUser(agentName, "agent", agentProvider); err != nil {
            return fmt.Errorf("failed to register agent user: %w", err)
        }
    }

    humanName, err := resolveHumanName()
    if err != nil {
        return fmt.Errorf("failed to determine user name: %w", err)
    }
    if err := eng.AddUser(humanName, "human", "local"); err != nil {
        return fmt.Errorf("failed to register human user: %w", err)
    }

    // Print registered users
    board, _ := eng.ListBoard()
    fmt.Println("\nUsers registered:")
    for _, u := range board.Users {
        fmt.Printf("  %-12s (%s, %s)\n", u.Name, u.Type, u.Provider)
    }
    return nil
}
```

```go
func agentDisplayName(provider string) (string, error) {
    names := map[string]string{
        "claude-code": "Claude",
    }
    if n, ok := names[provider]; ok {
        return n, nil
    }
    return "", fmt.Errorf("no display name configured for agent provider %q", provider)
}

func resolveHumanName() (string, error) {
    // 1. git config user.name
    if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
        if name := strings.TrimSpace(string(out)); name != "" {
            return name, nil
        }
    }
    // 2. OS display name
    if u, err := user.Current(); err == nil && u.Name != "" {
        return u.Name, nil
    }
    // 3. OS username
    if u, err := user.Current(); err == nil {
        return u.Username, nil
    }
    return "", fmt.Errorf("unable to determine user name: git config user.name is empty and os/user.Current() failed")
}
```

### Duplicate detection

`engine.AddUser` currently doesn't check for name collisions — it generates a new UUID every time. We need a check:

```go
func (e *Engine) AddUser(name, identityType, provider string) error {
    // Preserve existing type validation
    if err := domain.IdentityType(identityType).Validate(); err != nil {
        return err
    }

    return e.store.Transaction(func(board *domain.Board) error {
        // Skip if user with same name already exists (idempotent)
        for _, u := range board.Users {
            if strings.EqualFold(u.Name, name) {
                return nil
            }
        }
        identity := &domain.Identity{
            ID:       domain.GenerateID(),
            Name:     name,
            Type:     domain.IdentityType(identityType),
            Provider: provider,
        }
        board.Users[identity.ID] = identity
        return nil
    })
}
```

This is a behavior change to `AddUser` — currently it allows duplicates. The new behavior silently skips duplicates (returns nil), which is correct for init idempotency. The `IdentityType.Validate()` call is preserved from the existing implementation.

**CLI output change in `cmd/user.go`:** The `runUserAdd` function currently always prints `User "X" added`. After this change, it should check whether the user was actually created or skipped. The engine can return a sentinel (e.g., `ErrUserExists`) or the caller can check user count before/after. Simplest: `AddUser` returns a boolean `created` alongside the error, or the CLI checks `ob user list` before adding. The spec recommends adding a `bool` return: `AddUser(...) (bool, error)` where `false` means "already existed, skipped."

## Files Changed

| File | Change |
|------|--------|
| `cmd/init.go` | Extract `registerInitUsers` helper, call from all 3 init paths, add `agentDisplayName` and `resolveHumanName` |
| `internal/engine/engine.go` | Add duplicate name check to `AddUser`, change return to `(bool, error)` |
| `internal/engine/engine_test.go` | Test duplicate user registration is idempotent, update callers for new return signature |
| `cmd/init_test.go` | Test that init registers both users, test shared board paths |
| `cmd/user.go` | Update `runUserAdd` to print "already exists" vs "added" based on `AddUser` return |

## Testing

- `ob init --agent claude-code` → board has 2 users (agent + human)
- `ob init --shared myboard --agent claude-code` → shared board has 2 users
- `ob init --shared myboard` (no agent) → error: `--agent` is required
- `ob init` again on same board → no duplicate users, idempotent
- `ob user add "Claude"` after init → prints "already exists", no duplicate
- `ob user add "NewUser"` → prints "added", creates user
- `ob user list` after init → shows both users with correct types
- Non-git directory → human name falls back to OS display name
- All 3 name resolution sources fail → hard error, no fallback
- Unknown agent provider → hard error from `agentDisplayName`
- Init output includes "Users registered" section

## Migration

Existing boards are unaffected — they already have manually registered users. The change only affects new `ob init` calls.
