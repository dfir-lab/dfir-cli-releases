package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// usageEnvelope wraps a UsageResponse in the standard API envelope.
func usageEnvelope(data *client.UsageResponse) []byte {
	d, _ := json.Marshal(data)
	env := map[string]interface{}{
		"data": json.RawMessage(d),
		"meta": map[string]interface{}{
			"request_id":         "req-usage-001",
			"credits_used":       1,
			"credits_remaining":  99,
			"processing_time_ms": 42,
		},
	}
	b, _ := json.Marshal(env)
	return b
}

// sampleUsageResponse returns a realistic usage response for tests.
func sampleUsageResponse() *client.UsageResponse {
	return &client.UsageResponse{
		Period:        "March 2026",
		TotalRequests: 1247,
		TotalCredits:  6580,
		ByService: map[string]client.ServiceUsage{
			"enrichment": {Requests: 892, Credits: 2140},
			"phishing":   {Requests: 298, Credits: 3280},
			"exposure":   {Requests: 57, Credits: 1160},
		},
		TopOperations: []client.OperationUsage{
			{Operation: "lookup", Service: "enrichment", Requests: 892, Credits: 2140},
			{Operation: "analyze", Service: "phishing", Requests: 298, Credits: 3280},
			{Operation: "scan", Service: "exposure", Requests: 57, Credits: 1160},
		},
	}
}

// usageMockServer creates a test server that returns the given usage response.
func usageMockServer(resp *client.UsageResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(usageEnvelope(resp))
	}))
}

// usageTestClient creates a client pointing to the given test server.
func usageTestClient(ts *httptest.Server) *client.Client {
	c := client.New("sk-dfir-testapikey1234", ts.URL, "dfir-cli-test/1.0", 5*time.Second, false)
	return c
}

// ---------------------------------------------------------------------------
// TestRunUsage_Table
// ---------------------------------------------------------------------------

