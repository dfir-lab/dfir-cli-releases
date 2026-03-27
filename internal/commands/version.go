package commands

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/ForeGuards/dfir-cli/internal/version"
	"github.com/spf13/cobra"
)

// versionJSON represents the version information in a structured format.
type versionJSON struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	Commit    string `json:"commit"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// NewVersionCmd creates and returns the version subcommand.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print version and build information",
		Aliases: []string{"ver"},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, _ := cmd.Root().PersistentFlags().GetString("output")

			if outputFormat == "json" {
				info := versionJSON{
					Version:   version.Version,
					BuildDate: version.Date,
					Commit:    version.Commit,
					GoVersion: runtime.Version(),
					OS:        runtime.GOOS,
					Arch:      runtime.GOARCH,
				}

				data, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal version info: %w", err)
				}

				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), version.BuildInfo())
			return nil
		},
	}

	return cmd
}
