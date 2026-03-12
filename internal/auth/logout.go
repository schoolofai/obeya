package auth

import (
	"github.com/niladribose/obeya/internal/store"
)

// Logout removes the stored credentials file.
func Logout(credsPath string) error {
	return store.DeleteCredentials(credsPath)
}

// IsLoggedIn checks if valid credentials exist at the given path.
func IsLoggedIn(credsPath string) bool {
	creds, err := store.LoadCredentials(credsPath)
	if err != nil {
		return false
	}
	return creds.Token != ""
}
