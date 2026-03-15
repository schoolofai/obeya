# Agentic-First Human Review Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add human review capabilities to Obeya so humans can efficiently review agent-completed work with full context, confidence scoring, and a dedicated review queue.

**Architecture:** Extend the existing Item type with Sponsor, Confidence, ReviewContext, and HumanReview fields. Add engine operations for completing items with context and marking reviews. TUI gains a virtual human-review column, review context accordion, and past reviews pane. CLI gains `ob done` and `ob review` commands.

**Tech Stack:** Go 1.26, Cobra CLI, Bubble Tea TUI, lipgloss styling, teatest for TUI testing

---

## Chunk 1: Data Model & Engine Core

### Task 1: Add new domain types

**Files:**
- Modify: `internal/domain/types.go:97-113` (Item struct)
- Modify: `internal/domain/types.go` (add new structs after Item)

- [ ] **Step 1: Write tests for new types**

Create `internal/domain/types_review_test.go`:

```go
package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReviewContext_JSONRoundTrip(t *testing.T) {
	rc := ReviewContext{
		Purpose: "Replace cookie sessions",
		FilesChanged: []FileChange{
			{Path: "auth/middleware.go", Added: 82, Removed: 41, Diff: "+ new line\n- old line"},
		},
		TestsWritten: []TestResult{
			{Name: "TestJWT", Passed: true},
		},
		Proof: []ProofItem{
			{Check: "go vet", Status: "pass"},
			{Check: "edge cases", Status: "fail", Detail: "no concurrency tests"},
		},
		Reasoning: "JWT for debuggability",
		Reproduce: []string{"go test ./auth/ -run TestJWT"},
	}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ReviewContext
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Purpose != rc.Purpose {
		t.Errorf("Purpose = %q, want %q", got.Purpose, rc.Purpose)
	}
	if len(got.FilesChanged) != 1 || got.FilesChanged[0].Diff != rc.FilesChanged[0].Diff {
		t.Error("FilesChanged roundtrip failed")
	}
	if len(got.Reproduce) != 1 || got.Reproduce[0] != "go test ./auth/ -run TestJWT" {
		t.Error("Reproduce roundtrip failed")
	}
}

func TestHumanReview_JSONRoundTrip(t *testing.T) {
	hr := HumanReview{
		Status:     "reviewed",
		ReviewedBy: "user-123",
		ReviewedAt: time.Now().Truncate(time.Second),
	}
	data, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got HumanReview
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "reviewed" || got.ReviewedBy != "user-123" {
		t.Error("HumanReview roundtrip failed")
	}
}

func TestItem_ConfidencePointer(t *testing.T) {
	// nil = unset
	item := Item{ID: "a", Title: "test"}
	data, _ := json.Marshal(item)
	if string(data) != "" && json.Valid(data) {
		var got Item
		json.Unmarshal(data, &got)
		if got.Confidence != nil {
			t.Error("nil Confidence should remain nil after roundtrip")
		}
	}

	// explicit 0 = agent reports 0%
	zero := 0
	item.Confidence = &zero
	data, _ = json.Marshal(item)
	var got Item
	json.Unmarshal(data, &got)
	if got.Confidence == nil || *got.Confidence != 0 {
		t.Error("explicit 0 Confidence should survive roundtrip")
	}
}

func TestItem_BackwardCompatible(t *testing.T) {
	// Old JSON without new fields should deserialize cleanly
	oldJSON := `{"id":"abc","display_num":1,"type":"task","title":"old","status":"done","priority":"medium","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`
	var item Item
	if err := json.Unmarshal([]byte(oldJSON), &item); err != nil {
		t.Fatalf("backward compat unmarshal failed: %v", err)
	}
	if item.Sponsor != "" {
		t.Error("Sponsor should be empty for old items")
	}
	if item.Confidence != nil {
		t.Error("Confidence should be nil for old items")
	}
	if item.ReviewContext != nil {
		t.Error("ReviewContext should be nil for old items")
	}
	if item.HumanReview != nil {
		t.Error("HumanReview should be nil for old items")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/ -run "TestReviewContext|TestHumanReview|TestItem_Confidence|TestItem_Backward" -v`
Expected: FAIL — `ReviewContext`, `HumanReview`, `FileChange`, etc. types not defined

- [ ] **Step 3: Add new fields to Item struct**

In `internal/domain/types.go`, add to the `Item` struct after `History`:

```go
type Item struct {
	// ... existing fields through History ...

	Sponsor       string         `json:"sponsor,omitempty"`
	Confidence    *int           `json:"confidence,omitempty"`
	ReviewContext *ReviewContext  `json:"review_context,omitempty"`
	HumanReview   *HumanReview   `json:"human_review,omitempty"`
}
```

- [ ] **Step 4: Add new structs**

Append to `internal/domain/types.go` after the `Board` type and its methods:

```go
type ReviewContext struct {
	Purpose      string       `json:"purpose"`
	FilesChanged []FileChange `json:"files_changed,omitempty"`
	TestsWritten []TestResult `json:"tests_written,omitempty"`
	Proof        []ProofItem  `json:"proof,omitempty"`
	Reasoning    string       `json:"reasoning,omitempty"`
	Reproduce    []string     `json:"reproduce,omitempty"`
}

type FileChange struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Diff    string `json:"diff,omitempty"`
}

type TestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}

