package commands

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/config"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed ai_system_prompt.txt
var aiSystemPrompt string

const (
	maxAIContextSize = 100 * 1024 // 100KB max piped input for AI context
)

const aiIdentityDisclosureResponse = `I'm a specialized DFIR (Digital Forensics and Incident Response) assistant within the dfir-cli tool.

My specific model version and technical details aren't part of what I disclose in this context, but I'm designed to help with:

- Digital forensics analysis
- Incident response workflows
- Malware analysis
- Log analysis and IOC extraction
- Forensic artifact interpretation
- Threat intelligence correlation

How can I help with your investigation or DFIR work?`

// NewAICmd creates the top-level "ai" command.
func NewAICmd() *cobra.Command {
	var (
		flagModel    string
		flagNoStream bool
	)

	cmd := &cobra.Command{
		Use:   "ai [question]",
		Short: "AI-powered DFIR assistant",
		Long: `Interactive AI assistant specialized in digital forensics and incident response.

Ask questions about forensic artifacts, pipe in tool output for analysis,
or start an interactive chat session for extended investigations.

The AI assistant only answers DFIR-related questions. Topics outside
digital forensics and incident response will be declined.

Requires a Starter, Professional, or Enterprise plan.

Examples:
  # Ask a one-shot question
  dfir-cli ai "What Windows event IDs indicate lateral movement?"

  # Pipe tool output for analysis
  vol.py -f memory.dmp windows.pslist | dfir-cli ai "Analyze for anomalies"

  # Pipe dfir-cli results for deeper analysis
  dfir-cli enrichment lookup --ip 1.2.3.4 -j | dfir-cli ai "Explain these results"

  # Start an interactive chat session
  dfir-cli ai chat

  # Use a specific model
  dfir-cli ai --model haiku "Quick: what is shimcache?"`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args and no piped input, show help
			if len(args) == 0 && output.IsTerminal() {
				return cmd.Help()
			}

			question := strings.Join(args, " ")
			return runAIOneShot(question, flagModel, flagNoStream)
		},
	}

	// --model is persistent so subcommands (e.g. "ai chat") inherit it.
	cmd.PersistentFlags().StringVar(&flagModel, "model", "", `AI model to use: "haiku" (fast) or "sonnet" (thorough). Default from config or "sonnet"`)
	// --no-stream is local (not relevant for chat REPL).
	cmd.Flags().BoolVar(&flagNoStream, "no-stream", false, "Disable streaming (wait for complete response)")

	// Add chat subcommand
	cmd.AddCommand(newAIChatCmd())

	return cmd
}

// runAIOneShot handles a single AI question with optional piped context.
func runAIOneShot(question, model string, noStream bool) error {
	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

	if response, ok := localAIIdentityDisclosureResponse(question); ok {
		return renderLocalAIResponse(response, format)
	}

	c, err := newAIClient()
	if err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	// Resolve model
	if model == "" {
		model = resolveAIModel()
	}
	if model != "haiku" && model != "sonnet" {
		return &ExitError{Code: 1, Message: "model must be 'haiku' or 'sonnet', got " + model}
	}

	// Read piped input if available
	var pipedInput string
	if !output.IsTerminal() || question == "" {
		data, err := readStdin()
		if err == nil && len(data) > 0 {
			if len(data) > maxAIContextSize {
				data = data[:maxAIContextSize]
				pipedInput = string(data) + "\n\n[Input truncated at 100KB]"
			} else {
				pipedInput = string(data)
			}
		}
	}

	// Build user message
	userContent := buildUserMessage(question, pipedInput)
	if userContent == "" {
		return &ExitError{Code: 1, Message: "no question provided — pass a question as arguments or pipe data via stdin"}
	}

	// Build request
	req := &client.AIChatRequest{
		Messages: []client.AIChatMessage{
			{Role: "user", Content: userContent},
		},
		Model:  model,
		Stream: !noStream,
	}

	if noStream || format == output.FormatJSON || format == output.FormatJSONL {
		return runAINonStreaming(ctx, c, req, format)
	}

	return runAIStreaming(ctx, c, req, format)
}

// runAIStreaming handles a streaming AI request.
func runAIStreaming(ctx context.Context, c *client.Client, req *client.AIChatRequest, format output.Format) error {
	reader, err := c.AIChatStream(ctx, req)
	if err != nil {
		return handleAIError(err)
	}
	defer reader.Close()

	var fullText strings.Builder

	// Print header
	if output.IsTerminal() {
		output.PrintHeader("AI Assistant")
	}

	for reader.Next() {
		event := reader.Event()
		switch event.Type {
		case "content_delta":
			fmt.Print(event.Text)
			fullText.WriteString(event.Text)
		case "done":
			if event.Meta != nil {
				_ = SaveAPIState(event.Meta, "ai", "chat")
			}
			// Print newline after streaming content
			fmt.Println()
			if output.IsTerminal() && event.Meta != nil {
				fmt.Println()
				output.PrintCreditsFooter(event.Meta.CreditsUsed, event.Meta.CreditsRemaining)
			}
		}
	}

	if err := reader.Err(); err != nil {
		return handleAIError(err)
	}

	return nil
}

