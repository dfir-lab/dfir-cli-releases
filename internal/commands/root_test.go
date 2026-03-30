package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot creates a fresh root command (not the global singleton) so tests
// are isolated from each other.
func newTestRoot() *cobra.Command {
	cmd := newRootCmd()
	pflags := cmd.PersistentFlags()
	pflags.String("api-key", "", "Override API key")
	pflags.String("api-url", "", "Override API base URL")
	pflags.StringP("output", "o", "table", "Output format: table, json, jsonl, csv")
	pflags.Bool("no-color", false, "Disable colored output")
	pflags.BoolP("verbose", "v", false, "Show debug information")
	pflags.BoolP("quiet", "q", false, "Minimal output")
	pflags.BoolP("json", "j", false, "Shorthand for --output json")
	pflags.StringP("profile", "p", "default", "Named config profile")

	// Add a no-op subcommand so we can exercise the PersistentPreRunE.
	cmd.AddCommand(&cobra.Command{
		Use: "noop",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
	return cmd
}

func TestJsonFlagSetsOutputToJson(t *testing.T) {
	cmd := newTestRoot()
	cmd.SetArgs([]string{"noop", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, _ := cmd.PersistentFlags().GetString("output")
	if out != "json" {
		t.Errorf("expected output=json, got %q", out)
	}
}

func TestJsonShortFlagSetsOutputToJson(t *testing.T) {
	cmd := newTestRoot()
	cmd.SetArgs([]string{"noop", "-j"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, _ := cmd.PersistentFlags().GetString("output")
	if out != "json" {
		t.Errorf("expected output=json, got %q", out)
	}
}

func TestJsonAndQuietError(t *testing.T) {
	cmd := newTestRoot()
	cmd.SetArgs([]string{"noop", "--json", "--quiet"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --json and --quiet are both set")
	}
	want := "--json and --quiet cannot be used together"
	if err.Error() != want {
		t.Errorf("unexpected error message: got %q, want %q", err.Error(), want)
	}
}

func TestJsonOverridesOutputTable(t *testing.T) {
	cmd := newTestRoot()
	cmd.SetArgs([]string{"noop", "--output", "table", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, _ := cmd.PersistentFlags().GetString("output")
	if out != "json" {
		t.Errorf("expected output=json (--json should override --output table), got %q", out)
	}
}
