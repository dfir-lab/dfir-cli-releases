package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/client"
	"github.com/dfir-lab/dfir-cli/internal/config"
	"github.com/dfir-lab/dfir-cli/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// validConfigKeys lists all config keys that can be set/get.
var validConfigKeys = []string{
	"api-key",
	"api-url",
	"output-format",
	"timeout",
	"concurrency",
	"no-color",
	"ai-model",
}

// validOutputFormats lists accepted values for the output-format key.
var validOutputFormats = []string{"table", "json", "jsonl", "csv"}

// NewConfigCmd builds and returns the "config" command tree.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration and authentication",
	}

	cmd.AddCommand(
		newConfigInitCmd(),
		newConfigSetCmd(),
		newConfigGetCmd(),
		newConfigListCmd(),
	)

	return cmd
}

// ---------------------------------------------------------------------------
// config init
// ---------------------------------------------------------------------------

func newConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive first-run configuration wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := GetProfile()

			// Check if config already exists.
			if config.Exists() && !force {
				// Try to load the specific profile — if it exists, ask to overwrite.
				if _, err := config.Load(profile); err == nil {
					fmt.Printf("Configuration already exists for profile %q.\n", profile)
					fmt.Print("Overwrite? [y/N]: ")

					scanner := bufio.NewScanner(os.Stdin)
					if scanner.Scan() {
						answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
						if answer != "y" && answer != "yes" {
							fmt.Println("Aborted.")
							return nil
						}
					}
				}
			}

			// Prompt for API key.
			fmt.Println()
			fmt.Println("Welcome to dfir-cli configuration!")
			fmt.Println("----------------------------------")
			fmt.Println()
			fmt.Printf("Setting up profile: %s\n\n", profile)
			fmt.Print("Enter your API key: ")

			var apiKey string
			if term.IsTerminal(int(os.Stdin.Fd())) {
				// Terminal: read without echo for security.
				keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println() // newline after hidden input
				if err != nil {
					return fmt.Errorf("reading API key: %w", err)
				}
				apiKey = strings.TrimSpace(string(keyBytes))
			} else {
				// Piped input: read normally.
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					apiKey = strings.TrimSpace(scanner.Text())
				}
			}

			if apiKey == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			// Validate API key format.
			if err := config.ValidateAPIKeyFormat(apiKey); err != nil {
				fmt.Fprintln(os.Stderr, "\nYou can find your API key at: https://platform.dfir-lab.ch/settings/api-keys")
				return fmt.Errorf("invalid API key: %w", err)
			}
			if err := validateAPIKeyWithPlatform(apiKey); err != nil {
				return err
			}

			// Build default config and save.
			p := config.DefaultProfile()
			p.APIKey = apiKey

			if err := saveInitProfile(profile, p); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println()
			switch p.APIKeySource {
			case "keychain":
				fmt.Println("API key stored in system keychain.")
			default:
				fmt.Println("API key stored in config file.")
			}
			fmt.Printf("Configuration saved successfully! (profile: %s)\n", profile)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  dfir-cli config list                    Show current configuration")
			fmt.Println("  dfir-cli config set KEY VALUE           Change a setting")
			fmt.Println("  dfir-cli enrichment lookup --ip 8.8.8.8 Run your first enrichment")
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration without prompting")

	return cmd
}

// ---------------------------------------------------------------------------
// config set
// ---------------------------------------------------------------------------

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			profile := GetProfile()

			// Validate key.
			if !isValidKey(key) {
				return fmt.Errorf("unknown config key %q. Valid keys: %s",
					key, strings.Join(validConfigKeys, ", "))
			}

			// Load existing config or create defaults.
			p, err := config.Load(profile)
			if err != nil {
				// If config doesn't exist, start from defaults.
				p = config.DefaultProfile()
			}

			// Validate and apply value.
			if err := applyConfigValue(p, key, value); err != nil {
				return err
			}

			// Save.
			if err := config.Save(profile, p); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			displayValue := value
			if key == "api-key" {
				displayValue = config.MaskAPIKey(value)
				switch p.APIKeySource {
				case "keychain":
					fmt.Println("API key stored in system keychain.")
				default:
					fmt.Println("API key stored in config file.")
				}
			}
			fmt.Printf("Set %q to %q (profile: %s)\n", key, displayValue, profile)
			return nil
		},
	}

	return cmd
}

// ---------------------------------------------------------------------------
// config get
// ---------------------------------------------------------------------------

func newConfigGetCmd() *cobra.Command {
	var unmask bool

	cmd := &cobra.Command{
		Use:   "get KEY",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			profile := GetProfile()

			if !isValidKey(key) {
				return fmt.Errorf("unknown config key %q. Valid keys: %s",
					key, strings.Join(validConfigKeys, ", "))
			}

			p, err := config.Load(profile)
			if err != nil {
				return fmt.Errorf("config file not found. Run: dfir-cli config init")
			}

			val := getConfigValue(p, key)
			if key == "api-key" && !unmask {
				val = config.MaskAPIKey(val)
			}

			fmt.Println(val)
			return nil
		},
	}

	cmd.Flags().BoolVar(&unmask, "unmask", false, "Show full API key without masking")

	return cmd
}

// ---------------------------------------------------------------------------
// config list
// ---------------------------------------------------------------------------

