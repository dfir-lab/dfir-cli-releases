package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingDNSCmd() *cobra.Command {
	var flagDomain string

	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Perform DNS analysis on a domain",
		Long: `Perform DNS analysis on a domain, returning A, AAAA, MX, NS, TXT, CNAME,
and SOA records.

Input can be supplied via the --domain flag or piped via stdin.`,
		Example: `  dfir-cli phishing dns --domain example.com
  echo "example.com" | dfir-cli phishing dns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingDNS(flagDomain)
		},
	}

	cmd.Flags().StringVar(&flagDomain, "domain", "", "Domain to analyze")

	return cmd
}

func runPhishingDNS(domain string) error {
	// Resolve domain from flag or stdin.
	if domain == "" {
		data, err := readStdin()
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if data != nil {
			domain = strings.TrimSpace(string(data))
		}
	}

	if domain == "" {
		return fmt.Errorf("no domain provided. Use --domain or pipe via stdin")
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

	spin := output.NewSpinner("Performing DNS analysis...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingDNS(ctx, domain)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state.
	if resp != nil {
		_ = SaveAPIState(&resp.Meta, "phishing", "dns")
	}

	switch {
	case IsQuiet():
		count := countDNSRecords(result)
		fmt.Printf("%s %d\n", domain, count)
	case format == output.FormatJSON:
		return output.PrintJSON(result)
	case format == output.FormatJSONL:
		return output.PrintJSONL(result)
	default:
		renderDNSTable(domain, result, resp)
	}

	return nil
}

// dnsRecordTypes lists the record types we display in the table.
var dnsRecordTypes = []string{"a", "aaaa", "mx", "ns", "txt", "cname", "soa"}

// countDNSRecords counts the total number of DNS records across all types.
func countDNSRecords(result map[string]interface{}) int {
	count := 0
	records := dnsExtractRecords(result)
	for _, rtype := range dnsRecordTypes {
		if entries, ok := extractSlice(records, rtype); ok {
			count += len(entries)
		}
	}
	return count
}

// dnsExtractRecords returns the record map, checking top-level and nested keys.
func dnsExtractRecords(result map[string]interface{}) map[string]interface{} {
	if nested, ok := result["records"].(map[string]interface{}); ok {
		return nested
	}
	if nested, ok := result["dns_records"].(map[string]interface{}); ok {
		return nested
	}
	return result
}

// renderDNSTable renders DNS records in a formatted table.
func renderDNSTable(domain string, result map[string]interface{}, resp *client.Response) {
	fmt.Println()
	output.PrintHeader(fmt.Sprintf("DNS Analysis: %s", domain))

	records := dnsExtractRecords(result)

	hasRecords := false
	for _, rtype := range dnsRecordTypes {
		entries, ok := extractSlice(records, rtype)
		if !ok || len(entries) == 0 {
			continue
		}
		hasRecords = true

		fmt.Printf("  %s Records:\n", strings.ToUpper(rtype))

		t := output.NewTable()
		t.Style().Options.DrawBorder = false
		t.Style().Options.SeparateHeader = true
		t.Style().Options.SeparateRows = false
		t.Style().Options.SeparateColumns = true

		switch rtype {
		case "mx":
			t.AppendHeader(table.Row{"Priority", "Value"})
			for _, entry := range entries {
				if m, ok := entry.(map[string]interface{}); ok {
					priority := interfaceToString(m["priority"])
					value := interfaceToString(m["value"])
					if value == "" {
						value = interfaceToString(m["exchange"])
					}
					t.AppendRow(table.Row{priority, value})
				} else {
					t.AppendRow(table.Row{"-", interfaceToString(entry)})
				}
			}
		case "soa":
			t.AppendHeader(table.Row{"Field", "Value"})
			for _, entry := range entries {
				if m, ok := entry.(map[string]interface{}); ok {
					for k, v := range m {
						t.AppendRow(table.Row{k, interfaceToString(v)})
					}
				} else {
					t.AppendRow(table.Row{"value", interfaceToString(entry)})
				}
			}
		default:
			t.AppendHeader(table.Row{"Value"})
			for _, entry := range entries {
				if m, ok := entry.(map[string]interface{}); ok {
					value := interfaceToString(m["value"])
					if value == "" {
						value = interfaceToString(m["address"])
					}
					if value == "" {
						value = interfaceToString(entry)
					}
					t.AppendRow(table.Row{value})
				} else {
					t.AppendRow(table.Row{interfaceToString(entry)})
				}
			}
		}

		t.Render()
		fmt.Println()
	}

	if !hasRecords {
		fmt.Println("  No DNS records found.")
		fmt.Println()
	}

	// Credits footer.
	if resp != nil {
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}
	fmt.Println()
}

// extractSlice attempts to get a []interface{} from a map by key.
func extractSlice(m map[string]interface{}, key string) ([]interface{}, bool) {
	v, ok := m[key]
	if !ok {
		// Try uppercase variant.
		v, ok = m[strings.ToUpper(key)]
		if !ok {
			return nil, false
		}
	}
	slice, ok := v.([]interface{})
	return slice, ok
}

// interfaceToString converts an interface{} value to a string representation.
func interfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
