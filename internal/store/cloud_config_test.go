package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/store"
)

func TestCloudConfig_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	if err := os.MkdirAll(obeyaDir, 0755); err != nil {
		t.Fatalf("failed to create .obeya dir: %v", err)
	}

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc123",
		OrgID:   "org_456",
		User:    "niladribose",
	}

	path := filepath.Join(obeyaDir, "cloud.json")
	if err := store.SaveCloudConfig(path, cfg); err != nil {
		t.Fatalf("SaveCloudConfig failed: %v", err)
	}

	loaded, err := store.LoadCloudConfig(path)
	if err != nil {
		t.Fatalf("LoadCloudConfig failed: %v", err)
	}

	if loaded.APIURL != cfg.APIURL {
		t.Errorf("APIURL: got %q, want %q", loaded.APIURL, cfg.APIURL)
	}
	if loaded.BoardID != cfg.BoardID {
		t.Errorf("BoardID: got %q, want %q", loaded.BoardID, cfg.BoardID)
	}
	if loaded.OrgID != cfg.OrgID {
		t.Errorf("OrgID: got %q, want %q", loaded.OrgID, cfg.OrgID)
	}
	if loaded.User != cfg.User {
		t.Errorf("User: got %q, want %q", loaded.User, cfg.User)
	}
}

func TestCloudConfig_LoadMissing(t *testing.T) {
	_, err := store.LoadCloudConfig("/nonexistent/cloud.json")
	if err == nil {
		t.Fatal("expected error loading missing config, got nil")
	}
}

func TestCloudConfigExists(t *testing.T) {
	dir := t.TempDir()
	obeyaDir := filepath.Join(dir, ".obeya")
	os.MkdirAll(obeyaDir, 0755)

	if store.CloudConfigExists(dir) {
		t.Error("expected CloudConfigExists to return false before creation")
	}

	cfg := &store.CloudConfig{
		APIURL:  "https://obeya.app/api",
		BoardID: "board_abc",
	}
	path := filepath.Join(obeyaDir, "cloud.json")
	store.SaveCloudConfig(path, cfg)

	if !store.CloudConfigExists(dir) {
		t.Error("expected CloudConfigExists to return true after creation")
	}
}

func TestCredentials_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	creds := &store.Credentials{
		Token:     "ob_tok_abc123secret",
		UserID:    "usr_789",
		CreatedAt: "2026-03-12T10:00:00Z",
	}

	path := filepath.Join(dir, "credentials.json")
	if err := store.SaveCredentials(path, creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	loaded, err := store.LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loaded.Token != creds.Token {
		t.Errorf("Token: got %q, want %q", loaded.Token, creds.Token)
	}
	if loaded.UserID != creds.UserID {
		t.Errorf("UserID: got %q, want %q", loaded.UserID, creds.UserID)
	}
	if loaded.CreatedAt != creds.CreatedAt {
		t.Errorf("CreatedAt: got %q, want %q", loaded.CreatedAt, creds.CreatedAt)
	}
}

func TestCredentials_LoadMissing(t *testing.T) {
	_, err := store.LoadCredentials("/nonexistent/credentials.json")
	if err == nil {
		t.Fatal("expected error loading missing credentials, got nil")
	}
}

func TestCredentials_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := &store.Credentials{
		Token:     "ob_tok_secret",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	if err := store.SaveCredentials(path, creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat credentials file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("credentials file permissions: got %o, want 0600", perm)
	}
}

func TestCredentialsPath(t *testing.T) {
	path, err := store.DefaultCredentialsPath()
	if err != nil {
		t.Fatalf("DefaultCredentialsPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty credentials path")
	}
}

func TestCloudConfigPath(t *testing.T) {
	dir := t.TempDir()
	path := store.CloudConfigPath(dir)
	expected := filepath.Join(dir, ".obeya", "cloud.json")
	if path != expected {
		t.Errorf("CloudConfigPath: got %q, want %q", path, expected)
	}
}