type ProofItem struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type HumanReview struct {
	Status     string    `json:"status"`
	ReviewedBy string    `json:"reviewed_by,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at,omitempty"`
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/domain/ -run "TestReviewContext|TestHumanReview|TestItem_Confidence|TestItem_Backward" -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `./scripts/test.sh`
Expected: All 257+ tests pass (new fields have zero values, existing tests unaffected)

- [ ] **Step 7: Commit**

```bash
git add internal/domain/types.go internal/domain/types_review_test.go
git commit -m "feat: add ReviewContext, HumanReview, Sponsor, Confidence to domain model"
```

### Task 2: Add resolveSponsor and ResolveDownstream helpers

**Files:**
- Create: `internal/engine/sponsor.go`
- Create: `internal/engine/sponsor_test.go`
- Create: `internal/engine/downstream.go`
- Create: `internal/engine/downstream_test.go`

- [ ] **Step 1: Write sponsor resolution tests**

Create `internal/engine/sponsor_test.go`:

```go
package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func boardWithUsers(humans []string, agents []string) *domain.Board {
	b := domain.NewBoard("test")
	for _, name := range humans {
		id := domain.GenerateID()
		b.Users[id] = &domain.Identity{ID: id, Name: name, Type: domain.IdentityHuman}
	}
	for _, name := range agents {
		id := domain.GenerateID()
		b.Users[id] = &domain.Identity{ID: id, Name: name, Type: domain.IdentityAgent}
	}
	return b
}

func findUserID(b *domain.Board, name string) string {
	for _, u := range b.Users {
		if u.Name == name {
			return u.ID
		}
	}
	return ""
}

func TestResolveSponsor_HumanCreatesItem(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	humanID := findUserID(b, "alice")
	got, err := resolveSponsor(b, humanID, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("human sponsor = %q, want empty", got)
	}
}

func TestResolveSponsor_AutoAssignSingleHuman(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	got, err := resolveSponsor(b, agentID, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	aliceID := findUserID(b, "alice")
	if got != aliceID {
		t.Errorf("sponsor = %q, want %q (alice)", got, aliceID)
	}
}

func TestResolveSponsor_Explicit(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	bobID := findUserID(b, "bob")
	got, err := resolveSponsor(b, agentID, bobID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bobID {
		t.Errorf("sponsor = %q, want %q (bob)", got, bobID)
	}
}

func TestResolveSponsor_CopiedFromParent(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	aliceID := findUserID(b, "alice")
	agentID := findUserID(b, "claude")

	// Create parent epic with sponsor
	epic := &domain.Item{ID: "epic-1", Sponsor: aliceID}
	b.Items["epic-1"] = epic
	b.DisplayMap[1] = "epic-1"

	got, err := resolveSponsor(b, agentID, "", "epic-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != aliceID {
		t.Errorf("sponsor = %q, want %q (alice, from parent)", got, aliceID)
	}
}

func TestResolveSponsor_MultipleHumansNoSponsor(t *testing.T) {
	b := boardWithUsers([]string{"alice", "bob"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	_, err := resolveSponsor(b, agentID, "", "")
	if err == nil {
		t.Fatal("expected error for multiple humans with no sponsor")
	}
}

func TestResolveSponsor_ExplicitMustBeHuman(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude", "codex"})
	claudeID := findUserID(b, "claude")
	codexID := findUserID(b, "codex")
	_, err := resolveSponsor(b, claudeID, codexID, "")
	if err == nil {
		t.Fatal("expected error: sponsor must be human")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestResolveSponsor -v`
Expected: FAIL — `resolveSponsor` not defined

- [ ] **Step 3: Implement resolveSponsor**

Create `internal/engine/sponsor.go`:

```go
package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

func resolveSponsor(board *domain.Board, assigneeID string, explicitSponsor string, parentRef string) (string, error) {
	actor, ok := board.Users[assigneeID]
	if !ok {
		return "", nil
	}
	if actor.Type == domain.IdentityHuman {
		return "", nil
	}

	if explicitSponsor != "" {
		sponsor, ok := board.Users[explicitSponsor]
		if !ok {
			return "", fmt.Errorf("unknown sponsor %q: not found in board users", explicitSponsor)
		}
		if sponsor.Type != domain.IdentityHuman {
			return "", fmt.Errorf("sponsor %q is not a human identity", sponsor.Name)
		}
		return explicitSponsor, nil
	}

	humans := humanUsers(board)
	if len(humans) == 1 {
		return humans[0].ID, nil
	}

	if parentRef != "" {
		if parent, ok := board.Items[parentRef]; ok && parent.Sponsor != "" {
			return parent.Sponsor, nil
		}
	}

	names := make([]string, len(humans))
	for i, h := range humans {
		names[i] = h.Name
	}
	sort.Strings(names)
	return "", fmt.Errorf("board has %d humans. Specify --sponsor: %s", len(humans), strings.Join(names, ", "))
}

func humanUsers(board *domain.Board) []*domain.Identity {
	var humans []*domain.Identity
	for _, u := range board.Users {
		if u.Type == domain.IdentityHuman {
			humans = append(humans, u)
		}
	}
	return humans
}
```

- [ ] **Step 4: Run sponsor tests to verify they pass**

Run: `go test ./internal/engine/ -run TestResolveSponsor -v`
Expected: All 6 PASS

- [ ] **Step 5: Write downstream resolution tests**

Create `internal/engine/downstream_test.go`:

```go
package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestResolveDownstream_NoBlockers(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a"}
	got := ResolveDownstream("a", b)
	if len(got) != 0 {
		t.Errorf("got %d downstream, want 0", len(got))
	}
}

func TestResolveDownstream_FindsBlockedItems(t *testing.T) {
	b := domain.NewBoard("test")
	b.Items["a"] = &domain.Item{ID: "a", DisplayNum: 1}
	b.Items["b"] = &domain.Item{ID: "b", DisplayNum: 2, BlockedBy: []string{"a"}}
	b.Items["c"] = &domain.Item{ID: "c", DisplayNum: 3, BlockedBy: []string{"a"}}
	b.Items["d"] = &domain.Item{ID: "d", DisplayNum: 4, BlockedBy: []string{"b"}}

	got := ResolveDownstream("a", b)
	if len(got) != 2 {
		t.Fatalf("got %d downstream, want 2", len(got))
	}
	// Should contain b and c but not d (d is blocked by b, not a)
	ids := map[string]bool{}
	for _, id := range got {
		ids[id] = true
	}
	if !ids["b"] || !ids["c"] {
		t.Errorf("expected b and c, got %v", got)
	}
}
```

- [ ] **Step 6: Implement ResolveDownstream**

Create `internal/engine/downstream.go`:

```go
package engine

import "github.com/niladribose/obeya/internal/domain"

// ResolveDownstream returns IDs of items directly blocked by the given item.
func ResolveDownstream(itemID string, board *domain.Board) []string {
	var downstream []string
	for _, item := range board.Items {
		for _, blockerID := range item.BlockedBy {
			if blockerID == itemID {
				downstream = append(downstream, item.ID)
				break
			}
		}
	}
	return downstream
}
```

- [ ] **Step 7: Run downstream tests**

Run: `go test ./internal/engine/ -run TestResolveDownstream -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/engine/sponsor.go internal/engine/sponsor_test.go internal/engine/downstream.go internal/engine/downstream_test.go
git commit -m "feat: add resolveSponsor and ResolveDownstream helpers"
```

### Task 3: Add ResolveActorType helper

**Files:**
- Create: `internal/engine/actor.go`
- Create: `internal/engine/actor_test.go`

- [ ] **Step 1: Write tests**

Create `internal/engine/actor_test.go`:

```go
package engine

import (
	"testing"
)

func TestResolveActorType_Agent(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	eng := New(nil) // store not needed for this test
	got := resolveActorTypeFromBoard(b, "claude")
	if got != "agent" {
		t.Errorf("got %q, want agent", got)
	}
}

func TestResolveActorType_Human(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	got := resolveActorTypeFromBoard(b, "alice")
	if got != "human" {
		t.Errorf("got %q, want human", got)
	}
}

func TestResolveActorType_Unknown(t *testing.T) {
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	got := resolveActorTypeFromBoard(b, "unknown-user")
	if got != "human" {
		t.Errorf("got %q, want human (default for unknown)", got)
	}
}
```

- [ ] **Step 2: Implement**

Create `internal/engine/actor.go`:

```go
package engine

import (
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

// resolveActorTypeFromBoard checks if a raw user name/ID is a human or agent.
// Returns "human" (default for unknown), or "agent".
func resolveActorTypeFromBoard(board *domain.Board, rawUserID string) string {
	for _, u := range board.Users {
		if u.ID == rawUserID || strings.EqualFold(u.Name, rawUserID) {
			return string(u.Type)
		}
	}
	return "human"
}

// ResolveActorType loads the board and checks identity type.
func (e *Engine) ResolveActorType(rawUserID string) (string, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return "", err
	}
	return resolveActorTypeFromBoard(board, rawUserID), nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/engine/ -run TestResolveActorType -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/engine/actor.go internal/engine/actor_test.go
git commit -m "feat: add ResolveActorType helper for human/agent detection"
```

### Task 4: Update CreateItem with sponsor parameter

**Files:**
- Modify: `internal/engine/engine.go:25` (CreateItem signature)
- Modify: `internal/engine/engine.go:49` (buildItem call)
- Modify: `cmd/create.go:60` (caller)
- Modify: `internal/tui/app.go:677` (caller)
- Modify: all test files calling CreateItem

- [ ] **Step 1: Write test for sponsor on CreateItem**

Add to `internal/engine/sponsor_test.go`:

```go
func TestCreateItem_AgentWithSponsor(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.NewStore(dir, "")
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	s.SaveBoard(b)

	eng := New(s)
	agentID := findUserID(b, "claude")
	item, err := eng.CreateItem("task", "test", "", "desc", "medium", agentID, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	aliceID := findUserID(b, "alice")
	if item.Sponsor != aliceID {
		t.Errorf("sponsor = %q, want %q (auto-assigned)", item.Sponsor, aliceID)
	}
}
```

- [ ] **Step 2: Update CreateItem signature**

In `internal/engine/engine.go:25`, change:

```go
func (e *Engine) CreateItem(itemType, title, parentRef, description, priority, assignee string, tags []string) (*domain.Item, error) {
```

to:

```go
func (e *Engine) CreateItem(itemType, title, parentRef, description, priority, assignee string, tags []string, sponsor string) (*domain.Item, error) {
```

After the `resolvedAssignee` resolution (line ~47), add sponsor resolution:

```go
		resolvedSponsor, err := resolveSponsor(board, resolvedAssignee, sponsor, parentID)
		if err != nil {
			return err
		}

		item := buildItem(board, itemType, title, description, priority, resolvedAssignee, parentID, tags)
		item.Sponsor = resolvedSponsor
```

- [ ] **Step 3: Fix all call sites**

Add `""` as the trailing sponsor argument to every existing call:

- `cmd/create.go:60`: `eng.CreateItem(itemType, title, createParent, createDesc, createPriority, createAssign, tags, "")` — will gain `createSponsor` variable later
- `internal/tui/app.go:677`: append `""` to the CreateItem call
- All calls in `internal/engine/engine_test.go`
- All calls in `test/integration_test.go`
- All calls in `internal/store/cloud_store_integration_test.go` (if any)

Use project-wide search for `CreateItem(` to find all call sites.

- [ ] **Step 4: Run full test suite**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go cmd/create.go internal/tui/app.go internal/engine/engine_test.go test/
git commit -m "feat: add sponsor parameter to CreateItem with deterministic resolution"
```

### Task 5: Add ReviewItem engine operation

**Files:**
- Modify: `internal/engine/engine.go` (add new method)
- Add tests to: `internal/engine/engine_test.go`

- [ ] **Step 1: Write tests**

Add to `internal/engine/engine_test.go` (or create `internal/engine/review_test.go`):

Create `internal/engine/review_test.go`:

```go
package engine

import (
	"testing"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

func setupReviewTestEngine(t *testing.T) (*Engine, *domain.Board) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.NewStore(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	item := &domain.Item{
		ID: "item-1", DisplayNum: 1, Title: "test task",
		Type: domain.ItemTypeTask, Status: "done",
		Assignee: agentID, Priority: domain.PriorityMedium,
		ReviewContext: &domain.ReviewContext{Purpose: "test"},
	}
	b.Items["item-1"] = item
	b.DisplayMap[1] = "item-1"
	s.SaveBoard(b)
	return New(s), b
}

func TestReviewItem_Reviewed(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	if err := eng.ReviewItem("1", "reviewed", aliceID, "sess-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, _ := eng.GetItem("1")
	if item.HumanReview == nil || item.HumanReview.Status != "reviewed" {
		t.Error("expected HumanReview.Status == reviewed")
	}
	if item.HumanReview.ReviewedBy != aliceID {
		t.Errorf("ReviewedBy = %q, want %q", item.HumanReview.ReviewedBy, aliceID)
	}
}

func TestReviewItem_Hidden(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	if err := eng.ReviewItem("1", "hidden", aliceID, "sess-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, _ := eng.GetItem("1")
	if item.HumanReview == nil || item.HumanReview.Status != "hidden" {
		t.Error("expected HumanReview.Status == hidden")
	}
}

func TestReviewItem_AgentCannotReview(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	claudeID := findUserID(b, "claude")
	err := eng.ReviewItem("1", "reviewed", claudeID, "sess-1")
	if err == nil {
		t.Fatal("expected error: agents cannot review")
	}
}

func TestReviewItem_InvalidStatus(t *testing.T) {
	eng, b := setupReviewTestEngine(t)
	aliceID := findUserID(b, "alice")
	err := eng.ReviewItem("1", "invalid", aliceID, "sess-1")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}
```

- [ ] **Step 2: Implement ReviewItem**

Add to `internal/engine/engine.go`:

```go
func (e *Engine) ReviewItem(ref string, status string, userID string, sessionID string) error {
	if status != "reviewed" && status != "hidden" {
		return fmt.Errorf("invalid review status %q: must be 'reviewed' or 'hidden'", status)
	}

	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		actorType := resolveActorTypeFromBoard(board, userID)
		if actorType == "agent" {
			return fmt.Errorf("agents cannot review items — only humans can mark items as reviewed")
		}

		item := board.Items[id]
		item.HumanReview = &domain.HumanReview{
			Status:     status,
			ReviewedBy: userID,
			ReviewedAt: time.Now(),
		}
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "human-review", status)

		return nil
	})
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/engine/ -run TestReviewItem -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/engine/engine.go internal/engine/review_test.go
git commit -m "feat: add ReviewItem engine operation for human review marking"
```

### Task 6: Add CompleteItemWithContext engine operation

**Files:**
- Modify: `internal/engine/engine.go` (add new method)

- [ ] **Step 1: Write tests**

Add to `internal/engine/review_test.go`:

```go
func TestCompleteItemWithContext(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.NewStore(dir, "")
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	item := &domain.Item{
		ID: "item-1", DisplayNum: 1, Title: "test",
		Type: domain.ItemTypeTask, Status: "in-progress",
		Assignee: agentID, Priority: domain.PriorityMedium,
	}
	b.Items["item-1"] = item
	b.DisplayMap[1] = "item-1"
	s.SaveBoard(b)

	eng := New(s)
	ctx := domain.ReviewContext{Purpose: "JWT migration"}
	err := eng.CompleteItemWithContext("1", ctx, 45, agentID, "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := eng.GetItem("1")
	if got.Status != "done" {
		t.Errorf("Status = %q, want done", got.Status)
	}
	if got.ReviewContext == nil || got.ReviewContext.Purpose != "JWT migration" {
		t.Error("ReviewContext not set correctly")
	}
	if got.Confidence == nil || *got.Confidence != 45 {
		t.Error("Confidence not set to 45")
	}
	if got.HumanReview == nil || got.HumanReview.Status != "pending" {
		t.Error("HumanReview should be pending")
	}
}

func TestCompleteItemWithContext_Idempotent(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.NewStore(dir, "")
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	agentID := findUserID(b, "claude")
	item := &domain.Item{
		ID: "item-1", DisplayNum: 1, Title: "test",
		Type: domain.ItemTypeTask, Status: "done",
		Assignee: agentID, Priority: domain.PriorityMedium,
		ReviewContext: &domain.ReviewContext{Purpose: "old"},
		HumanReview:   &domain.HumanReview{Status: "reviewed"},
	}
	b.Items["item-1"] = item
	b.DisplayMap[1] = "item-1"
	s.SaveBoard(b)

	eng := New(s)
	ctx := domain.ReviewContext{Purpose: "new purpose"}
	err := eng.CompleteItemWithContext("1", ctx, 80, agentID, "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := eng.GetItem("1")
	if got.ReviewContext.Purpose != "new purpose" {
		t.Error("ReviewContext should be overwritten")
	}
	if got.HumanReview.Status != "pending" {
		t.Error("HumanReview should reset to pending")
	}
}

func TestCompleteItemWithContext_HumanIdentityAllowed(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.NewStore(dir, "")
	b := boardWithUsers([]string{"alice"}, []string{"claude"})
	aliceID := findUserID(b, "alice")
	item := &domain.Item{
		ID: "item-1", DisplayNum: 1, Title: "test",
		Type: domain.ItemTypeTask, Status: "in-progress",
		Assignee: aliceID, Priority: domain.PriorityMedium,
	}
	b.Items["item-1"] = item
	b.DisplayMap[1] = "item-1"
	s.SaveBoard(b)

	eng := New(s)
	ctx := domain.ReviewContext{Purpose: "human context"}
	err := eng.CompleteItemWithContext("1", ctx, 90, aliceID, "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: humans should be able to provide review context: %v", err)
	}
}
```

- [ ] **Step 2: Implement**

Add to `internal/engine/engine.go`:

```go
func (e *Engine) CompleteItemWithContext(ref string, ctx domain.ReviewContext, confidence int, userID string, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}

		oldStatus := item.Status
		item.Status = "done"
		item.ReviewContext = &ctx
		item.Confidence = &confidence
		item.HumanReview = &domain.HumanReview{Status: "pending"}
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "complete-with-context", fmt.Sprintf("status: %s -> done, purpose: %s", oldStatus, ctx.Purpose))

		return nil
	})
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/engine/ -run TestCompleteItemWithContext -v`
Expected: PASS

- [ ] **Step 4: Run full test suite**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: add CompleteItemWithContext for agent work completion with review data"
```

