package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// helpers -------------------------------------------------------------------

// successEnvelope returns the canonical JSON envelope that the mock servers use.
func successEnvelope(data interface{}) []byte {
	d, _ := json.Marshal(data)
	env := map[string]interface{}{
		"data": json.RawMessage(d),
		"meta": map[string]interface{}{
			"request_id":        "test-123",
			"credits_used":      1,
			"credits_remaining": 99,
			"processing_time_ms": 50,
		},
	}
	b, _ := json.Marshal(env)
	return b
}

func errorEnvelope(typ, code, msg, reqID string) []byte {
	env := map[string]interface{}{
		"error": map[string]interface{}{
			"type":       typ,
			"code":       code,
			"message":    msg,
			"request_id": reqID,
		},
	}
	b, _ := json.Marshal(env)
	return b
}

// testClient creates a Client pointed at the given test server with minimal
// retry delays so tests run fast.
func testClient(ts *httptest.Server) *Client {
	c := New("sk-dfir-testapikey1234", ts.URL, "dfir-cli-test/1.0", 5*time.Second, false)
	c.retryBaseDelay = 10 * time.Millisecond // speed up retries in tests
	return c
}

// tests ---------------------------------------------------------------------

func TestDo_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"key": "value"}))
	}))
	defer ts.Close()

	c := testClient(ts)

	var result map[string]string
	resp, err := c.Do(context.Background(), http.MethodGet, "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Fatalf("expected data.key=value, got %q", result["key"])
	}
	if resp.Meta.RequestID != "test-123" {
		t.Fatalf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
	if resp.Meta.CreditsUsed != 1 {
		t.Fatalf("expected credits_used=1, got %d", resp.Meta.CreditsUsed)
	}
	if resp.Meta.CreditsRemaining != 99 {
		t.Fatalf("expected credits_remaining=99, got %d", resp.Meta.CreditsRemaining)
	}
	if resp.Meta.ProcessingTimeMs != 50 {
		t.Fatalf("expected processing_time_ms=50, got %d", resp.Meta.ProcessingTimeMs)
	}
}

func TestDo_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(errorEnvelope("authentication_error", "invalid_key", "invalid API key", "req-401"))
	}))
	defer ts.Close()

	c := testClient(ts)

	_, err := c.Do(context.Background(), http.MethodGet, "/secret", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthenticationError, got %T: %v", err, err)
	}
	if authErr.RequestID != "req-401" {
		t.Fatalf("expected request_id=req-401, got %q", authErr.RequestID)
	}
}

func TestDo_RetryOn429(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(errorEnvelope("rate_limit", "rate_limited", "slow down", "req-429"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := testClient(ts)

	var result map[string]string
	_, err := c.Do(context.Background(), http.MethodGet, "/rate", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["ok"] != "true" {
		t.Fatalf("expected ok=true, got %q", result["ok"])
	}
	total := atomic.LoadInt64(&hits)
	if total != 2 {
		t.Fatalf("expected 2 requests (1 retry), got %d", total)
	}
}

func TestDo_RetryOn500(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(errorEnvelope("server_error", "internal", "something broke", "req-500"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"recovered": "yes"}))
	}))
	defer ts.Close()

	c := testClient(ts)

	var result map[string]string
	_, err := c.Do(context.Background(), http.MethodGet, "/flaky", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["recovered"] != "yes" {
		t.Fatalf("expected recovered=yes, got %q", result["recovered"])
	}
	total := atomic.LoadInt64(&hits)
	if total != 2 {
		t.Fatalf("expected 2 requests (1 retry), got %d", total)
	}
}

