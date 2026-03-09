package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var initColumns string
var initClaudeMD bool

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board in the current directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		columns := parseColumns(initColumns)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		if err := s.InitBoard(boardName, columns); err != nil {
			return err
		}

		printInitConfirmation(boardName, columns)

		if initClaudeMD {
			if err := appendClaudeMD(); err != nil {
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

func printInitConfirmation(name string, columns []string) {
	fmt.Printf("Board %q initialized in .obeya/\n", name)
	if len(columns) > 0 {
		fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
	} else {
		fmt.Println("Columns: backlog, todo, in-progress, review, done")
	}
}

func appendClaudeMD() error {
	claudeMDPath := "CLAUDE.md"

	content := `
## Task Tracking — Obeya

This project uses Obeya (` + "`ob`" + `) for task tracking. Before starting work:
1. Run ` + "`/ob:status`" + ` to check assigned tasks
2. Run ` + "`/ob:pick`" + ` to claim a task if none assigned
3. Run ` + "`/ob:done`" + ` when work is complete

Use ` + "`ob list --format json`" + ` for full board state.
`

	existing, err := os.ReadFile(claudeMDPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if strings.Contains(string(existing), "Task Tracking — Obeya") {
		return nil
	}

	f, err := os.OpenFile(claudeMDPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CLAUDE.md: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to CLAUDE.md: %w", err)
	}

	return nil
}
