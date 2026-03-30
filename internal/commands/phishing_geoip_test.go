package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// resolveGeoIPInputs
// ---------------------------------------------------------------------------

func TestResolveGeoIPInputs_SingleIP(t *testing.T) {
	ips, err := resolveGeoIPInputs("8.8.8.8", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "8.8.8.8" {
		t.Errorf("got %v, want [8.8.8.8]", ips)
	}
}

func TestResolveGeoIPInputs_BatchFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ips.txt")
	content := "1.1.1.1\n8.8.8.8\n# comment\n\n9.9.9.9\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	ips, err := resolveGeoIPInputs("", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 3 {
		t.Fatalf("got %d IPs, want 3: %v", len(ips), ips)
	}
	expected := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}
	for i, want := range expected {
		if ips[i] != want {
			t.Errorf("ips[%d] = %q, want %q", i, ips[i], want)
		}
	}
}

func TestResolveGeoIPInputs_NoInput(t *testing.T) {
	ips, err := resolveGeoIPInputs("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ips != nil {
		t.Errorf("expected nil, got %v", ips)
	}
}

func TestResolveGeoIPInputs_IPTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ips.txt")
	if err := os.WriteFile(path, []byte("9.9.9.9\n"), 0644); err != nil {
		t.Fatalf("failed to write batch file: %v", err)
	}

	ips, err := resolveGeoIPInputs("8.8.8.8", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "8.8.8.8" {
		t.Errorf("--ip should take precedence, got %v", ips)
	}
}

// ---------------------------------------------------------------------------
// strOrDash
// ---------------------------------------------------------------------------

func TestStrOrDash(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{"present", map[string]interface{}{"country": "US"}, "country", "US"},
		{"missing", map[string]interface{}{}, "country", "-"},
		{"nil_value", map[string]interface{}{"country": nil}, "country", "-"},
		{"empty_string", map[string]interface{}{"country": ""}, "country", "-"},
		{"numeric", map[string]interface{}{"asn": float64(15169)}, "asn", "15169"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strOrDash(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("strOrDash(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// newPhishingGeoIPCmd structure
// ---------------------------------------------------------------------------

func TestPhishingGeoIPCmd_Flags(t *testing.T) {
	cmd := newPhishingGeoIPCmd()

	if cmd.Use != "geoip" {
		t.Errorf("Use = %q, want %q", cmd.Use, "geoip")
	}

	ipFlag := cmd.Flags().Lookup("ip")
	if ipFlag == nil {
		t.Fatal("expected --ip flag to be registered")
	}

	batchFlag := cmd.Flags().Lookup("batch")
	if batchFlag == nil {
		t.Fatal("expected --batch flag to be registered")
	}
}
