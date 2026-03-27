package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/ForeGuards/dfir-cli/internal/client"
	"github.com/ForeGuards/dfir-cli/internal/output"
	"github.com/ForeGuards/dfir-cli/internal/version"
	"github.com/spf13/cobra"
)

// NewExposureCmd returns the top-level "exposure" command with its subcommands.
func NewExposureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exposure",
		Short: "Scan domains and IPs for external exposure",
		Long: `Scan domains and IPs for external exposure using the DFIR Lab API.

The exposure module probes a target for SSL/TLS configuration, DNS records,
and other publicly visible attack surface indicators, then computes an
overall risk score.`,
	}

	cmd.AddCommand(newExposureScanCmd())

	return cmd
}

func newExposureScanCmd() *cobra.Command {
	var (
		domain     string
		targetType string
		batchFile  string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan domains for external exposure",
		Long: `Scan one or more domains for external exposure.

Input methods:
  --domain example.com                          Single domain
  echo "example.com" | dfir-cli exposure scan   Stdin
  --batch domains.txt                           Batch file (one domain per line, use - for stdin)

The scan may take up to 3 minutes per target depending on the providers
queried. A spinner is displayed while waiting.`,
		Example: `  dfir-cli exposure scan --domain example.com
  dfir-cli exposure scan --domain example.com --target-type domain
  dfir-cli exposure scan --batch domains.txt
  echo "example.com" | dfir-cli exposure scan`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExposureScan(cmd, domain, targetType, batchFile)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Target domain to scan")
	cmd.Flags().StringVar(&targetType, "target-type", "auto", "Target type hint: domain, ip, auto")
	cmd.Flags().StringVar(&batchFile, "batch", "", "File with one domain per line (use - for stdin)")

	return cmd
}

func runExposureScan(cmd *cobra.Command, domain, targetType, batchFile string) error {
	// Resolve output format early.
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	quiet := IsQuiet()

	// Collect targets.
	targets, err := resolveExposureTargets(domain, batchFile)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		return fmt.Errorf("no target specified. Use --domain, --batch, or pipe via stdin")
	}

	// Build API client.
	apiKey := GetAPIKey()
	if apiKey == "" {
		return fmt.Errorf("no API key configured. Run: dfir-cli config init")
	}

	// Use a longer default timeout for exposure scans (3 minutes) unless the
	// user explicitly set a higher value via the global --timeout flag.
	timeout := GetTimeout()
	if timeout < 3*time.Minute {
		timeout = 3 * time.Minute
	}

	c := client.New(apiKey, GetAPIURL(), version.UserAgent(), timeout, IsVerbose())

	// Context with signal handling for Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Batch mode: iterate through all targets.
	isBatch := len(targets) > 1
	var exitCode int

	for i, target := range targets {
		if isBatch && !quiet {
			fmt.Fprintf(os.Stderr, "[%d/%d] Scanning %s...\n", i+1, len(targets), target)
		}

		code, scanErr := runSingleExposureScan(ctx, c, target, targetType, format, quiet)
		if scanErr != nil {
			fmt.Fprintf(os.Stderr, "Error scanning %s: %v\n", target, scanErr)
			if exitCode == 0 {
				exitCode = 1
			}
			continue
		}

		// Keep the highest severity exit code.
		if code > exitCode {
			exitCode = code
		}

		// Separator between batch results.
		if isBatch && i < len(targets)-1 && format == output.FormatTable && !quiet {
			fmt.Println()
		}
	}

	if exitCode != 0 {
		return &SilentExitError{Code: exitCode}
	}

	return nil
}

