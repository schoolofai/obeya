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

func TestNewStore_ReturnsCloudStore_WhenCloudConfigExists(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	// Create cloud.json
	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
		User:    "testuser",
	}
	store.SaveCloudConfig(filepath.Join(obeyaDir, "cloud.json"), cfg)

	// Create credentials in a temp home dir
	credsDir := t.TempDir()
	credsPath := filepath.Join(credsDir, "credentials.json")
	creds := &store.Credentials{
		Token:     "ob_tok_test",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	s, err := store.NewStore(dir, credsPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Should be CloudStore — BoardFilePath returns empty string
	if s.BoardFilePath() != "" {
		t.Error("expected CloudStore (empty BoardFilePath), got non-empty — likely JSONStore")
	}
}

func TestNewStore_ErrorsWhenCloudConfig_NoCredentials(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
	}
	store.SaveCloudConfig(filepath.Join(obeyaDir, "cloud.json"), cfg)

	_, err := store.NewStore(dir, "/nonexistent/credentials.json")
	if err == nil {
		t.Fatal("expected error when cloud config present but credentials missing")
	}
}
