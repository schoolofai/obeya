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

type ProviderInfo struct {
	Name      string
	ConfigDir string
	SkillFile string
	Supported bool
}

func GetProviders() []ProviderInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return []ProviderInfo{
		{Name: "claude-code", ConfigDir: filepath.Join(home, ".claude"), SkillFile: "obeya.md", Supported: true},
		{Name: "opencode", ConfigDir: filepath.Join(home, ".opencode"), SkillFile: "obeya.md", Supported: false},
		{Name: "codex", ConfigDir: filepath.Join(home, ".codex"), SkillFile: "obeya.md", Supported: false},
	}
}

func runSkillInstall(cmd *cobra.Command, args []string) {
	skillSource := findSkillSource()
	providers := GetProviders()

	if flagSkillProvider != "" {
		providers = filterProviders(providers, flagSkillProvider)
	}

	installed := 0
	for _, p := range providers {
		if !p.Supported {
			fmt.Fprintf(os.Stderr, "Error: provider %q is not yet supported.\n\n"+
				"Only 'claude-code' is currently supported. Run: ob skill install --provider claude-code\n", p.Name)
			os.Exit(1)
		}
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

func filterProviders(providers []ProviderInfo, name string) []ProviderInfo {
	for _, p := range providers {
		if p.Name == name {
			return []ProviderInfo{p}
		}
	}
	fmt.Fprintf(os.Stderr, "Error: unknown provider %q.\n\n"+
		"Only 'claude-code' is currently supported. Run: ob skill install --provider claude-code\n", name)
	os.Exit(1)
	return nil
}

func installSkillForProvider(p ProviderInfo, content []byte) error {
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
	providers := GetProviders()
	fmt.Printf("%-15s %-15s %s\n", "PROVIDER", "SUPPORTED", "STATUS")
	for _, p := range providers {
		supported := "yes"
		if !p.Supported {
			supported = "not yet"
		}
		dest := filepath.Join(p.ConfigDir, p.SkillFile)
		status := "not installed"
		if _, err := os.Stat(dest); err == nil {
			status = "installed"
		}
		fmt.Printf("%-15s %-15s %s\n", p.Name, supported, status)
	}
}
