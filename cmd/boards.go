package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var boardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "List all shared boards",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardsDir := filepath.Join(obeyaHome, "boards")
		entries, err := os.ReadDir(boardsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No shared boards found. Run 'ob init --shared <name>' to create one.")
				return nil
			}
			return fmt.Errorf("failed to read boards directory: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No shared boards found. Run 'ob init --shared <name>' to create one.")
			return nil
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			boardPath := filepath.Join(boardsDir, entry.Name())
			s := store.NewJSONStore(boardPath)
			if !s.BoardExists() {
				continue
			}

			board, err := s.LoadBoard()
			if err != nil {
				fmt.Printf("%-20s  (error: %v)\n", entry.Name(), err)
				continue
			}

			projectCount := len(board.Projects)
			noun := "projects"
			if projectCount == 1 {
				noun = "project"
			}
			fmt.Printf("%-20s  %d %s\n", entry.Name(), projectCount, noun)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(boardsCmd)
}
