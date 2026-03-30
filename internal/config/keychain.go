package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	keychainService = "dfir-cli"
)

// keychainUser returns the keychain username for a given profile.
func keychainUser(profile string) string {
	return "dfir-cli:" + profile
}

// SetKeychain stores the API key in the system keychain.
// Returns nil on success, an error if the keychain is unavailable.
func SetKeychain(profile, apiKey string) error {
	return keyring.Set(keychainService, keychainUser(profile), apiKey)
}

// GetKeychain retrieves the API key from the system keychain.
// Returns empty string and nil if no key is stored.
// Returns empty string and error if the keychain is unavailable.
func GetKeychain(profile string) (string, error) {
	secret, err := keyring.Get(keychainService, keychainUser(profile))
	if err == keyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("keychain access failed: %w", err)
	}
	return secret, nil
}

// DeleteKeychain removes the API key from the system keychain.
func DeleteKeychain(profile string) error {
	err := keyring.Delete(keychainService, keychainUser(profile))
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
