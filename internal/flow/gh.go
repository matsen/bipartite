package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GHAPI calls the GitHub API via the gh CLI.
// Returns the parsed JSON response.
func GHAPI(endpoint string) (json.RawMessage, error) {
	cmd := exec.Command("gh", "api", endpoint, "--paginate")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api %s: %s", endpoint, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}

	if len(output) == 0 {
		return json.RawMessage("[]"), nil
	}

	// Handle paginated output (multiple JSON arrays concatenated)
	output = normalizeJSONLines(output)

	return output, nil
}

// normalizeJSONLines handles paginated gh api output.
// The --paginate flag can output multiple JSON arrays on separate lines.
func normalizeJSONLines(data []byte) json.RawMessage {
	// Try to parse as single JSON first
	var single interface{}
	if err := json.Unmarshal(data, &single); err == nil {
		return data
	}

	// Try parsing as multiple JSON arrays (one per line)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var combined []json.RawMessage
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(line), &arr); err == nil {
			combined = append(combined, arr...)
		} else {
			// Single object
			combined = append(combined, json.RawMessage(line))
		}
	}

	result, _ := json.Marshal(combined)
	return result
}

// GHGraphQL executes a GraphQL query via the gh CLI.
// Variables can be any type: strings use -f flag, other types (int, bool) use -F flag
// for proper GraphQL type handling.
func GHGraphQL(query string, variables map[string]interface{}) (json.RawMessage, error) {
	args := []string{"api", "graphql", "-f", "query=" + query}
	for key, value := range variables {
		switch v := value.(type) {
		case string:
			// Use -f for string values
			args = append(args, "-f", key+"="+v)
		case int, int64, float64, bool:
			// Use -F for non-string types (gh CLI handles type conversion)
			args = append(args, "-F", fmt.Sprintf("%s=%v", key, v))
		default:
			return nil, fmt.Errorf("unsupported variable type %T for key %s", value, key)
		}
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh graphql: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh graphql: %w", err)
	}

	return output, nil
}

// GetGitHubUser returns the current authenticated GitHub user's login.
func GetGitHubUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting GitHub user: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// FetchIssues fetches issues updated since the given time.
func FetchIssues(repo string, since time.Time) ([]GitHubItem, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	endpoint := fmt.Sprintf("/repos/%s/issues?state=all&since=%s&sort=updated&direction=desc&per_page=100", repo, sinceStr)

	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var rawItems []struct {
		Number      int       `json:"number"`
		Title       string    `json:"title"`
		Body        string    `json:"body"`
		State       string    `json:"state"`
		HTMLURL     string    `json:"html_url"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		User        GitHubUser
		PullRequest *struct{} `json:"pull_request,omitempty"`
		Labels      []GitHubLabel
	}

	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("parsing issues: %w", err)
	}

	var items []GitHubItem
	for _, raw := range rawItems {
		items = append(items, GitHubItem{
			Number:    raw.Number,
			Title:     raw.Title,
			Body:      raw.Body,
			State:     raw.State,
			HTMLURL:   raw.HTMLURL,
			CreatedAt: raw.CreatedAt,
			UpdatedAt: raw.UpdatedAt,
			User:      raw.User,
			IsPR:      raw.PullRequest != nil,
			Labels:    raw.Labels,
		})
	}

	return items, nil
}

// FetchIssueComments fetches issue comments since the given time.
func FetchIssueComments(repo string, since time.Time) ([]GitHubComment, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	endpoint := fmt.Sprintf("/repos/%s/issues/comments?since=%s&sort=updated&direction=desc&per_page=100", repo, sinceStr)

	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var comments []GitHubComment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}

	return comments, nil
}

// FetchPRComments fetches PR review comments since the given time.
func FetchPRComments(repo string, since time.Time) ([]GitHubComment, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	endpoint := fmt.Sprintf("/repos/%s/pulls/comments?since=%s&sort=updated&direction=desc&per_page=100", repo, sinceStr)

	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var comments []GitHubComment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("parsing PR comments: %w", err)
	}

	return comments, nil
}

// FetchIssue fetches a single issue by number.
func FetchIssue(repo string, number int) (*GitHubItem, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d", repo, number)
	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Number      int       `json:"number"`
		Title       string    `json:"title"`
		Body        string    `json:"body"`
		State       string    `json:"state"`
		HTMLURL     string    `json:"html_url"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		User        GitHubUser
		PullRequest *struct{} `json:"pull_request,omitempty"`
		Labels      []GitHubLabel
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}

	return &GitHubItem{
		Number:    raw.Number,
		Title:     raw.Title,
		Body:      raw.Body,
		State:     raw.State,
		HTMLURL:   raw.HTMLURL,
		CreatedAt: raw.CreatedAt,
		UpdatedAt: raw.UpdatedAt,
		User:      raw.User,
		IsPR:      raw.PullRequest != nil,
		Labels:    raw.Labels,
	}, nil
}

