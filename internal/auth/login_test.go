package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/niladribose/obeya/internal/auth"
)

func TestParseCallbackToken(t *testing.T) {
	token, err := auth.ParseCallbackToken("http://localhost:9876/callback?token=ob_tok_abc123&user_id=usr_456")
	if err != nil {
		t.Fatalf("ParseCallbackToken failed: %v", err)
	}
	if token.Token != "ob_tok_abc123" {
		t.Errorf("Token: got %q, want 'ob_tok_abc123'", token.Token)
	}
	if token.UserID != "usr_456" {
		t.Errorf("UserID: got %q, want 'usr_456'", token.UserID)
	}
}

func TestParseCallbackToken_MissingToken(t *testing.T) {
	_, err := auth.ParseCallbackToken("http://localhost:9876/callback?user_id=usr_456")
	if err == nil {
		t.Fatal("expected error for missing token param")
	}
}

func TestParseCallbackToken_ErrorParam(t *testing.T) {
	_, err := auth.ParseCallbackToken("http://localhost:9876/callback?error=access_denied&error_description=User+denied+access")
	if err == nil {
		t.Fatal("expected error when error param present")
	}
}

func TestBuildLoginURL(t *testing.T) {
	url := auth.BuildLoginURL("https://obeya.app", "http://localhost:9876/callback")
	expected := "https://obeya.app/auth/cli?callback=http%3A%2F%2Flocalhost%3A9876%2Fcallback"
	if url != expected {
		t.Errorf("BuildLoginURL: got %q, want %q", url, expected)
	}
}

func TestCallbackServer_ReceivesToken(t *testing.T) {
	srv, tokenCh, errCh := auth.NewCallbackServer(0) // port 0 = random available port
	defer srv.Close()

	go func() {
		addr := srv.Addr()
		callbackURL := "http://" + addr + "/callback?token=ob_tok_test123&user_id=usr_test"
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	select {
	case result := <-tokenCh:
		if result.Token != "ob_tok_test123" {
			t.Errorf("Token: got %q, want 'ob_tok_test123'", result.Token)
		}
		if result.UserID != "usr_test" {
			t.Errorf("UserID: got %q, want 'usr_test'", result.UserID)
		}
	case err := <-errCh:
		t.Fatalf("callback server error: %v", err)
	}
}

func TestCallbackServer_HandlesError(t *testing.T) {
	srv, tokenCh, errCh := auth.NewCallbackServer(0)
	defer srv.Close()

	go func() {
		addr := srv.Addr()
		callbackURL := "http://" + addr + "/callback?error=access_denied&error_description=Denied"
		resp, err := http.Get(callbackURL)
		if err != nil {
			t.Errorf("callback request failed: %v", err)
			return
		}
		resp.Body.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	case <-tokenCh:
		t.Fatal("expected error, not token")
	}
}

func TestSaveAndVerifyCredentials(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "credentials.json")

	result := &auth.CallbackResult{
		Token:  "ob_tok_verify",
		UserID: "usr_v",
	}

	err := auth.SaveLoginCredentials(credsPath, result)
	if err != nil {
		t.Fatalf("SaveLoginCredentials failed: %v", err)
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("failed to read saved credentials: %v", err)
	}

	var creds map[string]string
	json.Unmarshal(data, &creds)
	if creds["token"] != "ob_tok_verify" {
		t.Errorf("token: got %q, want 'ob_tok_verify'", creds["token"])
	}
}

func TestCallbackServer_Addr(t *testing.T) {
	srv, _, _ := auth.NewCallbackServer(0)
	defer srv.Close()

	addr := srv.Addr()
	if addr == "" {
		t.Error("expected non-empty address from Addr()")
	}
}

// Verify constants are exported
func TestConstants(t *testing.T) {
	if auth.DefaultLoginPort != 9876 {
		t.Errorf("DefaultLoginPort: got %d, want 9876", auth.DefaultLoginPort)
	}
	if auth.DefaultAppURL != "https://obeya.app" {
		t.Errorf("DefaultAppURL: got %q, want 'https://obeya.app'", auth.DefaultAppURL)
	}
}

// Verify httptest import used (satisfies import requirement in test file)
var _ = httptest.NewServer
