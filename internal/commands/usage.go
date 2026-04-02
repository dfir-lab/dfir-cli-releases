package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// NewUsageCmd creates and returns the "usage" subcommand.
func NewUsageCmd() *cobra.Command {
	var (
		flagPeriod  string
		flagService string
		flagTop     int
	)

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Display locally recorded API usage statistics",
		Long: `Display locally recorded API usage statistics including request counts,
credit consumption, and a breakdown by service.

The period flag controls which billing period to display:
  current   — the current month (default)
  previous  — the previous month
  YYYY-MM   — a specific month (e.g. 2026-03)

Optionally filter by service (phishing, exposure, enrichment, ai) and limit
the number of top operations shown.

Usage is built from successful dfir-cli API calls recorded on this machine.
The command does not make a network request.`,
		Example: `  dfir-cli usage
  dfir-cli usage --period previous
  dfir-cli usage --period 2026-01 --service enrichment
  dfir-cli usage --top 5`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsage(cmd, flagPeriod, flagService, flagTop)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagPeriod, "period", "current", "Billing period: current, previous, or YYYY-MM")
	_ = cmd.RegisterFlagCompletionFunc("period", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"current", "previous"}, cobra.ShellCompDirectiveNoFileComp
	})
	flags.StringVar(&flagService, "service", "", "Filter by service: phishing, exposure, enrichment, ai")
	_ = cmd.RegisterFlagCompletionFunc("service", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"phishing", "exposure", "enrichment", "ai"}, cobra.ShellCompDirectiveNoFileComp
	})
	flags.IntVar(&flagTop, "top", 10, "Number of top operations to display")

	return cmd
}

func runUsage(cmd *cobra.Command, period, service string, top int) error {
	// Validate --service if provided.
	if service != "" {
		switch strings.ToLower(service) {
		case "phishing", "exposure", "enrichment", "ai":
			// OK
		default:
			return fmt.Errorf("invalid service %q. Valid services: phishing, exposure, enrichment, ai", service)
		}
	}

	// Determine output format.
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	result, err := buildUsageResponse(period, service)
	if err != nil {
		return err
	}

	// Truncate top operations if requested.
	if top > 0 && len(result.TopOperations) > top {
		result.TopOperations = result.TopOperations[:top]
	}

	// Render output.
	switch {
	case IsQuiet():
		return renderUsageQuiet(cmd, result)
	case format == output.FormatJSON:
		return renderUsageJSON(result)
	default:
		return renderUsageTable(cmd, result)
	}
}

// ---------------------------------------------------------------------------
// Output: table
// ---------------------------------------------------------------------------

func renderUsageTable(cmd *cobra.Command, result *client.UsageResponse) error {
	w := cmd.OutOrStdout()

	// Header.
	fmt.Fprintf(w, "\nLocal API Usage (%s)\n\n", result.Period)
	fmt.Fprintf(w, "  Total Requests:  %s\n", formatNumber(result.TotalRequests))
	fmt.Fprintf(w, "  Total Credits:   %s\n", formatNumber(result.TotalCredits))
	fmt.Fprintln(w, "  Source:          locally recorded dfir-cli activity")

	// By-service breakdown.
	if len(result.ByService) > 0 {
		fmt.Fprintf(w, "\n  By Service:\n")
		renderUsageByServiceTable(w, result.ByService)
	}

	// Top operations.
	if len(result.TopOperations) > 0 {
		fmt.Fprintf(w, "\n  Top Operations:\n")

		t := output.NewTable()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"Operation", "Service", "Requests", "Credits"})

		for _, op := range result.TopOperations {
			t.AppendRow(table.Row{
				op.Operation,
				titleCase(op.Service),
				formatNumber(op.Requests),
				formatNumber(op.Credits),
			})
		}

		t.Render()
	}
	if result.TotalRequests == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  No locally recorded API usage for this period yet.")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Output: JSON
// ---------------------------------------------------------------------------

func renderUsageJSON(result *client.UsageResponse) error {
	return output.PrintJSON(result)
}

// ---------------------------------------------------------------------------
// Output: quiet
// ---------------------------------------------------------------------------

func renderUsageQuiet(cmd *cobra.Command, result *client.UsageResponse) error {
	fmt.Fprintln(cmd.OutOrStdout(), result.TotalCredits)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// formatNumber formats an integer with comma separators for readability.
// E.g., 1247 -> "1,247".
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// renderUsageJSONRaw is a helper for tests that returns JSON bytes.
func renderUsageJSONRaw(result *client.UsageResponse) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func renderUsageByServiceTable(w io.Writer, byService map[string]client.ServiceUsage) {
	t := output.NewTable()
	t.SetOutputMirror(w)
	t.AppendHeader(table.Row{"Service", "Requests", "Credits"})

	services := make([]string, 0, len(byService))
	for svc := range byService {
		services = append(services, svc)
	}
	sort.Strings(services)

	for _, svc := range services {
		usage := byService[svc]
		t.AppendRow(table.Row{
			titleCase(svc),
			formatNumber(usage.Requests),
			formatNumber(usage.Credits),
		})
	}

	t.Render()
}

func resolveUsagePeriod(input string) (string, string, error) {
	now := usageNow().UTC()
	switch input {
	case "", "current":
		return now.Format("2006-01"), now.Format("January 2006"), nil
	case "previous":
		previous := now.AddDate(0, -1, 0)
		return previous.Format("2006-01"), previous.Format("January 2006"), nil
	default:
		parsed, err := time.Parse("2006-01", input)
		if err != nil {
			return "", "", fmt.Errorf("invalid period %q. Use current, previous, or YYYY-MM", input)
		}
		return parsed.Format("2006-01"), parsed.Format("January 2006"), nil
	}
}

var usageNow = func() time.Time {
	return time.Now().UTC()
}
