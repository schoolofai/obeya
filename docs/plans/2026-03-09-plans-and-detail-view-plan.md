# Plans & Full-Screen Detail View — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add plan documents as linkable entities to Obeya items, enrich task descriptions via `--body-file`, and replace the TUI detail overlay with a full-screen tabbed view.

**Architecture:** Plans are stored in `Board.Plans` map alongside items, sharing the display number counter. The engine gets plan CRUD + link/unlink methods. A new `ob plan` CLI subcommand group wraps these. The TUI detail view becomes full-screen with Fields/Plan/History tabs.

**Tech Stack:** Go, Cobra (CLI), Bubble Tea (TUI), Lipgloss (styling)

**Design Doc:** `docs/plans/2026-03-09-plans-and-detail-view-design.md`

**Obeya Board:** Epic #9, Stories #10-#15

---

## Task 1: Plan Data Model

**Obeya Story:** #10 (Plan data model and board storage)

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/domain/id.go`
- Test: `internal/domain/types_test.go`

**Step 1: Write the failing test**

Add to `internal/domain/types_test.go`:

```go
func TestBoard_PlansMapInitialized(t *testing.T) {
	board := NewBoard("test")
	if board.Plans == nil {
		t.Fatal("expected Plans map to be initialized")
	}
	if board.NextDisplay != 1 {
		t.Errorf("expected NextDisplay=1, got %d", board.NextDisplay)
	}
}

