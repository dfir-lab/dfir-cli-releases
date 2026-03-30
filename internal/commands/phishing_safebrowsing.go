package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingSafeBrowsingCmd() *cobra.Command {
	var (
		flagURL   string
		flagBatch string
	)

	cmd := &cobra.Command{
		Use:   "safe-browsing",
		Short: "Check URLs against Google Safe Browsing",
		Long: `Check one or more URLs against the Google Safe Browsing database to determine
if they are associated with known threats such as malware, phishing, or
unwanted software.

Input methods:
  --url https://example.com                              Single URL check
  --batch urls.txt                                       File with one URL per line
  echo "https://example.com" | dfir-cli phishing safe-browsing   Pipe via stdin`,
		Example: `  dfir-cli phishing safe-browsing --url https://example.com
  dfir-cli phishing safe-browsing --batch urls.txt
  echo "https://example.com" | dfir-cli phishing safe-browsing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingSafeBrowsing(flagURL, flagBatch)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagURL, "url", "", "URL to check")
	flags.StringVar(&flagBatch, "batch", "", "File with one URL per line (use - for stdin)")

	return cmd
}

func runPhishingSafeBrowsing(urlFlag, batchFlag string) error {
	urls, err := resolveSafeBrowsingInputs(urlFlag, batchFlag)
	if err != nil {
		return err
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided.\n\nUsage:\n" +
			"  dfir-cli phishing safe-browsing --url https://example.com\n" +
			"  dfir-cli phishing safe-browsing --batch urls.txt\n" +
			"  echo \"https://example.com\" | dfir-cli phishing safe-browsing")
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	spin := output.NewSpinner("Checking URLs against Safe Browsing...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingSafeBrowsing(ctx, urls)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for Safe Browsing checks")
	default:
		renderSafeBrowsingOutput(result, resp)
	}

	return nil
}

// resolveSafeBrowsingInputs collects URLs from --url, --batch, or stdin.
func resolveSafeBrowsingInputs(urlFlag, batchFlag string) ([]string, error) {
	if urlFlag != "" {
		return []string{urlFlag}, nil
	}

	if batchFlag != "" {
		return readLines(batchFlag)
	}

	data, err := readStdin()
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if data != nil {
		var urls []string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				urls = append(urls, line)
			}
		}
		return urls, nil
	}

	return nil, nil
}

func renderSafeBrowsingOutput(result map[string]interface{}, resp *client.Response) {
	if IsQuiet() {
		renderSafeBrowsingQuiet(result)
		return
	}

	fmt.Println()
	output.Bold.Println("Safe Browsing Results")
	fmt.Println()

	t := output.NewTable()
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true

	t.AppendHeader(table.Row{"URL", "Threat Type", "Status"})

	for key, val := range result {
		if key == "meta" {
			continue
		}
		entry, ok := val.(map[string]interface{})
		if !ok {
			t.AppendRow(table.Row{key, "-", "-"})
			continue
		}

		threatType := strOrDash(entry, "threat_type")
		status := safeBrowsingStatus(entry)
		statusBadge := safeBrowsingBadge(status)

		t.AppendRow(table.Row{key, threatType, statusBadge})
	}

	t.Render()

	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

func renderSafeBrowsingQuiet(result map[string]interface{}) {
	for key, val := range result {
		if key == "meta" {
			continue
		}
		entry, ok := val.(map[string]interface{})
		if !ok {
			fmt.Printf("%s unknown\n", key)
			continue
		}
		status := safeBrowsingStatus(entry)
		fmt.Printf("%s %s\n", key, status)
	}
}

// safeBrowsingStatus determines if a URL is safe or unsafe from the result entry.
func safeBrowsingStatus(entry map[string]interface{}) string {
	// Check for a "safe" boolean field.
	if safe, ok := entry["safe"]; ok {
		if b, ok := safe.(bool); ok {
			if b {
				return "safe"
			}
			return "unsafe"
		}
	}

	// Check for a "threat_type" field — if present and non-empty, it's unsafe.
	if tt, ok := entry["threat_type"]; ok {
		if s, ok := tt.(string); ok && s != "" && s != "none" && s != "NONE" {
			return "unsafe"
		}
	}

	// Check for a "status" field.
	if s, ok := entry["status"]; ok {
		if str, ok := s.(string); ok {
			return strings.ToLower(str)
		}
	}

	return "safe"
}

// safeBrowsingBadge returns a colored status badge.
func safeBrowsingBadge(status string) string {
	upper := strings.ToUpper(status)
	switch strings.ToLower(status) {
	case "safe":
		return output.Green.Sprint(upper)
	case "unsafe":
		return output.Red.Sprint(upper)
	default:
		return output.Dim.Sprint(upper)
	}
}
