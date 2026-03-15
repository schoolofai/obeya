package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/niladribose/obeya/internal/agent"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var initColumns string
var initAgent string
var initSkipPlugin bool
var initRoot string
var initShared string

// cloudInitHandler is set by init_cloud.go when built with -tags cloud.
// Returns (handled bool, err error). If handled is true, the command is done.
var cloudInitHandler func(cmd *cobra.Command, args []string, columns []string) (bool, error)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Obeya board",
	Long:  "Initialize a new Obeya board with agent integration. Requires --agent flag.\nUse --shared for storage-only boards. Combine --shared with --agent for global Obeya.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		columns := parseColumns(initColumns)

		// Cloud init is handled by init_cloud.go (built with -tags cloud)
		if cloudInitHandler != nil {
			handled, err := cloudInitHandler(cmd, args, columns)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
		}

		// Shared + agent = shared board with agent setup
		if initShared != "" && initAgent != "" {
			return initSharedBoardWithAgent(initShared, initAgent, columns)
		}

		// Shared without agent is no longer supported
		if initShared != "" {
			return fmt.Errorf("--shared requires --agent. Supported agents: %s", strings.Join(agent.SupportedNames(), ", "))
		}

		// --agent is required for non-shared boards
		if initAgent == "" {
			return fmt.Errorf("required flag --agent not provided. Supported: %s", strings.Join(agent.SupportedNames(), ", "))
		}

		// Validate agent name
		agentSetup, err := agent.Get(initAgent)
		if err != nil {
			return err
		}

		root, err := resolveInitRoot()
		if err != nil {
			return err
		}

		s := store.NewJSONStore(root)

		boardName := "obeya"
		if len(args) > 0 {
			boardName = args[0]
		}

		err = s.InitBoard(boardName, columns)
		if err != nil {
			if !strings.Contains(err.Error(), "already initialized") {
				return err
			}
			fmt.Printf("Board already initialized in %s/.obeya/\n", root)
		} else {
			fmt.Printf("Board %q initialized in %s/.obeya/\n", boardName, root)
			if len(columns) > 0 {
				fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))
			} else {
				fmt.Println("Columns: backlog, todo, in-progress, review, done")
			}
		}

		// Register agent and human users
		if err := registerInitUsers(s, initAgent); err != nil {
			return err
		}

		// Delegate to agent-specific setup
		ctx := agent.AgentContext{
			Root:       root,
			BoardName:  boardName,
			SkipPlugin: initSkipPlugin,
		}
		if err := agentSetup.Setup(ctx); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initColumns, "columns", "", "comma-separated column names (default: backlog,todo,in-progress,review,done)")
	initCmd.Flags().StringVar(&initAgent, "agent", "", "coding agent to configure (supported: claude-code)")
	initCmd.Flags().BoolVar(&initSkipPlugin, "skip-plugin", false, "skip plugin installation")
	initCmd.Flags().StringVar(&initRoot, "root", "", "directory to create .obeya in (default: git repository root)")
	initCmd.Flags().StringVar(&initShared, "shared", "", "create a shared board at ~/.obeya/boards/<name>")
	rootCmd.AddCommand(initCmd)
}

func initSharedBoardWithAgent(sharedName, agentName string, columns []string) error {
	agentSetup, err := agent.Get(agentName)
	if err != nil {
		return err
	}

	// Create shared board
	if err := initSharedBoard(sharedName, columns); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
		fmt.Printf("Shared board %q already exists — proceeding with agent setup\n", sharedName)
	}

	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}
	boardDir := store.SharedBoardDir(obeyaHome, sharedName)

	// Register agent and human users
	s := store.NewJSONStore(boardDir)
	if err := registerInitUsers(s, agentName); err != nil {
		return err
	}

	ctx := agent.AgentContext{
		Root:       boardDir,
		BoardName:  sharedName,
		SkipPlugin: initSkipPlugin,
		Shared:     true,
	}
	return agentSetup.Setup(ctx)
}

func initSharedBoard(boardName string, columns []string) error {
	obeyaHome, err := store.ObeyaHome()
	if err != nil {
		return err
	}

	boardDir := store.SharedBoardDir(obeyaHome, boardName)
	boardFile := filepath.Join(boardDir, ".obeya", "board.json")

	if _, err := os.Stat(boardFile); err == nil {
		return fmt.Errorf("board %q already exists — use 'ob link %s' to connect this project", boardName, boardName)
	}

	s := store.NewJSONStore(boardDir)
	if err := s.InitBoard(boardName, columns); err != nil {
		return err
	}

	fmt.Printf("Shared board %q initialized at %s\n", boardName, boardDir)
	return nil
}

func parseColumns(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func agentDisplayName(provider string) (string, error) {
	switch provider {
	case "claude-code":
		return "Claude", nil
	default:
		return "", fmt.Errorf("unknown agent provider %q — cannot determine display name", provider)
	}
}

func resolveHumanName() (string, error) {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err == nil {
		name := strings.TrimSpace(string(out))
		if name != "" {
			return name, nil
		}
	}

	u, err := user.Current()
	if err == nil {
		if u.Name != "" {
			return u.Name, nil
		}
		if u.Username != "" {
			return u.Username, nil
		}
	}

	return "", fmt.Errorf("cannot determine human user name: git config user.name is empty and os/user lookup failed")
}

func registerInitUsers(s store.Store, agentProvider string) error {
	eng := engine.New(s)

	agentName, err := agentDisplayName(agentProvider)
	if err != nil {
		return err
	}
	added, err := eng.AddUser(agentName, "agent", agentProvider)
	if err != nil {
		return fmt.Errorf("failed to register agent user: %w", err)
	}
	if added {
		fmt.Printf("Registered agent user: %s\n", agentName)
	}

	humanName, err := resolveHumanName()
	if err != nil {
		return err
	}
	added, err = eng.AddUser(humanName, "human", "local")
	if err != nil {
		return fmt.Errorf("failed to register human user: %w", err)
	}
	if added {
		fmt.Printf("Registered human user: %s\n", humanName)
	}

	return nil
}

func resolveInitRoot() (string, error) {
	if initRoot != "" {
		abs, err := filepath.Abs(initRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve --root path: %w", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("--root path does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("--root path is not a directory: %s", abs)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return store.FindGitRoot(cwd)
}

