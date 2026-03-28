package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ForeGuards/dfir-cli/internal/client"
)

// newTestMeta returns a ResponseMeta suitable for tests.
func newTestMeta() *client.ResponseMeta {
	return &client.ResponseMeta{
		RequestID:        "req-test-123",
		CreditsUsed:      5,
		CreditsRemaining: 95,
		ProcessingTimeMs: 200,
	}
}

func TestSaveCreditState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Pin time so the test is deterministic.
	origTimeNow := timeNowUTC
	timeNowUTC = func() string { return "2026-01-15T10:30:00Z" }
	t.Cleanup(func() { timeNowUTC = origTimeNow })

	meta := newTestMeta()

	if err := SaveCreditState(meta); err != nil {
		t.Fatalf("SaveCreditState returned error: %v", err)
	}

	path := filepath.Join(tmpDir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("state file not found: %v", err)
	}

	var state creditState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	if state.CreditsRemaining != 95 {
		t.Errorf("CreditsRemaining = %d, want 95", state.CreditsRemaining)
	}
	if state.LastCreditsUsed != 5 {
		t.Errorf("LastCreditsUsed = %d, want 5", state.LastCreditsUsed)
	}
	if state.LastRequestID != "req-test-123" {
		t.Errorf("LastRequestID = %q, want %q", state.LastRequestID, "req-test-123")
	}
	if state.LastRequestAt != "2026-01-15T10:30:00Z" {
		t.Errorf("LastRequestAt = %q, want %q", state.LastRequestAt, "2026-01-15T10:30:00Z")
	}
}

func TestLoadCreditState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	want := creditState{
		CreditsRemaining: 42,
		LastCreditsUsed:  8,
		LastRequestAt:    "2026-03-01T08:00:00Z",
		LastRequestID:    "req-load-456",
	}

	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("marshal test state: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, stateFileName), data, 0600); err != nil {
		t.Fatalf("write test state file: %v", err)
	}

	got, err := LoadCreditState()
	if err != nil {
		t.Fatalf("LoadCreditState returned error: %v", err)
	}

	if got.CreditsRemaining != want.CreditsRemaining {
		t.Errorf("CreditsRemaining = %d, want %d", got.CreditsRemaining, want.CreditsRemaining)
	}
	if got.LastCreditsUsed != want.LastCreditsUsed {
		t.Errorf("LastCreditsUsed = %d, want %d", got.LastCreditsUsed, want.LastCreditsUsed)
	}
	if got.LastRequestAt != want.LastRequestAt {
		t.Errorf("LastRequestAt = %q, want %q", got.LastRequestAt, want.LastRequestAt)
	}
	if got.LastRequestID != want.LastRequestID {
		t.Errorf("LastRequestID = %q, want %q", got.LastRequestID, want.LastRequestID)
	}
}

func TestLoadCreditState_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	state, err := LoadCreditState()
	if err == nil {
		t.Fatal("LoadCreditState should return an error when the state file is missing")
	}
	if state != nil {
		t.Errorf("expected nil state, got %+v", state)
	}
}

func TestSaveCreditState_NilMeta(t *testing.T) {
	// Passing nil should be a no-op and return nil.
	if err := SaveCreditState(nil); err != nil {
		t.Fatalf("SaveCreditState(nil) returned error: %v", err)
	}
}

func TestWriteCreditState_Atomic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	state := &creditState{
		CreditsRemaining: 50,
		LastCreditsUsed:  10,
		LastRequestAt:    "2026-02-20T12:00:00Z",
		LastRequestID:    "req-atomic-789",
	}

	if err := writeCreditState(state); err != nil {
		t.Fatalf("writeCreditState returned error: %v", err)
	}

	// The final state file must exist.
	finalPath := filepath.Join(tmpDir, stateFileName)
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("state file missing after write: %v", err)
	}

	// The temporary file used during atomic write must have been cleaned up.
	tmpPath := finalPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q still exists after successful write", tmpPath)
	}
}
