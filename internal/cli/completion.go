package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCommand returns a cobra command for shell completion script generation.
func NewCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autocomplete [shell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bash, zsh, fish, or PowerShell.

Examples:
  # Bash (Linux)
  orch-cli autocomplete bash > /etc/bash_completion.d/orch-cli

  # Zsh
  orch-cli autocomplete zsh > "${fpath[1]}/_orch-cli"

  # Fish
  orch-cli autocomplete fish > ~/.config/fish/completions/orch-cli.fish

  # PowerShell
  orch-cli autocomplete powershell | Out-String | Invoke-Expression
`,
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