func TestPlan_Fields(t *testing.T) {
	plan := &Plan{
		ID:          "abc12345",
		DisplayNum:  1,
		Title:       "Test Plan",
		Content:     "# My Plan\n\nSome content.",
		SourceFile:  "docs/plans/test.md",
		LinkedItems: []string{"item1", "item2"},
	}
	if plan.Title != "Test Plan" {
		t.Errorf("expected title 'Test Plan', got %q", plan.Title)
	}
	if len(plan.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(plan.LinkedItems))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -run TestBoard_PlansMapInitialized -v`
Expected: FAIL — `Plan` type not defined

**Step 3: Add Plan struct and update Board**

Add to `internal/domain/types.go` after the `Identity` struct:

```go
type Plan struct {
	ID          string    `json:"id"`
	DisplayNum  int       `json:"display_num"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	SourceFile  string    `json:"source_file,omitempty"`
	LinkedItems []string  `json:"linked_items"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

Add `Plans` field to `Board` struct (after `Users`):

```go
Plans map[string]*Plan `json:"plans"`
```

Update `NewBoardWithColumns` to initialize `Plans`:

```go
Plans: make(map[string]*Plan),
```

**Step 4: Add ResolvePlanID to id.go**

Add to `internal/domain/id.go`:

```go
func (b *Board) ResolvePlanID(ref string) (string, error) {
	// Try display number first
	if num, err := strconv.Atoi(ref); err == nil {
		if id, ok := b.DisplayMap[num]; ok {
			if _, isPlan := b.Plans[id]; isPlan {
				return id, nil
			}
			return "", fmt.Errorf("item #%d is not a plan", num)
		}
	}

	// Try exact ID match
	if _, ok := b.Plans[ref]; ok {
		return ref, nil
	}

	// Try prefix match
	var matches []string
	for id := range b.Plans {
		if len(id) >= len(ref) && id[:len(ref)] == ref {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("plan not found: %s", ref)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous plan reference %q — matches %d plans", ref, len(matches))
	}
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/domain/ -v`
Expected: PASS

**Step 6: Commit and update board**

```bash
git add internal/domain/types.go internal/domain/id.go internal/domain/types_test.go
git commit -m "feat(domain): add Plan struct, Board.Plans map, ResolvePlanID"
~/bin/ob move 10 in-progress
```

---

## Task 2: Plan Engine Methods

**Obeya Story:** #11 (Plan engine methods)

**Files:**
- Create: `internal/engine/plan.go`
- Test: `internal/engine/engine_test.go` (add plan tests)

**Step 1: Write failing tests**

Add to `internal/engine/engine_test.go`:

```go
func TestCreatePlan(t *testing.T) {
	eng := setupEngine(t)
	plan, err := eng.CreatePlan("My Plan", "# Content\nDetails here.", "")
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	if plan.Title != "My Plan" {
		t.Errorf("expected title 'My Plan', got %q", plan.Title)
	}
	if plan.DisplayNum < 1 {
		t.Error("expected positive display number")
	}
}

func TestImportPlan(t *testing.T) {
	eng := setupEngine(t)

	// Create items to link
	item1, _ := eng.CreateItem("task", "Task A", "", "", "medium", "", nil)
	item2, _ := eng.CreateItem("task", "Task B", "", "", "medium", "", nil)

	content := "# Import Test\nPlan body."
	plan, err := eng.ImportPlan(content, "docs/test.md", []string{
		fmt.Sprintf("%d", item1.DisplayNum),
		fmt.Sprintf("%d", item2.DisplayNum),
	})
	if err != nil {
		t.Fatalf("ImportPlan failed: %v", err)
	}
	if plan.Title != "Import Test" {
		t.Errorf("expected title 'Import Test', got %q", plan.Title)
	}
	if len(plan.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(plan.LinkedItems))
	}
}

func TestLinkUnlinkPlan(t *testing.T) {
	eng := setupEngine(t)
	plan, _ := eng.CreatePlan("Plan", "content", "")
	item, _ := eng.CreateItem("task", "T", "", "", "medium", "", nil)

	if err := eng.LinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", item.DisplayNum)}); err != nil {
		t.Fatalf("LinkPlan failed: %v", err)
	}

	p, _ := eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(p.LinkedItems) != 1 {
		t.Fatalf("expected 1 linked item, got %d", len(p.LinkedItems))
	}

	if err := eng.UnlinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", item.DisplayNum)}); err != nil {
		t.Fatalf("UnlinkPlan failed: %v", err)
	}

	p, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(p.LinkedItems) != 0 {
		t.Fatalf("expected 0 linked items, got %d", len(p.LinkedItems))
	}
}

func TestDeletePlan(t *testing.T) {
	eng := setupEngine(t)
	plan, _ := eng.CreatePlan("Deletable", "content", "")

	if err := eng.DeletePlan(fmt.Sprintf("%d", plan.DisplayNum)); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}

	_, err := eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if err == nil {
		t.Error("expected error showing deleted plan")
	}
}

func TestUpdatePlan(t *testing.T) {
	eng := setupEngine(t)
	plan, _ := eng.CreatePlan("Original", "old content", "")

	if err := eng.UpdatePlan(fmt.Sprintf("%d", plan.DisplayNum), "Updated", "new content"); err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}

	p, _ := eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if p.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", p.Title)
	}
	if p.Content != "new content" {
		t.Errorf("expected 'new content', got %q", p.Content)
	}
}

func TestListPlans(t *testing.T) {
	eng := setupEngine(t)
	eng.CreatePlan("Plan A", "a", "")
	eng.CreatePlan("Plan B", "b", "")

	plans, err := eng.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestCreatePlan -v`
Expected: FAIL — `CreatePlan` not defined

**Step 3: Create plan.go with all engine methods**

Create `internal/engine/plan.go`:

```go
package engine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

func (e *Engine) CreatePlan(title, content, sourceFile string) (*domain.Plan, error) {
	if title == "" {
		return nil, fmt.Errorf("plan title is required")
	}

	var created *domain.Plan
	err := e.store.Transaction(func(board *domain.Board) error {
		if board.Plans == nil {
			board.Plans = make(map[string]*domain.Plan)
		}
		now := time.Now()
		plan := &domain.Plan{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Title:       title,
			Content:     content,
			SourceFile:  sourceFile,
			LinkedItems: []string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		board.Plans[plan.ID] = plan
		board.DisplayMap[plan.DisplayNum] = plan.ID
		board.NextDisplay++
		created = plan
		return nil
	})
	return created, err
}

func (e *Engine) ImportPlan(content, sourceFile string, linkRefs []string) (*domain.Plan, error) {
	title := extractTitleFromMarkdown(content)
	if title == "" {
		title = "Untitled Plan"
	}

	var created *domain.Plan
	err := e.store.Transaction(func(board *domain.Board) error {
		if board.Plans == nil {
			board.Plans = make(map[string]*domain.Plan)
		}

		linkedIDs, err := resolveItemRefs(board, linkRefs)
		if err != nil {
			return err
		}

		now := time.Now()
		plan := &domain.Plan{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Title:       title,
			Content:     content,
			SourceFile:  sourceFile,
			LinkedItems: linkedIDs,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		board.Plans[plan.ID] = plan
		board.DisplayMap[plan.DisplayNum] = plan.ID
		board.NextDisplay++
		created = plan
		return nil
	})
	return created, err
}

func (e *Engine) UpdatePlan(ref, title, content string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}
		plan := board.Plans[id]
		if title != "" {
			plan.Title = title
		}
		if content != "" {
			plan.Content = content
		}
		plan.UpdatedAt = time.Now()
		return nil
	})
}

func (e *Engine) DeletePlan(ref string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}
		plan := board.Plans[id]
		delete(board.Plans, id)
		delete(board.DisplayMap, plan.DisplayNum)
		return nil
	})
}

func (e *Engine) ShowPlan(ref string) (*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}
	id, err := board.ResolvePlanID(ref)
	if err != nil {
		return nil, err
	}
	return board.Plans[id], nil
}

func (e *Engine) ListPlans() ([]*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}
	plans := make([]*domain.Plan, 0, len(board.Plans))
	for _, p := range board.Plans {
		plans = append(plans, p)
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].DisplayNum < plans[j].DisplayNum
	})
	return plans, nil
}

func (e *Engine) LinkPlan(ref string, itemRefs []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}
		plan := board.Plans[id]
		newIDs, err := resolveItemRefs(board, itemRefs)
		if err != nil {
			return err
		}
		for _, itemID := range newIDs {
			if !containsString(plan.LinkedItems, itemID) {
				plan.LinkedItems = append(plan.LinkedItems, itemID)
			}
		}
		plan.UpdatedAt = time.Now()
		return nil
	})
}

func (e *Engine) UnlinkPlan(ref string, itemRefs []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}
		plan := board.Plans[id]
		removeIDs, err := resolveItemRefs(board, itemRefs)
		if err != nil {
			return err
		}
		for _, rid := range removeIDs {
			filtered, _ := removeString(plan.LinkedItems, rid)
			plan.LinkedItems = filtered
		}
		plan.UpdatedAt = time.Now()
		return nil
	})
}

// PlansForItem returns all plans linked to a given item ID.
func (e *Engine) PlansForItem(itemID string) ([]*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}
	var plans []*domain.Plan
	for _, p := range board.Plans {
		if containsString(p.LinkedItems, itemID) {
			plans = append(plans, p)
		}
	}
	return plans, nil
}

func extractTitleFromMarkdown(content string) string {
	lines := strings.SplitN(content, "\n", 10)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return ""
}

func resolveItemRefs(board *domain.Board, refs []string) ([]string, error) {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		id, err := board.ResolveID(ref)
		if err != nil {
			return nil, fmt.Errorf("cannot link to %s: %w", ref, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: PASS

**Step 5: Commit and update board**

```bash
git add internal/engine/plan.go internal/engine/engine_test.go
git commit -m "feat(engine): add plan CRUD, link/unlink, and import methods"
~/bin/ob move 11 in-progress
```

---

## Task 3: `--body-file` Flag

**Obeya Story:** #13 (Body-file flag)

**Files:**
- Modify: `cmd/create.go`
- Modify: `cmd/edit.go`

**Step 1: Add `--body-file` to create.go**

Add a new flag variable at the top of `cmd/create.go`:

```go
var createBodyFile string
```

In the `RunE` function, after the title assignment, add body-file handling:

```go
// After: title := args[1]
if createBodyFile != "" && createDesc != "" {
    return fmt.Errorf("--body-file and -d/--description are mutually exclusive")
}
if createBodyFile != "" {
    data, err := os.ReadFile(createBodyFile)
    if err != nil {
        return fmt.Errorf("failed to read body file %q: %w", createBodyFile, err)
    }
    createDesc = string(data)
}
```

Add the flag in `init()`:

```go
createCmd.Flags().StringVar(&createBodyFile, "body-file", "", "read description from file")
```

Add `"os"` to imports.

**Step 2: Add `--body-file` to edit.go**

Add body-file flag and handling. In the `Run` function, after getting desc:

```go
bodyFile, _ := cmd.Flags().GetString("body-file")
if bodyFile != "" && desc != "" {
    fmt.Fprintf(os.Stderr, "Error: --body-file and -d/--description are mutually exclusive\n")
    os.Exit(1)
}
if bodyFile != "" {
    data, err := os.ReadFile(bodyFile)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: failed to read body file %q: %v\n", bodyFile, err)
        os.Exit(1)
    }
    desc = string(data)
}
```

Add the flag in `init()`:

```go
editCmd.Flags().String("body-file", "", "read description from file")
```

**Step 3: Build and verify**

Run: `go build -o /tmp/ob-test .`
Expected: Builds successfully

**Step 4: Manual test**

```bash
echo "# Detailed Description\n\nThis task involves..." > /tmp/test-desc.md
/tmp/ob-test create task "Test Body File" --body-file /tmp/test-desc.md
/tmp/ob-test show 1 --format json | grep description
```

Expected: Description contains the file content.

**Step 5: Commit and update board**

```bash
git add cmd/create.go cmd/edit.go
git commit -m "feat(cli): add --body-file flag to create and edit commands"
~/bin/ob move 13 in-progress
~/bin/ob move 13 done
```

---

## Task 4: `ob plan` CLI Subcommands

**Obeya Story:** #12 (ob plan CLI subcommands)

**Files:**
- Create: `cmd/plan.go`

**Step 1: Create cmd/plan.go**

Create `cmd/plan.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage plan documents",
}

var planCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an empty plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			return fmt.Errorf("--title is required")
		}
		eng, err := getEngine()
		if err != nil {
			return err
		}
		plan, err := eng.CreatePlan(title, "", "")
		if err != nil {
			return err
		}
		return printPlanCreated(plan)
	},
}

var planImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a plan from a markdown file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read plan file %q: %w", filePath, err)
		}
		content := string(data)

		linkFlag, _ := cmd.Flags().GetString("link")
		var linkRefs []string
		if linkFlag != "" {
			linkRefs = strings.Split(linkFlag, ",")
			for i := range linkRefs {
				linkRefs[i] = strings.TrimSpace(linkRefs[i])
			}
		}

		titleOverride, _ := cmd.Flags().GetString("title")

		eng, err := getEngine()
		if err != nil {
			return err
		}
		plan, err := eng.ImportPlan(content, filePath, linkRefs)
		if err != nil {
			return err
		}
		if titleOverride != "" {
			if err := eng.UpdatePlan(plan.ID, titleOverride, ""); err != nil {
				return err
			}
			plan.Title = titleOverride
		}
		return printPlanCreated(plan)
	},
}

var planUpdateCmd = &cobra.Command{
	Use:   "update <plan-id> [file]",
	Short: "Update plan content or title",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		title, _ := cmd.Flags().GetString("title")
		content := ""
		if len(args) == 2 {
			data, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read file %q: %w", args[1], err)
			}
			content = string(data)
		}
		if title == "" && content == "" {
			return fmt.Errorf("specify a file to update content or --title to update title")
		}
		eng, err := getEngine()
		if err != nil {
			return err
		}
		if err := eng.UpdatePlan(ref, title, content); err != nil {
			return err
		}
		fmt.Printf("Updated plan #%s\n", ref)
		return nil
	},
}

