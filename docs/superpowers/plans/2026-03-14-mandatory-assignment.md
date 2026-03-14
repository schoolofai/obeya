# Mandatory Assignment & Identity Simplification — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make assignee a mandatory first-class field on all board items, drop `OB_USER`, and guard all state-change commands against unassigned items.

**Architecture:** Two-layer change — engine guards (assignee validation inside transactions) + cmd-layer validation (mandatory `--assign` on create). TUI gets a cosmetic `@unassigned` badge. Skills get updated instructions.

**Tech Stack:** Go, Cobra CLI, Bubble Tea TUI, lipgloss styling, teatest

**Spec:** `docs/superpowers/specs/2026-03-14-mandatory-assignment-design.md`

---

## Chunk 1: Engine — Assignee Guard & Create Validation

### Task 1: Add `checkAssignee` helper to engine

**Files:**
- Modify: `internal/engine/engine.go:262` (helpers section)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test for checkAssignee**

```go
// In engine_test.go — add at bottom of file
// Add "strings" and "github.com/niladribose/obeya/internal/domain" to imports

func TestCheckAssignee_Unassigned(t *testing.T) {
	item := &domain.Item{DisplayNum: 5, Assignee: ""}
	err := engine.CheckAssignee(item)
	if err == nil {
		t.Fatal("expected error for unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ob assign 5") {
		t.Errorf("expected fix instructions in error, got: %v", err)
	}
}

func TestCheckAssignee_Assigned(t *testing.T) {
	item := &domain.Item{DisplayNum: 5, Assignee: "user123"}
	err := engine.CheckAssignee(item)
	if err != nil {
		t.Fatalf("expected no error for assigned item, got: %v", err)
	}
}
```

Also add a `createUnassignedItem` test helper that bypasses engine validation by writing directly to the board via `store.Transaction`. This simulates legacy items that exist without assignees — needed by Tasks 2-3 to test guards:

```go
// createUnassignedItem inserts an item directly into the board without
// going through engine.CreateItem, simulating legacy data with no assignee.
func createUnassignedItem(t *testing.T, s store.Store, title string) *domain.Item {
	t.Helper()
	var created *domain.Item
	err := s.Transaction(func(board *domain.Board) error {
		item := &domain.Item{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Type:        "task",
			Title:       title,
			Description: "test",
			Status:      board.Columns[0].Name,
			Priority:    "medium",
			Assignee:    "", // deliberately unassigned
		}
		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++
		created = item
		return nil
	})
	if err != nil {
		t.Fatalf("createUnassignedItem failed: %v", err)
	}
	return created
}
```

Update `setupEngine` to also return the store so tests can call `createUnassignedItem`:

```go
func setupEngine(t *testing.T) (*engine.Engine, store.Store) {
	t.Helper()
	dir := t.TempDir()
	s := store.NewJSONStore(dir)
	_ = s.InitBoard("test", nil)
	eng := engine.New(s)
	_ = eng.AddUser("testuser", "human", "local")
	return eng, s
}
```

**Important:** This changes `setupEngine`'s return signature. All existing callers must be updated from `eng := setupEngine(t)` to `eng, _ := setupEngine(t)` (or `eng, s := setupEngine(t)` when the store is needed). There are approximately 25-30 call sites in `engine_test.go`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestCheckAssignee -v`
Expected: FAIL — `CheckAssignee` not defined

- [ ] **Step 3: Implement checkAssignee**

Add to `internal/engine/engine.go` in the helpers section (after line 262):

```go
// CheckAssignee returns an error if the item has no assignee.
// Must be called inside a Transaction callback.
func CheckAssignee(item *domain.Item) error {
	if item.Assignee == "" {
		return fmt.Errorf("item #%d has no assignee. Assign it first:\n\n"+
			"  ob assign %d --to <user>\n\n"+
			"Examples:\n"+
			"  ob assign %d --to claude\n"+
			"  ob assign %d --to niladri\n\n"+
			"Run 'ob user list' to see registered users.",
			item.DisplayNum, item.DisplayNum, item.DisplayNum, item.DisplayNum)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestCheckAssignee -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: add CheckAssignee guard helper"
```

---

### Task 2: Add assignee guard to MoveItem

**Files:**
- Modify: `internal/engine/engine.go:51-70` (MoveItem method)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Uses `createUnassignedItem` helper from Task 1 to simulate legacy data:

```go
func TestMoveItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Unassigned task")

	err := eng.MoveItem(item.ID, "in-progress", "", "")
	if err == nil {
		t.Fatal("expected error moving unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' error, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestMoveItem_UnassignedFails -v`
Expected: FAIL — move succeeds without assignee check

