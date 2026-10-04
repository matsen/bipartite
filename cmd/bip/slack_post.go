package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var slackPostCmd = &cobra.Command{
	Use:   "post <channel> <text>",
	Short: "Post a message to a channel through its webhook",
	Long: `Post text to a Slack channel through the channel's incoming webhook.

The webhook URL is looked up as BIP_SLACK_WEBHOOK_<CHANNEL> (then
SLACK_WEBHOOK_<CHANNEL>) in the environment, then under the same names in
~/.config/bip/secrets.env, then as slack_webhooks.<channel> in
~/.config/bip/config.yml. <CHANNEL> is the channel name upper-cased with
non-alphanumerics replaced by '_'. The URL itself is never printed.

Examples:
  bip slack post dasm2 "Narrative digest: https://github.com/..."`,
	Args: cobra.MinimumNArgs(2),
	RunE: runSlackPost,
}

func init() {
	slackCmd.AddCommand(slackPostCmd)
}

// SlackPostResult is the JSON output of bip slack post.
type SlackPostResult struct {
	Channel string `json:"channel"`
	Posted  bool   `json:"posted"`
}

func runSlackPost(cmd *cobra.Command, args []string) error {
	channel, text := args[0], strings.Join(args[1:], " ")
	if err := flow.SendDigest(channel, text); err != nil {
		return outputSlackError(ExitError, "post_failed", err.Error())
	}
	if humanOutput {
		fmt.Printf("Posted to #%s\n", channel)
		return nil
	}
	return outputJSON(SlackPostResult{Channel: channel, Posted: true})
}
