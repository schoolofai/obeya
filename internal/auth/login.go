package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/niladribose/obeya/internal/store"
)

// DefaultLoginPort is the port used for the OAuth callback server.
const DefaultLoginPort = 9876

// DefaultAppURL is the default Obeya Cloud app URL.
const DefaultAppURL = "https://obeya.app"

// CallbackResult holds the token data received from the OAuth callback.
type CallbackResult struct {
	Token  string
	UserID string
}

// CallbackServer wraps an HTTP server that listens for the OAuth callback.
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
}

// NewCallbackServer creates and starts a local HTTP server for receiving OAuth callbacks.
// Pass port 0 for a random available port. Returns the server, a channel for the
// token result, and a channel for errors.
func NewCallbackServer(port int) (*CallbackServer, <-chan *CallbackResult, <-chan error) {
	tokenCh := make(chan *CallbackResult, 1)
	errCh := make(chan error, 1)

	listener, err := startListener(port)
	if err != nil {
		errCh <- err
		return &CallbackServer{}, tokenCh, errCh
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", buildCallbackHandler(tokenCh, errCh))

	server := &http.Server{Handler: mux}
	go serveCallback(server, listener, errCh)

	return &CallbackServer{server: server, listener: listener}, tokenCh, errCh
}

// startListener creates a TCP listener on the given port (0 = random).
func startListener(port int) (net.Listener, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on %s: %w", addr, err)
	}
	return listener, nil
}

// buildCallbackHandler returns an HTTP handler for the /callback endpoint.
func buildCallbackHandler(tokenCh chan<- *CallbackResult, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := ParseCallbackToken(r.URL.String())
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h1>Login Failed</h1><p>%s</p><p>You can close this window.</p></body></html>", err.Error())
			errCh <- err
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h1>Login Successful</h1><p>You can close this window and return to the terminal.</p></body></html>")
		tokenCh <- result
	}
}

// serveCallback runs the HTTP server and sends errors to errCh.
func serveCallback(server *http.Server, listener net.Listener, errCh chan<- error) {
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		errCh <- fmt.Errorf("callback server error: %w", err)
	}
}

// Addr returns the address the callback server is listening on.
func (cs *CallbackServer) Addr() string {
	if cs.listener == nil {
		return ""
	}
	return cs.listener.Addr().String()
}

// Close shuts down the callback server gracefully.
func (cs *CallbackServer) Close() error {
	if cs.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return cs.server.Shutdown(ctx)
}

// ParseCallbackToken extracts token and user_id from an OAuth callback URL.
func ParseCallbackToken(rawURL string) (*CallbackResult, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse callback URL: %w", err)
	}

	params := parsed.Query()

	if errParam := params.Get("error"); errParam != "" {
		desc := params.Get("error_description")
		return nil, fmt.Errorf("authentication failed: %s — %s", errParam, desc)
	}

	token := params.Get("token")
	if token == "" {
		return nil, fmt.Errorf("callback URL missing 'token' parameter")
	}

	return &CallbackResult{
		Token:  token,
		UserID: params.Get("user_id"),
	}, nil
}

// BuildLoginURL constructs the URL to open in the browser for CLI OAuth login.
func BuildLoginURL(appURL, callbackURL string) string {
	return fmt.Sprintf("%s/auth/cli?callback=%s", appURL, url.QueryEscape(callbackURL))
}

// SaveLoginCredentials saves the received token to the credentials file.
func SaveLoginCredentials(credsPath string, result *CallbackResult) error {
	creds := &store.Credentials{
		Token:     result.Token,
		UserID:    result.UserID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return store.SaveCredentials(credsPath, creds)
}
