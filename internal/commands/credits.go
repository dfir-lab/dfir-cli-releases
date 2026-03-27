package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ForeGuards/dfir-cli/internal/client"
	"github.com/ForeGuards/dfir-cli/internal/config"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Credit state — persisted between API calls
// ---------------------------------------------------------------------------

const (
	stateFileName        = "state.json"
	stateFilePerm        = os.FileMode(0600)
)

// creditState holds the latest credit information captured from an API response.
type creditState struct {
	CreditsRemaining int    `json:"credits_remaining"`
	LastCreditsUsed  int    `json:"last_credits_used"`
	LastRequestAt    string `json:"last_request_at"`
	LastRequestID    string `json:"last_request_id"`
}

// statePath returns the full path to the credit state file.
func statePath() string {
	return filepath.Join(config.Dir(), stateFileName)
}

// SaveCreditState persists credit information from an API response to disk.
// It should be called by every command that makes an API call so the credits
// command can display up-to-date information without consuming credits itself.
func SaveCreditState(meta *client.ResponseMeta) error {
	if meta == nil {
		return nil
	}

	state := creditState{
		CreditsRemaining: meta.CreditsRemaining,
		LastCreditsUsed:  meta.CreditsUsed,
		LastRequestID:    meta.RequestID,
	}

	// Use the current time as a fallback; callers can override if the API
	// returns an authoritative timestamp.
	state.LastRequestAt = timeNowUTC()

	return writeCreditState(&state)
}

// LoadCreditState reads the persisted credit state from disk.
// Returns nil and an error if the file does not exist or cannot be read.
func LoadCreditState() (*creditState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}

	var state creditState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse credit state: %w", err)
	}

	return &state, nil
}

// writeCreditState serialises state to JSON and writes it atomically.
func writeCreditState(state *creditState) error {
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credit state: %w", err)
	}
	data = append(data, '\n')

	path := statePath()
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, stateFilePerm); err != nil {
		return fmt.Errorf("write credit state: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename credit state: %w", err)
	}

	return nil
}

// timeNowUTC returns the current UTC time as an RFC 3339 string.
// Extracted to a package-level variable so tests can override it if needed.
var timeNowUTC = func() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// Command
// ---------------------------------------------------------------------------

// NewCreditsCmd creates and returns the "credits" subcommand.
func NewCreditsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credits",
		Short: "View API credit balance",
		Long: `Display the credit balance from the most recent API call.

Credit information is updated automatically after every API operation. This
command reads the cached balance — it does not make an API call and therefore
does not consume credits.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredits(cmd)
		},
	}

	return cmd
}

func runCredits(cmd *cobra.Command) error {
	outputFormat := GetOutputFormat()
	quiet := IsQuiet()

	state, err := LoadCreditState()
	if err != nil {
		// State file missing or unreadable — guide the user.
		if quiet {
			// In quiet mode, print nothing and exit with an error code.
			return fmt.Errorf("no credit information available")
		}

		if outputFormat == "json" {
			fmt.Fprintln(cmd.OutOrStdout(), `{"error":"no credit information available"}`)
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), "No credit information available yet.")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Run any API command to see your credit balance:")
		fmt.Fprintln(cmd.OutOrStdout(), "  dfir-cli enrichment lookup --ip 8.8.8.8")
		fmt.Fprintln(cmd.OutOrStdout(), "  dfir-cli phishing analyze --file email.eml")
		fmt.Fprintln(cmd.OutOrStdout(), "  dfir-cli exposure scan --domain example.com")
		return nil
	}

	// Quiet mode: just print the number.
	if quiet {
		fmt.Fprintln(cmd.OutOrStdout(), state.CreditsRemaining)
		return nil
	}

	// JSON mode.
	if outputFormat == "json" {
		out := struct {
			CreditsRemaining int    `json:"credits_remaining"`
			LastCreditsUsed  int    `json:"last_credits_used"`
			LastRequestAt    string `json:"last_request_at"`
		}{
			CreditsRemaining: state.CreditsRemaining,
			LastCreditsUsed:  state.LastCreditsUsed,
			LastRequestAt:    state.LastRequestAt,
		}

		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal credit info: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Table / default mode.
	fmt.Fprintln(cmd.OutOrStdout(), "Credit Balance (as of last API call)")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "  Credits Remaining:  %d\n", state.CreditsRemaining)
	fmt.Fprintf(cmd.OutOrStdout(), "  Last Used:          %d\n", state.LastCreditsUsed)
	fmt.Fprintf(cmd.OutOrStdout(), "  Last Request:       %s\n", state.LastRequestAt)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "  Note: Credit balance is updated after each API operation.")
	fmt.Fprintln(cmd.OutOrStdout(), "  Run any command to refresh, e.g.: dfir-cli enrichment lookup --ip 8.8.8.8")

	return nil
}
