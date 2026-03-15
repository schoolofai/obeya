# Auto-Register Users During `ob init`

**Date:** 2026-03-15
**Status:** Draft

## Problem

After `ob init --agent claude-code`, the board has zero registered users. With mandatory `--assign` on all `ob create` commands (v0.2.0), both agents and humans hit a dead end: they can't create tasks because there's no one to assign to, and `ob user list` returns nothing.

## Design

### What happens during `ob init --agent claude-code`

After creating the board and before delegating to agent-specific setup, init auto-registers two users:

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
// In cmd/init.go, after s.InitBoard() succeeds and before agent setup

eng := engine.New(s)

// Register agent user
agentName := agentDisplayName(initAgent) // "claude-code" → "Claude"
err = eng.AddUser(agentName, "agent", initAgent)
if err != nil && !isAlreadyExistsError(err) {
    return fmt.Errorf("failed to register agent user: %w", err)
}

// Register human user
humanName := resolveHumanName()  // git config → OS name → username
err = eng.AddUser(humanName, "human", "local")
if err != nil && !isAlreadyExistsError(err) {
    return fmt.Errorf("failed to register human user: %w", err)
}

// Print registered users
fmt.Println("\nUsers registered:")
board, _ := eng.ListBoard()
for _, u := range board.Users {
    fmt.Printf("  %-12s (%s, %s)\n", u.Name, u.Type, u.Provider)
}
fmt.Printf("\nReady to use: ob create task \"title\" --assign %s -d \"description\"\n", agentName)
```

```go
func agentDisplayName(provider string) string {
    names := map[string]string{
        "claude-code": "Claude",
    }
    if n, ok := names[provider]; ok {
        return n
    }
    return provider // fallback: use provider name as display name
}

func resolveHumanName() string {
    // 1. git config user.name
    if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
        if name := strings.TrimSpace(string(out)); name != "" {
            return name
        }
    }
    // 2. OS display name
    if u, err := user.Current(); err == nil && u.Name != "" {
        return u.Name
    }
    // 3. OS username
    if u, err := user.Current(); err == nil {
        return u.Username
    }
    return "human"
}
```

### Duplicate detection

`engine.AddUser` currently doesn't check for name collisions — it generates a new UUID every time. We need a check:

```go
func (e *Engine) AddUser(name, identityType, provider string) error {
    return e.store.Transaction(func(board *domain.Board) error {
        // Skip if user with same name already exists
        for _, u := range board.Users {
            if strings.EqualFold(u.Name, name) {
                return nil // idempotent — already registered
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

This is a behavior change to `AddUser` — currently it allows duplicates. The new behavior silently skips duplicates, which is correct for init idempotency and also prevents accidental duplicate registrations via `ob user add`.

## Files Changed

| File | Change |
|------|--------|
| `cmd/init.go` | Add user registration after board creation, add `agentDisplayName` and `resolveHumanName` helpers |
| `internal/engine/engine.go` | Add duplicate name check to `AddUser` |
| `internal/engine/engine_test.go` | Test duplicate user registration is idempotent |
| `cmd/init_test.go` | Test that init registers both users |

## Testing

- `ob init --agent claude-code` → board has 2 users (agent + human)
- `ob init` again on same board → no duplicate users
- `ob user add "Claude"` after init → no duplicate created
- `ob user list` after init → shows both users with correct types
- Non-git directory → human name falls back to OS display name
- Init output includes "Users registered" section

## Migration

Existing boards are unaffected — they already have manually registered users. The change only affects new `ob init` calls.