## Chunk 2: CLI Commands

### Task 7: Add `ob review` command

**Files:**
- Create: `cmd/review.go`
- Create: `cmd/review_test.go`

- [ ] **Step 1: Implement the command**

Create `cmd/review.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var reviewStatus string

var reviewCmd = &cobra.Command{
	Use:   "review <id>",
	Short: "Mark an item as reviewed or hidden",
	Long:  "Set the human review status of an agent-completed item. Only available to human identities.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if reviewStatus == "" {
			return fmt.Errorf("--status is required (reviewed or hidden)")
		}
		if reviewStatus != "reviewed" && reviewStatus != "hidden" {
			return fmt.Errorf("--status must be 'reviewed' or 'hidden', got %q", reviewStatus)
		}

		eng, err := getEngine()
		if err != nil {
			return err
		}

		if err := eng.ReviewItem(args[0], reviewStatus, getUserID(), getSessionID()); err != nil {
			return err
		}

		fmt.Printf("Marked #%s as %s\n", args[0], reviewStatus)
		return nil
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewStatus, "status", "", "review status: reviewed or hidden")
	rootCmd.AddCommand(reviewCmd)
}
```

- [ ] **Step 2: Test manually**

Run: `go build -o ob . && ./ob review --help`
Expected: Help text shows usage

