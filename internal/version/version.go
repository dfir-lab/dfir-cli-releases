package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// BuildInfo returns a full version string including build metadata.
func BuildInfo() string {
	return fmt.Sprintf("dfir-cli %s (build %s, commit %s, %s, %s/%s)",
		Version, Date, Commit, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// Short returns just the version string.
func Short() string {
	return Version
}

// UserAgent returns a string suitable for use as an HTTP User-Agent header.
func UserAgent() string {
	return fmt.Sprintf("dfir-cli/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
}
