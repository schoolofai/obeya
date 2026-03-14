package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestNewStore_ReturnsJSONStore_WhenNoCloudConfig(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	// Create a board.json so JSONStore works
	boardFile := filepath.Join(obeyaDir, "board.json")
	os.WriteFile(boardFile, []byte(`{"version":1,"name":"test","columns":[],"items":{},"display_map":{},"next_display":1,"users":{},"plans":{},"projects":{}}`), 0644)

	s, err := store.NewStore(dir, "")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Should be JSONStore — BoardFilePath returns a non-empty string
	if s.BoardFilePath() == "" {
		t.Error("expected JSONStore (non-empty BoardFilePath), got empty — likely CloudStore")
	}
}

