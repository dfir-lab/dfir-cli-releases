package commands

import (
	"fmt"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPhishingURLExpandCmd() *cobra.Command {
	var flagURL string

	cmd := &cobra.Command{
		Use:   "url-expand",
		Short: "Expand shortened URLs",
		Long: `Expand a shortened URL to reveal the final destination and redirect chain.

Input methods:
  --url https://bit.ly/abc123                                   Expand a single URL
  echo "https://bit.ly/abc123" | dfir-cli phishing url-expand   Pipe via stdin

Cost: 1 credit per request.`,
		Example: `  dfir-cli phishing url-expand --url https://bit.ly/abc123
  echo "https://t.co/xyz" | dfir-cli phishing url-expand`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingURLExpand(flagURL)
		},
	}

	cmd.Flags().StringVar(&flagURL, "url", "", "Shortened URL to expand")

	return cmd
}

func runPhishingURLExpand(urlFlag string) error {
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

	spin := output.NewSpinner("Expanding URL...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingURLExpand(ctx, url)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveAPIState(&resp.Meta, "phishing", "url-expand")
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for URL expansion")
	default:
		if IsQuiet() {
			expandedURL := mapString(result, "expanded_url")
			if expandedURL == "" {
				expandedURL = mapString(result, "final_url")
			}
			fmt.Println(expandedURL)
			return nil
		}
		renderURLExpandTable(result, resp)
	}

	return nil
}

// renderURLExpandTable prints the URL expansion results in table format.
func renderURLExpandTable(result map[string]interface{}, resp *client.Response) {
	fmt.Println()
	output.PrintHeader("URL Expansion")

	// Original URL.
	originalURL := mapString(result, "original_url")
	if originalURL == "" {
		originalURL = mapString(result, "url")
	}
	if originalURL != "" {
		output.PrintKeyValue("Original URL", originalURL)
	}

	// Expanded / final URL.
	expandedURL := mapString(result, "expanded_url")
	if expandedURL == "" {
		expandedURL = mapString(result, "final_url")
	}
	if expandedURL != "" {
		output.PrintKeyValue("Expanded URL", expandedURL)
	}

	// Final status code.
	if statusCode, ok := result["status_code"]; ok {
		output.PrintKeyValue("Status Code", fmt.Sprintf("%v", statusCode))
	} else if statusCode, ok := result["final_status_code"]; ok {
		output.PrintKeyValue("Status Code", fmt.Sprintf("%v", statusCode))
	}

	// Redirect chain.
	if chain, ok := result["redirect_chain"]; ok {
		if chainSlice, ok := chain.([]interface{}); ok && len(chainSlice) > 0 {
			fmt.Println()
			output.Bold.Println("  Redirect Chain:")
			for i, hop := range chainSlice {
				fmt.Printf("    %d. %v\n", i+1, hop)
			}
		}
	}

	// Credits footer.
	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

// mapString safely extracts a string value from a map[string]interface{}.
func mapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}
