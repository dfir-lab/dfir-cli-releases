package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingURLScanCmd() *cobra.Command {
	var flagURL string

	cmd := &cobra.Command{
		Use:   "urlscan",
		Short: "Scan a URL with URLScan.io",
		Long: `Submit a URL to URLScan.io for analysis.

Costs 3 credits per scan.

Input methods:
  --url https://example.com
  echo "https://example.com" | dfir-cli phishing urlscan`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingURLScan(flagURL)
		},
	}

	cmd.Flags().StringVar(&flagURL, "url", "", "URL to scan (required)")

	return cmd
}

func runPhishingURLScan(urlFlag string) error {
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

	spin := output.NewSpinner("Scanning URL with URLScan.io...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingURLScan(ctx, url)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveAPIState(&resp.Meta, "phishing", "urlscan")
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for urlscan")
	default:
		if IsQuiet() {
			verdict := mapStr(result, "verdict")
			fmt.Printf("%s %s\n", url, verdict)
			return nil
		}
		renderURLScanTable(result, url, resp)
	}

	return nil
}

func renderURLScanTable(result map[string]interface{}, url string, resp *client.Response) {
	fmt.Println()
	output.PrintHeader("URLScan.io Result")

	t := output.NewTable()
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true

	t.AppendHeader(table.Row{"Field", "Value"})

	t.AppendRow(table.Row{"URL", url})
	t.AppendRow(table.Row{"Verdict", output.VerdictBadge(mapStr(result, "verdict"))})

	if title := mapStr(result, "page_title"); title != "" {
		t.AppendRow(table.Row{"Page Title", title})
	}
	if screenshot := mapStr(result, "screenshot_url"); screenshot != "" {
		t.AppendRow(table.Row{"Screenshot", screenshot})
	}
	if ips := mapSliceStr(result, "ips_contacted"); len(ips) > 0 {
		t.AppendRow(table.Row{"IPs Contacted", strings.Join(ips, ", ")})
	}
	if domains := mapSliceStr(result, "domains_contacted"); len(domains) > 0 {
		t.AppendRow(table.Row{"Domains Contacted", strings.Join(domains, ", ")})
	}
	if country := mapStr(result, "country"); country != "" {
		t.AppendRow(table.Row{"Country", country})
	}
	if server := mapStr(result, "server"); server != "" {
		t.AppendRow(table.Row{"Server", server})
	}

	t.Render()

	// Credits footer.
	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}
