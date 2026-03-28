package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ForeGuards/dfir-cli/internal/output"
	"github.com/ForeGuards/dfir-cli/internal/update"
	"github.com/ForeGuards/dfir-cli/internal/version"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates and returns the "update" command.
func NewUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install updates",
		Long: `Check for a newer version of dfir-cli.

Use --check to only check without showing install instructions. When a
newer version is available, the command displays installation instructions
for your platform.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(checkOnly)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")

	return cmd
}

func runUpdate(checkOnly bool) error {
	currentVersion := version.Version

	// Show current version
	fmt.Fprintf(os.Stderr, "Current version: %s\n", currentVersion)

	if currentVersion == "dev" {
		return fmt.Errorf("cannot update a development build. Install from a release instead")
	}

	// Check for updates with a spinner
	spin := output.NewSpinner("Checking for updates...")
	output.StartSpinner(spin)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	release, err := update.CheckForUpdate(ctx, currentVersion)
	output.StopSpinner(spin)

	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release == nil {
		fmt.Println()
		output.Green.Println("You are already running the latest version.")
		return nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	fmt.Println()
	fmt.Printf("  New version available: %s → %s\n", currentVersion, output.Green.Sprint(latestVersion))
	if release.HTMLURL != "" {
		fmt.Printf("  Release notes: %s\n", output.Dim.Sprint(release.HTMLURL))
	}
	fmt.Println()

	if checkOnly {
		fmt.Println("Run 'dfir-cli update' to install the update.")
		return nil
	}

	// Show update instructions (actual binary replacement is complex and
	// platform-specific — for now, guide the user to reinstall)
	fmt.Println("To update, run the appropriate command for your installation method:")
	fmt.Println()
	fmt.Println("  Homebrew:     brew upgrade dfir-cli")
	fmt.Println("  Linux:        curl -fsSL https://dfir-lab.ch/install.sh | sh")
	fmt.Println("  Windows:      iwr https://dfir-lab.ch/install.ps1 | iex")
	fmt.Println("  Go install:   go install github.com/ForeGuards/dfir-cli/cmd/dfir-cli@latest")
	fmt.Println()

	return nil
}
