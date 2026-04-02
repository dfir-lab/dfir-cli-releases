package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitForBufferSubstring(t *testing.T, buf *lockedBuffer, want string) string {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		got := buf.String()
		if strings.Contains(got, want) {
			return got
		}
		if time.Now().After(deadline) {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// TestExitCodeForVerdict
// ---------------------------------------------------------------------------

func TestExitCodeForVerdict(t *testing.T) {
	tests := []struct {
		verdict string
		want    int
	}{
		{"malicious", 2},
		{"highly_malicious", 2},
		{"suspicious", 3},
		{"clean", 0},
		{"safe", 0},
		{"unknown", 0},
		{"", 0},
		{"MALICIOUS", 2},
		{"SUSPICIOUS", 3},
		{"Malicious", 2},
		{"Suspicious", 3},
		{"CLEAN", 0},
		{"other", 0},
	}

	for _, tt := range tests {
		name := tt.verdict
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if got := exitCodeForVerdict(tt.verdict); got != tt.want {
				t.Errorf("exitCodeForVerdict(%q) = %d, want %d", tt.verdict, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExitCodeForRisk
// ---------------------------------------------------------------------------

func TestExitCodeForRisk(t *testing.T) {
	tests := []struct {
		level string
		want  int
	}{
		{"critical", 2},
		{"high", 2},
		{"medium", 3},
		{"low", 0},
		{"none", 0},
		{"", 0},
		{"CRITICAL", 2},
		{"HIGH", 2},
		{"MEDIUM", 3},
		{"LOW", 0},
		{"other", 0},
	}

	for _, tt := range tests {
		name := tt.level
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if got := exitCodeForRisk(tt.level); got != tt.want {
				t.Errorf("exitCodeForRisk(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestReadLines
// ---------------------------------------------------------------------------

func TestReadLines(t *testing.T) {
	content := `# this is a comment
alpha
   # indented comment

bravo

# another comment
charlie
`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines(%q) returned error: %v", path, err)
	}

	want := []string{"alpha", "bravo", "charlie"}
	if len(lines) != len(want) {
		t.Fatalf("readLines returned %d lines, want %d: %v", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestReadLines_FileNotFound(t *testing.T) {
	_, err := readLines("/tmp/nonexistent_dfir_test_file.txt")
	if err == nil {
		t.Fatal("readLines on missing file should return an error")
	}
}

func TestReadLines_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines returned error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty file, got %d", len(lines))
	}
}

func TestReadLines_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "comments.txt")
	content := "# comment 1\n# comment 2\n# comment 3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines returned error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for comments-only file, got %d", len(lines))
	}
}

func TestReadLines_WhitespaceLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ws.txt")
	content := "  alpha  \n   \n  bravo  \n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines returned error: %v", err)
	}
	want := []string{"alpha", "bravo"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestReadLines_NoTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "notl.txt")
	content := "alpha\nbravo"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines returned error: %v", err)
	}
	want := []string{"alpha", "bravo"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
}

// ---------------------------------------------------------------------------
// TestSignalContext
// ---------------------------------------------------------------------------

func TestSignalContextFromChannel_FirstInterruptCancels(t *testing.T) {
	var out lockedBuffer
	exitCalls := make(chan int, 1)

	origWriter := interruptMessageWriter
	origForceExit := forceExitFn
	interruptMessageWriter = &out
	forceExitFn = func(code int) { exitCalls <- code }
	t.Cleanup(func() {
		interruptMessageWriter = origWriter
		forceExitFn = origForceExit
	})

	sigCh := make(chan os.Signal, 2)
	ctx, cancel := signalContextFromChannel(sigCh)
	defer cancel()

	sigCh <- os.Interrupt

	select {
	case <-ctx.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("context was not cancelled after first Ctrl+C")
	}

	select {
	case code := <-exitCalls:
		t.Fatalf("force exit unexpectedly called with code %d", code)
	default:
	}

	msg := waitForBufferSubstring(t, &out, "Finishing current operation... Press Ctrl+C again to force quit.")
	if !strings.Contains(msg, "Finishing current operation... Press Ctrl+C again to force quit.") {
		t.Fatalf("missing first Ctrl+C guidance message: %q", msg)
	}
}

func TestSignalContextFromChannel_SecondInterruptForcesExit(t *testing.T) {
	var out lockedBuffer
	exitCalls := make(chan int, 1)

	origWriter := interruptMessageWriter
	origForceExit := forceExitFn
	interruptMessageWriter = &out
	forceExitFn = func(code int) { exitCalls <- code }
	t.Cleanup(func() {
		interruptMessageWriter = origWriter
		forceExitFn = origForceExit
	})

	sigCh := make(chan os.Signal, 2)
	ctx, cancel := signalContextFromChannel(sigCh)
	defer cancel()

	sigCh <- os.Interrupt
	<-ctx.Done()
	sigCh <- os.Interrupt

	select {
	case code := <-exitCalls:
		if code != 130 {
			t.Fatalf("force exit code = %d, want 130", code)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second Ctrl+C did not trigger force exit")
	}

	msg := waitForBufferSubstring(t, &out, "Force quitting.")
	if !strings.Contains(msg, "Force quitting.") {
		t.Fatalf("missing force quit message: %q", msg)
	}
}
