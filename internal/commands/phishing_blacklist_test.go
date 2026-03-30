package commands

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
)

func TestPhishingBlacklist_TableOutput(t *testing.T) {
	blData := map[string]interface{}{
		"results": map[string]interface{}{
			"1.2.3.4": map[string]interface{}{
				"listed_count": float64(2),
				"total_count":  float64(5),
				"blacklists": map[string]interface{}{
					"zen.spamhaus.org":      true,
					"bl.spamcop.net":        true,
					"dnsbl.sorbs.net":       false,
					"b.barracudacentral.org": false,
					"cbl.abuseat.org":       false,
				},
			},
		},
	}

	meta := client.ResponseMeta{
		RequestID:        "test-bl-123",
		CreditsUsed:      1,
		CreditsRemaining: 98,
	}

	ts := newTestServer(t, blData, meta)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 10*time.Second, false)

	result, resp, err := c.PhishingBlacklist(t.Context(), []string{"1.2.3.4"})
	if err != nil {
		t.Fatalf("PhishingBlacklist returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Meta.CreditsUsed != 1 {
		t.Errorf("expected 1 credit used, got %d", resp.Meta.CreditsUsed)
	}

	// Verify results extraction.
	results := blacklistExtractResults(result)
	if len(results) != 1 {
		t.Errorf("expected 1 IP result, got %d", len(results))
	}

	ipData, ok := results["1.2.3.4"]
	if !ok {
		t.Fatal("expected result for 1.2.3.4")
	}

	listed := countBlacklistListings(ipData)
	if listed != 2 {
		t.Errorf("expected 2 listings, got %d", listed)
	}

	total := countBlacklistTotal(ipData)
	if total != 5 {
		t.Errorf("expected 5 total blacklists, got %d", total)
	}
}

func TestPhishingBlacklist_JSONOutput(t *testing.T) {
	blData := map[string]interface{}{
		"results": map[string]interface{}{
			"10.0.0.1": map[string]interface{}{
				"blacklists": map[string]interface{}{
					"zen.spamhaus.org": false,
				},
			},
		},
	}

	meta := client.ResponseMeta{RequestID: "test-bl-json"}
	ts := newTestServer(t, blData, meta)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 10*time.Second, false)

	result, _, err := c.PhishingBlacklist(t.Context(), []string{"10.0.0.1"})
	if err != nil {
		t.Fatalf("PhishingBlacklist returned error: %v", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestPhishingBlacklist_APIError(t *testing.T) {
	ts := newTestErrorServer(http.StatusInternalServerError, `{"error":"internal server error"}`)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 5*time.Second, false)

	_, _, err := c.PhishingBlacklist(t.Context(), []string{"1.2.3.4"})
	if err == nil {
		t.Fatal("expected error from API")
	}
}

func TestPhishingBlacklist_QuietCounting(t *testing.T) {
	// Test with bool-style blacklist entries.
	ipData := map[string]interface{}{
		"blacklists": map[string]interface{}{
			"list1": true,
			"list2": false,
			"list3": true,
		},
	}

	listed := countBlacklistListings(ipData)
	if listed != 2 {
		t.Errorf("expected 2 listings, got %d", listed)
	}
}

func TestBlacklistStatusString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{true, "listed"},
		{false, "not listed"},
		{"listed", "listed"},
		{"clean", "clean"},
	}

	for _, tt := range tests {
		got := blacklistStatusString(tt.input)
		if got != tt.expected {
			t.Errorf("blacklistStatusString(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsListed(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected bool
	}{
		{true, true},
		{false, false},
		{"listed", true},
		{"LISTED", true},
		{"not listed", false},
		{map[string]interface{}{"listed": true}, true},
		{map[string]interface{}{"listed": false}, false},
		{map[string]interface{}{"status": "listed"}, true},
		{map[string]interface{}{"status": "clean"}, false},
		{nil, false},
	}

	for _, tt := range tests {
		got := isListed(tt.input)
		if got != tt.expected {
			t.Errorf("isListed(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestResolveBlacklistIPs(t *testing.T) {
	// Single IP flag.
	ips, err := resolveBlacklistIPs("1.2.3.4", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "1.2.3.4" {
		t.Errorf("expected [1.2.3.4], got %v", ips)
	}

	// No input.
	ips, err = resolveBlacklistIPs("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ips != nil {
		t.Errorf("expected nil, got %v", ips)
	}
}

func TestBlacklistExtractResults(t *testing.T) {
	// Nested under "results".
	result := map[string]interface{}{
		"results": map[string]interface{}{
			"1.2.3.4": map[string]interface{}{},
		},
	}
	extracted := blacklistExtractResults(result)
	if _, ok := extracted["1.2.3.4"]; !ok {
		t.Error("expected to find 1.2.3.4 in extracted results")
	}

	// Nested under "ips".
	result2 := map[string]interface{}{
		"ips": map[string]interface{}{
			"5.6.7.8": map[string]interface{}{},
		},
	}
	extracted2 := blacklistExtractResults(result2)
	if _, ok := extracted2["5.6.7.8"]; !ok {
		t.Error("expected to find 5.6.7.8 in extracted results")
	}

	// Top-level.
	result3 := map[string]interface{}{
		"9.10.11.12": map[string]interface{}{},
	}
	extracted3 := blacklistExtractResults(result3)
	if _, ok := extracted3["9.10.11.12"]; !ok {
		t.Error("expected to find 9.10.11.12 in extracted results")
	}
}