var planShowCmd = &cobra.Command{
	Use:   "show <plan-id>",
	Short: "Show plan details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := getEngine()
		if err != nil {
			return err
		}
		plan, err := eng.ShowPlan(args[0])
		if err != nil {
			return err
		}
		if flagFormat == "json" {
			data, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		printPlanText(plan)
		return nil
	},
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := getEngine()
		if err != nil {
			return err
		}
		plans, err := eng.ListPlans()
		if err != nil {
			return err
		}
		if flagFormat == "json" {
			data, err := json.MarshalIndent(plans, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		for _, p := range plans {
			fmt.Printf("#%-4d %s (%d linked items)\n", p.DisplayNum, p.Title, len(p.LinkedItems))
		}
		return nil
	},
}

var planLinkCmd = &cobra.Command{
	Use:   "link <plan-id>",
	Short: "Link items to a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toFlag, _ := cmd.Flags().GetString("to")
		if toFlag == "" {
			return fmt.Errorf("--to is required (comma-separated item IDs)")
		}
		refs := strings.Split(toFlag, ",")
		for i := range refs {
			refs[i] = strings.TrimSpace(refs[i])
		}
		eng, err := getEngine()
		if err != nil {
			return err
		}
		if err := eng.LinkPlan(args[0], refs); err != nil {
			return err
		}
		fmt.Printf("Linked %d items to plan #%s\n", len(refs), args[0])
		return nil
	},
}

var planUnlinkCmd = &cobra.Command{
	Use:   "unlink <plan-id>",
	Short: "Unlink items from a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromFlag, _ := cmd.Flags().GetString("from")
		if fromFlag == "" {
			return fmt.Errorf("--from is required (comma-separated item IDs)")
		}
		refs := strings.Split(fromFlag, ",")
		for i := range refs {
			refs[i] = strings.TrimSpace(refs[i])
		}
		eng, err := getEngine()
		if err != nil {
			return err
		}
		if err := eng.UnlinkPlan(args[0], refs); err != nil {
			return err
		}
		fmt.Printf("Unlinked %d items from plan #%s\n", len(refs), args[0])
		return nil
	},
}

var planDeleteCmd = &cobra.Command{
	Use:   "delete <plan-id>",
	Short: "Delete a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := getEngine()
		if err != nil {
			return err
		}
		if err := eng.DeletePlan(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted plan #%s\n", args[0])
		return nil
	},
}

func init() {
	planCreateCmd.Flags().String("title", "", "plan title (required)")
	planImportCmd.Flags().String("link", "", "comma-separated item IDs to link")
	planImportCmd.Flags().String("title", "", "override title from file")
	planUpdateCmd.Flags().String("title", "", "new title")
	planLinkCmd.Flags().String("to", "", "comma-separated item IDs to link")
	planUnlinkCmd.Flags().String("from", "", "comma-separated item IDs to unlink")

	planCmd.AddCommand(planCreateCmd, planImportCmd, planUpdateCmd, planShowCmd, planListCmd, planLinkCmd, planUnlinkCmd, planDeleteCmd)
	rootCmd.AddCommand(planCmd)
}

