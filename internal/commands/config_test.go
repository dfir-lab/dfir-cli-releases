package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/config"
)

// ---------------------------------------------------------------------------
// TestIsValidKey
// ---------------------------------------------------------------------------

func TestIsValidKey(t *testing.T) {
	validKeys := []string{
		"api-key",
		"api-url",
		"output-format",
		"timeout",
		"concurrency",
		"no-color",
	}

	for _, key := range validKeys {
		t.Run("valid_"+key, func(t *testing.T) {
			if !isValidKey(key) {
				t.Errorf("isValidKey(%q) = false, want true", key)
			}
		})
	}

	invalidKeys := []string{
		"",
		"unknown",
		"API-KEY",
		"apikey",
		"api_key",
		"password",
		"api-key ",
		" api-key",
	}

	for _, key := range invalidKeys {
		t.Run("invalid_"+key, func(t *testing.T) {
			if isValidKey(key) {
				t.Errorf("isValidKey(%q) = true, want false", key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsValidOutputFormat
// ---------------------------------------------------------------------------

func TestIsValidOutputFormat(t *testing.T) {
	validFormats := []string{"table", "json", "jsonl", "csv"}

	for _, format := range validFormats {
		t.Run("valid_"+format, func(t *testing.T) {
			if !isValidOutputFormat(format) {
				t.Errorf("isValidOutputFormat(%q) = false, want true", format)
			}
		})
	}

	invalidFormats := []string{
		"",
		"xml",
		"yaml",
		"JSON",
		"Table",
		"text",
		"html",
	}

	for _, format := range invalidFormats {
		t.Run("invalid_"+format, func(t *testing.T) {
			if isValidOutputFormat(format) {
				t.Errorf("isValidOutputFormat(%q) = true, want false", format)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestApplyConfigValue
// ---------------------------------------------------------------------------

func TestApplyConfigValue(t *testing.T) {
	// Valid API key for testing (meets prefix + length requirements).
	validAPIKey := "sk-dfir-abc123def456ghij7890"

	tests := []struct {
		name      string
		key       string
		value     string
		wantErr   bool
		checkFunc func(t *testing.T, p *config.Profile)
	}{
		{
			name:  "api-key with valid key",
			key:   "api-key",
			value: validAPIKey,
			checkFunc: func(t *testing.T, p *config.Profile) {
				if p.APIKey != validAPIKey {
					t.Errorf("APIKey = %q, want %q", p.APIKey, validAPIKey)
				}
			},
		},
		{
			name:    "api-key with invalid key",
			key:     "api-key",
			value:   "invalid-key-no-prefix",
			wantErr: true,
		},
		{
			name:  "api-url with valid URL",
			key:   "api-url",
			value: "https://custom-api.example.com/v2",
			checkFunc: func(t *testing.T, p *config.Profile) {
				if p.APIURL != "https://custom-api.example.com/v2" {
					t.Errorf("APIURL = %q, want %q", p.APIURL, "https://custom-api.example.com/v2")
				}
			},
		},
		{
			name:    "api-url with empty value",
			key:     "api-url",
			value:   "",
			wantErr: true,
		},
		{
			name:  "output-format with json",
			key:   "output-format",
			value: "json",
			checkFunc: func(t *testing.T, p *config.Profile) {
				if p.OutputFormat != "json" {
					t.Errorf("OutputFormat = %q, want %q", p.OutputFormat, "json")
				}
			},
		},
		{
			name:    "output-format with xml (invalid)",
			key:     "output-format",
			value:   "xml",
			wantErr: true,
		},
		{
			name:  "timeout with 30s",
			key:   "timeout",
			value: "30s",
			checkFunc: func(t *testing.T, p *config.Profile) {
				want := 30 * time.Second
				if p.Timeout != want {
					t.Errorf("Timeout = %v, want %v", p.Timeout, want)
				}
			},
		},
		{
			name:    "timeout with invalid value",
			key:     "timeout",
			value:   "invalid",
			wantErr: true,
		},
		{
			name:    "timeout with negative duration",
			key:     "timeout",
			value:   "-5s",
			wantErr: true,
		},
		{
			name:  "concurrency with 10",
			key:   "concurrency",
			value: "10",
			checkFunc: func(t *testing.T, p *config.Profile) {
				if p.Concurrency != 10 {
					t.Errorf("Concurrency = %d, want %d", p.Concurrency, 10)
				}
			},
		},
		{
			name:    "concurrency with 0 (below minimum)",
			key:     "concurrency",
			value:   "0",
			wantErr: true,
		},
		{
			name:    "concurrency with 101 (above maximum)",
			key:     "concurrency",
			value:   "101",
			wantErr: true,
		},
		{
			name:  "no-color with true",
			key:   "no-color",
			value: "true",
			checkFunc: func(t *testing.T, p *config.Profile) {
				if !p.NoColor {
					t.Errorf("NoColor = false, want true")
				}
			},
		},
		{
			name:    "no-color with invalid value",
			key:     "no-color",
			value:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := config.DefaultProfile()
			err := applyConfigValue(p, tc.key, tc.value)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("applyConfigValue(%q, %q) returned nil error, want error", tc.key, tc.value)
				}
				return
			}

			if err != nil {
				t.Fatalf("applyConfigValue(%q, %q) returned unexpected error: %v", tc.key, tc.value, err)
			}

			if tc.checkFunc != nil {
				tc.checkFunc(t, p)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetConfigValue
// ---------------------------------------------------------------------------

func TestGetConfigValue(t *testing.T) {
	p := &config.Profile{
		APIKey:       "sk-dfir-test1234567890ab",
		APIURL:       "https://example.com/api/v1",
		OutputFormat: "json",
		Timeout:      45 * time.Second,
		Concurrency:  8,
		NoColor:      true,
	}

	tests := []struct {
		key  string
		want string
	}{
		{"api-key", "sk-dfir-test1234567890ab"},
		{"api-url", "https://example.com/api/v1"},
		{"output-format", "json"},
		{"timeout", "45s"},
		{"concurrency", fmt.Sprintf("%d", 8)},
		{"no-color", "true"},
		{"unknown-key", ""},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := getConfigValue(p, tc.key)
			if got != tc.want {
				t.Errorf("getConfigValue(p, %q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestSaveInitProfileFirstRunWritesActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := config.DefaultProfile()
	p.APIKey = "sk-dfir-inittestkey123456789"

	if err := saveInitProfile("default", p); err != nil {
		t.Fatalf("saveInitProfile failed: %v", err)
	}

	data, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}
	if !strings.Contains(string(data), "active_profile: default") {
		t.Fatalf("config file missing active_profile metadata:\n%s", string(data))
	}

	loaded, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") failed: %v", err)
	}
	if loaded.APIKey != p.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, p.APIKey)
	}
}

func TestSaveInitProfilePreservesExistingProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p1 := config.DefaultProfile()
	p1.APIKey = "sk-dfir-defaultprofilekey123"
	if err := saveInitProfile("default", p1); err != nil {
		t.Fatalf("saveInitProfile default failed: %v", err)
	}

	p2 := config.DefaultProfile()
	p2.APIKey = "sk-dfir-stagingprofilekey12"
	if err := saveInitProfile("staging", p2); err != nil {
		t.Fatalf("saveInitProfile staging failed: %v", err)
	}

	loadedDefault, err := config.Load("default")
	if err != nil {
		t.Fatalf("Load(default) failed: %v", err)
	}
	if loadedDefault.APIKey != p1.APIKey {
		t.Errorf("default APIKey = %q, want %q", loadedDefault.APIKey, p1.APIKey)
	}

	loadedStaging, err := config.Load("staging")
	if err != nil {
		t.Fatalf("Load(staging) failed: %v", err)
	}
	if loadedStaging.APIKey != p2.APIKey {
		t.Errorf("staging APIKey = %q, want %q", loadedStaging.APIKey, p2.APIKey)
	}
}

func TestIsFatalAPIKeyValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "authentication error is fatal",
			err:  &client.AuthenticationError{Message: "bad key"},
			want: true,
		},
		{
			name: "authorization error is fatal",
			err:  &client.AuthorizationError{Message: "forbidden"},
			want: true,
		},
		{
			name: "rate limit is non-fatal",
			err:  &client.RateLimitError{Message: "slow down"},
			want: false,
		},
		{
			name: "generic error is non-fatal",
			err:  errors.New("network down"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isFatalAPIKeyValidationError(tc.err)
			if got != tc.want {
				t.Errorf("isFatalAPIKeyValidationError(%T) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestValidateAPIKeyWithPlatform(t *testing.T) {
	orig := authValidateAPIKey
	t.Cleanup(func() { authValidateAPIKey = orig })

	t.Run("fatal auth error blocks config init", func(t *testing.T) {
		authValidateAPIKey = func(ctx context.Context, apiKey, apiURL string, timeout time.Duration, verbose bool) error {
			return &client.AuthenticationError{Message: "invalid API key"}
		}

		err := validateAPIKeyWithPlatform("sk-dfir-validformatkey12345")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var authErr *client.AuthenticationError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected AuthenticationError, got %T (%v)", err, err)
		}
	})

	t.Run("transient errors are non-fatal", func(t *testing.T) {
		authValidateAPIKey = func(ctx context.Context, apiKey, apiURL string, timeout time.Duration, verbose bool) error {
			return errors.New("dial tcp: timeout")
		}

		if err := validateAPIKeyWithPlatform("sk-dfir-validformatkey12345"); err != nil {
			t.Fatalf("expected nil error on transient failure, got %v", err)
		}
	})
}
