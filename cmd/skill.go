package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage agent skill files",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the obeya skill file for detected agent providers",
	Run:   runSkillInstall,
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show supported providers and install status",
	Run:   runSkillList,
}

var flagSkillProvider string

func init() {
	skillInstallCmd.Flags().StringVar(&flagSkillProvider, "provider", "", "install for a specific provider only")

	skillCmd.AddCommand(skillInstallCmd, skillListCmd)
	rootCmd.AddCommand(skillCmd)
}

type providerInfo struct {
	Name      string
	ConfigDir string
	SkillFile string
}

func getProviders() []providerInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return []providerInfo{
		{Name: "claude-code", ConfigDir: filepath.Join(home, ".claude"), SkillFile: "obeya.md"},
		{Name: "opencode", ConfigDir: filepath.Join(home, ".opencode"), SkillFile: "obeya.md"},
		{Name: "codex", ConfigDir: filepath.Join(home, ".codex"), SkillFile: "obeya.md"},
	}
}

func runSkillInstall(cmd *cobra.Command, args []string) {
	skillSource := findSkillSource()
	providers := getProviders()

	if flagSkillProvider != "" {
		providers = filterProviders(providers, flagSkillProvider)
	}

	installed := 0
	for _, p := range providers {
		if err := installSkillForProvider(p, skillSource); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing for %s: %v\n", p.Name, err)
			os.Exit(1)
		}
		fmt.Printf("Installed skill for %s -> %s\n", p.Name, filepath.Join(p.ConfigDir, p.SkillFile))
		installed++
	}

	if installed == 0 {
		fmt.Fprintf(os.Stderr, "Error: no matching providers found\n")
		os.Exit(1)
	}
}

func findSkillSource() []byte {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	path := filepath.Join(cwd, "skill", "obeya.md")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: skill source not found at %s: %v\n", path, err)
		os.Exit(1)
	}
	return data
}

func filterProviders(providers []providerInfo, name string) []providerInfo {
	for _, p := range providers {
		if p.Name == name {
			return []providerInfo{p}
		}
	}
	fmt.Fprintf(os.Stderr, "Error: unknown provider %q\n", name)
	os.Exit(1)
	return nil
}

func installSkillForProvider(p providerInfo, content []byte) error {
	if err := os.MkdirAll(p.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir %s: %w", p.ConfigDir, err)
	}
	dest := filepath.Join(p.ConfigDir, p.SkillFile)
	if err := os.WriteFile(dest, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}
	return nil
}

func runSkillList(cmd *cobra.Command, args []string) {
	providers := getProviders()
	for _, p := range providers {
		dest := filepath.Join(p.ConfigDir, p.SkillFile)
		status := "not installed"
		if _, err := os.Stat(dest); err == nil {
			status = "installed"
		}
		fmt.Printf("%-15s %s\n", p.Name, status)
	}
}
