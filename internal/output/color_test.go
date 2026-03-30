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

func TestVerdictColor_AllVerdicts(t *testing.T) {
	// Ensure all known verdicts return distinct, correct colors.
	malicious := VerdictColor("malicious")
	suspicious := VerdictColor("suspicious")
	clean := VerdictColor("clean")
	unknown := VerdictColor("unknown")

	if malicious != Red {
		t.Error("malicious should map to Red")
	}
	if suspicious != Yellow {
		t.Error("suspicious should map to Yellow")
	}
	if clean != Green {
		t.Error("clean should map to Green")
	}
	if unknown != Dim {
		t.Error("unknown should map to Dim")
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

	// Restore for other tests.
	SetNoColor(true)
}

func TestSetNoColor_Toggle(t *testing.T) {
	// Toggle multiple times to verify consistency.
	for _, v := range []bool{true, false, true, false} {
		SetNoColor(v)
		if NoColor != v {
			t.Errorf("after SetNoColor(%v), NoColor = %v", v, NoColor)
		}
		if color.NoColor != v {
			t.Errorf("after SetNoColor(%v), color.NoColor = %v", v, color.NoColor)
		}
	}
	// Restore.
	SetNoColor(true)
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

func TestPreConfiguredColors_AreDistinct(t *testing.T) {
	// Red, Yellow, Green, Cyan should be distinct from each other.
	distinctColors := []*color.Color{Red, Yellow, Green, Cyan}
	for i := 0; i < len(distinctColors); i++ {
		for j := i + 1; j < len(distinctColors); j++ {
			if distinctColors[i] == distinctColors[j] {
				t.Errorf("color[%d] and color[%d] point to the same object", i, j)
			}
		}
	}
}
