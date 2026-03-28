package output

import (
	"testing"

	"github.com/fatih/color"
)

func TestVerdictColor(t *testing.T) {
	tests := []struct {
		verdict string
		want    *color.Color
	}{
		{"malicious", Red},
		{"MALICIOUS", Red},
		{" malicious ", Red},
		{"suspicious", Yellow},
		{"SUSPICIOUS", Yellow},
		{" suspicious ", Yellow},
		{"clean", Green},
		{"CLEAN", Green},
		{" clean ", Green},
		{"unknown", Dim},
		{"UNKNOWN", Dim},
		{" unknown ", Dim},
		{"other", Dim},
		{"", Dim},
	}

	for _, tt := range tests {
		t.Run(tt.verdict, func(t *testing.T) {
			got := VerdictColor(tt.verdict)
			if got != tt.want {
				t.Errorf("VerdictColor(%q) returned unexpected color", tt.verdict)
			}
		})
	}
}

func TestSetNoColor(t *testing.T) {
	// Enable no-color mode.
	SetNoColor(true)

	if !color.NoColor {
		t.Error("expected color.NoColor to be true after SetNoColor(true)")
	}
	if !NoColor {
		t.Error("expected output.NoColor to be true after SetNoColor(true)")
	}

	// Re-enable color mode.
	SetNoColor(false)

	if color.NoColor {
		t.Error("expected color.NoColor to be false after SetNoColor(false)")
	}
	if NoColor {
		t.Error("expected output.NoColor to be false after SetNoColor(false)")
	}
}

func TestPreConfiguredColors(t *testing.T) {
	colors := map[string]*color.Color{
		"Red":    Red,
		"Yellow": Yellow,
		"Green":  Green,
		"Cyan":   Cyan,
		"Bold":   Bold,
		"Dim":    Dim,
		"Accent": Accent,
	}

	for name, c := range colors {
		if c == nil {
			t.Errorf("pre-configured color %s is nil", name)
		}
	}
}
