//go:build cloud

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initCloud bool
var initLocal bool

func init() {
	initCmd.Flags().BoolVar(&initCloud, "cloud", false, "create or migrate to a cloud board")
	initCmd.Flags().BoolVar(&initLocal, "local", false, "switch from cloud mode back to local board")

	cloudInitHandler = handleCloudInit
}

func handleCloudInit(_ *cobra.Command, args []string, columns []string) (bool, error) {
	if initCloud {
		root, err := resolveInitRoot()
		if err != nil {
			return false, err
		}
		return true, initCloudBoard(root, columns, args)
	}

	if initLocal {
		root, err := resolveInitRoot()
		if err != nil {
			return false, err
		}
		return true, initLocalFromCloud(root, columns, args)
	}

	return false, nil
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
