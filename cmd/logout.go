package cmd

import (
	"fmt"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored Obeya Cloud credentials",
	Long:  "Removes the API token from ~/.obeya/credentials.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		credsPath, err := store.DefaultCredentialsPath()
		if err != nil {
			return err
		}

		if !auth.IsLoggedIn(credsPath) {
			fmt.Println("Not currently logged in.")
			return nil
		}

		if err := auth.Logout(credsPath); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		fmt.Println("Logged out. Credentials removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
