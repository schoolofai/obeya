package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink this project from its shared board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		gitRoot, err := store.FindGitRoot(cwd)
		if err != nil {
			return err
		}

		linkFile := filepath.Join(gitRoot, ".obeya-link")
		data, err := os.ReadFile(linkFile)
		if err != nil {
			return fmt.Errorf("this project is not linked to any shared board")
		}

		boardName := strings.TrimSpace(string(data))
		projectName := resolveProjectName(gitRoot)

		if err := unregisterProject(boardName, projectName); err != nil {
			return err
		}

		if err := os.Remove(linkFile); err != nil {
			return fmt.Errorf("failed to remove .obeya-link: %w", err)
		}

		fmt.Printf("Unlinked project %q from shared board %q\n", projectName, boardName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

func unregisterProject(boardName, projectName string) error {
	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}

	boardDir := store.SharedBoardDir(obeyaHome, boardName)
	boardJsonPath := filepath.Join(boardDir, ".obeya", "board.json")
	if _, err := os.Stat(boardJsonPath); err != nil {
		if os.IsNotExist(err) {
			return nil // board was already deleted, nothing to unregister
		}
		return fmt.Errorf("failed to check shared board at %s: %w", boardJsonPath, err)
	}

	sharedStore := store.NewJSONStore(boardDir)
	return sharedStore.Transaction(func(b *domain.Board) error {
		delete(b.Projects, projectName)
		return nil
	})
}
