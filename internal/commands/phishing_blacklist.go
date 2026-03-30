package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingBlacklistCmd() *cobra.Command {
	var (
		flagIP    string
		flagBatch string
	)

	cmd := &cobra.Command{
		Use:   "blacklist",
		Short: "Check IPs against DNS blacklists",
		Long: `Check one or more IP addresses against DNS blacklists (DNSBLs).

Input can be supplied via --ip for a single address, --batch for a file with
one IP per line, or piped via stdin.`,
		Example: `  dfir-cli phishing blacklist --ip 1.2.3.4
  dfir-cli phishing blacklist --batch ips.txt
  echo "1.2.3.4" | dfir-cli phishing blacklist`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingBlacklist(flagIP, flagBatch)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagIP, "ip", "", "Single IP address to check")
	flags.StringVar(&flagBatch, "batch", "", "File with one IP per line (use - for stdin)")

	return cmd
}

func runPhishingBlacklist(ipFlag, batchFlag string) error {
	ips, err := resolveBlacklistIPs(ipFlag, batchFlag)
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return fmt.Errorf("no IPs provided. Use --ip, --batch, or pipe via stdin")
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

	spin := output.NewSpinner("Checking IPs against blacklists...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingBlacklist(ctx, ips)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	switch {
	case IsQuiet():
		renderBlacklistQuiet(result)
	case format == output.FormatJSON:
		return output.PrintJSON(result)
	case format == output.FormatJSONL:
		return output.PrintJSONL(result)
	default:
		renderBlacklistTable(result, resp)
	}

	return nil
}

// resolveBlacklistIPs gathers IP addresses from --ip, --batch, or stdin.
func resolveBlacklistIPs(ipFlag, batchFlag string) ([]string, error) {
	// Single IP flag takes precedence.
	if ipFlag != "" {
		return []string{ipFlag}, nil
	}

	// Batch file.
	if batchFlag != "" {
		lines, err := readLines(batchFlag)
		if err != nil {
			return nil, fmt.Errorf("reading batch file: %w", err)
		}
		return lines, nil
	}

	// Try stdin.
	data, err := readStdin()
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if data != nil {
		var ips []string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				ips = append(ips, line)
			}
		}
		return ips, nil
	}

	return nil, nil
}

// renderBlacklistQuiet prints each IP with the count of blacklists it is listed on.
func renderBlacklistQuiet(result map[string]interface{}) {
	results := blacklistExtractResults(result)
	for ip, data := range results {
		listed := countBlacklistListings(data)
		fmt.Printf("%s %d\n", ip, listed)
	}
}

// renderBlacklistTable renders blacklist results in a formatted table.
func renderBlacklistTable(result map[string]interface{}, resp *client.Response) {
	fmt.Println()
	output.PrintHeader("Blacklist Check Results")

	results := blacklistExtractResults(result)

	if len(results) == 0 {
		fmt.Println("  No results returned.")
		fmt.Println()
		return
	}

	for ip, data := range results {
		listed := countBlacklistListings(data)
		total := countBlacklistTotal(data)

		output.Bold.Printf("  IP: %s", ip)
		if total > 0 {
			fmt.Printf("  (%d/%d blacklists)\n", listed, total)
		} else {
			fmt.Println()
		}

		t := output.NewTable()
		t.Style().Options.DrawBorder = false
		t.Style().Options.SeparateHeader = true
		t.Style().Options.SeparateRows = false
		t.Style().Options.SeparateColumns = true

		t.AppendHeader(table.Row{"Blacklist", "Status"})

		if ipData, ok := data.(map[string]interface{}); ok {
			// Check for a "blacklists" sub-key.
			blacklists := ipData
			if nested, ok := ipData["blacklists"].(map[string]interface{}); ok {
				blacklists = nested
			}
			if nested, ok := ipData["results"].(map[string]interface{}); ok {
				blacklists = nested
			}

			for blName, blStatus := range blacklists {
				status := blacklistStatusString(blStatus)
				badge := blacklistBadge(status)
				t.AppendRow(table.Row{blName, badge})
			}

			// Handle array-style blacklist results.
			if arr, ok := ipData["blacklists"].([]interface{}); ok {
				for _, entry := range arr {
					if m, ok := entry.(map[string]interface{}); ok {
						name := interfaceToString(m["name"])
						if name == "" {
							name = interfaceToString(m["blacklist"])
						}
						status := blacklistStatusString(m["listed"])
						if s, ok := m["status"].(string); ok {
							status = s
						}
						badge := blacklistBadge(status)
						t.AppendRow(table.Row{name, badge})
					}
				}
			}
		}

		t.Render()
		fmt.Println()
	}

	// Credits footer.
	if resp != nil {
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}
	fmt.Println()
}

// blacklistExtractResults gets the per-IP results map from the API response.
func blacklistExtractResults(result map[string]interface{}) map[string]interface{} {
	// Try common nested keys.
	if nested, ok := result["results"].(map[string]interface{}); ok {
		return nested
	}
	if nested, ok := result["ips"].(map[string]interface{}); ok {
		return nested
	}
	return result
}

// countBlacklistListings counts how many blacklists an IP is listed on.
func countBlacklistListings(data interface{}) int {
	ipData, ok := data.(map[string]interface{})
	if !ok {
		return 0
	}

	// Check for a "listed_count" field.
	if count, ok := ipData["listed_count"].(float64); ok {
		return int(count)
	}

	// Count from individual blacklist entries.
	count := 0
	blacklists := ipData
	if nested, ok := ipData["blacklists"].(map[string]interface{}); ok {
		blacklists = nested
	}
	for _, v := range blacklists {
		if isListed(v) {
			count++
		}
	}
	return count
}

// countBlacklistTotal counts total number of blacklists checked.
func countBlacklistTotal(data interface{}) int {
	ipData, ok := data.(map[string]interface{})
	if !ok {
		return 0
	}

	if total, ok := ipData["total_count"].(float64); ok {
		return int(total)
	}
	if total, ok := ipData["total"].(float64); ok {
		return int(total)
	}

	blacklists := ipData
	if nested, ok := ipData["blacklists"].(map[string]interface{}); ok {
		blacklists = nested
	}
	return len(blacklists)
}

// isListed checks whether a blacklist entry indicates the IP is listed.
func isListed(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.EqualFold(val, "listed") || strings.EqualFold(val, "true")
	case map[string]interface{}:
		if listed, ok := val["listed"].(bool); ok {
			return listed
		}
		if status, ok := val["status"].(string); ok {
			return strings.EqualFold(status, "listed")
		}
	}
	return false
}

// blacklistStatusString converts various status representations to a string.
func blacklistStatusString(v interface{}) string {
	switch val := v.(type) {
	case bool:
		if val {
			return "listed"
		}
		return "not listed"
	case string:
		return val
	default:
		return interfaceToString(v)
	}
}

// blacklistBadge returns a colored status badge for blacklist results.
func blacklistBadge(status string) string {
	upper := strings.ToUpper(status)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "listed", "true":
		return output.Red.Sprint(upper)
	case "not listed", "not_listed", "clean", "false":
		return output.Green.Sprint(upper)
	default:
		return output.Dim.Sprint(upper)
	}
}
