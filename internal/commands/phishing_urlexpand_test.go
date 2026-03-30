package commands

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// resolveURLInput
// ---------------------------------------------------------------------------

func TestURLExpandResolveURLInput_Flag(t *testing.T) {
	url, err := resolveURLInput("https://bit.ly/abc123")
	if err != nil {
		t.Fatalf("resolveURLInput returned error: %v", err)
	}
	if url != "https://bit.ly/abc123" {
		t.Errorf("url = %q, want %q", url, "https://bit.ly/abc123")
	}
}

func TestURLExpandResolveURLInput_NoInput(t *testing.T) {
	_, err := resolveURLInput("")
	if err == nil {
		t.Fatal("expected error when no input is provided")
	}
	if !strings.Contains(err.Error(), "no URL provided") {
		t.Errorf("error should mention 'no URL provided', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// mapString
// ---------------------------------------------------------------------------

func TestMapString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "string_value",
			m:    map[string]interface{}{"url": "https://example.com"},
			key:  "url",
			want: "https://example.com",
		},
		{
			name: "int_value",
			m:    map[string]interface{}{"status_code": 200},
			key:  "status_code",
			want: "200",
		},
		{
			name: "missing_key",
			m:    map[string]interface{}{"url": "https://example.com"},
			key:  "missing",
			want: "",
		},
		{
			name: "nil_map",
			m:    nil,
			key:  "url",
			want: "",
		},
		{
			name: "float_value",
			m:    map[string]interface{}{"score": 85.5},
			key:  "score",
			want: "85.5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapString(tc.m, tc.key)
			if got != tc.want {
				t.Errorf("mapString(%v, %q) = %q, want %q", tc.m, tc.key, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// renderURLExpandTable (smoke test — ensure no panics)
// ---------------------------------------------------------------------------

func TestRenderURLExpandTable_NoPanic(t *testing.T) {
	// A minimal result map — just make sure the render function does not panic.
	result := map[string]interface{}{
		"original_url": "https://bit.ly/abc123",
		"expanded_url": "https://example.com/landing",
		"status_code":  float64(200),
		"redirect_chain": []interface{}{
			"https://bit.ly/abc123",
			"https://example.com/landing",
		},
	}

	// Should not panic even with nil response.
	renderURLExpandTable(result, nil)
}

func TestRenderURLExpandTable_EmptyResult(t *testing.T) {
	// An empty result — should not panic.
	result := map[string]interface{}{}
	renderURLExpandTable(result, nil)
}
