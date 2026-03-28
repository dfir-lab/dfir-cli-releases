package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "1.0.0", false},
		{"0.9.0", "1.0.0", false},
		{"1.0.0-rc1", "1.0.0", false},   // pre-release is older than stable
		{"1.0.0", "1.0.0-rc1", true},    // stable release is newer than pre-release
		{"1.0.0-rc2", "1.0.0-rc1", true}, // later pre-release is newer
	}

	for _, tt := range tests {
		t.Run(tt.latest+"_vs_"+tt.current, func(t *testing.T) {
			got := isNewer(tt.latest, tt.current)
			if got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestDownloadURL(t *testing.T) {
	url := DownloadURL("v1.2.3")

	// Should contain the version without the "v" prefix in the filename part.
	if !strings.Contains(url, "1.2.3") {
		t.Errorf("DownloadURL missing version: %s", url)
	}
	if !strings.Contains(url, runtime.GOOS) {
		t.Errorf("DownloadURL missing OS (%s): %s", runtime.GOOS, url)
	}
	if !strings.Contains(url, runtime.GOARCH) {
		t.Errorf("DownloadURL missing arch (%s): %s", runtime.GOARCH, url)
	}

	expected := "https://github.com/dfir-lab/dfir-cli/releases/download/v1.2.3/"
	if !strings.HasPrefix(url, expected) {
		t.Errorf("DownloadURL unexpected prefix:\n  got:  %s\n  want prefix: %s", url, expected)
	}
}

func TestChecksumURL(t *testing.T) {
	url := ChecksumURL("v1.2.3")

	expected := "https://github.com/dfir-lab/dfir-cli/releases/download/v1.2.3/dfir-cli_1.2.3_checksums.txt"
	if url != expected {
		t.Errorf("ChecksumURL:\n  got:  %s\n  want: %s", url, expected)
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	release, err := CheckForUpdate(context.Background(), "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release != nil {
		t.Errorf("expected nil for dev version, got %+v", release)
	}
}

// newMockGitHubServer creates an httptest server that returns the given tag as
// the latest release. It also overrides githubAPIURL and returns a cleanup
// function that restores the original value.
func newMockGitHubServer(t *testing.T, tagName string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name":     tagName,
			"name":         tagName,
			"html_url":     "https://github.com/dfir-lab/dfir-cli/releases/tag/" + tagName,
			"published_at": time.Now().Format(time.RFC3339),
		})
	}))

	original := githubAPIURL
	githubAPIURL = server.URL
	t.Cleanup(func() {
		githubAPIURL = original
		server.Close()
	})

	return server
}

func TestCheckForUpdate_UpToDate(t *testing.T) {
	newMockGitHubServer(t, "v1.0.0")

	release, err := CheckForUpdate(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release != nil {
		t.Errorf("expected nil when up to date, got %+v", release)
	}
}

func TestCheckForUpdate_NewVersion(t *testing.T) {
	newMockGitHubServer(t, "v1.1.0")

	release, err := CheckForUpdate(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release == nil {
		t.Fatal("expected release info for newer version, got nil")
	}
	if release.TagName != "v1.1.0" {
		t.Errorf("expected tag v1.1.0, got %s", release.TagName)
	}
	if release.HTMLURL == "" {
		t.Error("expected non-empty HTMLURL")
	}
}
