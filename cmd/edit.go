package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an item's title, description, or priority",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile != "" && desc != "" {
			fmt.Fprintf(os.Stderr, "Error: --body-file and -d/--description are mutually exclusive\n")
			os.Exit(1)
		}
		if bodyFile != "" {
			data, err := os.ReadFile(bodyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to read body file %q: %v\n", bodyFile, err)
				os.Exit(1)
			}
			desc = string(data)
		}
		priority, _ := cmd.Flags().GetString("priority")

		eng, err := getEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := eng.EditItem(args[0], title, desc, priority, getUserID(), getSessionID()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated #%s\n", args[0])
	},
}

func init() {
	editCmd.Flags().String("title", "", "new title for the item")
	editCmd.Flags().StringP("description", "d", "", "new description for the item")
	editCmd.Flags().String("priority", "", "new priority (low, medium, high, critical)")
	editCmd.Flags().String("body-file", "", "read description from file")
	rootCmd.AddCommand(editCmd)
}