- [ ] **Step 3: Commit**

```bash
git add cmd/review.go
git commit -m "feat: add 'ob review' command for marking items reviewed/hidden"
```

### Task 8: Add `ob done` command

**Files:**
- Create: `cmd/done.go`

- [ ] **Step 1: Implement the command**

Create `cmd/done.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/spf13/cobra"
)

var (
	doneConfidence int
	donePurpose    string
	doneReasoning  string
	doneFiles      string
	doneTests      string
	doneReproduce  []string
	doneProof      string
	doneContextIn  bool
)

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Complete an item with review context",
	Long:  "Mark an item as done and attach review context (files changed, tests, confidence, reasoning).\nPreferred over 'ob move <id> done' for agent-completed work.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if doneConfidence < 0 || doneConfidence > 100 {
			return fmt.Errorf("--confidence must be 0-100, got %d", doneConfidence)
		}

		eng, err := getEngine()
		if err != nil {
			return err
		}

		var ctx domain.ReviewContext

		if doneContextIn {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			if err := json.Unmarshal(data, &ctx); err != nil {
				return fmt.Errorf("invalid JSON on stdin: %w", err)
			}
		} else {
			ctx.Purpose = donePurpose
			ctx.Reasoning = doneReasoning
			ctx.Reproduce = doneReproduce
			files, err := parseFiles(doneFiles)
			if err != nil {
				return fmt.Errorf("invalid --files: %w", err)
			}
			ctx.FilesChanged = files
			ctx.TestsWritten = parseTests(doneTests)
			ctx.Proof = parseProof(doneProof)
		}

		if err := eng.CompleteItemWithContext(args[0], ctx, doneConfidence, getUserID(), getSessionID()); err != nil {
			return err
		}

		fmt.Printf("Completed #%s with confidence %d%%\n", args[0], doneConfidence)
		return nil
	},
}

func parseFiles(raw string) ([]domain.FileChange, error) {
	if raw == "" {
		return nil, nil
	}
	var files []domain.FileChange
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, ":", 2)
		fc := domain.FileChange{Path: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			counts := parts[1]
			if idx := strings.Index(counts, "-"); idx > 0 {
				added, err := strconv.Atoi(strings.TrimPrefix(counts[:idx], "+"))
				if err != nil {
					return nil, fmt.Errorf("invalid added count for %q: %w", fc.Path, err)
				}
				removed, err := strconv.Atoi(counts[idx+1:])
				if err != nil {
					return nil, fmt.Errorf("invalid removed count for %q: %w", fc.Path, err)
				}
				fc.Added = added
				fc.Removed = removed
			}
		}
		files = append(files, fc)
	}
	return files, nil
}

func parseTests(raw string) []domain.TestResult {
	if raw == "" {
		return nil
	}
	var tests []domain.TestResult
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, ":", 2)
		tr := domain.TestResult{Name: parts[0]}
		if len(parts) == 2 {
			tr.Passed = parts[1] == "pass"
		}
		tests = append(tests, tr)
	}
	return tests
}

func parseProof(raw string) []domain.ProofItem {
	if raw == "" {
		return nil
	}
	var proof []domain.ProofItem
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, ":", 3)
		pi := domain.ProofItem{Check: parts[0]}
		if len(parts) >= 2 {
			pi.Status = parts[1]
		}
		if len(parts) >= 3 {
			pi.Detail = parts[2]
		}
		proof = append(proof, pi)
	}
	return proof
}

func init() {
	doneCmd.Flags().IntVar(&doneConfidence, "confidence", -1, "confidence level 0-100 (required)")
	doneCmd.MarkFlagRequired("confidence")
	doneCmd.Flags().StringVar(&donePurpose, "purpose", "", "purpose of the change")
	doneCmd.Flags().StringVar(&doneReasoning, "reasoning", "", "agent decision rationale")
	doneCmd.Flags().StringVar(&doneFiles, "files", "", "files changed (path:+added-removed,...)")
	doneCmd.Flags().StringVar(&doneTests, "tests", "", "tests (name:pass|fail,...)")
	doneCmd.Flags().StringSliceVar(&doneReproduce, "reproduce", nil, "commands to reproduce tests (repeatable)")
	doneCmd.Flags().StringVar(&doneProof, "proof", "", "proof items (check:status:detail,...)")
	doneCmd.Flags().BoolVar(&doneContextIn, "context-stdin", false, "read ReviewContext JSON from stdin")
	rootCmd.AddCommand(doneCmd)
}
```

