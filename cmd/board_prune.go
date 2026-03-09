package cmd

import (
	"fmt"
	"os"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var boardPruneCmd = &cobra.Command{
	Use:   "prune <board-name>",
	Short: "Remove dead project entries from a shared board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardName := args[0]

		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardDir := store.SharedBoardDir(obeyaHome, boardName)
		s := store.NewJSONStore(boardDir)
		if !s.BoardExists() {
			return fmt.Errorf("board %q not found", boardName)
		}

		pruned := 0
		err = s.Transaction(func(b *domain.Board) error {
			for name, proj := range b.Projects {
				if _, statErr := os.Stat(proj.LocalPath); os.IsNotExist(statErr) {
					delete(b.Projects, name)
					pruned++
					fmt.Printf("Removed dead project: %s (%s)\n", name, proj.LocalPath)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		if pruned == 0 {
			fmt.Println("No dead projects found.")
		} else {
			fmt.Printf("Pruned %d dead project(s).\n", pruned)
		}
		return nil
	},
}

func init() {
	boardCmd.AddCommand(boardPruneCmd)
}