func TestRunUsage_Table(t *testing.T) {
	ts := usageMockServer(sampleUsageResponse())
	defer ts.Close()

	// Set up a temporary config dir so SaveCreditState works.
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Build a command that mimics the real one.
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	// Override the root flags for the test.
	origGetAPIKey := GetAPIKey
	origGetAPIURL := GetAPIURL
	origGetOutputFormat := GetOutputFormat
	origIsQuiet := IsQuiet
	defer func() {
		// These are package-level functions, so we restore them.
		_ = origGetAPIKey
		_ = origGetAPIURL
		_ = origGetOutputFormat
		_ = origIsQuiet
	}()

	// Directly call the client and render.
	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.UsageRequest{Period: "current"}
	result, resp, err := apiClient.Usage(ctx, req)
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	if err := renderUsageTable(cmd, result, resp); err != nil {
		t.Fatalf("renderUsageTable returned error: %v", err)
	}

	out := buf.String()

	// Verify key content is present.
	checks := []string{
		"API Usage",
		"March 2026",
		"Total Requests",
		"1,247",
		"Total Credits",
		"6,580",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\nGot:\n%s", want, out)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_JSON
// ---------------------------------------------------------------------------

func TestRunUsage_JSON(t *testing.T) {
	ts := usageMockServer(sampleUsageResponse())
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.UsageRequest{Period: "current"}
	result, resp, err := apiClient.Usage(ctx, req)
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	data, err := renderUsageJSONRaw(result, resp)
	if err != nil {
		t.Fatalf("renderUsageJSONRaw returned error: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON output is not valid: %v\nGot:\n%s", err, string(data))
	}

	// Check key fields.
	if period, ok := parsed["period"].(string); !ok || period != "March 2026" {
		t.Errorf("expected period 'March 2026', got %v", parsed["period"])
	}

	totalReqs, ok := parsed["total_requests"].(float64)
	if !ok || int(totalReqs) != 1247 {
		t.Errorf("expected total_requests 1247, got %v", parsed["total_requests"])
	}

	totalCreds, ok := parsed["total_credits"].(float64)
	if !ok || int(totalCreds) != 6580 {
		t.Errorf("expected total_credits 6580, got %v", parsed["total_credits"])
	}

	// Verify meta is present.
	if _, ok := parsed["meta"]; !ok {
		t.Error("expected 'meta' field in JSON output")
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_Quiet
// ---------------------------------------------------------------------------

func TestRunUsage_Quiet(t *testing.T) {
	ts := usageMockServer(sampleUsageResponse())
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.UsageRequest{Period: "current"}
	result, _, err := apiClient.Usage(ctx, req)
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	if err := renderUsageQuiet(cmd, result); err != nil {
		t.Fatalf("renderUsageQuiet returned error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "6580" {
		t.Errorf("quiet output = %q, want %q", out, "6580")
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_PeriodFilter
// ---------------------------------------------------------------------------

func TestRunUsage_PeriodFilter(t *testing.T) {
	var receivedPeriod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPeriod = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(usageEnvelope(sampleUsageResponse()))
	}))
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.UsageRequest{Period: "2026-01"}
	_, _, err := apiClient.Usage(ctx, req)
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	if receivedPeriod != "2026-01" {
		t.Errorf("server received period=%q, want %q", receivedPeriod, "2026-01")
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_ServiceFilter
// ---------------------------------------------------------------------------

func TestRunUsage_ServiceFilter(t *testing.T) {
	var receivedService string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedService = r.URL.Query().Get("service")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(usageEnvelope(sampleUsageResponse()))
	}))
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	req := &client.UsageRequest{Service: "enrichment"}
	_, _, err := apiClient.Usage(ctx, req)
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	if receivedService != "enrichment" {
		t.Errorf("server received service=%q, want %q", receivedService, "enrichment")
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_AuthError
// ---------------------------------------------------------------------------

func TestRunUsage_AuthError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		env := map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "authentication_error",
				"message": "invalid API key",
			},
		}
		b, _ := json.Marshal(env)
		w.Write(b)
	}))
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	_, _, err := apiClient.Usage(ctx, &client.UsageRequest{Period: "current"})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}

	var authErr *client.AuthenticationError
	if !isAuthError(err, &authErr) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// TestRunUsage_EmptyResponse
// ---------------------------------------------------------------------------

func TestRunUsage_EmptyResponse(t *testing.T) {
	emptyResp := &client.UsageResponse{
		Period:        "March 2026",
		TotalRequests: 0,
		TotalCredits:  0,
		ByService:     map[string]client.ServiceUsage{},
	}

	ts := usageMockServer(emptyResp)
	defer ts.Close()

	apiClient := usageTestClient(ts)
	ctx, cancel := signalContext()
	defer cancel()

	result, resp, err := apiClient.Usage(ctx, &client.UsageRequest{Period: "current"})
	if err != nil {
		t.Fatalf("Usage() returned error: %v", err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	if err := renderUsageTable(cmd, result, resp); err != nil {
		t.Fatalf("renderUsageTable returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total Requests") {
		t.Errorf("expected table output header, got:\n%s", out)
	}
	if !strings.Contains(out, "0") {
		t.Errorf("expected zero values in output, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// TestFormatNumber
// ---------------------------------------------------------------------------

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{1247, "1,247"},
		{6580, "6,580"},
		{1000000, "1,000,000"},
		{-1247, "-1,247"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := formatNumber(tc.input)
			if got != tc.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isAuthError checks if the error is an AuthenticationError.
func isAuthError(err error, target **client.AuthenticationError) bool {
	var authErr *client.AuthenticationError
	if ok := isErrType(err, &authErr); ok {
		*target = authErr
		return true
	}
	return false
}

// isErrType is a generic error type check helper.
func isErrType[T error](err error, target *T) bool {
	return err != nil && errorAs(err, target)
}

// errorAs wraps errors.As for use in generic helper.
func errorAs[T error](err error, target *T) bool {
	var zero T
	_ = zero
	return errors.As(err, target)
}