- [ ] **Step 2: Test the command builds**

Run: `go build -o ob . && ./ob done --help`
Expected: Help text with all flags

- [ ] **Step 3: Commit**

```bash
git add cmd/done.go
git commit -m "feat: add 'ob done' command for agent work completion with review context"
```

### Task 9: Add --sponsor flag to `ob create`

**Files:**
- Modify: `cmd/create.go`

- [ ] **Step 1: Add the flag**

In `cmd/create.go`, add variable:

```go
var createSponsor string
```

In the `init()` function, add:

```go
createCmd.Flags().StringVar(&createSponsor, "sponsor", "", "human sponsor for agent-created items")
```

Update the `eng.CreateItem` call to pass `createSponsor`:

```go
item, err := eng.CreateItem(itemType, title, createParent, createDesc, createPriority, createAssign, tags, createSponsor)
```

- [ ] **Step 2: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/create.go
git commit -m "feat: add --sponsor flag to 'ob create' for agent accountability"
```

### Task 10: Add agent warning to `ob move done`

**Files:**
- Modify: `cmd/move.go`

- [ ] **Step 1: Add warning logic**

In `cmd/move.go`, in the `Run` function, after the successful move (line 38), add (note: `os` and `fmt` are already imported in `move.go`):

```go
		// Warn agents to use 'ob done' instead
		if args[1] == "done" {
			actorType, _ := eng.ResolveActorType(getUserID())
			if actorType == "agent" {
				fmt.Fprintln(os.Stderr, "Warning: use 'ob done "+args[0]+"' to include review context for human review.")
			}
		}
```

- [ ] **Step 2: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/move.go
git commit -m "feat: warn agents to use 'ob done' when moving items to done"
```

## Chunk 3: TUI Card Rendering

### Task 11: Add review-related styles

**Files:**
- Modify: `internal/tui/styles.go`

- [ ] **Step 1: Add new styles**

Append to `internal/tui/styles.go`:

```go
	// Agent badge
	agentBadgeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Bold(true)

	// Confidence indicators
	confLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	confMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	confHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	// Sponsor
	sponsorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)

	// Review queue column (amber instead of cyan)
	reviewQueueColumnStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 0).
		MarginRight(1)

	activeReviewQueueColumnStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(0, 0).
		MarginRight(1)

	// Reviewed card
	reviewedCardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Faint(true).
		Padding(0, 1)

	// Review context accordion
	reviewContextIndicatorStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("3"))

	// Downstream impact
	downstreamStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
```

Add helper functions:

```go
func confidenceIndicator(confidence *int) string {
	if confidence == nil {
		return ""
	}
	c := *confidence
	switch {
	case c <= 50:
		return confLow.Render(fmt.Sprintf("%d%% ⚠ LOW", c))
	case c <= 75:
		return confMedium.Render(fmt.Sprintf("%d%%", c))
	default:
		return confHigh.Render(fmt.Sprintf("%d%% ✓", c))
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: Compiles cleanly

- [ ] **Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add TUI styles for agent badge, confidence, review queue"
```

### Task 12: Update card rendering with agent badge, confidence, sponsor

**Files:**
- Modify: `internal/tui/card.go:12-78` (renderCard method)

- [ ] **Step 1: Write tests for agent badge rendering**

Add to `internal/tui/app_test.go`:

```go
func TestAgentBadge_Renders(t *testing.T) {
	// Setup: board with agent identity, item assigned to agent
	// Render card, assert output contains "AGENT"
}

func TestConfidenceIndicator_Colors(t *testing.T) {
	// Test nil → no output
	// Test 0 → "0% ⚠ LOW"
	// Test 45 → "45% ⚠ LOW"
	// Test 72 → "72%"
	// Test 95 → "95% ✓"
}
```

- [ ] **Step 2: Update renderCard**

In `internal/tui/card.go:renderCard`, modify the title line to include agent badge:

After `prefix := fmt.Sprintf("#%d ", item.DisplayNum)`:

```go
	// Agent badge
	isAgent := false
	if u, ok := a.board.Users[item.Assignee]; ok && u.Type == domain.IdentityAgent {
		isAgent = true
		prefix = "AGENT " + prefix
	}
```

Style the "AGENT" prefix with `agentBadgeStyle`.

On the type/priority line (line2), append confidence:

```go
	confStr := confidenceIndicator(item.Confidence)
	if confStr != "" {
		line2 = fmt.Sprintf("%s %s  %s", typLabel, priorityIndicator(string(item.Priority)), confStr)
	}
```

After the meta line (assignee), add sponsor:

```go
	// resolveUserName already exists in board.go:215
	if item.Sponsor != "" {
		sponsorName := resolveUserName(a.board, item.Sponsor)
		metaParts = append(metaParts, sponsorStyle.Render("sponsor:@"+sponsorName))
	}
```

After meta, add downstream impact:

```go
	downstream := engine.ResolveDownstream(item.ID, a.board)
	if len(downstream) > 0 {
		lines = append(lines, downstreamStyle.Render(fmt.Sprintf("⚡ unblocks %d tasks", len(downstream))))
	}
```

- [ ] **Step 3: Add review context accordion (separate from description)**

After the existing description accordion block in `renderCard`, add:

```go
	// Review context accordion — only when ReviewContext exists
	if selected && item.ReviewContext != nil {
		if a.reviewExpanded == item.ID {
			lines = append(lines, reviewContextIndicatorStyle.Render("▼ review context"))
			sep := strings.Repeat("┄", contentW)
			lines = append(lines, lipgloss.NewStyle().Faint(true).Render(sep))
			lines = append(lines, a.renderReviewContext(item.ReviewContext, contentW, a.reviewScrollY, 5)...)
		} else {
			lines = append(lines, reviewContextIndicatorStyle.Render("▶ review context"))
		}
	}
```

- [ ] **Step 4: Add reviewExpanded/reviewScrollY to App struct**

In `internal/tui/app.go`, add to the App struct:

```go
	// Review context accordion
	reviewExpanded string // item ID whose review context is expanded
	reviewScrollY  int
```

- [ ] **Step 5: Create renderReviewContext function**

