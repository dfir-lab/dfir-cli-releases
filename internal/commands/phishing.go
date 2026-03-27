package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForeGuards/dfir-cli/internal/client"
	"github.com/ForeGuards/dfir-cli/internal/output"
	"github.com/ForeGuards/dfir-cli/internal/version"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const maxEmailFileSize = 5 * 1024 * 1024 // 5 MB

// NewPhishingCmd builds and returns the "phishing" command tree.
func NewPhishingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "phishing",
		Short: "Analyse phishing emails and URLs",
	}

	cmd.AddCommand(newPhishingAnalyzeCmd())

	return cmd
}

// ---------------------------------------------------------------------------
// phishing analyze
// ---------------------------------------------------------------------------

func newPhishingAnalyzeCmd() *cobra.Command {
	var (
		flagURL       string
		flagFile      string
		flagRaw       string
		flagInputType string
		flagAI        bool
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze an email for phishing indicators",
		Long: `Analyze a raw email, .eml file, or email headers for phishing indicators.

Input methods:
  --file suspicious.eml                         Read from an .eml file
  --raw "From: attacker@evil.com\n..."          Inline raw email content
  cat email.eml | dfir-cli phishing analyze     Pipe via stdin

Use --ai for AI-enhanced analysis (costs 10 credits instead of 1).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPhishingAnalyze(flagURL, flagFile, flagRaw, flagInputType, flagAI)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&flagURL, "url", "", "Analyze a URL (not yet supported — use enrichment lookup)")
	flags.StringVar(&flagFile, "file", "", "Path to an .eml email file")
	flags.StringVar(&flagRaw, "raw", "", "Raw email content as a string")
	flags.StringVar(&flagInputType, "type", "", "Input type: headers, eml, raw (auto-detected if omitted)")
	flags.BoolVar(&flagAI, "ai", false, "Use AI-enhanced analysis (10 credits)")

	return cmd
}

func runPhishingAnalyze(urlFlag, fileFlag, rawFlag, inputType string, ai bool) error {
	// --url is not supported by the analysis API.
	if urlFlag != "" {
		return fmt.Errorf("--url is not supported for phishing analysis.\n" +
			"The API analyses full emails, not individual URLs.\n" +
			"Tip: use  dfir-cli enrichment lookup --url <url>  to check a URL")
	}

	// Resolve content and input type.
	emailContent, detectedType, err := resolvePhishingInput(fileFlag, rawFlag)
	if err != nil {
		return err
	}

	if inputType == "" {
		inputType = detectedType
	}

	// Build API client inline (avoids helper name collisions across files).
	apiKey := GetAPIKey()
	if apiKey == "" {
		return fmt.Errorf("no API key configured. Run: dfir-cli config init")
	}
	c := client.New(apiKey, GetAPIURL(), version.UserAgent(), GetTimeout(), IsVerbose())

	// Context with Ctrl+C handling.
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.PhishingAnalyzeRequest{
		InputType: inputType,
		Content:   emailContent,
		Options: &client.PhishingAnalyzeOptions{
			IncludeIOCs:               true,
			IncludeBodyAnalysis:       true,
			IncludeHomoglyphCheck:     true,
			IncludeLinkAnalysis:       true,
			IncludeAttachmentAnalysis: true,
		},
	}

	// Determine output format.
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	if ai {
		return executePhishingAI(ctx, c, req, format)
	}
	return executePhishingStandard(ctx, c, req, format)
}

// ---------------------------------------------------------------------------
// Standard analysis
// ---------------------------------------------------------------------------

func executePhishingStandard(ctx context.Context, c *client.Client, req *client.PhishingAnalyzeRequest, format output.Format) error {
	spin := output.NewSpinner("Analyzing email for phishing indicators...")
	output.StartSpinner(spin)

	result, resp, err := c.PhishingAnalyze(ctx, req)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for phishing analysis")
	default:
		renderPhishingTable(result, nil, resp)
	}

	return phishingVerdictExit(result.Verdict.Level)
}

// ---------------------------------------------------------------------------
// AI-enhanced analysis
// ---------------------------------------------------------------------------

func executePhishingAI(ctx context.Context, c *client.Client, req *client.PhishingAnalyzeRequest, format output.Format) error {
	spin := output.NewSpinner("Running AI-enhanced phishing analysis...")
	output.StartSpinner(spin)

	result, resp, err := c.PhishingAnalyzeAI(ctx, req)
	output.StopSpinner(spin)
	if err != nil {
		return handlePhishingAPIError(err)
	}

	// Persist credit state for the credits command.
	if resp != nil {
		_ = SaveCreditState(&resp.Meta)
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(result)
	case output.FormatJSONL:
		return output.PrintJSONL(result)
	case output.FormatCSV:
		return fmt.Errorf("CSV output is not supported for phishing analysis")
	default:
		renderPhishingTable(&result.Analysis, result.AIVerdict, resp)
	}

	return phishingVerdictExit(result.Analysis.Verdict.Level)
}

// ---------------------------------------------------------------------------
// Input resolution
// ---------------------------------------------------------------------------

// resolvePhishingInput returns (content, inputType, error) from --file, --raw,
// or stdin.
func resolvePhishingInput(fileFlag, rawFlag string) (string, string, error) {
	const usage = "Usage:\n" +
		"  dfir-cli phishing analyze --file email.eml\n" +
		"  dfir-cli phishing analyze --raw \"From: ...\"\n" +
		"  cat email.eml | dfir-cli phishing analyze"

	// --file takes precedence.
	if fileFlag != "" {
		return readPhishingEmailFile(fileFlag)
	}

	// --raw is next.
	if rawFlag != "" {
		return rawFlag, "raw", nil
	}

	// Try stdin.
	data, err := readStdin()
	if err != nil {
		return "", "", fmt.Errorf("reading stdin: %w", err)
	}
	if data != nil {
		content := strings.TrimSpace(string(data))
		if content == "" {
			return "", "", fmt.Errorf("no input provided.\n\n%s", usage)
		}
		return content, "raw", nil
	}

	return "", "", fmt.Errorf("no input provided.\n\n%s", usage)
}

// readPhishingEmailFile reads and validates an email file from disk.
func readPhishingEmailFile(path string) (string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("file not found: %s", path)
		}
		return "", "", fmt.Errorf("accessing file: %w", err)
	}

	if info.Size() > maxEmailFileSize {
		return "", "", fmt.Errorf("file too large (max 5MB)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("reading file: %w", err)
	}

	// Auto-detect input type from extension.
	inputType := "raw"
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".eml" {
		inputType = "eml"
	}

	return string(data), inputType, nil
}

// ---------------------------------------------------------------------------
// Table output
// ---------------------------------------------------------------------------

func renderPhishingTable(r *client.PhishingAnalyzeResponse, aiVerdict *client.AIVerdict, resp *client.Response) {
	// Quiet mode: just verdict and score.
	if IsQuiet() {
		fmt.Printf("%s %d\n", strings.ToUpper(r.Verdict.Level), r.Verdict.Score)
		return
	}

	fmt.Println()
	output.Bold.Println("Phishing Analysis")
	fmt.Println()

	// Verdict line.
	verdictStr := fmt.Sprintf("%s", strings.ToUpper(r.Verdict.Level))
	vc := phishingLevelColor(r.Verdict.Level)
	fmt.Printf("  Verdict:    %s\n", vc.Sprint(verdictStr))
	fmt.Printf("  Score:      %s\n", output.ScoreBar(r.Verdict.Score, 100))
	if r.Verdict.Summary != "" {
		fmt.Printf("  Summary:    %s\n", r.Verdict.Summary)
	}

	// AI verdict (if present).
	if aiVerdict != nil {
		fmt.Println()
		output.Bold.Println("  AI Assessment")
		aiStr := fmt.Sprintf("%s (confidence: %d%%)", strings.ToUpper(aiVerdict.RiskLevel), aiVerdict.ConfidenceScore)
		aiVC := phishingLevelColor(aiVerdict.RiskLevel)
		fmt.Printf("    Risk Level:  %s\n", aiVC.Sprint(aiStr))
		if aiVerdict.ExecutiveSummary != "" {
			fmt.Printf("    Summary:     %s\n", aiVerdict.ExecutiveSummary)
		}
		if aiVerdict.Model != "" {
			fmt.Printf("    Model:       %s\n", output.Dim.Sprint(aiVerdict.Model))
		}

		if len(aiVerdict.KeyFindings) > 0 {
			fmt.Println()
			fmt.Println("    AI Key Findings:")
			for _, f := range aiVerdict.KeyFindings {
				fmt.Printf("      - %s\n", f)
			}
		}

		if len(aiVerdict.RecommendedActions) > 0 {
			fmt.Println()
			fmt.Println("    AI Recommended Actions:")
			for i, a := range aiVerdict.RecommendedActions {
				fmt.Printf("      %d. %s\n", i+1, a)
			}
		}
	}

	// Authentication results.
	if r.AuthenticationResults != nil {
		fmt.Println()
		output.Bold.Println("  Authentication Results:")
		printPhishingAuthField("SPF", r.AuthenticationResults.SPF)
		printPhishingAuthField("DKIM", r.AuthenticationResults.DKIM)
		printPhishingAuthField("DMARC", r.AuthenticationResults.DMARC)
		if r.AuthenticationResults.ARC != "" {
			printPhishingAuthField("ARC", r.AuthenticationResults.ARC)
		}
	}

	// Key findings.
	if len(r.KeyFindings) > 0 {
		fmt.Println()
		output.Bold.Println("  Key Findings:")
		for _, f := range r.KeyFindings {
			fmt.Printf("    - %s\n", f)
		}
	}

	// Suspicious indicators table.
	if len(r.SuspiciousIndicators) > 0 {
		fmt.Println()
		output.Bold.Println("  Suspicious Indicators:")

		t := output.NewTable()
		t.Style().Options.DrawBorder = false
		t.Style().Options.SeparateHeader = true
		t.Style().Options.SeparateRows = false
		t.Style().Options.SeparateColumns = true

		t.AppendHeader(table.Row{"Category", "Description", "Severity"})

		for _, si := range r.SuspiciousIndicators {
			t.AppendRow(table.Row{si.Category, si.Description, output.SeverityBadge(si.Severity)})
		}

		t.Render()
	}

	// Extracted IOCs table.
	if len(r.ExtractedIOCs) > 0 {
		fmt.Println()
		output.Bold.Println("  Extracted IOCs:")

		t := output.NewTable()
		t.Style().Options.DrawBorder = false
		t.Style().Options.SeparateHeader = true
		t.Style().Options.SeparateRows = false
		t.Style().Options.SeparateColumns = true

		t.AppendHeader(table.Row{"Type", "Value", "Verdict"})

		for _, ioc := range r.ExtractedIOCs {
			verdict := ioc.EnrichmentVerdict
			if verdict == "" {
				verdict = "-"
			}
			t.AppendRow(table.Row{ioc.Type, ioc.Value, output.VerdictBadge(verdict)})
		}

		t.Render()
	}

	// Recommended actions.
	if len(r.RecommendedActions) > 0 {
		fmt.Println()
		output.Bold.Println("  Recommended Actions:")
		for i, a := range r.RecommendedActions {
			fmt.Printf("    %d. %s\n", i+1, a)
		}
	}

	// Credits footer.
	if resp != nil {
		fmt.Println()
		output.PrintCreditsFooter(resp.Meta.CreditsUsed, resp.Meta.CreditsRemaining)
	}

	fmt.Println()
}

// printPhishingAuthField prints a single authentication result field with color.
func printPhishingAuthField(label, value string) {
	if value == "" {
		value = "none"
	}
	fmt.Printf("    %-10s %s\n", label+":", output.AuthBadge(value))
}

// phishingLevelColor maps verdict/risk levels to terminal colors. This extends
// the generic VerdictColor with phishing-specific levels such as
// "highly_malicious" and "safe".
func phishingLevelColor(level string) *color.Color {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "highly_malicious":
		return color.New(color.FgRed, color.Bold)
	case "malicious":
		return output.Red
	case "suspicious":
		return output.Yellow
	case "safe", "clean":
		return output.Green
	default:
		return output.Dim
	}
}

// ---------------------------------------------------------------------------
// Error handling & exit codes
// ---------------------------------------------------------------------------

// phishingVerdictExit returns a SilentExitError for non-safe verdicts so the
// process exits with the correct code. Returns nil for safe verdicts (exit 0).
func phishingVerdictExit(level string) error {
	code := exitCodeForVerdict(level)
	if code == 0 {
		return nil
	}
	return &SilentExitError{Code: code}
}

// handlePhishingAPIError wraps API errors with user-friendly messages and exit
// codes. Insufficient credit errors get exit code 4.
func handlePhishingAPIError(err error) error {
	if err == nil {
		return nil
	}

	var creditsErr *client.InsufficientCreditsError
	if errors.As(err, &creditsErr) {
		fmt.Fprintln(os.Stderr, "Error: insufficient credits to perform this analysis.")
		fmt.Fprintln(os.Stderr, "       Check your balance: dfir-cli credits")
		return &SilentExitError{Code: 4}
	}

	return err
}
