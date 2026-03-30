package commands

import (
	"testing"
)

// ---------------------------------------------------------------------------
// newPhishingURLScanCmd
// ---------------------------------------------------------------------------

func TestNewPhishingURLScanCmd_HasURLFlag(t *testing.T) {
	cmd := newPhishingURLScanCmd()

	if cmd.Use != "urlscan" {
		t.Errorf("Use = %q, want %q", cmd.Use, "urlscan")
	}

	f := cmd.Flags().Lookup("url")
	if f == nil {
		t.Fatal("expected --url flag to be defined")
	}
}

// ---------------------------------------------------------------------------
// NewPhishingCmd includes both new subcommands
// ---------------------------------------------------------------------------

func TestNewPhishingCmd_HasCheckPhishAndURLScan(t *testing.T) {
	cmd := NewPhishingCmd()

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["checkphish"] {
		t.Error("phishing command missing 'checkphish' subcommand")
	}
	if !subcommands["urlscan"] {
		t.Error("phishing command missing 'urlscan' subcommand")
	}
	if !subcommands["analyze"] {
		t.Error("phishing command missing 'analyze' subcommand")
	}
}
