package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List board items with optional filters",
	Long:  "List items on the board. Supports filtering by status, assignee, type, tag, and blocked state. Supports tree and flat views.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().String("status", "", "filter by status column")
	listCmd.Flags().String("assignee", "", "filter by assignee")
	listCmd.Flags().String("type", "", "filter by item type (epic, story, task)")
	listCmd.Flags().String("tag", "", "filter by tag")
	listCmd.Flags().Bool("blocked", false, "show only blocked items")
	listCmd.Flags().Bool("flat", false, "flat list instead of tree view")
	listCmd.Flags().String("format", "", "output format (json)")
}

func runList(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	filter, err := buildListFilter(cmd)
	if err != nil {
		return err
	}

	items, err := eng.ListItems(filter)
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	flat, _ := cmd.Flags().GetBool("flat")

	if format == "json" {
		return printJSON(items)
	}

	sortItemsByDisplayNum(items)

	if flat || filter.Flat {
		printFlat(items)
		return nil
	}

	printTree(items)
	return nil
}

func buildListFilter(cmd *cobra.Command) (engine.ListFilter, error) {
	status, _ := cmd.Flags().GetString("status")
	assignee, _ := cmd.Flags().GetString("assignee")
	itemType, _ := cmd.Flags().GetString("type")
	tag, _ := cmd.Flags().GetString("tag")
	blocked, _ := cmd.Flags().GetBool("blocked")
	flat, _ := cmd.Flags().GetBool("flat")

	return engine.ListFilter{
		Status:   status,
		Assignee: assignee,
		Type:     itemType,
		Tag:      tag,
		Blocked:  blocked,
		Flat:     flat,
	}, nil
}

func printJSON(items []*domain.Item) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal items to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func sortItemsByDisplayNum(items []*domain.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayNum < items[j].DisplayNum
	})
}

func printFlat(items []*domain.Item) {
	if len(items) == 0 {
		fmt.Println("No items found.")
		return
	}
	fmt.Fprintf(os.Stdout, "%-6s %-8s %-10s %-12s %s\n", "#", "TYPE", "PRIORITY", "STATUS", "TITLE")
	fmt.Fprintf(os.Stdout, "%-6s %-8s %-10s %-12s %s\n", "---", "----", "--------", "------", "-----")
	for _, item := range items {
		printFlatRow(item)
	}
}

func printFlatRow(item *domain.Item) {
	title := item.Title
	if len(item.BlockedBy) > 0 {
		title = "[BLOCKED] " + title
	}
	fmt.Fprintf(os.Stdout, "%-6d %-8s %-10s %-12s %s\n",
		item.DisplayNum, string(item.Type), string(item.Priority), item.Status, title)
}

func printTree(items []*domain.Item) {
	if len(items) == 0 {
		fmt.Println("No items found.")
		return
	}

	roots, childMap := buildTreeMaps(items)
	for _, root := range roots {
		printTreeNode(root, childMap, 0)
	}
}

func buildTreeMaps(items []*domain.Item) ([]*domain.Item, map[string][]*domain.Item) {
	itemSet := make(map[string]bool, len(items))
	for _, item := range items {
		itemSet[item.ID] = true
	}

	childMap := make(map[string][]*domain.Item)
	var roots []*domain.Item

	for _, item := range items {
		if item.ParentID == "" || !itemSet[item.ParentID] {
			roots = append(roots, item)
		} else {
			childMap[item.ParentID] = append(childMap[item.ParentID], item)
		}
	}

	return roots, childMap
}

func printTreeNode(item *domain.Item, childMap map[string][]*domain.Item, depth int) {
	indent := strings.Repeat("  ", depth)
	blocked := ""
	if len(item.BlockedBy) > 0 {
		blocked = " [BLOCKED]"
	}
	fmt.Fprintf(os.Stdout, "%s#%-4d [%s] %-10s %s%s\n",
		indent, item.DisplayNum, item.Type, item.Status, item.Title, blocked)

	children := childMap[item.ID]
	sortItemsByDisplayNum(children)
	for _, child := range children {
		printTreeNode(child, childMap, depth+1)
	}
}
