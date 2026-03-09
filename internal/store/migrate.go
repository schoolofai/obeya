package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

// MigrateLocalToShared copies all items from a local board to a shared board,
// tagging each with the project name. Renames local .obeya/ to .obeya-local-backup/.
// Returns the number of migrated items.
func MigrateLocalToShared(localRoot, sharedBoardDir, projectName string) (int, error) {
	localStore := NewJSONStore(localRoot)
	localBoard, err := localStore.LoadBoard()
	if err != nil {
		return 0, fmt.Errorf("failed to load local board: %w", err)
	}

	itemCount := len(localBoard.Items)
	if itemCount == 0 {
		if err := backupLocalObeya(localRoot); err != nil {
			return 0, err
		}
		return 0, nil
	}

	sharedStore := NewJSONStore(sharedBoardDir)
	err = sharedStore.Transaction(func(shared *domain.Board) error {
		for _, item := range localBoard.Items {
			newID := fmt.Sprintf("%s-%s", projectName, item.ID)
			migrated := *item
			migrated.ID = newID
			migrated.Project = projectName
			migrated.DisplayNum = shared.NextDisplay
			shared.Items[newID] = &migrated
			shared.DisplayMap[shared.NextDisplay] = newID
			shared.NextDisplay++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to write migrated items to shared board: %w", err)
	}

	if err := backupLocalObeya(localRoot); err != nil {
		return 0, err
	}

	return itemCount, nil
}

func backupLocalObeya(root string) error {
	src := filepath.Join(root, ".obeya")
	dst := filepath.Join(root, ".obeya-local-backup")
	if _, err := os.Stat(dst); err == nil {
		dst = fmt.Sprintf("%s-%d", dst, time.Now().Unix())
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to rename .obeya to backup: %w", err)
	}
	return nil
}
