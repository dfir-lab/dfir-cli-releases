package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"strings"
	"time"
)

const defaultBaseURL = "https://dfir-lab.ch/api/v1"

// ResponseMeta contains metadata returned by the API alongside every response.
type ResponseMeta struct {
	RequestID        string `json:"request_id"`
	CreditsUsed      int    `json:"credits_used"`
	CreditsRemaining int    `json:"credits_remaining"`
	ProcessingTimeMs int    `json:"processing_time_ms"`
}

// Response wraps the metadata returned by the API for every successful call.
type Response struct {
	Meta ResponseMeta
}

// apiResponse is the top-level JSON envelope returned by the DFIR Lab API.
type apiResponse struct {
	Data json.RawMessage `json:"data"`
	Meta ResponseMeta    `json:"meta"`
}

// Client is the HTTP client for the DFIR Lab API.
type Client struct {
	httpClient     *http.Client
	streamClient   *http.Client // no timeout for SSE streams
	baseURL        string
	apiKey         string
	userAgent      string
	verbose        bool
	maxRetries     int
	retryBaseDelay time.Duration
}

// New creates a new API client. If baseURL is empty, the default production
// endpoint is used. The userAgent string is sent with every request.
func New(apiKey, baseURL, userAgent string, timeout time.Duration, verbose bool) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		httpClient:     &http.Client{Timeout: timeout},
		streamClient:   &http.Client{}, // no timeout for SSE streaming
		baseURL:        baseURL,
		apiKey:         apiKey,
		userAgent:      userAgent,
		verbose:        verbose,
		maxRetries:     3,
		retryBaseDelay: 1 * time.Second,
	}
}

// SetAPIKey replaces the current API key used for authentication.
func (c *Client) SetAPIKey(key string) {
	c.apiKey = key
}

// Do executes an HTTP request against the API with automatic retries on
// transient errors (429 and 5xx).
//
// method is the HTTP verb (GET, POST, etc.). path is appended to the base URL
// (e.g. "/enrichment/lookup"). If body is non-nil it is marshalled to JSON and
// sent as the request body. If result is non-nil the "data" field of the
// response envelope is decoded into it. The returned Response contains
// metadata such as request ID and credit usage.
func (c *Client) Do(ctx context.Context, method, path string, body interface{}, result interface{}) (*Response, error) {
	url := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	// Marshal request body once so it can be replayed on retries.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	// Verbose request logging — redact the API key for security.
	if c.verbose {
		redacted := redactKey(c.apiKey)
		fmt.Fprintf(os.Stderr, "[verbose] %s %s (auth: %s)\n", method, path, redacted)
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// On retries, wait with exponential backoff (or Retry-After).
		if attempt > 0 {
			// Check context before sleeping.
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("context cancelled before retry: %w", err)
			}

			delay := c.retryBaseDelay * (1 << (attempt - 1))

			// If the last error was a rate limit with Retry-After, use that.
			if rle, ok := lastErr.(*RateLimitError); ok && rle.RetryAfter > 0 {
				delay = rle.RetryAfter
			}

			if c.verbose {
				fmt.Fprintf(os.Stderr, "[verbose] retrying in %s (attempt %d/%d)\n",
					delay, attempt+1, c.maxRetries+1)
			}

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
			case <-timer.C:
			}
		}

		// Build request with a fresh body reader each attempt.
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		// Set headers.
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			// Network errors are not retried; they are typically context
			// cancellations or DNS/TLS failures.
			return nil, fmt.Errorf("execute request: %w", err)
		}

		// Check if we should retry.
		if c.isRetryable(resp.StatusCode) && attempt < c.maxRetries {
			lastErr = ParseError(resp) // consumes and closes body

			if c.verbose {
				statusDesc := http.StatusText(resp.StatusCode)
				if resp.StatusCode == http.StatusTooManyRequests {
					statusDesc = "rate limited"
				}
				retryDelay := c.retryBaseDelay * (1 << attempt)
				if rle, ok := lastErr.(*RateLimitError); ok && rle.RetryAfter > 0 {
					retryDelay = rle.RetryAfter
				}
				fmt.Fprintf(os.Stderr, "[verbose] %d %s, retrying in %s (attempt %d/%d)\n",
					resp.StatusCode, statusDesc, retryDelay, attempt+1, c.maxRetries)
			}
			continue
		}

		// Non-success and non-retryable (or exhausted retries).
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, ParseError(resp) // consumes and closes body
		}

		// Success — decode the response envelope.
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close() // close explicitly, not deferred inside loop
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		var envelope apiResponse
		if err := json.Unmarshal(respBody, &envelope); err != nil {
			return nil, fmt.Errorf("decode response envelope: %w", err)
		}

		// Verbose response logging.
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[verbose] %d %s (%dms, credits: %d used, %d remaining)\n",
				resp.StatusCode,
				http.StatusText(resp.StatusCode),
				elapsed.Milliseconds(),
				envelope.Meta.CreditsUsed,
				envelope.Meta.CreditsRemaining,
			)
		}

		// Decode the data payload into the caller's result, if requested.
		if result != nil && len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, result); err != nil {
				return nil, fmt.Errorf("decode response data: %w", err)
			}
		}

		return &Response{Meta: envelope.Meta}, nil
	}

	// All retries exhausted.
	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// DoRaw executes an HTTP request and returns the raw *http.Response without
// retries or JSON decoding. The caller is responsible for closing the response
// body. This is useful for streaming responses or cases where the caller needs
// full control over response handling.
func (c *Client) DoRaw(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers.
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.verbose {
		redacted := redactKey(c.apiKey)
		fmt.Fprintf(os.Stderr, "[verbose] %s %s (auth: %s) [raw]\n", method, path, redacted)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// DoStream executes an HTTP request intended for server-sent events (SSE).
// It uses a dedicated HTTP client with no timeout, since streams can run
// indefinitely. On success the raw *http.Response is returned and the caller
// is responsible for reading and closing the body. On non-2xx responses the
// body is consumed and a typed error is returned.
func (c *Client) DoStream(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers.
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.verbose {
		redacted := redactKey(c.apiKey)
		fmt.Fprintf(os.Stderr, "[verbose] %s %s (auth: %s) [stream]\n", method, path, redacted)
	}

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	// On non-success status, consume the body and return a typed error.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseError(resp) // consumes and closes body
	}

	return resp, nil
}

// isRetryable returns true for status codes that warrant an automatic retry.
// 429 (rate limit) and 5xx (server errors) are retryable. Client errors such
// as 400, 401, 402, 403, and 404 are never retried.
func (c *Client) isRetryable(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode >= 500
}

// redactKey returns a redacted version of an API key for safe verbose logging.
// Only the "sk-dfir-" prefix and last 4 characters are shown; the rest are
// replaced with asterisks. An empty key returns "<none>".
func redactKey(key string) string {
	if key == "" {
		return "<none>"
	}
	const prefix = "sk-dfir-"
	if strings.HasPrefix(key, prefix) && len(key) > len(prefix)+4 {
		return key[:len(prefix)] + "***..." + key[len(key)-4:]
	}
	// Non-standard key: show only last 4 chars.
	if len(key) > 4 {
		return "***..." + key[len(key)-4:]
	}
	return "***"
}
