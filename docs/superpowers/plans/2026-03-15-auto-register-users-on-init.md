# Auto-Register Users on Init — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-register both agent and human users during `ob init`, make `--agent` mandatory on all paths, and make `AddUser` idempotent.

**Architecture:** Three changes — engine `AddUser` gets duplicate detection + `(bool, error)` return, `cmd/init.go` gets `registerInitUsers` helper called from both init paths, shared-only path requires `--agent`.

**Tech Stack:** Go, Cobra CLI

**Spec:** `docs/superpowers/specs/2026-03-15-auto-register-users-on-init-design.md`

---

## Task 1: Make AddUser idempotent with (bool, error) return

**Files:**
- Modify: `internal/engine/engine.go:221-236` (AddUser method)
- Modify: `internal/engine/engine_test.go`
- Modify: `cmd/user.go:69` (caller of AddUser)

- [ ] **Step 1: Write failing test for duplicate detection**

```go
func TestAddUser_DuplicateIsIdempotent(t *testing.T) {
	eng, _ := setupEngine(t)
	created1, err := eng.AddUser("DupUser", "human", "local")
	if err != nil {
		t.Fatalf("first AddUser failed: %v", err)
	}
	if !created1 {
		t.Error("expected created=true for first add")
	}

	created2, err := eng.AddUser("DupUser", "human", "local")
	if err != nil {
		t.Fatalf("second AddUser failed: %v", err)
	}
	if created2 {
		t.Error("expected created=false for duplicate")
	}

	// Case-insensitive duplicate
	created3, err := eng.AddUser("dupuser", "human", "local")
	if err != nil {
		t.Fatalf("case-insensitive AddUser failed: %v", err)
	}
	if created3 {
		t.Error("expected created=false for case-insensitive duplicate")
	}
}
```

- [ ] **Step 2: Change AddUser signature to (bool, error)**

In `internal/engine/engine.go`, replace `AddUser`:

```go
func (e *Engine) AddUser(name, identityType, provider string) (bool, error) {
	if err := domain.IdentityType(identityType).Validate(); err != nil {
		return false, err
	}

	created := false
	err := e.store.Transaction(func(board *domain.Board) error {
		for _, u := range board.Users {
			if strings.EqualFold(u.Name, name) {
				return nil // already exists, skip
			}
		}
		identity := &domain.Identity{
			ID:       domain.GenerateID(),
			Name:     name,
			Type:     domain.IdentityType(identityType),
			Provider: provider,
		}
		board.Users[identity.ID] = identity
		created = true
		return nil
	})
	return created, err
}
```

Add `"strings"` to engine.go imports if not present.

- [ ] **Step 3: Update all AddUser callers**

In `cmd/user.go:69`, update:
```go
created, err := eng.AddUser(args[0], flagUserType, flagUserProvider)
```
Then change the success message:
```go
if created {
    fmt.Printf("User %q added\n", args[0])
} else {
    fmt.Printf("User %q already exists\n", args[0])
}
```

In `internal/engine/engine_test.go`, update all `eng.AddUser(...)` calls — they currently ignore the return value with `_ = eng.AddUser(...)` or use `err` only. Change to `_, err :=` or `_, _ =` as appropriate. There are ~3-4 calls in `setupEngine` and test functions.

In `test/integration_test.go`, same pattern — update `eng.AddUser(...)` callers.

- [ ] **Step 4: Run tests**

