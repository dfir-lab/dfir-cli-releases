package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// TestExposureExitCode
// ---------------------------------------------------------------------------

func TestExposureExitCode(t *testing.T) {
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
		{"CRITICAL", 2}, // case insensitive
	}

	for _, tc := range tests {
		t.Run("level_"+tc.level, func(t *testing.T) {
			got := exposureExitCode(tc.level)
			if got != tc.want {
				t.Errorf("exposureExitCode(%q) = %d, want %d", tc.level, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveExposureTargets_Domain
// ---------------------------------------------------------------------------

func TestResolveExposureTargets_Domain(t *testing.T) {
	targets, err := resolveExposureTargets("example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0] != "example.com" {
		t.Errorf("expected target %q, got %q", "example.com", targets[0])
	}
}

// ---------------------------------------------------------------------------
// TestResolveExposureTargets_NoInput
// ---------------------------------------------------------------------------

func TestResolveExposureTargets_NoInput(t *testing.T) {
	// When no domain, no batch, and stdin is a terminal (not piped),
	// resolveExposureTargets should return nil with no error.
	targets, err := resolveExposureTargets("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

// ---------------------------------------------------------------------------
// TestReadExposureBatch
// ---------------------------------------------------------------------------

func TestReadExposureBatch(t *testing.T) {
	content := `# This is a comment
example.com

# Another comment
test.org
foo.io
`
	tmp := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	targets, err := readExposureBatch(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"example.com", "test.org", "foo.io"}
	if len(targets) != len(want) {
		t.Fatalf("expected %d targets, got %d", len(want), len(targets))
	}
	for i, w := range want {
		if targets[i] != w {
			t.Errorf("target[%d] = %q, want %q", i, targets[i], w)
		}
	}
}

// ---------------------------------------------------------------------------
// TestReadExposureBatch_Empty
// ---------------------------------------------------------------------------

func TestReadExposureBatch_Empty(t *testing.T) {
	content := `# only comments
# nothing else
`
	tmp := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := readExposureBatch(tmp)
	if err == nil {
		t.Fatal("expected error for empty batch, got nil")
	}
	if got := err.Error(); got != "batch input contained no targets" {
		t.Errorf("unexpected error message: %q", got)
	}
}
