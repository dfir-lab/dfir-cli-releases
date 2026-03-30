package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mockResp builds an *http.Response with the given status code, body, and
// optional headers (key/value pairs).
func mockResp(status int, body string, headers ...string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for i := 0; i+1 < len(headers); i += 2 {
		resp.Header.Set(headers[i], headers[i+1])
	}
	return resp
}

// failReader is an io.Reader that always returns an error.
type failReader struct{}

func (f *failReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

// ---------------------------------------------------------------------------
// Error() message tests
// ---------------------------------------------------------------------------

func TestAuthenticationError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &AuthenticationError{}
		got := e.Error()
		if !strings.Contains(got, "invalid API key") {
			t.Errorf("expected default message, got: %s", got)
		}
		if !strings.Contains(got, "dfir-cli config init") {
			t.Errorf("expected hint about config init, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &AuthenticationError{Message: "token expired"}
		got := e.Error()
		if !strings.Contains(got, "token expired") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})
}

func TestAuthorizationError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &AuthorizationError{}
		if got := e.Error(); !strings.Contains(got, "access denied") {
			t.Errorf("expected default message, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &AuthorizationError{Message: "admin only"}
		if got := e.Error(); !strings.Contains(got, "admin only") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})
}

func TestInsufficientCreditsError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &InsufficientCreditsError{}
		got := e.Error()
		if !strings.Contains(got, "not enough credits") {
			t.Errorf("expected default message, got: %s", got)
		}
		if !strings.Contains(got, "dfir-lab.ch/billing") {
			t.Errorf("expected billing URL, got: %s", got)
		}
		if !strings.Contains(got, "dfir-cli credits") {
			t.Errorf("expected credits hint, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &InsufficientCreditsError{Message: "0 credits remaining"}
		if got := e.Error(); !strings.Contains(got, "0 credits remaining") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})
}

func TestValidationError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &ValidationError{}
		if got := e.Error(); !strings.Contains(got, "bad request") {
			t.Errorf("expected default message, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &ValidationError{Message: "missing field: name"}
		if got := e.Error(); !strings.Contains(got, "missing field: name") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})
}

func TestRateLimitError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &RateLimitError{RetryAfter: 30 * time.Second}
		got := e.Error()
		if !strings.Contains(got, "too many requests") {
			t.Errorf("expected default message, got: %s", got)
		}
		if !strings.Contains(got, "30s") {
			t.Errorf("expected retry duration, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &RateLimitError{Message: "slow down", RetryAfter: 10 * time.Second}
		got := e.Error()
		if !strings.Contains(got, "slow down") {
			t.Errorf("expected custom message, got: %s", got)
		}
		if !strings.Contains(got, "10s") {
			t.Errorf("expected retry duration, got: %s", got)
		}
	})
}

func TestNotFoundError_Error(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		e := &NotFoundError{}
		if got := e.Error(); !strings.Contains(got, "resource not found") {
			t.Errorf("expected default message, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &NotFoundError{Message: "sample abc123 not found"}
		if got := e.Error(); !strings.Contains(got, "sample abc123 not found") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})
}

