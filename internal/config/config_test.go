package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestDefaultProfile
// ---------------------------------------------------------------------------

func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()

	if p.APIURL != "https://dfir-lab.ch/api/v1" {
		t.Errorf("APIURL = %q, want %q", p.APIURL, "https://dfir-lab.ch/api/v1")
	}
	if p.OutputFormat != "table" {
		t.Errorf("OutputFormat = %q, want %q", p.OutputFormat, "table")
	}
	if p.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", p.Timeout, 60*time.Second)
	}
	if p.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want %d", p.Concurrency, 5)
	}
	if p.NoColor {
		t.Error("NoColor should be false by default")
	}
	if p.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", p.APIKey)
	}
}

// ---------------------------------------------------------------------------
// TestValidateAPIKeyFormat
// ---------------------------------------------------------------------------

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid key",
			key:     "sk-dfir-abcdefghijklmnop",
			wantErr: false,
		},
		{
			name:    "valid key at max boundary",
			key:     "sk-dfir-" + strings.Repeat("a", 120),
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
			errMsg:  "must start with",
		},
		{
			name:    "wrong prefix",
			key:     "pk-dfir-abcdefghijklmnop",
			wantErr: true,
			errMsg:  "must start with",
		},
		{
			name:    "too short",
			key:     "sk-dfir-abc",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "too long",
			key:     "sk-dfir-" + strings.Repeat("x", 125),
			wantErr: true,
			errMsg:  "too long",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMaskAPIKey
// ---------------------------------------------------------------------------

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "normal key",
			key:  "sk-dfir-abcdefghij1234",
			want: "sk-dfir-**********1234",
		},
		{
			name: "empty key",
			key:  "",
			want: "",
		},
		{
			name: "short key equal to prefix plus 4",
			key:  "sk-dfir-abcd",
			want: "************",
		},
		{
			name: "short key less than prefix plus 4",
			key:  "sk-dfir-ab",
			want: "**********",
		},
		{
			name: "very short key",
			key:  "abc",
			want: "***",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskAPIKey(tc.key)
			if got != tc.want {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateProfileName
// ---------------------------------------------------------------------------

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default",
			profile: "default",
			wantErr: false,
		},
		{
			name:    "valid staging",
			profile: "staging",
			wantErr: false,
		},
		{
			name:    "valid with hyphens",
			profile: "my-profile",
			wantErr: false,
		},
		{
			name:    "empty name",
			profile: "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "contains dots",
			profile: "my.profile",
			wantErr: true,
			errMsg:  "cannot contain dots",
		},
		{
			name:    "contains spaces",
			profile: "my profile",
			wantErr: true,
			errMsg:  "cannot contain whitespace",
		},
		{
			name:    "contains tab",
			profile: "my\tprofile",
			wantErr: true,
			errMsg:  "cannot contain whitespace",
		},
		{
			name:    "too long 65 chars",
			profile: strings.Repeat("a", 65),
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "exactly 64 chars is ok",
			profile: strings.Repeat("b", 64),
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProfileName(tc.profile)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSaveAndLoad
// ---------------------------------------------------------------------------

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	original := &Profile{
		APIKey:       "sk-dfir-testkey123456789",
		APIURL:       "https://custom.api/v2",
		OutputFormat: "json",
		Timeout:      30 * time.Second,
		Concurrency:  10,
		NoColor:      true,
	}

	if err := Save("default", original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("default")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.APIURL != original.APIURL {
		t.Errorf("APIURL = %q, want %q", loaded.APIURL, original.APIURL)
	}
	if loaded.OutputFormat != original.OutputFormat {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, original.OutputFormat)
	}
	if loaded.Timeout != original.Timeout {
		t.Errorf("Timeout = %v, want %v", loaded.Timeout, original.Timeout)
	}
	if loaded.Concurrency != original.Concurrency {
		t.Errorf("Concurrency = %d, want %d", loaded.Concurrency, original.Concurrency)
	}
	if loaded.NoColor != original.NoColor {
		t.Errorf("NoColor = %v, want %v", loaded.NoColor, original.NoColor)
	}
}

// ---------------------------------------------------------------------------
// TestLoadNonexistent
// ---------------------------------------------------------------------------

func TestLoadNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	_, err := Load("default")
	if err == nil {
		t.Fatal("expected error loading from nonexistent config, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestSaveMultipleProfiles
// ---------------------------------------------------------------------------

func TestSaveMultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	defaultProfile := &Profile{
		APIKey:       "sk-dfir-defaultkey123456",
		APIURL:       "https://default.api/v1",
		OutputFormat: "table",
		Timeout:      60 * time.Second,
		Concurrency:  5,
		NoColor:      false,
	}

	stagingProfile := &Profile{
		APIKey:       "sk-dfir-stagingkey789012",
		APIURL:       "https://staging.api/v1",
		OutputFormat: "json",
		Timeout:      120 * time.Second,
		Concurrency:  3,
		NoColor:      true,
	}

	if err := Save("default", defaultProfile); err != nil {
		t.Fatalf("Save default failed: %v", err)
	}
	if err := Save("staging", stagingProfile); err != nil {
		t.Fatalf("Save staging failed: %v", err)
	}

	// Load and verify default profile.
	loadedDefault, err := Load("default")
	if err != nil {
		t.Fatalf("Load default failed: %v", err)
	}
	if loadedDefault.APIKey != defaultProfile.APIKey {
		t.Errorf("default APIKey = %q, want %q", loadedDefault.APIKey, defaultProfile.APIKey)
	}
	if loadedDefault.APIURL != defaultProfile.APIURL {
		t.Errorf("default APIURL = %q, want %q", loadedDefault.APIURL, defaultProfile.APIURL)
	}
	if loadedDefault.OutputFormat != defaultProfile.OutputFormat {
		t.Errorf("default OutputFormat = %q, want %q", loadedDefault.OutputFormat, defaultProfile.OutputFormat)
	}
	if loadedDefault.NoColor != defaultProfile.NoColor {
		t.Errorf("default NoColor = %v, want %v", loadedDefault.NoColor, defaultProfile.NoColor)
	}

	// Load and verify staging profile.
	loadedStaging, err := Load("staging")
	if err != nil {
		t.Fatalf("Load staging failed: %v", err)
	}
	if loadedStaging.APIKey != stagingProfile.APIKey {
		t.Errorf("staging APIKey = %q, want %q", loadedStaging.APIKey, stagingProfile.APIKey)
	}
	if loadedStaging.APIURL != stagingProfile.APIURL {
		t.Errorf("staging APIURL = %q, want %q", loadedStaging.APIURL, stagingProfile.APIURL)
	}
	if loadedStaging.OutputFormat != stagingProfile.OutputFormat {
		t.Errorf("staging OutputFormat = %q, want %q", loadedStaging.OutputFormat, stagingProfile.OutputFormat)
	}
	if loadedStaging.NoColor != stagingProfile.NoColor {
		t.Errorf("staging NoColor = %v, want %v", loadedStaging.NoColor, stagingProfile.NoColor)
	}
}

// ---------------------------------------------------------------------------
// TestDir
// ---------------------------------------------------------------------------

func TestDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	got := Dir()
	if got != tmpDir {
		t.Errorf("Dir() = %q, want %q", got, tmpDir)
	}
}

func TestDirFallsBackWithoutEnv(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	dir := Dir()
	if dir == "" {
		t.Error("Dir() returned empty string")
	}
	if !strings.HasSuffix(dir, "dfir-cli") {
		t.Errorf("Dir() = %q, expected it to end with 'dfir-cli'", dir)
	}
}

// ---------------------------------------------------------------------------
// TestPath
// ---------------------------------------------------------------------------

func TestPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	got := Path()
	want := filepath.Join(tmpDir, "config.yaml")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestExists
// ---------------------------------------------------------------------------

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	if Exists() {
		t.Error("Exists() = true before any config is written")
	}

	// Create the config file.
	p := DefaultProfile()
	p.APIKey = "sk-dfir-existscheck12345"
	if err := Save("default", p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !Exists() {
		t.Error("Exists() = false after config was written")
	}
}

// ---------------------------------------------------------------------------
// TestEnsureDir
// ---------------------------------------------------------------------------

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "nested", "config")
	t.Setenv("DFIR_LAB_CONFIG_DIR", configDir)

	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Stat failed after EnsureDir: %v", err)
	}
	if !info.IsDir() {
		t.Error("config path is not a directory")
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}
}

func TestEnsureDirIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Call twice; second call should not fail.
	if err := EnsureDir(); err != nil {
		t.Fatalf("first EnsureDir failed: %v", err)
	}
	if err := EnsureDir(); err != nil {
		t.Fatalf("second EnsureDir failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestSaveInvalidProfileName
// ---------------------------------------------------------------------------

func TestSaveInvalidProfileName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := DefaultProfile()

	if err := Save("", p); err != nil {
		// Empty profile name defaults to "default", so no error expected.
		// But let's verify it actually saved under "default".
		t.Logf("Save with empty name: %v (checking default fallback)", err)
	}

	if err := Save("bad.name", p); err == nil {
		t.Error("expected error for profile name with dots, got nil")
	}

	if err := Save("bad name", p); err == nil {
		t.Error("expected error for profile name with spaces, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestLoadEmptyProfileUsesDefault
// ---------------------------------------------------------------------------

func TestLoadEmptyProfileUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	original := &Profile{
		APIKey:       "sk-dfir-emptyprofile12345",
		APIURL:       "https://dfir-lab.ch/api/v1",
		OutputFormat: "table",
		Timeout:      60 * time.Second,
		Concurrency:  5,
	}

	if err := Save("default", original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load with empty string should fall back to "default".
	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load('') failed: %v", err)
	}

	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, original.APIKey)
	}
}

// ---------------------------------------------------------------------------
// TestLoadProfileNotFound
// ---------------------------------------------------------------------------

func TestLoadProfileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Save one profile so the config file exists.
	if err := Save("default", DefaultProfile()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Try to load a profile that does not exist.
	_, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestConfigFilePermissions
// ---------------------------------------------------------------------------

func TestConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := DefaultProfile()
	p.APIKey = "sk-dfir-permscheck123456"
	if err := Save("default", p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(Path())
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}

// ---------------------------------------------------------------------------
// TestWriteInitialConfig
// ---------------------------------------------------------------------------

func TestWriteInitialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := &Profile{
		APIKey:       "sk-dfir-initialconfig12345",
		APIURL:       "https://init.api/v1",
		OutputFormat: "json",
		Timeout:      45 * time.Second,
		Concurrency:  8,
		NoColor:      true,
	}

	if err := WriteInitialConfig("production", p); err != nil {
		t.Fatalf("WriteInitialConfig failed: %v", err)
	}

	// Config file should now exist.
	if !Exists() {
		t.Fatal("config file does not exist after WriteInitialConfig")
	}

	// Load the profile back.
	loaded, err := Load("production")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.APIKey != p.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, p.APIKey)
	}
	if loaded.APIURL != p.APIURL {
		t.Errorf("APIURL = %q, want %q", loaded.APIURL, p.APIURL)
	}
	if loaded.OutputFormat != p.OutputFormat {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, p.OutputFormat)
	}
}

func TestWriteInitialConfigEmptyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := DefaultProfile()
	p.APIKey = "sk-dfir-writeempty1234567"

	// Empty profile name should default to "default".
	if err := WriteInitialConfig("", p); err != nil {
		t.Fatalf("WriteInitialConfig with empty name failed: %v", err)
	}

	loaded, err := Load("default")
	if err != nil {
		t.Fatalf("Load default failed: %v", err)
	}
	if loaded.APIKey != p.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, p.APIKey)
	}
}

func TestWriteInitialConfigInvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := DefaultProfile()
	if err := WriteInitialConfig("bad.name", p); err == nil {
		t.Error("expected error for invalid profile name, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestSetActiveProfile
// ---------------------------------------------------------------------------

func TestSetActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Create two profiles first.
	p1 := DefaultProfile()
	p1.APIKey = "sk-dfir-profile1key12345"
	p2 := DefaultProfile()
	p2.APIKey = "sk-dfir-profile2key12345"

	if err := Save("default", p1); err != nil {
		t.Fatalf("Save default failed: %v", err)
	}
	if err := Save("staging", p2); err != nil {
		t.Fatalf("Save staging failed: %v", err)
	}

	// Set active profile to staging.
	if err := SetActiveProfile("staging"); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}

	// Load with empty string should now return the staging profile.
	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load('') failed: %v", err)
	}
	if loaded.APIKey != p2.APIKey {
		t.Errorf("APIKey = %q, want %q (staging)", loaded.APIKey, p2.APIKey)
	}
}

func TestSetActiveProfileNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Create a config file with one profile.
	if err := Save("default", DefaultProfile()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err := SetActiveProfile("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error %q does not contain 'does not exist'", err.Error())
	}
}

func TestSetActiveProfileInvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	err := SetActiveProfile("bad.name")
	if err == nil {
		t.Fatal("expected error for invalid profile name, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestListProfiles
// ---------------------------------------------------------------------------

func TestListProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p1 := &Profile{
		APIKey:       "sk-dfir-listprofile1key1",
		APIURL:       "https://one.api/v1",
		OutputFormat: "table",
		Timeout:      60 * time.Second,
		Concurrency:  5,
	}
	p2 := &Profile{
		APIKey:       "sk-dfir-listprofile2key2",
		APIURL:       "https://two.api/v1",
		OutputFormat: "json",
		Timeout:      30 * time.Second,
		Concurrency:  3,
		NoColor:      true,
	}

	if err := Save("default", p1); err != nil {
		t.Fatalf("Save default failed: %v", err)
	}
	if err := Save("staging", p2); err != nil {
		t.Fatalf("Save staging failed: %v", err)
	}

	profiles, active, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}

	if active != "default" {
		t.Errorf("active = %q, want %q", active, "default")
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}

	if profiles["default"].APIKey != p1.APIKey {
		t.Errorf("default APIKey = %q, want %q", profiles["default"].APIKey, p1.APIKey)
	}
	if profiles["staging"].APIKey != p2.APIKey {
		t.Errorf("staging APIKey = %q, want %q", profiles["staging"].APIKey, p2.APIKey)
	}
	if profiles["staging"].OutputFormat != "json" {
		t.Errorf("staging OutputFormat = %q, want %q", profiles["staging"].OutputFormat, "json")
	}
}

func TestListProfilesNoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	_, _, err := ListProfiles()
	if err == nil {
		t.Fatal("expected error when config file does not exist, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestApplyDefaults (via Save/Load with zero-value fields)
// ---------------------------------------------------------------------------

func TestApplyDefaultsOnLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Save a profile with only APIKey set; everything else zero.
	sparse := &Profile{
		APIKey: "sk-dfir-sparseprofile12345",
	}
	if err := Save("sparse", sparse); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("sparse")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Defaults should be applied for zero-value fields.
	if loaded.APIURL != "https://dfir-lab.ch/api/v1" {
		t.Errorf("APIURL = %q, want default", loaded.APIURL)
	}
	if loaded.OutputFormat != "table" {
		t.Errorf("OutputFormat = %q, want default", loaded.OutputFormat)
	}
	if loaded.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want default", loaded.Timeout)
	}
	if loaded.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want default", loaded.Concurrency)
	}
}

