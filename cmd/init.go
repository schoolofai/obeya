package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initColumns string
var initAgent string
var initSkipPlugin bool
var initRoot string
var initShared string
var initCloud bool
var initLocal bool

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board with agent integration. Requires --agent flag.\nUse --shared for storage-only boards. Combine --shared with --agent for global Obeya.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		columns := parseColumns(initColumns)

		// Handle --cloud
		if initCloud {
			root, err := resolveInitRoot()
			if err != nil {
				return err
			}
			return initCloudBoard(root, columns, args)
		}

		// Handle --local
		if initLocal {
			root, err := resolveInitRoot()
			if err != nil {
				return err
			}
			return initLocalFromCloud(root, columns, args)
		}

		// Shared + agent = shared board with agent setup
		if initShared != "" && initAgent != "" {
			return initSharedBoardWithAgent(initShared, initAgent, columns)
		}

		// Shared board path (no agent)
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
	initCmd.Flags().BoolVar(&initCloud, "cloud", false, "create or migrate to a cloud board")
	initCmd.Flags().BoolVar(&initLocal, "local", false, "switch from cloud mode back to local board")
	rootCmd.AddCommand(initCmd)
}

func initSharedBoardWithAgent(sharedName, agentName string, columns []string) error {
	agentSetup, err := agent.Get(agentName)
	if err != nil {
		return err
	}

	// Create shared board
	if err := initSharedBoard(sharedName, columns); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
		fmt.Printf("Shared board %q already exists — proceeding with agent setup\n", sharedName)
	}

	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}
	boardDir := store.SharedBoardDir(obeyaHome, sharedName)

	ctx := agent.AgentContext{
		Root:       boardDir,
		BoardName:  sharedName,
		SkipPlugin: initSkipPlugin,
		Shared:     true,
	}
	return agentSetup.Setup(ctx)
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

func initCloudBoard(root string, columns []string, args []string) error {
	credsPath, err := store.DefaultCredentialsPath()
	if err != nil {
		return err
	}

	if store.CloudConfigExists(root) {
		return reportAlreadyCloudBoard(root)
	}

	if !auth.IsLoggedIn(credsPath) {
		fmt.Println("Not logged in. Running 'ob login' first...")
		if err := loginCmd.RunE(loginCmd, nil); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	creds, err := store.LoadCredentials(credsPath)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	client := store.NewCloudClient(auth.DefaultAppURL+"/api", creds.Token)
	localStore := store.NewJSONStore(root)

	if localStore.BoardExists() {
		return migrateLocalToCloud(root, client)
	}

	return createFreshCloudBoard(root, columns, args, client)
}

func reportAlreadyCloudBoard(root string) error {
	cfg, err := store.LoadCloudConfig(store.CloudConfigPath(root))
	if err != nil {
		return err
	}
	fmt.Printf("Already connected to cloud board %s.\nRun 'ob init --local' to switch to local mode.\n", cfg.BoardID)
	return nil
}

func migrateLocalToCloud(root string, client *store.CloudClient) error {
	localStore := store.NewJSONStore(root)
	board, err := localStore.LoadBoard()
	if err != nil {
		return fmt.Errorf("failed to load local board: %w", err)
	}

	itemCount := len(board.Items)
	fmt.Printf("Local board found with %d items.\n", itemCount)
	fmt.Printf("Migrating to cloud...\n")

	boardID, err := client.ImportBoard(board, "")
	if err != nil {
		return fmt.Errorf("failed to import board to cloud: %w", err)
	}

	if err := backupAndReplaceObeya(root); err != nil {
		return err
	}

	_, username, _ := client.GetMe()
	cfg := &store.CloudConfig{
		APIURL:  auth.DefaultAppURL + "/api",
		BoardID: boardID,
		User:    username,
	}
	if err := store.SaveCloudConfig(store.CloudConfigPath(root), cfg); err != nil {
		return fmt.Errorf("failed to save cloud config: %w", err)
	}

	backupDir := filepath.Join(root, ".obeya-local-backup")
	fmt.Printf("Migrated %d items to cloud board %s\n", itemCount, boardID)
	fmt.Printf("Local backup saved to %s\n", backupDir)
	return nil
}

func backupAndReplaceObeya(root string) error {
	obeyaDir := filepath.Join(root, ".obeya")
	backupDir := filepath.Join(root, ".obeya-local-backup")

	if err := os.Rename(obeyaDir, backupDir); err != nil {
		return fmt.Errorf("failed to backup local .obeya directory: %w", err)
	}

	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate .obeya directory: %w", err)
	}

	return nil
}

func createFreshCloudBoard(root string, columns []string, args []string, client *store.CloudClient) error {
	boardName := "obeya"
	if len(args) > 0 {
		boardName = args[0]
	}

	if len(columns) == 0 {
		columns = []string{"backlog", "todo", "in-progress", "review", "done"}
	}

	boardID, err := client.CreateBoard(boardName, columns, "")
	if err != nil {
		return fmt.Errorf("failed to create cloud board: %w", err)
	}

	obeyaDir := filepath.Join(root, ".obeya")
	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .obeya directory: %w", err)
	}

	_, username, _ := client.GetMe()
	cfg := &store.CloudConfig{
		APIURL:  auth.DefaultAppURL + "/api",
		BoardID: boardID,
		User:    username,
	}
	if err := store.SaveCloudConfig(store.CloudConfigPath(root), cfg); err != nil {
		return fmt.Errorf("failed to save cloud config: %w", err)
	}

	fmt.Printf("Cloud board %q created (ID: %s)\n", boardName, boardID)
	fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
	return nil
}

func initLocalFromCloud(root string, columns []string, args []string) error {
	if !store.CloudConfigExists(root) {
		fmt.Println("Not in cloud mode — already using local storage.")
		return nil
	}

	cloudPath := store.CloudConfigPath(root)
	if err := os.Remove(cloudPath); err != nil {
		return fmt.Errorf("failed to remove cloud config: %w", err)
	}

	backupDir := filepath.Join(root, ".obeya-local-backup")
	if _, err := os.Stat(backupDir); err == nil {
		return restoreFromBackup(root, backupDir)
	}

	return initFreshLocalBoard(root, columns, args)
}

func restoreFromBackup(root, backupDir string) error {
	obeyaDir := filepath.Join(root, ".obeya")
	if err := os.RemoveAll(obeyaDir); err != nil {
		return fmt.Errorf("failed to remove .obeya directory: %w", err)
	}
	if err := os.Rename(backupDir, obeyaDir); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	fmt.Println("Restored local board from backup.")
	return nil
}

func initFreshLocalBoard(root string, columns []string, args []string) error {
	boardName := "obeya"
	if len(args) > 0 {
		boardName = args[0]
	}

	s := store.NewJSONStore(root)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Switched to local mode. Board %q initialized.\n", boardName)
	return nil
}
