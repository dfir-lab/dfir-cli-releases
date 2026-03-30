package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// resolveSafeBrowsingInputs
// ---------------------------------------------------------------------------

func TestResolveSafeBrowsingInputs_SingleURL(t *testing.T) {
	urls, err := resolveSafeBrowsingInputs("https://example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://example.com" {
		t.Errorf("got %v, want [https://example.com]", urls)
	}
}

func TestResolveSafeBrowsingInputs_BatchFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")
	content := "https://example.com\nhttps://evil.com\n# comment\n\nhttps://safe.com\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	urls, err := resolveSafeBrowsingInputs("", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("got %d URLs, want 3: %v", len(urls), urls)
	}
	expected := []string{"https://example.com", "https://evil.com", "https://safe.com"}
	for i, want := range expected {
		if urls[i] != want {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], want)
		}
	}
}

func TestResolveSafeBrowsingInputs_NoInput(t *testing.T) {
	urls, err := resolveSafeBrowsingInputs("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urls != nil {
		t.Errorf("expected nil, got %v", urls)
	}
}

func TestResolveSafeBrowsingInputs_URLTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")
	if err := os.WriteFile(path, []byte("https://other.com\n"), 0644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	urls, err := resolveSafeBrowsingInputs("https://example.com", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://example.com" {
		t.Errorf("--url should take precedence, got %v", urls)
	}
}

// ---------------------------------------------------------------------------
// safeBrowsingStatus
// ---------------------------------------------------------------------------

func TestSafeBrowsingStatus(t *testing.T) {
	tests := []struct {
		name  string
		entry map[string]interface{}
		want  string
	}{
		{
			name:  "safe_bool_true",
			entry: map[string]interface{}{"safe": true},
			want:  "safe",
		},
		{
			name:  "safe_bool_false",
			entry: map[string]interface{}{"safe": false},
			want:  "unsafe",
		},
		{
			name:  "threat_type_present",
			entry: map[string]interface{}{"threat_type": "MALWARE"},
			want:  "unsafe",
		},
		{
			name:  "threat_type_none",
			entry: map[string]interface{}{"threat_type": "none"},
			want:  "safe",
		},
		{
			name:  "status_field",
			entry: map[string]interface{}{"status": "SAFE"},
			want:  "safe",
		},
		{
			name:  "empty_entry",
			entry: map[string]interface{}{},
			want:  "safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeBrowsingStatus(tt.entry)
			if got != tt.want {
				t.Errorf("safeBrowsingStatus(%v) = %q, want %q", tt.entry, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// newPhishingSafeBrowsingCmd structure
// ---------------------------------------------------------------------------

func TestPhishingSafeBrowsingCmd_Flags(t *testing.T) {
	cmd := newPhishingSafeBrowsingCmd()

	if cmd.Use != "safe-browsing" {
		t.Errorf("Use = %q, want %q", cmd.Use, "safe-browsing")
	}

	urlFlag := cmd.Flags().Lookup("url")
	if urlFlag == nil {
		t.Fatal("expected --url flag to be registered")
	}

	batchFlag := cmd.Flags().Lookup("batch")
	if batchFlag == nil {
		t.Fatal("expected --batch flag to be registered")
	}
}
