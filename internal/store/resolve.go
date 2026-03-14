package store

import (
	"fmt"
)

// cloudStoreResolver is set by resolve_cloud.go when built with -tags cloud.
// Returns (Store, error). If Store is nil and error is nil, falls through to JSONStore.
var cloudStoreResolver func(rootDir, credsPath string) (Store, error)

// NewStore resolves the appropriate Store implementation based on project configuration.
// Without the cloud build tag, always returns a JSONStore.
// With -tags cloud: if .obeya/cloud.json exists in rootDir, returns a CloudStore.
func NewStore(rootDir, credsPath string) (Store, error) {
	if cloudStoreResolver != nil {
		s, err := cloudStoreResolver(rootDir, credsPath)
		if s != nil || err != nil {
			return s, err
		}
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
