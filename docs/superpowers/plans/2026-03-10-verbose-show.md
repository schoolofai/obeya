# `ob show --verbose` Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `--verbose` / `-v` flag to `ob show` that displays richer children detail (priority, assignee, blocked-by, description snippet).

**Architecture:** Add a boolean flag to the existing cobra command. Pass it through to `printItemChildren`, which gains a verbose code path rendering extra lines per child. The SKILL.md is updated to document the new flag.

**Tech Stack:** Go, cobra, standard testing

---

## File Structure

- **Modify:** `cmd/show.go` — add flag registration, pass verbose bool to children printer, add verbose rendering logic
- **Create:** `cmd/show_test.go` — unit tests for the verbose children formatting
- **Modify:** `obeya-plugin/skills/ob-show/SKILL.md` — document `--verbose` flag

---

## Chunk 1: Implementation

### Task 1: Add `--verbose` flag and verbose children rendering

**Files:**
- Modify: `cmd/show.go:22-24` (init function — add flag)
- Modify: `cmd/show.go:43-44` (runShow — read flag, pass to printItemChildren)
- Modify: `cmd/show.go:100-116` (printItemChildren — add verbose parameter and rendering)

- [ ] **Step 1: Write failing test for verbose children output**

Create `cmd/show_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/niladribose/obeya/internal/domain"
)

func TestPrintItemChildrenVerbose(t *testing.T) {
	children := []*domain.Item{
		{
			DisplayNum:  3,
			Type:        domain.ItemTypeTask,
			Status:      "in-progress",
			Title:       "Fix login bug",
			Priority:    domain.PriorityHigh,
			Assignee:    "niladri",
			BlockedBy:   []string{"#5"},
			Description: "This is a short description",
		},
		{
			DisplayNum:  4,
			Type:        domain.ItemTypeTask,
			Status:      "backlog",
			Title:       "Add tests",
			Priority:    domain.PriorityMedium,
			Assignee:    "",
			BlockedBy:   nil,
			Description: "Write unit tests for the auth module that cover all edge cases and ensure full coverage of the login flow",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printChildrenVerbose(children)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check child #3 has priority, assignee, and blocked-by on same line
	if !strings.Contains(output, "Priority: high | Assignee: niladri | Blocked by: #5") {
		t.Errorf("expected pipe-delimited detail line with blocked-by, got:\n%s", output)
	}

	// Check child #4 has dash for missing assignee, no blocked-by
	if !strings.Contains(output, "Priority: medium | Assignee: —") {
		t.Errorf("expected 'Priority: medium | Assignee: —' for unset assignee, got:\n%s", output)
	}

	// Check description truncation (80 chars + ...)
	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated description with '...', got:\n%s", output)
	}

	// Check short description is NOT truncated
	if !strings.Contains(output, "This is a short description") {
		t.Errorf("expected full short description, got:\n%s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/niladribose/code/obeya && go test ./cmd/ -run TestPrintItemChildrenVerbose -v`
Expected: FAIL — `printChildrenVerbose` undefined

- [ ] **Step 3: Register `--verbose` flag in init()**

In `cmd/show.go`, update the `init()` function:

```go
func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().String("format", "", "output format (json)")
	showCmd.Flags().BoolP("verbose", "v", false, "show detailed children info (priority, assignee, description)")
}
```

- [ ] **Step 4: Pass verbose flag through runShow**

In `cmd/show.go`, update `runShow`:

```go
func runShow(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	item, err := eng.GetItem(args[0])
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return printShowJSON(item, eng)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")

	printItemDetail(item)
	printItemChildren(eng, item, verbose)
	printItemHistory(item)
	return nil
}
```

- [ ] **Step 5: Add `truncateDesc` helper and `printChildrenVerbose` function**

Add to `cmd/show.go`:

```go
func truncateDesc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func printChildrenVerbose(children []*domain.Item) {
	for _, child := range children {
		fmt.Fprintf(os.Stdout, "  #%-4d [%s] %s — %s\n",
			child.DisplayNum, child.Type, child.Status, child.Title)

		assignee := "—"
		if child.Assignee != "" {
			assignee = child.Assignee
		}
		line := fmt.Sprintf("      Priority: %s | Assignee: %s", child.Priority, assignee)
		if len(child.BlockedBy) > 0 {
			line += fmt.Sprintf(" | Blocked by: %s", strings.Join(child.BlockedBy, ", "))
		}
		fmt.Fprintln(os.Stdout, line)

		if child.Description != "" {
			fmt.Fprintf(os.Stdout, "      Desc: %s\n", truncateDesc(child.Description, 80))
		}

		fmt.Fprintln(os.Stdout)
	}
}
```

- [ ] **Step 6: Update `printItemChildren` to accept verbose param**

Replace the existing `printItemChildren` in `cmd/show.go`:

```go
func printItemChildren(eng *engine.Engine, item *domain.Item, verbose bool) {
	children, err := eng.GetChildren(item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to get children: %v\n", err)
		return
	}
	if len(children) == 0 {
		return
	}

	fmt.Fprintln(os.Stdout, "\nChildren:")
	sortItemsByDisplayNum(children)

	if verbose {
		printChildrenVerbose(children)
		return
	}

	for _, child := range children {
		fmt.Fprintf(os.Stdout, "  #%-4d [%s] %s — %s\n",
			child.DisplayNum, child.Type, child.Status, child.Title)
	}
}
```

- [ ] **Step 7: Run tests**

Run: `cd /Users/niladribose/code/obeya && go test ./cmd/ -run TestPrintItemChildrenVerbose -v`
Expected: PASS

- [ ] **Step 8: Run full test suite**

Run: `cd /Users/niladribose/code/obeya && go test ./...`
Expected: All pass

- [ ] **Step 9: Commit**

```bash
git add cmd/show.go cmd/show_test.go
git commit -m "feat: add --verbose flag to ob show for richer children detail"
```

---

### Task 2: Update SKILL.md documentation

**Files:**
- Modify: `obeya-plugin/skills/ob-show/SKILL.md`

- [ ] **Step 1: Update SKILL.md**

Add `--verbose` / `-v` flag documentation. Update the usage section to:

```markdown
## Usage

`/ob-show <id>` — show item details
`/ob-show <id> --verbose` — show item details with rich children info
```

And add to step 4 after "Children: list any child items with their status":

```markdown
   - **Children (verbose)**: if `--verbose`, show each child's priority, assignee, blocked-by, and description snippet (80 char max)
```

- [ ] **Step 2: Commit**

```bash
git add obeya-plugin/skills/ob-show/SKILL.md
git commit -m "docs: document --verbose flag in ob-show skill"
```

---

### Task 3: Manual smoke test

- [ ] **Step 1: Build and test**

```bash
cd /Users/niladribose/code/obeya && go build -o ob . && ./ob show 1 --verbose
```

Verify verbose children output renders correctly with real board data.