Run: `./scripts/test.sh`
Expected: ALL pass

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: make AddUser idempotent with duplicate detection and (bool, error) return"
```

---

## Task 2: Add registerInitUsers, agentDisplayName, resolveHumanName to init.go

**Files:**
- Modify: `cmd/init.go`

- [ ] **Step 1: Add helper functions**

Add to `cmd/init.go` (after `resolveInitRoot`):

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
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name, nil
		}
	}
	if u, err := user.Current(); err == nil && u.Name != "" {
		return u.Name, nil
	}
	if u, err := user.Current(); err == nil {
		return u.Username, nil
	}
	return "", fmt.Errorf("unable to determine user name: git config user.name is empty and os/user.Current() failed")
}

func registerInitUsers(s store.Store, agentProvider string) error {
	eng := engine.New(s)

	agentName, err := agentDisplayName(agentProvider)
	if err != nil {
		return err
	}
	created, err := eng.AddUser(agentName, "agent", agentProvider)
	if err != nil {
		return fmt.Errorf("failed to register agent user: %w", err)
	}
	if created {
		fmt.Printf("  Registered agent: %s\n", agentName)
	}

	humanName, err := resolveHumanName()
	if err != nil {
		return err
	}
	created, err = eng.AddUser(humanName, "human", "local")
	if err != nil {
		return fmt.Errorf("failed to register human user: %w", err)
	}
	if created {
		fmt.Printf("  Registered human: %s\n", humanName)
	}

	board, _ := eng.ListBoard()
	fmt.Println("\nUsers:")
	for _, u := range board.Users {
		fmt.Printf("  %-12s (%s, %s)\n", u.Name, u.Type, u.Provider)
	}
	return nil
}
```

Add imports: `"os/exec"`, `"os/user"`, `"github.com/niladribose/obeya/internal/engine"`.

- [ ] **Step 2: Wire into main init path**

In the main init path (after board creation around line 89, before agent setup), add:

```go
if err := registerInitUsers(s, initAgent); err != nil {
    return err
}
```

- [ ] **Step 3: Wire into shared+agent path**

In `initSharedBoardWithAgent` (around line 132, after getting `boardDir`), add:

```go
sharedStore := store.NewJSONStore(boardDir)
if err := registerInitUsers(sharedStore, agentName); err != nil {
    return err
}
```

- [ ] **Step 4: Make --agent mandatory on shared path**

Replace lines 48-51 (the shared-only path):

```go
// Shared board path (no agent) — agent is always required
if initShared != "" {
    return fmt.Errorf("--agent is required. Supported: %s", strings.Join(agent.SupportedNames(), ", "))
}
```

This makes `ob init --shared myboard` without `--agent` return an error.

- [ ] **Step 5: Run tests, fix any broken init tests**

Run: `./scripts/test.sh`
Some init tests may need updating for the new `--agent` requirement on shared paths.

- [ ] **Step 6: Commit**

```bash
git commit -m "feat: auto-register agent and human users during ob init"
```

---

## Task 3: CLI and integration tests

**Files:**
- Modify: `cmd/init_test.go`
- Modify: `test/integration_test.go` (if needed)

- [ ] **Step 1: Test init registers both users**

```go
func TestInit_RegistersBothUsers(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %s\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "Claude") {
		t.Errorf("expected agent 'Claude' in output, got:\n%s", output)
	}
	// Human name varies by system, just check "human" type appears
	if !strings.Contains(output, "human") {
		t.Errorf("expected human user in output, got:\n%s", output)
	}

	// Verify users exist via user list
	cmd = exec.Command(bin, "user", "list")
	cmd.Dir = dir
	out, _ = cmd.CombinedOutput()
	if !strings.Contains(string(out), "Claude") {
		t.Errorf("expected Claude in user list, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Test init idempotent (no duplicates on re-run)**

```go
func TestInit_IdempotentUsers(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	// First init
	cmd := exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	cmd.CombinedOutput()

	// Second init
	cmd = exec.Command(bin, "init", "--agent", "claude-code", "--skip-plugin")
	cmd.Dir = dir
	cmd.CombinedOutput()

	// Count users — should be exactly 2
	cmd = exec.Command(bin, "user", "list", "--format", "json")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	// Count "id" occurrences as proxy for user count
	count := strings.Count(string(out), `"id"`)
	if count != 2 {
		t.Errorf("expected 2 users after double init, got %d:\n%s", count, out)
	}
}
```

- [ ] **Step 3: Test shared without agent fails**

```go
func TestInit_SharedRequiresAgent(t *testing.T) {
	bin := buildOb(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cmd := exec.Command(bin, "init", "--shared", "test-board-"+t.Name())
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when --shared without --agent")
	}
	if !strings.Contains(string(out), "--agent is required") {
		t.Errorf("expected '--agent is required' error, got:\n%s", out)
	}
}
```

- [ ] **Step 4: Run full test suite**

Run: `./scripts/test.sh`
Expected: ALL pass

- [ ] **Step 5: Commit**

```bash
git commit -m "test: add init user registration and idempotency tests"
```