func runSingleExposureScan(
	ctx context.Context,
	c *client.Client,
	target, targetType string,
	format output.Format,
	quiet bool,
) (int, error) {
	req := &client.ExposureScanRequest{
		Target:     target,
		TargetType: targetType,
	}

	// Start spinner unless quiet or non-table output.
	spin := output.NewSpinner(fmt.Sprintf("Scanning %s...", target))
	if !quiet && format == output.FormatTable {
		output.StartSpinner(spin)
	}

	result, resp, err := c.ExposureScan(ctx, req)

	// Stop spinner before any output.
	output.StopSpinner(spin)

	if err != nil {
		// Check for insufficient credits.
		if client.IsCreditsError(err) {
			return 4, err
		}
		return 1, err
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	// Render output.
	switch format {
	case output.FormatJSON:
		payload := map[string]interface{}{
			"data": result,
			"meta": resp.Meta,
		}
		if err := output.PrintJSON(payload); err != nil {
			return 1, err
		}

	case output.FormatJSONL:
		payload := map[string]interface{}{
			"data": result,
			"meta": resp.Meta,
		}
		if err := output.PrintJSONL(payload); err != nil {
			return 1, err
		}

	default:
		if quiet {
			// Quiet mode: LEVEL SCORE TARGET
			fmt.Printf("%s %d %s\n",
				strings.ToUpper(result.RiskLevel),
				result.RiskScore,
				result.Target,
			)
		} else {
			printExposureTable(result, resp)
		}
	}

	return exposureExitCode(result.RiskLevel), nil
}

// printExposureTable renders the human-friendly table output.
func printExposureTable(result *client.ExposureScanResponse, resp *client.Response) {
	fmt.Println()
	output.PrintHeader(fmt.Sprintf("Exposure Scan: %s", result.Target))
	fmt.Println()

	// Risk level with color.
	fmt.Printf("  Risk Level:  %s\n", output.RiskBadge(result.RiskLevel))
	fmt.Printf("  Risk Score:  %s\n", output.ScoreBar(result.RiskScore, 100))

	fmt.Printf("  Status:      %s\n", result.Status)

	cached := "No"
	if result.Cached {
		cached = "Yes (cached result)"
	}
	fmt.Printf("  Cached:      %s\n", cached)

	if len(result.Providers) > 0 {
		fmt.Printf("  Providers:   %s\n", strings.Join(result.Providers, ", "))
	}

	// Show key fields from results.
	if result.Results != nil {
		// SSL grade if present.
		if ssl, ok := result.Results["ssl"]; ok {
			if sslMap, ok := ssl.(map[string]interface{}); ok {
				if grade, ok := sslMap["grade"]; ok {
					gradeStr := fmt.Sprintf("%v", grade)
					var gradeColor *color.Color
					switch {
					case gradeStr == "A" || gradeStr == "A+":
						gradeColor = output.Green
					case gradeStr == "B":
						gradeColor = output.Yellow
					default:
						gradeColor = output.Red
					}
					fmt.Printf("  SSL Grade:   %s\n", gradeColor.Sprint(gradeStr))
				}
			}
		}
	}

	// Duration.
	if result.Stats != nil && result.Stats.DurationMs > 0 {
		dur := float64(result.Stats.DurationMs) / 1000.0
		fmt.Printf("  Duration:    %.1fs\n", dur)
	}

	fmt.Println()

	// Credits.
	output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)

	// Suggest JSON for full details.
	if result.Results != nil && len(result.Results) > 0 {
		fmt.Println()
		output.Dim.Println("  For full details, re-run with --output json")
	}

	fmt.Println()
}

// exposureExitCode maps a risk level to a CLI exit code.
func exposureExitCode(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical", "high":
		return 2
	case "medium":
		return 3
	case "low", "none", "":
		return 0
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Input resolution helpers
// ---------------------------------------------------------------------------

// resolveExposureTargets collects scan targets from flag, batch file, or stdin.
func resolveExposureTargets(domain, batchFile string) ([]string, error) {
	// If --domain is set, use that.
	if domain != "" {
		return []string{strings.TrimSpace(domain)}, nil
	}

	// If --batch is set, read from file (or stdin if "-").
	if batchFile != "" {
		return readExposureBatch(batchFile)
	}

	// Check stdin for piped input.
	stdinTarget, err := readExposureStdin()
	if err != nil {
		return nil, err
	}
	if stdinTarget != "" {
		return []string{stdinTarget}, nil
	}

	return nil, nil
}

// readExposureStdin reads a single domain from stdin if data is piped.
func readExposureStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", nil
	}

	// Check if stdin has piped data (not a terminal).
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}

	return "", nil
}

// readExposureBatch reads domains from a file, one per line. Use "-" for stdin.
func readExposureBatch(path string) ([]string, error) {
	var r *os.File

	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open batch file: %w", err)
		}
		defer f.Close()
		r = f
	}

	var targets []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading batch input: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("batch input contained no targets")
	}

	return targets, nil
}