func printPlanCreated(plan *domain.Plan) error {
	if flagFormat == "json" {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Created plan #%d [%s]: %s\n", plan.DisplayNum, plan.ID[:6], plan.Title)
	if len(plan.LinkedItems) > 0 {
		fmt.Printf("  Linked to %d items\n", len(plan.LinkedItems))
	}
	return nil
}

func printPlanText(plan *domain.Plan) {
	fmt.Printf("Plan #%d: %s\n", plan.DisplayNum, plan.Title)
	if plan.SourceFile != "" {
		fmt.Printf("  Source: %s\n", plan.SourceFile)
	}
	fmt.Printf("  Linked items: %d\n", len(plan.LinkedItems))
	fmt.Printf("  Created: %s\n", plan.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("  Updated: %s\n", plan.UpdatedAt.Format("2006-01-02 15:04"))
	if plan.Content != "" {
		fmt.Printf("\n%s\n", plan.Content)
	}
}
```

Note: Add `"github.com/niladribose/obeya/internal/domain"` to the imports for `printPlanCreated`.

**Step 2: Build and verify**

Run: `go build -o /tmp/ob-test .`
Expected: Builds successfully

**Step 3: Manual smoke test**

```bash
cd /tmp && rm -rf plan-test && mkdir plan-test && cd plan-test
/tmp/ob-test init plan-board
/tmp/ob-test create epic "Auth System"
/tmp/ob-test create task "JWT" -p 1
echo "# Auth Plan\n\nBuild JWT auth." > /tmp/auth-plan.md
/tmp/ob-test plan import /tmp/auth-plan.md --link 1,2
/tmp/ob-test plan list
/tmp/ob-test plan show 3
/tmp/ob-test plan link 3 --to 2
/tmp/ob-test plan unlink 3 --from 2
/tmp/ob-test plan delete 3
```

**Step 4: Commit and update board**

```bash
cd /Users/niladribose/code/obeya
git add cmd/plan.go
git commit -m "feat(cli): add ob plan subcommand group (CRUD + link/unlink)"
~/bin/ob move 12 in-progress
~/bin/ob move 10 done
~/bin/ob move 11 done
~/bin/ob move 12 done
```

---

## Task 5: Full-Screen Tabbed Detail View

**Obeya Story:** #14 (Full-screen tabbed detail view)

**Files:**
- Rewrite: `internal/tui/detail.go`
- Modify: `internal/tui/keys.go` (add tab state)
- Modify: `internal/tui/app.go` (pass board + plans, handle tab switching)

**Step 1: Add detail tab state to keys.go**

Add to `internal/tui/keys.go`:

```go
type detailTab int

const (
	tabFields detailTab = iota
	tabPlan
	tabHistory
)
```

**Step 2: Rewrite detail.go for full-screen tabbed view**

Rewrite `internal/tui/detail.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
)

type DetailModel struct {
	item      *domain.Item
	board     *domain.Board
	plans     []*domain.Plan
	activeTab detailTab
	scrollY   int
	width     int
	height    int
}

func newDetailModel(item *domain.Item, board ...*domain.Board) DetailModel {
	d := DetailModel{item: item, activeTab: tabFields}
	if len(board) > 0 {
		d.board = board[0]
		d.plans = plansForItem(board[0], item.ID)
	}
	return d
}

func (d *DetailModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DetailModel) NextTab() {
	d.activeTab = (d.activeTab + 1) % 3
	d.scrollY = 0
}

func (d *DetailModel) PrevTab() {
	d.activeTab = (d.activeTab + 2) % 3
	d.scrollY = 0
}

func (d *DetailModel) ScrollDown() {
	d.scrollY++
}

func (d *DetailModel) ScrollUp() {
	if d.scrollY > 0 {
		d.scrollY--
	}
}

func (d DetailModel) View() string {
	if d.item == nil {
		return "No item selected"
	}

	w := d.width
	if w < 40 {
		w = 80
	}
	h := d.height
	if h < 10 {
		h = 24
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(" #%d %s", d.item.DisplayNum, d.item.Title)
	headerStyle := typeStyle(string(d.item.Type)).Bold(true)
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	// Tab bar
	sb.WriteString(d.renderTabBar())
	sb.WriteString("\n\n")

	// Tab content
	var content string
	switch d.activeTab {
	case tabFields:
		content = d.renderFieldsTab()
	case tabPlan:
		content = d.renderPlanTab()
	case tabHistory:
		content = d.renderHistoryTab()
	}

	// Apply scroll
	lines := strings.Split(content, "\n")
	viewH := h - 6 // header + tabs + help
	if d.scrollY >= len(lines) {
		d.scrollY = max(0, len(lines)-1)
	}
	end := d.scrollY + viewH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[d.scrollY:end]
	sb.WriteString(strings.Join(visible, "\n"))

	// Help bar
	sb.WriteString("\n\n")
	help := "Tab/Shift-Tab:switch  j/k:scroll  m:move  a:assign  p:priority  Esc:back"
	sb.WriteString(helpStyle.Render(help))

	// Full-screen border
	contentW := w - 4
	if contentW < 40 {
		contentW = 40
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2).
		Width(contentW).
		Height(h - 2).
		Render(sb.String())
}

func (d DetailModel) renderTabBar() string {
	tabs := []string{"Fields", "Plan", "History"}
	var parts []string
	for i, name := range tabs {
		if detailTab(i) == d.activeTab {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).Underline(true).
				Foreground(lipgloss.Color("14")).
				Render("["+name+"]"))
		} else {
			parts = append(parts, helpStyle.Render(" "+name+" "))
		}
	}
	return strings.Join(parts, "  ")
}

