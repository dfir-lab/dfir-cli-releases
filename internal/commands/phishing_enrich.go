package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPhishingEnrichCmd() *cobra.Command {
	var flagURL string

	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "Enrich a URL with threat intelligence",
		Long: `Enrich a URL with threat intelligence data including risk scoring,
categorization, and threat intel provider results.

Input methods:
  --url https://suspicious.example.com                       Enrich a single URL
  echo "https://suspicious.example.com" | dfir-cli phishing enrich   Pipe via stdin

Cost: 2 credits per request.`,
		Example: `  dfir-cli phishing enrich --url https://suspicious.example.com
  echo "https://evil.com/phish" | dfir-cli phishing enrich`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingEnrich(flagURL)
		},
	}

	cmd.Flags().StringVar(&flagURL, "url", "", "URL to enrich with threat intelligence")

	return cmd
}

func runPhishingEnrich(urlFlag string) error {
	// Resolve URL from flag or stdin (reuses the same helper as url-expand).
	url, err := resolveEnrichURLInput(urlFlag)
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

	spin := output.NewSpinner("Enriching URL with threat intelligence...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingEnrich(ctx, url)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveAPIState(&resp.Meta, "phishing", "enrich")
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for phishing enrichment")
	default:
		if IsQuiet() {
			enrichURL := mapString(result, "url")
			if enrichURL == "" {
				enrichURL = url
			}
			riskLevel := mapString(result, "risk_level")
			if riskLevel == "" {
				riskLevel = mapString(result, "verdict")
			}
			fmt.Printf("%s %s\n", enrichURL, strings.ToUpper(riskLevel))
			return nil
		}
		renderPhishingEnrichTable(result, resp)
	}

	return nil
}

// renderPhishingEnrichTable prints the phishing enrichment results in table format.
func renderPhishingEnrichTable(result map[string]interface{}, resp *client.Response) {
	fmt.Println()
	output.PrintHeader("Phishing URL Enrichment")

	// URL.
	url := mapString(result, "url")
	if url != "" {
		output.PrintKeyValue("URL", url)
	}

	// Risk score.
	if score, ok := result["risk_score"]; ok {
		scoreInt := 0
		switch v := score.(type) {
		case float64:
			scoreInt = int(v)
		case int:
			scoreInt = v
		}
		fmt.Printf("  %-14s %s\n", "Risk Score:", output.ScoreBar(scoreInt, 100))
	}

	// Risk level / verdict.
	riskLevel := mapString(result, "risk_level")
	if riskLevel == "" {
		riskLevel = mapString(result, "verdict")
	}
	if riskLevel != "" {
		output.PrintKeyValueColored("Risk Level", strings.ToUpper(riskLevel), phishingLevelColor(riskLevel))
	}

	// Categories.
	if cats, ok := result["categories"]; ok {
		if catSlice, ok := cats.([]interface{}); ok && len(catSlice) > 0 {
			parts := make([]string, 0, len(catSlice))
			for _, c := range catSlice {
				parts = append(parts, fmt.Sprintf("%v", c))
			}
			output.PrintKeyValue("Categories", strings.Join(parts, ", "))
		}
	}

	// Threat intel data — show any additional fields as key-value pairs.
	printEnrichmentExtraFields(result)

	// Credits footer.
	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

// printEnrichmentExtraFields prints additional threat intel fields from the
// enrichment response that are not already rendered by the main table.
func printEnrichmentExtraFields(result map[string]interface{}) {
	// Known keys already handled above.
	handled := map[string]bool{
		"url":        true,
		"risk_score": true,
		"risk_level": true,
		"verdict":    true,
		"categories": true,
	}

	for key, val := range result {
		if handled[key] {
			continue
		}

		// Skip complex nested objects in table view — they are available in JSON.
		switch v := val.(type) {
		case map[string]interface{}:
			// Print a summary line pointing to JSON for full details.
			if len(v) > 0 {
				label := strings.ReplaceAll(key, "_", " ")
				label = titleCase(label)
				output.PrintKeyValue(label, fmt.Sprintf("(%d fields — use --output json for details)", len(v)))
			}
		case []interface{}:
			if len(v) > 0 {
				label := strings.ReplaceAll(key, "_", " ")
				label = titleCase(label)
				// Print list items.
				fmt.Println()
				output.Bold.Printf("  %s:\n", label)
				for i, item := range v {
					fmt.Printf("    %d. %v\n", i+1, item)
				}
			}
		default:
			if val != nil {
				label := strings.ReplaceAll(key, "_", " ")
				label = titleCase(label)
				output.PrintKeyValue(label, fmt.Sprintf("%v", val))
			}
		}
	}
}

// resolveEnrichURLInput resolves the URL from --url flag or stdin for the
// enrich subcommand.
func resolveEnrichURLInput(urlFlag string) (string, error) {
	const usage = "Usage:\n" +
		"  dfir-cli phishing enrich --url https://suspicious.example.com\n" +
		"  echo \"https://suspicious.example.com\" | dfir-cli phishing enrich"

	if urlFlag != "" {
		return urlFlag, nil
	}

	// Try stdin.
	data, err := readStdin()
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	if data != nil {
		content := strings.TrimSpace(string(data))
		if content == "" {
			return "", fmt.Errorf("no URL provided.\n\n%s", usage)
		}
		return content, nil
	}

	return "", fmt.Errorf("no URL provided.\n\n%s", usage)
}
