package commands

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dfir-lab/dfir-cli/internal/client"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = orig
	})

	fn()

	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stderr): %v", err)
	}
	_ = r.Close()
	return string(data)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = orig
	})

	fn()

	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout): %v", err)
	}
	_ = r.Close()
	return string(data)
}

func TestHandleAIError_NotFound(t *testing.T) {
	t.Setenv("DFIR_LAB_API_URL", "https://platform.dfir-lab.ch/api/v1")

	var got error
	stderr := captureStderr(t, func() {
		got = handleAIError(&client.NotFoundError{Message: "Not Found"})
	})

	var silentErr *SilentExitError
	if !errors.As(got, &silentErr) {
		t.Fatalf("handleAIError() = %T, want *SilentExitError", got)
	}
	if silentErr.Code != 1 {
		t.Fatalf("SilentExitError.Code = %d, want 1", silentErr.Code)
	}
	if !strings.Contains(stderr, "AI chat is not available on the configured DFIR Platform API endpoint") {
		t.Fatalf("stderr missing AI availability message:\n%s", stderr)
	}
	if !strings.Contains(stderr, "https://api.dfir-lab.ch/v1") {
		t.Fatalf("stderr missing API URL:\n%s", stderr)
	}
}

func TestHandleAIChatREPLError_ContextCanceled(t *testing.T) {
	if err := handleAIChatREPLError(context.Canceled); err != nil {
		t.Fatalf("handleAIChatREPLError(context.Canceled) = %v, want nil", err)
	}
}

func TestLocalAIIdentityDisclosureResponse(t *testing.T) {
	response, ok := localAIIdentityDisclosureResponse("hello what model are you exactly?")
	if !ok {
		t.Fatal("expected identity disclosure match, got false")
	}
	if response != aiIdentityDisclosureResponse {
		t.Fatalf("unexpected response:\n%s", response)
	}

	if _, ok := localAIIdentityDisclosureResponse("What event IDs indicate lateral movement?"); ok {
		t.Fatal("unexpected identity disclosure match for DFIR question")
	}
}

func TestRunAIOneShot_IdentityDisclosureShortCircuitsWithoutAPIKey(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", t.TempDir())
	t.Setenv("DFIR_LAB_API_KEY", "")
	_ = rootCmd.PersistentFlags().Lookup("output").Value.Set("table")

	stdout := captureStdout(t, func() {
		if err := runAIOneShot("What model are you exactly?", "", false); err != nil {
			t.Fatalf("runAIOneShot returned error: %v", err)
		}
	})

	if !strings.Contains(stdout, aiIdentityDisclosureResponse) {
		t.Fatalf("stdout missing identity disclosure response:\n%s", stdout)
	}
}

func TestRunAIOneShot_IdentityDisclosureJSON(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", t.TempDir())
	t.Setenv("DFIR_LAB_API_KEY", "")
	_ = rootCmd.PersistentFlags().Lookup("output").Value.Set("json")
	t.Cleanup(func() {
		_ = rootCmd.PersistentFlags().Lookup("output").Value.Set("table")
	})

	stdout := captureStdout(t, func() {
		if err := runAIOneShot("Who are you?", "", false); err != nil {
			t.Fatalf("runAIOneShot returned error: %v", err)
		}
	})

	var payload map[string]string
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("identity disclosure JSON invalid: %v\n%s", err, stdout)
	}
	if payload["response"] != aiIdentityDisclosureResponse {
		t.Fatalf("response = %q, want exact disclosure text", payload["response"])
	}
}
