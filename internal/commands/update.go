package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dfir-lab/dfir-cli/internal/output"
	"github.com/dfir-lab/dfir-cli/internal/update"
	"github.com/dfir-lab/dfir-cli/internal/version"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates and returns the "update" command.
func NewUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install updates",
		Long: `Check for a newer version of dfir-cli and install it.

Without flags, dfir-cli downloads and installs the latest version
automatically. Homebrew and Scoop installations are detected and
updated through their respective package managers.

Use --check to only check for updates without installing.`,
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

	fmt.Fprintf(os.Stderr, "Current version: %s\n", currentVersion)

	if currentVersion == "dev" {
		return fmt.Errorf("cannot update a development build. Install from a release instead")
	}

	// Check for updates with a spinner.
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
		// Show the single relevant update command for this platform.
		method := update.DetectInstallMethod()
		switch method {
		case update.InstallHomebrew:
			fmt.Println("  To update, run:")
			fmt.Println("    brew upgrade dfir-lab/tap/dfir-cli")
		case update.InstallScoop:
			fmt.Println("  To update, run:")
			fmt.Println("    scoop update dfir-cli")
		default:
			fmt.Println("  To update, run:")
			fmt.Println("    dfir-cli update")
		}
		return nil
	}

	// Perform the update.
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer updateCancel()

	if err := update.SelfUpdate(updateCtx, release, IsVerbose()); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println()
	output.Green.Printf("  Successfully updated to %s\n", latestVersion)
	fmt.Println()

	return nil
}