// runAINonStreaming handles a non-streaming AI request (JSON output).
func runAINonStreaming(ctx context.Context, c *client.Client, req *client.AIChatRequest, format output.Format) error {
	// Always stream from API, just don't display incrementally
	req.Stream = true
	reader, err := c.AIChatStream(ctx, req)
	if err != nil {
		return handleAIError(err)
	}
	defer reader.Close()

	spin := output.NewSpinner("Thinking...")
	output.StartSpinner(spin)

	var fullText strings.Builder
	var usage *client.AIChatTokenUsage
	var meta *client.ResponseMeta

	for reader.Next() {
		event := reader.Event()
		switch event.Type {
		case "content_delta":
			fullText.WriteString(event.Text)
		case "done":
			usage = event.Usage
			meta = event.Meta
		}
	}
	output.StopSpinner(spin)

	if err := reader.Err(); err != nil {
		return handleAIError(err)
	}
	if meta != nil {
		_ = SaveAPIState(meta, "ai", "chat")
	}

	response := map[string]interface{}{
		"response": fullText.String(),
		"model":    req.Model,
	}
	if usage != nil {
		response["usage"] = usage
	}

	switch format {
	case output.FormatJSON:
		return output.PrintJSON(response)
	case output.FormatJSONL:
		return output.PrintJSONL(response)
	default:
		// Table mode — render markdown
		rendered := renderMarkdown(fullText.String())
		fmt.Print(rendered)
		return nil
	}
}

// buildUserMessage constructs the user message with optional piped context.
func buildUserMessage(question, pipedInput string) string {
	var parts []string

	if pipedInput != "" {
		parts = append(parts, "<piped_input>\n"+pipedInput+"\n</piped_input>")
	}

	if question != "" {
		parts = append(parts, question)
	} else if pipedInput != "" {
		parts = append(parts, "Analyze the data above. Identify any anomalies, suspicious patterns, or indicators of compromise.")
	}

	return strings.Join(parts, "\n\n")
}

func localAIIdentityDisclosureResponse(question string) (string, bool) {
	normalized := normalizeAIIdentityQuestion(question)
	if normalized == "" {
		return "", false
	}

	patterns := []string{
		"who are you",
		"who is this",
		"what model are you",
		"which model are you",
		"what ai model are you",
		"what kind of model are you",
		"what assistant are you",
		"what kind of assistant are you",
		"what are you exactly",
		"tell me what model you are",
		"tell me who you are",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return aiIdentityDisclosureResponse, true
		}
	}

	return "", false
}

func normalizeAIIdentityQuestion(input string) string {
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func renderLocalAIResponse(response string, format output.Format) error {
	switch format {
	case output.FormatJSON:
		return output.PrintJSON(map[string]interface{}{"response": response})
	case output.FormatJSONL:
		return output.PrintJSONL(map[string]interface{}{"response": response})
	default:
		fmt.Println(response)
		return nil
	}
}

// resolveAIModel returns the configured AI model, defaulting to "sonnet".
func resolveAIModel() string {
	profile := viper.GetString("profile")
	if profile == "" {
		profile = "default"
	}
	p, err := config.Load(profile)
	if err == nil && p.AIModel != "" {
		return p.AIModel
	}
	return "sonnet"
}

// handleAIError provides user-friendly error messages for AI-specific errors.
// It uses the typed error system from the client package rather than fragile
// string matching.
func handleAIError(err error) error {
	var creditsErr *client.InsufficientCreditsError
	if errors.As(err, &creditsErr) {
		fmt.Fprintln(os.Stderr, "Error: insufficient credits for AI chat.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Top up at: https://platform.dfir-lab.ch/billing")
		return &SilentExitError{Code: 4}
	}

	var authzErr *client.AuthorizationError
	if errors.As(err, &authzErr) {
		// 403 can mean plan gating or missing permission
		msg := authzErr.Message
		if strings.Contains(strings.ToLower(msg), "plan") {
			fmt.Fprintln(os.Stderr, "Error: AI features require a Starter or Professional plan.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  Upgrade at: https://platform.dfir-lab.ch/billing")
		} else {
			fmt.Fprintln(os.Stderr, "Error: your API key does not have the ai:chat permission.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  Update permissions at: https://platform.dfir-lab.ch/api-keys")
		}
		return &SilentExitError{Code: 1}
	}

	var authnErr *client.AuthenticationError
	if errors.As(err, &authnErr) {
		return err // Let the default error handler show the "run config init" message
	}

	var notFoundErr *client.NotFoundError
	if errors.As(err, &notFoundErr) {
		fmt.Fprintln(os.Stderr, "Error: AI chat is not available on the configured DFIR Platform API endpoint.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  API URL: %s\n", GetAIAPIURL())
		fmt.Fprintln(os.Stderr, "  The configured host is responding, but the /ai/chat route currently returns 404.")
		fmt.Fprintln(os.Stderr, "  The CLI is using the dedicated AI API host for this request.")
		return &SilentExitError{Code: 1}
	}

	return err
}