func (d DetailModel) renderFieldsTab() string {
	var sb strings.Builder

	writeDetailField(&sb, "Type", string(d.item.Type))
	writeDetailField(&sb, "Status", d.item.Status)
	writeDetailField(&sb, "Priority", priorityIndicator(string(d.item.Priority))+" "+string(d.item.Priority))

	if d.item.Assignee != "" {
		name := d.resolveUserName(d.item.Assignee)
		writeDetailField(&sb, "Assignee", assigneeStyle.Render("@"+name))
	}

	if d.item.ParentID != "" && d.board != nil {
		if parent, ok := d.board.Items[d.item.ParentID]; ok {
			writeDetailField(&sb, "Parent", fmt.Sprintf("#%d %s", parent.DisplayNum, parent.Title))
		}
	}

	if len(d.item.Tags) > 0 {
		writeDetailField(&sb, "Tags", strings.Join(d.item.Tags, ", "))
	}

	if len(d.item.BlockedBy) > 0 {
		refs := formatBlockedRefs(d.board, d.item.BlockedBy)
		writeDetailField(&sb, "Blocked", blockedStyle.Render(refs))
	} else {
		writeDetailField(&sb, "Blocked", "—")
	}

	if d.item.Description != "" {
		sb.WriteString("\n  Description:\n")
		for _, line := range strings.Split(d.item.Description, "\n") {
			sb.WriteString("    " + line + "\n")
		}
	}

	if d.board != nil {
		children := findChildren(d.board, d.item.ID)
		if len(children) > 0 {
			sb.WriteString("\n  Children:\n")
			for _, c := range children {
				sb.WriteString(fmt.Sprintf("    #%-4d %-8s %-12s %s\n",
					c.DisplayNum, string(c.Type), c.Status, c.Title))
			}
		}
	}

	return sb.String()
}

func (d DetailModel) renderPlanTab() string {
	if len(d.plans) == 0 {
		return "  No plan linked."
	}

	var sb strings.Builder
	for i, plan := range d.plans {
		if i > 0 {
			sb.WriteString("\n" + strings.Repeat("─", 40) + "\n\n")
		}
		header := fmt.Sprintf("  Plan #%d: %s", plan.DisplayNum, plan.Title)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header))
		sb.WriteString("\n\n")
		// Indent plan content
		for _, line := range strings.Split(plan.Content, "\n") {
			sb.WriteString("  " + line + "\n")
		}
	}
	return sb.String()
}

func (d DetailModel) renderHistoryTab() string {
	if len(d.item.History) == 0 {
		return "  No history."
	}

	var sb strings.Builder
	for _, h := range d.item.History {
		ts := h.Timestamp.Format("2006-01-02 15:04")
		sb.WriteString(fmt.Sprintf("  %s  %-10s  %s\n", ts, h.Action, h.Detail))
	}
	return sb.String()
}

func (d DetailModel) resolveUserName(userID string) string {
	if d.board != nil && d.board.Users != nil {
		if u, ok := d.board.Users[userID]; ok {
			return u.Name
		}
	}
	return userID
}

func writeDetailField(sb *strings.Builder, label, value string) {
	sb.WriteString(fmt.Sprintf("  %-10s %s\n", label+":", value))
}

func plansForItem(board *domain.Board, itemID string) []*domain.Plan {
	if board.Plans == nil {
		return nil
	}
	var plans []*domain.Plan
	for _, p := range board.Plans {
		for _, lid := range p.LinkedItems {
			if lid == itemID {
				plans = append(plans, p)
				break
			}
		}
	}
	return plans
}
```

Note: The `domain.Plan` type needs to be accessible — if it's already defined in `internal/domain/types.go` from Task 1, this will compile.

**Step 3: Update app.go for full-screen detail**

Modify `handleDetailKey` in `internal/tui/app.go` to handle tab switching and scrolling:

Replace the existing `handleDetailKey` method with:

```go
func (a App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	item := a.detail.item
	switch msg.String() {
	case "esc":
		a.state = stateBoard
	case "q":
		return a, tea.Quit
	case "tab":
		a.detail.NextTab()
	case "shift+tab":
		a.detail.PrevTab()
	case "j", "down":
		a.detail.ScrollDown()
	case "k", "up":
		a.detail.ScrollUp()
	case "m":
		if item != nil {
			a.picker = newPickerModel(
				fmt.Sprintf("Move #%d to:", item.DisplayNum),
				pickerColumn, a.columns,
			)
			a.state = statePicker
			a.prevState = stateDetail
		}
	case "a":
		if item != nil {
			users := userNames(a.board)
			a.picker = newPickerModel(
				fmt.Sprintf("Assign #%d to:", item.DisplayNum),
				pickerUser, users,
			)
			a.state = statePicker
			a.prevState = stateDetail
		}
	case "p":
		if item != nil {
			nextPri := cyclePriority(string(item.Priority))
			_ = a.engine.EditItem(item.ID, "", "", nextPri, "", "")
			return a, a.loadBoard()
		}
	}
	return a, nil
}
```

Update the `View()` method's `stateDetail` case to render full-screen instead of overlay:

In `app.go`, change the View switch case for detail:

```go
case stateDetail:
    a.detail.SetSize(a.width, a.height)
    return a.detail.View()
