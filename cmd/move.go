package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Move an item to a new status column",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.MoveItem(args[0], args[1], getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Moved #%s to %q\n", args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
}
