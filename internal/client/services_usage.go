package client

import (
	"context"
	"net/http"
	"net/url"
)

// Usage retrieves API usage statistics.
// GET /usage?period=X&service=Y
func (c *Client) Usage(ctx context.Context, req *UsageRequest) (*UsageResponse, *Response, error) {
	path := "/usage"
	params := url.Values{}
	if req != nil {
		if req.Period != "" {
			params.Set("period", req.Period)
		}
		if req.Service != "" {
			params.Set("service", req.Service)
		}
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result UsageResponse
	resp, err := c.Do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, resp, err
	}
	return &result, resp, nil
}
