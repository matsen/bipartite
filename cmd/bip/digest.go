package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Generate activity digest (preview only by default)",
	Long: `Generate an LLM-summarized digest of GitHub activity for a channel.

Channels are defined in sources.yml via the "channel" field on repos.
By default, shows a preview only. Use --post to actually send to Slack.`,
	Run: runDigest,
}

var (
	digestChannel string
	digestSince   string
	digestPostTo  string
	digestRepos   string
	digestPost    bool
	digestVerbose bool
)

func init() {
	rootCmd.AddCommand(digestCmd)

	digestCmd.Flags().StringVar(&digestChannel, "channel", "", "Channel whose repos to scan (required)")
	digestCmd.Flags().StringVar(&digestSince, "since", "1w", "Time period to summarize (e.g., 1w, 2d, 12h)")
	digestCmd.Flags().StringVar(&digestPostTo, "post-to", "", "Override destination channel for posting")
	digestCmd.Flags().StringVar(&digestRepos, "repos", "", "Override repos to scan (comma-separated)")
	digestCmd.Flags().BoolVar(&digestPost, "post", false, "Actually post to Slack (default: preview only)")
	digestCmd.Flags().BoolVar(&digestVerbose, "verbose", false, "Fetch PR/issue bodies and include LLM summaries")
	digestCmd.MarkFlagRequired("channel")
}

func runDigest(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

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
		repos, err = flow.LoadReposByChannel(nexusPath, digestChannel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: loading repos for channel %s: %v\n", digestChannel, err)
			os.Exit(1)
		}
	}

	// Validate we have repos
	if len(repos) == 0 {
		channels, _ := flow.ListChannels(nexusPath)
		if len(channels) == 0 {
			fmt.Println("No channels configured in sources.yml.")
			fmt.Println("Add 'channel' field to repos in the 'code' section.")
			os.Exit(1)
		}
		fmt.Printf("No repos configured for channel '%s'.\n", digestChannel)
		fmt.Printf("Available channels: %s\n", strings.Join(channels, ", "))
		fmt.Println("Or use --repos to specify repos directly.")
		os.Exit(1)
	}

	// Check webhook is configured for destination (only if posting)
	if digestPost {
		webhookURL := flow.GetWebhookURL(postTo)
		if webhookURL == "" {
			fmt.Printf("No webhook configured for channel '%s'.\n", postTo)
			fmt.Printf("Add to ~/.config/bip/config.yml: \"slack_webhooks\": {\"%s\": \"https://...\"}\n", postTo)
			fmt.Printf("Or set SLACK_WEBHOOK_%s environment variable.\n", strings.ToUpper(postTo))
			os.Exit(1)
		}
	}

	// Determine time range
	duration, err := flow.ParseDuration(digestSince)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid --since value: %v\n", err)
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

	// Fetch digest items from GitHub activity
	items, err := fetchDigestItems(repos, since, digestVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: building digest items: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d items\n", len(items))

	// Generate summaries if verbose mode
	if digestVerbose && len(items) > 0 {
		fmt.Println("Generating summaries with Claude Haiku...")
		items, err = flow.SummarizeDigestItems(items)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: generating summaries: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate summary
	fmt.Println("Generating summary...")
	message, err := flow.GenerateDigestSummary(items, digestChannel, dateRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generating digest summary: %v\n", err)
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

	// Post to Slack (only if --post flag is set)
	if digestPost {
		fmt.Printf("Posting to #%s...\n", postTo)
		if err := flow.SendDigest(postTo, message); err != nil {
			fmt.Fprintf(os.Stderr, "error: posting to Slack: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Posted successfully!")
	} else {
		fmt.Println("(preview only: use --post to send to Slack)")
	}
}

// fetchDigestItems fetches GitHub activity and transforms it into digest items.
// For each repo, it fetches issues/PRs updated since the given time, collects
// contributors (author, commenters, reviewers), and builds DigestItem structs.
// Returns an error if all repo fetches fail (to distinguish from "no activity").
func fetchDigestItems(repos []string, since time.Time, includeBody bool) ([]flow.DigestItem, error) {
	var items []flow.DigestItem
	var successfulFetches int

	for _, repo := range repos {
		allItems, err := flow.FetchIssues(repo, since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s: %v\n", repo, err)
			continue // Skip repos with errors
		}
		successfulFetches++

		for _, item := range allItems {
			// Collect contributors
			contributors := make(map[string]bool)
			contributors[item.User.Login] = true

			commenters, err := flow.FetchItemCommenters(repo, item.Number)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch commenters for %s#%d: %v\n", repo, item.Number, err)
			} else {
				for _, c := range commenters {
					contributors[c] = true
				}
			}

			if item.IsPR {
				reviewers, err := flow.FetchPRReviewers(repo, item.Number)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to fetch reviewers for %s#%d: %v\n", repo, item.Number, err)
				} else {
					for _, r := range reviewers {
						contributors[r] = true
					}
				}
			}

			// Remove "unknown" and sort
			delete(contributors, "unknown")
			delete(contributors, "")
			var sortedContributors []string
			for c := range contributors {
				sortedContributors = append(sortedContributors, c)
			}
			sort.Strings(sortedContributors)

			digestItem := flow.DigestItem{
				Ref:          fmt.Sprintf("%s#%d", repo, item.Number),
				Number:       item.Number,
				Title:        item.Title,
				Author:       item.User.Login,
				IsPR:         item.IsPR,
				State:        item.State,
				HTMLURL:      item.HTMLURL,
				CreatedAt:    item.CreatedAt.Format(time.RFC3339),
				UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
				Contributors: sortedContributors,
			}

			if includeBody {
				digestItem.Body = item.Body
			}

			items = append(items, digestItem)
		}
	}

	// Fail if all repos failed to fetch (distinguishes from "no activity")
	if successfulFetches == 0 && len(repos) > 0 {
		return nil, fmt.Errorf("failed to fetch activity from all %d repos", len(repos))
	}

	return items, nil
}
