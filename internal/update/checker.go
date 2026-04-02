package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/config"
)

const (
	stateFileName = "update-state.json"
	checkCooldown = 24 * time.Hour
	stateFilePerm = os.FileMode(0600)
)

var isUpdateNoticeTTY = func() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// CheckState persists the last update check result.
type CheckState struct {
	LastCheckedAt  time.Time `json:"last_checked_at"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	CurrentVersion string    `json:"current_version,omitempty"`
}

// statePath returns the path to the update state file.
func statePath() string {
	return filepath.Join(config.Dir(), stateFileName)
}

// ShouldCheck returns true if enough time has elapsed since the last check.
func ShouldCheck() bool {
	state, err := loadState()
	if err != nil {
		return true // file missing or corrupt → check
	}
	return time.Since(state.LastCheckedAt) >= checkCooldown
}

// RunBackgroundCheck performs an update check in a goroutine and sends the
// result (if a newer version is available) on the returned channel.
// The caller should read from the channel after the main command completes
// and print the notice.
//
// The check is skipped if:
//   - The cooldown has not elapsed
//   - Running in a CI environment (CI env var is set)
//   - DFIR_LAB_NO_UPDATE_NOTIFIER is set
//   - stderr is not a TTY
func RunBackgroundCheck(currentVersion string) <-chan *ReleaseInfo {
	ch := make(chan *ReleaseInfo, 1)

	// Check suppression conditions
	if !ShouldCheck() {
		close(ch)
		return ch
	}
	if ci := os.Getenv("CI"); ci != "" {
		close(ch)
		return ch
	}
	if os.Getenv("DFIR_LAB_NO_UPDATE_NOTIFIER") != "" {
		close(ch)
		return ch
	}
	if !isUpdateNoticeTTY() {
		close(ch)
		return ch
	}

	go func() {
		defer close(ch)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		release, err := CheckForUpdate(ctx, currentVersion)

		// Always update the state, even on error (to avoid hammering the API)
		_ = saveState(&CheckState{
			LastCheckedAt:  time.Now().UTC(),
			LatestVersion:  releaseVersion(release),
			CurrentVersion: currentVersion,
		})

		if err != nil || release == nil {
			return
		}

		ch <- release
	}()

	return ch
}

// PrintUpdateNotice prints the update notice to stderr if a release is available.
func PrintUpdateNotice(release *ReleaseInfo) {
	if release == nil {
		return
	}
	version := strings.TrimPrefix(release.TagName, "v")
	fmt.Fprintf(os.Stderr, "\nA new version of dfir-cli is available: %s\n", version)
	fmt.Fprintf(os.Stderr, "Update with: dfir-cli update\n")
	fmt.Fprintf(os.Stderr, "Release notes: %s\n\n", release.HTMLURL)
}

// releaseVersion extracts the version string from a release, or returns empty.
func releaseVersion(r *ReleaseInfo) string {
	if r == nil {
		return ""
	}
	return strings.TrimPrefix(r.TagName, "v")
}

// loadState reads the update check state from disk.
func loadState() (*CheckState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var state CheckState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveState writes the update check state to disk atomically.
func saveState(state *CheckState) error {
	_ = config.EnsureDir()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := statePath()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, stateFilePerm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
