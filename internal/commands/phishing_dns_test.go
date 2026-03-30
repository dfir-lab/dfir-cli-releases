package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
)

// newTestServer creates an httptest server that returns the given data payload
// wrapped in the standard API envelope.
func newTestServer(t *testing.T, data interface{}, meta client.ResponseMeta) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("marshal test data: %v", err)
		}
		envelope := struct {
			Data json.RawMessage    `json:"data"`
			Meta client.ResponseMeta `json:"meta"`
		}{
			Data: dataBytes,
			Meta: meta,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
}

func newTestErrorServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func TestPhishingDNS_TableOutput(t *testing.T) {
	dnsData := map[string]interface{}{
		"a":    []interface{}{"93.184.216.34"},
		"aaaa": []interface{}{"2606:2800:220:1:248:1893:25c8:1946"},
		"mx": []interface{}{
			map[string]interface{}{"priority": 10, "value": "mail.example.com"},
		},
		"ns":  []interface{}{"ns1.example.com", "ns2.example.com"},
		"txt": []interface{}{"v=spf1 -all"},
	}

	meta := client.ResponseMeta{
		RequestID:        "test-123",
		CreditsUsed:      1,
		CreditsRemaining: 99,
	}

	ts := newTestServer(t, dnsData, meta)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 10*time.Second, false)

	result, resp, err := c.PhishingDNS(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("PhishingDNS returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Meta.CreditsUsed != 1 {
		t.Errorf("expected 1 credit used, got %d", resp.Meta.CreditsUsed)
	}

	// Verify record counts.
	aRecords, ok := extractSlice(result, "a")
	if !ok || len(aRecords) != 1 {
		t.Errorf("expected 1 A record, got %d (ok=%v)", len(aRecords), ok)
	}

	mxRecords, ok := extractSlice(result, "mx")
	if !ok || len(mxRecords) != 1 {
		t.Errorf("expected 1 MX record, got %d (ok=%v)", len(mxRecords), ok)
	}

	nsRecords, ok := extractSlice(result, "ns")
	if !ok || len(nsRecords) != 2 {
		t.Errorf("expected 2 NS records, got %d (ok=%v)", len(nsRecords), ok)
	}

	// Test countDNSRecords.
	count := countDNSRecords(result)
	if count != 6 { // 1 A + 1 AAAA + 1 MX + 2 NS + 1 TXT
		t.Errorf("expected 6 records, got %d", count)
	}
}

func TestPhishingDNS_NestedRecords(t *testing.T) {
	dnsData := map[string]interface{}{
		"records": map[string]interface{}{
			"a":  []interface{}{"1.2.3.4"},
			"ns": []interface{}{"ns1.test.com"},
		},
	}

	meta := client.ResponseMeta{RequestID: "test-456"}
	ts := newTestServer(t, dnsData, meta)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 10*time.Second, false)

	result, _, err := c.PhishingDNS(t.Context(), "test.com")
	if err != nil {
		t.Fatalf("PhishingDNS returned error: %v", err)
	}

	count := countDNSRecords(result)
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}
}

func TestPhishingDNS_JSONOutput(t *testing.T) {
	dnsData := map[string]interface{}{
		"a": []interface{}{"10.0.0.1"},
	}

	meta := client.ResponseMeta{RequestID: "test-json"}
	ts := newTestServer(t, dnsData, meta)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 10*time.Second, false)

	result, _, err := c.PhishingDNS(t.Context(), "json-test.com")
	if err != nil {
		t.Fatalf("PhishingDNS returned error: %v", err)
	}

	// Verify we can marshal the result to JSON (as PrintJSON would).
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestPhishingDNS_APIError(t *testing.T) {
	ts := newTestErrorServer(http.StatusInternalServerError, `{"error":"internal server error"}`)
	defer ts.Close()

	c := client.New("test-key", ts.URL, "test-agent", 5*time.Second, false)
	c.SetAPIKey("test-key")

	_, _, err := c.PhishingDNS(t.Context(), "error-test.com")
	if err == nil {
		t.Fatal("expected error from API")
	}
}

func TestInterfaceToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
	}

	for _, tt := range tests {
		got := interfaceToString(tt.input)
		if got != tt.expected {
			t.Errorf("interfaceToString(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractSlice(t *testing.T) {
	m := map[string]interface{}{
		"a":  []interface{}{"1.2.3.4"},
		"mx": "not-a-slice",
	}

	// Valid slice extraction.
	s, ok := extractSlice(m, "a")
	if !ok || len(s) != 1 {
		t.Errorf("expected slice with 1 element, got ok=%v len=%d", ok, len(s))
	}

	// Non-slice value.
	_, ok = extractSlice(m, "mx")
	if ok {
		t.Error("expected ok=false for non-slice value")
	}

	// Missing key.
	_, ok = extractSlice(m, "missing")
	if ok {
		t.Error("expected ok=false for missing key")
	}
}
