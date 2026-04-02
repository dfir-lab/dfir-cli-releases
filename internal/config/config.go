// Package config manages CLI configuration for dfir-cli using Viper.
// It supports multiple named profiles, XDG-compliant config paths,
// and secure file permissions.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	appName           = "dfir-cli"
	configFileName    = "config"
	configFileType    = "yaml"
	configFileExt     = ".yaml"
	defaultProfileKey = "default"

	defaultAPIURL       = "https://dfir-lab.ch/api/v1"
	platformAPIURL      = "https://platform.dfir-lab.ch/api/v1"
	aiDefaultAPIURL     = defaultAPIURL
	authValidateAPIURL  = platformAPIURL
	defaultOutputFormat = "table"
	defaultTimeout      = 60 * time.Second
	defaultConcurrency  = 5

	apiKeyPrefix    = "sk-dfir-"
	apiKeyMinLength = 20
	apiKeyMaxLength = 128

	dirPerm  os.FileMode = 0700
	filePerm os.FileMode = 0600
)

// Profile holds the configuration for a single named profile.
type Profile struct {
	APIKey       string        `yaml:"api_key"        mapstructure:"api_key"`
	APIKeySource string        `yaml:"api_key_source" mapstructure:"api_key_source"` // "keychain", "config", or ""
	APIURL       string        `yaml:"api_url"        mapstructure:"api_url"`
	OutputFormat string        `yaml:"output_format"  mapstructure:"output_format"`
	Timeout      time.Duration `yaml:"timeout"        mapstructure:"timeout"`
	Concurrency  int           `yaml:"concurrency"    mapstructure:"concurrency"`
	NoColor      bool          `yaml:"no_color"       mapstructure:"no_color"`
	AIModel      string        `yaml:"ai_model"       mapstructure:"ai_model"`
}

// DefaultProfile returns a Profile populated with default values.
func DefaultProfile() *Profile {
	return &Profile{
		APIURL:       defaultAPIURL,
		OutputFormat: defaultOutputFormat,
		Timeout:      defaultTimeout,
		Concurrency:  defaultConcurrency,
		NoColor:      false,
	}
}

// Dir returns the configuration directory path.
//
// Resolution order:
//  1. $DFIR_LAB_CONFIG_DIR (if set)
//  2. $XDG_CONFIG_HOME/dfir-cli (macOS/Linux, if set)
//  3. os.UserConfigDir()/dfir-cli (platform default)
func Dir() string {
	if dir := os.Getenv("DFIR_LAB_CONFIG_DIR"); dir != "" {
		return dir
	}

	if runtime.GOOS != "windows" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, appName)
		}
	}

	base, err := os.UserConfigDir()
	if err != nil {
		// Fallback to ~/.config on Unix, which mirrors os.UserConfigDir behaviour.
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appName)
}

// Path returns the full path to the configuration file.
func Path() string {
	return filepath.Join(Dir(), configFileName+configFileExt)
}

// Exists reports whether the configuration file exists on disk.
func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

// EnsureDir creates the configuration directory (and parents) with 0700
// permissions if it does not already exist.
func EnsureDir() error {
	return os.MkdirAll(Dir(), dirPerm)
}

// validateProfileName checks that a profile name is safe for use as a Viper
// key segment (no dots, no whitespace, not empty, reasonable length).
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if strings.Contains(name, ".") {
		return fmt.Errorf("profile name cannot contain dots")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("profile name cannot contain whitespace")
	}
	if len(name) > 64 {
		return fmt.Errorf("profile name too long (max 64 characters)")
	}
	return nil
}

// Load reads the configuration file and returns the requested profile.
// If profile is empty, the active profile is used. If no active profile is
// set, "default" is assumed.
func Load(profile string) (*Profile, error) {
	v, err := readConfig()
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if profile == "" {
		profile = v.GetString("active_profile")
		if profile == "" {
			profile = defaultProfileKey
		}
	}

	if err := validateProfileName(profile); err != nil {
		return nil, err
	}

	key := "profiles." + profile
	if !v.IsSet(key) {
		return nil, fmt.Errorf("profile %q not found in config", profile)
	}

	p := DefaultProfile()
	sub := v.Sub(key)
	if sub == nil {
		return p, nil
	}

	if err := sub.Unmarshal(p); err != nil {
		return nil, fmt.Errorf("unmarshalling profile %q: %w", profile, err)
	}

	// Apply defaults for any zero-valued fields that should have defaults.
	applyDefaults(p)
	p.APIURL = NormalizeAPIURL(p.APIURL)

	// If the API key source is "keychain", try to retrieve from the system keychain.
	if p.APIKeySource == "keychain" {
		secret, err := GetKeychain(profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read API key from keychain: %v\n", err)
		}
		if secret != "" {
			p.APIKey = secret
		}
	}

	return p, nil
}

