package commands

import (
	"errors"
	"os"
	"path/filepath"
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
		{name: "IPv6_full", input: "2001:0db8:85a3:0000:0000:8a2e:0370:7334", want: "ip"},
		{name: "IPv4_loopback", input: "127.0.0.1", want: "ip"},
		{name: "IPv6_loopback", input: "::1", want: "ip"},
		{name: "domain", input: "evil.com", want: "domain"},
		{name: "domain_subdomain", input: "sub.evil.com", want: "domain"},
		{name: "URL_https", input: "https://evil.com/path", want: "url"},
		{name: "URL_http", input: "http://evil.com", want: "url"},
		{name: "URL_with_query", input: "https://evil.com/path?q=1&r=2", want: "url"},
		{name: "URL_http_upper", input: "HTTP://EVIL.COM", want: "url"},
		{name: "URL_https_upper", input: "HTTPS://EVIL.COM", want: "url"},
		{name: "email", input: "user@evil.com", want: "email"},
		{name: "email_plus", input: "user+tag@evil.com", want: "email"},
		{name: "hash_MD5_32", input: "44d88612fea8a8f36de82e1278abb02f", want: "hash"},
		{name: "hash_SHA1_40", input: "da39a3ee5e6b4b0d3255bfef95601890afd80709", want: "hash"},
		{name: "hash_SHA256_64", input: strings.Repeat("ab", 32), want: "hash"},
		{name: "hash_MD5_upper", input: "44D88612FEA8A8F36DE82E1278ABB02F", want: "hash"},
		{name: "not_hex_falls_to_domain", input: "notahex123xyz", want: "domain"},
		{name: "CIDR_notation_not_IP", input: "192.168.1.0/24", want: "domain"},
		{name: "defanged_IP_not_detected", input: "1[.]2[.]3[.]4", want: "domain"},
		{name: "URL_without_scheme_is_domain", input: "evil.com/path", want: "domain"},
		{name: "hex_wrong_length_31", input: strings.Repeat("a", 31), want: "domain"},
		{name: "hex_wrong_length_33", input: strings.Repeat("a", 33), want: "domain"},
		{name: "whitespace_trimmed", input: "  1.2.3.4  ", want: "ip"},
		{name: "empty_string", input: "", want: "domain"},
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
		{name: "1_item_chunk10", count: 1, chunkSize: 10, wantChunks: 1, wantLast: 1},
		{name: "1_item_chunk1", count: 1, chunkSize: 1, wantChunks: 1, wantLast: 1},
		{name: "5_items_chunk1", count: 5, chunkSize: 1, wantChunks: 5, wantLast: 1},
		{name: "negative_chunk_uses_default", count: 3, chunkSize: -1, wantChunks: 1, wantLast: 3},
		{name: "zero_chunk_uses_default", count: 3, chunkSize: 0, wantChunks: 1, wantLast: 3},
		{name: "exact_multiple", count: 20, chunkSize: 10, wantChunks: 2, wantLast: 10},
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
		{name: "whitespace_only", input: "   ", wantNil: true},
		{name: "single", input: "VirusTotal", wantKeys: []string{"virustotal"}},
		{name: "two_providers", input: "VirusTotal,AbuseIPDB", wantKeys: []string{"virustotal", "abuseipdb"}},
		{name: "whitespace", input: " VirusTotal , AbuseIPDB ", wantKeys: []string{"virustotal", "abuseipdb"}},
		{name: "trailing_comma", input: "VirusTotal,", wantKeys: []string{"virustotal"}},
		{name: "leading_comma", input: ",VirusTotal", wantKeys: []string{"virustotal"}},
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

	t.Run("provider_filter_comma_separated", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		filtered := filterResults(results, "VirusTotal,AbuseIPDB", 0)
		if len(filtered[0].Providers) != 2 {
			t.Fatalf("expected 2 providers, got %d", len(filtered[0].Providers))
		}
		if _, ok := filtered[0].Providers["VirusTotal"]; !ok {
			t.Error("expected VirusTotal to remain")
		}
		if _, ok := filtered[0].Providers["AbuseIPDB"]; !ok {
			t.Error("expected AbuseIPDB to remain")
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

	t.Run("min_score_filter_high", func(t *testing.T) {
		results := []client.EnrichmentResult{makeResult(copyProviders(baseProviders))}
		filtered := filterResults(results, "", 100)
		if len(filtered[0].Providers) != 0 {
			t.Errorf("expected 0 providers with min-score 100, got %d", len(filtered[0].Providers))
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

	t.Run("multiple_results", func(t *testing.T) {
		r1 := makeResult(copyProviders(baseProviders))
		r2 := makeResult(map[string]client.ProviderResult{
			"VirusTotal": {Verdict: "clean", Score: 5},
		})
		results := filterResults([]client.EnrichmentResult{r1, r2}, "VirusTotal", 0)
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if len(results[0].Providers) != 1 {
			t.Errorf("result[0] expected 1 provider, got %d", len(results[0].Providers))
		}
		if len(results[1].Providers) != 1 {
			t.Errorf("result[1] expected 1 provider, got %d", len(results[1].Providers))
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

// ---------------------------------------------------------------------------
// TestParseIOCLines
// ---------------------------------------------------------------------------

func TestParseIOCLines(t *testing.T) {
	t.Run("with_comments_and_empty_lines", func(t *testing.T) {
		data := "# comment\n1.2.3.4\n\n# another comment\n5.6.7.8\n"
		indicators := parseIOCLines(data, "ip")
		if len(indicators) != 2 {
			t.Fatalf("expected 2 indicators, got %d", len(indicators))
		}
		if indicators[0].Value != "1.2.3.4" {
			t.Errorf("indicators[0].Value = %q, want %q", indicators[0].Value, "1.2.3.4")
		}
		if indicators[1].Value != "5.6.7.8" {
			t.Errorf("indicators[1].Value = %q, want %q", indicators[1].Value, "5.6.7.8")
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		indicators := parseIOCLines("", "ip")
		if len(indicators) != 0 {
			t.Errorf("expected 0 indicators for empty input, got %d", len(indicators))
		}
	})

	t.Run("only_comments", func(t *testing.T) {
		data := "# comment 1\n# comment 2\n"
		indicators := parseIOCLines(data, "ip")
		if len(indicators) != 0 {
			t.Errorf("expected 0 indicators for comments-only input, got %d", len(indicators))
		}
	})

	t.Run("auto_detect_type_when_empty", func(t *testing.T) {
		data := "1.2.3.4\nevil.com\n"
		indicators := parseIOCLines(data, "")
		if len(indicators) != 2 {
			t.Fatalf("expected 2 indicators, got %d", len(indicators))
		}
		if indicators[0].Type != "ip" {
			t.Errorf("indicators[0].Type = %q, want %q", indicators[0].Type, "ip")
		}
		if indicators[1].Type != "domain" {
			t.Errorf("indicators[1].Type = %q, want %q", indicators[1].Type, "domain")
		}
	})

	t.Run("whitespace_trimmed", func(t *testing.T) {
		data := "  1.2.3.4  \n  evil.com  \n"
		indicators := parseIOCLines(data, "ip")
		if len(indicators) != 2 {
			t.Fatalf("expected 2 indicators, got %d", len(indicators))
		}
		if indicators[0].Value != "1.2.3.4" {
			t.Errorf("indicators[0].Value = %q, want %q", indicators[0].Value, "1.2.3.4")
		}
	})
}

// ---------------------------------------------------------------------------
// TestReadBatchIndicators
// ---------------------------------------------------------------------------

func TestReadBatchIndicators(t *testing.T) {
	t.Run("from_file_with_auto_detect", func(t *testing.T) {
		content := "# IOC list\n1.2.3.4\nevil.com\n\nuser@bad.com\n"
		dir := t.TempDir()
		path := filepath.Join(dir, "iocs.txt")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		indicators, err := readBatchIndicators(path, "")
		if err != nil {
			t.Fatalf("readBatchIndicators returned error: %v", err)
		}
		if len(indicators) != 3 {
			t.Fatalf("expected 3 indicators, got %d", len(indicators))
		}
		if indicators[0].Type != "ip" {
			t.Errorf("indicators[0].Type = %q, want %q", indicators[0].Type, "ip")
		}
		if indicators[1].Type != "domain" {
			t.Errorf("indicators[1].Type = %q, want %q", indicators[1].Type, "domain")
		}
		if indicators[2].Type != "email" {
			t.Errorf("indicators[2].Type = %q, want %q", indicators[2].Type, "email")
		}
	})

	t.Run("from_file_with_forced_type", func(t *testing.T) {
		content := "1.2.3.4\n5.6.7.8\n"
		dir := t.TempDir()
		path := filepath.Join(dir, "ips.txt")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		indicators, err := readBatchIndicators(path, "ip")
		if err != nil {
			t.Fatalf("readBatchIndicators returned error: %v", err)
		}
		if len(indicators) != 2 {
			t.Fatalf("expected 2 indicators, got %d", len(indicators))
		}
		for _, ind := range indicators {
			if ind.Type != "ip" {
				t.Errorf("expected type %q, got %q", "ip", ind.Type)
			}
		}
	})

	t.Run("stdin_without_type_returns_error", func(t *testing.T) {
		_, err := readBatchIndicators("-", "")
		if err == nil {
			t.Fatal("expected error when reading from stdin without --type")
		}
		if !strings.Contains(err.Error(), "--type is required") {
			t.Errorf("error should mention --type, got: %v", err)
		}
	})

	t.Run("file_not_found", func(t *testing.T) {
		_, err := readBatchIndicators("/tmp/nonexistent_dfir_test.txt", "ip")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// TestEnrichmentExitCode
// ---------------------------------------------------------------------------

func TestEnrichmentExitCode(t *testing.T) {
	t.Run("all_clean", func(t *testing.T) {
		results := []client.EnrichmentResult{
			{Verdict: "clean"},
			{Verdict: "unknown"},
		}
		err := enrichmentExitCode(results)
		if err != nil {
			t.Errorf("expected nil for clean results, got %v", err)
		}
	})

	t.Run("malicious", func(t *testing.T) {
		results := []client.EnrichmentResult{
			{Verdict: "clean"},
			{Verdict: "malicious"},
		}
		err := enrichmentExitCode(results)
		if err == nil {
			t.Fatal("expected error for malicious result")
		}
		var exitErr *SilentExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *SilentExitError, got %T", err)
		}
		if exitErr.Code != 2 {
			t.Errorf("exit code = %d, want 2", exitErr.Code)
		}
	})

	t.Run("suspicious_no_malicious", func(t *testing.T) {
		results := []client.EnrichmentResult{
			{Verdict: "clean"},
			{Verdict: "suspicious"},
		}
		err := enrichmentExitCode(results)
		if err == nil {
			t.Fatal("expected error for suspicious result")
		}
		var exitErr *SilentExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *SilentExitError, got %T", err)
		}
		if exitErr.Code != 3 {
			t.Errorf("exit code = %d, want 3", exitErr.Code)
		}
	})

	t.Run("malicious_overrides_suspicious", func(t *testing.T) {
		results := []client.EnrichmentResult{
			{Verdict: "suspicious"},
			{Verdict: "malicious"},
		}
		err := enrichmentExitCode(results)
		var exitErr *SilentExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *SilentExitError, got %T", err)
		}
		if exitErr.Code != 2 {
			t.Errorf("exit code = %d, want 2 (malicious overrides suspicious)", exitErr.Code)
		}
	})

	t.Run("empty_results", func(t *testing.T) {
		err := enrichmentExitCode(nil)
		if err != nil {
			t.Errorf("expected nil for empty results, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestClassifyEnrichmentError
// ---------------------------------------------------------------------------

func TestClassifyEnrichmentError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		err := classifyEnrichmentError(nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("insufficient_credits", func(t *testing.T) {
		creditsErr := &client.InsufficientCreditsError{Message: "no credits"}
		err := classifyEnrichmentError(creditsErr)
		var exitErr *SilentExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *SilentExitError, got %T", err)
		}
		if exitErr.Code != 4 {
			t.Errorf("exit code = %d, want 4", exitErr.Code)
		}
	})

	t.Run("generic_error_passed_through", func(t *testing.T) {
		genericErr := errors.New("some network error")
		err := classifyEnrichmentError(genericErr)
		if err != genericErr {
			t.Errorf("expected error to be passed through, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestCountFlaggedProviders
// ---------------------------------------------------------------------------

func TestCountFlaggedProviders(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]client.ProviderResult
		want      int
	}{
		{
			name:      "empty",
			providers: map[string]client.ProviderResult{},
			want:      0,
		},
		{
			name: "all_clean",
			providers: map[string]client.ProviderResult{
				"A": {Verdict: "clean"},
				"B": {Verdict: "unknown"},
			},
			want: 0,
		},
		{
			name: "one_malicious",
			providers: map[string]client.ProviderResult{
				"A": {Verdict: "malicious"},
				"B": {Verdict: "clean"},
			},
			want: 1,
		},
		{
			name: "one_suspicious",
			providers: map[string]client.ProviderResult{
				"A": {Verdict: "suspicious"},
				"B": {Verdict: "clean"},
			},
			want: 1,
		},
		{
			name: "mixed",
			providers: map[string]client.ProviderResult{
				"A": {Verdict: "malicious"},
				"B": {Verdict: "suspicious"},
				"C": {Verdict: "clean"},
			},
			want: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countFlaggedProviders(tc.providers)
			if got != tc.want {
				t.Errorf("countFlaggedProviders() = %d, want %d", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFormatProviderDetails
// ---------------------------------------------------------------------------

func TestFormatProviderDetails(t *testing.T) {
	t.Run("nil_details", func(t *testing.T) {
		got := formatProviderDetails(nil)
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("empty_details", func(t *testing.T) {
		got := formatProviderDetails(map[string]interface{}{})
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("single_detail", func(t *testing.T) {
		got := formatProviderDetails(map[string]interface{}{"country": "US"})
		if got != "country: US" {
			t.Errorf("expected %q, got %q", "country: US", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestResolveIndicators
// ---------------------------------------------------------------------------

func TestResolveIndicators(t *testing.T) {
	t.Run("typed_flag_ip", func(t *testing.T) {
		f := enrichmentLookupFlags{ip: "1.2.3.4"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "ip" || indicators[0].Value != "1.2.3.4" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("typed_flag_domain", func(t *testing.T) {
		f := enrichmentLookupFlags{domain: "evil.com"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "domain" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("typed_flag_url", func(t *testing.T) {
		f := enrichmentLookupFlags{url: "https://evil.com"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "url" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("typed_flag_hash", func(t *testing.T) {
		f := enrichmentLookupFlags{hash: "abc123"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "hash" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("typed_flag_email", func(t *testing.T) {
		f := enrichmentLookupFlags{email: "user@evil.com"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "email" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("ioc_flag_auto_detect", func(t *testing.T) {
		f := enrichmentLookupFlags{ioc: "1.2.3.4"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "ip" {
			t.Errorf("unexpected indicators: %v", indicators)
		}
	})

	t.Run("ioc_flag_with_forced_type", func(t *testing.T) {
		f := enrichmentLookupFlags{ioc: "1.2.3.4", iocType: "domain"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Type != "domain" {
			t.Errorf("expected forced type 'domain', got: %v", indicators)
		}
	})

	t.Run("batch_from_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "batch.txt")
		if err := os.WriteFile(path, []byte("1.2.3.4\n5.6.7.8\n"), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		f := enrichmentLookupFlags{batch: path, iocType: "ip"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 2 {
			t.Fatalf("expected 2 indicators, got %d", len(indicators))
		}
	})

	t.Run("no_input_returns_nil", func(t *testing.T) {
		f := enrichmentLookupFlags{}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indicators != nil {
			t.Errorf("expected nil indicators, got %v", indicators)
		}
	})

	t.Run("typed_flag_precedence_over_ioc", func(t *testing.T) {
		// Typed flag (ip) takes precedence over --ioc.
		f := enrichmentLookupFlags{ip: "1.2.3.4", ioc: "evil.com"}
		indicators, err := resolveIndicators(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(indicators) != 1 || indicators[0].Value != "1.2.3.4" {
			t.Errorf("expected ip flag to take precedence, got %v", indicators)
		}
	})
}

// ---------------------------------------------------------------------------
// TestEnrichmentLookupConcurrencyFlag
// ---------------------------------------------------------------------------

func TestEnrichmentLookupConcurrencyFlag(t *testing.T) {
	cmd := newEnrichmentLookupCmd()

	t.Run("default_value", func(t *testing.T) {
		f := cmd.Flags().Lookup("concurrency")
		if f == nil {
			t.Fatal("--concurrency flag not found on enrichment lookup command")
		}
		if f.DefValue != "5" {
			t.Errorf("default concurrency = %q, want %q", f.DefValue, "5")
		}
	})

	t.Run("flag_is_parseable", func(t *testing.T) {
		testCmd := newEnrichmentLookupCmd()
		testCmd.SetArgs([]string{"--concurrency", "10", "--ip", "1.2.3.4"})
		f := testCmd.Flags().Lookup("concurrency")
		if f == nil {
			t.Fatal("--concurrency flag not found")
		}
	})
}

// ---------------------------------------------------------------------------
// TestEnrichmentConcurrencyValidation
// ---------------------------------------------------------------------------

func TestEnrichmentConcurrencyValidation(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
		errContains string
	}{
		{name: "valid_1", concurrency: 1, wantErr: false},
		{name: "valid_5", concurrency: 5, wantErr: false},
		{name: "valid_20", concurrency: 20, wantErr: false},
		{name: "too_low_0", concurrency: 0, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
		{name: "too_high_21", concurrency: 21, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
		{name: "negative", concurrency: -1, wantErr: true, errContains: "--concurrency must be between 1 and 20"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newEnrichmentLookupCmd()
			err := runEnrichmentLookup(cmd, enrichmentLookupFlags{
				ip:          "1.2.3.4",
				concurrency: tc.concurrency,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
			}
			if !tc.wantErr && err != nil {
				if strings.Contains(err.Error(), "--concurrency") {
					t.Errorf("unexpected concurrency error: %v", err)
				}
			}
		})
	}
}

func TestEnrichmentConcurrencyValidation_BeforeAPIKeyResolution(t *testing.T) {
	cmd := newEnrichmentLookupCmd()

	err := runEnrichmentLookup(cmd, enrichmentLookupFlags{
		ip:          "1.2.3.4",
		concurrency: 0,
	})
	if err == nil {
		t.Fatal("expected concurrency validation error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "--concurrency must be between 1 and 20") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(msg, "no API key configured") {
		t.Fatalf("concurrency should be validated before API key lookup, got: %v", err)
	}
}
