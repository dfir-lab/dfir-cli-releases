package commands

import (
	"testing"
)

func TestExitError_Error(t *testing.T) {
	e := &ExitError{Code: 1, Message: "something went wrong"}
	if got := e.Error(); got != "something went wrong" {
		t.Errorf("ExitError.Error() = %q, want %q", got, "something went wrong")
	}
}

func TestNewExitError(t *testing.T) {
	e := NewExitError(42, "bad input")
	if e.Code != 42 {
		t.Errorf("NewExitError().Code = %d, want 42", e.Code)
	}
	if e.Message != "bad input" {
		t.Errorf("NewExitError().Message = %q, want %q", e.Message, "bad input")
	}
}

func TestNewExitErrorf(t *testing.T) {
	e := NewExitErrorf(3, "file %s not found (attempt %d)", "config.yaml", 2)
	wantMsg := "file config.yaml not found (attempt 2)"
	if e.Code != 3 {
		t.Errorf("NewExitErrorf().Code = %d, want 3", e.Code)
	}
	if e.Message != wantMsg {
		t.Errorf("NewExitErrorf().Message = %q, want %q", e.Message, wantMsg)
	}
}

func TestSilentExitError_Error(t *testing.T) {
	e := &SilentExitError{Code: 1}
	if got := e.Error(); got != "" {
		t.Errorf("SilentExitError.Error() = %q, want empty string", got)
	}
}

func TestExitError_ImplementsError(t *testing.T) {
	var _ error = (*ExitError)(nil)
}

func TestSilentExitError_ImplementsError(t *testing.T) {
	var _ error = (*SilentExitError)(nil)
}