Create `internal/tui/review_context.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

func (a App) renderReviewContext(rc *domain.ReviewContext, maxWidth int, scrollY int, maxLines int) []string {
	lines := reviewContextLines(rc, maxWidth)
	return a.renderScrollableContent(lines, maxWidth, scrollY, maxLines, "Ctrl+J/K")
}

func reviewContextLines(rc *domain.ReviewContext, maxWidth int) []string {
	var lines []string

	if rc.Purpose != "" {
		lines = append(lines, " Purpose: "+rc.Purpose)
	}

	if len(rc.FilesChanged) > 0 {
		lines = append(lines, "")
		for i, f := range rc.FilesChanged {
			prefix := " Files:  "
			if i > 0 {
				prefix = "         "
			}
			lines = append(lines, fmt.Sprintf("%s%s  (+%d -%d)", prefix, f.Path, f.Added, f.Removed))
		}
	}

	if len(rc.TestsWritten) > 0 {
		lines = append(lines, "")
		passed := 0
		for _, t := range rc.TestsWritten {
			if t.Passed {
				passed++
			}
		}
		lines = append(lines, fmt.Sprintf(" Tests:  %d total, %d pass", len(rc.TestsWritten), passed))
	}

	if len(rc.Reproduce) > 0 {
		lines = append(lines, "")
		lines = append(lines, " Reproduce:")
		for _, cmd := range rc.Reproduce {
			lines = append(lines, "   $ "+cmd)
		}
	}

	if len(rc.Proof) > 0 {
		lines = append(lines, "")
		lines = append(lines, " Proof:")
		for _, p := range rc.Proof {
			icon := "✓"
			if p.Status == "fail" {
				icon = "✗"
			} else if p.Status == "warn" {
				icon = "⚠"
			}
			line := fmt.Sprintf("   %s %s", icon, p.Check)
			if p.Detail != "" {
				line += ": " + p.Detail
			}
			lines = append(lines, line)
		}
	}

	return lines
}
```

- [ ] **Step 6: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 7: Update golden files if needed**

Run: `./scripts/test.sh --update`
Then: `./scripts/test.sh --golden`

- [ ] **Step 8: Commit**

```bash
git add internal/tui/card.go internal/tui/app.go internal/tui/review_context.go internal/tui/styles.go
git commit -m "feat: add agent badge, confidence, sponsor, review context accordion to TUI cards"
```

## Chunk 4: TUI Human-Review Column & Past Reviews

### Task 13: Add virtual human-review column

**Files:**
- Modify: `internal/tui/board.go` (visibleItemsInColumn, renderBoard)
- Modify: `internal/tui/app.go` (column initialization, key routing)
- Modify: `internal/tui/column.go` (review queue header variant)

- [ ] **Step 1: Write tests**

```go
func TestHumanReviewColumn_Renders(t *testing.T) {
	// Setup: board with agent-completed items in done with ReviewContext
	// Assert: virtual column appears after done
}

func TestHumanReviewColumn_SortedByConfidence(t *testing.T) {
	// 3 items with confidence 95, 45, 72
	// Assert: order is 45, 72, 95 (ascending)
}

func TestHumanReviewColumn_EmptyHidesColumn(t *testing.T) {
	// No agent-completed items in done
	// Assert: virtual column not rendered
}
```

- [ ] **Step 2: Add helper to check if column is human-review**

In `internal/tui/board.go`:

```go
const humanReviewColName = "human-review"

func isHumanReviewColumn(columns []string, colIdx int) bool {
	return colIdx >= 0 && colIdx < len(columns) && columns[colIdx] == humanReviewColName
}
```

- [ ] **Step 3: Update visibleItemsInColumn for human-review**

In `internal/tui/board.go:visibleItemsInColumn`, add a special case at the top:

```go
	if colName == humanReviewColName {
		return a.humanReviewItems()
	}
```

Add the filtering function:

```go
func (a App) humanReviewItems() []*domain.Item {
	var items []*domain.Item
	for _, item := range a.board.Items {
		if item.Status != "done" || item.ReviewContext == nil {
			continue
		}
		if item.HumanReview != nil && item.HumanReview.Status == "hidden" {
			continue
		}
		items = append(items, item)
	}
	// Sort by confidence ascending (nil first)
	sort.Slice(items, func(i, j int) bool {
		ci := confidenceValue(items[i].Confidence)
		cj := confidenceValue(items[j].Confidence)
		if ci != cj {
			return ci < cj
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return items
}

func confidenceValue(c *int) int {
	if c == nil {
		return -1 // nil sorts first (lowest)
	}
	return *c
}
```

- [ ] **Step 4: Inject virtual column at board load**

In `internal/tui/app.go`, where columns are loaded from the board (in the `loadBoard` handler), after populating `a.columns`:

```go
	// Remove stale virtual column if present (handles reload after R/x actions)
	if len(a.columns) > 0 && a.columns[len(a.columns)-1] == humanReviewColName {
		a.columns = a.columns[:len(a.columns)-1]
	}
	// Append virtual human-review column if there are reviewable items
	if a.hasReviewableItems() {
		a.columns = append(a.columns, humanReviewColName)
	}
```

```go
func (a App) hasReviewableItems() bool {
	if a.board == nil {
		return false
	}
	for _, item := range a.board.Items {
		if item.Status == "done" && item.ReviewContext != nil {
			if item.HumanReview == nil || item.HumanReview.Status != "hidden" {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 5: Add review queue column header styling**

In `internal/tui/column.go`, update the `View` method to use amber styling when the column is `human-review`. Add a header variant that shows "⚡ REVIEW QUEUE" with subtitle.

- [ ] **Step 6: Add key routing for R and x in human-review column**

In `internal/tui/app.go:handleBoardKey`, add before existing key cases:

```go
	if isHumanReviewColumn(a.columns, a.cursorCol) {
		switch msg.String() {
		case "R":
			sel := a.selectedItem()
			if sel != nil {
				a.engine.ReviewItem(fmt.Sprint(sel.DisplayNum), "reviewed", a.userID, a.sessionID)
				return a, a.loadBoard()
			}
		case "x":
			sel := a.selectedItem()
			if sel != nil {
				a.engine.ReviewItem(fmt.Sprint(sel.DisplayNum), "hidden", a.userID, a.sessionID)
				return a, a.loadBoard()
			}
		}
	}
```

- [ ] **Step 7: Add userID and sessionID to App struct**

In `internal/tui/app.go`, add to App struct:

```go
	userID    string
	sessionID string
```

Initialize in `NewApp`:

```go
func NewApp(eng *engine.Engine, boardPath string) App {
	u, _ := user.Current()
	uid := "unknown"
	if u != nil {
		uid = u.Username
	}
	return App{
		engine:    eng,
		boardPath: boardPath,
		collapsed: make(map[string]bool),
		state:     stateBoard,
		userID:    uid,
		sessionID: domain.GenerateID(),
	}
}
```

- [ ] **Step 8: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 9: Commit**

```bash
git add internal/tui/board.go internal/tui/app.go internal/tui/column.go
git commit -m "feat: add virtual human-review column with confidence sorting"
```

### Task 14: Add Past Reviews pane

**Files:**
- Create: `internal/tui/past_reviews.go`
- Modify: `internal/tui/keys.go` (add statePastReviews)
- Modify: `internal/tui/app.go` (state transitions)

- [ ] **Step 1: Add state constant**

In `internal/tui/keys.go`, add to the viewState iota block:

```go
	stateDAG
	statePastReviews
```

Add detail tab for diffs:

```go
	tabFields detailTab = iota
	tabPlan
	tabHistory
	tabDiffs
