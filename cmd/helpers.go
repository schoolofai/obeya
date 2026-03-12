package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

func getStore() store.Store {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	s, err := store.NewStore(root, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return s
}

func getEngine() (*engine.Engine, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	root, err := store.FindProjectRoot(cwd)
	if err != nil {
		return nil, err
	}
	s, err := store.NewStore(root, "")
	if err != nil {
		return nil, err
	}
	if !s.BoardExists() {
		return nil, fmt.Errorf("no board found — run 'ob init' first")
	}
	return engine.New(s), nil
}

func getUserID() string {
	if flagAs != "" {
		return flagAs
	}
	if id := os.Getenv("OB_USER"); id != "" {
		return id
	}
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}

func getProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	gitRoot, err := store.FindGitRoot(cwd)
	if err != nil {
		return ""
	}
	linkFile := filepath.Join(gitRoot, ".obeya-link")
	if _, err := os.Stat(linkFile); err != nil {
		return "" // not linked, no project tag
	}
	return resolveProjectName(gitRoot)
}

func getSessionID() string {
	if flagSession != "" {
		return flagSession
	}
	if id := os.Getenv("OB_SESSION"); id != "" {
		return id
	}
	return fmt.Sprintf("pid-%d", os.Getpid())
}
