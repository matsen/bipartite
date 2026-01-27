package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var (
	slackHistoryDays  int
	slackHistorySince string
	slackHistoryLimit int
)

var slackHistoryCmd = &cobra.Command{
	Use:   "history <channel>",
	Short: "Fetch message history from a Slack channel",
	Long: `Fetch recent messages from a configured Slack channel.

The channel must be configured in sources.json under slack.channels.
The bot must be a member of the channel to read messages.

Examples:
  bip slack history fortnight-goals
  bip slack history fortnight-goals --days 7
  bip slack history fortnight-goals --since 2025-01-13
  bip slack history fortnight-goals --human
  bip slack history fortnight-goals --limit 50`,
	Args: cobra.ExactArgs(1),
	RunE: runSlackHistory,
}

func init() {
	slackCmd.AddCommand(slackHistoryCmd)
	slackHistoryCmd.Flags().IntVar(&slackHistoryDays, "days", 14, "Number of days to fetch")
	slackHistoryCmd.Flags().StringVar(&slackHistorySince, "since", "", "Start date (YYYY-MM-DD), overrides --days")
	slackHistoryCmd.Flags().IntVar(&slackHistoryLimit, "limit", 100, "Maximum messages to return")
}

func runSlackHistory(cmd *cobra.Command, args []string) error {
	channelName := args[0]

	// Get channel configuration
	channelConfig, err := flow.GetSlackChannel(channelName)
	if err != nil {
		return outputSlackError(ExitSlackChannelNotFound, "channel_not_found", err.Error())
	}

	// Create Slack client
	client, err := flow.NewSlackClient()
	if err != nil {
		return outputSlackError(ExitSlackMissingToken, "missing_token", err.Error())
	}

	// Load user cache first (or fetch if empty)
	if _, err := client.GetUsers(); err != nil {
		// Non-fatal: we can still show messages with user IDs
		fmt.Fprintf(os.Stderr, "Warning: could not load users: %v\n", err)
	}

	// Calculate time range
	var oldest time.Time
	var startDate string

	if slackHistorySince != "" {
		// Parse --since date
		t, err := time.Parse("2006-01-02", slackHistorySince)
		if err != nil {
			return outputSlackError(1, "invalid_date", fmt.Sprintf("invalid date format '%s'; use YYYY-MM-DD", slackHistorySince))
		}
		oldest = t
		startDate = slackHistorySince
	} else {
		// Use --days
		oldest = time.Now().AddDate(0, 0, -slackHistoryDays)
		startDate = oldest.Format("2006-01-02")
	}

	// Fetch history
	messages, err := client.GetChannelHistory(channelConfig.ID, oldest, slackHistoryLimit)
	if err != nil {
		if strings.Contains(err.Error(), "not_in_channel") {
			return outputSlackError(ExitSlackNotMember, "not_member",
				fmt.Sprintf("Bot is not a member of channel '%s'. Invite the bot with /invite @bot-name", channelName))
		}
		return outputSlackError(1, "api_error", err.Error())
	}

	// Build response
	response := flow.HistoryResponse{
		Channel:   channelName,
		ChannelID: channelConfig.ID,
		Period: flow.Period{
			Start: startDate,
			End:   time.Now().Format("2006-01-02"),
		},
		Messages: messages,
	}

	// Output
	if humanOutput {
		return outputSlackHistoryHuman(response)
	}
	return outputSlackHistoryJSON(response)
}

func outputSlackHistoryJSON(response flow.HistoryResponse) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

func outputSlackHistoryHuman(response flow.HistoryResponse) error {
	fmt.Printf("# Channel: %s\n", response.Channel)
	fmt.Printf("Period: %s to %s\n\n", response.Period.Start, response.Period.End)

	if len(response.Messages) == 0 {
		fmt.Println("No messages found in this period.")
		return nil
	}

	// Group messages by date
	byDate := make(map[string][]flow.Message)
	for _, msg := range response.Messages {
		byDate[msg.Date] = append(byDate[msg.Date], msg)
	}

	// Sort dates (newest first, matching Slack's order)
	var dates []string
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, date := range dates {
		fmt.Printf("## %s\n\n", date)
		for _, msg := range byDate[date] {
			// Truncate long messages for display
			text := msg.Text
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			fmt.Printf("**%s**: %s\n\n", msg.UserName, text)
		}
	}

	fmt.Printf("---\nTotal: %d messages\n", len(response.Messages))
	return nil
}

// SlackErrorResult is the JSON output for Slack errors.
type SlackErrorResult struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func outputSlackError(exitCode int, errorCode, message string) error {
	suggestion := ""
	switch errorCode {
	case "missing_token":
		suggestion = "Set SLACK_BOT_TOKEN environment variable with a bot token that has channels:history, channels:read, and users:read scopes"
	case "channel_not_found":
		suggestion = "Check that the channel is configured in sources.json under slack.channels"
	case "not_member":
		suggestion = "Invite the bot to the channel with /invite @bot-name"
	}

	result := SlackErrorResult{
		Error:      errorCode,
		Message:    message,
		Suggestion: suggestion,
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		if suggestion != "" {
			fmt.Fprintf(os.Stderr, "Suggestion: %s\n", suggestion)
		}
	} else {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}

	os.Exit(exitCode)
	return nil
}
