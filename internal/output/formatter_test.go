package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns
// everything that was written to stdout during the call.
func captureStdout(t *testing.T, fn func()) string {
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

// ---------------------------------------------------------------------------
// ParseFormat
// ---------------------------------------------------------------------------

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"table", FormatTable, false},
		{"json", FormatJSON, false},
		{"jsonl", FormatJSONL, false},
		{"csv", FormatCSV, false},
		{"xml", "", true},
		{"", "", true},
		{"yaml", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseFormat(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormat(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseFormat_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"JSON", FormatJSON},
		{"Json", FormatJSON},
		{"TABLE", FormatTable},
		{"Table", FormatTable},
		{"JSONL", FormatJSONL},
		{"Csv", FormatCSV},
		{"  json  ", FormatJSON}, // leading/trailing whitespace
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseFormat(tc.input)
			if err != nil {
				t.Fatalf("ParseFormat(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PrintJSON
// ---------------------------------------------------------------------------

func TestPrintJSON(t *testing.T) {
	sample := map[string]interface{}{
		"name":  "test",
		"count": float64(42),
	}

	out := captureStdout(t, func() {
		if err := PrintJSON(sample); err != nil {
			t.Fatalf("PrintJSON returned error: %v", err)
		}
	})

	// The output must be valid JSON.
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("PrintJSON output is not valid JSON: %v\noutput: %s", err, out)
	}

	// Verify pretty-printing (indented with two spaces).
	if !strings.Contains(out, "  ") {
		t.Errorf("expected indented output, got: %s", out)
	}

	// Verify values round-trip correctly.
	if decoded["name"] != "test" {
		t.Errorf("name = %v, want %q", decoded["name"], "test")
	}
	if decoded["count"] != float64(42) {
		t.Errorf("count = %v, want 42", decoded["count"])
	}
}

func TestPrintJSON_Slice(t *testing.T) {
	sample := []string{"alpha", "beta"}

	out := captureStdout(t, func() {
		if err := PrintJSON(sample); err != nil {
			t.Fatalf("PrintJSON returned error: %v", err)
		}
	})

	var decoded []string
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("PrintJSON output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(decoded) != 2 || decoded[0] != "alpha" || decoded[1] != "beta" {
		t.Errorf("unexpected decoded slice: %v", decoded)
	}
}

// ---------------------------------------------------------------------------
// PrintJSONL
// ---------------------------------------------------------------------------

func TestPrintJSONL(t *testing.T) {
	sample := map[string]interface{}{
		"name":  "test",
		"count": float64(42),
	}

	out := captureStdout(t, func() {
		if err := PrintJSONL(sample); err != nil {
			t.Fatalf("PrintJSONL returned error: %v", err)
		}
	})

	// Must be a single line (trimmed).
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d:\n%s", len(lines), out)
	}

	// Must be valid JSON.
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Fatalf("PrintJSONL output is not valid JSON: %v\nline: %s", err, lines[0])
	}

	if decoded["name"] != "test" {
		t.Errorf("name = %v, want %q", decoded["name"], "test")
	}
	if decoded["count"] != float64(42) {
		t.Errorf("count = %v, want 42", decoded["count"])
	}
}

func TestPrintJSONL_Compact(t *testing.T) {
	// Ensure the output does NOT contain pretty-print indentation.
	sample := map[string]string{"a": "1", "b": "2"}

	out := captureStdout(t, func() {
		if err := PrintJSONL(sample); err != nil {
			t.Fatalf("PrintJSONL returned error: %v", err)
		}
	})

	trimmed := strings.TrimSpace(out)
	// Compact JSON should not contain newlines within the object.
	if strings.Count(trimmed, "\n") != 0 {
		t.Errorf("expected compact single-line JSON, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Format constants
// ---------------------------------------------------------------------------

func TestFormatConstants(t *testing.T) {
	// All format constants must be distinct.
	formats := []Format{FormatTable, FormatJSON, FormatJSONL, FormatCSV}
	seen := make(map[Format]bool, len(formats))
	for _, f := range formats {
		if seen[f] {
			t.Errorf("duplicate format constant value: %q", f)
		}
		seen[f] = true
	}

	// Verify expected string representations.
	if FormatTable != "table" {
		t.Errorf("FormatTable = %q, want %q", FormatTable, "table")
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %q, want %q", FormatJSON, "json")
	}
	if FormatJSONL != "jsonl" {
		t.Errorf("FormatJSONL = %q, want %q", FormatJSONL, "jsonl")
	}
	if FormatCSV != "csv" {
		t.Errorf("FormatCSV = %q, want %q", FormatCSV, "csv")
	}
}
