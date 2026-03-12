package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CloudConfig is stored at .obeya/cloud.json in the project directory.
// This file is committed to the repo — it contains no secrets.
type CloudConfig struct {
	APIURL  string `json:"api_url"`
	BoardID string `json:"board_id"`
	OrgID   string `json:"org_id,omitempty"`
	User    string `json:"user,omitempty"`
}

// Credentials is stored at ~/.obeya/credentials.json in the user's home directory.
// This file is NOT committed — it contains the API token.
type Credentials struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

// CloudConfigPath returns the path to cloud.json for a given project root.
func CloudConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".obeya", "cloud.json")
}

// CloudConfigExists checks if a cloud.json file exists in the project.
func CloudConfigExists(rootDir string) bool {
	_, err := os.Stat(CloudConfigPath(rootDir))
	return err == nil
}

// LoadCloudConfig reads and parses a cloud.json file.
func LoadCloudConfig(path string) (*CloudConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cloud config at %s: %w", path, err)
	}

	var cfg CloudConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse cloud config: %w", err)
	}

	if cfg.APIURL == "" {
		return nil, fmt.Errorf("cloud config missing required field: api_url")
	}
	if cfg.BoardID == "" {
		return nil, fmt.Errorf("cloud config missing required field: board_id")
	}

	return &cfg, nil
}

// SaveCloudConfig writes a CloudConfig to the given path as JSON.
func SaveCloudConfig(path string, cfg *CloudConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for cloud config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cloud config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cloud config: %w", err)
	}

	return nil
}

// DefaultCredentialsPath returns ~/.obeya/credentials.json.
func DefaultCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".obeya", "credentials.json"), nil
}

// LoadCredentials reads and parses the credentials file.
func LoadCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials at %s: %w", path, err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.Token == "" {
		return nil, fmt.Errorf("credentials file missing required field: token")
	}

	return &creds, nil
}

// SaveCredentials writes credentials to the given path with 0600 permissions.
func SaveCredentials(path string, creds *Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// DeleteCredentials removes the credentials file.
func DeleteCredentials(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}
