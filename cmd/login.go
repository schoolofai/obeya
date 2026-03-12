package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/niladribose/obeya/internal/auth"
	"github.com/niladribose/obeya/internal/store"
	"github.com/spf13/cobra"
)

var loginAppURL string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Obeya Cloud",
	Long:  "Opens a browser for OAuth authentication. On success, stores API token in ~/.obeya/credentials.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		credsPath, err := store.DefaultCredentialsPath()
		if err != nil {
			return err
		}

		if auth.IsLoggedIn(credsPath) {
			fmt.Println("Already logged in. Run 'ob logout' first to re-authenticate.")
			return nil
		}

		return runLoginFlow(credsPath)
	},
}

func runLoginFlow(credsPath string) error {
	srv, tokenCh, errCh := auth.NewCallbackServer(auth.DefaultLoginPort)
	defer srv.Close()

	addr := srv.Addr()
	if addr == "" {
		return fmt.Errorf("failed to start callback server")
	}

	callbackURL := fmt.Sprintf("http://%s/callback", addr)
	loginURL := auth.BuildLoginURL(loginAppURL, callbackURL)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", loginURL)
	fmt.Println("Waiting for authentication...")

	openBrowser(loginURL)

	return awaitLoginResult(tokenCh, errCh, credsPath)
}

func awaitLoginResult(tokenCh <-chan *auth.CallbackResult, errCh <-chan error, credsPath string) error {
	select {
	case result := <-tokenCh:
		if err := auth.SaveLoginCredentials(credsPath, result); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		fmt.Printf("\nLogged in successfully. Token stored at %s\n", credsPath)
		return nil

	case err := <-errCh:
		return fmt.Errorf("login failed: %w", err)

	case <-time.After(5 * time.Minute):
		return fmt.Errorf("login timed out after 5 minutes — no callback received")
	}
}

func init() {
	loginCmd.Flags().StringVar(&loginAppURL, "app-url", auth.DefaultAppURL, "Obeya Cloud app URL")
	rootCmd.AddCommand(loginCmd)
}

// openBrowser opens a URL in the default browser.
func openBrowser(browserURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", browserURL)
	case "linux":
		cmd = exec.Command("xdg-open", browserURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", browserURL)
	default:
		return
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
