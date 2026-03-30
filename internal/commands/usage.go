package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
		Short: "Display API usage statistics",
		Long: `Display API usage statistics including request counts, credit consumption,
and a breakdown by service.

The period flag controls which billing period to display:
  current   — the current month (default)
  previous  — the previous month
  YYYY-MM   — a specific month (e.g. 2026-03)

Optionally filter by service (phishing, exposure, enrichment) and limit
the number of top operations shown.`,
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
	flags.StringVar(&flagService, "service", "", "Filter by service: phishing, exposure, enrichment")
	_ = cmd.RegisterFlagCompletionFunc("service", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"phishing", "exposure", "enrichment"}, cobra.ShellCompDirectiveNoFileComp
	})
	flags.IntVar(&flagTop, "top", 10, "Number of top operations to display")

	return cmd
}

func runUsage(cmd *cobra.Command, period, service string, top int) error {
	// Validate --service if provided.
	if service != "" {
		switch strings.ToLower(service) {
		case "phishing", "exposure", "enrichment":
			// OK
		default:
			return fmt.Errorf("invalid service %q. Valid services: phishing, exposure, enrichment", service)
		}
	}

	// Create API client.
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	// Set up a cancellable context for signal handling.
	ctx, cancel := signalContext()
	defer cancel()

	// Determine output format.
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	spin := output.NewSpinner("Fetching usage statistics...")
	output.StartSpinner(spin)

	req := &client.UsageRequest{
		Period:  period,
		Service: service,
	}

	result, resp, apiErr := apiClient.Usage(ctx, req)
	output.StopSpinner(spin)

	if apiErr != nil {
		return apiErr
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
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
		return renderUsageJSON(result, resp)
	default:
		return renderUsageTable(cmd, result, resp)
	}
}

// ---------------------------------------------------------------------------
// Output: table
// ---------------------------------------------------------------------------

func renderUsageTable(cmd *cobra.Command, result *client.UsageResponse, meta *client.Response) error {
	w := cmd.OutOrStdout()

	// Header.
	fmt.Fprintf(w, "\nAPI Usage (%s)\n\n", result.Period)
	fmt.Fprintf(w, "  Total Requests:  %s\n", formatNumber(result.TotalRequests))
	fmt.Fprintf(w, "  Total Credits:   %s\n", formatNumber(result.TotalCredits))

	// By-service breakdown.
	if len(result.ByService) > 0 {
		fmt.Fprintf(w, "\n  By Service:\n")

		t := output.NewTable()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"Service", "Requests", "Credits"})

		// Sort services for deterministic output.
		services := make([]string, 0, len(result.ByService))
		for svc := range result.ByService {
			services = append(services, svc)
		}
		sort.Strings(services)

		for _, svc := range services {
			usage := result.ByService[svc]
			t.AppendRow(table.Row{
				titleCase(svc),
				formatNumber(usage.Requests),
				formatNumber(usage.Credits),
			})
		}

		t.Render()
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

	// Credits footer.
	if meta != nil {
		fmt.Fprintln(w)
		output.PrintCreditsFooter(meta.Meta.CreditsUsed, meta.Meta.CreditsRemaining)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Output: JSON
// ---------------------------------------------------------------------------

func renderUsageJSON(result *client.UsageResponse, meta *client.Response) error {
	payload := struct {
		*client.UsageResponse
		Meta *client.ResponseMeta `json:"meta,omitempty"`
	}{
		UsageResponse: result,
	}
	if meta != nil {
		payload.Meta = &meta.Meta
	}
	return output.PrintJSON(payload)
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
func renderUsageJSONRaw(result *client.UsageResponse, meta *client.Response) ([]byte, error) {
	payload := struct {
		*client.UsageResponse
		Meta *client.ResponseMeta `json:"meta,omitempty"`
	}{
		UsageResponse: result,
	}
	if meta != nil {
		payload.Meta = &meta.Meta
	}
	return json.MarshalIndent(payload, "", "  ")
}
