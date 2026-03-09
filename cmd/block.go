package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <id>",
	Short: "Mark an item as blocked by another item",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		by, err := cmd.Flags().GetString("by")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.BlockItem(args[0], by, getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Blocked #%s by #%s\n", args[0], by)
	},
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <id>",
	Short: "Remove a blocker from an item",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		by, err := cmd.Flags().GetString("by")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.UnblockItem(args[0], by, getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Unblocked #%s from #%s\n", args[0], by)
	},
}

func init() {
	blockCmd.Flags().String("by", "", "ID of the blocking item (required)")
	blockCmd.MarkFlagRequired("by")
	rootCmd.AddCommand(blockCmd)

	unblockCmd.Flags().String("by", "", "ID of the blocking item (required)")
	unblockCmd.MarkFlagRequired("by")
	rootCmd.AddCommand(unblockCmd)
}
