package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var assignCmd = &cobra.Command{
	Use:   "assign <id>",
	Short: "Assign an item to a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		to, err := cmd.Flags().GetString("to")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.AssignItem(args[0], to, getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Assigned #%s to %s\n", args[0], to)
	},
}

func init() {
	assignCmd.Flags().String("to", "", "user to assign the item to (required)")
	assignCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(assignCmd)
}