func TestDo_NoRetryOn400(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorEnvelope("validation_error", "invalid_param", "bad field", "req-400"))
	}))
	defer ts.Close()

	c := testClient(ts)

	_, err := c.Do(context.Background(), http.MethodPost, "/validate", map[string]string{"bad": "data"}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	total := atomic.LoadInt64(&hits)
	if total != 1 {
		t.Fatalf("expected exactly 1 request (no retry), got %d", total)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow handler -- the request should never complete.
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := testClient(ts)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Do(ctx, http.MethodGet, "/slow", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDoRaw_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers are set.
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-dfir-testapikey1234" {
			t.Errorf("expected Authorization header, got %q", auth)
		}
		if ua := r.Header.Get("User-Agent"); ua != "dfir-cli-test/1.0" {
			t.Errorf("expected User-Agent header, got %q", ua)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("raw-binary-data"))
	}))
	defer ts.Close()

	c := testClient(ts)

	resp, err := c.DoRaw(context.Background(), http.MethodGet, "/download", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "raw-binary-data" {
		t.Fatalf("expected raw-binary-data, got %q", string(body))
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "standard key with prefix",
			key:  "sk-dfir-abcdefghijklmnop1234",
			want: "sk-dfir-***...1234",
		},
		{
			name: "empty key",
			key:  "",
			want: "<none>",
		},
		{
			name: "short non-standard key over 4 chars",
			key:  "short",
			want: "***...hort",
		},
		{
			name: "very short key 2 chars",
			key:  "ab",
			want: "***",
		},
		{
			name: "exactly 4 char non-standard key",
			key:  "abcd",
			want: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactKey(tt.key)
			if got != tt.want {
				t.Errorf("redactKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestNew_Defaults(t *testing.T) {
	c := New("key", "", "agent", 30*time.Second, false)
	if c.baseURL != defaultBaseURL {
		t.Fatalf("expected default baseURL %q, got %q", defaultBaseURL, c.baseURL)
	}

	custom := "https://custom.example.com/api"
	c2 := New("key", custom, "agent", 30*time.Second, true)
	if c2.baseURL != custom {
		t.Fatalf("expected custom baseURL %q, got %q", custom, c2.baseURL)
	}
	if !c2.verbose {
		t.Fatal("expected verbose=true")
	}
}

func TestSetAPIKey(t *testing.T) {
	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := testClient(ts)

	// Change the key.
	c.SetAPIKey("sk-dfir-newkey5678")

	_, err := c.Do(context.Background(), http.MethodGet, "/check", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Bearer sk-dfir-newkey5678"
	if receivedAuth != expected {
		t.Fatalf("expected Authorization %q, got %q", expected, receivedAuth)
	}
}

func TestDo_WithRequestBody(t *testing.T) {
	var receivedBody map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"echo": receivedBody["input"]}))
	}))
	defer ts.Close()

	c := testClient(ts)

	reqBody := map[string]string{"input": "hello"}
	var result map[string]string
	_, err := c.Do(context.Background(), http.MethodPost, "/echo", reqBody, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["echo"] != "hello" {
		t.Fatalf("expected echo=hello, got %q", result["echo"])
	}
}

func TestDo_VerboseLogging(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := testClient(ts)
	c.verbose = true

	// Just verify it doesn't panic with verbose logging enabled.
	_, err := c.Do(context.Background(), http.MethodGet, "/verbose", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoRaw_WithBody(t *testing.T) {
	var receivedBody map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := testClient(ts)

	resp, err := c.DoRaw(context.Background(), http.MethodPost, "/upload", map[string]string{"file": "data"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if receivedBody["file"] != "data" {
		t.Fatalf("expected file=data in body, got %v", receivedBody)
	}
}

func TestDo_NilResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ignored": "yes"}))
	}))
	defer ts.Close()

	c := testClient(ts)

	// Pass nil result -- should not error even though data is present.
	resp, err := c.Do(context.Background(), http.MethodGet, "/fire-and-forget", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.RequestID != "test-123" {
		t.Fatalf("expected request_id=test-123, got %q", resp.Meta.RequestID)
	}
}

func TestDo_ContextCancelledDuringRetryWait(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorEnvelope("server_error", "internal", "fail", "req-cancel"))
	}))
	defer ts.Close()

	c := testClient(ts)
	c.retryBaseDelay = 2 * time.Second // long enough to cancel during wait

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Do(ctx, http.MethodGet, "/cancel-during-wait", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// Could also be context.Canceled depending on timing -- just check non-nil.
		if ctx.Err() == nil {
			t.Fatalf("expected context error, got: %v", err)
		}
	}
}

func TestDo_VerboseRetryLogging(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(errorEnvelope("rate_limit", "rate_limited", "slow down", "req-verbose-retry"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := testClient(ts)
	c.verbose = true // exercise verbose retry logging branches

	_, err := c.Do(context.Background(), http.MethodGet, "/verbose-retry", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	total := atomic.LoadInt64(&hits)
	if total != 2 {
		t.Fatalf("expected 2 requests, got %d", total)
	}
}

func TestDo_VerboseRetryOn500(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(errorEnvelope("server_error", "internal", "oops", "req-v500"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := testClient(ts)
	c.verbose = true

	_, err := c.Do(context.Background(), http.MethodGet, "/verbose-500", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoRaw_VerboseNoKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no Authorization header when key is empty.
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := New("", ts.URL, "", 5*time.Second, true) // empty key, empty user-agent, verbose

	resp, err := c.DoRaw(context.Background(), http.MethodGet, "/noauth", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
}

func TestDo_NoAuthHeaderWhenKeyEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		if ua := r.Header.Get("User-Agent"); ua == "" {
			// Go's default user agent should still be set, or empty -- just don't crash.
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(successEnvelope(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	c := New("", ts.URL, "", 5*time.Second, false)

	_, err := c.Do(context.Background(), http.MethodGet, "/nokey", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDo_RetriesExhausted(t *testing.T) {
	var hits int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorEnvelope("server_error", "internal", "always broken", "req-exhaust"))
	}))
	defer ts.Close()

	c := testClient(ts)

	_, err := c.Do(context.Background(), http.MethodGet, "/always-fail", nil, nil)
	if err == nil {
		t.Fatal("expected error after exhausted retries, got nil")
	}

	// maxRetries is 3, so 1 initial + 3 retries = 4 total requests.
	total := atomic.LoadInt64(&hits)
	if total != 4 {
		t.Fatalf("expected 4 total requests (initial + 3 retries), got %d", total)
	}
}
