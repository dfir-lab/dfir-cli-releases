package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/fatih/color"
)

// ---------------------------------------------------------------------------
// resolvePhishingInput
// ---------------------------------------------------------------------------

func TestResolvePhishingInput_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.eml")

	body := "From: sender@example.com\r\nTo: victim@example.com\r\nSubject: Test\r\n\r\nHello"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	content, inputType, err := resolvePhishingInput(path, "")
	if err != nil {
		t.Fatalf("resolvePhishingInput returned error: %v", err)
	}

	if content != body {
		t.Errorf("content = %q, want %q", content, body)
	}
	if inputType != "eml" {
		t.Errorf("inputType = %q, want %q", inputType, "eml")
	}
}

func TestResolvePhishingInput_RawFlag(t *testing.T) {
	raw := "From: attacker@evil.com\nSubject: Urgent"

	content, inputType, err := resolvePhishingInput("", raw)
	if err != nil {
		t.Fatalf("resolvePhishingInput returned error: %v", err)
	}

	if content != raw {
		t.Errorf("content = %q, want %q", content, raw)
	}
	if inputType != "raw" {
		t.Errorf("inputType = %q, want %q", inputType, "raw")
	}
}

func TestResolvePhishingInput_NoInput(t *testing.T) {
	// No file, no raw, and stdin is not a pipe — should return an error with
	// a usage hint.
	_, _, err := resolvePhishingInput("", "")
	if err == nil {
		t.Fatal("expected error when no input is provided")
	}
	if !strings.Contains(err.Error(), "no input provided") {
		t.Errorf("error should mention 'no input provided', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Usage:") {
		t.Errorf("error should contain usage hint, got: %v", err)
	}
}

func TestResolvePhishingInput_FilePrecedesRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.eml")

	body := "From: sender@example.com\r\nSubject: Test\r\n\r\nBody"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// When both --file and --raw are supplied, --file takes precedence.
	content, inputType, err := resolvePhishingInput(path, "raw content")
	if err != nil {
		t.Fatalf("resolvePhishingInput returned error: %v", err)
	}
	if content != body {
		t.Errorf("content = %q, want file body %q", content, body)
	}
	if inputType != "eml" {
		t.Errorf("inputType = %q, want %q", inputType, "eml")
	}
}

func TestNewPhishingAnalyzeCmd_URLFlagIsHiddenAndDeprecated(t *testing.T) {
	cmd := newPhishingAnalyzeCmd()
	flag := cmd.Flags().Lookup("url")
	if flag == nil {
		t.Fatal("expected legacy --url flag to exist")
	}
	if !flag.Hidden {
		t.Error("expected --url flag to be hidden")
	}
	if flag.Deprecated == "" {
		t.Error("expected --url flag to be deprecated")
	}
	if !strings.Contains(flag.Deprecated, "enrichment lookup --url") {
		t.Errorf("deprecated message should contain migration hint, got: %q", flag.Deprecated)
	}
}

func TestRunPhishingAnalyze_LegacyURLFlagError(t *testing.T) {
	err := runPhishingAnalyze("https://phishing.example.com", "", "", "", false)
	if err == nil {
		t.Fatal("expected error for legacy --url flag")
	}
	if !strings.Contains(err.Error(), "--url is not supported for phishing analysis") {
		t.Errorf("error should mention unsupported --url, got: %v", err)
	}
	if !strings.Contains(err.Error(), "enrichment lookup --url") {
		t.Errorf("error should include alternative command, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// readPhishingEmailFile
// ---------------------------------------------------------------------------

func TestReadPhishingEmailFile_NotFound(t *testing.T) {
	_, _, err := readPhishingEmailFile("/tmp/nonexistent_dfir_phishing_test.eml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error should mention 'file not found', got: %v", err)
	}
}

func TestReadPhishingEmailFile_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.eml")

	// Write a file that exceeds the 5 MB limit by one byte.
	data := make([]byte, maxEmailFileSize+1)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write oversized file: %v", err)
	}

	_, _, err := readPhishingEmailFile(path)
	if err == nil {
		t.Fatal("expected error for file exceeding size limit")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error should mention 'too large', got: %v", err)
	}
}

