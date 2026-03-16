package cmd

import (
	"fmt"
	"os"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Move an item to a new status column",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if dryRun {
			if err := previewMove(eng, args[0], args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if args[1] == "done" {
			warnIfAgent(eng)
		}

		if err := eng.MoveItem(args[0], args[1], getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Moved #%s to %q\n", args[0], args[1])
	},
}

func previewMove(eng *engine.Engine, ref, status string) error {
	item, err := eng.GetItem(ref)
	if err != nil {
		return err
	}

	board, err := eng.ListBoard()
	if err != nil {
		return err
	}

	if !board.HasColumn(status) {
		return fmt.Errorf("invalid status %q — available columns: %s", status, boardColumnNames(board))
	}

	fmt.Printf("[dry-run] Would move #%d %q (%s)\n", item.DisplayNum, item.Title, item.Type)
	fmt.Printf("  Status: %s → %s\n", item.Status, status)
	return nil
}

func boardColumnNames(board *domain.Board) string {
	names := ""
	for i, c := range board.Columns {
		if i > 0 {
			names += ", "
		}
		names += c.Name
	}
	return names
}

// warnIfAgent emits a warning when an agent uses "ob move <ref> done"
// instead of "ob done <ref>" which provides review context.
func warnIfAgent(eng *engine.Engine) {
	actorType, err := eng.ResolveActorType(getUserID())
	if err != nil {
		return // non-fatal: skip warning if resolution fails
	}
	if actorType == "agent" {
		fmt.Fprintf(os.Stderr, "Warning: use 'ob done <ref>' to include review context for human review.\n")
	}
}

func init() {
	moveCmd.Flags().Bool("dry-run", false, "Preview what would change without moving")
	rootCmd.AddCommand(moveCmd)
}
