package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initColumns string
var initClaudeMD bool
var initRoot string
var initShared string

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board. Defaults to the git repository root. Use --root to specify a custom location.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		columns := parseColumns(initColumns)

		if initShared != "" {
			return initSharedBoard(initShared, columns)
		}

		root, err := resolveInitRoot()
		if err != nil {
			return err
		}

		s := store.NewJSONStore(root)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		err = s.InitBoard(boardName, columns)
		if err != nil {
			// If board exists, that's OK — we may still need to update CLAUDE.md
			if !strings.Contains(err.Error(), "already initialized") {
				return err
			}
			fmt.Printf("Board already initialized in %s/.obeya/\n", root)
		} else {
			fmt.Printf("Board %q initialized in %s/.obeya/\n", boardName, root)
			if len(columns) > 0 {
				fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
			} else {
				fmt.Println("Columns: backlog, todo, in-progress, review, done")
			}
		}

		if initClaudeMD {
			claudePath := filepath.Join(root, "CLAUDE.md")
			if err := appendClaudeMDAt(claudePath); err != nil {
				return fmt.Errorf("could not update CLAUDE.md: %w", err)
			}
			fmt.Println("Updated CLAUDE.md with Obeya board instructions")
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().BoolVar(&initClaudeMD, "claude-md", true, "append Obeya instructions to project CLAUDE.md")
	initCmd.Flags().StringVar(&initRoot, "root", "", "directory to create .obeya in (default: git repository root)")
	initCmd.Flags().StringVar(&initShared, "shared", "", "create a shared board at ~/.obeya/boards/<name>")
	rootCmd.AddCommand(initCmd)
}

func initSharedBoard(boardName string, columns []string) error {
	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}

	boardDir := store.SharedBoardDir(obeyaHome, boardName)
	boardFile := filepath.Join(boardDir, ".obeya", "board.json")

	if _, err := os.Stat(boardFile); err == nil {
		return fmt.Errorf("board %q already exists — use 'ob link %s' to connect this project", boardName, boardName)
	}

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Shared board %q initialized at %s\n", boardName, boardDir)
	return nil
}

func parseColumns(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func resolveInitRoot() (string, error) {
	if initRoot != "" {
		abs, err := filepath.Abs(initRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve --root path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("--root path does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("--root path is not a directory: %s", abs)
		}
		return abs, nil
	}

	// Default: find git root by walking up
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return store.FindGitRoot(cwd)
}

const obeyaSectionStart = "<!-- obeya:start -->"
const obeyaSectionEnd = "<!-- obeya:end -->"
const obeyaSectionVersion = "v2"

func obeyaClaudeMDContent() string {
	return obeyaSectionStart + " " + obeyaSectionVersion + `

## Task Tracking — Obeya

This project uses Obeya (` + "`ob`" + `) for task tracking.

### Before starting work
1. Run ` + "`/ob:status`" + ` to check assigned tasks
2. Run ` + "`/ob:pick`" + ` to claim a task if none assigned
3. Run ` + "`/ob:done`" + ` when work is complete

### Plan management
When a plan document is created, discussed, or approved in this session:
1. Import it: ` + "`ob plan import <path-to-plan.md>`" + `
2. Link tasks to it: ` + "`ob plan link <plan-id> --to <task-ids>`" + `
3. When creating subtasks under a plan-linked parent, link them too: ` + "`ob plan link <plan-id> --to <new-task-id>`" + `

### Task lifecycle
- Starting work: ` + "`ob move <id> in-progress`" + `
- Blocked: ` + "`ob block <id> --by <blocker-id>`" + `
- Done: ` + "`ob move <id> done`" + `

Use ` + "`ob list --format json`" + ` for full board state.

` + obeyaSectionEnd + `
`
}

func appendClaudeMDAt(claudePath string) error {
	content := obeyaClaudeMDContent()

	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	existingStr := string(existing)

	// Replace existing section if present (handles version upgrades)
	if startIdx := strings.Index(existingStr, obeyaSectionStart); startIdx != -1 {
		endIdx := strings.Index(existingStr, obeyaSectionEnd)
		if endIdx == -1 {
			return fmt.Errorf("found obeya section start but no end marker in CLAUDE.md")
		}
		endIdx += len(obeyaSectionEnd)
		// Skip trailing newline if present
		if endIdx < len(existingStr) && existingStr[endIdx] == '\n' {
			endIdx++
		}
		updated := existingStr[:startIdx] + content + existingStr[endIdx:]
		return os.WriteFile(claudePath, []byte(updated), 0644)
	}

	// Legacy check: replace old section without markers
	if strings.Contains(existingStr, "Task Tracking — Obeya") {
		legacyStart := strings.Index(existingStr, "## Task Tracking — Obeya")
		if legacyStart > 0 {
			// Find next heading or end of file
			rest := existingStr[legacyStart+1:]
			nextHeading := strings.Index(rest, "\n## ")
			var legacyEnd int
			if nextHeading != -1 {
				legacyEnd = legacyStart + 1 + nextHeading + 1
			} else {
				legacyEnd = len(existingStr)
			}
			updated := existingStr[:legacyStart] + content + existingStr[legacyEnd:]
			return os.WriteFile(claudePath, []byte(updated), 0644)
		}
	}

	// Fresh append
	f, err := os.OpenFile(claudePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CLAUDE.md: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to CLAUDE.md: %w", err)
	}

	return nil
}
