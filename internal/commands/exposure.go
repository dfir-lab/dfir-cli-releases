package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/dfir-lab/dfir-cli/internal/version"
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
		domain      string
		targetType  string
		batchFile   string
		concurrency int
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
			return runExposureScan(cmd, domain, targetType, batchFile, concurrency)
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Target domain to scan")
	cmd.Flags().StringVar(&targetType, "target-type", "auto", "Target type hint: domain, ip, auto")
	cmd.Flags().StringVar(&batchFile, "batch", "", "File with one domain per line (use - for stdin)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Parallel requests for batch mode (1-20)")

	return cmd
}

func runExposureScan(cmd *cobra.Command, domain, targetType, batchFile string, concurrency int) error {
	// Validate concurrency.
	if concurrency < 1 || concurrency > 20 {
		return fmt.Errorf("--concurrency must be between 1 and 20, got %d", concurrency)
	}

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

	// Context with shared Ctrl+C handling.
	ctx, cancel := signalContext()
	defer cancel()

	// Batch mode: iterate through all targets.
	isBatch := len(targets) > 1
	var exitCode int

	if !isBatch {
		// Single target path.
		code, scanErr := runSingleExposureScan(ctx, c, targets[0], targetType, format, quiet)
		if scanErr != nil {
			fmt.Fprintf(os.Stderr, "Error scanning %s: %v\n", targets[0], scanErr)
			exitCode = 1
		} else if code > exitCode {
			exitCode = code
		}
	} else {
		// Multiple targets: fetch results concurrently, render sequentially.
		type scanResult struct {
			target string
			result *client.ExposureScanResponse
			resp   *client.Response
			err    error
		}

		results := make([]scanResult, len(targets))
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup

		for i, target := range targets {
			wg.Add(1)
			sem <- struct{}{} // acquire semaphore
			go func(i int, target string) {
				defer wg.Done()
				defer func() { <-sem }() // release semaphore

				if !quiet {
					fmt.Fprintf(os.Stderr, "[%d/%d] Scanning %s...\n", i+1, len(targets), target)
				}

				req := &client.ExposureScanRequest{
					Target:     target,
					TargetType: targetType,
				}
				result, resp, err := c.ExposureScan(ctx, req)
				results[i] = scanResult{target: target, result: result, resp: resp, err: err}
			}(i, target)
		}
		wg.Wait()

		// Render results sequentially (safe for stdout).
		var lastResp *client.Response
		for i, r := range results {
			if r.err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning %s: %v\n", r.target, r.err)
				if client.IsCreditsError(r.err) {
					exitCode = 4
				} else if exitCode == 0 {
					exitCode = 1
				}
				continue
			}

			if r.resp != nil {
				lastResp = r.resp
			}

			code := renderExposureResult(r.result, r.resp, format, quiet)
			if code > exitCode {
				exitCode = code
			}

			// Separator between batch results (only for table format).
			if i < len(results)-1 && format == output.FormatTable && !quiet {
				fmt.Println()
			}
		}

		// Persist credit state once after all scans.
		if lastResp != nil {
			_ = SaveCreditState(&lastResp.Meta)
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
		_ = SaveAPIState(&resp.Meta, "exposure", "scan")
	}

	code := renderExposureResult(result, resp, format, quiet)
	return code, nil
}

// renderExposureResult renders a single exposure result and returns the exit code.
// This function is safe to call from the main goroutine only.
func renderExposureResult(result *client.ExposureScanResponse, resp *client.Response, format output.Format, quiet bool) int {
	switch format {
	case output.FormatJSON:
		payload := map[string]interface{}{
			"data": result,
		}
		if resp != nil {
			payload["meta"] = resp.Meta
		}
		_ = output.PrintJSON(payload)

	case output.FormatJSONL:
		payload := map[string]interface{}{
			"data": result,
		}
		if resp != nil {
			payload["meta"] = resp.Meta
		}
		_ = output.PrintJSONL(payload)

	default:
		if quiet {
			fmt.Printf("%s %d %s\n",
				strings.ToUpper(result.RiskLevel),
				result.RiskScore,
				result.Target,
			)
		} else {
			printExposureTable(result, resp)
		}
	}

	return exposureExitCode(result.RiskLevel)
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
		names := make([]string, 0, len(result.Providers))
		for _, p := range result.Providers {
			names = append(names, p.Name)
		}
		fmt.Printf("  Providers:   %s\n", strings.Join(names, ", "))
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
