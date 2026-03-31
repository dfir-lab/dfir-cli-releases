package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// newAIChatCmd builds and returns the "ai chat" subcommand that starts an
// interactive REPL session with the DFIR AI assistant.
func newAIChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive AI chat session",
		Long: `Start an interactive DFIR AI assistant chat session.

The assistant specializes in digital forensics and incident response.
Conversation history is maintained within the session.

Special commands:
  /help       Show available commands
  /clear      Clear conversation history
  /model NAME Switch model (haiku or sonnet)
  /exit       Exit the chat session
  Ctrl+C      Cancel current response

Requires a Starter, Professional, or Enterprise plan.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !output.IsTerminal() {
				return &ExitError{
					Code:    1,
					Message: "interactive chat requires a terminal — use 'dfir-cli ai \"question\"' for non-interactive mode",
				}
			}
			model, _ := cmd.Flags().GetString("model")
			if model == "" {
				model = resolveAIModel()
			}
			return runChatREPL(model)
		},
	}
	return cmd
}

// runChatREPL runs the interactive chat read-eval-print loop. It maintains
// conversation history in memory and streams responses from the AI API.
func runChatREPL(model string) error {
	c, err := newAPIClient()
	if err != nil {
		return err
	}

	// Print greeting banner.
	accent := color.New(color.FgGreen, color.Bold)
	dim := color.New(color.FgHiBlack)

	fmt.Println()
	accent.Println("  DFIR AI Assistant")
	dim.Println("  Digital Forensics & Incident Response")
	fmt.Println()
	dim.Printf("  Model: %s | Type /help for commands | /exit to quit\n", model)
	fmt.Println()

	// Conversation history kept in memory for the duration of the session.
	history := make([]client.AIChatMessage, 0, 32)

	scanner := bufio.NewScanner(os.Stdin)
	// Increase scanner buffer for long inputs (up to 1 MB).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		// Print prompt.
		accent.Print("  > ")

		if !scanner.Scan() {
			// EOF (Ctrl+D) or read error — exit gracefully.
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle special slash commands.
		if strings.HasPrefix(input, "/") {
			handled, shouldExit := handleChatCommand(input, &history, &model)
			if shouldExit {
				fmt.Println()
				dim.Println("  Goodbye!")
				fmt.Println()
				return nil
			}
			if handled {
				continue
			}
		}

		// Add user message to history.
		history = append(history, client.AIChatMessage{
			Role:    "user",
			Content: input,
		})

		// Cap history to prevent unbounded context growth.
		const maxHistoryTurns = 40
		if len(history) > maxHistoryTurns {
			history = history[len(history)-maxHistoryTurns:]
		}

		// Build request with conversation history.
		req := &client.AIChatRequest{
			Messages: history,
			Model:    model,
			Stream:   true,
		}

		// Create a per-request context so Ctrl+C cancels only the current
		// streaming response, not the entire REPL session.
		ctx, cancel := signalContext()

		reader, err := c.AIChatStream(ctx, req)
		if err != nil {
			cancel()
			fmt.Fprintln(os.Stderr)
			handleAIError(err)
			// Remove the failed user message from history.
			history = history[:len(history)-1]
			continue
		}

		// Stream the response to stdout.
		fmt.Println()
		var fullText strings.Builder

		for reader.Next() {
			event := reader.Event()
			switch event.Type {
			case "content_delta":
				fmt.Print(event.Text)
				fullText.WriteString(event.Text)
			case "done":
				fmt.Println()
				if event.Usage != nil {
					fmt.Println()
					dim.Printf("  [%s | %d input + %d output tokens | %d credits]\n",
						event.Usage.Model,
						event.Usage.InputTokens,
						event.Usage.OutputTokens,
						event.Usage.CreditsUsed,
					)
				}
				if event.Meta != nil {
					dim.Printf("  [%d credits remaining]\n", event.Meta.CreditsRemaining)
				}
			}
		}
		fmt.Println()

		reader.Close()
		cancel()

		if reader.Err() != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n\n", reader.Err())
			// Remove the failed user message so it is not sent again.
			history = history[:len(history)-1]
			continue
		}

		// Add assistant response to history for multi-turn context.
		if fullText.Len() > 0 {
			history = append(history, client.AIChatMessage{
				Role:    "assistant",
				Content: fullText.String(),
			})
		}
	}

	return nil
}

// handleChatCommand processes slash commands inside the REPL. It returns two
// booleans: handled (the input was a command) and shouldExit (the user wants
// to leave the REPL).
func handleChatCommand(input string, history *[]client.AIChatMessage, model *string) (handled bool, shouldExit bool) {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	dim := color.New(color.FgHiBlack)

	switch cmd {
	case "/exit", "/quit", "/q":
		return true, true

	case "/clear":
		*history = (*history)[:0]
		dim.Println("  Conversation cleared.")
		fmt.Println()
		return true, false

	case "/model":
		if len(parts) < 2 {
			dim.Printf("  Current model: %s\n", *model)
			dim.Println("  Usage: /model <haiku|sonnet>")
			fmt.Println()
			return true, false
		}
		newModel := strings.ToLower(parts[1])
		if newModel != "haiku" && newModel != "sonnet" {
			fmt.Fprintln(os.Stderr, "  Error: model must be 'haiku' or 'sonnet'")
			fmt.Println()
			return true, false
		}
		*model = newModel
		dim.Printf("  Switched to model: %s\n", newModel)
		fmt.Println()
		return true, false

	case "/help", "/?":
		fmt.Println()
		dim.Println("  Commands:")
		dim.Println("    /help       Show this help")
		dim.Println("    /clear      Clear conversation history")
		dim.Println("    /model NAME Switch model (haiku or sonnet)")
		dim.Println("    /exit       Exit the chat")
		dim.Println("    Ctrl+C      Cancel current response")
		fmt.Println()
		return true, false

	default:
		fmt.Fprintf(os.Stderr, "  Unknown command: %s (type /help)\n\n", cmd)
		return true, false
	}
}
