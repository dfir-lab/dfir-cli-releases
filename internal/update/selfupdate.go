package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// InstallMethod describes how the CLI was installed.
type InstallMethod int

const (
	// InstallBinary means a standalone binary (curl | sh, manual download, etc.)
	InstallBinary InstallMethod = iota
	// InstallHomebrew means the binary was installed via Homebrew.
	InstallHomebrew
	// InstallScoop means the binary was installed via Scoop (Windows).
	InstallScoop
)

// DetectInstallMethod determines how the running binary was installed by
// inspecting its path and known package manager locations.
func DetectInstallMethod() InstallMethod {
	execPath, err := os.Executable()
	if err != nil {
		return InstallBinary
	}

	// Resolve symlinks to get the real path.
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	switch runtime.GOOS {
	case "darwin", "linux":
		// Check for Homebrew: binary lives under the Homebrew Cellar.
		if strings.Contains(realPath, "/Cellar/") || strings.Contains(realPath, "/homebrew/") {
			return InstallHomebrew
		}

		// Also check if `brew --prefix` output is a parent of the binary.
		if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
			prefix := strings.TrimSpace(string(out))
			if prefix != "" && strings.HasPrefix(realPath, prefix) {
				return InstallHomebrew
			}
		}

	case "windows":
		// Check for Scoop: binary lives under ~/scoop/
		home, _ := os.UserHomeDir()
		if home != "" {
			scoopDir := filepath.Join(home, "scoop")
			if strings.HasPrefix(strings.ToLower(realPath), strings.ToLower(scoopDir)) {
				return InstallScoop
			}
		}
	}

	return InstallBinary
}

// SelfUpdate downloads and installs the specified release version. It returns
// a human-readable message describing what happened.
func SelfUpdate(ctx context.Context, release *ReleaseInfo, verbose bool) error {
	version := release.TagName

	method := DetectInstallMethod()

	switch method {
	case InstallHomebrew:
		return updateViaHomebrew(verbose)
	case InstallScoop:
		return updateViaScoop(verbose)
	default:
		return updateViaBinaryDownload(ctx, version, verbose)
	}
}

// updateViaHomebrew runs brew upgrade to update the CLI.
func updateViaHomebrew(verbose bool) error {
	fmt.Println("  Updating via Homebrew...")
	fmt.Println()

	args := []string{"upgrade", "dfir-lab/tap/dfir-cli"}
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// brew upgrade returns non-zero if already up to date — check for that.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			fmt.Println("  Already up to date.")
			return nil
		}
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	return nil
}

// updateViaScoop runs scoop update to update the CLI.
func updateViaScoop(verbose bool) error {
	fmt.Println("  Updating via Scoop...")
	fmt.Println()

	cmd := exec.Command("scoop", "update", "dfir-cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scoop update failed: %w", err)
	}

	return nil
}

// updateViaBinaryDownload downloads the release archive, verifies the checksum,
// extracts the binary, and replaces the running executable.
func updateViaBinaryDownload(ctx context.Context, version string, verbose bool) error {
	dlURL := DownloadURL(version)
	csURL := ChecksumURL(version)

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] download URL: %s\n", dlURL)
		fmt.Fprintf(os.Stderr, "[verbose] checksum URL: %s\n", csURL)
	}

	// 1. Download checksums file.
	fmt.Println("  Downloading checksums...")
	checksums, err := httpGet(ctx, csURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	// 2. Download the archive.
	fmt.Println("  Downloading archive...")
	archive, err := httpGet(ctx, dlURL)
	if err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	// 3. Verify checksum.
	fmt.Println("  Verifying checksum...")
	archiveFilename := filepath.Base(dlURL)
	if err := verifyChecksum(archive, checksums, archiveFilename); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// 4. Extract the binary from the archive.
	fmt.Println("  Extracting binary...")
	binaryName := "dfir-cli"
	if runtime.GOOS == "windows" {
		binaryName = "dfir-cli.exe"
	}

	var binaryData []byte
	if strings.HasSuffix(dlURL, ".zip") {
		binaryData, err = extractFromZip(archive, binaryName)
	} else {
		binaryData, err = extractFromTarGz(archive, binaryName)
	}
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// 5. Replace the running binary.
	fmt.Println("  Installing...")
	if err := replaceBinary(binaryData); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// httpGet downloads a URL and returns the body as bytes.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "dfir-cli-updater")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

// verifyChecksum checks the SHA256 of data against the checksums file.
func verifyChecksum(data, checksums []byte, filename string) error {
	// Compute SHA256 of the archive.
	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])

	// Parse checksums file (format: "hash  filename" per line).
	lines := strings.Split(string(checksums), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			expected := strings.ToLower(parts[0])
			if actual != expected {
				return fmt.Errorf("expected %s, got %s", expected, actual)
			}
			return nil
		}
	}

	return fmt.Errorf("checksum for %q not found in checksums file", filename)
}

// extractFromTarGz extracts a named file from a .tar.gz archive.
func extractFromTarGz(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}

		// Match by base name (archives often have a directory prefix).
		if filepath.Base(hdr.Name) == name && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("binary %q not found in archive", name)
}

// extractFromZip extracts a named file from a .zip archive.
func extractFromZip(data []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}

	for _, f := range zr.File {
		if filepath.Base(f.Name) == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("binary %q not found in archive", name)
}

// replaceBinary atomically replaces the running executable with new binary data.
func replaceBinary(newBinary []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current executable: %w", err)
	}

	// Resolve symlinks so we replace the actual file, not the symlink.
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	// Get the permissions of the current binary to preserve them.
	info, err := os.Stat(realPath)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}
	perm := info.Mode().Perm()

	// Write new binary to a temp file in the same directory (for atomic rename).
	dir := filepath.Dir(realPath)
	tmp, err := os.CreateTemp(dir, "dfir-cli-update-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmp.Close()

	// Set executable permissions.
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	// On Windows, we can't replace a running binary directly.
	// Rename the current binary to .old first.
	if runtime.GOOS == "windows" {
		oldPath := realPath + ".old"
		_ = os.Remove(oldPath) // clean up any previous .old file
		if err := os.Rename(realPath, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("renaming current binary: %w", err)
		}
	}

	// Atomic rename: replace the current binary with the new one.
	if err := os.Rename(tmpPath, realPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}
