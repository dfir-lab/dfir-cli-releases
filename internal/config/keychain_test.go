package config

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	// Use the in-memory mock provider so tests do not touch the real keychain.
	keyring.MockInit()
}

func TestSetAndGetKeychain(t *testing.T) {
	profile := "test-profile"
	apiKey := "sk-dfir-test1234567890abcdef"

	if err := SetKeychain(profile, apiKey); err != nil {
		t.Fatalf("SetKeychain() error = %v", err)
	}

	got, err := GetKeychain(profile)
	if err != nil {
		t.Fatalf("GetKeychain() error = %v", err)
	}
	if got != apiKey {
		t.Errorf("GetKeychain() = %q, want %q", got, apiKey)
	}
}

func TestGetKeychainNotFound(t *testing.T) {
	got, err := GetKeychain("nonexistent-profile")
	if err != nil {
		t.Fatalf("GetKeychain() error = %v, want nil for missing key", err)
	}
	if got != "" {
		t.Errorf("GetKeychain() = %q, want empty string for missing key", got)
	}
}

func TestDeleteKeychain(t *testing.T) {
	profile := "delete-test"
	apiKey := "sk-dfir-deletetest1234567890"

	if err := SetKeychain(profile, apiKey); err != nil {
		t.Fatalf("SetKeychain() error = %v", err)
	}

	if err := DeleteKeychain(profile); err != nil {
		t.Fatalf("DeleteKeychain() error = %v", err)
	}

	got, err := GetKeychain(profile)
	if err != nil {
		t.Fatalf("GetKeychain() after delete error = %v", err)
	}
	if got != "" {
		t.Errorf("GetKeychain() after delete = %q, want empty string", got)
	}
}

func TestDeleteKeychainNotFound(t *testing.T) {
	// Deleting a non-existent entry should not return an error.
	if err := DeleteKeychain("no-such-profile"); err != nil {
		t.Errorf("DeleteKeychain() for missing key = %v, want nil", err)
	}
}

func TestKeychainUser(t *testing.T) {
	got := keychainUser("myprofile")
	want := "dfir-cli:myprofile"
	if got != want {
		t.Errorf("keychainUser() = %q, want %q", got, want)
	}
}