// ---------------------------------------------------------------------------
// TestDirXDGConfigHome
// ---------------------------------------------------------------------------

func TestDirXDGConfigHome(t *testing.T) {
	t.Setenv("DFIR_LAB_CONFIG_DIR", "")
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	got := Dir()
	want := filepath.Join(xdgDir, "dfir-cli")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestValidateProfileName additional edge cases
// ---------------------------------------------------------------------------

func TestValidateProfileName_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{
			name:    "single character",
			profile: "a",
			wantErr: false,
		},
		{
			name:    "hyphens and underscores",
			profile: "my-profile_v2",
			wantErr: false,
		},
		{
			name:    "numeric only",
			profile: "12345",
			wantErr: false,
		},
		{
			name:    "newline in name",
			profile: "bad\nname",
			wantErr: true,
		},
		{
			name:    "carriage return in name",
			profile: "bad\rname",
			wantErr: true,
		},
		{
			name:    "multiple dots",
			profile: "a.b.c",
			wantErr: true,
		},
		{
			name:    "leading space",
			profile: " leading",
			wantErr: true,
		},
		{
			name:    "trailing space",
			profile: "trailing ",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProfileName(tc.profile)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMaskAPIKey additional edge cases
// ---------------------------------------------------------------------------

func TestMaskAPIKey_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "exactly prefix length",
			key:  "sk-dfir-",
			want: "********",
		},
		{
			name: "prefix plus 1",
			key:  "sk-dfir-a",
			want: "*********",
		},
		{
			name: "prefix plus 3",
			key:  "sk-dfir-abc",
			want: "***********",
		},
		{
			name: "prefix plus exactly 4",
			key:  "sk-dfir-abcd",
			want: "************",
		},
		{
			name: "prefix plus 5 shows last 4",
			key:  "sk-dfir-abcde",
			want: "sk-dfir-*bcde",
		},
		{
			name: "very long key",
			key:  "sk-dfir-" + strings.Repeat("x", 100),
			want: "sk-dfir-" + strings.Repeat("*", 96) + "xxxx",
		},
		{
			name: "single char",
			key:  "x",
			want: "*",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskAPIKey(tc.key)
			if got != tc.want {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateAPIKeyFormat additional edge cases
// ---------------------------------------------------------------------------

func TestValidateAPIKeyFormat_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "just the prefix",
			key:     "sk-dfir-",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "prefix plus few chars",
			key:     "sk-dfir-abcdef",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "exactly minimum length",
			key:     "sk-dfir-abcdefghijkl",
			wantErr: false,
		},
		{
			name:    "exactly max boundary",
			key:     "sk-dfir-" + strings.Repeat("a", 120),
			wantErr: false,
		},
		{
			name:    "one over max",
			key:     "sk-dfir-" + strings.Repeat("a", 121),
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "lowercase prefix variant is invalid",
			key:     "SK-DFIR-abcdefghijklmnop",
			wantErr: true,
			errMsg:  "must start with",
		},
		{
			name:    "no prefix at all",
			key:     strings.Repeat("a", 30),
			wantErr: true,
			errMsg:  "must start with",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestConfigDirPermissions (0700 after EnsureDir)
// ---------------------------------------------------------------------------

func TestConfigDirPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "deep", "nested", "dir")
	t.Setenv("DFIR_LAB_CONFIG_DIR", configDir)

	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}
}

// ---------------------------------------------------------------------------
// TestMultipleProfilesCreateSwitchList
// ---------------------------------------------------------------------------

