package output

import (
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
		wantFill int // expected filled blocks (out of 20)
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
	}

	for _, tc := range tests {
		t.Run("risk_"+tc.level, func(t *testing.T) {
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
	}

	for _, tc := range tests {
		t.Run("severity_"+tc.severity, func(t *testing.T) {
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
	}

	for _, tc := range tests {
		t.Run("auth_"+tc.result, func(t *testing.T) {
			got := stripANSI(AuthBadge(tc.result))
			if got != tc.want {
				t.Errorf("AuthBadge(%q) = %q; want %q", tc.result, got, tc.want)
			}
		})
	}
}
