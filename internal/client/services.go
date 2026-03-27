package client

import (
	"context"
	"net/http"
)

// ---------------------------------------------------------------------------
// Enrichment
// ---------------------------------------------------------------------------

// EnrichmentLookup looks up IOCs across threat intelligence providers.
// POST /enrichment/lookup
func (c *Client) EnrichmentLookup(ctx context.Context, req *EnrichmentRequest) (*EnrichmentResponse, *Response, error) {
	var result EnrichmentResponse
	resp, err := c.Do(ctx, http.MethodPost, "/enrichment/lookup", req, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}

// ---------------------------------------------------------------------------
// Phishing
// ---------------------------------------------------------------------------

// PhishingAnalyze analyzes an email or URL for phishing indicators.
// POST /phishing/analyze
func (c *Client) PhishingAnalyze(ctx context.Context, req *PhishingAnalyzeRequest) (*PhishingAnalyzeResponse, *Response, error) {
	var result PhishingAnalyzeResponse
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/analyze", req, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}

// PhishingAnalyzeAI performs AI-enhanced phishing analysis.
// POST /phishing/analyze/ai
//
// When the --ai flag is passed in the CLI, this endpoint is called directly.
// The API handles chaining the heuristic and AI analysis internally.
func (c *Client) PhishingAnalyzeAI(ctx context.Context, req *PhishingAnalyzeRequest) (*PhishingAIResponse, *Response, error) {
	var result PhishingAIResponse
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/analyze/ai", req, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}

// PhishingDNS performs DNS analysis on a domain.
// POST /phishing/dns
func (c *Client) PhishingDNS(ctx context.Context, domain string) (map[string]interface{}, *Response, error) {
	body := map[string]string{"domain": domain}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/dns", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// PhishingBlacklist checks IPs against blacklists.
// POST /phishing/blacklist
func (c *Client) PhishingBlacklist(ctx context.Context, ips []string) (map[string]interface{}, *Response, error) {
	body := map[string][]string{"ips": ips}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/blacklist", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// PhishingURLExpand expands shortened URLs.
// POST /phishing/url-expand
func (c *Client) PhishingURLExpand(ctx context.Context, url string) (map[string]interface{}, *Response, error) {
	body := map[string]string{"url": url}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/url-expand", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// PhishingSafeBrowsing checks URLs against Google Safe Browsing.
// POST /phishing/safe-browsing
func (c *Client) PhishingSafeBrowsing(ctx context.Context, urls []string) (map[string]interface{}, *Response, error) {
	body := map[string][]string{"urls": urls}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/safe-browsing", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// ---------------------------------------------------------------------------
// Exposure
// ---------------------------------------------------------------------------

// ExposureScan scans a domain or IP for exposure.
// POST /exposure/scan
func (c *Client) ExposureScan(ctx context.Context, req *ExposureScanRequest) (*ExposureScanResponse, *Response, error) {
	var result ExposureScanResponse
	resp, err := c.Do(ctx, http.MethodPost, "/exposure/scan", req, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

// Health checks the API health status.
// GET /health
func (c *Client) Health(ctx context.Context) (*HealthResponse, *Response, error) {
	var result HealthResponse
	resp, err := c.Do(ctx, http.MethodGet, "/health", nil, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}
