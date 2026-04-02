package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/version"
)

var (
	interruptMessageWriter = io.Writer(os.Stderr)
	forceExitFn            = func(code int) { os.Exit(code) }
)

// newAPIClient creates an authenticated API client using the resolved
// configuration (flag > env > config file). Returns an error if no API key
// is configured.
func newAPIClient() (*client.Client, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured. Run: dfir-cli config init")
	}
	return client.New(apiKey, GetAPIURL(), version.UserAgent(), GetTimeout(), IsVerbose()), nil
}

// newAIClient creates an authenticated API client for AI chat requests. AI is
// currently served from a different production host than the rest of the API.
func newAIClient() (*client.Client, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured. Run: dfir-cli config init")
	}
	return client.New(apiKey, GetAIAPIURL(), version.UserAgent(), GetTimeout(), IsVerbose()), nil
}

// signalContext returns a context that is cancelled when the user presses
// Ctrl+C (SIGINT). First Ctrl+C requests graceful cancellation; a second
// Ctrl+C force-quits immediately.
func signalContext() (context.Context, context.CancelFunc) {
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signalContextFromChannel(sigCh)
	var once sync.Once

	return ctx, func() {
		once.Do(func() {
			signal.Stop(sigCh)
			cancel()
		})
	}
}

// signalContextFromChannel is a test seam that allows injecting a synthetic
// signal source.
func signalContextFromChannel(sigCh <-chan os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(doneCh)
		interruptCount := 0
		for {
			select {
			case <-stopCh:
				return
			case <-sigCh:
				interruptCount++
				if interruptCount == 1 {
					cancel()
					fmt.Fprintln(interruptMessageWriter, "Finishing current operation... Press Ctrl+C again to force quit.")
					continue
				}

				fmt.Fprintln(interruptMessageWriter, "Force quitting.")
				forceExitFn(130)
				return
			}
		}
	}()

	return ctx, func() {
		once.Do(func() {
			close(stopCh)
			cancel()
			<-doneCh
		})
	}
}

// exitCodeForVerdict returns the appropriate exit code based on a verdict string.
// malicious/highly_malicious → 2, suspicious → 3, clean/safe/unknown → 0
func exitCodeForVerdict(verdict string) int {
	switch strings.ToLower(verdict) {
	case "malicious", "highly_malicious":
		return 2
	case "suspicious":
		return 3
	default:
		return 0
	}
}

// exitCodeForRisk returns the appropriate exit code based on a risk level string.
// critical/high → 2, medium → 3, low/none → 0
func exitCodeForRisk(level string) int {
	switch strings.ToLower(level) {
	case "critical", "high":
		return 2
	case "medium":
		return 3
	default:
		return 0
	}
}

// maxStdinSize is the maximum bytes we will read from stdin to prevent
// accidental memory exhaustion when large files are piped in.
const maxStdinSize = 10 * 1024 * 1024 // 10 MB

// readStdin reads all of stdin if it's being piped (not a terminal).
// Returns nil if stdin is a terminal (interactive mode).
// Reads up to maxStdinSize bytes to prevent memory exhaustion.
func readStdin() ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil // stdin is a terminal, not piped
	}
	return io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
}

// readLines reads a file (or stdin if path is "-") and returns non-empty,
// non-comment lines. Lines starting with # are treated as comments.
func readLines(path string) ([]string, error) {
	var data []byte
	var err error

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

// titleCase converts the first character of s to uppercase and the rest to
// lowercase. This replaces the deprecated strings.Title for simple ASCII
// strings (service names, field names, etc.).
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
