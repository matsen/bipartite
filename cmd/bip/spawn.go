package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/flow/spawn"
	"github.com/spf13/cobra"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [ref]",
	Short: "Spawn tmux window for GitHub issue or PR review",
	Long: `Spawn a tmux window for reviewing a GitHub issue or PR.

The ref can be:
  - org/repo#123 (issue or PR number)
  - https://github.com/org/repo/issues/123
  - https://github.com/org/repo/pull/123

Or use --prompt without a ref for adhoc sessions:
  - bip spawn --prompt "Explore the clamping question"

Requires:
  - Running inside tmux
  - Repository defined in sources.yml (unless using --prompt alone)
  - Local clone of the repository (unless using --prompt alone)`,
	Args: cobra.MaximumNArgs(1),
	Run:  runSpawn,
}

var spawnPrompt string

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().StringVar(&spawnPrompt, "prompt", "", "Custom prompt override")
}

func runSpawn(cmd *cobra.Command, args []string) {
	// Handle adhoc mode (--prompt without ref) - doesn't need nexus directory
	if len(args) == 0 {
		if spawnPrompt == "" {
			fmt.Fprintf(os.Stderr, "Error: Either provide a GitHub reference or use --prompt for adhoc sessions\n")
			os.Exit(1)
		}
		runAdhocSpawn()
		return
	}

	// Get nexus path (required for issue/PR lookups)
	nexusPath := config.MustGetNexusPath()

	// Parse GitHub reference
	ref := flow.ParseGitHubRef(args[0])
	if ref == nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid format. Expected org/repo#number or GitHub URL\n")
		os.Exit(1)
	}

	// Validate repo is in sources.yml and has local clone
	repoPath, found := flow.GetRepoLocalPath(nexusPath, ref.Repo)
	if !found {
		fmt.Fprintf(os.Stderr, "Error: Repo %s not found in sources.yml\n", ref.Repo)
		fmt.Fprintf(os.Stderr, "Add it to sources.yml under 'code' or 'writing' category\n")
		os.Exit(1)
	}

	// Check local clone exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Local clone not found at %s\n", repoPath)
		fmt.Fprintf(os.Stderr, "Clone it with: git clone git@github.com:%s.git %s\n", ref.Repo, repoPath)
		os.Exit(1)
	}

	// Detect item type if not known from URL
	itemType := ref.ItemType
	if itemType == "" {
		fmt.Fprintf(os.Stderr, "Detecting type for %s#%d...\n", ref.Repo, ref.Number)
		var err error
		itemType, err = flow.DetectItemType(ref.Repo, ref.Number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not find issue or PR #%d: %v\n", ref.Number, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  → %s\n", itemType)
	}

	// Fetch data
	var data *ItemData
	var err error
	if itemType == "pr" {
		data, err = fetchPRData(ref.Repo, ref.Number)
	} else {
		data, err = fetchIssueData(ref.Repo, ref.Number)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build window name
	repoName := flow.ExtractRepoName(ref.Repo)
	windowName := fmt.Sprintf("%s#%d", repoName, ref.Number)

	// Print spawning message first
	fmt.Printf("Spawning tmux window %s...\n", windowName)

	// Build prompt
	var prompt string
	if spawnPrompt != "" {
		prompt = buildCustomPrompt(ref.Repo, ref.Number, itemType, spawnPrompt)
	} else if itemType == "pr" {
		prompt = buildPRPrompt(ref.Repo, ref.Number, data)
	} else {
		prompt = buildIssuePrompt(ref.Repo, ref.Number, data)
	}

	// Add project context if available
	contextPath := flow.GetRepoContextPath(nexusPath, ref.Repo)
	if contextPath != "" {
		fmt.Printf("Context: %s\n", contextPath)
		if contextData, err := os.ReadFile(contextPath); err == nil {
			prompt = fmt.Sprintf("## Project Context\n\n%s\n\n---\n\n%s", string(contextData), prompt)
		}
	}

	// Create tmux window
	url := flow.GitHubURL(ref.Repo, ref.Number, itemType)
	spawnWindow(windowName, repoPath, prompt, url)

	// Print URL as last line for easy clicking
	fmt.Println(url)
}

func runAdhocSpawn() {
	windowName := fmt.Sprintf("adhoc-%s", time.Now().Format("2006-01-02-150405"))
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not get working directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Spawning tmux window %s...\n", windowName)
	spawnWindow(windowName, workDir, spawnPrompt, "")
}

// spawnWindow validates tmux, checks for duplicates, and creates the window.
func spawnWindow(windowName, workDir, prompt, url string) {
	if !spawn.IsInTmux() {
		fmt.Fprintf(os.Stderr, "Error: Must be running inside tmux\n")
		os.Exit(1)
	}

	if spawn.WindowExists(windowName) {
		fmt.Printf("Window %s already exists, skipping\n", windowName)
		return
	}

	if err := spawn.CreateWindow(windowName, workDir, prompt, url); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ItemData contains fetched issue/PR data.
type ItemData struct {
	Title     string
	Body      string
	State     string
	Author    string
	Labels    []string
	CreatedAt time.Time
	Comments  []CommentData
	// PR-specific
	Files     []FileData
	Reviews   []ReviewData
	Additions int
	Deletions int
	Commits   int
}

type CommentData struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

type FileData struct {
	Path      string
	Additions int
	Deletions int
}

type ReviewData struct {
	Author string
	State  string
	Body   string
}

func fetchIssueData(repo string, number int) (*ItemData, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "title,body,state,comments,labels,author,createdAt")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("fetching issue: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var raw struct {
		Title     string                  `json:"title"`
		Body      string                  `json:"body"`
		State     string                  `json:"state"`
		Author    struct{ Login string }  `json:"author"`
		CreatedAt time.Time               `json:"createdAt"`
		Labels    []struct{ Name string } `json:"labels"`
		Comments  []struct {
			Author    struct{ Login string } `json:"author"`
			Body      string                 `json:"body"`
			CreatedAt time.Time              `json:"createdAt"`
		} `json:"comments"`
	}

	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}

	data := &ItemData{
		Title:     raw.Title,
		Body:      raw.Body,
		State:     raw.State,
		Author:    raw.Author.Login,
		CreatedAt: raw.CreatedAt,
	}

	for _, l := range raw.Labels {
		data.Labels = append(data.Labels, l.Name)
	}

	for _, c := range raw.Comments {
		data.Comments = append(data.Comments, CommentData{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	return data, nil
}

func fetchPRData(repo string, number int) (*ItemData, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "title,body,state,comments,labels,author,createdAt,files,reviews,additions,deletions,commits")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("fetching PR: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var raw struct {
		Title     string                  `json:"title"`
		Body      string                  `json:"body"`
		State     string                  `json:"state"`
		Author    struct{ Login string }  `json:"author"`
		CreatedAt time.Time               `json:"createdAt"`
		Labels    []struct{ Name string } `json:"labels"`
		Comments  []struct {
			Author    struct{ Login string } `json:"author"`
			Body      string                 `json:"body"`
			CreatedAt time.Time              `json:"createdAt"`
		} `json:"comments"`
		Files []struct {
			Path      string `json:"path"`
			Additions int    `json:"additions"`
			Deletions int    `json:"deletions"`
		} `json:"files"`
		Reviews []struct {
			Author struct{ Login string } `json:"author"`
			State  string                 `json:"state"`
			Body   string                 `json:"body"`
		} `json:"reviews"`
		Additions int        `json:"additions"`
		Deletions int        `json:"deletions"`
		Commits   []struct{} `json:"commits"`
	}

	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("parsing PR: %w", err)
	}

	data := &ItemData{
		Title:     raw.Title,
		Body:      raw.Body,
		State:     raw.State,
		Author:    raw.Author.Login,
		CreatedAt: raw.CreatedAt,
		Additions: raw.Additions,
		Deletions: raw.Deletions,
		Commits:   len(raw.Commits),
	}

	for _, l := range raw.Labels {
		data.Labels = append(data.Labels, l.Name)
	}

	for _, c := range raw.Comments {
		data.Comments = append(data.Comments, CommentData{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	for _, f := range raw.Files {
		data.Files = append(data.Files, FileData{
			Path:      f.Path,
			Additions: f.Additions,
			Deletions: f.Deletions,
		})
	}

	for _, r := range raw.Reviews {
		data.Reviews = append(data.Reviews, ReviewData{
			Author: r.Author.Login,
			State:  r.State,
			Body:   r.Body,
		})
	}

	return data, nil
}

func buildIssuePrompt(repo string, number int, data *ItemData) string {
	body := data.Body
	if body == "" {
		body = "(No description)"
	}

	labelsStr := "(none)"
	if len(data.Labels) > 0 {
		labelsStr = strings.Join(data.Labels, ", ")
	}

	relativeCreated := flow.FormatRelativeTime(data.CreatedAt)
	commentsSection := formatComments(data.Comments)

	hasComments := len(data.Comments) > 0
	var taskSection string
	if hasComments {
		taskSection = `Your task:
1. Read the issue and all comments carefully
2. Prepare the user to respond to the latest comment
3. If anything is unclear, explore the codebase to understand it
4. Summarize the discussion and suggest a response

Do NOT make changes, close, or comment on the issue. Analysis only.`
	} else {
		taskSection = `Your task:
1. Read the issue carefully
2. Summarize what the issue is asking for
3. If anything is unclear from the issue itself, explore the codebase to understand it

Do NOT make changes, close, or comment on the issue. Analysis only.`
	}

	return fmt.Sprintf(`GitHub issue: %s
Repository: %s
URL: https://github.com/%s/issues/%d
State: %s
Author: %s
Labels: %s
Created: %s

## Issue Body
%s

%s

---

%s`, data.Title, repo, repo, number, data.State, data.Author, labelsStr, relativeCreated, body, commentsSection, taskSection)
}

func buildPRPrompt(repo string, number int, data *ItemData) string {
	body := data.Body
	if body == "" {
		body = "(No description)"
	}

	labelsStr := "(none)"
	if len(data.Labels) > 0 {
		labelsStr = strings.Join(data.Labels, ", ")
	}

	relativeCreated := flow.FormatRelativeTime(data.CreatedAt)
	commentsSection := formatComments(data.Comments)
	filesSection := formatFiles(data.Files)
	reviewsSection := formatReviews(data.Reviews)

	// Determine if user has engaged
	githubUser, _ := flow.GetGitHubUser()
	engaged := userHasEngaged(data, githubUser)

	var taskSection string
	if engaged {
		taskSection = `Your task:
1. Read the PR and all comments/reviews carefully
2. Prepare the user to respond to the latest activity
3. If anything is unclear, explore the codebase to understand it
4. Summarize the discussion and suggest a response

Do NOT approve, merge, comment, or make changes. Analysis only.`
	} else {
		taskSection = `Your task:
1. Check @CLAUDE.md in this repo for PR review guidelines and follow them
2. If no guidelines exist, review the PR for correctness, style, and potential issues
3. Summarize what the PR does and any concerns
4. Prepare a review for the user

Do NOT approve, merge, comment, or make changes. Analysis only.`
	}

	return fmt.Sprintf(`GitHub PR: %s
Repository: %s
URL: https://github.com/%s/pull/%d
State: %s
Author: %s
Labels: %s
Created: %s
Stats: +%d/-%d in %d commit(s)

## PR Description
%s

%s

%s

%s

---

%s`, data.Title, repo, repo, number, data.State, data.Author, labelsStr, relativeCreated,
		data.Additions, data.Deletions, data.Commits, body, filesSection, reviewsSection, commentsSection, taskSection)
}

func buildCustomPrompt(repo string, number int, itemType, customPrompt string) string {
	url := flow.GitHubURL(repo, number, itemType)
	itemLabel := "Issue"
	if itemType == "pr" {
		itemLabel = "PR"
	}
	return fmt.Sprintf(`GitHub %s: %s#%d
URL: %s

%s`, itemLabel, repo, number, url, customPrompt)
}

func formatComments(comments []CommentData) string {
	if len(comments) == 0 {
		return "(No comments)"
	}

	limit := 10
	display := comments
	header := fmt.Sprintf("(%d total)", len(comments))
	if len(comments) > limit {
		display = comments[len(comments)-limit:]
		header = fmt.Sprintf("(%d total, showing last %d)", len(comments), limit)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Comments %s\n", header))
	for _, c := range display {
		author := c.Author
		if author == "" {
			author = "unknown"
		}
		relTime := flow.FormatRelativeTime(c.CreatedAt)
		sb.WriteString(fmt.Sprintf("\n@%s (%s):\n%s\n", author, relTime, c.Body))
	}

	return sb.String()
}

func formatFiles(files []FileData) string {
	if len(files) == 0 {
		return "(No files changed)"
	}

	limit := 20
	display := files
	header := fmt.Sprintf("(%d files)", len(files))
	if len(files) > limit {
		display = files[:limit]
		header = fmt.Sprintf("(%d total, showing first %d)", len(files), limit)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Files Changed %s\n", header))
	for _, f := range display {
		sb.WriteString(fmt.Sprintf("  %s (+%d/-%d)\n", f.Path, f.Additions, f.Deletions))
	}

	return sb.String()
}

func formatReviews(reviews []ReviewData) string {
	if len(reviews) == 0 {
		return "(No reviews)"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Reviews (%d total)\n", len(reviews)))
	for _, r := range reviews {
		author := r.Author
		if author == "" {
			author = "unknown"
		}
		sb.WriteString(fmt.Sprintf("\n@%s: %s\n", author, r.State))
		if r.Body != "" {
			body := r.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s\n", body))
		}
	}

	return sb.String()
}

func userHasEngaged(data *ItemData, username string) bool {
	if username == "" {
		return false
	}

	for _, c := range data.Comments {
		if c.Author == username {
			return true
		}
	}

	for _, r := range data.Reviews {
		if r.Author == username {
			return true
		}
	}

	return false
}
