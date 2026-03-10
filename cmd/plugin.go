package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage the obeya Claude Code plugin",
}

var pluginSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync plugin source to Claude Code cache (creates symlinks for live development)",
	RunE:  runPluginSync,
}

func init() {
	pluginCmd.AddCommand(pluginSyncCmd)
	rootCmd.AddCommand(pluginCmd)
}

type installedPluginsFile struct {
	Version int                                `json:"version"`
	Plugins map[string][]installedPluginEntry  `json:"plugins"`
}

type installedPluginEntry struct {
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

func runPluginSync(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	pluginSourceDir, err := findPluginSource()
	if err != nil {
		return err
	}

	installedPath := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	data, err := os.ReadFile(installedPath)
	if err != nil {
		return fmt.Errorf("cannot read installed_plugins.json: %w", err)
	}

	var installed installedPluginsFile
	if err := json.Unmarshal(data, &installed); err != nil {
		return fmt.Errorf("cannot parse installed_plugins.json: %w", err)
	}

	// Find the obeya plugin entry
	var cachePath string
	for key, entries := range installed.Plugins {
		if strings.HasPrefix(key, "obeya@") && len(entries) > 0 {
			cachePath = entries[0].InstallPath
			break
		}
	}

	if cachePath == "" {
		return fmt.Errorf("obeya plugin not found in installed_plugins.json — install it first via Claude Code")
	}

	// Check if already a symlink pointing to the right place
	linkTarget, err := os.Readlink(cachePath)
	if err == nil && linkTarget == pluginSourceDir {
		fmt.Printf("Already synced: %s -> %s\n", cachePath, pluginSourceDir)
		return nil
	}

	// Remove the cache directory (it's a copy)
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("cannot remove cache directory %s: %w", cachePath, err)
	}

	// Create symlink from cache path to plugin source
	if err := os.Symlink(pluginSourceDir, cachePath); err != nil {
		return fmt.Errorf("cannot create symlink: %w", err)
	}

	fmt.Printf("Synced: %s -> %s\n", cachePath, pluginSourceDir)
	fmt.Println("Plugin changes now take effect immediately (restart Claude Code session to pick up hook changes)")
	return nil
}

func findPluginSource() (string, error) {
	// Look for obeya-plugin/ relative to git root
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Walk up to find obeya-plugin directory
	dir := cwd
	for {
		candidate := filepath.Join(dir, "obeya-plugin")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			pluginJSON := filepath.Join(candidate, ".claude-plugin", "plugin.json")
			if _, err := os.Stat(pluginJSON); err == nil {
				return candidate, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("cannot find obeya-plugin/ directory — run this from the obeya source tree")
}
