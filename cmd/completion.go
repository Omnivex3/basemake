package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion script for basemake.

To enable completions in your current shell:

  bash:
    source <(basemake completion bash)

  zsh:
    source <(basemake completion zsh)

  fish:
    basemake completion fish | source

  powershell:
    basemake completion powershell | Out-String | Invoke-Expression

To persist across sessions (bash):
    basemake completion bash > /etc/bash_completion.d/basemake`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		switch args[0] {
		case "bash":
			err = rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			err = rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			err = rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			err = rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell: %s (use bash, zsh, fish, or powershell)", args[0])
		}
		return err
	},
}
