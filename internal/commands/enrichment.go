package commands

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/ForeGuards/dfir-cli/internal/client"
	"github.com/ForeGuards/dfir-cli/internal/output"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// batchAPILimit is the maximum number of indicators per API request.
const batchAPILimit = 10

// validIOCTypes lists the accepted values for --type.
var validIOCTypes = []string{"ip", "domain", "url", "hash", "email"}

// ---------------------------------------------------------------------------
// IOC type auto-detection
// ---------------------------------------------------------------------------

var hexRe = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// detectIOCType inspects value and returns the most likely IOC type:
// ip, email, hash, url, or domain.
func detectIOCType(value string) string {
	value = strings.TrimSpace(value)

	// IPv4 and IPv6 — use net.ParseIP for correctness.
	if net.ParseIP(value) != nil {
		return "ip"
	}

	// Email
	if strings.Contains(value, "@") {
		return "email"
	}

	// Hashes: MD5 (32), SHA-1 (40), SHA-256 (64)
	if hexRe.MatchString(value) {
		switch len(value) {
		case 32, 40, 64:
			return "hash"
		}
	}

	// URL
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "url"
	}

	// Default: domain
	return "domain"
}

// isValidIOCType returns true if t is one of the accepted IOC types.
func isValidIOCType(t string) bool {
	for _, v := range validIOCTypes {
		if v == t {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Command constructors
// ---------------------------------------------------------------------------

// NewEnrichmentCmd returns the top-level "enrichment" command with all
// subcommands attached.
func NewEnrichmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enrichment",
		Short: "Enrich IOCs across threat intelligence providers",
	}

	cmd.AddCommand(newEnrichmentLookupCmd())
	return cmd
}

func newEnrichmentLookupCmd() *cobra.Command {
	var (
		flagIP        string
		flagDomain    string
		flagURL       string
		flagHash      string
		flagEmail     string
		flagIOC       string
		flagType      string
		flagBatch     string
		flagProviders string
		flagMinScore  int
	)

	cmd := &cobra.Command{
		Use:   "lookup",
		Short: "Enrich IOCs across threat intelligence providers",
		Long: `Look up indicators of compromise (IOCs) across multiple threat intelligence
providers. Supports IPs, domains, URLs, hashes, and email addresses.

Input can be supplied via typed flags, the generic --ioc flag, a batch file,
or stdin.`,
		Example: `  dfir-cli enrichment lookup --ip 1.2.3.4
  dfir-cli enrichment lookup --domain evil.com
  dfir-cli enrichment lookup --ioc evil.com
  dfir-cli enrichment lookup --batch iocs.txt --type ip
  echo "1.2.3.4" | dfir-cli enrichment lookup --type ip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnrichmentLookup(cmd, enrichmentLookupFlags{
				ip:        flagIP,
				domain:    flagDomain,
				url:       flagURL,
				hash:      flagHash,
				email:     flagEmail,
				ioc:       flagIOC,
				iocType:   flagType,
				batch:     flagBatch,
				providers: flagProviders,
				minScore:  flagMinScore,
			})
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagIP, "ip", "", "Look up an IP address")
	flags.StringVar(&flagDomain, "domain", "", "Look up a domain")
	flags.StringVar(&flagURL, "url", "", "Look up a URL")
	flags.StringVar(&flagHash, "hash", "", "Look up a file hash")
	flags.StringVar(&flagEmail, "email", "", "Look up an email address")
	flags.StringVar(&flagIOC, "ioc", "", "Look up any IOC (auto-detect type)")
	flags.StringVar(&flagType, "type", "", "Force IOC type: ip, domain, url, hash, email")
	flags.StringVar(&flagBatch, "batch", "", "File with one IOC per line (use - for stdin)")
	flags.StringVar(&flagProviders, "providers", "", "Comma-separated provider filter")
	flags.IntVar(&flagMinScore, "min-score", 0, "Only show providers above this score (0-100)")

	return cmd
}

// enrichmentLookupFlags holds the parsed flag values for the lookup subcommand.
type enrichmentLookupFlags struct {
	ip        string
	domain    string
	url       string
	hash      string
	email     string
	ioc       string
	iocType   string
	batch     string
	providers string
	minScore  int
}

// ---------------------------------------------------------------------------
// Core execution logic
// ---------------------------------------------------------------------------

func runEnrichmentLookup(cmd *cobra.Command, f enrichmentLookupFlags) error {
	// Validate --type if provided.
	if f.iocType != "" && !isValidIOCType(f.iocType) {
		return fmt.Errorf("invalid IOC type %q. Valid types: %s",
			f.iocType, strings.Join(validIOCTypes, ", "))
	}

	// Build indicators list using precedence: typed flag > --ioc > --batch > stdin.
	indicators, err := resolveIndicators(f)
	if err != nil {
		return err
	}

	if len(indicators) == 0 {
		return cmd.Help()
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

	// Chunk indicators into batches of batchAPILimit.
	chunks := chunkIndicators(indicators, batchAPILimit)

	var allResults []client.EnrichmentResult
	var lastMeta *client.Response

	spin := output.NewSpinner("Enriching indicators...")
	output.StartSpinner(spin)

	for _, chunk := range chunks {
		req := &client.EnrichmentRequest{Indicators: chunk}
		result, resp, apiErr := apiClient.EnrichmentLookup(ctx, req)
		if apiErr != nil {
			output.StopSpinner(spin)
			return classifyEnrichmentError(apiErr)
		}
		allResults = append(allResults, result.Results...)
		lastMeta = resp
	}

	output.StopSpinner(spin)

	// Persist credit state for the credits command.
	if lastMeta != nil {
		_ = SaveCreditState(&lastMeta.Meta)
	}

	// Apply --providers and --min-score filters.
	allResults = filterResults(allResults, f.providers, f.minScore)

	// Render output.
	switch {
	case IsQuiet():
		return renderEnrichmentQuiet(allResults)
	case format == output.FormatJSON:
		return renderEnrichmentJSON(allResults, lastMeta)
	case format == output.FormatJSONL:
		return renderEnrichmentJSONL(allResults)
	default:
		return renderEnrichmentTable(allResults, lastMeta)
	}
}

// ---------------------------------------------------------------------------
// Indicator resolution
// ---------------------------------------------------------------------------

// resolveIndicators builds the indicator list from flags, batch file, or stdin.
func resolveIndicators(f enrichmentLookupFlags) ([]client.Indicator, error) {
	// 1. Typed flags (highest precedence).
	if f.ip != "" {
		return []client.Indicator{{Type: "ip", Value: f.ip}}, nil
	}
	if f.domain != "" {
		return []client.Indicator{{Type: "domain", Value: f.domain}}, nil
	}
	if f.url != "" {
		return []client.Indicator{{Type: "url", Value: f.url}}, nil
	}
	if f.hash != "" {
		return []client.Indicator{{Type: "hash", Value: f.hash}}, nil
	}
	if f.email != "" {
		return []client.Indicator{{Type: "email", Value: f.email}}, nil
	}

	// 2. Generic --ioc flag.
	if f.ioc != "" {
		iocType := f.iocType
		if iocType == "" {
			iocType = detectIOCType(f.ioc)
		}
		return []client.Indicator{{Type: iocType, Value: f.ioc}}, nil
	}

	// 3. Batch file.
	if f.batch != "" {
		return readBatchIndicators(f.batch, f.iocType)
	}

	// 4. Stdin (only when not a terminal).
	stdinData, err := readStdin()
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if stdinData != nil {
		if f.iocType == "" {
			return nil, fmt.Errorf("--type is required when reading from stdin")
		}
		return parseIOCLines(string(stdinData), f.iocType), nil
	}

	return nil, nil
}

// readBatchIndicators reads IOCs from a file (or stdin if path is "-").
func readBatchIndicators(path, iocType string) ([]client.Indicator, error) {
	if path == "-" && iocType == "" {
		return nil, fmt.Errorf("--type is required when reading from stdin")
	}

	lines, err := readLines(path)
	if err != nil {
		return nil, fmt.Errorf("reading batch file: %w", err)
	}

	var indicators []client.Indicator
	for _, line := range lines {
		t := iocType
		if t == "" {
			t = detectIOCType(line)
		}
		indicators = append(indicators, client.Indicator{Type: t, Value: line})
	}
	return indicators, nil
}

// parseIOCLines splits raw text into IOC indicators, skipping empty lines
// and comments.
func parseIOCLines(data, iocType string) []client.Indicator {
	var indicators []client.Indicator
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		t := iocType
		if t == "" {
			t = detectIOCType(line)
		}
		indicators = append(indicators, client.Indicator{Type: t, Value: line})
	}
	return indicators
}

// chunkIndicators splits indicators into groups of at most n.
func chunkIndicators(indicators []client.Indicator, n int) [][]client.Indicator {
	if n <= 0 {
		n = batchAPILimit
	}
	var chunks [][]client.Indicator
	for i := 0; i < len(indicators); i += n {
		end := i + n
		if end > len(indicators) {
			end = len(indicators)
		}
		chunks = append(chunks, indicators[i:end])
	}
	return chunks
}

// ---------------------------------------------------------------------------
// Filtering
// ---------------------------------------------------------------------------

// filterResults applies --providers and --min-score filters to the results.
func filterResults(results []client.EnrichmentResult, providers string, minScore int) []client.EnrichmentResult {
	providerSet := parseProviderFilter(providers)

	for i := range results {
		if len(providerSet) > 0 || minScore > 0 {
			filtered := make(map[string]client.ProviderResult)
			for name, pr := range results[i].Providers {
				if len(providerSet) > 0 {
					if _, ok := providerSet[strings.ToLower(name)]; !ok {
						continue
					}
				}
				if minScore > 0 && pr.Score < minScore {
					continue
				}
				filtered[name] = pr
			}
			results[i].Providers = filtered
		}
	}
	return results
}

// parseProviderFilter splits a comma-separated provider list into a set of
// lowercase names. Returns nil if the input is empty.
func parseProviderFilter(s string) map[string]struct{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	set := make(map[string]struct{})
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			set[p] = struct{}{}
		}
	}
	return set
}

// ---------------------------------------------------------------------------
// Output: table
// ---------------------------------------------------------------------------

func renderEnrichmentTable(results []client.EnrichmentResult, meta *client.Response) error {
	for idx, r := range results {
		if idx > 0 {
			fmt.Println()
		}
		printEnrichmentHeader(r)
		printProviderTable(r)
	}

	// Credits footer.
	if meta != nil {
		fmt.Printf("\n  Credits: %d used, %d remaining\n",
			meta.Meta.CreditsUsed, meta.Meta.CreditsRemaining)
	}

	return enrichmentExitCode(results)
}

// printEnrichmentHeader renders the summary block above the provider table.
func printEnrichmentHeader(r client.EnrichmentResult) {
	iocLabel := strings.ToUpper(r.Indicator.Type)
	fmt.Printf("\nIOC Enrichment: %s (%s)\n\n", r.Indicator.Value, iocLabel)

	vc := output.VerdictColor(r.Verdict)
	fmt.Printf("  Verdict:    %s\n", vc.Sprint(strings.ToUpper(r.Verdict)))
	fmt.Printf("  Score:      %d/100\n", r.Score)

	total := len(r.Providers)
	flagged := countFlaggedProviders(r.Providers)
	fmt.Printf("  Consensus:  %d/%d providers flagged\n\n", flagged, total)
}

// countFlaggedProviders returns the number of providers with a malicious or
// suspicious verdict.
func countFlaggedProviders(providers map[string]client.ProviderResult) int {
	count := 0
	for _, pr := range providers {
		v := strings.ToLower(pr.Verdict)
		if v == "malicious" || v == "suspicious" {
			count++
		}
	}
	return count
}

// printProviderTable renders the per-provider table using go-pretty.
func printProviderTable(r client.EnrichmentResult) {
	if len(r.Providers) == 0 {
		fmt.Println("  No provider results.")
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Format.Header = text.FormatDefault

	t.AppendHeader(table.Row{"Provider", "Verdict", "Score", "Details"})

	for name, pr := range r.Providers {
		vc := output.VerdictColor(pr.Verdict)
		details := formatProviderDetails(pr.Details)
		t.AppendRow(table.Row{
			name,
			vc.Sprint(strings.ToUpper(pr.Verdict)),
			fmt.Sprintf("%d", pr.Score),
			details,
		})
	}

	t.Render()
}

// formatProviderDetails flattens the details map into a human-readable string.
func formatProviderDetails(details map[string]interface{}) string {
	if len(details) == 0 {
		return ""
	}
	var parts []string
	for k, v := range details {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(parts, ", ")
}

// ---------------------------------------------------------------------------
// Output: JSON
// ---------------------------------------------------------------------------

func renderEnrichmentJSON(results []client.EnrichmentResult, meta *client.Response) error {
	payload := struct {
		Results []client.EnrichmentResult `json:"results"`
		Meta    *client.ResponseMeta      `json:"meta,omitempty"`
	}{
		Results: results,
	}
	if meta != nil {
		payload.Meta = &meta.Meta
	}
	if err := output.PrintJSON(payload); err != nil {
		return err
	}
	return enrichmentExitCode(results)
}

// ---------------------------------------------------------------------------
// Output: JSONL
// ---------------------------------------------------------------------------

func renderEnrichmentJSONL(results []client.EnrichmentResult) error {
	for _, r := range results {
		if err := output.PrintJSONL(r); err != nil {
			return err
		}
	}
	return enrichmentExitCode(results)
}

// ---------------------------------------------------------------------------
// Output: quiet
// ---------------------------------------------------------------------------

func renderEnrichmentQuiet(results []client.EnrichmentResult) error {
	for _, r := range results {
		fmt.Printf("%s %d %s\n", strings.ToUpper(r.Verdict), r.Score, r.Indicator.Value)
	}
	return enrichmentExitCode(results)
}

// ---------------------------------------------------------------------------
// Exit codes
// ---------------------------------------------------------------------------

// enrichmentExitCode returns a SilentExitError encoding the appropriate exit
// code based on the highest-severity verdict found:
//
//	0 — all clean or unknown
//	2 — at least one malicious
//	3 — at least one suspicious (but no malicious)
func enrichmentExitCode(results []client.EnrichmentResult) error {
	hasMalicious := false
	hasSuspicious := false

	for _, r := range results {
		switch strings.ToLower(r.Verdict) {
		case "malicious":
			hasMalicious = true
		case "suspicious":
			hasSuspicious = true
		}
	}

	switch {
	case hasMalicious:
		return &SilentExitError{Code: 2}
	case hasSuspicious:
		return &SilentExitError{Code: 3}
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Error classification
// ---------------------------------------------------------------------------

// classifyEnrichmentError maps known API errors to appropriate exit codes.
func classifyEnrichmentError(err error) error {
	if err == nil {
		return nil
	}

	var creditsErr *client.InsufficientCreditsError
	if errors.As(err, &creditsErr) {
		fmt.Fprintln(os.Stderr, "Error:", creditsErr.Error())
		return &SilentExitError{Code: 4}
	}

	return err
}
