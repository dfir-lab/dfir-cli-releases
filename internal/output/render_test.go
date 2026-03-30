package output

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// stripANSI removes ANSI escape sequences from a string for test comparison.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func init() {
	// Disable color globally so output is predictable (no ANSI codes).
	color.NoColor = true
	NoColor = true
}

// captureStdoutRender redirects os.Stdout to a pipe, runs fn, and returns
// everything that was written to stdout during the call.
func captureStdoutRender(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// captureStderrRender redirects os.Stderr to a pipe, runs fn, and returns
// everything that was written to stderr during the call.
func captureStderrRender(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// ---------------------------------------------------------------------------
// TestNewTable
// ---------------------------------------------------------------------------

func TestNewTable(t *testing.T) {
	tw := NewTable()
	if tw == nil {
		t.Fatal("NewTable returned nil")
	}
}

// ---------------------------------------------------------------------------
// TestScoreBar
// ---------------------------------------------------------------------------

func TestScoreBar(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		max      int
		wantFill int    // expected filled blocks (out of 20)
		wantTail string // expected "score/max" suffix
	}{
		{
			name:     "zero score empty bar",
			score:    0,
			max:      100,
			wantFill: 0,
			wantTail: "0/100",
		},
		{
			name:     "half score",
			score:    50,
			max:      100,
			wantFill: 10,
			wantTail: "50/100",
		},
		{
			name:     "full score",
			score:    100,
			max:      100,
			wantFill: 20,
			wantTail: "100/100",
		},
		{
			name:     "negative score clamped to zero",
			score:    -10,
			max:      100,
			wantFill: 0,
			wantTail: "0/100",
		},
		{
			name:     "score exceeds max clamped",
			score:    200,
			max:      100,
			wantFill: 20, // clamped to width
			wantTail: "200/100",
		},
		{
			name:     "custom max 50",
			score:    25,
			max:      50,
			wantFill: 10,
			wantTail: "25/50",
		},
		{
			name:     "zero max defaults to 100",
			score:    50,
			max:      0,
			wantFill: 10,
			wantTail: "50/100",
		},
		{
			name:     "negative max defaults to 100",
			score:    50,
			max:      -10,
			wantFill: 10,
			wantTail: "50/100",
		},
		{
			name:     "max score",
			score:    100,
			max:      100,
			wantFill: 20,
			wantTail: "100/100",
		},
		{
			name:     "score_1",
			score:    1,
			max:      100,
			wantFill: 0, // 1*20/100 = 0
			wantTail: "1/100",
		},
		{
			name:     "score_5",
			score:    5,
			max:      100,
			wantFill: 1,
			wantTail: "5/100",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreBar(tc.score, tc.max)

			// Verify suffix.
			if !strings.HasSuffix(got, tc.wantTail) {
				t.Errorf("ScoreBar(%d, %d) = %q; want suffix %q", tc.score, tc.max, got, tc.wantTail)
			}

			// Verify bar brackets.
			if !strings.HasPrefix(got, "[") {
				t.Errorf("ScoreBar(%d, %d) should start with '['; got %q", tc.score, tc.max, got)
			}

			// Count filled blocks (█ U+2588) and empty blocks (░ U+2591).
			filledCount := strings.Count(got, "\u2588")
			emptyCount := strings.Count(got, "\u2591")

			if filledCount != tc.wantFill {
				t.Errorf("ScoreBar(%d, %d): filled blocks = %d; want %d", tc.score, tc.max, filledCount, tc.wantFill)
			}

			wantEmpty := 20 - tc.wantFill
			if emptyCount != wantEmpty {
				t.Errorf("ScoreBar(%d, %d): empty blocks = %d; want %d", tc.score, tc.max, emptyCount, wantEmpty)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestVerdictBadge
// ---------------------------------------------------------------------------

func TestVerdictBadge(t *testing.T) {
	tests := []struct {
		verdict string
		want    string
	}{
		{"malicious", "MALICIOUS"},
		{"suspicious", "SUSPICIOUS"},
		{"clean", "CLEAN"},
		{"unknown", "UNKNOWN"},
		{"", ""},
		{"Safe", "SAFE"},
	}

	for _, tc := range tests {
		t.Run("verdict_"+tc.verdict, func(t *testing.T) {
			got := stripANSI(VerdictBadge(tc.verdict))
			if got != tc.want {
				t.Errorf("VerdictBadge(%q) = %q; want %q", tc.verdict, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRiskBadge
// ---------------------------------------------------------------------------

func TestRiskBadge(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"critical", "CRITICAL"},
		{"high", "HIGH"},
		{"medium", "MEDIUM"},
		{"low", "LOW"},
		{"none", "NONE"},
		{"", ""},
		{"CRITICAL", "CRITICAL"},
		{"other", "OTHER"},
	}

	for _, tc := range tests {
		name := tc.level
		if name == "" {
			name = "empty"
		}
		t.Run("risk_"+name, func(t *testing.T) {
			got := stripANSI(RiskBadge(tc.level))
			if got != tc.want {
				t.Errorf("RiskBadge(%q) = %q; want %q", tc.level, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSeverityBadge
// ---------------------------------------------------------------------------

func TestSeverityBadge(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"high", "HIGH"},
		{"medium", "MEDIUM"},
		{"low", "LOW"},
		{"", ""},
		{"other", "OTHER"},
		{"HIGH", "HIGH"},
	}

	for _, tc := range tests {
		name := tc.severity
		if name == "" {
			name = "empty"
		}
		t.Run("severity_"+name, func(t *testing.T) {
			got := stripANSI(SeverityBadge(tc.severity))
			if got != tc.want {
				t.Errorf("SeverityBadge(%q) = %q; want %q", tc.severity, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthBadge
// ---------------------------------------------------------------------------

func TestAuthBadge(t *testing.T) {
	tests := []struct {
		result string
		want   string
	}{
		{"pass", "PASS"},
		{"fail", "FAIL"},
		{"none", "NONE"},
		{"softfail", "SOFTFAIL"},
		{"", ""},
		{"PASS", "PASS"},
	}

	for _, tc := range tests {
		name := tc.result
		if name == "" {
			name = "empty"
		}
		t.Run("auth_"+name, func(t *testing.T) {
			got := stripANSI(AuthBadge(tc.result))
			if got != tc.want {
				t.Errorf("AuthBadge(%q) = %q; want %q", tc.result, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPrintHeader
// ---------------------------------------------------------------------------

func TestPrintHeader(t *testing.T) {
	t.Run("no_color_mode", func(t *testing.T) {
		// NoColor is already set to true in init().
		out := captureStdoutRender(t, func() {
			PrintHeader("Test Header")
		})
		if !strings.Contains(out, "Test Header") {
			t.Errorf("expected output to contain 'Test Header', got %q", out)
		}
	})

	t.Run("with_color_mode", func(t *testing.T) {
		oldNoColor := NoColor
		NoColor = false
		defer func() { NoColor = oldNoColor }()

		out := captureStdoutRender(t, func() {
			PrintHeader("Colored Header")
		})
		// Strip ANSI to verify content.
		stripped := stripANSI(out)
		if !strings.Contains(stripped, "Colored Header") {
			t.Errorf("expected output to contain 'Colored Header', got %q", stripped)
		}
	})
}

// ---------------------------------------------------------------------------
// TestPrintKeyValue
// ---------------------------------------------------------------------------

func TestPrintKeyValue(t *testing.T) {
	out := captureStdoutRender(t, func() {
		PrintKeyValue("Status", "active")
	})
	if !strings.Contains(out, "Status:") {
		t.Errorf("expected output to contain 'Status:', got %q", out)
	}
	if !strings.Contains(out, "active") {
		t.Errorf("expected output to contain 'active', got %q", out)
	}
}

func TestPrintKeyValueColored(t *testing.T) {
	out := captureStdoutRender(t, func() {
		PrintKeyValueColored("Verdict", "MALICIOUS", Red)
	})
	stripped := stripANSI(out)
	if !strings.Contains(stripped, "Verdict:") {
		t.Errorf("expected output to contain 'Verdict:', got %q", stripped)
	}
	if !strings.Contains(stripped, "MALICIOUS") {
		t.Errorf("expected output to contain 'MALICIOUS', got %q", stripped)
	}
}

func TestPrintKeyValueColored_NilColor(t *testing.T) {
	out := captureStdoutRender(t, func() {
		PrintKeyValueColored("Key", "value", nil)
	})
	if !strings.Contains(out, "value") {
		t.Errorf("expected output to contain 'value', got %q", out)
	}
}

// ---------------------------------------------------------------------------
// TestPrintError
// ---------------------------------------------------------------------------

func TestPrintError(t *testing.T) {
	out := captureStderrRender(t, func() {
		PrintError("something failed")
	})
	if !strings.Contains(out, "Error:") {
		t.Errorf("expected output to contain 'Error:', got %q", out)
	}
	if !strings.Contains(out, "something failed") {
		t.Errorf("expected output to contain 'something failed', got %q", out)
	}
}

// ---------------------------------------------------------------------------
// TestPrintErrorWithHint
// ---------------------------------------------------------------------------

func TestPrintErrorWithHint(t *testing.T) {
	out := captureStderrRender(t, func() {
		PrintErrorWithHint("auth failed", "run dfir-cli config init")
	})
	if !strings.Contains(out, "Error:") {
		t.Errorf("expected output to contain 'Error:', got %q", out)
	}
	if !strings.Contains(out, "auth failed") {
		t.Errorf("expected output to contain 'auth failed', got %q", out)
	}
	if !strings.Contains(out, "Hint:") {
		t.Errorf("expected output to contain 'Hint:', got %q", out)
	}
	if !strings.Contains(out, "run dfir-cli config init") {
		t.Errorf("expected output to contain hint text, got %q", out)
	}
}

// ---------------------------------------------------------------------------
// TestPrintCreditsFooter
// ---------------------------------------------------------------------------

func TestPrintCreditsFooter(t *testing.T) {
	out := captureStdoutRender(t, func() {
		PrintCreditsFooter(3, 97)
	})
	if !strings.Contains(out, "Credits:") {
		t.Errorf("expected output to contain 'Credits:', got %q", out)
	}
	if !strings.Contains(out, "3 used") {
		t.Errorf("expected output to contain '3 used', got %q", out)
	}
	if !strings.Contains(out, "97 remaining") {
		t.Errorf("expected output to contain '97 remaining', got %q", out)
	}
}
