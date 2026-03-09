package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var initColumns string

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
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
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
