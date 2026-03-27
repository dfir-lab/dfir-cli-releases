package output

import (
	"strings"

	"github.com/fatih/color"
)

// NoColor can be set to true to disable all colored output.
var NoColor bool

// Pre-configured color printers.
var (
	// Red is used for errors and malicious verdicts.
	Red = color.New(color.FgRed)

	// Yellow is used for warnings and suspicious verdicts.
	Yellow = color.New(color.FgYellow)

	// Green is used for success and clean verdicts.
	Green = color.New(color.FgGreen)

	// Cyan is used for info and highlights.
	Cyan = color.New(color.FgCyan)

	// Bold is used for emphasis.
	Bold = color.New(color.Bold)

	// Dim is used for secondary/muted text.
	Dim = color.New(color.Faint)

	// Accent is an emerald/green color used for branding.
	Accent = color.New(color.FgGreen)
)

// SetNoColor enables or disables colored output globally.
func SetNoColor(disable bool) {
	NoColor = disable
	color.NoColor = disable
}

// VerdictColor returns the appropriate color for a verdict string.
// Recognised verdicts: malicious, suspicious, clean, unknown.
func VerdictColor(verdict string) *color.Color {
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case "malicious":
		return Red
	case "suspicious":
		return Yellow
	case "clean":
		return Green
	case "unknown":
		return Dim
	default:
		return Dim
	}
}
