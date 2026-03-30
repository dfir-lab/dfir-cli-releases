package commands

import (
	"testing"
)

// ---------------------------------------------------------------------------
// resolveURLInput
// ---------------------------------------------------------------------------

func TestResolveURLInput_Flag(t *testing.T) {
	url, err := resolveURLInput("https://example.com")
	if err != nil {
		t.Fatalf("resolveURLInput returned error: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("url = %q, want %q", url, "https://example.com")
	}
}

func TestResolveURLInput_NoInput(t *testing.T) {
	// No flag and stdin is not a pipe — should return an error.
	_, err := resolveURLInput("")
	if err == nil {
		t.Fatal("expected error when no URL is provided")
	}
}

// ---------------------------------------------------------------------------
// mapStr
// ---------------------------------------------------------------------------

func TestMapStr(t *testing.T) {
	m := map[string]interface{}{
		"disposition": "phishing",
		"score":       42.0,
		"empty":       "",
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"existing_string", "disposition", "phishing"},
		{"non_string_value", "score", "42"},
		{"empty_value", "empty", ""},
		{"missing_key", "nonexistent", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapStr(m, tc.key)
			if got != tc.want {
				t.Errorf("mapStr(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestMapStr_NilMap(t *testing.T) {
	got := mapStr(nil, "key")
	if got != "" {
		t.Errorf("mapStr(nil, key) = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// mapSliceStr
// ---------------------------------------------------------------------------

func TestMapSliceStr(t *testing.T) {
	m := map[string]interface{}{
		"ips": []interface{}{"1.2.3.4", "5.6.7.8"},
	}

	got := mapSliceStr(m, "ips")
	if len(got) != 2 {
		t.Fatalf("mapSliceStr returned %d items, want 2", len(got))
	}
	if got[0] != "1.2.3.4" || got[1] != "5.6.7.8" {
		t.Errorf("mapSliceStr = %v, want [1.2.3.4 5.6.7.8]", got)
	}
}

func TestMapSliceStr_Missing(t *testing.T) {
	m := map[string]interface{}{}
	got := mapSliceStr(m, "missing")
	if got != nil {
		t.Errorf("mapSliceStr for missing key = %v, want nil", got)
	}
}

func TestMapSliceStr_NilMap(t *testing.T) {
	got := mapSliceStr(nil, "key")
	if got != nil {
		t.Errorf("mapSliceStr(nil, key) = %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// newPhishingCheckPhishCmd
// ---------------------------------------------------------------------------

func TestNewPhishingCheckPhishCmd_HasURLFlag(t *testing.T) {
	cmd := newPhishingCheckPhishCmd()

	if cmd.Use != "checkphish" {
		t.Errorf("Use = %q, want %q", cmd.Use, "checkphish")
	}

	f := cmd.Flags().Lookup("url")
	if f == nil {
		t.Fatal("expected --url flag to be defined")
	}
}
