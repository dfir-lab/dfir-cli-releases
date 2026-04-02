package commands

import (
	"fmt"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newPhishingGeoIPCmd() *cobra.Command {
	var (
		flagIP    string
		flagBatch string
	)

	cmd := &cobra.Command{
		Use:   "geoip",
		Short: "Look up geographic location of IPs",
		Long: `Look up the geographic location of one or more IP addresses using GeoIP data.

Input methods:
  --ip 1.2.3.4                                   Single IP lookup
  --batch ips.txt                                 File with one IP per line
  echo "1.2.3.4" | dfir-cli phishing geoip       Pipe via stdin`,
		Example: `  dfir-cli phishing geoip --ip 8.8.8.8
  dfir-cli phishing geoip --batch ips.txt
  echo "1.2.3.4" | dfir-cli phishing geoip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingGeoIP(flagIP, flagBatch)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagIP, "ip", "", "IP address to look up")
	flags.StringVar(&flagBatch, "batch", "", "File with one IP per line (use - for stdin)")

	return cmd
}

func runPhishingGeoIP(ipFlag, batchFlag string) error {
	ips, err := resolveGeoIPInputs(ipFlag, batchFlag)
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return fmt.Errorf("no IPs provided.\n\nUsage:\n" +
			"  dfir-cli phishing geoip --ip 1.2.3.4\n" +
			"  dfir-cli phishing geoip --batch ips.txt\n" +
			"  echo \"1.2.3.4\" | dfir-cli phishing geoip")
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

	spin := output.NewSpinner("Looking up GeoIP data...")
	output.StartSpinner(spin)

	result, resp, err := apiClient.PhishingGeoIP(ctx, ips)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	if resp != nil {
		_ = SaveAPIState(&resp.Meta, "phishing", "geoip")
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for GeoIP lookups")
	default:
		renderGeoIPOutput(result, resp)
	}

	return nil
}

// resolveGeoIPInputs collects IPs from --ip, --batch, or stdin.
func resolveGeoIPInputs(ipFlag, batchFlag string) ([]string, error) {
	if ipFlag != "" {
		return []string{ipFlag}, nil
	}

	if batchFlag != "" {
		return readLines(batchFlag)
	}

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

func renderGeoIPOutput(result map[string]interface{}, resp *client.Response) {
	if IsQuiet() {
		renderGeoIPQuiet(result)
		return
	}

	fmt.Println()
	output.Bold.Println("GeoIP Lookup Results")
	fmt.Println()

	t := output.NewTable()
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true

	t.AppendHeader(table.Row{"IP", "Country", "City", "ISP", "ASN"})

	for key, val := range result {
		if key == "meta" {
			continue
		}
		entry, ok := val.(map[string]interface{})
		if !ok {
			t.AppendRow(table.Row{key, "-", "-", "-", "-"})
			continue
		}
		t.AppendRow(table.Row{
			key,
			strOrDash(entry, "country"),
			strOrDash(entry, "city"),
			strOrDash(entry, "isp"),
			strOrDash(entry, "asn"),
		})
	}

	t.Render()

	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

func renderGeoIPQuiet(result map[string]interface{}) {
	for key, val := range result {
		if key == "meta" {
			continue
		}
		entry, ok := val.(map[string]interface{})
		if !ok {
			fmt.Printf("%s -\n", key)
			continue
		}
		fmt.Printf("%s %s\n", key, strOrDash(entry, "country_code"))
	}
}

// strOrDash safely extracts a string from a map, returning "-" if absent or empty.
func strOrDash(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return "-"
	}
	s, ok := v.(string)
	if ok {
		if s == "" {
			return "-"
		}
		return s
	}
	return fmt.Sprintf("%v", v)
}
