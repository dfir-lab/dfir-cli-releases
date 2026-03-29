package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/config"
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
				fmt.Fprintln(os.Stderr, "\nYou can find your API key at: https://dfir-lab.ch/settings/api-keys")
				return fmt.Errorf("invalid API key: %w", err)
			}

			// Build default config and save.
			p := config.DefaultProfile()
			p.APIKey = apiKey

			if err := config.Save(profile, p); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println()
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

			fmt.Printf("Profile: %s\n\n", profile)
			fmt.Printf("  api-key:        %s\n", apiKey)
			fmt.Printf("  api-url:        %s\n", p.APIURL)
			fmt.Printf("  output-format:  %s\n", p.OutputFormat)
			fmt.Printf("  timeout:        %s\n", p.Timeout)
			fmt.Printf("  concurrency:    %d\n", p.Concurrency)
			fmt.Printf("  no-color:       %v\n", p.NoColor)
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
