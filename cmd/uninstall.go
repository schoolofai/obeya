package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var flagUninstallYes bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Obeya agent integrations (preserves board data)",
	Long:  "Removes the Claude Code plugin, skill files, and CLAUDE.md obeya sections.\nBoard data in .obeya/ directories is preserved.",
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVarP(&flagUninstallYes, "yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx, err := buildUninstallContext()
	if err != nil {
		return err
	}

	printUninstallPreview(ctx)

	if !flagUninstallYes {
		if !promptConfirm("Proceed? [y/N] ") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := executeUninstall(ctx); err != nil {
		return err
	}

	printUninstallBanner()
	return nil
}

type uninstallContext struct {
	inProject      bool
	gitRoot        string
	globalClaudeMD string
	skillFiles     []skillFileInfo
}

type skillFileInfo struct {
	provider string
	path     string
	exists   bool
}

func buildUninstallContext() (*uninstallContext, error) {
	if err := agent.CheckClaudeCLI(); err != nil {
		return nil, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	ctx := &uninstallContext{
		globalClaudeMD: filepath.Join(home, ".claude", "CLAUDE.md"),
	}

	cwd, err := os.Getwd()
	if err == nil {
		if gitRoot, err := store.FindGitRoot(cwd); err == nil {
			claudePath := filepath.Join(gitRoot, "CLAUDE.md")
			if data, err := os.ReadFile(claudePath); err == nil {
				if strings.Contains(string(data), agent.ObeyaSectionStart) {
					ctx.inProject = true
					ctx.gitRoot = gitRoot
				}
			}
		}
	}

	for _, p := range GetProviders() {
		path := filepath.Join(p.ConfigDir, p.SkillFile)
		_, statErr := os.Stat(path)
		ctx.skillFiles = append(ctx.skillFiles, skillFileInfo{
			provider: p.Name,
			path:     path,
			exists:   statErr == nil,
		})
	}

	return ctx, nil
}

func printUninstallPreview(ctx *uninstallContext) {
	fmt.Println("ob uninstall — the following changes will be made:")
	fmt.Println()
	fmt.Println("  CLAUDE CODE PLUGIN")
	fmt.Println("  ├── Uninstall plugin: obeya@obeya-local")
	fmt.Println("  └── Remove marketplace: obeya-local")
	fmt.Println()

	existing := collectExistingSkills(ctx.skillFiles)
	if len(existing) > 0 {
		fmt.Println("  SKILL FILES")
		for i, sf := range existing {
			prefix := "  ├──"
			if i == len(existing)-1 {
				prefix = "  └──"
			}
			fmt.Printf("%s Remove %s\n", prefix, sf.path)
		}
		fmt.Println()
	}

	fmt.Println("  CLAUDE.md")
	if ctx.inProject {
		fmt.Printf("  ├── Strip obeya section from %s\n", ctx.globalClaudeMD)
		fmt.Printf("  └── Strip obeya section from %s\n", filepath.Join(ctx.gitRoot, "CLAUDE.md"))
	} else {
		fmt.Printf("  └── Strip obeya section from %s\n", ctx.globalClaudeMD)
	}
	fmt.Println()

	fmt.Println("  PRESERVED (not touched)")
	fmt.Println("  ├── .obeya/          (board data, cloud config)")
	fmt.Println("  ├── ~/.obeya/        (shared boards, credentials)")
	fmt.Println("  └── .obeya-link      (board links)")
	fmt.Println()
}

func collectExistingSkills(files []skillFileInfo) []skillFileInfo {
	var result []skillFileInfo
	for _, sf := range files {
		if sf.exists {
			result = append(result, sf)
		}
	}
	return result
}

func executeUninstall(ctx *uninstallContext) error {
	if err := agent.UninstallPlugin(); err != nil {
		return fmt.Errorf("plugin removal failed: %w", err)
	}
	fmt.Println("  ✓ Plugin and marketplace removed")

	for _, sf := range ctx.skillFiles {
		if !sf.exists {
			continue
		}
		if err := os.Remove(sf.path); err != nil {
			return fmt.Errorf("failed to remove skill file %s: %w", sf.path, err)
		}
		fmt.Printf("  ✓ Removed %s\n", sf.path)
	}

	if err := agent.StripClaudeMDAt(ctx.globalClaudeMD); err != nil {
		return fmt.Errorf("failed to clean global CLAUDE.md: %w", err)
	}
	fmt.Printf("  ✓ Cleaned %s\n", ctx.globalClaudeMD)

	if ctx.inProject {
		projectPath := filepath.Join(ctx.gitRoot, "CLAUDE.md")
		if err := agent.StripClaudeMDAt(projectPath); err != nil {
			return fmt.Errorf("failed to clean project CLAUDE.md: %w", err)
		}
		fmt.Printf("  ✓ Cleaned %s\n", projectPath)
	}

	return nil
}

func printUninstallBanner() {
	fmt.Println()
	fmt.Println("┌───────────────────────────────────────────────────────┐")
	fmt.Println("│                                                       │")
	fmt.Println("│  Obeya agent integrations removed successfully.       │")
	fmt.Println("│                                                       │")
	fmt.Println("│  To fully remove ob from your system:                 │")
	fmt.Println("│                                                       │")
	fmt.Println("│    brew uninstall obeya                               │")
	fmt.Println("│                                                       │")
	fmt.Println("│  Board data and cloud config were preserved.          │")
	fmt.Println("│  Delete them manually if no longer needed.            │")
	fmt.Println("│                                                       │")
	fmt.Println("└───────────────────────────────────────────────────────┘")
}

func promptConfirm(prompt string) bool {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
