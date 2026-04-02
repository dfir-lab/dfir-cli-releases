package commands

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/config"
	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/dfir-lab/dfir-cli/internal/update"
	"github.com/dfir-lab/dfir-cli/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd *cobra.Command

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dfir-cli",
		Short: "DFIR Lab CLI — Threat intelligence from the command line",
		Long: `DFIR Lab CLI wraps the DFIR Platform API to bring threat intelligence
directly into your terminal. Analyze phishing campaigns, scan for credential
exposures, and enrich indicators of compromise (IOCs) — all from the command line.

Capabilities include:
  - Phishing analysis: analyze emails and investigate URLs via phishing toolkit commands
  - Exposure scanning: search for leaked credentials across breach datasets
  - IOC enrichment: look up domains, IPs, hashes, and emails against curated threat feeds

Authenticate with an API key from https://platform.dfir-lab.ch and start investigating.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			v, _ := cmd.Flags().GetBool("verbose")
			q, _ := cmd.Flags().GetBool("quiet")
			if v && q {
				return fmt.Errorf("--verbose and --quiet cannot be used together")
			}

			// Handle --json / -j shorthand.
			j, _ := cmd.Flags().GetBool("json")
			if j && q {
				return fmt.Errorf("--json and --quiet cannot be used together")
			}
			if j {
				if f := cmd.Flags().Lookup("output"); f != nil {
					_ = f.Value.Set("json")
				}
			}

			// Warn when --api-key is passed on the command line.
			if f := cmd.Flags().Lookup("api-key"); f != nil && f.Changed {
				fmt.Fprintln(os.Stderr, "Warning: passing --api-key on the command line exposes it in process listings and shell history.")
				fmt.Fprintln(os.Stderr, "         Prefer: export DFIR_LAB_API_KEY=sk-dfir-...")
			}

			// Configure color output globally.
			noColor, _ := cmd.Flags().GetBool("no-color")
			if noColor || viper.GetBool("no_color") {
				output.SetNoColor(true)
			} else if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
				output.SetNoColor(true)
			}

			// Auto-detect non-TTY: disable colors and switch to JSON output.
			if !output.IsTerminal() {
				output.SetNoColor(true)
				// Auto-switch to JSON if the user didn't explicitly choose a format.
				if f := cmd.Flags().Lookup("output"); f != nil && !f.Changed {
					_ = f.Value.Set("json")
				}
			}

			return nil
		},
		Version: version.Short(),
	}
}

func init() {
	rootCmd = newRootCmd()

	// Global persistent flags
	pflags := rootCmd.PersistentFlags()

	pflags.String("api-key", "", "Override API key for this invocation")
	pflags.String("api-url", "", "Override API base URL (default from config)")
	pflags.StringP("output", "o", "table", "Output format: table, json, jsonl, csv")
	_ = rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "jsonl", "csv"}, cobra.ShellCompDirectiveNoFileComp
	})
	pflags.BoolP("json", "j", false, "Shorthand for --output json")
	pflags.Bool("no-color", false, "Disable colored output")
	pflags.BoolP("verbose", "v", false, "Show debug information (HTTP requests/responses)")
	pflags.BoolP("quiet", "q", false, "Minimal output")
	pflags.Duration("timeout", 60*time.Second, "HTTP request timeout")
	pflags.StringP("profile", "p", "default", "Named config profile")
	_ = rootCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return profileCompletionCandidates(), cobra.ShellCompDirectiveNoFileComp
	})

	// Bind flags to environment variables via Viper
	_ = viper.BindPFlag("api_key", pflags.Lookup("api-key"))
	_ = viper.BindPFlag("api_url", pflags.Lookup("api-url"))
	_ = viper.BindPFlag("profile", pflags.Lookup("profile"))
	_ = viper.BindPFlag("timeout", pflags.Lookup("timeout"))
	_ = viper.BindPFlag("no_color", pflags.Lookup("no-color"))

	_ = viper.BindEnv("api_key", "DFIR_LAB_API_KEY")
	_ = viper.BindEnv("api_url", "DFIR_LAB_API_URL")
	_ = viper.BindEnv("profile", "DFIR_LAB_PROFILE")
	_ = viper.BindEnv("timeout", "DFIR_LAB_TIMEOUT")
	_ = viper.BindEnv("no_color", "DFIR_LAB_NO_COLOR", "NO_COLOR")

	// Register subcommands — Phase 1
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewCompletionCmd())

	// Phase 2 commands
	rootCmd.AddCommand(NewEnrichmentCmd())
	rootCmd.AddCommand(NewPhishingCmd())
	rootCmd.AddCommand(NewExposureCmd())
	rootCmd.AddCommand(NewCreditsCmd())
	rootCmd.AddCommand(NewUsageCmd())

	// Phase 3 commands — AI
	rootCmd.AddCommand(NewAICmd())

	// Phase 6 commands
	rootCmd.AddCommand(NewUpdateCmd())

	// Custom help template
	rootCmd.SetUsageTemplate(usageTemplate)
}

// ---------------------------------------------------------------------------
// Execute is the single entry-point called by main.go.
// ---------------------------------------------------------------------------

// RootCmd returns the root command for use by documentation generators.
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the root command and returns any error.
func Execute() error {
	// Start background update check (non-blocking).
	updateCh := update.RunBackgroundCheck(version.Short())

	err := rootCmd.Execute()

	// After command completes, check if an update notice is available.
	select {
	case release := <-updateCh:
		update.PrintUpdateNotice(release)
	default:
		// Background check hasn't finished — don't block.
	}

	return err
}

// ---------------------------------------------------------------------------
// Global flag access helpers
// ---------------------------------------------------------------------------

// loadProfileCached loads the config profile, caching the result for the
// duration of the process. Returns nil if the config file does not exist.
var (
	cachedProfile     *config.Profile
	cachedProfileOnce sync.Once
)

func loadProfile() *config.Profile {
	cachedProfileOnce.Do(func() {
		p, err := config.Load(GetProfile())
		if err != nil {
			return
		}
		cachedProfile = p
	})
	return cachedProfile
}

// GetAPIKey returns the API key using the following precedence:
// flag > env var > keychain > config file.
func GetAPIKey() string {
	// 1. Explicit flag
	if v := rootCmd.PersistentFlags().Lookup("api-key"); v != nil && v.Changed {
		return v.Value.String()
	}
	// 2. Environment variable
	if key := viper.GetString("api_key"); key != "" {
		return key
	}
	// 3. Config file profile (Load already resolves keychain source)
	if p := loadProfile(); p != nil && p.APIKey != "" {
		return p.APIKey
	}
	return ""
}

// GetAPIURL returns the API base URL from flag, env var, or config file.
func GetAPIURL() string {
	// 1. Explicit flag
	if v := rootCmd.PersistentFlags().Lookup("api-url"); v != nil && v.Changed {
		return config.NormalizeAPIURL(v.Value.String())
	}
	// 2. Environment variable
	if url := viper.GetString("api_url"); url != "" {
		return config.NormalizeAPIURL(url)
	}
	// 3. Config file profile
	if p := loadProfile(); p != nil && p.APIURL != "" {
		return config.NormalizeAPIURL(p.APIURL)
	}
	return config.NormalizeAPIURL("")
}

// GetAIAPIURL returns the API base URL used by AI chat requests.
func GetAIAPIURL() string {
	// 1. Explicit flag
	if v := rootCmd.PersistentFlags().Lookup("api-url"); v != nil && v.Changed {
		return config.NormalizeAIAPIURL(v.Value.String())
	}
	// 2. Environment variable
	if url := viper.GetString("api_url"); url != "" {
		return config.NormalizeAIAPIURL(url)
	}
	// 3. Config file profile
	if p := loadProfile(); p != nil && p.APIURL != "" {
		return config.NormalizeAIAPIURL(p.APIURL)
	}
	return config.NormalizeAIAPIURL("")
}

// GetAuthValidateAPIURL returns the API base URL used for auth validation.
func GetAuthValidateAPIURL() string {
	// 1. Explicit flag
	if v := rootCmd.PersistentFlags().Lookup("api-url"); v != nil && v.Changed {
		return config.NormalizeAuthValidateAPIURL(v.Value.String())
	}
	// 2. Environment variable
	if url := viper.GetString("api_url"); url != "" {
		return config.NormalizeAuthValidateAPIURL(url)
	}
	// 3. Config file profile
	if p := loadProfile(); p != nil && p.APIURL != "" {
		return config.NormalizeAuthValidateAPIURL(p.APIURL)
	}
	return config.NormalizeAuthValidateAPIURL("")
}

// GetOutputFormat returns the selected output format (table, json, jsonl, csv).
func GetOutputFormat() string {
	val, _ := rootCmd.PersistentFlags().GetString("output")
	return val
}

// IsVerbose returns true when --verbose / -v is set.
func IsVerbose() bool {
	val, _ := rootCmd.PersistentFlags().GetBool("verbose")
	return val
}

// IsQuiet returns true when --quiet / -q is set.
func IsQuiet() bool {
	val, _ := rootCmd.PersistentFlags().GetBool("quiet")
	return val
}

// GetTimeout returns the configured HTTP request timeout.
func GetTimeout() time.Duration {
	// 1. Explicit flag
	if v := rootCmd.PersistentFlags().Lookup("timeout"); v != nil && v.Changed {
		d, _ := rootCmd.PersistentFlags().GetDuration("timeout")
		return d
	}
	// 2. Environment variable
	d := viper.GetDuration("timeout")
	if d != 0 {
		return d
	}
	// 3. Config file profile
	if p := loadProfile(); p != nil && p.Timeout != 0 {
		return p.Timeout
	}
	return 60 * time.Second
}

// GetProfile returns the active config profile name.
func GetProfile() string {
	if v := rootCmd.PersistentFlags().Lookup("profile"); v != nil && v.Changed {
		return v.Value.String()
	}
	p := viper.GetString("profile")
	if p == "" {
		return "default"
	}
	return p
}

// IsNoColor returns true when colored output should be suppressed.
func IsNoColor() bool {
	if val, _ := rootCmd.PersistentFlags().GetBool("no-color"); val {
		return true
	}
	if viper.GetBool("no_color") {
		return true
	}
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return true
	}
	return false
}

// profileCompletionCandidates returns known profile names for shell completion.
// It always includes "default" and, when config can be read, all configured profiles.
func profileCompletionCandidates() []string {
	candidates := map[string]struct{}{
		"default": {},
	}
	if profiles, _, err := config.ListProfiles(); err == nil {
		for name := range profiles {
			if name != "" {
				candidates[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(candidates))
	for name := range candidates {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Custom help / usage template
// ---------------------------------------------------------------------------

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <command> [flags]{{end}}
{{if not .HasParent}}
DFIR Lab CLI — Threat intelligence from the command line.

COMMANDS:
  phishing      Analyse phishing emails and URLs
  exposure      Search for leaked credentials across breach datasets
  enrichment    Enrich IOCs — domains, IPs, hashes, and emails

AI ASSISTANT:
  ai            AI-powered DFIR analysis and chat (Starter+)

ACCOUNT:
  credits       View cached API credit balance from the last API call
  usage         Display locally recorded API usage statistics

CONFIGURATION:
  config        Manage CLI configuration and profiles

OTHER:
  version       Show version and build information
  completion    Generate shell completion scripts
  update        Check for and install updates
{{else}}{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{rpad .Name .NamePadding}} {{.Short}}
{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
FLAGS:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

GLOBAL FLAGS:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
{{if not .HasParent}}
GETTING STARTED:
  $ dfir-cli config init                         Set up your API key
  $ dfir-cli enrichment lookup --ip 1.2.3.4      Enrich an IP address
  $ dfir-cli phishing analyze --file email.eml    Analyse a suspicious email
  $ dfir-cli exposure scan --domain example.com   Scan for exposures
  $ dfir-cli usage --period current               Review locally recorded usage this month
  $ dfir-cli ai "What artifacts show persistence?" Ask the AI assistant

LEARN MORE:
  https://platform.dfir-lab.ch/docs
{{end}}`