// GetIssueNodeID returns the GraphQL node ID for an issue.
func GetIssueNodeID(repo string, number int) (string, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d", repo, number)
	data, err := GHAPI(endpoint)
	if err != nil {
		return "", err
	}

	var result struct {
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	return result.NodeID, nil
}

// FetchItemComments fetches comments for a specific issue or PR.
func FetchItemComments(repo string, number int, limit int) ([]CommentSummary, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d/comments?per_page=%d", repo, number, limit)
	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var rawComments []struct {
		User      GitHubUser `json:"user"`
		Body      string     `json:"body"`
		CreatedAt time.Time  `json:"created_at"`
	}
	if err := json.Unmarshal(data, &rawComments); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}

	// Take the most recent comments
	var comments []CommentSummary
	start := 0
	if len(rawComments) > limit {
		start = len(rawComments) - limit
	}
	for _, c := range rawComments[start:] {
		comments = append(comments, CommentSummary{
			Author:    c.User.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	return comments, nil
}

// DetectItemType determines whether a GitHub number is an issue or PR.
func DetectItemType(repo string, number int) (string, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d", repo, number)
	data, err := GHAPI(endpoint)
	if err != nil {
		return "", err
	}

	var result struct {
		PullRequest *struct{} `json:"pull_request,omitempty"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if result.PullRequest != nil {
		return "pr", nil
	}
	return "issue", nil
}

// rawPRReview represents a single review from the GitHub API.
type rawPRReview struct {
	User        GitHubUser `json:"user"`
	SubmittedAt time.Time  `json:"submitted_at"`
	State       string     `json:"state"`
	Body        string     `json:"body"`
}

// fetchPRReviews fetches all reviews for a single PR.
func fetchPRReviews(repo string, number int) ([]rawPRReview, error) {
	endpoint := fmt.Sprintf("/repos/%s/pulls/%d/reviews", repo, number)
	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching reviews for %s#%d: %w", repo, number, err)
	}

	var reviews []rawPRReview
	if err := json.Unmarshal(data, &reviews); err != nil {
		return nil, fmt.Errorf("parsing reviews for %s#%d: %w", repo, number, err)
	}
	return reviews, nil
}

// FetchPRReviewers fetches deduplicated reviewer logins for a PR.
func FetchPRReviewers(repo string, number int) ([]string, error) {
	reviews, err := fetchPRReviews(repo, number)
	if err != nil {
		return nil, err
	}

	reviewerSet := make(map[string]bool)
	for _, r := range reviews {
		if r.User.Login != "" {
			reviewerSet[r.User.Login] = true
		}
	}

	var reviewers []string
	for login := range reviewerSet {
		reviewers = append(reviewers, login)
	}
	return reviewers, nil
}

// fetchPRReviewsBatch fetches reviews for multiple PRs in a single GraphQL call.
// Returns a map from PR number to its reviews. PRs that fail to fetch are omitted.
func fetchPRReviewsBatch(repo string, prNumbers []int) (map[int][]rawPRReview, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/name", repo)
	}
	owner, name := parts[0], parts[1]

	// Build aliased query: one pullRequest alias per PR number.
	var fragments []string
	for _, n := range prNumbers {
		fragments = append(fragments, fmt.Sprintf(`pr_%d: pullRequest(number: %d) {
      reviews(first: 100) {
        nodes { author { login } submittedAt state body }
      }
    }`, n, n))
	}

	query := fmt.Sprintf(`query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    %s
  }
}`, strings.Join(fragments, "\n    "))

	data, err := GHGraphQL(query, map[string]interface{}{
		"owner": owner,
		"name":  name,
	})
	if err != nil {
		return nil, fmt.Errorf("batch fetching reviews for %s: %w", repo, err)
	}

	// Parse the dynamic response structure.
	var top struct {
		Data struct {
			Repository json.RawMessage `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("parsing batch review response: %w", err)
	}

	// Parse repository as map of aliases -> PR data.
	var prMap map[string]json.RawMessage
	if err := json.Unmarshal(top.Data.Repository, &prMap); err != nil {
		return nil, fmt.Errorf("parsing repository aliases: %w", err)
	}

	result := make(map[int][]rawPRReview)
	for _, n := range prNumbers {
		alias := fmt.Sprintf("pr_%d", n)
		raw, ok := prMap[alias]
		if !ok {
			continue
		}

		var prData struct {
			Reviews struct {
				Nodes []struct {
					Author      struct{ Login string } `json:"author"`
					SubmittedAt time.Time              `json:"submittedAt"`
					State       string                 `json:"state"`
					Body        string                 `json:"body"`
				} `json:"nodes"`
			} `json:"reviews"`
		}
		if err := json.Unmarshal(raw, &prData); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: parsing reviews for %s#%d: %v\n", repo, n, err)
			continue
		}

		var reviews []rawPRReview
		for _, node := range prData.Reviews.Nodes {
			reviews = append(reviews, rawPRReview{
				User:        GitHubUser{Login: node.Author.Login},
				SubmittedAt: node.SubmittedAt,
				State:       node.State,
				Body:        node.Body,
			})
		}
		result[n] = reviews
	}

	return result, nil
}

