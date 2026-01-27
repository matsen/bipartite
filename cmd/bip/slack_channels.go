package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var slackChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List configured Slack channels",
	Long: `List all Slack channels configured in sources.json.

Shows channel name, ID, and purpose for each configured channel.

Examples:
  bip slack channels
  bip slack channels --human`,
	Args: cobra.NoArgs,
	RunE: runSlackChannels,
}

func init() {
	slackCmd.AddCommand(slackChannelsCmd)
}

func runSlackChannels(cmd *cobra.Command, args []string) error {
	channels, err := flow.LoadSlackChannels()
	if err != nil {
		return outputSlackError(1, "config_error", err.Error())
	}

	// Build response
	var channelInfos []flow.ChannelInfo
	for name, config := range channels {
		channelInfos = append(channelInfos, flow.ChannelInfo{
			Name:    name,
			ID:      config.ID,
			Purpose: config.Purpose,
		})
	}

	// Sort by name for consistent output
	sort.Slice(channelInfos, func(i, j int) bool {
		return channelInfos[i].Name < channelInfos[j].Name
	})

	response := flow.ChannelsResponse{
		Channels: channelInfos,
	}

	// Output
	if humanOutput {
		return outputSlackChannelsHuman(response)
	}
	return outputSlackChannelsJSON(response)
}

func outputSlackChannelsJSON(response flow.ChannelsResponse) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

func outputSlackChannelsHuman(response flow.ChannelsResponse) error {
	fmt.Println("# Configured Slack Channels")
	fmt.Println()

	if len(response.Channels) == 0 {
		fmt.Println("No channels configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tPURPOSE")
	fmt.Fprintln(w, "----\t--\t-------")
	for _, ch := range response.Channels {
		fmt.Fprintf(w, "%s\t%s\t%s\n", ch.Name, ch.ID, ch.Purpose)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d channels\n", len(response.Channels))
	return nil
}