- [ ] **Step 3: Add guard to MoveItem**

In `internal/engine/engine.go`, inside `MoveItem`'s transaction callback, after resolving the item (line 62), add:

```go
item := board.Items[id]
if err := CheckAssignee(item); err != nil {
    return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestMoveItem_Unassigned -v`
Expected: PASS

- [ ] **Step 5: Fix all existing tests for new setupEngine signature**

The `setupEngine` change from Task 1 returns `(*engine.Engine, store.Store)`. Update ALL existing callers:
- Change `eng := setupEngine(t)` to `eng, _ := setupEngine(t)` everywhere
- Also update all existing `CreateItem` calls to pass `"testuser"` as the assignee parameter instead of `""` — there are approximately 25-30 calls across the entire test file
- Also update any `MoveItem` calls that use items created without assignees

Run: `go test ./internal/engine/ -v`
Expected: ALL tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: add assignee guard to MoveItem"
```

---

### Task 3: Add assignee guard to EditItem, BlockItem, UnblockItem, DeleteItem

**Files:**
- Modify: `internal/engine/engine.go` — methods at lines 93, 117, 138, 160
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write failing tests for all four methods**

All tests use `createUnassignedItem` to simulate legacy data, so they work regardless of whether `CreateItem` enforces assignment:

```go
func TestEditItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Test")
	err := eng.EditItem(item.ID, "New title", "", "", "", "")
	if err == nil {
		t.Fatal("expected error editing unassigned item")
	}
	if !strings.Contains(err.Error(), "no assignee") {
		t.Errorf("expected 'no assignee' error, got: %v", err)
	}
}

func TestBlockItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item1, _ := eng.CreateItem("task", "Task1", "", "desc", "medium", "testuser", nil)
	item2 := createUnassignedItem(t, s, "Task2")
	err := eng.BlockItem(item2.ID, item1.ID, "", "")
	if err == nil {
		t.Fatal("expected error blocking unassigned item")
	}
}

func TestUnblockItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item1, _ := eng.CreateItem("task", "Task1", "", "desc", "medium", "testuser", nil)
	item2 := createUnassignedItem(t, s, "Task2")
	err := eng.UnblockItem(item2.ID, item1.ID, "", "")
	if err == nil {
		t.Fatal("expected error unblocking unassigned item")
	}
}

