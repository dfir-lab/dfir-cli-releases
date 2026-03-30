package client

import (
	"context"
	"net/http"
)

// PhishingEnrich performs enrichment on phishing indicators.
// POST /phishing/enrich
func (c *Client) PhishingEnrich(ctx context.Context, url string) (map[string]interface{}, *Response, error) {
	body := map[string]string{"url": url}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/enrich", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
