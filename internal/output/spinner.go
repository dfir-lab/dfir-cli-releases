package output

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// NewSpinner creates a pre-configured spinner that writes to stderr so it does
// not interfere with piped stdout output. It uses a clean dot-style character
// set (CharSet 14) with a 100ms refresh interval and cyan color.
func NewSpinner(suffix string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " " + suffix
	s.Color("cyan") //nolint:errcheck
	return s
}

// StartSpinner starts the spinner only when stdout is a TTY and colors are
// enabled. In non-interactive or no-color mode the spinner is silently skipped
// so automated pipelines stay clean.
func StartSpinner(s *spinner.Spinner) {
	if s == nil {
		return
	}
	if IsTerminal() && !NoColor {
		s.Start()
	}
}

// StopSpinner stops and clears the spinner. It is safe to call on a nil or
// already-stopped spinner.
func StopSpinner(s *spinner.Spinner) {
	if s == nil {
		return
	}
	s.Stop()
}
