# Agentic-First Human Review

**Date:** 2026-03-15
**Status:** Draft

## Problem

When agents and humans collaborate on a board, humans shift from doing work to reviewing agent work. This creates six pain points:

1. **Alert fatigue** — agents produce work faster than humans can review
2. **Context collapse** — reviewing without knowing why the agent made those choices
3. **Invisible dependencies** — approving X without knowing Y is blocked by X
4. **Accountability gaps** — "the agent did it" with no clear human ownership
5. **Confidence miscalibration** — agents report high confidence but are wrong in edge cases
6. **Interruption cost** — getting pulled into reviews mid-flow

The board must serve human judgment, not just task tracking.

## Design Principles

1. **Agent workflow unchanged** — agents still move items to review/done as today
2. **Human review is a filtered overlay** — not a new workflow state, a lens on done work
3. **Context at point of decision** — everything needed to review is on the card, no context-switching
4. **Low cognitive load** — confidence sorting does the triage, humans just work top to bottom
5. **Feature parity** — TUI and web should implement the same capabilities; web implementation is tracked separately
6. **Deterministic sponsorship** — every agent-created item has a human sponsor, resolved and stored at creation time

## Data Model Changes

**File:** `internal/domain/types.go`

### New fields on Item

```go
type Item struct {
    // ...existing fields...

    Sponsor       string         `json:"sponsor,omitempty"`        // human Identity ID
    Confidence    *int           `json:"confidence,omitempty"`     // 0-100, agent-reported; nil = unset
    ReviewContext *ReviewContext  `json:"review_context,omitempty"` // nil for human tasks
    HumanReview   *HumanReview   `json:"human_review,omitempty"`   // nil until acted on
}
```

### New structs

```go
// ReviewContext is the structured context an agent provides when completing work.
// Agents populate this when moving items to done.
type ReviewContext struct {
    Purpose      string       `json:"purpose"`                  // why the change was made
    FilesChanged []FileChange `json:"files_changed,omitempty"`  // what files were touched
    TestsWritten []TestResult `json:"tests_written,omitempty"`  // test outcomes
    Proof        []ProofItem  `json:"proof,omitempty"`          // evidence for confidence
    Reasoning    string       `json:"reasoning,omitempty"`      // agent decision rationale
    Reproduce    []string     `json:"reproduce,omitempty"`      // commands to reproduce/verify tests
}

// Note: Downstream impact (items unblocked by this item) is computed at render
// time via ResolveDownstream(), not stored on ReviewContext.

type FileChange struct {
    Path    string `json:"path"`
    Added   int    `json:"added"`
    Removed int    `json:"removed"`
}

type TestResult struct {
    Name   string `json:"name"`
    Passed bool   `json:"passed"`
}

// ProofItem is a single verification check the agent performed.
type ProofItem struct {
    Check  string `json:"check"`            // e.g. "go vet clean"
    Status string `json:"status"`           // "pass", "fail", "warn"
    Detail string `json:"detail,omitempty"` // optional explanation
}

// HumanReview tracks the human review state of an agent-completed item.
type HumanReview struct {
    Status     string    `json:"status"`                // "pending", "reviewed", "hidden"
    ReviewedBy string    `json:"reviewed_by,omitempty"` // human Identity ID
    ReviewedAt time.Time `json:"reviewed_at,omitempty"`
}
```

### Sponsor resolution (deterministic)

Sponsor is resolved at **creation time** and stored directly on every item. There is no render-time parent-chain walking — every item carries its own sponsor value.

**Invariant:** `ob init` always registers at least 1 human and 1 agent on the board. This guarantees a human exists for auto-assignment.

```go
// resolveSponsor is called inside CreateItem and CompleteItemWithContext.
// It returns the sponsor ID to store on the item, or an error.
func resolveSponsor(board Board, assignee string, explicitSponsor string, parentRef string) (string, error)
```

Resolution order:

