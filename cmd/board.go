package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Board configuration and column management",
}

var boardConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show board configuration",
	Run:   runBoardConfig,
}

var boardColumnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "Manage board columns",
}

var boardColumnsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a column to the board",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardColumnsAdd,
}

var boardColumnsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a column from the board",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardColumnsRemove,
}

var boardColumnsReorderCmd = &cobra.Command{
	Use:   "reorder <col1,col2,...>",
	Short: "Reorder board columns",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardColumnsReorder,
}

func init() {
	boardColumnsCmd.AddCommand(boardColumnsAddCmd, boardColumnsRemoveCmd, boardColumnsReorderCmd)
	boardCmd.AddCommand(boardConfigCmd, boardColumnsCmd)
	rootCmd.AddCommand(boardCmd)
}

func runBoardConfig(cmd *cobra.Command, args []string) {
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
		printBoardConfigJSON(board)
		return
	}
	printBoardConfigText(board)
}

type boardConfigOutput struct {
	Name      string   `json:"name"`
	Version   int      `json:"version"`
	AgentRole string   `json:"agent_role"`
	Columns   []string `json:"columns"`
	ItemCount int      `json:"item_count"`
	UserCount int      `json:"user_count"`
}

func buildBoardConfigOutput(board *domain.Board) boardConfigOutput {
	cols := make([]string, len(board.Columns))
	for i, c := range board.Columns {
		cols[i] = c.Name
	}
	return boardConfigOutput{
		Name:      board.Name,
		Version:   board.Version,
		AgentRole: board.AgentRole,
		Columns:   cols,
		ItemCount: len(board.Items),
		UserCount: len(board.Users),
	}
}

func printBoardConfigJSON(board *domain.Board) {
	data, err := json.MarshalIndent(buildBoardConfigOutput(board), "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func printBoardConfigText(board *domain.Board) {
	cfg := buildBoardConfigOutput(board)
	fmt.Printf("Board:      %s\n", cfg.Name)
	fmt.Printf("Version:    %d\n", cfg.Version)
	fmt.Printf("Agent Role: %s\n", cfg.AgentRole)
	fmt.Printf("Columns:    %s\n", strings.Join(cfg.Columns, ", "))
	fmt.Printf("Items:      %d\n", cfg.ItemCount)
	fmt.Printf("Users:      %d\n", cfg.UserCount)
}

func runBoardColumnsAdd(cmd *cobra.Command, args []string) {
	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := eng.AddColumn(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Column %q added\n", args[0])
}

func runBoardColumnsRemove(cmd *cobra.Command, args []string) {
	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := eng.RemoveColumn(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Column %q removed\n", args[0])
}

func runBoardColumnsReorder(cmd *cobra.Command, args []string) {
	eng, err := getEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	names := strings.Split(args[0], ",")
	if err := eng.ReorderColumns(names); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Columns reordered: %s\n", strings.Join(names, ", "))
}
