package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		{"malicious", false, 2},
		{"suspicious", false, 3},
		{"highly_malicious", false, 2},
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
