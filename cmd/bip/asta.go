package main

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var astaHuman bool

var astaCmd = &cobra.Command{
	Use:   "asta",
	Short: "ASTA (Academic Search Tool API) commands",
	Long: `Commands for searching and exploring academic papers via ASTA MCP.

ASTA provides read-only access to Semantic Scholar's paper database with
powerful text snippet search capabilities.

All commands output JSON by default for agent consumption.
Use --human flag for human-readable output.

Environment Variables:
  ASTA_API_KEY  Your ASTA API key (required)`,
}

func init() {
	// Load .env file if present (for ASTA_API_KEY)
	_ = godotenv.Load()

	astaCmd.PersistentFlags().BoolVar(&astaHuman, "human", false, "Output human-readable format instead of JSON")
	rootCmd.AddCommand(astaCmd)
}
