package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AIChatStream sends a streaming AI chat request and returns a reader for SSE events.
func (c *Client) AIChatStream(ctx context.Context, req *AIChatRequest) (*AIChatStreamReader, error) {
	req.Stream = true
	resp, err := c.DoStream(ctx, http.MethodPost, "/ai/chat", req)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB per SSE line
	return &AIChatStreamReader{
		resp:    resp,
		scanner: scanner,
	}, nil
}

// AIChatStreamReader reads SSE events from a streaming AI chat response.
type AIChatStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	current AIChatStreamEvent
	err     error
}

// Next advances to the next SSE event. Returns false when the stream ends or errors.
func (r *AIChatStreamReader) Next() bool {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		// SSE format: lines starting with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// End of stream marker
		if data == "[DONE]" {
			return false
		}

		var event AIChatStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			r.err = fmt.Errorf("failed to parse SSE event: %w", err)
			return false
		}

		r.current = event

		// If it's an error event, store error and stop
		if event.Type == "error" {
			r.err = fmt.Errorf("AI service error: %s", event.Error)
			return false
		}

		return true
	}

	if err := r.scanner.Err(); err != nil {
		r.err = fmt.Errorf("stream read error: %w", err)
	}
	return false
}

// Event returns the current SSE event.
func (r *AIChatStreamReader) Event() AIChatStreamEvent {
	return r.current
}

// Err returns any error that occurred during streaming.
func (r *AIChatStreamReader) Err() error {
	return r.err
}

// Close releases the underlying HTTP response body.
func (r *AIChatStreamReader) Close() error {
	if r.resp != nil && r.resp.Body != nil {
		return r.resp.Body.Close()
	}
	return nil
}
