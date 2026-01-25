package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Generate and post activity digest to Slack",
	Long: `Generate an LLM-summarized digest of GitHub activity for a channel.

Channels are defined in sources.json via the "channel" field on repos.
The digest can be posted to Slack if a webhook is configured.`,
	Run: runDigest,
}

var (
	digestChannel string
	digestSince   string
	digestPostTo  string
	digestRepos   string
	digestDryRun  bool
)

func init() {
	rootCmd.AddCommand(digestCmd)

	digestCmd.Flags().StringVar(&digestChannel, "channel", "", "Channel whose repos to scan (required)")
	digestCmd.Flags().StringVar(&digestSince, "since", "1w", "Time period to summarize (e.g., 1w, 2d, 12h)")
	digestCmd.Flags().StringVar(&digestPostTo, "post-to", "", "Override destination channel for posting")
	digestCmd.Flags().StringVar(&digestRepos, "repos", "", "Override repos to scan (comma-separated)")
	digestCmd.Flags().BoolVar(&digestDryRun, "dry-run", false, "Preview digest without posting to Slack")
	digestCmd.MarkFlagRequired("channel")
}

func runDigest(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	postTo := digestPostTo
	if postTo == "" {
		postTo = digestChannel
	}

	// Get repos to scan
	var repos []string
	var err error
	if digestRepos != "" {
		for _, r := range strings.Split(digestRepos, ",") {
			repos = append(repos, strings.TrimSpace(r))
		}
	} else {
		repos, err = flow.LoadReposByChannel(digestChannel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Validate we have repos
	if len(repos) == 0 {
		channels, _ := flow.ListChannels()
		if len(channels) == 0 {
			fmt.Println("No channels configured in sources.json.")
			fmt.Println("Add 'channel' field to repos in the 'code' section.")
			os.Exit(1)
		}
		fmt.Printf("No repos configured for channel '%s'.\n", digestChannel)
		fmt.Printf("Available channels: %s\n", strings.Join(channels, ", "))
		fmt.Println("Or use --repos to specify repos directly.")
		os.Exit(1)
	}

	// Check webhook is configured for destination (skip if dry-run)
	if !digestDryRun {
		webhookURL := flow.GetWebhookURL(postTo)
		if webhookURL == "" {
			fmt.Printf("No webhook configured for channel '%s'.\n", postTo)
			fmt.Printf("Set SLACK_WEBHOOK_%s in .env file.\n", strings.ToUpper(postTo))
			os.Exit(1)
		}
	}

	// Determine time range
	duration, err := flow.ParseDuration(digestSince)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid --since value: %v\n", err)
		os.Exit(1)
	}
	until := time.Now().UTC()
	since := until.Add(-duration)
	dateRange := flow.FormatDateRange(since, until)

	fmt.Printf("Generating digest for #%s (%s)...\n", digestChannel, dateRange)
	if postTo != digestChannel {
		fmt.Printf("(posting to #%s)\n", postTo)
	}

	fmt.Printf("Scanning %d repos...\n", len(repos))

	// Fetch activity
	items, err := fetchChannelActivity(repos, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d items\n", len(items))

	// Generate summary
	fmt.Println("Generating summary...")
	message, err := flow.GenerateDigestSummary(items, digestChannel, dateRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating summary: %v\n", err)
		os.Exit(1)
	}

	if message == "" {
		fmt.Println("Failed to generate summary")
		os.Exit(1)
	}

	// Print preview
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("DIGEST PREVIEW")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(message)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Post to Slack (skip if dry-run)
	if digestDryRun {
		fmt.Println("(dry-run: not posting to Slack)")
	} else {
		fmt.Printf("Posting to #%s...\n", postTo)
		if err := flow.SendDigest(postTo, message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Posted successfully!")
	}
}

func fetchChannelActivity(repos []string, since time.Time) ([]flow.DigestItem, error) {
	var items []flow.DigestItem

	for _, repo := range repos {
		allItems, err := flow.FetchIssues(repo, since)
		if err != nil {
			continue // Skip repos with errors
		}

		for _, item := range allItems {
			// Collect contributors
			contributors := make(map[string]bool)
			contributors[item.User.Login] = true

			commenters, _ := flow.FetchItemCommenters(repo, item.Number)
			for _, c := range commenters {
				contributors[c] = true
			}

			if item.IsPR {
				reviewers, _ := flow.FetchPRReviewers(repo, item.Number)
				for _, r := range reviewers {
					contributors[r] = true
				}
			}

			// Remove "unknown" and sort
			delete(contributors, "unknown")
			delete(contributors, "")
			var sortedContribs []string
			for c := range contributors {
				sortedContribs = append(sortedContribs, c)
			}
			sort.Strings(sortedContribs)

			items = append(items, flow.DigestItem{
				Ref:          fmt.Sprintf("%s#%d", repo, item.Number),
				Number:       item.Number,
				Title:        item.Title,
				Author:       item.User.Login,
				IsPR:         item.IsPR,
				State:        item.State,
				HTMLURL:      item.HTMLURL,
				CreatedAt:    item.CreatedAt.Format(time.RFC3339),
				UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
				Contributors: sortedContribs,
			})
		}
	}

	return items, nil
}
