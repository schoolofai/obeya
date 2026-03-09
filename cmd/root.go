package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagAs      string
	flagSession string
	flagFormat  string
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

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAs, "as", "", "user ID for this operation (or set OB_USER)")
	rootCmd.PersistentFlags().StringVar(&flagSession, "session", "", "session ID for audit trail (or set OB_SESSION)")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "text", "output format: text or json")
}