func TestReadPhishingEmailFile_EmlExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "message.eml")

	body := "From: test@example.com\r\nSubject: Hi\r\n\r\nBody"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	content, inputType, err := readPhishingEmailFile(path)
	if err != nil {
		t.Fatalf("readPhishingEmailFile returned error: %v", err)
	}
	if content != body {
		t.Errorf("content = %q, want %q", content, body)
	}
	if inputType != "eml" {
		t.Errorf("inputType = %q, want %q", inputType, "eml")
	}
}

func TestReadPhishingEmailFile_TxtExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headers.txt")

	body := "From: test@example.com\r\nSubject: Hi"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	content, inputType, err := readPhishingEmailFile(path)
	if err != nil {
		t.Fatalf("readPhishingEmailFile returned error: %v", err)
	}
	if content != body {
		t.Errorf("content = %q, want %q", content, body)
	}
	if inputType != "raw" {
		t.Errorf("inputType = %q, want %q", inputType, "raw")
	}
}

func TestReadPhishingEmailFile_UppercaseEml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "message.EML")

	body := "From: test@example.com\r\nSubject: Hi\r\n\r\nBody"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, inputType, err := readPhishingEmailFile(path)
	if err != nil {
		t.Fatalf("readPhishingEmailFile returned error: %v", err)
	}
	if inputType != "eml" {
		t.Errorf("inputType = %q, want %q for .EML extension", inputType, "eml")
	}
}

func TestReadPhishingEmailFile_ExactSizeLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exact.eml")

	// Write a file exactly at the limit -- should succeed.
	data := make([]byte, maxEmailFileSize)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, _, err := readPhishingEmailFile(path)
	if err != nil {
		t.Errorf("file at exact size limit should not error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// phishingLevelColor
// ---------------------------------------------------------------------------

func TestPhishingLevelColor(t *testing.T) {
	tests := []struct {
		level string
		want  string // description of expected color for verification
	}{
		{"highly_malicious", "red_bold"},
		{"malicious", "red"},
		{"suspicious", "yellow"},
		{"safe", "green"},
		{"clean", "green"},
		{"unknown", "dim"},
		{"", "dim"},
		{"other", "dim"},
	}

	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			c := phishingLevelColor(tc.level)
			if c == nil {
				t.Fatal("phishingLevelColor returned nil")
			}
		})
	}

	// Verify specific color mappings more precisely.
	t.Run("highly_malicious_is_bold", func(t *testing.T) {
		c := phishingLevelColor("highly_malicious")
		expected := color.New(color.FgRed, color.Bold)
		// Compare by rendering the same string.
		got := c.Sprint("X")
		want := expected.Sprint("X")
		if got != want {
			t.Errorf("highly_malicious color mismatch")
		}
	})
}

// ---------------------------------------------------------------------------
// phishingVerdictExit
// ---------------------------------------------------------------------------

func TestPhishingVerdictExit(t *testing.T) {
	tests := []struct {
		level    string
		wantNil  bool
		wantCode int
	}{
		{"safe", true, 0},
		{"clean", true, 0},
		{"unknown", true, 0},
		{"malicious", false, 2},
		{"suspicious", false, 3},
		{"highly_malicious", false, 2},
		{"", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			err := phishingVerdictExit(tt.level)

			if tt.wantNil {
				if err != nil {
					t.Fatalf("phishingVerdictExit(%q) = %v, want nil", tt.level, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("phishingVerdictExit(%q) = nil, want exit code %d", tt.level, tt.wantCode)
			}

			var exitErr *SilentExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("phishingVerdictExit(%q) returned %T, want *SilentExitError", tt.level, err)
			}
			if exitErr.Code != tt.wantCode {
				t.Errorf("phishingVerdictExit(%q) exit code = %d, want %d", tt.level, exitErr.Code, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handlePhishingAPIError
// ---------------------------------------------------------------------------

func TestHandlePhishingAPIError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		err := handlePhishingAPIError(nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("insufficient_credits", func(t *testing.T) {
		creditsErr := &client.InsufficientCreditsError{Message: "no credits left"}
		err := handlePhishingAPIError(creditsErr)
		if err == nil {
			t.Fatal("expected error")
		}
		var exitErr *SilentExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *SilentExitError, got %T", err)
		}
		if exitErr.Code != 4 {
			t.Errorf("exit code = %d, want 4", exitErr.Code)
		}
	})

	t.Run("generic_error_passed_through", func(t *testing.T) {
		genericErr := errors.New("connection refused")
		err := handlePhishingAPIError(genericErr)
		if err != genericErr {
			t.Errorf("expected error to be passed through, got %v", err)
		}
	})
}