```

- [ ] **Step 2: Write tests for BuildReviewTree**

```go
func TestBuildReviewTree_Hierarchical(t *testing.T) {
	// Epic with 2 reviewed children → tree has epic as root with 2 children
}

func TestBuildReviewTree_OrphanItems(t *testing.T) {
	// Reviewed item with no parent → appears at root level
}

func TestBuildReviewTree_StructuralNodes(t *testing.T) {
	// Epic not reviewed, but child is → epic appears as structural node
}
```

- [ ] **Step 3: Implement PastReviewsModel**

Create `internal/tui/past_reviews.go`:

```go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

type TreeNode struct {
	Item       *domain.Item
	Children   []*TreeNode
	IsReviewed bool // false = structural-only ancestor
}

type PastReviewsModel struct {
	nodes    []*TreeNode
	board    *domain.Board
	cursor   int
	flatList []*domain.Item // flattened for cursor navigation
	scrollY  int
	width    int
	height   int
}

func newPastReviewsModel(board *domain.Board) PastReviewsModel {
	nodes := BuildReviewTree(board)
	flat := flattenTree(nodes)
	return PastReviewsModel{
		nodes:    nodes,
		board:    board,
		flatList: flat,
	}
}

func BuildReviewTree(board *domain.Board) []*TreeNode {
	reviewed := map[string]bool{}
	for _, item := range board.Items {
		if item.HumanReview != nil {
			reviewed[item.ID] = true
		}
	}

	if len(reviewed) == 0 {
		return nil
	}

	// Build tree from reviewed items and their ancestors
	nodeMap := map[string]*TreeNode{}
	var roots []*TreeNode

	var ensureNode func(id string) *TreeNode
	ensureNode = func(id string) *TreeNode {
		if n, ok := nodeMap[id]; ok {
			return n
		}
		item, ok := board.Items[id]
		if !ok {
			return nil
		}
		n := &TreeNode{Item: item, IsReviewed: reviewed[id]}
		nodeMap[id] = n

		if item.ParentID != "" {
			parent := ensureNode(item.ParentID)
			if parent != nil {
				parent.Children = append(parent.Children, n)
				return n
			}
		}
		roots = append(roots, n)
		return n
	}

	for id := range reviewed {
		ensureNode(id)
	}

	sortNodes(roots)
	return roots
}

func sortNodes(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Item.DisplayNum < nodes[j].Item.DisplayNum
	})
	for _, n := range nodes {
		sortNodes(n.Children)
	}
}

func flattenTree(nodes []*TreeNode) []*domain.Item {
	var flat []*domain.Item
	var walk func([]*TreeNode, int)
	walk = func(ns []*TreeNode, depth int) {
		for _, n := range ns {
			flat = append(flat, n.Item)
			walk(n.Children, depth+1)
		}
	}
	walk(nodes, 0)
	return flat
}

func (m PastReviewsModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Past Reviews") +
		lipgloss.NewStyle().Faint(true).Render("  [Esc close]")

	var lines []string
	var renderTree func([]*TreeNode, string)
	renderTree = func(nodes []*TreeNode, indent string) {
		for i, n := range nodes {
			prefix := "├── "
			childIndent := indent + "│   "
			if i == len(nodes)-1 {
				prefix = "└── "
				childIndent = indent + "    "
			}

			check := "✓ "
			style := lipgloss.NewStyle()
			if !n.IsReviewed {
				check = ""
				style = style.Faint(true)
			}

			label := fmt.Sprintf("%s%s%s#%d %s",
				indent, prefix, check,
				n.Item.DisplayNum, n.Item.Title)
			lines = append(lines, style.Render(label))

			renderTree(n.Children, childIndent)
		}
	}

	// Roots don't get tree connectors
	for _, n := range m.nodes {
		check := "✓ "
		style := lipgloss.NewStyle()
		if !n.IsReviewed {
			check = ""
			style = style.Faint(true)
		}
		typeLabel := string(n.Item.Type)
		label := fmt.Sprintf("%s%s %s#%d %s", check, strings.Title(typeLabel), "", n.Item.DisplayNum, n.Item.Title)
		lines = append(lines, style.Render(label))
		renderTree(n.Children, "  ")
	}

	content := strings.Join(lines, "\n")

	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2)

	if m.width > 0 {
		border = border.Width(m.width - 4)
	}

	return border.Render(title + "\n\n" + content)
}

func (m *PastReviewsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *PastReviewsModel) CursorDown() {
	if m.cursor < len(m.flatList)-1 {
		m.cursor++
	}
}

func (m *PastReviewsModel) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *PastReviewsModel) SelectedItem() *domain.Item {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		return m.flatList[m.cursor]
	}
	return nil
}
```

- [ ] **Step 4: Wire up state transitions in app.go**

Add `pastReviews PastReviewsModel` to App struct.

In `handleBoardKey`, add `P` key handler:

```go
	case "P":
		a.pastReviews = newPastReviewsModel(a.board)
		a.pastReviews.SetSize(a.width, a.height)
		a.state = statePastReviews
		return a, nil
```

Add `handlePastReviewsKey` function:

```go
func (a *App) handlePastReviewsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		a.pastReviews.CursorDown()
	case "k", "up":
		a.pastReviews.CursorUp()
	case "enter":
		sel := a.pastReviews.SelectedItem()
		if sel != nil {
			a.detail = newDetailModel(sel, a.board)
			a.detail.SetSize(a.width, a.height)
			a.prevState = statePastReviews
			a.state = stateDetail
		}
	case "esc", "q":
		a.state = stateBoard
	}
	return a, nil
}
```

In the main `Update` switch on state, add:

```go
	case statePastReviews:
		return a.handlePastReviewsKey(msg)
```

In `View()`, add:

```go
	case statePastReviews:
		return a.pastReviews.View()
```

- [ ] **Step 5: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 6: Update golden files**

Run: `./scripts/test.sh --update`

- [ ] **Step 7: Commit**

```bash
git add internal/tui/past_reviews.go internal/tui/keys.go internal/tui/app.go
git commit -m "feat: add Past Reviews pane with hierarchical tree view"
```

## Chunk 5: TUI Detail View Diffs Tab & Plugin Updates

### Task 15: Add Diffs tab to detail view

**Files:**
- Modify: `internal/tui/detail.go`
- Modify: `internal/tui/keys.go` (tabDiffs already added in Task 14)

- [ ] **Step 1: Write tests**

```go
func TestDetailView_DiffsTab_Renders(t *testing.T) {
	// Item with ReviewContext containing FileChanges with Diff content
	// Switch to Diffs tab, assert diff content visible
}

func TestDetailView_DiffsTab_Hidden(t *testing.T) {
	// Item without ReviewContext
	// Assert only 3 tabs visible (no Diffs)
}
```

- [ ] **Step 2: Update tab count dynamically**

In `internal/tui/detail.go`, modify `NextTab` and `PrevTab` to use dynamic tab count:

```go
func (d *DetailModel) tabCount() int {
	if d.item.ReviewContext != nil && hasAnyDiff(d.item.ReviewContext) {
		return 4
	}
	return 3
}

func hasAnyDiff(rc *domain.ReviewContext) bool {
	for _, f := range rc.FilesChanged {
		if f.Diff != "" {
			return true
		}
	}
	return false
}

