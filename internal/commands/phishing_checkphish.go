package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingCheckPhishCmd() *cobra.Command {
	var flagURL string

	cmd := &cobra.Command{
		Use:   "checkphish",
		Short: "Check a URL with CheckPhish",
		Long: `Submit a URL to the CheckPhish service for phishing analysis.

Costs 2 credits per lookup.

Input methods:
  --url https://example.com
  echo "https://example.com" | dfir-cli phishing checkphish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingCheckPhish(flagURL)
		},
	}

	cmd.Flags().StringVar(&flagURL, "url", "", "URL to check (required)")

	return cmd
}

func runPhishingCheckPhish(urlFlag string) error {
	// Resolve URL from flag or stdin.
	url, err := resolveURLInput(urlFlag)
	if err != nil {
		return err
	}

	// Build API client.
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	// Context with Ctrl+C handling.
	ctx, cancel := signalContext()
	defer cancel()

	// Determine output format.
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	spin := output.NewSpinner("Checking URL with CheckPhish...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingCheckPhish(ctx, url)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for checkphish")
	default:
		if IsQuiet() {
			disposition := mapStr(result, "disposition")
			fmt.Printf("%s %s\n", url, disposition)
			return nil
		}
		renderCheckPhishTable(result, url, resp)
	}

	return nil
}

func renderCheckPhishTable(result map[string]interface{}, url string, resp *client.Response) {
	fmt.Println()
	output.PrintHeader("CheckPhish Result")

	t := output.NewTable()
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true

	t.AppendHeader(table.Row{"Field", "Value"})

	t.AppendRow(table.Row{"URL", url})
	t.AppendRow(table.Row{"Disposition", output.VerdictBadge(mapStr(result, "disposition"))})
	if brand := mapStr(result, "brand"); brand != "" {
		t.AppendRow(table.Row{"Brand Targeted", brand})
	}
	if insights := mapStr(result, "insights"); insights != "" {
		t.AppendRow(table.Row{"Insights", insights})
	}

	t.Render()

	// Credits footer.
	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

// ---------------------------------------------------------------------------
// Shared helpers for URL-based phishing subcommands
// ---------------------------------------------------------------------------

// resolveURLInput resolves a URL from the --url flag or stdin.
// It validates that the URL uses an http:// or https:// scheme.
func resolveURLInput(urlFlag string) (string, error) {
	var resolved string

	if urlFlag != "" {
		resolved = urlFlag
	} else {
		// Try stdin.
		data, err := readStdin()
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		if data != nil {
			resolved = strings.TrimSpace(string(data))
		}
	}

	if resolved == "" {
		return "", fmt.Errorf("no URL provided. Use --url or pipe via stdin")
	}

	// Validate URL scheme.
	lower := strings.ToLower(resolved)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", fmt.Errorf("URL must use http:// or https:// scheme, got: %s", resolved)
	}

	return resolved, nil
}

// mapStr safely extracts a string value from a map[string]interface{}.
// Returns an empty string if the key is missing or the value is not a string.
func mapStr(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// mapSliceStr extracts a []string from a map[string]interface{} key that holds
// a []interface{} of strings. Returns nil if the key is missing or not the
// expected type.
func mapSliceStr(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range slice {
		if s, ok := item.(string); ok {
			out = append(out, s)
		} else {
			out = append(out, fmt.Sprintf("%v", item))
		}
	}
	return out
}
