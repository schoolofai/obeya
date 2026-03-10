package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initColumns string
var initAgent string
var initSkipPlugin bool
var initRoot string
var initShared string

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board with agent integration. Requires --agent flag.\nUse --shared for storage-only boards (no agent integration).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		columns := parseColumns(initColumns)

		// --shared and --agent are mutually exclusive
		if initShared != "" && initAgent != "" {
			return fmt.Errorf("--shared and --agent are mutually exclusive. Shared boards do not support agent integration")
		}

		// Shared board path (no agent needed)
		if initShared != "" {
			return initSharedBoard(initShared, columns)
		}

		// --agent is required for non-shared boards
		if initAgent == "" {
			return fmt.Errorf("required flag --agent not provided. Supported: %s", strings.Join(agent.SupportedNames(), ", "))
		}

		// Validate agent name
		agentSetup, err := agent.Get(initAgent)
		if err != nil {
			return err
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

		// Delegate to agent-specific setup
		ctx := agent.AgentContext{
			Root:       root,
			BoardName:  boardName,
			SkipPlugin: initSkipPlugin,
		}
		if err := agentSetup.Setup(ctx); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().StringVar(&initAgent, "agent", "", "coding agent to configure (supported: claude-code)")
	initCmd.Flags().BoolVar(&initSkipPlugin, "skip-plugin", false, "skip plugin installation")
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

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return store.FindGitRoot(cwd)
}