func TestMultipleProfilesCreateSwitchList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Create three profiles.
	for _, name := range []string{"default", "staging", "production"} {
		p := DefaultProfile()
		p.APIKey = "sk-dfir-" + name + "key12345"
		if err := Save(name, p); err != nil {
			t.Fatalf("Save %q failed: %v", name, err)
		}
	}

	// List profiles -- should have 3.
	profiles, active, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(profiles))
	}
	if active != "default" {
		t.Errorf("expected active=default, got %q", active)
	}

	// Switch to production.
	if err := SetActiveProfile("production"); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}

	// Verify active profile changed.
	_, active, err = ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if active != "production" {
		t.Errorf("expected active=production, got %q", active)
	}

	// Load with empty string should return the production profile.
	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load('') failed: %v", err)
	}
	if loaded.APIKey != "sk-dfir-productionkey12345" {
		t.Errorf("expected production key, got %q", loaded.APIKey)
	}
}

// ---------------------------------------------------------------------------
// TestLoadNonExistentProfileReturnsError
// ---------------------------------------------------------------------------

func TestLoadNonExistentProfileReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Create a config with default profile only.
	p := DefaultProfile()
	p.APIKey = "sk-dfir-onlydefault12345"
	if err := Save("default", p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load a profile that does not exist.
	_, err := Load("nonexistent-profile")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestApplyDefaults fills zero values correctly
// ---------------------------------------------------------------------------

func TestApplyDefaults_ZeroValues(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	// Save a profile with ALL zero values except APIKey.
	sparse := &Profile{
		APIKey: "sk-dfir-allzeros123456789",
	}
	if err := Save("zero", sparse); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("zero")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check each default is applied.
	if loaded.APIURL != "https://dfir-lab.ch/api/v1" {
		t.Errorf("APIURL = %q, want default", loaded.APIURL)
	}
	if loaded.OutputFormat != "table" {
		t.Errorf("OutputFormat = %q, want 'table'", loaded.OutputFormat)
	}
	if loaded.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", loaded.Timeout)
	}
	if loaded.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", loaded.Concurrency)
	}
	// NoColor should remain false (zero value is the same as default).
	if loaded.NoColor {
		t.Error("NoColor should be false")
	}
}

// ---------------------------------------------------------------------------
// TestConfigFileAtomicWrite (temp file renamed)
// ---------------------------------------------------------------------------

func TestConfigFileAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p := DefaultProfile()
	p.APIKey = "sk-dfir-atomicwrite12345"
	if err := Save("default", p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// After save, the temp file should NOT exist.
	tmpPath := filepath.Join(tmpDir, "config.yaml.tmp.yaml")
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temp file should not exist after successful write")
	}

	// The actual config file should exist.
	configPath := filepath.Join(tmpDir, "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("config file should not be empty")
	}
}

// ---------------------------------------------------------------------------
// TestDFIRLabConfigDirEnvOverride
// ---------------------------------------------------------------------------

func TestDFIRLabConfigDirEnvOverride(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", customDir)

	got := Dir()
	if got != customDir {
		t.Errorf("Dir() = %q, want %q", got, customDir)
	}

	// Save and load in the custom dir.
	p := DefaultProfile()
	p.APIKey = "sk-dfir-envoverride12345"
	if err := Save("default", p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("default")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.APIKey != p.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, p.APIKey)
	}

	// Verify the file is in the right directory.
	expectedPath := filepath.Join(customDir, "config.yaml")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("config file not in expected location %q: %v", expectedPath, err)
	}
}

// ---------------------------------------------------------------------------
// TestSaveOverwritesExistingProfile
// ---------------------------------------------------------------------------

func TestSaveOverwritesExistingProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFIR_LAB_CONFIG_DIR", tmpDir)

	p1 := DefaultProfile()
	p1.APIKey = "sk-dfir-originalkey12345"
	if err := Save("default", p1); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	p2 := DefaultProfile()
	p2.APIKey = "sk-dfir-updatedkey123456"
	p2.OutputFormat = "json"
	if err := Save("default", p2); err != nil {
		t.Fatalf("Save overwrite failed: %v", err)
	}

	loaded, err := Load("default")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.APIKey != p2.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, p2.APIKey)
	}
	if loaded.OutputFormat != "json" {
		t.Errorf("OutputFormat = %q, want 'json'", loaded.OutputFormat)
	}
}
