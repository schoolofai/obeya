package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an item from the board",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.DeleteItem(args[0], getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted #%s\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
