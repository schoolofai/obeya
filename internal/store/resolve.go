package store

import (
	"fmt"
)

// NewStore resolves the appropriate Store implementation based on project configuration.
// If .obeya/cloud.json exists in rootDir, returns a CloudStore.
// Otherwise, returns a JSONStore.
// The credsPath parameter specifies where to find credentials. Pass empty string
// to use the default (~/.obeya/credentials.json).
func NewStore(rootDir, credsPath string) (Store, error) {
	if CloudConfigExists(rootDir) {
		return newCloudStoreFromConfig(rootDir, credsPath)
	}

	return NewJSONStore(rootDir), nil
}

// newCloudStoreFromConfig loads cloud config and credentials, then creates a CloudStore.
func newCloudStoreFromConfig(rootDir, credsPath string) (Store, error) {
	cfgPath := CloudConfigPath(rootDir)
	cfg, err := LoadCloudConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cloud config: %w", err)
	}

	if credsPath == "" {
		credsPath, err = DefaultCredentialsPath()
		if err != nil {
			return nil, err
		}
	}

	creds, err := LoadCredentials(credsPath)
	if err != nil {
		return nil, fmt.Errorf("cloud mode requires authentication — run 'ob login' first: %w", err)
	}

	return NewCloudStore(cfg.APIURL, creds.Token, cfg.BoardID), nil
}