func (d *DetailModel) NextTab() {
	d.activeTab = (d.activeTab + 1) % detailTab(d.tabCount())
	d.scrollY = 0
}

func (d *DetailModel) PrevTab() {
	count := detailTab(d.tabCount())
	d.activeTab = (d.activeTab + count - 1) % count
	d.scrollY = 0
}
```

- [ ] **Step 3: Add Diffs tab rendering**

Create `internal/tui/detail_diffs.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

var (
	diffAddStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	diffDelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	diffHunkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Faint(true)
	diffFileStyle = lipgloss.NewStyle().Bold(true)
)

func renderDiffsTab(rc *domain.ReviewContext, width int) string {
	if rc == nil {
		return ""
	}
	var sections []string
	for _, fc := range rc.FilesChanged {
		if fc.Diff == "" {
			continue
		}
		header := diffFileStyle.Render(fmt.Sprintf("%s  (+%d -%d)", fc.Path, fc.Added, fc.Removed))
		sep := strings.Repeat("─", width-4)
		diffLines := colorizeDiff(fc.Diff)
		sections = append(sections, header+"\n"+sep+"\n"+diffLines)
	}
	return strings.Join(sections, "\n\n")
}

func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var styled []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "@@"):
			styled = append(styled, diffHunkStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			styled = append(styled, diffAddStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			styled = append(styled, diffDelStyle.Render(line))
		default:
			styled = append(styled, line)
		}
	}
	return strings.Join(styled, "\n")
}
```

- [ ] **Step 4: Wire into detail View()**

In the detail view's `View()` method, add a case for `tabDiffs`:

```go
	case tabDiffs:
		content = renderDiffsTab(d.item.ReviewContext, d.width)
```

Update the tab bar rendering to include "Diffs" when applicable.

- [ ] **Step 5: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/tui/detail.go internal/tui/detail_diffs.go
git commit -m "feat: add Diffs tab to detail view with syntax-colored unified diffs"
```

### Task 16: Update V key for review context accordion

**Files:**
- Modify: `internal/tui/app.go` (key handling)

- [ ] **Step 1: Add V key handler in handleBoardKey**

In the board key handler, add:

```go
	case "V":
		sel := a.selectedItem()
		if sel != nil && sel.ReviewContext != nil {
			if a.reviewExpanded == sel.ID {
				a.reviewExpanded = ""
				a.reviewScrollY = 0
			} else {
				a.reviewExpanded = sel.ID
				a.reviewScrollY = 0
			}
		}
```

Add `Ctrl+J`/`Ctrl+K` for review context scrolling:

```go
	case "ctrl+j":
		if a.reviewExpanded != "" {
			a.reviewScrollY++
			// clamp handled in render
		}
	case "ctrl+k":
		if a.reviewExpanded != "" && a.reviewScrollY > 0 {
			a.reviewScrollY--
		}
```

- [ ] **Step 2: Update help bar**

In `internal/tui/board.go:renderBoard`, update the help text to include `V:review` and `P:past-reviews`:

```go
	help := helpStyle.Render(
		"  h/l:columns  j/k:items  v:desc  V:review  m:move  a:assign  c:create  d:delete  " +
			"p:priority  R:reviewed  P:past-reviews  Enter:detail  G:dag  D:dash  q:quit",
	)
```

- [ ] **Step 3: Run tests**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go internal/tui/board.go
git commit -m "feat: add V key for review context accordion, Ctrl+J/K scroll, P for past reviews"
```

### Task 17: Update plugin skills

**Files:**
- Modify: `obeya-plugin/skills/ob-done/SKILL.md`
- Create: `obeya-plugin/skills/ob-review/SKILL.md`

- [ ] **Step 1: Update ob-done skill**

Update the skill to guide agents to use `ob done` with review context instead of `ob move <id> done`. Include the ReviewContext JSON schema and instructions for:
- Determining confidence
- Gathering file diffs via `git diff`
- Collecting test results
- Building reproduce commands

- [ ] **Step 2: Create ob-review skill**

Create `obeya-plugin/skills/ob-review/SKILL.md` for humans to mark items reviewed/hidden.

- [ ] **Step 3: Commit**

```bash
git add obeya-plugin/
git commit -m "feat: update ob-done plugin skill for review context, add ob-review skill"
```

### Task 18: Add golden file tests for new TUI features

**Files:**
- Modify: `internal/tui/golden_test.go`
- Create: `internal/tui/testdata/*.golden` (generated)

- [ ] **Step 1: Add golden test cases**

Add test cases for:
- `TestGolden_HumanReviewColumn` — board with agent-completed items showing virtual column
- `TestGolden_AgentCardExpanded` — agent card with review context accordion expanded
- `TestGolden_PastReviewsTree` — past reviews hierarchical view

Each test should:
1. Create a board with appropriate test data (agent items with ReviewContext, Confidence, Sponsor)
2. Render at 120x40
3. Use `teatest.RequireEqualOutput` for snapshot

- [ ] **Step 2: Generate golden files**

Run: `./scripts/test.sh --update`

- [ ] **Step 3: Verify golden files**

Run: `./scripts/test.sh --golden`
Expected: All golden tests pass

- [ ] **Step 4: Run full test suite**

Run: `./scripts/test.sh`
Expected: All tests pass (existing + new)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/golden_test.go internal/tui/testdata/
git commit -m "test: add golden file snapshots for human-review column, agent cards, past reviews"
```

### Task 19: Final integration verification

- [ ] **Step 1: Run full test suite**

Run: `./scripts/test.sh`
Expected: All tests pass

- [ ] **Step 2: Manual smoke test**

```bash
go build -o ob .
./ob init --agent claude
./ob create epic "Auth Rewrite" --assign claude -d "Rewrite auth system" --sponsor niladri
./ob create task "Refactor middleware" --assign claude -d "Replace sessions" --parent 1
./ob done 2 --confidence 45 --purpose "JWT replaces cookies" --reproduce "go test ./auth/"
./ob tui
# Verify: virtual column appears, card shows AGENT badge, confidence, sponsor
# Press V to expand review context
# Press R to mark reviewed
# Press P to open past reviews
```

- [ ] **Step 3: Verify ob review works**

```bash
./ob review 2 --status reviewed
./ob review 2 --status hidden
```

- [ ] **Step 4: Final commit if any fixes needed**

---

## Summary

| Chunk | Tasks | Key Files |
|-------|-------|-----------|
| 1: Data Model & Engine | 1-6 | `types.go`, `engine.go`, `sponsor.go`, `downstream.go`, `actor.go` |
| 2: CLI Commands | 7-10 | `cmd/done.go`, `cmd/review.go`, `cmd/create.go`, `cmd/move.go` |
| 3: TUI Card Rendering | 11-12 | `card.go`, `styles.go`, `review_context.go` |
| 4: TUI Column & Pane | 13-14 | `board.go`, `past_reviews.go`, `keys.go` |
| 5: Detail & Plugin | 15-19 | `detail.go`, `detail_diffs.go`, plugin skills, golden tests |

**Total: 19 tasks, ~95 steps**

Each task is independently testable and committable. TDD throughout: write failing test → implement → verify → commit.