func TestAPIError_Error(t *testing.T) {
	t.Run("default message from status", func(t *testing.T) {
		e := &APIError{StatusCode: 503}
		got := e.Error()
		if !strings.Contains(got, "503") {
			t.Errorf("expected status code, got: %s", got)
		}
		if !strings.Contains(got, "Service Unavailable") {
			t.Errorf("expected status text, got: %s", got)
		}
	})

	t.Run("custom message", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Message: "internal explosion"}
		if got := e.Error(); !strings.Contains(got, "internal explosion") {
			t.Errorf("expected custom message, got: %s", got)
		}
	})

	t.Run("with request ID", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Message: "boom", RequestID: "req-999"}
		got := e.Error()
		if !strings.Contains(got, "req-999") {
			t.Errorf("expected request ID, got: %s", got)
		}
	})

	t.Run("without request ID", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Message: "boom"}
		got := e.Error()
		if strings.Contains(got, "request_id") {
			t.Errorf("should not include request_id tag, got: %s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// ParseError tests — status code routing
// ---------------------------------------------------------------------------

func TestParseError_StatusCodes(t *testing.T) {
	tests := []struct {
		status   int
		wantType interface{}
	}{
		{http.StatusUnauthorized, &AuthenticationError{}},
		{http.StatusForbidden, &AuthorizationError{}},
		{http.StatusPaymentRequired, &InsufficientCreditsError{}},
		{http.StatusBadRequest, &ValidationError{}},
		{http.StatusTooManyRequests, &RateLimitError{}},
		{http.StatusNotFound, &NotFoundError{}},
		{http.StatusInternalServerError, &APIError{}},
		{http.StatusBadGateway, &APIError{}},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			resp := mockResp(tc.status, `{"error":"test msg"}`)
			err := ParseError(resp)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			// Verify the returned error type matches what we expect.
			got := fmt.Sprintf("%T", err)
			want := fmt.Sprintf("%T", tc.wantType)
			if got != want {
				t.Errorf("expected type %s, got %s", want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseError — structured JSON body
// ---------------------------------------------------------------------------

func TestParseError_StructuredJSON(t *testing.T) {
	body := `{"error":{"type":"auth_error","message":"invalid key","request_id":"req-123"}}`
	resp := mockResp(http.StatusUnauthorized, body)

	err := ParseError(resp)

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthenticationError, got %T: %v", err, err)
	}
	if authErr.Message != "invalid key" {
		t.Errorf("expected message 'invalid key', got %q", authErr.Message)
	}
	if authErr.RequestID != "req-123" {
		t.Errorf("expected request_id 'req-123', got %q", authErr.RequestID)
	}
}

func TestParseError_StructuredWithCode(t *testing.T) {
	body := `{"error":{"type":"validation_error","code":"MISSING_FIELD","message":"field 'name' is required","request_id":"req-456"}}`
	resp := mockResp(http.StatusBadRequest, body)

	err := ParseError(resp)

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Code != "MISSING_FIELD" {
		t.Errorf("expected code 'MISSING_FIELD', got %q", valErr.Code)
	}
	if valErr.Message != "field 'name' is required" {
		t.Errorf("expected message about required field, got %q", valErr.Message)
	}
	if valErr.RequestID != "req-456" {
		t.Errorf("expected request_id 'req-456', got %q", valErr.RequestID)
	}
}

func TestParseError_GenericWithAllFields(t *testing.T) {
	body := `{"error":{"type":"server_error","code":"INTERNAL","message":"unexpected failure","request_id":"req-abc"}}`
	resp := mockResp(http.StatusInternalServerError, body)

	err := ParseError(resp)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Type != "server_error" {
		t.Errorf("expected type 'server_error', got %q", apiErr.Type)
	}
	if apiErr.Code != "INTERNAL" {
		t.Errorf("expected code 'INTERNAL', got %q", apiErr.Code)
	}
	if apiErr.Message != "unexpected failure" {
		t.Errorf("expected message 'unexpected failure', got %q", apiErr.Message)
	}
	if apiErr.RequestID != "req-abc" {
		t.Errorf("expected request_id 'req-abc', got %q", apiErr.RequestID)
	}
}

// ---------------------------------------------------------------------------
// ParseError — simple string JSON body
// ---------------------------------------------------------------------------

func TestParseError_SimpleStringJSON(t *testing.T) {
	body := `{"error":"something went wrong"}`
	resp := mockResp(http.StatusBadRequest, body)

	err := ParseError(resp)

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Message != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %q", valErr.Message)
	}
}

// ---------------------------------------------------------------------------
// ParseError — empty / malformed body
// ---------------------------------------------------------------------------

func TestParseError_EmptyBody(t *testing.T) {
	resp := mockResp(http.StatusForbidden, "")

	err := ParseError(resp)

	var authzErr *AuthorizationError
	if !errors.As(err, &authzErr) {
		t.Fatalf("expected *AuthorizationError, got %T: %v", err, err)
	}
	// Should fall back to HTTP status text.
	if authzErr.Message != "Forbidden" {
		t.Errorf("expected fallback message 'Forbidden', got %q", authzErr.Message)
	}
}

func TestParseError_MalformedBody(t *testing.T) {
	resp := mockResp(http.StatusNotFound, "this is not json at all")

	err := ParseError(resp)

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Message != "Not Found" {
		t.Errorf("expected fallback 'Not Found', got %q", nfErr.Message)
	}
}

func TestParseError_ReadBodyError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{},
		Body:       io.NopCloser(&failReader{}),
	}

	err := ParseError(resp)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !strings.Contains(apiErr.Message, "failed to read error response") {
		t.Errorf("expected read failure message, got %q", apiErr.Message)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// ParseError — 429 with Retry-After header
// ---------------------------------------------------------------------------

func TestParseError_429WithRetryAfterHeader(t *testing.T) {
	body := `{"error":"rate limit hit"}`
	resp := mockResp(http.StatusTooManyRequests, body, "Retry-After", "120")

	err := ParseError(resp)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 120*time.Second {
		t.Errorf("expected RetryAfter 120s, got %v", rlErr.RetryAfter)
	}
	if rlErr.Message != "rate limit hit" {
		t.Errorf("expected message 'rate limit hit', got %q", rlErr.Message)
	}
}

func TestParseError_429WithoutRetryAfterHeader(t *testing.T) {
	body := `{"error":"rate limit"}`
	resp := mockResp(http.StatusTooManyRequests, body)

	err := ParseError(resp)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	// Default should be 60 seconds.
	if rlErr.RetryAfter != 60*time.Second {
		t.Errorf("expected default RetryAfter 60s, got %v", rlErr.RetryAfter)
	}
}

func TestParseError_429WithInvalidRetryAfter(t *testing.T) {
	body := `{"error":"rate limit"}`
	resp := mockResp(http.StatusTooManyRequests, body, "Retry-After", "not-a-number")

	err := ParseError(resp)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	// Should fall back to default 60s.
	if rlErr.RetryAfter != 60*time.Second {
		t.Errorf("expected default RetryAfter 60s, got %v", rlErr.RetryAfter)
	}
}

// ---------------------------------------------------------------------------
// IsRetryable tests
// ---------------------------------------------------------------------------

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "RateLimitError is retryable",
			err:  &RateLimitError{Message: "slow down", RetryAfter: 30 * time.Second},
			want: true,
		},
		{
			name: "APIError 500 is retryable",
			err:  &APIError{StatusCode: 500, Message: "server error"},
			want: true,
		},
		{
			name: "APIError 502 is retryable",
			err:  &APIError{StatusCode: 502, Message: "bad gateway"},
			want: true,
		},
		{
			name: "APIError 503 is retryable",
			err:  &APIError{StatusCode: 503, Message: "service unavailable"},
			want: true,
		},
		{
			name: "APIError 400 is not retryable",
			err:  &APIError{StatusCode: 400, Message: "bad request"},
			want: false,
		},
		{
			name: "APIError 404 is not retryable",
			err:  &APIError{StatusCode: 404, Message: "not found"},
			want: false,
		},
		{
			name: "AuthenticationError is not retryable",
			err:  &AuthenticationError{Message: "invalid key"},
			want: false,
		},
		{
			name: "AuthorizationError is not retryable",
			err:  &AuthorizationError{Message: "forbidden"},
			want: false,
		},
		{
			name: "InsufficientCreditsError is not retryable",
			err:  &InsufficientCreditsError{Message: "no credits"},
			want: false,
		},
		{
			name: "ValidationError is not retryable",
			err:  &ValidationError{Message: "bad field"},
			want: false,
		},
		{
			name: "NotFoundError is not retryable",
			err:  &NotFoundError{Message: "missing"},
			want: false,
		},
		{
			name: "generic error is not retryable",
			err:  errors.New("random error"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRetryable(tc.err); got != tc.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsAuthError tests
// ---------------------------------------------------------------------------

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "AuthenticationError is auth error",
			err:  &AuthenticationError{Message: "bad key"},
			want: true,
		},
		{
			name: "AuthorizationError is not auth error",
			err:  &AuthorizationError{Message: "forbidden"},
			want: false,
		},
		{
			name: "APIError is not auth error",
			err:  &APIError{StatusCode: 401, Message: "unauthorized"},
			want: false,
		},
		{
			name: "generic error is not auth error",
			err:  errors.New("something"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAuthError(tc.err); got != tc.want {
				t.Errorf("IsAuthError() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsCreditsError tests
// ---------------------------------------------------------------------------

func TestIsCreditsError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "InsufficientCreditsError is credits error",
			err:  &InsufficientCreditsError{Message: "empty"},
			want: true,
		},
		{
			name: "AuthenticationError is not credits error",
			err:  &AuthenticationError{Message: "bad"},
			want: false,
		},
		{
			name: "APIError 402 is not credits error",
			err:  &APIError{StatusCode: 402, Message: "payment"},
			want: false,
		},
		{
			name: "generic error is not credits error",
			err:  errors.New("nope"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCreditsError(tc.err); got != tc.want {
				t.Errorf("IsCreditsError() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseError — malformed JSON body
// ---------------------------------------------------------------------------

func TestParseError_MalformedJSON(t *testing.T) {
	resp := mockResp(http.StatusBadRequest, `{"error": {"type": "broken`)

	err := ParseError(resp)

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	// Should fall back to HTTP status text since JSON is malformed.
	if valErr.Message != "Bad Request" {
		t.Errorf("expected fallback 'Bad Request', got %q", valErr.Message)
	}
}

// ---------------------------------------------------------------------------
// ParseError — string error format (not object)
// ---------------------------------------------------------------------------

func TestParseError_StringErrorFormat(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantType   string
		wantMsg    string
	}{
		{
			name:     "401 with string error",
			status:   http.StatusUnauthorized,
			body:     `{"error":"key is invalid"}`,
			wantType: "*client.AuthenticationError",
			wantMsg:  "key is invalid",
		},
		{
			name:     "403 with string error",
			status:   http.StatusForbidden,
			body:     `{"error":"no permission"}`,
			wantType: "*client.AuthorizationError",
			wantMsg:  "no permission",
		},
		{
			name:     "402 with string error",
			status:   http.StatusPaymentRequired,
			body:     `{"error":"out of credits"}`,
			wantType: "*client.InsufficientCreditsError",
			wantMsg:  "out of credits",
		},
		{
			name:     "429 with string error",
			status:   http.StatusTooManyRequests,
			body:     `{"error":"rate limit hit"}`,
			wantType: "*client.RateLimitError",
			wantMsg:  "rate limit hit",
		},
		{
			name:     "404 with string error",
			status:   http.StatusNotFound,
			body:     `{"error":"item not found"}`,
			wantType: "*client.NotFoundError",
			wantMsg:  "item not found",
		},
		{
			name:     "503 with string error",
			status:   http.StatusServiceUnavailable,
			body:     `{"error":"service unavailable"}`,
			wantType: "*client.APIError",
			wantMsg:  "service unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := mockResp(tc.status, tc.body)
			err := ParseError(resp)
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			got := fmt.Sprintf("%T", err)
			if got != tc.wantType {
				t.Errorf("expected type %s, got %s", tc.wantType, got)
			}

			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("expected message to contain %q, got %q", tc.wantMsg, err.Error())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error messages contain actionable hints
// ---------------------------------------------------------------------------

func TestErrorMessages_ActionableHints(t *testing.T) {
	t.Run("AuthenticationError has config init hint", func(t *testing.T) {
		e := &AuthenticationError{Message: "expired key"}
		got := e.Error()
		if !strings.Contains(got, "dfir-cli config init") {
			t.Errorf("expected hint about config init, got: %s", got)
		}
	})

	t.Run("InsufficientCreditsError has billing URL and credits hint", func(t *testing.T) {
		e := &InsufficientCreditsError{}
		got := e.Error()
		if !strings.Contains(got, "https://platform.dfir-lab.ch/billing") {
			t.Errorf("expected billing URL, got: %s", got)
		}
		if !strings.Contains(got, "dfir-cli credits") {
			t.Errorf("expected credits command hint, got: %s", got)
		}
	})

	t.Run("RateLimitError includes retry duration", func(t *testing.T) {
		e := &RateLimitError{RetryAfter: 45 * time.Second}
		got := e.Error()
		if !strings.Contains(got, "45s") {
			t.Errorf("expected 45s in message, got: %s", got)
		}
	})

	t.Run("APIError with request ID includes it in output", func(t *testing.T) {
		e := &APIError{StatusCode: 500, Message: "crash", RequestID: "req-id-xyz"}
		got := e.Error()
		if !strings.Contains(got, "req-id-xyz") {
			t.Errorf("expected request ID in message, got: %s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// All HTTP status codes mapped correctly
// ---------------------------------------------------------------------------

func TestParseError_AllStatusCodesMapped(t *testing.T) {
	tests := []struct {
		status   int
		wantType string
	}{
		{401, "*client.AuthenticationError"},
		{402, "*client.InsufficientCreditsError"},
		{403, "*client.AuthorizationError"},
		{404, "*client.NotFoundError"},
		{429, "*client.RateLimitError"},
		{500, "*client.APIError"},
		{502, "*client.APIError"},
		{503, "*client.APIError"},
		{504, "*client.APIError"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			body := `{"error":{"type":"test","message":"test msg","request_id":"req-map"}}`
			resp := mockResp(tc.status, body)
			err := ParseError(resp)
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			got := fmt.Sprintf("%T", err)
			if got != tc.wantType {
				t.Errorf("status %d: expected type %s, got %s", tc.status, tc.wantType, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IsRetryable with wrapped errors
// ---------------------------------------------------------------------------

func TestIsRetryable_WrappedErrors(t *testing.T) {
	t.Run("wrapped RateLimitError is retryable", func(t *testing.T) {
		inner := &RateLimitError{Message: "slow down", RetryAfter: 10 * time.Second}
		wrapped := fmt.Errorf("something went wrong: %w", inner)
		if !IsRetryable(wrapped) {
			t.Error("expected wrapped RateLimitError to be retryable")
		}
	})

	t.Run("wrapped APIError 500 is retryable", func(t *testing.T) {
		inner := &APIError{StatusCode: 500, Message: "crash"}
		wrapped := fmt.Errorf("outer: %w", inner)
		if !IsRetryable(wrapped) {
			t.Error("expected wrapped APIError 500 to be retryable")
		}
	})

	t.Run("wrapped APIError 400 is not retryable", func(t *testing.T) {
		inner := &APIError{StatusCode: 400, Message: "bad"}
		wrapped := fmt.Errorf("outer: %w", inner)
		if IsRetryable(wrapped) {
			t.Error("expected wrapped APIError 400 to not be retryable")
		}
	})

	t.Run("nil error is not retryable", func(t *testing.T) {
		if IsRetryable(nil) {
			t.Error("expected nil error to not be retryable")
		}
	})
}

// ---------------------------------------------------------------------------
// IsAuthError and IsCreditsError with wrapped errors
// ---------------------------------------------------------------------------

func TestIsAuthError_Wrapped(t *testing.T) {
	inner := &AuthenticationError{Message: "bad key"}
	wrapped := fmt.Errorf("outer: %w", inner)
	if !IsAuthError(wrapped) {
		t.Error("expected wrapped AuthenticationError to be detected")
	}
}

func TestIsCreditsError_Wrapped(t *testing.T) {
	inner := &InsufficientCreditsError{Message: "empty"}
	wrapped := fmt.Errorf("outer: %w", inner)
	if !IsCreditsError(wrapped) {
		t.Error("expected wrapped InsufficientCreditsError to be detected")
	}
}

// ---------------------------------------------------------------------------
// ParseError — 429 with zero Retry-After
// ---------------------------------------------------------------------------

func TestParseError_429WithZeroRetryAfter(t *testing.T) {
	body := `{"error":"rate limit"}`
	resp := mockResp(http.StatusTooManyRequests, body, "Retry-After", "0")

	err := ParseError(resp)

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 0 {
		t.Errorf("expected RetryAfter 0, got %v", rlErr.RetryAfter)
	}
}

// ---------------------------------------------------------------------------
// ParseError — empty body for various status codes
// ---------------------------------------------------------------------------

func TestParseError_EmptyBodyVariousCodes(t *testing.T) {
	tests := []struct {
		status      int
		wantType    string
		wantFallback string
	}{
		{401, "*client.AuthenticationError", "Unauthorized"},
		{402, "*client.InsufficientCreditsError", "Payment Required"},
		{404, "*client.NotFoundError", "Not Found"},
		{500, "*client.APIError", "Internal Server Error"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("empty_body_%d", tc.status), func(t *testing.T) {
			resp := mockResp(tc.status, "")
			err := ParseError(resp)
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			got := fmt.Sprintf("%T", err)
			if got != tc.wantType {
				t.Errorf("expected type %s, got %s", tc.wantType, got)
			}
		})
	}
}
