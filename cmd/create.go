package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/spf13/cobra"
)

var (
	createParent   string
	createPriority string
	createAssign   string
	createTags     string
	createDesc     string
)

var createCmd = &cobra.Command{
	Use:   "create <type> <title>",
	Short: "Create an epic, story, or task",
	Long:  "Create a new item on the board. Types: epic, story, task.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		itemType := args[0]
		title := args[1]

		tags := parseTags(createTags)

		eng, err := getEngine()
		if err != nil {
			return err
		}
		item, err := eng.CreateItem(itemType, title, createParent, createDesc, createPriority, createAssign, tags)
		if err != nil {
			return err
		}

		return printCreatedItem(item)
	},
}

func init() {
	createCmd.Flags().StringVarP(&createParent, "parent", "p", "", "parent item ID or display number")
	createCmd.Flags().StringVar(&createPriority, "priority", "medium", "priority: low, medium, high, critical")
	createCmd.Flags().StringVar(&createAssign, "assign", "", "assign to user ID")
	createCmd.Flags().StringVar(&createTags, "tag", "", "comma-separated tags")
	createCmd.Flags().StringVarP(&createDesc, "description", "d", "", "item description")
	rootCmd.AddCommand(createCmd)
}

func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func printCreatedItem(item *domain.Item) error {
	if flagFormat == "json" {
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal item to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Created %s #%d [%s]: %s\n", item.Type, item.DisplayNum, item.ID[:6], item.Title)
	if item.ParentID != "" {
		fmt.Printf("  Parent: %s\n", item.ParentID[:6])
	}
	return nil
}
