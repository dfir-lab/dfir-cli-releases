package commands

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRootRegistersUsageCommand(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("root command is nil")
	}
	cmd, _, err := rootCmd.Find([]string{"usage"})
	if err != nil {
		t.Fatalf("unexpected error finding usage command: %v", err)
	}
	if cmd == nil || cmd.Name() != "usage" {
		t.Fatalf("usage command not registered")
	}
}

func TestRootUsageTemplateReflectsShippedBehavior(t *testing.T) {
	if !strings.Contains(usageTemplate, "usage         Display locally recorded API usage statistics") {
		t.Fatalf("usage template does not list usage command")
	}
	if strings.Contains(usageTemplate, "phishing analyze --url example.com") {
		t.Fatalf("usage template still suggests unsupported phishing analyze --url flow")
	}
}

func TestProfileCompletionCandidates_DefaultFallback(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", filepath.Join(t.TempDir(), "no-config"))
	got := profileCompletionCandidates()

	if len(got) != 1 || got[0] != "default" {
		t.Fatalf("expected only default fallback profile, got: %v", got)
	}
}

func TestProfileCompletionCandidates_FromConfigFile(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", cfgDir)

	configYAML := `active_profile: default
profiles:
  default:
    api_url: https://dfir-lab.ch/api/v1
  staging:
    api_url: https://staging.dfir-lab.ch/api/v1
  prod:
    api_url: https://api.dfir-lab.ch/api/v1
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(configYAML), 0600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	got := profileCompletionCandidates()
	want := []string{"default", "prod", "staging"}
	if len(got) != len(want) {
		t.Fatalf("unexpected profile count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected profile at index %d: got %q want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}
