package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage plans (create, import, link to items)",
}

var planCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new plan",
	Args:  cobra.NoArgs,
	RunE:  runPlanCreate,
}

var planImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a plan from a markdown file",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanImport,
}

var planUpdateCmd = &cobra.Command{
	Use:   "update <plan-id> [file]",
	Short: "Update a plan's title or content",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPlanUpdate,
}

var planShowCmd = &cobra.Command{
	Use:   "show <plan-id>",
	Short: "Show plan details",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanShow,
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plans",
	Args:  cobra.NoArgs,
	RunE:  runPlanList,
}

var planLinkCmd = &cobra.Command{
	Use:   "link <plan-id>",
	Short: "Link items to a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanLink,
}

var planUnlinkCmd = &cobra.Command{
	Use:   "unlink <plan-id>",
	Short: "Unlink items from a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanUnlink,
}

var planDeleteCmd = &cobra.Command{
	Use:   "delete <plan-id>",
	Short: "Delete a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanDelete,
}

func init() {
	planCreateCmd.Flags().String("title", "", "plan title (required)")
	planImportCmd.Flags().String("link", "", "comma-separated item IDs to link")
	planImportCmd.Flags().String("title", "", "override title from file")
	planUpdateCmd.Flags().String("title", "", "new title")
	planLinkCmd.Flags().String("to", "", "comma-separated item IDs to link")
	planUnlinkCmd.Flags().String("from", "", "comma-separated item IDs to unlink")

	planCmd.AddCommand(planCreateCmd, planImportCmd, planUpdateCmd, planShowCmd, planListCmd, planLinkCmd, planUnlinkCmd, planDeleteCmd)
	rootCmd.AddCommand(planCmd)
}

func runPlanCreate(cmd *cobra.Command, args []string) error {
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	eng, err := getEngine()
	if err != nil {
		return err
	}

	plan, err := eng.CreatePlan(title, "", "")
	if err != nil {
		return err
	}

	return printPlanCreated(plan.DisplayNum, plan.ID, plan.Title)
}

func runPlanImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	linkFlag, _ := cmd.Flags().GetString("link")
	linkRefs := parseCommaSeparated(linkFlag)

	eng, err := getEngine()
	if err != nil {
		return err
	}

	plan, err := eng.ImportPlan(string(data), filePath, linkRefs)
	if err != nil {
		return err
	}

	titleOverride, _ := cmd.Flags().GetString("title")
	if titleOverride != "" {
		if err := eng.UpdatePlan(fmt.Sprintf("%d", plan.DisplayNum), titleOverride, ""); err != nil {
			return fmt.Errorf("imported plan but failed to override title: %w", err)
		}
	}

	return printPlanCreated(plan.DisplayNum, plan.ID, plan.Title)
}

func runPlanUpdate(cmd *cobra.Command, args []string) error {
	ref := args[0]
	title, _ := cmd.Flags().GetString("title")

	var content string
	if len(args) == 2 {
		data, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("failed to read file %q: %w", args[1], err)
		}
		content = string(data)
	}

	eng, err := getEngine()
	if err != nil {
		return err
	}

	if err := eng.UpdatePlan(ref, title, content); err != nil {
		return err
	}

	fmt.Printf("Plan %s updated\n", ref)
	return nil
}

func runPlanShow(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	plan, err := eng.ShowPlan(args[0])
	if err != nil {
		return err
	}

	if flagFormat == "json" {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal plan to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Plan #%d: %s\n", plan.DisplayNum, plan.Title)
	if plan.SourceFile != "" {
		fmt.Printf("  Source: %s\n", plan.SourceFile)
	}
	fmt.Printf("  Linked items: %d\n", len(plan.LinkedItems))
	fmt.Printf("  Created: %s\n", plan.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("  Updated: %s\n", plan.UpdatedAt.Format("2006-01-02 15:04"))
	if plan.Content != "" {
		fmt.Printf("\n%s\n", plan.Content)
	}
	return nil
}

func runPlanList(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	plans, err := eng.ListPlans()
	if err != nil {
		return err
	}

	if flagFormat == "json" {
		data, err := json.MarshalIndent(plans, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal plans to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(plans) == 0 {
		fmt.Println("No plans found.")
		return nil
	}

	for _, p := range plans {
		fmt.Printf("#%d  %s  (%d linked items)\n", p.DisplayNum, p.Title, len(p.LinkedItems))
	}
	return nil
}

func runPlanLink(cmd *cobra.Command, args []string) error {
	toFlag, _ := cmd.Flags().GetString("to")
	if toFlag == "" {
		return fmt.Errorf("--to is required")
	}

	eng, err := getEngine()
	if err != nil {
		return err
	}

	refs := parseCommaSeparated(toFlag)
	if err := eng.LinkPlan(args[0], refs); err != nil {
		return err
	}

	fmt.Printf("Linked %d item(s) to plan %s\n", len(refs), args[0])
	return nil
}

func runPlanUnlink(cmd *cobra.Command, args []string) error {
	fromFlag, _ := cmd.Flags().GetString("from")
	if fromFlag == "" {
		return fmt.Errorf("--from is required")
	}

	eng, err := getEngine()
	if err != nil {
		return err
	}

	refs := parseCommaSeparated(fromFlag)
	if err := eng.UnlinkPlan(args[0], refs); err != nil {
		return err
	}

	fmt.Printf("Unlinked %d item(s) from plan %s\n", len(refs), args[0])
	return nil
}

func runPlanDelete(cmd *cobra.Command, args []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	if err := eng.DeletePlan(args[0]); err != nil {
		return err
	}

	fmt.Printf("Plan %s deleted\n", args[0])
	return nil
}

func printPlanCreated(displayNum int, id, title string) error {
	if flagFormat == "json" {
		data, err := json.Marshal(map[string]interface{}{
			"display_num": displayNum,
			"id":          id,
			"title":       title,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal plan to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Created plan #%d [%s]: %s\n", displayNum, id[:6], title)
	return nil
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
