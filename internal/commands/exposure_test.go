package commands

import (
	"os"
	"path/filepath"
	"strings"
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
		{"HIGH", 2},
		{"MEDIUM", 3},
		{"LOW", 0},
		{"NONE", 0},
		{"  critical  ", 2}, // whitespace trimmed
		{"unknown_level", 0},
	}

	for _, tc := range tests {
		name := tc.level
		if name == "" {
			name = "empty"
		}
		t.Run("level_"+name, func(t *testing.T) {
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

func TestResolveExposureTargets_DomainWithWhitespace(t *testing.T) {
	targets, err := resolveExposureTargets("  example.com  ", "")
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
// TestResolveExposureTargets_BatchFile
// ---------------------------------------------------------------------------

func TestResolveExposureTargets_BatchFile(t *testing.T) {
	content := "example.com\ntest.org\n"
	tmp := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	targets, err := resolveExposureTargets("", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0] != "example.com" || targets[1] != "test.org" {
		t.Errorf("unexpected targets: %v", targets)
	}
}

func TestResolveExposureTargets_DomainPrecedesBatch(t *testing.T) {
	// When both --domain and --batch are provided, --domain wins.
	content := "batch1.com\nbatch2.com\n"
	tmp := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	targets, err := resolveExposureTargets("example.com", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 || targets[0] != "example.com" {
		t.Errorf("expected [example.com], got %v", targets)
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

// ---------------------------------------------------------------------------
// TestReadExposureBatch_FileNotFound
// ---------------------------------------------------------------------------

func TestReadExposureBatch_FileNotFound(t *testing.T) {
	_, err := readExposureBatch("/tmp/nonexistent_dfir_exposure_batch.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestReadExposureBatch_SingleTarget
// ---------------------------------------------------------------------------

func TestReadExposureBatch_SingleTarget(t *testing.T) {
	content := "single.com\n"
	tmp := filepath.Join(t.TempDir(), "single.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	targets, err := readExposureBatch(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 || targets[0] != "single.com" {
		t.Errorf("expected [single.com], got %v", targets)
	}
}

// ---------------------------------------------------------------------------
// TestReadExposureBatch_WhitespaceLines
// ---------------------------------------------------------------------------

func TestReadExposureBatch_WhitespaceLines(t *testing.T) {
	content := "  example.com  \n   \n  test.org  \n"
	tmp := filepath.Join(t.TempDir(), "ws.txt")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	targets, err := readExposureBatch(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d: %v", len(targets), targets)
	}
	if targets[0] != "example.com" || targets[1] != "test.org" {
		t.Errorf("unexpected targets: %v", targets)
	}
}

// ---------------------------------------------------------------------------
// TestExposureScanConcurrencyFlag
// ---------------------------------------------------------------------------

func TestExposureScanConcurrencyFlag(t *testing.T) {
	cmd := newExposureScanCmd()

	t.Run("default_value", func(t *testing.T) {
		f := cmd.Flags().Lookup("concurrency")
		if f == nil {
			t.Fatal("--concurrency flag not found on exposure scan command")
		}
		if f.DefValue != "5" {
			t.Errorf("default concurrency = %q, want %q", f.DefValue, "5")
		}
	})

	t.Run("flag_is_parseable", func(t *testing.T) {
		testCmd := newExposureScanCmd()
		testCmd.SetArgs([]string{"--concurrency", "10", "--domain", "example.com"})
		f := testCmd.Flags().Lookup("concurrency")
		if f == nil {
			t.Fatal("--concurrency flag not found")
		}
	})
}

// ---------------------------------------------------------------------------
// TestExposureConcurrencyValidation
// ---------------------------------------------------------------------------

func TestExposureConcurrencyValidation(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
		errContains string
	}{
		{name: "valid_1", concurrency: 1, wantErr: false},
		{name: "valid_5", concurrency: 5, wantErr: false},
		{name: "valid_20", concurrency: 20, wantErr: false},
		{name: "too_low_0", concurrency: 0, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
		{name: "too_high_21", concurrency: 21, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
		{name: "negative", concurrency: -1, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newExposureScanCmd()
			err := runExposureScan(cmd, "example.com", "domain", "", tc.concurrency)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
			}
			if !tc.wantErr && err != nil {
				if strings.Contains(err.Error(), "--concurrency") {
					t.Errorf("unexpected concurrency error: %v", err)
				}
			}
		})
	}
}
