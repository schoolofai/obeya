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

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board. Defaults to the git repository root. Use --root to specify a custom location.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := resolveInitRoot()
		if err != nil {
			return err
		}

		s := store.NewJSONStore(root)
		columns := parseColumns(initColumns)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		if err := s.InitBoard(boardName, columns); err != nil {
			return err
		}

		fmt.Printf("Board %q initialized in %s/.obeya/\n", boardName, root)
		if len(columns) > 0 {
			fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
		} else {
			fmt.Println("Columns: backlog, todo, in-progress, review, done")
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
	rootCmd.AddCommand(initCmd)
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
	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no git repository found — use 'ob init --root <path>' to specify a board location")
}

func appendClaudeMDAt(claudePath string) error {
	content := `
## Task Tracking — Obeya

This project uses Obeya (` + "`ob`" + `) for task tracking. Before starting work:
1. Run ` + "`/ob:status`" + ` to check assigned tasks
2. Run ` + "`/ob:pick`" + ` to claim a task if none assigned
3. Run ` + "`/ob:done`" + ` when work is complete

Use ` + "`ob list --format json`" + ` for full board state.
`

	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if strings.Contains(string(existing), "Task Tracking — Obeya") {
		return nil
	}

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