```

Update the `handleBoardKey` enter case to pass board:

```go
case "enter":
    if item := a.selectedItem(); item != nil {
        a.detail = newDetailModel(item, a.board)
        a.prevState = stateBoard
        a.state = stateDetail
    }
```

**Step 4: Build and verify**

Run: `go build -o /tmp/ob-test .`
Expected: Builds successfully

**Step 5: Commit and update board**

```bash
git add internal/tui/detail.go internal/tui/keys.go internal/tui/app.go
git commit -m "feat(tui): full-screen tabbed detail view with Fields/Plan/History"
~/bin/ob move 14 in-progress
~/bin/ob move 14 done
```

---

## Task 6: Skill and Plugin Updates

**Obeya Story:** #15 (Skill and plugin updates)

**Files:**
- Modify: `skill/obeya.md`
- Create: `obeya-plugin/skills/ob-plan/SKILL.md`

**Step 1: Update skill/obeya.md**

Add these sections to `skill/obeya.md` after the "### 5. Complete Work" section:

```markdown
### 6. Rich Task Descriptions

When creating tasks, provide detailed descriptions using `--body-file`:

1. Write the description to a temporary file:

```bash
cat > /tmp/task-desc.md << 'TASKEOF'
## What This Task Involves

Implement JWT token validation middleware for the auth service.

## Files

- Create: `internal/auth/jwt.go`
- Modify: `internal/middleware/auth.go:45-60`
- Test: `internal/auth/jwt_test.go`

## Acceptance Criteria

- Validates token signature and expiry
- Returns 401 for invalid/expired tokens
- Extracts user ID from claims
TASKEOF
```

2. Create the task with the description file:

```bash
ob create task "Add JWT validation" -p 2 --body-file /tmp/task-desc.md --priority high
```

Short one-line descriptions (via `-d`) are acceptable only for trivial tasks. For any task that another agent might pick up, use `--body-file` with full context.

### 7. Plan Management

After creating a task breakdown from an implementation plan:

1. Import the plan document and link it to all related items:

```bash
ob plan import docs/plans/your-plan.md --link 1,2,3,4,5
```

2. When creating additional items related to an existing plan:

```bash
ob plan link <plan-id> --to <new-item-id>
```

3. Query plans:

```bash
ob plan list --format json
ob plan show <plan-id> --format json
```

Plans provide full context for anyone (human or agent) picking up a task. Always import your implementation plan after creating the task breakdown.
```

Also add to the Command Reference section:

```markdown
### Plan Management

```bash
# Create an empty plan
ob plan create --title "Feature Plan"

# Import plan from file and link to items
ob plan import docs/plans/plan.md --link 1,2,3

# Update plan content
ob plan update <plan-id> docs/plans/updated.md
ob plan update <plan-id> --title "New Title"

# Show plan details
ob plan show <plan-id>
ob plan show <plan-id> --format json

# List all plans
ob plan list
ob plan list --format json

# Link/unlink items
ob plan link <plan-id> --to 4,5,6
ob plan unlink <plan-id> --from 3

# Delete a plan
ob plan delete <plan-id>
```
```

**Step 2: Create ob-plan skill**

Create `obeya-plugin/skills/ob-plan/SKILL.md`:

```markdown
---
description: Manage Obeya plan documents — import, link, show
disable-model-invocation: false
user-invocable: true
---

# /ob:plan — Plan Management

Run `ob plan list --format json` to see all plans.

If `$ARGUMENTS` is provided:
- If it starts with "import", run `ob plan import $ARGUMENTS`
- If it starts with "show", run `ob plan show $ARGUMENTS --format json`
- If it starts with "link", run `ob plan link $ARGUMENTS`
- Otherwise, show the plan list

