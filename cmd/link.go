package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var linkMigrate bool

var linkCmd = &cobra.Command{
	Use:   "link <board-name>",
	Short: "Link this project to a shared board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardName := args[0]

		obeyaHome, err := store.ObeyaHome()
		if err != nil {
			return err
		}

		boardDir := store.SharedBoardDir(obeyaHome, boardName)
		if err := validateSharedBoardExists(boardDir, boardName); err != nil {
			return err
		}

		gitRoot, err := resolveGitRoot()
		if err != nil {
			return err
		}

		if err := ensureNotAlreadyLinked(gitRoot); err != nil {
			return err
		}

		if err := handleLocalBoardMigration(gitRoot, boardDir, boardName); err != nil {
			return err
		}

		if err := writeLinkFile(gitRoot, boardName); err != nil {
			return err
		}

		if err := registerProject(gitRoot, boardDir); err != nil {
			return err
		}

		projectName := resolveProjectName(gitRoot)
		fmt.Printf("Linked project %q to shared board %q\n", projectName, boardName)
		return nil
	},
}

func init() {
	linkCmd.Flags().BoolVar(&linkMigrate, "migrate", false, "migrate local board tasks to the shared board")
	rootCmd.AddCommand(linkCmd)
}

func validateSharedBoardExists(boardDir, boardName string) error {
	boardFile := filepath.Join(boardDir, ".obeya", "board.json")
	if _, err := os.Stat(boardFile); err != nil {
		return fmt.Errorf("board %q not found — run 'ob init --shared %s' first", boardName, boardName)
	}
	return nil
}

func resolveGitRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return store.FindGitRoot(cwd)
}

func ensureNotAlreadyLinked(gitRoot string) error {
	linkFile := filepath.Join(gitRoot, ".obeya-link")
	if _, err := os.Stat(linkFile); err == nil {
		existing, _ := os.ReadFile(linkFile)
		return fmt.Errorf("this project is already linked to board %q", strings.TrimSpace(string(existing)))
	}
	return nil
}

func handleLocalBoardMigration(gitRoot, boardDir, boardName string) error {
	localBoard := filepath.Join(gitRoot, ".obeya", "board.json")
	if _, err := os.Stat(localBoard); err != nil {
		return nil // no local board, nothing to migrate
	}

	localStore := store.NewJSONStore(gitRoot)
	board, err := localStore.LoadBoard()
	if err != nil {
		return fmt.Errorf("failed to read local board: %w", err)
	}

	taskCount := len(board.Items)
	if taskCount > 0 && !linkMigrate {
		return fmt.Errorf(
			"this project has %d tasks on a local board — rerun with --migrate to move them to %q",
			taskCount, boardName,
		)
	}

	migrated, err := store.MigrateLocalToShared(gitRoot, boardDir, resolveProjectName(gitRoot))
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	if migrated > 0 {
		fmt.Printf("Migrated %d tasks to shared board %q\n", migrated, boardName)
	}

	return nil
}

func writeLinkFile(gitRoot, boardName string) error {
	linkFile := filepath.Join(gitRoot, ".obeya-link")
	if err := os.WriteFile(linkFile, []byte(boardName), 0644); err != nil {
		return fmt.Errorf("failed to write .obeya-link: %w", err)
	}
	return nil
}

func registerProject(gitRoot, boardDir string) error {
	projectName := resolveProjectName(gitRoot)
	gitRemote := resolveGitRemote(gitRoot)

	sharedStore := store.NewJSONStore(boardDir)
	return sharedStore.Transaction(func(b *domain.Board) error {
		if b.Projects == nil {
			b.Projects = make(map[string]*domain.LinkedProject)
		}
		b.Projects[projectName] = &domain.LinkedProject{
			Name:      projectName,
			LocalPath: gitRoot,
			GitRemote: gitRemote,
			LinkedAt:  time.Now().Format(time.RFC3339),
		}
		return nil
	})
}

func resolveProjectName(gitRoot string) string {
	remote := resolveGitRemote(gitRoot)
	if remote != "" {
		remote = strings.TrimSuffix(remote, ".git")
		parts := strings.Split(remote, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	}
	return filepath.Base(gitRoot)
}

func resolveGitRemote(gitRoot string) string {
	cmd := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
