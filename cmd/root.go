package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ob",
	Short: "Obeya — CLI Kanban board for humans and AI agents",
	Long:  "A CLI-based Kanban board manager that serves both humans (via TUI) and AI agents (via CLI commands).",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
