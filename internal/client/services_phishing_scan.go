package client

import (
	"context"
	"net/http"
)

// PhishingCheckPhish submits a URL to the CheckPhish service.
// POST /phishing/checkphish
func (c *Client) PhishingCheckPhish(ctx context.Context, url string) (map[string]interface{}, *Response, error) {
	body := map[string]string{"url": url}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/checkphish", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// PhishingURLScan submits a URL to URLScan.io for analysis.
// POST /phishing/urlscan
func (c *Client) PhishingURLScan(ctx context.Context, url string) (map[string]interface{}, *Response, error) {
	body := map[string]string{"url": url}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/urlscan", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