func newConfigListCmd() *cobra.Command {
	var unmask bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := GetProfile()

			p, err := config.Load(profile)
			if err != nil {
				return fmt.Errorf("config file not found. Run: dfir-cli config init")
			}

			apiKey := p.APIKey
			if !unmask {
				apiKey = config.MaskAPIKey(apiKey)
			}

			apiKeySource := p.APIKeySource
			if apiKeySource == "" {
				apiKeySource = "config"
			}

			fmt.Printf("Profile: %s\n\n", profile)
			fmt.Printf("  api-key:        %s (source: %s)\n", apiKey, apiKeySource)
			fmt.Printf("  api-url:        %s\n", p.APIURL)
			fmt.Printf("  output-format:  %s\n", p.OutputFormat)
			fmt.Printf("  timeout:        %s\n", p.Timeout)
			fmt.Printf("  concurrency:    %d\n", p.Concurrency)
			fmt.Printf("  no-color:       %v\n", p.NoColor)
			aiModel := p.AIModel
			if aiModel == "" {
				aiModel = "sonnet"
			}
			fmt.Printf("  ai-model:       %s\n", aiModel)
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&unmask, "unmask", false, "Show full API key without masking")

	return cmd
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// authValidateAPIKey is a test seam for remote API key validation.
var authValidateAPIKey = func(ctx context.Context, apiKey, apiURL string, timeout time.Duration, verbose bool) error {
	c := client.New(apiKey, apiURL, version.UserAgent(), timeout, verbose)
	_, _, err := c.AuthValidate(ctx)
	return err
}

// saveInitProfile persists the profile during config init. On first run we
// write active_profile metadata; on existing configs we preserve other profiles.
func saveInitProfile(profile string, p *config.Profile) error {
	if config.Exists() {
		return config.Save(profile, p)
	}
	return config.WriteInitialConfig(profile, p)
}

// validateAPIKeyWithPlatform performs best-effort remote API key validation.
// Authentication/authorization failures block setup; transient network/server
// issues show a warning and fall back to local format validation.
func validateAPIKeyWithPlatform(apiKey string) error {
	timeout := GetTimeout()
	if timeout <= 0 || timeout > 15*time.Second {
		timeout = 15 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := authValidateAPIKey(ctx, apiKey, GetAPIURL(), timeout, IsVerbose())
	if err == nil {
		return nil
	}

	if isFatalAPIKeyValidationError(err) {
		return err
	}

	fmt.Fprintf(os.Stderr, "Warning: could not validate API key with DFIR Platform (%v). Continuing with local validation only.\n", err)
	return nil
}

// isFatalAPIKeyValidationError reports whether config init should fail when
// remote validation returns err.
func isFatalAPIKeyValidationError(err error) bool {
	var authErr *client.AuthenticationError
	if errors.As(err, &authErr) {
		return true
	}

	var authorizationErr *client.AuthorizationError
	return errors.As(err, &authorizationErr)
}

// isValidKey returns true if key is in the valid config keys list.
func isValidKey(key string) bool {
	for _, k := range validConfigKeys {
		if k == key {
			return true
		}
	}
	return false
}

// getConfigValue returns the string representation of a config key's value.
func getConfigValue(p *config.Profile, key string) string {
	switch key {
	case "api-key":
		return p.APIKey
	case "api-url":
		return p.APIURL
	case "output-format":
		return p.OutputFormat
	case "timeout":
		return p.Timeout.String()
	case "concurrency":
		return fmt.Sprintf("%d", p.Concurrency)
	case "no-color":
		return fmt.Sprintf("%v", p.NoColor)
	case "ai-model":
		val := p.AIModel
		if val == "" {
			val = "sonnet"
		}
		return val
	default:
		return ""
	}
}

// applyConfigValue validates and sets a value on the given profile.
func applyConfigValue(p *config.Profile, key, value string) error {
	switch key {
	case "api-key":
		if err := config.ValidateAPIKeyFormat(value); err != nil {
			return fmt.Errorf("invalid API key: %w", err)
		}
		p.APIKey = value

	case "api-url":
		if value == "" {
			return fmt.Errorf("api-url cannot be empty")
		}
		p.APIURL = value

	case "output-format":
		if !isValidOutputFormat(value) {
			return fmt.Errorf("invalid output format %q. Valid formats: %s",
				value, strings.Join(validOutputFormats, ", "))
		}
		p.OutputFormat = value

	case "timeout":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid timeout value %q: must be a valid duration (e.g., 30s, 2m): %w", value, err)
		}
		if d <= 0 {
			return fmt.Errorf("timeout must be a positive duration")
		}
		p.Timeout = d

	case "concurrency":
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil || n < 1 || n > 100 {
			return fmt.Errorf("invalid concurrency value %q: must be a positive integer between 1 and 100", value)
		}
		p.Concurrency = n

	case "no-color":
		switch strings.ToLower(value) {
		case "true", "1", "yes":
			p.NoColor = true
		case "false", "0", "no":
			p.NoColor = false
		default:
			return fmt.Errorf("invalid no-color value %q: must be true or false", value)
		}

	case "ai-model":
		v := strings.ToLower(value)
		if v != "haiku" && v != "sonnet" {
			return fmt.Errorf("ai-model must be 'haiku' or 'sonnet', got %q", value)
		}
		p.AIModel = v
	}

	return nil
}

// isValidOutputFormat checks whether format is in the allowed list.
func isValidOutputFormat(format string) bool {
	for _, f := range validOutputFormats {
		if f == format {
			return true
		}
	}
	return false
}
