package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// Table Builder
// ---------------------------------------------------------------------------

// NewTable creates a pre-styled table writer that respects NoColor settings.
// Writes to os.Stdout.
func NewTable() table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Use a clean, minimal style.
	style := table.StyleLight
	if NoColor {
		// Use ASCII borders when color is disabled.
		style = table.StyleDefault
	}
	t.SetStyle(style)

	// Header styling.
	t.Style().Format.HeaderAlign = text.AlignLeft
	t.Style().Format.Header = text.FormatUpper

	// Respect terminal width so tables do not overflow.
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		t.SetAllowedRowLength(w)
	}

	return t
}

// ---------------------------------------------------------------------------
// Score Bar
// ---------------------------------------------------------------------------

// ScoreBar renders a visual score bar like: [████████████░░░░░░░░] 75/100
// When colors are enabled the filled portion is colored based on the score:
//
//	0-30  green
//	31-60 yellow
//	61+   red
func ScoreBar(score, max int) string {
	if max <= 0 {
		max = 100
	}
	if score < 0 {
		score = 0
	}

	const width = 20
	filled := (score * width) / max
	if filled > width {
		filled = width
	}
	empty := width - filled

	filledStr := strings.Repeat("\u2588", filled)
	emptyStr := strings.Repeat("\u2591", empty)

	if !NoColor {
		var c *color.Color
		switch {
		case score <= 30:
			c = Green
		case score <= 60:
			c = Yellow
		default:
			c = Red
		}
		filledStr = c.Sprint(filledStr)
	}

	return fmt.Sprintf("[%s%s] %d/%d", filledStr, emptyStr, score, max)
}

// ---------------------------------------------------------------------------
// Verdict Badge
// ---------------------------------------------------------------------------

// VerdictBadge returns a colored verdict string with appropriate formatting.
// E.g., "MALICIOUS" in bold red, "CLEAN" in green, etc.
func VerdictBadge(verdict string) string {
	upper := strings.ToUpper(verdict)
	c := VerdictColor(verdict)
	return c.Sprint(upper)
}

// ---------------------------------------------------------------------------
// Risk Level Badge
// ---------------------------------------------------------------------------

// RiskBadge returns a colored risk level string.
// Maps: critical -> red, high -> red, medium -> yellow, low -> green.
func RiskBadge(level string) string {
	upper := strings.ToUpper(level)
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical", "high":
		return Red.Sprint(upper)
	case "medium":
		return Yellow.Sprint(upper)
	case "low":
		return Green.Sprint(upper)
	default:
		return Dim.Sprint(upper)
	}
}

// ---------------------------------------------------------------------------
// Severity Badge
// ---------------------------------------------------------------------------

// SeverityBadge returns a colored severity string.
// Maps: high -> red, medium -> yellow, low -> green/dim.
func SeverityBadge(severity string) string {
	upper := strings.ToUpper(severity)
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return Red.Sprint(upper)
	case "medium":
		return Yellow.Sprint(upper)
	case "low":
		return Green.Sprint(upper)
	default:
		return Dim.Sprint(upper)
	}
}

// ---------------------------------------------------------------------------
// Auth Result Badge
// ---------------------------------------------------------------------------

// AuthBadge returns a colored authentication result.
// "pass" -> green, "fail" -> red, everything else -> yellow.
func AuthBadge(result string) string {
	upper := strings.ToUpper(result)
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "pass":
		return Green.Sprint(upper)
	case "fail":
		return Red.Sprint(upper)
	default:
		return Yellow.Sprint(upper)
	}
}

// ---------------------------------------------------------------------------
// Credits Footer
// ---------------------------------------------------------------------------

// PrintCreditsFooter prints the credits info line at the bottom of command output.
// E.g., "  Credits: 3 used, 97 remaining"
func PrintCreditsFooter(used, remaining int) {
	label := Dim.Sprint("Credits:")
	value := fmt.Sprintf("%d used, %d remaining", used, remaining)
	fmt.Fprintf(os.Stdout, "  %s %s\n", label, value)
}

// ---------------------------------------------------------------------------
// Section Header
// ---------------------------------------------------------------------------

// PrintHeader prints a section header with consistent styling.
// E.g., "IOC Enrichment: 1.2.3.4 (IPv4)"
func PrintHeader(title string) {
	fmt.Fprintln(os.Stdout)
	if NoColor {
		fmt.Fprintln(os.Stdout, title)
	} else {
		Bold.Fprintln(os.Stdout, title)
	}
	fmt.Fprintln(os.Stdout)
}

// ---------------------------------------------------------------------------
// Key-Value Display
// ---------------------------------------------------------------------------

// PrintKeyValue prints an aligned key-value pair.
// E.g., "  Verdict:    MALICIOUS"
func PrintKeyValue(key, value string) {
	label := fmt.Sprintf("%-14s", key+":")
	if !NoColor {
		label = Dim.Sprint(label)
	}
	fmt.Fprintf(os.Stdout, "  %s %s\n", label, value)
}

// PrintKeyValueColored prints a key-value pair where the value is colored.
func PrintKeyValueColored(key string, value string, c *color.Color) {
	label := fmt.Sprintf("%-14s", key+":")
	if !NoColor {
		label = Dim.Sprint(label)
	}
	coloredValue := value
	if c != nil {
		coloredValue = c.Sprint(value)
	}
	fmt.Fprintf(os.Stdout, "  %s %s\n", label, coloredValue)
}

// ---------------------------------------------------------------------------
// Error Display
// ---------------------------------------------------------------------------

// PrintError prints a formatted error message to stderr.
// Format: "Error: <message>"
func PrintError(msg string) {
	label := "Error:"
	if !NoColor {
		label = Red.Sprint(label)
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", label, msg)
}

// PrintErrorWithHint prints an error with a hint for resolution.
func PrintErrorWithHint(msg, hint string) {
	PrintError(msg)
	hintLabel := "Hint:"
	if !NoColor {
		hintLabel = Yellow.Sprint(hintLabel)
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", hintLabel, hint)
}
