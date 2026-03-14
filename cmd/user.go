package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage board users",
}

var userAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a user to the board",
	Args:  cobra.ExactArgs(1),
	Run:   runUserAdd,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users on the board",
	Run:   runUserList,
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a user from the board",
	Args:  cobra.ExactArgs(1),
	Run:   runUserRemove,
}

var (
	flagUserType     string
	flagUserProvider string
)

func init() {
	userAddCmd.Flags().StringVar(&flagUserType, "type", "human", "user type: human or agent")
	userAddCmd.Flags().StringVar(&flagUserProvider, "provider", "local", "identity provider. Supported: 'local' (human), 'claude-code' (agent). Other providers are not yet supported.")

	userCmd.AddCommand(userAddCmd, userListCmd, userRemoveCmd)
	rootCmd.AddCommand(userCmd)
}

var supportedProviders = map[string]bool{
	"local":      true,
	"claude-code": true,
}

func runUserAdd(cmd *cobra.Command, args []string) {
	if !supportedProviders[flagUserProvider] {
		fmt.Fprintf(os.Stderr, "Error: unsupported provider %q.\n\n"+
			"Only Claude Code is currently supported as an agent provider.\n"+
			"Supported providers: local (human), claude-code (agent)\n\n"+
			"Other agent providers are planned but not yet available.\n", flagUserProvider)
		os.Exit(1)
	}

	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := eng.AddUser(args[0], flagUserType, flagUserProvider); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User %q added\n", args[0])
}

func runUserList(cmd *cobra.Command, args []string) {
	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	board, err := eng.ListBoard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if flagFormat == "json" {
		printUserListJSON(board.Users)
		return
	}
	printUserListText(board.Users)
}

func printUserListJSON(users map[string]*domain.Identity) {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func printUserListText(users map[string]*domain.Identity) {
	if len(users) == 0 {
		fmt.Println("No users registered")
		return
	}
	for _, u := range users {
		fmt.Printf("%-10s %-20s %-8s %s\n", u.ID[:8], u.Name, u.Type, u.Provider)
	}
}

func runUserRemove(cmd *cobra.Command, args []string) {
	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := eng.RemoveUser(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User %q removed\n", args[0])
}
