package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestShort(t *testing.T) {
	// Default value
	if got := Short(); got != "dev" {
		t.Errorf("Short() = %q, want %q", got, "dev")
	}
}

func TestBuildInfo(t *testing.T) {
	info := BuildInfo()
	if !strings.Contains(info, "dfir-cli") {
		t.Errorf("BuildInfo() should contain 'dfir-cli', got %q", info)
	}
	if !strings.Contains(info, runtime.Version()) {
		t.Errorf("BuildInfo() should contain Go version, got %q", info)
	}
	if !strings.Contains(info, runtime.GOOS) {
		t.Errorf("BuildInfo() should contain OS, got %q", info)
	}
}

func TestUserAgent(t *testing.T) {
	ua := UserAgent()
	if !strings.HasPrefix(ua, "dfir-cli/") {
		t.Errorf("UserAgent() should start with 'dfir-cli/', got %q", ua)
	}
	if !strings.Contains(ua, runtime.GOOS) {
		t.Errorf("UserAgent() should contain OS, got %q", ua)
	}
}
