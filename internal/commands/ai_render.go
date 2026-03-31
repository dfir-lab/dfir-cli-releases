package commands

import (
	"os"

	"github.com/charmbracelet/glamour"
)

// renderMarkdown renders markdown text for terminal display using glamour.
// Falls back to raw text if rendering fails or colors are disabled.
func renderMarkdown(text string) string {
	// If no-color mode, return raw text
	if os.Getenv("NO_COLOR") != "" {
		return text
	}

	style := "dark"
	if os.Getenv("GLAMOUR_STYLE") != "" {
		style = os.Getenv("GLAMOUR_STYLE")
	}

	rendered, err := glamour.Render(text, style)
	if err != nil {
		return text
	}
	return rendered
}
