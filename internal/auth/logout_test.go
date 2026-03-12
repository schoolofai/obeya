package auth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
)

func TestLogout_RemovesCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	// Create credentials first
	creds := &store.Credentials{
		Token:     "ob_tok_to_remove",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	// Verify they exist
	if _, err := os.Stat(credsPath); err != nil {
		t.Fatalf("credentials file should exist before logout")
	}

	// Logout
	err := auth.Logout(credsPath)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Verify they're gone
	if _, err := os.Stat(credsPath); !os.IsNotExist(err) {
		t.Error("credentials file should not exist after logout")
	}
}

func TestLogout_NoCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	// Should not error if file doesn't exist
	err := auth.Logout(credsPath)
	if err != nil {
		t.Fatalf("Logout should succeed even when no credentials exist: %v", err)
	}
}

func TestIsLoggedIn_True(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	creds := &store.Credentials{
		Token:     "ob_tok_check",
		UserID:    "usr_1",
		CreatedAt: "2026-03-12T10:00:00Z",
	}
	store.SaveCredentials(credsPath, creds)

	if !auth.IsLoggedIn(credsPath) {
		t.Error("expected IsLoggedIn to return true")
	}
}

func TestIsLoggedIn_False(t *testing.T) {
	if auth.IsLoggedIn("/nonexistent/credentials.json") {
		t.Error("expected IsLoggedIn to return false for missing file")
	}
}
