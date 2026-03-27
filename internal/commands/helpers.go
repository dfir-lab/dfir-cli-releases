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

	"github.com/ForeGuards/dfir-cli/internal/client"
	"github.com/ForeGuards/dfir-cli/internal/version"
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

// signalContext returns a context that is cancelled when the user presses
// Ctrl+C (SIGINT) or receives SIGTERM. The cancel function should be deferred.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	return ctx, cancel
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
