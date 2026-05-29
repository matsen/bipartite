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

Environment Variables (take precedence over asta_api_key in config.yml):
  BIP_ASTA_API_KEY  Your ASTA API key (recommended, bip-scoped)
  ASTA_API_KEY      Your ASTA API key (fallback; also loaded from a .env file)

Without a key, requests are sent anonymously: the cheap endpoints work, but
the search endpoint may time out. Register at https://allenai.org/asta/resources/mcp`,
}

func init() {
	// Load .env file if present (for ASTA_API_KEY)
	_ = godotenv.Load()

	astaCmd.PersistentFlags().BoolVar(&astaHuman, "human", false, "Output human-readable format instead of JSON")
	rootCmd.AddCommand(astaCmd)
}
