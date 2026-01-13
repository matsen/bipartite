package main

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

Bash:
  $ source <(bp completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ bp completion bash > /etc/bash_completion.d/bp
  # macOS:
  $ bp completion bash > $(brew --prefix)/etc/bash_completion.d/bp

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ bp completion zsh > "${fpath[1]}/_bp"

  # You may need to start a new shell for this setup to take effect.

Fish:
  $ bp completion fish | source

  # To load completions for each session, execute once:
  $ bp completion fish > ~/.config/fish/completions/bp.fish

PowerShell:
  PS> bp completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> bp completion powershell > bp.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
