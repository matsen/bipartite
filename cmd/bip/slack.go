package main

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// Exit codes specific to slack commands (from spec.md)
const (
	ExitSlackMissingToken    = 1 // SLACK_BOT_TOKEN not set
	ExitSlackChannelNotFound = 2 // Channel not in configuration
	ExitSlackNotMember       = 3 // Bot not member of channel
)

var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slack channel integration commands",
	Long: `Commands for reading from Slack channels.

Fetch message history, list configured channels, and analyze team activity.
Requires SLACK_BOT_TOKEN environment variable with channels:history,
channels:read, and users:read scopes.

All commands output JSON by default for agent consumption.
Use --human flag for human-readable output.`,
}

func init() {
	// Load .env file if present (for SLACK_BOT_TOKEN)
	_ = godotenv.Load()

	rootCmd.AddCommand(slackCmd)
}
