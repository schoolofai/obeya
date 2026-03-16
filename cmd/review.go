package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reviewStatus string

var reviewCmd = &cobra.Command{
	Use:   "review <ref>",
	Short: "Mark an item's human review status",
	Long: `Mark an item as reviewed or hidden. Only available to human identities.

Examples:
  ob review 34 --status reviewed
  ob review 34 --status hidden`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]

		if reviewStatus == "" {
			return fmt.Errorf("--status is required. Must be 'reviewed' or 'hidden'")
		}
		if reviewStatus != "reviewed" && reviewStatus != "hidden" {
			return fmt.Errorf("invalid --status %q: must be 'reviewed' or 'hidden'", reviewStatus)
		}

		eng, err := getEngine()
		if err != nil {
			return err
		}

		if err := eng.ReviewItem(ref, reviewStatus, getUserID(), getSessionID()); err != nil {
			return err
		}

		fmt.Printf("Marked #%s as %s\n", ref, reviewStatus)
		return nil
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewStatus, "status", "", "review status: reviewed or hidden (required)")
	rootCmd.AddCommand(reviewCmd)
}