1. If the acting identity (`board.Users[assignee]`) is human → return `""` (humans don't need sponsors, they are the owner)
2. If `explicitSponsor != ""` → validate it references a human identity in `board.Users` → return it
3. If the board has exactly 1 human → auto-assign that human as sponsor (zero friction, fully deterministic)
4. If `parentRef != ""` and the parent has a non-empty `Sponsor` → copy the parent's sponsor value onto this item (copy, not reference — the child owns its own value)
5. If none of the above resolve → **hard fail** with actionable error:
   ```
   Error: board has N humans. Specify --sponsor: alice, bob, carol
   ```

**Why copy-on-create instead of inheritance:**
- No render-time resolution needed — just read `item.Sponsor`
- Parent deletion doesn't orphan children's sponsorship
- No recursive parent-chain walking — O(1) lookup
- Every item is self-contained and queryable ("show me everything I sponsor")

**Edge cases:**

| Scenario | Behavior |
|---|---|
| 1 human on board, agent creates anything | Auto-assigned to that human, no flag needed |
| N humans, agent creates under epic with sponsor | Copied from parent, stored on child |
| N humans, agent creates orphan, no `--sponsor` | Hard fail with human names listed |
| Human creates anything | Sponsor field left empty (human IS the owner) |
| Parent deleted after child created | Child retains its own sponsor copy |
| Sponsor identity removed from board | Sponsor string persists as dangling reference; rendering displays raw ID gracefully |

### Downstream resolution

Downstream impact is computed at render time, not stored on any struct. Any item whose `BlockedBy` list includes the current item is a downstream dependency.

```
func ResolveDownstream(itemID string, board Board) []string
```

Returns item IDs (e.g. `["abc123", "def456"]`) for items blocked by the given item. Display formatting (e.g. `#31`, `#32`) is done in the TUI/web render layer, not in this function.

## Column Layout

```
BACKLOG → IN-PROGRESS → REVIEW → DONE → [HUMAN-REVIEW]
                                          (virtual column)
```

- Existing columns (including `todo` if present) are unchanged. Removing `todo` is out of scope for this spec.
- `human-review` is a virtual column. Items are not physically in this status.
- `human-review` renders items where:
  - `item.Status == "done"`
  - `item.ReviewContext != nil` (agent-completed)
  - `item.HumanReview == nil || item.HumanReview.Status != "hidden"`
- Items in `human-review` are sorted by `Confidence` ascending (lowest first)
- Human-completed items in `done` (no `ReviewContext`) do not appear in `human-review`

### Default columns

**File:** `internal/domain/types.go` (default column list)

Default columns remain unchanged:

```go
var defaultColumns = []string{"backlog", "todo", "in-progress", "review", "done"}
```

`human-review` is NOT in this list. It is rendered as a virtual column by the TUI/web, appended after the last real column.

**Column width consideration:** Adding a virtual 6th column affects `columnWidth()` (in `board.go`) which divides terminal width by `len(a.columns)`. Since `human-review` is not in `a.columns` (populated from `board.Columns`), inject it at render time:

In the `App.Init()` or board-loading path, after populating `a.columns` from the board, append a virtual column entry for `human-review`. This ensures `columnWidth()` accounts for it in width division. The virtual column uses the same `ColumnModel` wrapper but with a custom filter for its items (see Human-Review Column section).

On a 120-char terminal with 6 columns, each column is ~18 chars (near the existing minimum cap). This is acceptable.

**Empty state:** When there are no items to show in the review queue (all hidden or no agent-completed items in done), the virtual column is not rendered at all. The board columns reclaim the full terminal width. The column only appears when there is at least one visible item to review.

**Item filtering:** The existing `visibleItemsInColumn` filters by `item.Status == colName`. Since no item has `Status == "human-review"`, this function needs a special case: when the column name is `"human-review"`, use the review filter criteria (Status == "done", ReviewContext != nil, HumanReview.Status != "hidden") instead of status matching. Sort the results by Confidence ascending.

## Card Rendering Changes

**File:** `internal/tui/card.go`

### Agent badge

Cards where the assignee's `Identity.Type == "agent"` render a badge:

```
 ╭──────────────────────────────╮
 │ 🤖 AGENT  #34 Refactor auth │
 │   middleware                 │
 │ task ●●  confidence: 45% ⚠  │
 │ ▲ #10 Auth Rewrite          │
 │ @claude  sponsor: @niladri  │
 │ ⚡ unblocks 3 tasks          │
 │ ▶ description               │
 │ ▶ review context            │
 ╰──────────────────────────────╯
```

The card shows **two separate accordions** when `ReviewContext != nil`:
- `▶ description` — the existing description accordion (`v` key)
- `▶ review context` — a new accordion (`V` key, uppercase), only rendered when `item.ReviewContext != nil`

When `ReviewContext` is nil (human tasks or agent tasks not yet completed), only the description accordion is shown.

TUI uses a colored text badge `AGENT` (lipgloss Color 5, magenta) since emoji rendering is unreliable in terminals.

### Confidence indicator

Rendered on the type/priority line when `Confidence != nil`:

| Range | Color | Label |
|-------|-------|-------|
| 0-50 | Color 1 (red) | `⚠ LOW` |
| 51-75 | Color 3 (yellow) | (no label) |
| 76-100 | Color 2 (green) | `✓` |

When `Confidence == nil` (unset), no indicator is shown. This is distinct from `Confidence == 0` which means "the agent explicitly reports zero confidence."

### Sponsor line

Rendered on the meta line after assignee when `item.Sponsor != ""`:

```
@claude  sponsor: @niladri
```

No inheritance resolution at render time — the value is read directly from the item.

### Review context accordion (separate from description)

The review context accordion is a **separate, independent accordion** alongside the existing description accordion. It is not a replacement — both can be expanded simultaneously.

| Accordion | Key | Guard | Scroll keys |
|---|---|---|---|
| Description | `v` (lowercase) | `item.Description != ""` (unchanged) | `J`/`K` (existing) |
| Review context | `V` (uppercase) | `item.ReviewContext != nil` | `Ctrl+J`/`Ctrl+K` (new) |

The `▶ review context` line is only rendered on the card when `item.ReviewContext != nil`. It does not appear at all on items without review context.

**Expanded review context:**

```
 ▼ review context
 ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
  Purpose: Replace cookie-based sessions
           with JWT tokens

  Files:  auth/middleware.go  (+82 -41)
          auth/session.go    (+15 -8)

  Tests:  4 new, 2 modified (all pass)

  Reproduce:
    $ go test ./internal/auth/ -run TestJWT
    $ go test ./internal/auth/ -run TestSession
    $ go test ./internal/auth/ -v

  Proof:
    ✓ 4/4 tests pass
    ✓ go vet clean
    ✗ No edge case tests for concurrency
    ⚠ Untested: token refresh race

  ⚡ Unblocks: #31, #32, #35
```

The `Reproduce` section shows the exact commands the reviewer can copy-paste to run the agent's tests themselves. This is critical for verification — the reviewer can confirm the agent's proof claims independently.

**Scroll handling:** Each accordion tracks its own scroll offset independently. The existing `descScroll` field in the App model handles description scrolling. Add a new `reviewScroll` field for review context scrolling. `clampDescScroll` remains unchanged for description; create a parallel `clampReviewScroll` function that computes scroll bounds from the rendered review context lines.

**Render functions:** Create separate functions to respect the 50-line function limit:
- `renderReviewContext(rc *ReviewContext, width int) string` — renders the full review context block
- `reviewContextLines(rc *ReviewContext, width int) []string` — produces wrapped lines for scroll calculation

### Downstream impact

When `ResolveDownstream` returns items, a line is rendered on the card (below meta, above accordion):

```
⚡ unblocks 3 tasks
```

Full list shown in the expanded review context accordion and in the detail view.

## Human-Review Column

**File:** `internal/tui/board.go` (new rendering logic)

### Column header — visually distinct from workflow columns

The virtual column uses a different header treatment to communicate "this is a prioritized review queue, not a workflow stage":

```
 ┌─── ⚡ REVIEW QUEUE ───┐
 │  sorted by confidence  │
 │  [P] past reviews      │
 │ ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄ │
```

Key differences from regular column headers:
- **Title:** `⚡ REVIEW QUEUE (N)` instead of a status name — communicates purpose, not workflow stage
- **Subtitle:** `sorted by confidence` — self-documenting sort order
- **Past reviews link:** `[P] past reviews` — reminds user of the key binding
- **Color:** Column border and header use Color 3 (yellow/amber) instead of Color 6 (cyan) used by regular columns. This creates an immediate visual break.
- **Active state:** When cursor is in this column, border uses bright yellow (Color 11) instead of bright cyan (Color 14)

Regular columns for comparison:
```
 ┌── DONE (4) ──────────┐
 │                       │
```

The `[P]` link text is rendered in faint style. Pressing `P` from any column opens the Past Reviews pane.

### Card states and styling

| State | Styling | How to enter |
|-------|---------|--------------|
| `pending` (default) | Normal card colors | Automatic when agent completes item |
| `reviewed` | Dimmed/green-tinted border and text | Press `R` on card |
| `hidden` | Removed from column | Press `x` on card |

### Sorting

Items sorted by `Confidence` ascending. Items with `Confidence == nil` (unset) are treated as lowest confidence and appear first, followed by items with explicit `Confidence` values ascending. Within the same confidence level, sort by `UpdatedAt` ascending (oldest first).

### Key bindings (human-review column only)

These keys are **column-conditional**: they only activate when the cursor is in the virtual human-review column. In all other columns, existing bindings are preserved (e.g., `r` remains "reload board").

Implementation: in `handleBoardKey`, check `isHumanReviewColumn(app.cursorCol)` before dispatching these handlers. If false, fall through to existing key handling.

| Key | Action (human-review column) | Action (other columns) |
|-----|-----|-----|
| `R` | Mark as reviewed — sets `HumanReview.Status = "reviewed"`, changes card color | (unbound) |
| `x` | Hide from view — sets `HumanReview.Status = "hidden"`, removes from column | (unbound) |
| `P` | Open Past Reviews pane | Open Past Reviews pane (accessible globally) |
| `v` | Expand review context accordion (existing) | Expand description accordion (existing) |
| `Enter` | Open detail view (existing) | Open detail view (existing) |

Note: `R` (uppercase) is used instead of `r` to avoid conflict with the existing `r` = reload binding. `P` is globally accessible since past reviews is a cross-column concept.

## Past Reviews Pane

**File:** `internal/tui/past_reviews.go` (new file)

A full-screen overlay (like Dashboard or DAG views) showing a hierarchical tree of all reviewed/hidden items.

### Layout

```
 ╔══════════════════════════════════════════╗
 ║  Past Reviews                [Esc close] ║
 ╠══════════════════════════════════════════╣
 ║                                          ║
 ║  ▼ Epic #10 Auth Rewrite                 ║
 ║    ├── Story #15 Session management      ║
 ║    │   ├── ✓ #34 Refactor middleware     ║
 ║    │   └── ✓ #35 Update session store    ║
 ║    └── Story #16 Token validation        ║
 ║        └── ✓ #36 Add JWT validation      ║
 ║                                          ║
 ║  ▼ Epic #20 API Rate Limiting            ║
 ║    ├── ✓ #28 Rate limiting middleware    ║
 ║    └── ✓ #41 Fix README                 ║
 ║                                          ║
 ║  ▸ #44 Orphan task (no parent)           ║
 ║                                          ║
 ╚══════════════════════════════════════════╝
```

### Behavior

- Shows all items where `HumanReview != nil` (both `reviewed` and `hidden`)
- Grouped hierarchically: epics → stories → tasks. Orphan items (no parent) listed at root level.
- `j/k` to navigate, `Space` to collapse/expand epics
- `Enter` on any node (epic, story, or task) opens the existing detail view for that item
- `Esc` returns to the board

### Tree construction

```
func BuildReviewTree(board Board) []TreeNode
```

1. Collect all items where `HumanReview != nil`
2. For each, walk up `ParentID` chain to find root ancestor — include ancestors in tree even if they themselves are not reviewed (they provide structure)
3. Sort roots by `DisplayNum` ascending
4. Children sorted by `DisplayNum` ascending within each parent

Structural-only nodes (ancestors included for hierarchy but not themselves reviewed) are rendered in faint/dim style without a checkmark, visually distinguishing them from actually-reviewed items.

## Engine Changes

**File:** `internal/engine/engine.go`

### CreateItem (updated signature)

The existing `CreateItem` gains a `sponsor` parameter:

```go
func (e *Engine) CreateItem(itemType, title, parentRef, description, priority, assignee string, tags []string, sponsor string) (*Item, error)
```

Note: The current `CreateItem` does not take `userID`/`sessionID` parameters (unlike `MoveItem`, `EditItem`, etc.). This is a pre-existing gap — `buildItem` uses empty strings for history entries. Fixing this is out of scope for this spec but noted for future cleanup.

**Breaking change:** Adding `sponsor` to the signature breaks all existing call sites. Known call sites that must add `""` as the trailing sponsor argument:
- `cmd/create.go` — CLI entry point (also gains `--sponsor` flag)
- `internal/tui/app.go:677` — TUI item creation
- `internal/engine/engine_test.go` — all `CreateItem` calls in engine tests
- `test/integration_test.go` — integration test calls
- `internal/store/cloud_store_integration_test.go` — cloud store test calls

All non-CLI call sites pass `""` for sponsor (the CLI is the only place the flag is exposed).

Note: `CloudClient.CreateItem` takes a `*domain.Item` directly (not the engine method). Adding `Sponsor` to `domain.Item` is a zero-value-safe change (empty string), so cloud sync won't break. However, the cloud sync path must be verified to correctly round-trip the `Sponsor` field through serialization.

Sponsor is resolved deterministically via `resolveSponsor(board, assignee, sponsor, parentRef)` (see Sponsor resolution section). The resolved value is stored directly on the new item's `Sponsor` field.

```go
resolvedSponsor, err := resolveSponsor(board, assignee, sponsor, parentRef)
if err != nil {
    return nil, err  // hard fail with actionable message
}
item.Sponsor = resolvedSponsor
```

New `--sponsor` flag on `ob create` commands. Optional when the board has exactly 1 human (auto-assigned). Required when the board has multiple humans and the item has no parent with a sponsor.

### ReviewItem (new operation)

```go
func (e *Engine) ReviewItem(ref string, status string, userID string, sessionID string) error
```

- Validates `status` is `"reviewed"` or `"hidden"`
- Validates `userID` references a human identity in `board.Users` (agents cannot review their own work)
- Sets `item.HumanReview = &HumanReview{Status: status, ReviewedBy: userID, ReviewedAt: now}`
- Appends to item history via `appendHistory(item, userID, sessionID, "human-review", status)`

Follows the same `(userID, sessionID)` pattern as all existing engine mutations.

### CompleteItemWithContext (new operation)

```go
func (e *Engine) CompleteItemWithContext(ref string, ctx ReviewContext, confidence int, userID string, sessionID string) error
```

- Moves item to `done` (sets `item.Status = "done"`)
- Sets `item.ReviewContext = &ctx`
- Sets `item.Confidence = &confidence`
- Sets `item.HumanReview = &HumanReview{Status: "pending"}`
- Appends to item history via `appendHistory(item, userID, sessionID, "complete-with-context", ctx.Purpose)`

**Pre-conditions:** None on current status — mirrors `MoveItem` which allows moving from any status. If the item is already in `done`, calling this is idempotent for the status change but will overwrite `ReviewContext`, `Confidence`, and reset `HumanReview` to pending.

**`--confidence` is required** on `ob done`. The engine signature takes `confidence int`, but the domain model uses `*int`. The CLI must require the flag and convert to pointer. If no confidence is provided, the CLI errors: `"--confidence is required. Provide 0-100."` This eliminates the zero-value ambiguity.

Follows the same `(userID, sessionID)` pattern as all existing engine mutations. This is the operation agents call instead of a plain `MoveItem` when completing work. Humans can also call it if they want to provide review context on their own work.

## CLI Changes

### `ob review <ref> --status reviewed|hidden`

**File:** `cmd/review.go` (new file)

Marks an item's human review status. Only available to human identities.

```
ob review 34 --status reviewed
ob review 34 --status hidden
```

### `ob done <ref>` (new command)

**File:** `cmd/done.go` (new file)

This is a new CLI command (distinct from `ob move <ref> done`). It wraps `CompleteItemWithContext` and is the agent-facing way to complete work with review context. `ob move <ref> done` continues to work for simple moves without review context.

Agents use this to complete work with context:

```
ob done 34 \
  --confidence 45 \
  --purpose "Replace cookie sessions with JWT" \
  --files "auth/middleware.go:+82-41,auth/session.go:+15-8" \
  --tests "TestJWTValidation:pass,TestSessionMigration:pass" \
  --reproduce "go test ./internal/auth/ -run TestJWT" \
  --reproduce "go test ./internal/auth/ -v" \
  --proof "go vet clean:pass,edge case tests:fail:No concurrent session tests" \
  --reasoning "JWT chosen over opaque tokens for debuggability"
```

Alternatively (and preferably for agents), review context can be provided as JSON via stdin:

```
echo '{"purpose":"...","files_changed":[...]}' | ob done 34 --confidence 45 --context-stdin
```

The `--context-stdin` JSON path is the primary agent interface. The inline flags exist for manual use and debugging.

**Relationship to `ob move`:** `ob move <ref> done` continues to work as before (moves item to done without review context). When the acting identity is an agent, `ob move <ref> done` should emit a warning: `"Warning: use 'ob done <ref>' to include review context for human review."`

To determine whether the actor is an agent, `move.go` must: (1) load the board via the store, (2) call `board.ResolveUserID(getUserID())` to fuzzy-match the raw username against registered identities, (3) look up the resolved identity and check `Identity.Type == "agent"`.

Extract this into a shared helper in `internal/engine/`:

```go
func (e *Engine) ResolveActorType(rawUserID string) (string, error)
```

Returns `"human"`, `"agent"`, or `""` (unknown). When the user is not found in `board.Users` (common for unregistered OS usernames), return `"human"` as the default — unregistered users are assumed human. The warning is only emitted when the resolved type is explicitly `"agent"`.

This helper is needed in multiple commands (`move`, `review`, `create`).

### `ob create` with `--sponsor`

**File:** `cmd/create.go`

New optional flag. Required when agent creates an item with no parent:

```
ob create task "Fix bug" --assign claude --sponsor niladri
```

## Plugin Changes

**File:** `obeya-plugin/` (SKILL.md files — Claude Code slash command definitions)

The Obeya plugin consists of markdown-based skill definitions (SKILL.md files), not Go code. Changes are to the skill instruction text that guides agent behavior.

### `/ob:done` (updated skill)

**File:** `obeya-plugin/skills/done/SKILL.md`

Update the skill instructions to guide the agent to:

1. Determine confidence level (0-100) based on test results, code complexity, and edge case coverage
2. Gather files changed from the current session (git diff or session context)
3. Collect test results (which tests were added/modified, pass/fail status)
4. Construct the review context JSON and pipe it to `ob done <ref> --confidence <N> --context-stdin`

The skill text should include the JSON schema for `ReviewContext` so agents know the expected format.

### `/ob:review` (new skill)

**File:** `obeya-plugin/skills/review/SKILL.md`

New skill that instructs the user (human) to mark items as reviewed or hidden:

```
ob review <ref> --status reviewed
ob review <ref> --status hidden
```

## TUI State Machine

**File:** `internal/tui/app.go`

### New state

Appended to the existing iota block in `keys.go`:

```go
statePastReviews // after stateDAG, added via iota continuation
```

### State transitions

- From `stateBoard` (cursor in human-review column): `P` → `statePastReviews`
- From `statePastReviews`: `Enter` → `stateDetail` (for selected item)
- From `statePastReviews`: `Esc` → `stateBoard`

### Key routing additions

**TUI identity resolution:** The `App` struct must gain `userID` and `sessionID` fields, initialized at startup. `userID` is resolved from `os/user.Current().Username` (matching how CLI commands use `getUserID()`). `sessionID` is generated once via `domain.GenerateID()` at TUI launch. These are passed to all engine-mutating operations. This is a pre-existing gap (current TUI operations like `CreateItem` pass empty strings) — this spec fixes it for `ReviewItem` and new operations.

**Important:** `ReviewItem` validates that `userID` references a human identity. The TUI resolves the OS username against `board.Users` via `board.ResolveUserID(app.userID)`. If the user is not registered on the board, `ReviewItem` should treat them as human (unregistered users are assumed human per the `ResolveActorType` convention). The validation only rejects explicitly registered agent identities.

In the board state:
- When cursor is in the virtual human-review column:
  - `R` → call `engine.ReviewItem(ref, "reviewed", app.userID, app.sessionID)`
  - `x` → call `engine.ReviewItem(ref, "hidden", app.userID, app.sessionID)`
- From any column:
  - `P` → switch to `statePastReviews`

## Testing

### Unit tests

**File:** `internal/engine/engine_test.go`

- `TestCompleteItemWithContext` — verify ReviewContext and Confidence are set
- `TestCompleteItemWithContext_SetsHumanReviewPending` — verify HumanReview initialized
- `TestReviewItem_Reviewed` — verify status change and history entry
- `TestReviewItem_Hidden` — verify status change
- `TestReviewItem_AgentCannotReview` — verify agent identity is rejected
- `TestResolveSponsor_Explicit` — explicit --sponsor flag stored on item
- `TestResolveSponsor_AutoAssignSingleHuman` — 1 human on board, auto-assigned without flag
- `TestResolveSponsor_CopiedFromParent` — sponsor copied from parent, stored on child
- `TestResolveSponsor_MultipleHumansNoSponsor` — N humans, no parent, no flag = hard fail with names
- `TestResolveSponsor_HumanCreatesItem` — human actor, sponsor field left empty
- `TestResolveDownstream` — verify blocked items are returned
- `TestCompleteItemWithContext_HumanIdentityAllowed` — humans can also provide review context (not agent-only)

### TUI tests

**File:** `internal/tui/app_test.go`

- `TestHumanReviewColumn_Renders` — virtual column appears after done
- `TestHumanReviewColumn_SortedByConfidence` — lowest confidence first
- `TestHumanReviewColumn_ReviewedCardColorChange` — color changes after `R` key
- `TestHumanReviewColumn_HiddenCardDisappears` — card removed after `x` key
- `TestPastReviews_HierarchicalTree` — correct tree structure
- `TestPastReviews_EnterOpensDetail` — detail view opens on enter
- `TestAgentBadge_Renders` — agent badge visible on agent-assigned cards
- `TestConfidenceIndicator_Colors` — correct colors for confidence ranges

### Golden file tests

**File:** `internal/tui/golden_test.go`

New golden snapshots:
- `testdata/human-review-column.golden` — human-review column with mixed confidence items
- `testdata/agent-card-expanded.golden` — agent card with review context expanded
- `testdata/past-reviews-tree.golden` — past reviews hierarchical view

## Migration

Existing boards have no `Sponsor`, `Confidence`, `ReviewContext`, or `HumanReview` fields. Because all new fields use `omitempty` JSON tags and are pointer/zero-value types, existing `board.json` files deserialize cleanly with nil/zero values. No migration script needed.

Items already in `done` without `ReviewContext` are treated as human-completed and do not appear in the human-review column.

## Out of Scope

- Push notifications / batch review alerts (deferred to notification phase)
- Historical confidence accuracy tracking (future enhancement)
- Web UI implementation details (follows same spec, separate implementation ticket)
- Agent-to-agent review (only human review is in scope)
