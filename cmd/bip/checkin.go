package main

import (
	"fmt"
	"os"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var checkinCmd = &cobra.Command{
	Use:   "checkin",
	Short: "Check in on GitHub activity across tracked repos",
	Long: `Check in on GitHub activity across tracked repositories.

By default, shows only items where the "ball is in your court" - items
that need your attention or response. For items authored by someone else
with no activity in the window, the default filter requires some signal
of involvement: you are assigned, requested as reviewer, @mentioned in
the body, or have previously commented. Use --broad to restore the older
behavior (every teammate item counts as needing review), or --all to
disable ball-in-my-court filtering entirely.

The activity window defaults to the timestamp in .last-checkin.json (falling
back to 3 days if the file doesn't exist). Each run updates .last-checkin.json
so the next run picks up where you left off. Using --since overrides this
window and does NOT update .last-checkin.json.

Requires nexus_path configured in ~/.config/bip/config.yml.`,
	Run: runCheckin,
}

var (
	checkinSince     string
	checkinRepo      string
	checkinCategory  string
	checkinAll       bool
	checkinBroad     bool
	checkinSummarize bool
)

func init() {
	rootCmd.AddCommand(checkinCmd)

	checkinCmd.Flags().StringVar(&checkinSince, "since", "3d", "Time period (e.g., 2d, 12h, 1w); does not update .last-checkin.json")
	checkinCmd.Flags().StringVar(&checkinRepo, "repo", "", "Check single repo only")
	checkinCmd.Flags().StringVar(&checkinCategory, "category", "", "Check repos in category only (code, writing)")
	checkinCmd.Flags().BoolVar(&checkinAll, "all", false, "Show all activity (disable ball-in-my-court filtering)")
	checkinCmd.Flags().BoolVar(&checkinBroad, "broad", false, "Use the legacy broad filter (count every teammate item as needing review)")
	checkinCmd.Flags().BoolVar(&checkinSummarize, "summarize", false, "Generate LLM take-home summaries")
}

func runCheckin(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

	now := time.Now()

	// Determine the "since" time.
	// If --since was explicitly provided, use it.
	// Otherwise, read from .last-checkin.json, falling back to 3d.
	var since time.Time
	if cmd.Flags().Changed("since") {
		duration, err := flow.ParseDuration(checkinSince)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid --since value: %v\n", err)
			os.Exit(1)
		}
		since = now.Add(-duration)
	} else {
		lastCheckin := flow.ReadLastCheckin(nexusPath)
		if lastCheckin.IsZero() {
			duration, _ := flow.ParseDuration("3d")
			since = now.Add(-duration)
		} else {
			since = lastCheckin
		}
	}

	// Get repos to check
	var err error
	var repos []string
	if checkinRepo != "" {
		repos = []string{checkinRepo}
	} else if checkinCategory != "" {
		repos, err = flow.LoadReposByCategory(nexusPath, checkinCategory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		repos, err = flow.LoadAllRepos(nexusPath)
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

		issueComments, err := flow.FetchIssueComments(repo, since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch issue comments for %s: %v\n", repo, err)
		}
		prComments, err := flow.FetchPRComments(repo, since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch PR comments for %s: %v\n", repo, err)
		}
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

		// Fetch ALL PR reviews (no time filter) in a single batch call.
		// For display, only since-window reviews are added to allComments.
		// For ball-in-court, all reviews are included as actions.
		var allReviewComments []flow.GitHubComment
		if len(prs) > 0 {
			var prNumbers []int
			for _, pr := range prs {
				prNumbers = append(prNumbers, pr.Number)
			}
			allReviewComments = flow.FetchPRReviewsAsComments(repo, prNumbers, time.Time{})
			// Add only since-window reviews to display comments
			for _, rc := range allReviewComments {
				if !rc.UpdatedAt.Before(since) {
					allComments = append(allComments, rc)
				}
			}
		}

		// Convert to unified actions for ball-in-court filtering.
		// Only comments and reviews drive ball-in-court logic;
		// close/merge events are administrative, not conversational.
		allActions := flow.CommentsToActions(allComments)

		// Apply ball-in-my-court filtering if enabled
		if githubUser != "" {
			// Include ALL PR reviews (not just since-filtered) for ball-in-court,
			// since a review predating the window is still relevant. This may
			// duplicate since-window reviews already in allActions; duplicates
			// are harmless since only the last actor per item matters.
			allActions = append(allActions, flow.CommentsToActions(allReviewComments)...)

			// Enrich actions: for items with no actions at all, fetch their
			// last comment so ball-in-court doesn't fall through to the default.
			// This fixes the bug where the user's comment predates the since window.
			itemsForEnrich := append(append([]flow.GitHubItem{}, issues...), prs...)
			enriched := flow.EnrichActionsWithLastComments(repo, itemsForEnrich, allActions)
			allActions = append(allActions, enriched...)

			if checkinBroad {
				issues = flow.FilterByBallInCourt(issues, allActions, githubUser)
				prs = flow.FilterByBallInCourt(prs, allActions, githubUser)
			} else {
				// Strict filter: for teammate items with no window activity, require
				// some signal of involvement. Populate PR requested reviewers and
				// past commenters for items that would otherwise fall through.
				prs = enrichPRsWithRequestedReviewers(repo, prs)
				inv := flow.Involvement{
					Commenters: fetchCommentersForUnengaged(repo, issues, prs, allActions, githubUser),
				}
				issues = flow.FilterByBallInCourtStrict(issues, allActions, githubUser, inv)
				prs = flow.FilterByBallInCourtStrict(prs, allActions, githubUser, inv)
			}
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

	// Update state file with current timestamp (only when --since was not explicit)
	if !cmd.Flags().Changed("since") {
		if err := flow.WriteLastCheckin(nexusPath, now); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not update %s: %v\n", flow.StatePath(nexusPath), err)
		}
	}

	// Generate take-home summaries if requested
	if checkinSummarize && len(allItems) > 0 {
		fmt.Println("\n## Take-home Summaries")
		summaries, err := flow.GenerateTakehomeSummaries(allItems)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating summaries: %v\n", err)
		} else {
			printSummariesByRepo(allItems, summaries)
		}
	}
}

// enrichPRsWithRequestedReviewers populates RequestedReviewers on each PR via a
// single batched GraphQL call. On failure, PRs keep empty reviewer lists and a
// warning goes to stderr — the strict filter still runs using the other
// involvement signals (assignees, mentions, past comments).
func enrichPRsWithRequestedReviewers(repo string, prs []flow.GitHubItem) []flow.GitHubItem {
	if len(prs) == 0 {
		return prs
	}
	var numbers []int
	for _, p := range prs {
		numbers = append(numbers, p.Number)
	}
	reviewers, err := flow.FetchPRsRequestedReviewers(repo, numbers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch requested reviewers for %s: %v\n", repo, err)
		return prs
	}
	for i, pr := range prs {
		logins := reviewers[pr.Number]
		var users []flow.GitHubUser
		for _, l := range logins {
			users = append(users, flow.GitHubUser{Login: l})
		}
		prs[i].RequestedReviewers = users
	}
	return prs
}

// fetchCommentersForUnengaged returns commenter logins for items where there is
// no action from githubUser in the window. These are the items the strict
// filter would otherwise drop — fetching their commenters lets the filter keep
// items the user previously engaged with (stale engagement still counts).
//
// Only items that (a) have no actions from githubUser AND (b) are authored by
// someone other than githubUser are queried, since those are the only ones
// where past commenters change the outcome.
func fetchCommentersForUnengaged(repo string, issues, prs []flow.GitHubItem, actions []flow.ItemAction, githubUser string) map[int][]string {
	// Which items have the user already acted on in the window?
	userActed := make(map[int]bool)
	for _, a := range actions {
		if a.Actor == githubUser {
			userActed[a.ItemNumber] = true
		}
	}

	var targets []int
	for _, it := range issues {
		if it.User.Login == githubUser {
			continue
		}
		if userActed[it.Number] {
			continue
		}
		targets = append(targets, it.Number)
	}
	for _, it := range prs {
		if it.User.Login == githubUser {
			continue
		}
		if userActed[it.Number] {
			continue
		}
		targets = append(targets, it.Number)
	}

	if len(targets) == 0 {
		return map[int][]string{}
	}

	commenters, err := flow.FetchItemsCommenters(repo, targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch past commenters for %s: %v\n", repo, err)
		return map[int][]string{}
	}
	return commenters
}

// printSummariesByRepo prints summaries grouped by repo with clickable URLs.
func printSummariesByRepo(items []flow.ItemDetails, summaries flow.TakehomeSummary) {
	// Build lookup for item info (IsPR) by ref
	itemInfo := make(map[string]flow.ItemDetails)
	for _, item := range items {
		itemInfo[item.Ref] = item
	}

	// Group refs by repo (preserving order)
	type repoItems struct {
		repo string
		refs []string
	}
	var repoOrder []repoItems
	repoMap := make(map[string]int) // repo -> index in repoOrder

	for _, item := range items {
		ref := item.Ref
		// Extract repo from ref (e.g., "org/repo#123" -> "org/repo")
		repo := extractRepoFromRef(ref)
		if idx, exists := repoMap[repo]; exists {
			repoOrder[idx].refs = append(repoOrder[idx].refs, ref)
		} else {
			repoMap[repo] = len(repoOrder)
			repoOrder = append(repoOrder, repoItems{repo: repo, refs: []string{ref}})
		}
	}

	// Print grouped output
	for _, ri := range repoOrder {
		repoName := flow.ExtractRepoName(ri.repo)
		fmt.Printf("\n  %s\n\n", repoName)

		for _, ref := range ri.refs {
			summary, ok := summaries[ref]
			if !ok {
				continue
			}

			// Generate URL
			info := itemInfo[ref]
			number := extractNumberFromRef(ref)
			itemType := "issue"
			if info.IsPR {
				itemType = "pr"
			}
			url := flow.GitHubURL(ri.repo, number, itemType)

			fmt.Printf("  - %s\n", summary)
			fmt.Printf("    %s\n", url)
		}
	}
}

// extractRepoFromRef extracts "org/repo" from "org/repo#123".
func extractRepoFromRef(ref string) string {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '#' {
			return ref[:i]
		}
	}
	return ref
}

// extractNumberFromRef extracts 123 from "org/repo#123".
func extractNumberFromRef(ref string) int {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '#' {
			num := 0
			for _, ch := range ref[i+1:] {
				if ch >= '0' && ch <= '9' {
					num = num*10 + int(ch-'0')
				}
			}
			return num
		}
	}
	return 0
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
