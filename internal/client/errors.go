package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// AuthenticationError is returned when the API responds with 401 Unauthorized.
type AuthenticationError struct {
	Message   string
	RequestID string
}

func (e *AuthenticationError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "invalid API key"
	}
	return fmt.Sprintf("authentication failed: %s. Run: dfir-cli config init", msg)
}

// AuthorizationError is returned when the API responds with 403 Forbidden.
type AuthorizationError struct {
	Message   string
	RequestID string
}

func (e *AuthorizationError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "access denied"
	}
	return fmt.Sprintf("permission denied: %s", msg)
}

// InsufficientCreditsError is returned when the API responds with 402 Payment Required.
type InsufficientCreditsError struct {
	Message   string
	RequestID string
}

func (e *InsufficientCreditsError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "not enough credits to complete this request"
	}
	return fmt.Sprintf("insufficient credits: %s\n  Add credits: https://dfir-lab.ch/billing\n  Check balance: dfir-cli credits", msg)
}

// ValidationError is returned when the API responds with 400 Bad Request.
type ValidationError struct {
	Message   string
	Code      string
	RequestID string
}

func (e *ValidationError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "bad request"
	}
	return fmt.Sprintf("invalid request: %s", msg)
}

// RateLimitError is returned when the API responds with 429 Too Many Requests.
type RateLimitError struct {
	Message    string
	RetryAfter time.Duration
	RequestID  string
}

func (e *RateLimitError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "too many requests"
	}
	return fmt.Sprintf("rate limited: %s. Retry after %.0fs", msg, e.RetryAfter.Seconds())
}

// NotFoundError is returned when the API responds with 404 Not Found.
type NotFoundError struct {
	Message   string
	RequestID string
}

func (e *NotFoundError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "resource not found"
	}
	return fmt.Sprintf("not found: %s", msg)
}

// APIError is a generic error for any non-success API response not covered by
// a more specific type.
type APIError struct {
	StatusCode int
	Type       string
	Code       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	s := fmt.Sprintf("API error (%d): %s", e.StatusCode, msg)
	if e.RequestID != "" {
		s += fmt.Sprintf(" [request_id=%s]", e.RequestID)
	}
	return s
}

// apiErrorResponse is the expected JSON envelope returned by the API on errors.
// The "error" field can be either a structured object or a plain string.
type apiErrorResponse struct {
	Error apiErrorDetail `json:"error"`
}

type apiErrorDetail struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// apiErrorStringResponse handles the backwards-compatible format where "error"
// is a plain string rather than an object.
type apiErrorStringResponse struct {
	Error string `json:"error"`
}

// ParseError reads the response body and returns a typed error based on the
// HTTP status code and the error type from the JSON payload.
// The response body is always fully consumed and closed.
func ParseError(resp *http.Response) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	var detail apiErrorDetail

	// Try structured format first: {"error": {"type": "...", ...}}
	var structured apiErrorResponse
	if err := json.Unmarshal(body, &structured); err == nil && structured.Error.Type != "" {
		detail = structured.Error
	} else {
		// Fall back to simple string format: {"error": "message"}
		var simple apiErrorStringResponse
		if err := json.Unmarshal(body, &simple); err == nil && simple.Error != "" {
			detail.Message = simple.Error
		}
	}

	// If we still have no message, use the HTTP status text.
	if detail.Message == "" {
		detail.Message = http.StatusText(resp.StatusCode)
	}

	// Parse Retry-After header for 429 responses.
	var retryAfter time.Duration
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter = 60 * time.Second // sensible default
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &AuthenticationError{
			Message:   detail.Message,
			RequestID: detail.RequestID,
		}

	case http.StatusForbidden:
		return &AuthorizationError{
			Message:   detail.Message,
			RequestID: detail.RequestID,
		}

	case http.StatusPaymentRequired:
		return &InsufficientCreditsError{
			Message:   detail.Message,
			RequestID: detail.RequestID,
		}

	case http.StatusBadRequest:
		return &ValidationError{
			Message:   detail.Message,
			Code:      detail.Code,
			RequestID: detail.RequestID,
		}

	case http.StatusTooManyRequests:
		return &RateLimitError{
			Message:    detail.Message,
			RetryAfter: retryAfter,
			RequestID:  detail.RequestID,
		}

	case http.StatusNotFound:
		return &NotFoundError{
			Message:   detail.Message,
			RequestID: detail.RequestID,
		}

	default:
		return &APIError{
			StatusCode: resp.StatusCode,
			Type:       detail.Type,
			Code:       detail.Code,
			Message:    detail.Message,
			RequestID:  detail.RequestID,
		}
	}
}

// IsRetryable returns true if the error is a rate limit error or a server-side
// error (status >= 500) that may succeed on retry.
func IsRetryable(err error) bool {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
		return true
	}

	return false
}

// IsAuthError returns true if the error is an authentication failure.
func IsAuthError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr)
}

// IsCreditsError returns true if the error is an insufficient credits error.
func IsCreditsError(err error) bool {
	var creditsErr *InsufficientCreditsError
	return errors.As(err, &creditsErr)
}
