package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// EnrichmentLookup
// ---------------------------------------------------------------------------

func TestEnrichmentLookup_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/enrichment/lookup" {
			t.Errorf("expected /enrichment/lookup, got %s", r.URL.Path)
		}

		// Verify request body.
		body, _ := io.ReadAll(r.Body)
		var req EnrichmentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}
		if len(req.Indicators) != 1 || req.Indicators[0].Value != "1.2.3.4" {
			t.Errorf("unexpected request body: %+v", req)
		}

		data := EnrichmentResponse{
			Results: []EnrichmentResult{
				{
					Indicator: Indicator{Type: "ip", Value: "1.2.3.4"},
					Verdict:   "malicious",
					Score:     85,
				},
			},
			Summary: EnrichmentSummary{Total: 1, Malicious: 1},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &EnrichmentRequest{
		Indicators: []Indicator{{Type: "ip", Value: "1.2.3.4"}},
	}

	result, resp, err := c.EnrichmentLookup(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Errorf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Verdict != "malicious" {
		t.Errorf("expected verdict=malicious, got %q", result.Results[0].Verdict)
	}
	if result.Summary.Total != 1 {
		t.Errorf("expected summary.total=1, got %d", result.Summary.Total)
	}
}

func TestEnrichmentLookup_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorEnvelope("validation_error", "invalid_indicator", "invalid IP", "req-400"))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &EnrichmentRequest{
		Indicators: []Indicator{{Type: "ip", Value: "bad"}},
	}

	result, _, err := c.EnrichmentLookup(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// PhishingAnalyze
// ---------------------------------------------------------------------------

func TestPhishingAnalyze_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/phishing/analyze" {
			t.Errorf("expected /phishing/analyze, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req PhishingAnalyzeRequest
		json.Unmarshal(body, &req)
		if req.InputType != "headers" {
			t.Errorf("expected input_type=headers, got %q", req.InputType)
		}

		data := PhishingAnalyzeResponse{
			Verdict: PhishingVerdict{
				Level:   "suspicious",
				Score:   65,
				Summary: "Suspicious indicators found",
			},
			KeyFindings: []string{"SPF fail"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &PhishingAnalyzeRequest{
		InputType: "headers",
		Content:   "From: attacker@evil.com\nTo: victim@corp.com",
	}

	result, resp, err := c.PhishingAnalyze(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Errorf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if result.Verdict.Level != "suspicious" {
		t.Errorf("expected verdict.level=suspicious, got %q", result.Verdict.Level)
	}
	if result.Verdict.Score != 65 {
		t.Errorf("expected verdict.score=65, got %d", result.Verdict.Score)
	}
	if len(result.KeyFindings) != 1 {
		t.Fatalf("expected 1 key finding, got %d", len(result.KeyFindings))
	}
}

func TestPhishingAnalyze_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errorEnvelope("authentication_error", "invalid_key", "bad key", "req-401"))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &PhishingAnalyzeRequest{InputType: "headers", Content: "test"}

	result, _, err := c.PhishingAnalyze(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// PhishingAnalyzeAI
// ---------------------------------------------------------------------------

func TestPhishingAnalyzeAI_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phishing/analyze/ai" {
			t.Errorf("expected /phishing/analyze/ai, got %s", r.URL.Path)
		}

		data := PhishingAIResponse{
			Analysis: PhishingAnalyzeResponse{
				Verdict: PhishingVerdict{Level: "malicious", Score: 95, Summary: "Highly suspicious"},
			},
			AIVerdict: &AIVerdict{
				RiskLevel:        "malicious",
				ConfidenceScore:  92,
				ExecutiveSummary: "This is a phishing email",
				Model:            "claude-3",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &PhishingAnalyzeRequest{InputType: "eml", Content: "full eml content"}

	result, _, err := c.PhishingAnalyzeAI(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AIVerdict == nil {
		t.Fatal("expected non-nil AIVerdict")
	}
	if result.AIVerdict.RiskLevel != "malicious" {
		t.Errorf("expected risk_level=malicious, got %q", result.AIVerdict.RiskLevel)
	}
	if result.AIVerdict.ConfidenceScore != 92 {
		t.Errorf("expected confidence_score=92, got %d", result.AIVerdict.ConfidenceScore)
	}
}

func TestPhishingAnalyzeAI_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write(errorEnvelope("credits_error", "insufficient_credits", "no credits", "req-402"))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &PhishingAnalyzeRequest{InputType: "eml", Content: "test"}

	result, _, err := c.PhishingAnalyzeAI(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// ExposureScan
// ---------------------------------------------------------------------------

func TestExposureScan_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/exposure/scan" {
			t.Errorf("expected /exposure/scan, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req ExposureScanRequest
		json.Unmarshal(body, &req)
		if req.Target != "example.com" {
			t.Errorf("expected target=example.com, got %q", req.Target)
		}

		data := ExposureScanResponse{
			ScanID:     "scan-123",
			Target:     "example.com",
			TargetType: "domain",
			Status:     "READY",
			RiskScore:  35,
			RiskLevel:  "low",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &ExposureScanRequest{Target: "example.com", TargetType: "domain"}

	result, resp, err := c.ExposureScan(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Errorf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if result.ScanID != "scan-123" {
		t.Errorf("expected scan_id=scan-123, got %q", result.ScanID)
	}
	if result.RiskLevel != "low" {
		t.Errorf("expected risk_level=low, got %q", result.RiskLevel)
	}
}

func TestExposureScan_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(errorEnvelope("authorization_error", "forbidden", "access denied", "req-403"))
	}))
	defer ts.Close()

	c := testClient(ts)
	req := &ExposureScanRequest{Target: "example.com"}

	result, _, err := c.ExposureScan(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// PhishingDNS
// ---------------------------------------------------------------------------

func TestPhishingDNS_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phishing/dns" {
			t.Errorf("expected /phishing/dns, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		json.Unmarshal(body, &reqBody)
		if reqBody["domain"] != "evil.com" {
			t.Errorf("expected domain=evil.com, got %q", reqBody["domain"])
		}

		data := map[string]interface{}{
			"domain":    "evil.com",
			"a_records": []string{"1.2.3.4"},
			"mx_exists": true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, _, err := c.PhishingDNS(context.Background(), "evil.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["domain"] != "evil.com" {
		t.Errorf("expected domain=evil.com, got %v", result["domain"])
	}
}

// ---------------------------------------------------------------------------
// PhishingBlacklist
// ---------------------------------------------------------------------------

func TestPhishingBlacklist_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phishing/blacklist" {
			t.Errorf("expected /phishing/blacklist, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string][]string
		json.Unmarshal(body, &reqBody)
		if len(reqBody["ips"]) != 2 {
			t.Errorf("expected 2 IPs, got %d", len(reqBody["ips"]))
		}

		data := map[string]interface{}{
			"results": map[string]interface{}{
				"1.2.3.4": map[string]interface{}{"listed": true},
				"5.6.7.8": map[string]interface{}{"listed": false},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, _, err := c.PhishingBlacklist(context.Background(), []string{"1.2.3.4", "5.6.7.8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["results"] == nil {
		t.Error("expected non-nil results")
	}
}

// ---------------------------------------------------------------------------
// PhishingURLExpand
// ---------------------------------------------------------------------------

func TestPhishingURLExpand_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phishing/url-expand" {
			t.Errorf("expected /phishing/url-expand, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		json.Unmarshal(body, &reqBody)
		if reqBody["url"] != "https://bit.ly/abc" {
			t.Errorf("expected url=https://bit.ly/abc, got %q", reqBody["url"])
		}

		data := map[string]interface{}{
			"original_url":   "https://bit.ly/abc",
			"expanded_url":   "https://example.com/full-page",
			"redirect_chain": []string{"https://bit.ly/abc", "https://example.com/full-page"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, _, err := c.PhishingURLExpand(context.Background(), "https://bit.ly/abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["expanded_url"] != "https://example.com/full-page" {
		t.Errorf("expected expanded_url, got %v", result["expanded_url"])
	}
}

// ---------------------------------------------------------------------------
// PhishingSafeBrowsing
// ---------------------------------------------------------------------------

func TestPhishingSafeBrowsing_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/phishing/safe-browsing" {
			t.Errorf("expected /phishing/safe-browsing, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string][]string
		json.Unmarshal(body, &reqBody)
		if len(reqBody["urls"]) != 1 {
			t.Errorf("expected 1 URL, got %d", len(reqBody["urls"]))
		}

		data := map[string]interface{}{
			"results": map[string]interface{}{
				"https://evil.com": map[string]interface{}{
					"safe":   false,
					"threat": "MALWARE",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, _, err := c.PhishingSafeBrowsing(context.Background(), []string{"https://evil.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["results"] == nil {
		t.Error("expected non-nil results")
	}
}

// ---------------------------------------------------------------------------
// AuthValidate
// ---------------------------------------------------------------------------

func TestAuthValidate_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/auth/validate" {
			t.Errorf("expected /auth/validate, got %s", r.URL.Path)
		}

		data := AuthValidateResponse{
			Plan:             "starter",
			Credits:          1234,
			OrganizationName: "DFIR Lab",
			OrganizationID:   "org-123",
			Permissions:      []string{"api:read", "api:write"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, resp, err := c.AuthValidate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Errorf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if result.Plan != "starter" {
		t.Errorf("expected plan=starter, got %q", result.Plan)
	}
	if result.Credits != 1234 {
		t.Errorf("expected credits=1234, got %d", result.Credits)
	}
}

func TestAuthValidate_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errorEnvelope("authentication_error", "invalid_key", "bad key", "req-auth-401"))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, _, err := c.AuthValidate(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func TestHealth_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/health" {
			t.Errorf("expected /health, got %s", r.URL.Path)
		}

		data := HealthResponse{
			Status:    "operational",
			Version:   "1.0.0",
			Timestamp: "2026-01-01T00:00:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(data))
	}))
	defer ts.Close()

	c := testClient(ts)
	result, resp, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Errorf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if result.Status != "operational" {
		t.Errorf("expected status=operational, got %q", result.Status)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %q", result.Version)
	}
}

func TestHealth_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorEnvelope("server_error", "internal", "service down", "req-health-500"))
	}))
	defer ts.Close()

	c := testClient(ts)
	// Exhausts retries then fails.
	result, _, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}
