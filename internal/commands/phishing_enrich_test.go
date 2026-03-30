package commands

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// resolveEnrichURLInput
// ---------------------------------------------------------------------------

func TestResolveEnrichURLInput_Flag(t *testing.T) {
	url, err := resolveEnrichURLInput("https://suspicious.example.com")
	if err != nil {
		t.Fatalf("resolveEnrichURLInput returned error: %v", err)
	}
	if url != "https://suspicious.example.com" {
		t.Errorf("url = %q, want %q", url, "https://suspicious.example.com")
	}
}

func TestResolveEnrichURLInput_NoInput(t *testing.T) {
	_, err := resolveEnrichURLInput("")
	if err == nil {
		t.Fatal("expected error when no input is provided")
	}
	if !strings.Contains(err.Error(), "no URL provided") {
		t.Errorf("error should mention 'no URL provided', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Usage:") {
		t.Errorf("error should contain usage hint, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// renderPhishingEnrichTable (smoke test — ensure no panics)
// ---------------------------------------------------------------------------

func TestRenderPhishingEnrichTable_NoPanic(t *testing.T) {
	result := map[string]interface{}{
		"url":        "https://suspicious.example.com",
		"risk_score": float64(75),
		"risk_level": "high",
		"categories": []interface{}{"phishing", "malware"},
	}

	// Should not panic even with nil response.
	renderPhishingEnrichTable(result, nil)
}

func TestRenderPhishingEnrichTable_EmptyResult(t *testing.T) {
	result := map[string]interface{}{}
	renderPhishingEnrichTable(result, nil)
}

func TestRenderPhishingEnrichTable_WithExtraFields(t *testing.T) {
	result := map[string]interface{}{
		"url":        "https://evil.example.com",
		"risk_score": float64(90),
		"risk_level": "critical",
		"categories": []interface{}{"phishing"},
		"threat_intel": map[string]interface{}{
			"source_a": "malicious",
			"source_b": "suspicious",
		},
		"tags": []interface{}{"credential-harvesting", "brand-impersonation"},
		"note": "Flagged by multiple providers",
	}

	// Should not panic — exercises the extra fields rendering path.
	renderPhishingEnrichTable(result, nil)
}

// ---------------------------------------------------------------------------
// printEnrichmentExtraFields (smoke test)
// ---------------------------------------------------------------------------

func TestPrintEnrichmentExtraFields_NoPanic(t *testing.T) {
	tests := []struct {
		name   string
		result map[string]interface{}
	}{
		{
			name:   "empty",
			result: map[string]interface{}{},
		},
		{
			name: "only_handled_keys",
			result: map[string]interface{}{
				"url":        "https://example.com",
				"risk_score": float64(50),
			},
		},
		{
			name: "with_map_field",
			result: map[string]interface{}{
				"providers": map[string]interface{}{
					"vt":  "clean",
					"gsb": "malicious",
				},
			},
		},
		{
			name: "with_slice_field",
			result: map[string]interface{}{
				"findings": []interface{}{"finding1", "finding2"},
			},
		},
		{
			name: "with_scalar_field",
			result: map[string]interface{}{
				"scan_id": "abc-123",
			},
		},
		{
			name: "with_nil_value",
			result: map[string]interface{}{
				"empty_field": nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic.
			printEnrichmentExtraFields(tc.result)
		})
	}
}
