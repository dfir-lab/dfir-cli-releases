package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestVerifyChecksum_Valid(t *testing.T) {
	data := []byte("hello world")
	hash := sha256.Sum256(data)
	expected := hex.EncodeToString(hash[:])

	checksums := fmt.Sprintf("%s  test.tar.gz\n%s  other.tar.gz\n", expected, "aabbcc")

	if err := verifyChecksum(data, []byte(checksums), "test.tar.gz"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	data := []byte("hello world")
	checksums := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"

	err := verifyChecksum(data, []byte(checksums), "test.tar.gz")
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestVerifyChecksum_MissingFile(t *testing.T) {
	data := []byte("hello world")
	checksums := "aabbcc  other.tar.gz\n"

	err := verifyChecksum(data, []byte(checksums), "test.tar.gz")
	if err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestExtractFromTarGz(t *testing.T) {
	// Create a tar.gz with a binary inside a directory.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("fake-binary-content")

	hdr := &tar.Header{
		Name: "dfir-cli_0.1.1_darwin_arm64/dfir-cli",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()

	result, err := extractFromTarGz(buf.Bytes(), "dfir-cli")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if !bytes.Equal(result, content) {
		t.Errorf("extracted content mismatch: got %q, want %q", result, content)
	}
}

func TestExtractFromTarGz_NotFound(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tw.Close()
	gw.Close()

	_, err := extractFromTarGz(buf.Bytes(), "dfir-cli")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestDetectInstallMethod_Default(t *testing.T) {
	// When running tests, the binary is a test binary — should default to InstallBinary.
	method := DetectInstallMethod()
	if method == InstallHomebrew {
		t.Skip("running under Homebrew Go install — skip")
	}
	if method != InstallBinary {
		t.Errorf("expected InstallBinary, got %d", method)
	}
}