Format the output as a readable summary showing plan title, linked item count, and source file.
```

**Step 3: Commit and update board**

```bash
git add skill/obeya.md obeya-plugin/skills/ob-plan/SKILL.md
git commit -m "feat(skill): add plan management and verbose description instructions"
~/bin/ob move 15 in-progress
~/bin/ob move 15 done
```

---

## Task 7: Integration Test

**Files:**
- Modify: `test/integration_test.go`

**Step 1: Add plan integration test**

Add to `test/integration_test.go`:

```go
func TestIntegration_PlanWorkflow(t *testing.T) {
	eng := setupTestEngine(t)

	// Create items
	epic, err := eng.CreateItem("epic", "Auth System", "", "", "high", "", nil)
	if err != nil {
		t.Fatalf("CreateItem epic failed: %v", err)
	}
	task, err := eng.CreateItem("task", "JWT Validation", fmt.Sprintf("%d", epic.DisplayNum), "", "medium", "", nil)
	if err != nil {
		t.Fatalf("CreateItem task failed: %v", err)
	}

	// Import plan
	content := "# Auth Implementation Plan\n\nBuild JWT auth with middleware.\n\n## Task 1: JWT Library\n\nImplement token parsing."
	plan, err := eng.ImportPlan(content, "docs/plans/auth-plan.md", []string{
		fmt.Sprintf("%d", epic.DisplayNum),
		fmt.Sprintf("%d", task.DisplayNum),
	})
	if err != nil {
		t.Fatalf("ImportPlan failed: %v", err)
	}
	if plan.Title != "Auth Implementation Plan" {
		t.Errorf("expected title from markdown heading, got %q", plan.Title)
	}
	if len(plan.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(plan.LinkedItems))
	}

	// Show plan
	shown, err := eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if err != nil {
		t.Fatalf("ShowPlan failed: %v", err)
	}
	if shown.Content != content {
		t.Error("plan content mismatch")
	}

	// List plans
	plans, err := eng.ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}

	// Link additional item
	task2, _ := eng.CreateItem("task", "Middleware", fmt.Sprintf("%d", epic.DisplayNum), "", "medium", "", nil)
	if err := eng.LinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", task2.DisplayNum)}); err != nil {
		t.Fatalf("LinkPlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(shown.LinkedItems) != 3 {
		t.Errorf("expected 3 linked items after link, got %d", len(shown.LinkedItems))
	}

	// Unlink
	if err := eng.UnlinkPlan(fmt.Sprintf("%d", plan.DisplayNum), []string{fmt.Sprintf("%d", task.DisplayNum)}); err != nil {
		t.Fatalf("UnlinkPlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if len(shown.LinkedItems) != 2 {
		t.Errorf("expected 2 linked items after unlink, got %d", len(shown.LinkedItems))
	}

	// PlansForItem
	itemPlans, err := eng.PlansForItem(epic.ID)
	if err != nil {
		t.Fatalf("PlansForItem failed: %v", err)
	}
	if len(itemPlans) != 1 {
		t.Errorf("expected 1 plan for epic, got %d", len(itemPlans))
	}

	// Update plan
	if err := eng.UpdatePlan(fmt.Sprintf("%d", plan.DisplayNum), "Updated Auth Plan", "# Updated\n\nNew content."); err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}
	shown, _ = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if shown.Title != "Updated Auth Plan" {
		t.Errorf("expected updated title, got %q", shown.Title)
	}

	// Delete plan
	if err := eng.DeletePlan(fmt.Sprintf("%d", plan.DisplayNum)); err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}
	_, err = eng.ShowPlan(fmt.Sprintf("%d", plan.DisplayNum))
	if err == nil {
		t.Error("expected error showing deleted plan")
	}
}
```

**Step 2: Run tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 3: Commit and update board**

```bash
git add test/integration_test.go
git commit -m "test: add plan workflow integration test"
```

---

## Task 8: Backwards Compatibility — Nil Plans Map

**Files:**
- Modify: `internal/store/json_store.go`

**Step 1: Handle nil Plans on load**

In `json_store.go`, after `json.Unmarshal` in `LoadBoard()`, add:

```go
if board.Plans == nil {
    board.Plans = make(map[string]*domain.Plan)
}
```

This ensures boards created before the Plans feature work correctly.

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All pass

**Step 3: Build final binary**

```bash
go build -o /tmp/ob-test .
cp /tmp/ob-test ~/bin/ob
```

**Step 4: Commit and close epic**

```bash
git add internal/store/json_store.go
git commit -m "fix: initialize nil Plans map on board load for backwards compatibility"
~/bin/ob move 9 done
```

---

## Summary

| Task | Story | Description | Files |
|------|-------|-------------|-------|
| 1 | #10 | Plan data model + ResolvePlanID | types.go, id.go |
| 2 | #11 | Plan engine methods (CRUD + link) | engine/plan.go |
| 3 | #13 | --body-file flag | cmd/create.go, cmd/edit.go |
| 4 | #12 | ob plan CLI subcommands | cmd/plan.go |
| 5 | #14 | Full-screen tabbed detail view | tui/detail.go, tui/keys.go, tui/app.go |
| 6 | #15 | Skill and plugin updates | skill/obeya.md, plugin skill |
| 7 | — | Integration test | test/integration_test.go |
| 8 | — | Backwards compatibility | store/json_store.go |

**Total: 8 tasks**

**Parallelizable groups:**
- Tasks 1-2 (data model + engine) — sequential
- Tasks 3-4 (CLI changes) — parallel after Task 2
- Task 5 (TUI) — parallel after Task 2
- Task 6 (skills) — independent, can run anytime
- Tasks 7-8 (integration) — after all above