// Save writes the given profile into the configuration file under the
// specified profile name. If the config file does not exist it is created.
// If an API key is present, Save attempts to store it in the system keychain
// first. On success the plaintext key is cleared from the config file and
// api_key_source is set to "keychain". If the keychain is unavailable the key
// is stored in the config file with api_key_source set to "config".
func Save(profile string, p *Profile) error {
	if profile == "" {
		profile = defaultProfileKey
	}
	if err := validateProfileName(profile); err != nil {
		return err
	}

	v, err := readConfigOrNew()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	// Try to store the API key in the system keychain.
	if p.APIKey != "" {
		if err := SetKeychain(profile, p.APIKey); err == nil {
			// Keychain succeeded: clear plaintext from config.
			p.APIKeySource = "keychain"
			saved := *p
			saved.APIKey = ""
			saved.APIURL = NormalizeAPIURL(saved.APIURL)
			setProfileInViper(v, profile, &saved)
			return writeConfig(v)
		}
		// Keychain unavailable: fall back to storing in config file.
		p.APIKeySource = "config"
	}

	p.APIURL = NormalizeAPIURL(p.APIURL)
	setProfileInViper(v, profile, p)

	return writeConfig(v)
}

// SetActiveProfile sets the active_profile key in the configuration file.
func SetActiveProfile(name string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}
	v, err := readConfigOrNew()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	// Verify the profile exists.
	key := "profiles." + name
	if !v.IsSet(key) {
		return fmt.Errorf("profile %q does not exist", name)
	}

	v.Set("active_profile", name)
	return writeConfig(v)
}

// ListProfiles returns all configured profiles and the name of the currently
// active profile.
func ListProfiles() (map[string]*Profile, string, error) {
	v, err := readConfig()
	if err != nil {
		return nil, "", fmt.Errorf("reading config: %w", err)
	}

	active := v.GetString("active_profile")
	if active == "" {
		active = defaultProfileKey
	}

	profilesRaw := v.GetStringMap("profiles")
	profiles := make(map[string]*Profile, len(profilesRaw))

	for name := range profilesRaw {
		sub := v.Sub("profiles." + name)
		if sub == nil {
			profiles[name] = DefaultProfile()
			continue
		}
		p := DefaultProfile()
		if err := sub.Unmarshal(p); err != nil {
			return nil, "", fmt.Errorf("unmarshalling profile %q: %w", name, err)
		}
		applyDefaults(p)
		p.APIURL = NormalizeAPIURL(p.APIURL)

		// Resolve keychain-sourced API keys.
		if p.APIKeySource == "keychain" {
			if secret, err := GetKeychain(name); err == nil && secret != "" {
				p.APIKey = secret
			}
		}

		profiles[name] = p
	}

	return profiles, active, nil
}

// WriteInitialConfig creates the configuration file with the given profile as
// the active profile. The file is created with 0600 permissions and the
// directory is ensured to exist with 0700.
func WriteInitialConfig(profile string, p *Profile) error {
	if profile == "" {
		profile = defaultProfileKey
	}
	if err := validateProfileName(profile); err != nil {
		return err
	}

	if err := EnsureDir(); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	v := viper.New()
	v.Set("active_profile", profile)

	// Try to store the API key in the system keychain.
	if p.APIKey != "" {
		if err := SetKeychain(profile, p.APIKey); err == nil {
			p.APIKeySource = "keychain"
			saved := *p
			saved.APIKey = ""
			saved.APIURL = NormalizeAPIURL(saved.APIURL)
			setProfileInViper(v, profile, &saved)
			return writeConfig(v)
		}
		p.APIKeySource = "config"
	}

	p.APIURL = NormalizeAPIURL(p.APIURL)
	setProfileInViper(v, profile, p)

	return writeConfig(v)
}

// ValidateAPIKeyFormat checks that the given API key has the expected prefix
// and length.
func ValidateAPIKeyFormat(key string) error {
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return fmt.Errorf("API key must start with %q", apiKeyPrefix)
	}

	if len(key) < apiKeyMinLength {
		return fmt.Errorf("API key is too short (minimum %d characters)", apiKeyMinLength)
	}

	if len(key) > apiKeyMaxLength {
		return fmt.Errorf("API key is too long (maximum %d characters)", apiKeyMaxLength)
	}

	return nil
}

