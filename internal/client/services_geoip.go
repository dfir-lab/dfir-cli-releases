package client

import (
	"context"
	"net/http"
)

// PhishingGeoIP performs GeoIP lookup on IPs.
// POST /phishing/geoip
func (c *Client) PhishingGeoIP(ctx context.Context, ips []string) (map[string]interface{}, *Response, error) {
	body := map[string][]string{"ips": ips}
	var result map[string]interface{}
	resp, err := c.Do(ctx, http.MethodPost, "/phishing/geoip", body, &result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
