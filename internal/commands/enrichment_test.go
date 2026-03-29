package commands

import (
	"strings"
	"testing"

	"github.com/dfir-lab/dfir-cli/internal/client"
)

// ---------------------------------------------------------------------------
// TestDetectIOCType
// ---------------------------------------------------------------------------

func TestDetectIOCType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "IPv4", input: "1.2.3.4", want: "ip"},
		{name: "IPv6", input: "2001:db8::1", want: "ip"},
		{name: "domain", input: "evil.com", want: "domain"},
		{name: "URL_https", input: "https://evil.com/path", want: "url"},
		{name: "URL_http", input: "http://evil.com", want: "url"},
		{name: "email", input: "user@evil.com", want: "email"},
		{name: "hash_MD5_32", input: "44d88612fea8a8f36de82e1278abb02f", want: "hash"},
		{name: "hash_SHA1_40", input: "da39a3ee5e6b4b0d3255bfef95601890afd80709", want: "hash"},
		{name: "hash_SHA256_64", input: strings.Repeat("ab", 32), want: "hash"},
		{name: "not_hex_falls_to_domain", input: "notahex123xyz", want: "domain"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectIOCType(tc.input)
			if got != tc.want {
				t.Errorf("detectIOCType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsValidIOCType
// ---------------------------------------------------------------------------

func TestIsValidIOCType(t *testing.T) {
	valid := []string{"ip", "domain", "url", "hash", "email"}
	for _, v := range valid {
		t.Run("valid_"+v, func(t *testing.T) {
			if !isValidIOCType(v) {
				t.Errorf("isValidIOCType(%q) = false, want true", v)
			}
		})
	}

	invalid := []string{"", "IP", "Domain", "ftp", "unknown", "md5"}
	for _, v := range invalid {
		name := v
		if name == "" {
			name = "empty"
		}
		t.Run("invalid_"+name, func(t *testing.T) {
			if isValidIOCType(v) {
				t.Errorf("isValidIOCType(%q) = true, want false", v)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestChunkIndicators
// ---------------------------------------------------------------------------

func TestChunkIndicators(t *testing.T) {
	makeIndicators := func(n int) []client.Indicator {
		out := make([]client.Indicator, n)
		for i := range out {
			out[i] = client.Indicator{Type: "ip", Value: "1.2.3.4"}
		}
		return out
	}

	tests := []struct {
		name       string
		count      int
		chunkSize  int
		wantChunks int
		wantLast   int // expected length of the last chunk
	}{
		{name: "3_items_chunk10", count: 3, chunkSize: 10, wantChunks: 1, wantLast: 3},
		{name: "15_items_chunk10", count: 15, chunkSize: 10, wantChunks: 2, wantLast: 5},
		{name: "10_items_chunk10", count: 10, chunkSize: 10, wantChunks: 1, wantLast: 10},
		{name: "0_items", count: 0, chunkSize: 10, wantChunks: 0, wantLast: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			indicators := makeIndicators(tc.count)
			chunks := chunkIndicators(indicators, tc.chunkSize)

			if len(chunks) != tc.wantChunks {
				t.Fatalf("chunkIndicators(%d, %d): got %d chunks, want %d",
					tc.count, tc.chunkSize, len(chunks), tc.wantChunks)
			}

			if tc.wantChunks > 0 {
				lastLen := len(chunks[len(chunks)-1])
				if lastLen != tc.wantLast {
					t.Errorf("last chunk length = %d, want %d", lastLen, tc.wantLast)
				}
			}

			// Verify total item count across all chunks.
			total := 0
			for _, c := range chunks {
				total += len(c)
			}
			if total != tc.count {
				t.Errorf("total items across chunks = %d, want %d", total, tc.count)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestParseProviderFilter
// ---------------------------------------------------------------------------

func TestParseProviderFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantKeys []string
	}{
		{name: "empty", input: "", wantNil: true},
		{name: "single", input: "VirusTotal", wantKeys: []string{"virustotal"}},
		{name: "two_providers", input: "VirusTotal,AbuseIPDB", wantKeys: []string{"virustotal", "abuseipdb"}},
		{name: "whitespace", input: " VirusTotal , AbuseIPDB ", wantKeys: []string{"virustotal", "abuseipdb"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseProviderFilter(tc.input)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("parseProviderFilter(%q) = %v, want nil", tc.input, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("parseProviderFilter(%q) = nil, want map with keys %v", tc.input, tc.wantKeys)
			}

			if len(got) != len(tc.wantKeys) {
				t.Fatalf("parseProviderFilter(%q): got %d keys, want %d", tc.input, len(got), len(tc.wantKeys))
			}

			for _, k := range tc.wantKeys {
				if _, ok := got[k]; !ok {
					t.Errorf("parseProviderFilter(%q): missing key %q", tc.input, k)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFilterResults
// ---------------------------------------------------------------------------

func TestFilterResults(t *testing.T) {
	// Helper to build a result with the given providers.
	makeResult := func(providers map[string]client.ProviderResult) client.EnrichmentResult {
		return client.EnrichmentResult{
			Indicator: client.Indicator{Type: "ip", Value: "1.2.3.4"},
			Verdict:   "malicious",
			Score:     80,
			Providers: providers,
		}
	}

	baseProviders := map[string]client.ProviderResult{
		"VirusTotal": {Verdict: "malicious", Score: 90},
		"AbuseIPDB":  {Verdict: "suspicious", Score: 60},
		"Shodan":     {Verdict: "clean", Score: 10},
	}

	t.Run("no_filters", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		filtered := filterResults(results, "", 0)
		if len(filtered[0].Providers) != 3 {
			t.Errorf("expected 3 providers, got %d", len(filtered[0].Providers))
		}
	})

	t.Run("provider_filter", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		filtered := filterResults(results, "VirusTotal", 0)
		if len(filtered[0].Providers) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(filtered[0].Providers))
		}
		if _, ok := filtered[0].Providers["VirusTotal"]; !ok {
			t.Error("expected VirusTotal to remain")
		}
	})

	t.Run("min_score_filter", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		filtered := filterResults(results, "", 50)
		// Only VirusTotal (90) and AbuseIPDB (60) should pass.
		if len(filtered[0].Providers) != 2 {
			t.Fatalf("expected 2 providers, got %d", len(filtered[0].Providers))
		}
		if _, ok := filtered[0].Providers["Shodan"]; ok {
			t.Error("Shodan (score 10) should have been filtered out")
		}
	})

	t.Run("combined_filters", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		// Filter to AbuseIPDB only, with min score 70 -- AbuseIPDB has 60, so nothing remains.
		filtered := filterResults(results, "AbuseIPDB", 70)
		if len(filtered[0].Providers) != 0 {
			t.Errorf("expected 0 providers, got %d", len(filtered[0].Providers))
		}
	})
}

// copyProviders returns a shallow copy of a provider map so each sub-test
// starts from the same baseline.
func copyProviders(src map[string]client.ProviderResult) map[string]client.ProviderResult {
	dst := make(map[string]client.ProviderResult, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