// MaskAPIKey returns a masked version of the API key, showing only the prefix
// and the last 4 characters (e.g. "sk-dfir-****...****f7a2").
// If the key is too short to mask meaningfully, the entire key is replaced
// with asterisks after the prefix.
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= len(apiKeyPrefix)+4 {
		// Too short to partially reveal — mask entirely.
		return strings.Repeat("*", len(key))
	}
	tail := key[len(key)-4:]
	masked := len(key) - len(apiKeyPrefix) - 4
	return apiKeyPrefix + strings.Repeat("*", masked) + tail
}

// NormalizeAPIURL resolves the live operational API base URL while leaving
// custom endpoints untouched. Older configs may still point at the platform
// host, which serves docs/app pages but not most API routes.
func NormalizeAPIURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAPIURL
	}

	raw = strings.TrimRight(raw, "/")
	if raw == platformAPIURL {
		return defaultAPIURL
	}
	if strings.HasPrefix(raw, platformAPIURL+"/") {
		return defaultAPIURL + strings.TrimPrefix(raw, platformAPIURL)
	}
	return raw
}

// NormalizeAIAPIURL resolves the base URL for AI chat requests.
func NormalizeAIAPIURL(raw string) string {
	return NormalizeAPIURL(raw)
}

// NormalizeAuthValidateAPIURL resolves the API base URL used for
// /auth/validate. That route is currently served from platform.dfir-lab.ch,
// even though the operational API routes are served from dfir-lab.ch.
func NormalizeAuthValidateAPIURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return authValidateAPIURL
	}

	raw = strings.TrimRight(raw, "/")
	switch {
	case raw == defaultAPIURL, raw == platformAPIURL:
		return authValidateAPIURL
	case strings.HasPrefix(raw, defaultAPIURL+"/"):
		return authValidateAPIURL + strings.TrimPrefix(raw, defaultAPIURL)
	default:
		return raw
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// readConfig creates a Viper instance and reads the config file.
func readConfig() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(Path())
	v.SetConfigType(configFileType)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	return v, nil
}

// readConfigOrNew reads the config if it exists, otherwise returns a fresh
// Viper instance.
func readConfigOrNew() (*viper.Viper, error) {
	if Exists() {
		return readConfig()
	}
	v := viper.New()
	v.SetConfigFile(Path())
	v.SetConfigType(configFileType)
	return v, nil
}

// writeConfig writes the Viper state to the config file with secure
// permissions. It uses an atomic write pattern (temp file + rename) to
// avoid a window where the file is world-readable.
func writeConfig(v *viper.Viper) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	path := Path()

	// Create temp file with an unpredictable name and restrictive permissions.
	f, err := os.CreateTemp(Dir(), "config-*.tmp.yaml")
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}
	tmpPath := f.Name()
	f.Close()

	// Write config content to the temp file.
	v.SetConfigFile(tmpPath)
	v.SetConfigType(configFileType)
	if err := v.WriteConfig(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("writing config file: %w", err)
	}

	// Enforce permissions (belt and suspenders).
	if err := os.Chmod(tmpPath, filePerm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting config file permissions: %w", err)
	}

	// Atomic rename into place.
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("moving config file into place: %w", err)
	}

	return nil
}

// setProfileInViper sets all profile fields under profiles.<name> in the
// given Viper instance.
func setProfileInViper(v *viper.Viper, name string, p *Profile) {
	prefix := "profiles." + name + "."
	v.Set(prefix+"api_key", p.APIKey)
	v.Set(prefix+"api_key_source", p.APIKeySource)
	v.Set(prefix+"api_url", p.APIURL)
	v.Set(prefix+"output_format", p.OutputFormat)
	v.Set(prefix+"timeout", p.Timeout.String())
	v.Set(prefix+"concurrency", p.Concurrency)
	v.Set(prefix+"no_color", p.NoColor)
	v.Set(prefix+"ai_model", p.AIModel)
}

// applyDefaults fills in default values for any zero-valued fields that have
// defined defaults.
func applyDefaults(p *Profile) {
	if p.APIURL == "" {
		p.APIURL = defaultAPIURL
	}
	p.APIURL = NormalizeAPIURL(p.APIURL)
	if p.OutputFormat == "" {
		p.OutputFormat = defaultOutputFormat
	}
	if p.Timeout == 0 {
		p.Timeout = defaultTimeout
	}
	if p.Concurrency == 0 {
		p.Concurrency = defaultConcurrency
	}
}
