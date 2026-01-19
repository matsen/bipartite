package main

import (
	"github.com/spf13/cobra"
)

// Exit codes specific to asta commands (from contracts/cli.md)
const (
	ExitAstaNotFound  = 1 // Paper not found in Semantic Scholar
	ExitAstaDuplicate = 2 // Paper already exists (without --update)
	ExitAstaAPIError  = 3 // API error (rate limit, network)
)

var astaCmd = &cobra.Command{
	Use:   "asta",
	Short: "Semantic Scholar (ASTA) integration commands",
	Long: `Commands for integrating with Semantic Scholar's Academic Graph API.

Add papers by DOI, explore citation graphs, discover literature gaps,
and link preprints to published versions.

All commands output JSON by default for agent consumption.
Use --human flag for human-readable output.`,
}

func init() {
	rootCmd.AddCommand(astaCmd)
}
