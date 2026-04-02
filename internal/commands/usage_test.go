package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/spf13/cobra"
)

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

func testMeta(requestID string, creditsUsed int) *client.ResponseMeta {
	return &client.ResponseMeta{
		RequestID:        requestID,
		CreditsUsed:      creditsUsed,
		CreditsRemaining: 100 - creditsUsed,
	}
}

func seedUsageLedger(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	origTimeNowMonth := timeNowMonth
	origUsageNow := usageNow
	timeNowMonth = func() string { return "2026-03" }
	usageNow = func() time.Time { return time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		timeNowMonth = origTimeNowMonth
		usageNow = origUsageNow
	})

	events := []struct {
		meta      *client.ResponseMeta
		service   string
		operation string
	}{
		{testMeta("req-1", 3), "enrichment", "lookup"},
		{testMeta("req-2", 3), "enrichment", "lookup"},
		{testMeta("req-3", 1), "phishing", "analyze"},
		{testMeta("req-4", 10), "exposure", "scan"},
	}

	for _, event := range events {
		if err := SaveUsageEvent(event.meta, event.service, event.operation); err != nil {
			t.Fatalf("SaveUsageEvent(%s/%s) failed: %v", event.service, event.operation, err)
		}
	}

	timeNowMonth = func() string { return "2026-02" }
	if err := SaveUsageEvent(testMeta("req-prev", 10), "phishing", "analyze-ai"); err != nil {
		t.Fatalf("SaveUsageEvent(previous month) failed: %v", err)
	}
	timeNowMonth = func() string { return "2026-03" }
}

func TestBuildUsageResponse_CurrentPeriod(t *testing.T) {
	seedUsageLedger(t)

	result, err := buildUsageResponse("current", "")
	if err != nil {
		t.Fatalf("buildUsageResponse returned error: %v", err)
	}

	if result.Period != "March 2026" {
		t.Fatalf("Period = %q, want %q", result.Period, "March 2026")
	}
	if result.TotalRequests != 4 {
		t.Fatalf("TotalRequests = %d, want 4", result.TotalRequests)
	}
	if result.TotalCredits != 17 {
		t.Fatalf("TotalCredits = %d, want 17", result.TotalCredits)
	}
	if result.ByService["enrichment"].Requests != 2 {
		t.Fatalf("enrichment requests = %d, want 2", result.ByService["enrichment"].Requests)
	}
	if len(result.TopOperations) != 3 {
		t.Fatalf("TopOperations len = %d, want 3", len(result.TopOperations))
	}
	if got := result.TopOperations[0]; got.Service != "enrichment" || got.Operation != "lookup" || got.Requests != 2 {
		t.Fatalf("top operation[0] = %+v, want enrichment/lookup with 2 requests", got)
	}
}

func TestBuildUsageResponse_ServiceFilter(t *testing.T) {
	seedUsageLedger(t)

	result, err := buildUsageResponse("current", "enrichment")
	if err != nil {
		t.Fatalf("buildUsageResponse returned error: %v", err)
	}

	if result.TotalRequests != 2 {
		t.Fatalf("TotalRequests = %d, want 2", result.TotalRequests)
	}
	if result.TotalCredits != 6 {
		t.Fatalf("TotalCredits = %d, want 6", result.TotalCredits)
	}
	if len(result.ByService) != 1 {
		t.Fatalf("ByService len = %d, want 1", len(result.ByService))
	}
	if len(result.TopOperations) != 1 {
		t.Fatalf("TopOperations len = %d, want 1", len(result.TopOperations))
	}
	if got := result.TopOperations[0]; got.Service != "enrichment" || got.Operation != "lookup" {
		t.Fatalf("TopOperations[0] = %+v, want enrichment/lookup", got)
	}
}

func TestBuildUsageResponse_PreviousPeriod(t *testing.T) {
	seedUsageLedger(t)

	result, err := buildUsageResponse("previous", "")
	if err != nil {
		t.Fatalf("buildUsageResponse returned error: %v", err)
	}

	if result.Period != "February 2026" {
		t.Fatalf("Period = %q, want %q", result.Period, "February 2026")
	}
	if result.TotalRequests != 1 || result.TotalCredits != 10 {
		t.Fatalf("unexpected totals: %+v", result)
	}
}

func TestBuildUsageResponse_NoState(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", t.TempDir())

	origUsageNow := usageNow
	usageNow = func() time.Time { return time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { usageNow = origUsageNow })

	result, err := buildUsageResponse("current", "")
	if err != nil {
		t.Fatalf("buildUsageResponse returned error: %v", err)
	}
	if result.TotalRequests != 0 || result.TotalCredits != 0 {
		t.Fatalf("expected empty totals, got %+v", result)
	}
}

func TestResolveUsagePeriod_Invalid(t *testing.T) {
	if _, _, err := resolveUsagePeriod("2026/03"); err == nil {
		t.Fatal("expected invalid period error, got nil")
	}
}

func TestRunUsage_Table(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	if err := renderUsageTable(cmd, sampleUsageResponse()); err != nil {
		t.Fatalf("renderUsageTable returned error: %v", err)
	}

	out := buf.String()
	checks := []string{
		"Local API Usage",
		"March 2026",
		"Total Requests",
		"1,247",
		"Total Credits",
		"6,580",
		"locally recorded dfir-cli activity",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestRunUsage_JSON(t *testing.T) {
	data, err := renderUsageJSONRaw(sampleUsageResponse())
	if err != nil {
		t.Fatalf("renderUsageJSONRaw returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON output is not valid: %v\nGot:\n%s", err, string(data))
	}

	if period, ok := parsed["period"].(string); !ok || period != "March 2026" {
		t.Errorf("expected period 'March 2026', got %v", parsed["period"])
	}
	if totalReqs, ok := parsed["total_requests"].(float64); !ok || int(totalReqs) != 1247 {
		t.Errorf("expected total_requests 1247, got %v", parsed["total_requests"])
	}
	if _, ok := parsed["meta"]; ok {
		t.Error("did not expect legacy remote meta field in JSON output")
	}
}

func TestRunUsage_Quiet(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	if err := renderUsageQuiet(cmd, sampleUsageResponse()); err != nil {
		t.Fatalf("renderUsageQuiet returned error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "6580" {
		t.Errorf("quiet output = %q, want %q", out, "6580")
	}
}

func TestRunUsage_EmptyResponse(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "usage"}
	cmd.SetOut(&buf)

	emptyResp := &client.UsageResponse{
		Period:        "March 2026",
		TotalRequests: 0,
		TotalCredits:  0,
		ByService:     map[string]client.ServiceUsage{},
	}
	if err := renderUsageTable(cmd, emptyResp); err != nil {
		t.Fatalf("renderUsageTable returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No locally recorded API usage for this period yet.") {
		t.Fatalf("expected empty-state guidance, got:\n%s", out)
	}
}

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
