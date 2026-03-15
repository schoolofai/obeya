package cmd

import (
	"fmt"
	"log"
	"os"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	obeyamcp "github.com/niladribose/obeya/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Model Context Protocol) server",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server (stdio transport)",
	Long: `Start an MCP server that exposes all Obeya board operations as tools.

Any MCP-compatible host (Claude Code, Claude Desktop, Cursor, Windsurf, ChatGPT)
can connect to this server to manage your board.

Configuration for Claude Code (.mcp.json):
  {
    "mcpServers": {
      "obeya": {
        "command": "ob",
        "args": ["mcp", "serve"]
      }
    }
  }`,
	RunE:          runMCPServe,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var flagMCPBoard string

func init() {
	mcpServeCmd.Flags().StringVar(&flagMCPBoard, "board", "", "path to the project directory containing the board")
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	// Change to board directory if specified
	if flagMCPBoard != "" {
		if err := os.Chdir(flagMCPBoard); err != nil {
			return fmt.Errorf("cannot change to board directory %s: %w", flagMCPBoard, err)
		}
	}

	eng, err := getEngine()
	if err != nil {
		return fmt.Errorf("no board found: %w\n\nRun 'ob init' to create a board first", err)
	}

	srv := obeyamcp.New(eng)

	// Serve over stdio — blocks until stdin closes
	if err := mcpserver.ServeStdio(srv.MCPServer()); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}

	return nil
}
