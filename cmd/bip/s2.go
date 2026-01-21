package main

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// Exit codes specific to s2 commands (from contracts/cli.md)
const (
	ExitS2NotFound  = 1 // Paper not found in Semantic Scholar
	ExitS2Duplicate = 2 // Paper already exists (without --update)
	ExitS2APIError  = 3 // API error (rate limit, network)
)

var s2Cmd = &cobra.Command{
	Use:   "s2",
	Short: "Semantic Scholar (S2) integration commands",
	Long: `Commands for integrating with Semantic Scholar's Academic Graph API.

Add papers by DOI, explore citation graphs, discover literature gaps,
and link preprints to published versions.

All commands output JSON by default for agent consumption.
Use --human flag for human-readable output.`,
}

func init() {
	// Load .env file if present (for S2_API_KEY)
	_ = godotenv.Load()

	rootCmd.AddCommand(s2Cmd)
}
