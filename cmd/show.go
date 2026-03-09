package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed information about an item",
	Long:  "Display full details of a board item including type, status, priority, description, assignee, parent, tags, blocked_by, children, and history.",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().String("format", "", "output format (json)")
}

func runShow(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	item, err := eng.GetItem(args[0])
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return printShowJSON(item, eng)
	}

	printItemDetail(item)
	printItemChildren(eng, item)
	printItemHistory(item)
	return nil
}

func printShowJSON(item *domain.Item, eng *engine.Engine) error {
	children, err := eng.GetChildren(item.ID)
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}

	output := struct {
		*domain.Item
		Children []*domain.Item `json:"children,omitempty"`
	}{Item: item, Children: children}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal item to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printItemDetail(item *domain.Item) {
	fmt.Fprintf(os.Stdout, "#%d  %s\n", item.DisplayNum, item.Title)
	fmt.Fprintf(os.Stdout, "%-12s %s\n", "Type:", string(item.Type))
	fmt.Fprintf(os.Stdout, "%-12s %s\n", "Status:", item.Status)
	fmt.Fprintf(os.Stdout, "%-12s %s\n", "Priority:", string(item.Priority))
	printOptionalField("Assignee:", item.Assignee)
	printOptionalField("Parent:", item.ParentID)
	printOptionalField("Description:", item.Description)
	printBlockedBy(item.BlockedBy)
	printTags(item.Tags)
	fmt.Fprintf(os.Stdout, "%-12s %s\n", "Created:", item.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stdout, "%-12s %s\n", "Updated:", item.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func printOptionalField(label, value string) {
	if value != "" {
		fmt.Fprintf(os.Stdout, "%-12s %s\n", label, value)
	}
}

func printBlockedBy(blockedBy []string) {
	if len(blockedBy) > 0 {
		fmt.Fprintf(os.Stdout, "%-12s %s\n", "Blocked by:", strings.Join(blockedBy, ", "))
	}
}

func printTags(tags []string) {
	if len(tags) > 0 {
		fmt.Fprintf(os.Stdout, "%-12s %s\n", "Tags:", strings.Join(tags, ", "))
	}
}

func printItemChildren(eng *engine.Engine, item *domain.Item) {
	children, err := eng.GetChildren(item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to get children: %v\n", err)
		return
	}
	if len(children) == 0 {
		return
	}

	fmt.Fprintln(os.Stdout, "\nChildren:")
	sortItemsByDisplayNum(children)
	for _, child := range children {
		fmt.Fprintf(os.Stdout, "  #%-4d [%s] %s — %s\n",
			child.DisplayNum, child.Type, child.Status, child.Title)
	}
}

func printItemHistory(item *domain.Item) {
	if len(item.History) == 0 {
		return
	}

	fmt.Fprintln(os.Stdout, "\nHistory:")
	for _, entry := range item.History {
		printHistoryEntry(entry)
	}
}

func printHistoryEntry(entry domain.ChangeRecord) {
	ts := entry.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stdout, "  %s  %-10s %s\n", ts, entry.Action, entry.Detail)
}
