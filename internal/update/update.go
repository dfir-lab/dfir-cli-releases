package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const githubRepo = "ForeGuards/dfir-cli-releases"

// githubAPIURL is a variable (not const) so tests can override it with httptest servers.
var githubAPIURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"

// ReleaseInfo contains information about a GitHub release.
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
}

// CheckForUpdate checks GitHub Releases for a newer version.
// Returns the release info if a newer version is available, nil if current is latest.
// currentVersion should NOT have a "v" prefix.
func CheckForUpdate(ctx context.Context, currentVersion string) (*ReleaseInfo, error) {
	// Skip if current version is "dev" (development build)
	if currentVersion == "dev" || currentVersion == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "dfir-cli/"+currentVersion)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check for update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if !isNewer(latestVersion, currentVersion) {
		return nil, nil // already up to date
	}

	return &release, nil
}

// isNewer returns true if latest is a higher semantic version than current.
// Compares major, minor, and patch as integers. Pre-release suffixes
// (e.g., "1.0.0-rc1") are treated as older than the corresponding release
// version (e.g., "1.0.0").
func isNewer(latest, current string) bool {
	latestParts, latestPre := parseSemver(latest)
	currentParts, currentPre := parseSemver(current)

	// Compare major, minor, patch numerically.
	for i := 0; i < 3; i++ {
		l, c := 0, 0
		if i < len(latestParts) {
			l = latestParts[i]
		}
		if i < len(currentParts) {
			c = currentParts[i]
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	// Numeric parts are equal. A version without a pre-release suffix is
	// considered newer than one with a pre-release suffix.
	// e.g. "1.0.0" is newer than "1.0.0-rc1".
	if currentPre != "" && latestPre == "" {
		return true
	}
	if currentPre == "" && latestPre != "" {
		return false
	}

	// Both have pre-release suffixes (or neither does). Compare lexically.
	return latestPre > currentPre
}

// parseSemver splits a version string like "1.2.3-rc1" into its numeric
// parts ([1, 2, 3]) and the pre-release suffix ("rc1"). If any numeric
// part cannot be parsed it is treated as 0.
func parseSemver(version string) (parts []int, preRelease string) {
	// Separate pre-release suffix from the numeric portion.
	numericStr := version
	if idx := strings.IndexByte(version, '-'); idx != -1 {
		numericStr = version[:idx]
		preRelease = version[idx+1:]
	}

	for _, seg := range strings.Split(numericStr, ".") {
		n, err := strconv.Atoi(seg)
		if err != nil {
			n = 0
		}
		parts = append(parts, n)
	}
	return parts, preRelease
}

// DownloadURL returns the expected download URL for the current platform.
// TODO: Used by future auto-update feature (currently the update command shows manual instructions).
func DownloadURL(version string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	ext := "tar.gz"
	if os == "windows" {
		ext = "zip"
	}
	ver := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("https://github.com/%s/releases/download/v%s/dfir-cli_%s_%s_%s.%s",
		githubRepo, ver, ver, os, arch, ext)
}

// ChecksumURL returns the URL for the checksums file.
// TODO: Used by future auto-update feature.
func ChecksumURL(version string) string {
	ver := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("https://github.com/%s/releases/download/v%s/dfir-cli_%s_checksums.txt",
		githubRepo, ver, ver)
}
