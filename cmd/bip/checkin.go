package main

import (
	"fmt"
	"os"
	"time"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var checkinCmd = &cobra.Command{
	Use:   "checkin",
	Short: "Check in on GitHub activity across tracked repos",
	Long: `Check in on GitHub activity across tracked repositories.

By default, shows only items where the "ball is in your court" - items
that need your attention or response. Use --all to see all activity.

Requires sources.json in the current directory (run from nexus directory).`,
	Run: runCheckin,
}

var (
	checkinSince     string
	checkinRepo      string
	checkinCategory  string
	checkinAll       bool
	checkinSummarize bool
)

func init() {
	rootCmd.AddCommand(checkinCmd)

	checkinCmd.Flags().StringVar(&checkinSince, "since", "3d", "Time period (e.g., 2d, 12h, 1w)")
	checkinCmd.Flags().StringVar(&checkinRepo, "repo", "", "Check single repo only")
	checkinCmd.Flags().StringVar(&checkinCategory, "category", "", "Check repos in category only (code, writing)")
	checkinCmd.Flags().BoolVar(&checkinAll, "all", false, "Show all activity (disable ball-in-my-court filtering)")
	checkinCmd.Flags().BoolVar(&checkinSummarize, "summarize", false, "Generate LLM take-home summaries")
}

func runCheckin(cmd *cobra.Command, args []string) {
	// Validate nexus directory
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse duration
	duration, err := flow.ParseDuration(checkinSince)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid --since value: %v\n", err)
		os.Exit(1)
	}
	since := time.Now().Add(-duration)

	// Get repos to check
	var repos []string
	if checkinRepo != "" {
		repos = []string{checkinRepo}
	} else if checkinCategory != "" {
		repos, err = flow.LoadReposByCategory(checkinCategory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		repos, err = flow.LoadAllRepos()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if len(repos) == 0 {
		fmt.Println("No repos to check.")
		return
	}

	// Get GitHub user for ball-in-my-court filtering
	var githubUser string
	if !checkinAll {
		githubUser, err = flow.GetGitHubUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not get GitHub user: %v\n", err)
			fmt.Fprintf(os.Stderr, "Showing all activity (ball-in-my-court filtering disabled)\n\n")
		}
	}

	// Fetch and display activity
	var totalIssues, totalPRs, totalComments int
	var allItems []flow.ItemDetails

	for _, repo := range repos {
		items, err := flow.FetchIssues(repo, since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", repo, err)
			continue
		}

		issueComments, _ := flow.FetchIssueComments(repo, since)
		prComments, _ := flow.FetchPRComments(repo, since)
		allComments := append(issueComments, prComments...)

		// Split into issues and PRs
		var issues, prs []flow.GitHubItem
		for _, item := range items {
			if item.IsPR {
				prs = append(prs, item)
			} else {
				issues = append(issues, item)
			}
		}

		// Apply ball-in-my-court filtering if enabled
		if githubUser != "" {
			issues = flow.FilterByBallInCourt(issues, allComments, githubUser)
			prs = flow.FilterByBallInCourt(prs, allComments, githubUser)
			allComments = flow.FilterCommentsByItems(allComments, append(issues, prs...))
		}

		if len(issues) == 0 && len(prs) == 0 && len(allComments) == 0 {
			continue
		}

		fmt.Printf("## %s\n", repo)

		if len(issues) > 0 {
			printItems(issues, "Issues", since)
			totalIssues += len(issues)
		}

		if len(prs) > 0 {
			printItems(prs, "Pull Requests", since)
			totalPRs += len(prs)
		}

		if len(allComments) > 0 {
			printComments(allComments)
			totalComments += len(allComments)
		}

		fmt.Println()

		// Collect items for summarization
		if checkinSummarize {
			for _, item := range append(issues, prs...) {
				comments, _ := flow.FetchItemComments(repo, item.Number, 10)
				allItems = append(allItems, flow.ItemDetails{
					Ref:      fmt.Sprintf("%s#%d", repo, item.Number),
					Title:    item.Title,
					Author:   item.User.Login,
					Body:     item.Body,
					IsPR:     item.IsPR,
					State:    item.State,
					Comments: comments,
				})
			}
		}
	}

	// Print summary
	if totalIssues > 0 || totalPRs > 0 {
		fmt.Printf("---\nTotal: %d issues, %d PRs, %d comments\n", totalIssues, totalPRs, totalComments)
	} else {
		fmt.Println("No activity found.")
	}

	// Generate take-home summaries if requested
	if checkinSummarize && len(allItems) > 0 {
		fmt.Println("\n## Take-home Summaries")
		summaries, err := flow.GenerateTakehomeSummaries(allItems)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating summaries: %v\n", err)
		} else {
			for ref, summary := range summaries {
				fmt.Printf("  %s: %s\n", ref, summary)
			}
		}
	}
}

func printItems(items []flow.GitHubItem, label string, since time.Time) {
	fmt.Printf("\n### %s (%d)\n", label, len(items))

	limit := 10
	for i, item := range items {
		if i >= limit {
			fmt.Printf("  ... and %d more\n", len(items)-limit)
			break
		}

		marker := "upd"
		if item.CreatedAt.After(since) {
			marker = "NEW"
		}

		timeAgo := flow.FormatTimeAgo(item.UpdatedAt)
		fmt.Printf("  [%s] %s - %s (%s)\n", marker, item.HTMLURL, item.Title, timeAgo)
	}
}

func printComments(comments []flow.GitHubComment) {
	fmt.Printf("\n### Comments (%d)\n", len(comments))

	// Group by item
	byItem := make(map[int][]flow.GitHubComment)
	for _, c := range comments {
		num := getItemNumber(c)
		byItem[num] = append(byItem[num], c)
	}

	limit := 10
	count := 0
	for itemNum, itemComments := range byItem {
		if count >= limit {
			fmt.Printf("  ... and %d more items with comments\n", len(byItem)-limit)
			break
		}

		// Get URL from first comment
		url := itemComments[0].HTMLURL
		// Strip comment anchor
		if idx := len(url) - 1; idx > 0 {
			for i := len(url) - 1; i >= 0; i-- {
				if url[i] == '#' {
					url = url[:i]
					break
				}
			}
		}

		fmt.Printf("  #%d: %d new comment(s)\n", itemNum, len(itemComments))

		for j, c := range itemComments {
			if j >= 3 {
				break
			}
			timeAgo := flow.FormatTimeAgo(c.UpdatedAt)
			preview := c.Body
			if len(preview) > 80 {
				preview = preview[:80]
			}
			preview = oneLine(preview)
			fmt.Printf("    @%s (%s): %s...\n", c.User.Login, timeAgo, preview)
		}

		count++
	}
}

func getItemNumber(c flow.GitHubComment) int {
	url := c.IssueURL
	if url == "" {
		url = c.PRURL
	}
	if url == "" {
		return 0
	}

	// Extract number from URL
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			num := 0
			for _, ch := range url[i+1:] {
				if ch >= '0' && ch <= '9' {
					num = num*10 + int(ch-'0')
				}
			}
			return num
		}
	}
	return 0
}

func oneLine(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if c == '\n' || c == '\r' {
			result = append(result, ' ')
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
