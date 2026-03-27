package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd creates and returns the completion command.
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Long: `Generate shell completion scripts for dfir-cli.

To install completions:

  bash:
    $ dfir-cli completion bash > /etc/bash_completion.d/dfir-cli

  zsh:
    $ dfir-cli completion zsh > "${fpath[1]}/_dfir-cli"

  fish:
    $ dfir-cli completion fish > ~/.config/fish/completions/dfir-cli.fish

  powershell:
    PS> dfir-cli completion powershell | Out-String | Invoke-Expression`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}
