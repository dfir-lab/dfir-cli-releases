package commands

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

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
	c, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	format, err := output.ParseFormat(GetOutputFormat())
	if err != nil {
		return err
	}

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

	for reader.Next() {
		event := reader.Event()
		switch event.Type {
		case "content_delta":
			fullText.WriteString(event.Text)
		case "done":
			usage = event.Usage
		}
	}
	output.StopSpinner(spin)

	if err := reader.Err(); err != nil {
		return handleAIError(err)
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

	return err
}
