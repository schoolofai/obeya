package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/niladribose/obeya/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive board TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := getEngine()
		if err != nil {
			return err
		}
		model := tui.New(eng)

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