func TestDeleteItem_UnassignedFails(t *testing.T) {
	eng, s := setupEngine(t)
	item := createUnassignedItem(t, s, "Test")
	err := eng.DeleteItem(item.ID, "", "")
	if err == nil {
		t.Fatal("expected error deleting unassigned item")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run "Unassigned" -v`
Expected: 4 new FAILs

- [ ] **Step 3: Add guard to all four methods**

In `internal/engine/engine.go`, add `CheckAssignee` call inside each transaction callback, right after resolving the item:

**EditItem** (line 138): after `item := board.Items[id]` (line 145):
```go
if err := CheckAssignee(item); err != nil {
    return err
}
```

**BlockItem** (line 93): after `item := board.Items[id]` (line 104):
```go
if err := CheckAssignee(item); err != nil {
    return err
}
```

**UnblockItem** (line 117): after `item := board.Items[id]` (line 124):
```go
if err := CheckAssignee(item); err != nil {
    return err
}
```

**DeleteItem** (line 160): after `item := board.Items[id]` (line 170):
```go
if err := CheckAssignee(item); err != nil {
    return err
}
```

- [ ] **Step 4: Run tests to verify all pass**

Run: `go test ./internal/engine/ -v`
Expected: ALL pass

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: add assignee guard to EditItem, BlockItem, UnblockItem, DeleteItem"
```

---

### Task 4: Add ResolveUserID validation to CreateItem

**Files:**
- Modify: `internal/engine/engine.go:24-49` (CreateItem method)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestCreateItem_AssigneeResolved(t *testing.T) {
	eng := setupEngine(t)
	item, err := eng.CreateItem("task", "Test", "", "desc", "medium", "testuser", nil)
	if err != nil {
		t.Fatalf("expected success with valid assignee, got: %v", err)
	}
	// Assignee should be resolved to the user's ID, not the name
	if item.Assignee == "" {
		t.Error("expected assignee to be set")
	}
}

func TestCreateItem_UnknownAssigneeFails(t *testing.T) {
	eng := setupEngine(t)
	_, err := eng.CreateItem("task", "Test", "", "desc", "medium", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown assignee")
	}
	if !strings.Contains(err.Error(), "ob user list") {
		t.Errorf("expected 'ob user list' in error, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run "TestCreateItem_.*Assignee" -v`
Expected: `TestCreateItem_UnknownAssigneeFails` fails (create succeeds with any string)

- [ ] **Step 3: Add mandatory assignee + ResolveUserID to CreateItem**

In `internal/engine/engine.go`, inside `CreateItem`'s transaction callback (line 33), after resolving parent and before `buildItem`:

```go
// Mandatory assignee — enforce at engine level, not just CLI
if assignee == "" {
    return fmt.Errorf("assignee is required. Every item must have an owner.\n\n" +
        "Run 'ob user list' to see registered users.")
}
resolvedAssignee, err := board.ResolveUserID(assignee)
if err != nil {
    return fmt.Errorf("unknown assignee %q: %w\nRun 'ob user list' to see registered users", assignee, err)
}

item := buildItem(board, itemType, title, description, priority, resolvedAssignee, parentID, tags)
```

Replace the existing `buildItem` call (line 39) with the version that uses `resolvedAssignee` instead of raw `assignee`.

**Important:** This enforces mandatory assignment at the engine level — not just the CLI. Direct API callers (future SDKs, tests) cannot bypass it. The `createUnassignedItem` test helper from Task 1 bypasses this intentionally to simulate legacy data.

Also add a test for empty assignee at engine level:

```go
func TestCreateItem_EmptyAssigneeFails(t *testing.T) {
	eng, _ := setupEngine(t)
	_, err := eng.CreateItem("task", "Test", "", "desc", "medium", "", nil)
	if err == nil {
		t.Fatal("expected error for empty assignee")
	}
	if !strings.Contains(err.Error(), "assignee is required") {
		t.Errorf("expected 'assignee is required' error, got: %v", err)
	}
}
```

- [ ] **Step 4: Run tests to verify all pass**

Run: `go test ./internal/engine/ -v`
Expected: ALL pass

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: validate assignee via ResolveUserID in CreateItem"
```

---

### Task 5: Mandatory `--assign` on `ob create`

**Files:**
- Modify: `cmd/create.go:27-73` (RunE function)
- Test: manual CLI test (Cobra commands are tested via integration)

- [ ] **Step 1: Add mandatory --assign validation**

In `cmd/create.go`, at the top of the `RunE` function (after line 30, before the body-file check), add:

```go
if createAssign == "" {
    return fmt.Errorf("--assign is required. Every item must have an owner.\n\n" +
        "Examples:\n" +
        "  ob create task \"Fix bug\" --assign claude\n" +
        "  ob create epic \"Auth system\" --assign niladri\n\n" +
        "If you are an agent, assign yourself:\n" +
        "  Claude agent:  --assign claude\n" +
        "  Codex agent:   --assign codex\n" +
        "  Cursor agent:  --assign cursor\n\n" +
        "Run 'ob user list' to see registered users.")
}
```

- [ ] **Step 2: Run full test suite to find broken tests**

Run: `./scripts/test.sh`
Expected: Some engine tests that create items without assignees may now fail at the cmd level if tested end-to-end. Engine-level tests were already fixed in Task 2.

- [ ] **Step 3: Fix any broken tests**

If any tests call `ob create` without `--assign` through CLI integration, update them. The engine tests should already be fixed from Task 2's setupEngine changes.

- [ ] **Step 4: Verify manually**

```bash
./ob create task "Test no assign" -d "test"
# Expected: Error: --assign is required...

./ob create task "Test with assign" --assign testuser -d "test"
# Expected: Created task #XX...
```

- [ ] **Step 5: Commit**

```bash
git add cmd/create.go
git commit -m "feat: mandatory --assign on ob create"
```

---

## Chunk 2: Drop OB_USER & Update Metadata

### Task 6: Drop `OB_USER` from helpers and add deprecation warning

**Files:**
- Modify: `cmd/helpers.go:51-63` (getUserID function)
- Modify: `cmd/root.go:40-41` (--as flag help text)

- [ ] **Step 1: Update getUserID in helpers.go**

Replace lines 51-63 in `cmd/helpers.go`:

```go
func getUserID() string {
	if flagAs != "" {
		return flagAs
	}
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}
```

This removes the `OB_USER` env var check entirely.

- [ ] **Step 2: Add deprecation warning**

In `cmd/helpers.go`, add at the top of the file (after imports), or in `cmd/root.go`'s `init()`:

```go
func init() {
	if os.Getenv("OB_USER") != "" {
		fmt.Fprintln(os.Stderr, "Warning: OB_USER is deprecated and ignored. Use --assign for ownership, --as for audit.")
	}
}
```

If adding to `helpers.go`, make sure `"os"` is in the imports (it already is).

- [ ] **Step 3: Update --as flag help text in root.go**

In `cmd/root.go`, line 41, change:

```go
rootCmd.PersistentFlags().StringVar(&flagAs, "as", "", "user ID for this operation (or set OB_USER)")
```

to:

```go
rootCmd.PersistentFlags().StringVar(&flagAs, "as", "", "user ID for audit trail (who ran this command)")
```

- [ ] **Step 4: Run full test suite**

Run: `./scripts/test.sh`
Expected: ALL pass

- [ ] **Step 5: Commit**

```bash
git add cmd/helpers.go cmd/root.go
git commit -m "feat: drop OB_USER, add deprecation warning, update --as help text"
```

---

## Chunk 3: TUI — @unassigned Badge

### Task 7: Add `unassignedStyle` and render `@unassigned`

**Files:**
- Modify: `internal/tui/styles.go:28-30` (add new style)
- Modify: `internal/tui/card.go:43-53` (metadata rendering)
- Test: `internal/tui/card_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/card_test.go`:

```go
func TestRenderCard_UnassignedBadge(t *testing.T) {
	board := &domain.Board{
		Items: map[string]*domain.Item{},
		Users: map[string]*domain.Identity{},
	}
	app := App{board: board}

	item := &domain.Item{
		ID:         "test-id",
		DisplayNum: 5,
		Type:       "task",
		Title:      "Test task",
		Priority:   "medium",
		Assignee:   "",
	}

	rendered := app.renderCard(item, false)
	if !strings.Contains(rendered, "@unassigned") {
		t.Errorf("expected '@unassigned' in card output, got:\n%s", rendered)
	}
}

func TestRenderCard_AssignedBadge(t *testing.T) {
	board := &domain.Board{
		Items: map[string]*domain.Item{},
		Users: map[string]*domain.Identity{
			"user1": {ID: "user1", Name: "Claude", Type: "agent"},
		},
	}
	app := App{board: board}

	item := &domain.Item{
		ID:         "test-id",
		DisplayNum: 5,
		Type:       "task",
		Title:      "Test task",
		Priority:   "medium",
		Assignee:   "user1",
	}

	rendered := app.renderCard(item, false)
	if !strings.Contains(rendered, "@Claude") {
		t.Errorf("expected '@Claude' in card output, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "@unassigned") {
		t.Errorf("should not contain '@unassigned' for assigned card")
	}
}
```

Note: add `import "github.com/niladribose/obeya/internal/domain"` to the test file if not already present. The test may need adjustments based on the App struct's required fields — `columnWidth()` needs the `width` field set. Initialize `app` with `width: 120` or whatever field controls it.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRenderCard_ -v`
Expected: FAIL — `@unassigned` not in output

- [ ] **Step 3: Add unassignedStyle to styles.go**

In `internal/tui/styles.go`, after `assigneeStyle` on line 30, add:

```go
unassignedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
```

- [ ] **Step 4: Update card.go metadata section**

In `internal/tui/card.go`, replace lines 43-53:

```go
var metaParts []string
if item.Assignee != "" {
    name := resolveUserName(a.board, item.Assignee)
    metaParts = append(metaParts, assigneeStyle.Render("@"+name))
} else {
    metaParts = append(metaParts, unassignedStyle.Render("@unassigned"))
}
if len(item.BlockedBy) > 0 {
    metaParts = append(metaParts, blockedStyle.Render("[!]"))
}
lines = append(lines, strings.Join(metaParts, " "))
```

Note: the `if len(metaParts) > 0` check is removed — there will always be at least one meta part (assignee or unassigned).

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestRenderCard_ -v`
Expected: PASS

- [ ] **Step 6: Update golden files**

Run: `./scripts/test.sh --update`
Expected: Golden files regenerated with `@unassigned` badges

- [ ] **Step 7: Run full test suite**

Run: `./scripts/test.sh`
Expected: ALL pass (including updated golden files)

- [ ] **Step 8: Commit**

```bash
git add internal/tui/styles.go internal/tui/card.go internal/tui/card_test.go internal/tui/testdata/
git commit -m "feat: render @unassigned badge on cards without owner"
```

---

## Chunk 4: Skill Updates

### Task 8: Update ob-create skill

**Files:**
- Modify: `obeya-plugin/skills/ob-create/SKILL.md`

- [ ] **Step 1: Update the skill file**

Key changes:
- In the Usage section, change `--assign <user>` from optional to **required**
- Add a note: `--assign` is mandatory. If you are an agent, assign yourself (e.g., `--assign claude`).
- Remove any `OB_USER` references
- Update the Steps section: step 1 should validate that `--assign` is provided before running the command. If the caller didn't provide one, the agent should assign itself.

- [ ] **Step 2: Commit**

```bash
git add obeya-plugin/skills/ob-create/SKILL.md
git commit -m "skill: update ob-create for mandatory --assign"
```

---

### Task 9: Update ob-subtask skill

**Files:**
- Modify: `obeya-plugin/skills/ob-subtask/SKILL.md`

- [ ] **Step 1: Update the skill file**

Key changes:
- `--assign` is mandatory on the underlying `ob create` call
- Add a step before creating: read the parent item's assignee via `ob show <parent> --format json`, and pass it as `--assign <parent-assignee>` unless the user specifies a different one
- Remove any `OB_USER` references
- Make clear: there is no automatic inheritance — the skill must always pass `--assign` explicitly

- [ ] **Step 2: Commit**

```bash
git add obeya-plugin/skills/ob-subtask/SKILL.md
git commit -m "skill: update ob-subtask for mandatory --assign with parent lookup"
```

---

### Task 10: Update ob-pick skill

**Files:**
- Modify: `obeya-plugin/skills/ob-pick/SKILL.md`

- [ ] **Step 1: Update the skill file**

Key changes:
- Update step 3: tasks can be in any assignment state, not just unassigned
- Add step between finding task and moving it: if the picked task has no assignee, run `ob assign <id> --to <self>` first. The agent determines its own name from `ob user list --format json` (filter for `type: agent`).
- Update step 5 (move): the move will now fail if assignee is not set, so the assign step must come first
- Remove the Environment section referencing `OB_USER`
- Add `--as <agent-name>` to the move command for audit trail

- [ ] **Step 2: Commit**

```bash
git add obeya-plugin/skills/ob-pick/SKILL.md
git commit -m "skill: update ob-pick with assign-then-move flow"
```

---

### Task 11: Update ob-status, ob-done, ob-show skills

**Files:**
- Modify: `obeya-plugin/skills/ob-status/SKILL.md`
- Modify: `obeya-plugin/skills/ob-done/SKILL.md`
- Modify: `obeya-plugin/skills/ob-show/SKILL.md`

- [ ] **Step 1: Update ob-status**

Key changes:
- Remove step 2's `OB_USER` detection
- Replace with: determine current user by checking `--as` flag value, or run `ob user list --format json` and ask the user to confirm which identity to use
- Filter using `ob list --assignee <user> --format json`

- [ ] **Step 2: Update ob-done**

Key changes:
- Remove line 14's `OB_USER` filtering reference
- Replace with: filter in-progress items by assignee field directly (`ob list --status in-progress --assignee <user> --format json`)

- [ ] **Step 3: Update ob-show**

Key changes:
- In the display output format, show `Assignee: <name>` when set
- Show `Assignee: unassigned` in red/warning when empty

- [ ] **Step 4: Commit**

```bash
git add obeya-plugin/skills/ob-status/SKILL.md obeya-plugin/skills/ob-done/SKILL.md obeya-plugin/skills/ob-show/SKILL.md
git commit -m "skill: update ob-status, ob-done, ob-show — drop OB_USER, use assignee field"
```

---

### Task 12: Update skill/obeya.md top-level skill

**Files:**
- Modify: `skill/obeya.md`

- [ ] **Step 1: Remove all OB_USER references**

Search for `OB_USER` and `$OB_USER` in `skill/obeya.md` and replace with guidance about `--assign` for ownership and `--as` for audit.

- [ ] **Step 2: Commit**

```bash
git add skill/obeya.md
git commit -m "skill: remove OB_USER references from top-level obeya skill"
```

---

## Chunk 5: Final Validation

### Task 13: Full test suite and manual validation

**Files:** None (validation only)

- [ ] **Step 1: Run full test suite**

Run: `./scripts/test.sh`
Expected: ALL 4 checks pass (build, vet, tests, golden files)

- [ ] **Step 2: Manual CLI smoke test**

```bash
# Should fail: no --assign
./ob create task "Test" -d "test description"

# Should succeed
./ob create task "Test" --assign "Niladri" -d "test description"

# Should fail: unassigned legacy item
./ob move 1 in-progress  # (if item #1 has no assignee)

# Should succeed: assign then move
./ob assign 1 --to "Claude Opus"
./ob move 1 in-progress

# Should show deprecation warning
OB_USER=test ./ob list
```

- [ ] **Step 3: Launch TUI and verify visually**

Run: `./ob tui`
Expected: cards without assignees show `@unassigned` in red. Cards with assignees show `@<name>` in cyan.

- [ ] **Step 4: Commit any final fixes**

```bash
git add -A
git commit -m "fix: address any issues found during final validation"
```