// FetchPRReviewsAsComments fetches PR reviews for a set of PRs and returns
// them as GitHubComment entries so they participate in ball-in-court filtering.
// Only reviews submitted since the given time are included.
// Uses a single batched GraphQL call, falling back to per-PR REST calls on failure.
//
// Excluded review states:
//   - PENDING: draft reviews not yet submitted (only visible to the author)
//   - DISMISSED: reviews invalidated by the PR author or admin
func FetchPRReviewsAsComments(repo string, prNumbers []int, since time.Time) []GitHubComment {
	reviewsByPR, err := fetchPRReviewsBatch(repo, prNumbers)
	if err != nil {
		// Fall back to sequential REST calls.
		fmt.Fprintf(os.Stderr, "Warning: %v; falling back to per-PR fetching\n", err)
		reviewsByPR = make(map[int][]rawPRReview)
		for _, number := range prNumbers {
			reviews, err := fetchPRReviews(repo, number)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				continue
			}
			reviewsByPR[number] = reviews
		}
	}

	var comments []GitHubComment
	for _, number := range prNumbers {
		reviews := reviewsByPR[number]
		issueURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, number)
		for _, r := range reviews {
			if r.SubmittedAt.Before(since) {
				continue
			}
			if r.State == "PENDING" || r.State == "DISMISSED" {
				continue
			}
			comments = append(comments, GitHubComment{
				User:      r.User,
				UpdatedAt: r.SubmittedAt,
				CreatedAt: r.SubmittedAt,
				IssueURL:  issueURL,
				Body:      r.Body,
			})
		}
	}
	return comments
}

// FetchItemCommenters fetches commenters for an issue or PR.
func FetchItemCommenters(repo string, number int) ([]string, error) {
	endpoint := fmt.Sprintf("/repos/%s/issues/%d/comments", repo, number)
	data, err := GHAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var comments []struct {
		User GitHubUser `json:"user"`
	}
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, err
	}

	// Deduplicate commenters
	commenterSet := make(map[string]bool)
	for _, c := range comments {
		if c.User.Login != "" {
			commenterSet[c.User.Login] = true
		}
	}

	var commenters []string
	for login := range commenterSet {
		commenters = append(commenters, login)
	}
	return commenters, nil
}
