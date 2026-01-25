// Package flow provides functionality for GitHub activity management.
// It implements the flowc commands (checkin, board, spawn, digest, tree)
// as part of the bip CLI.
package flow

import "time"

// Sources represents the sources.json configuration file.
type Sources struct {
	Boards  map[string]string `json:"boards"`  // "owner/N" -> bead_id
	Context map[string]string `json:"context"` // repo -> context file path
	Code    []RepoEntry       `json:"code"`
	Writing []RepoEntry       `json:"writing"`
}

// RepoEntry represents a repository entry in sources.json.
// It can be either a string (repo name only) or an object with channel info.
type RepoEntry struct {
	Repo    string `json:"repo"`
	Channel string `json:"channel,omitempty"`
}

// Bead represents an issue in .beads/issues.jsonl.
type Bead struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	IssueType   string    `json:"issue_type"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GitHubRef represents a parsed GitHub reference.
type GitHubRef struct {
	Repo     string // org/repo
	Number   int    // issue or PR number
	ItemType string // "issue", "pr", or "" (unknown)
}

// GitHubItem represents an issue or PR from the GitHub API.
type GitHubItem struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      GitHubUser
	IsPR      bool
	Labels    []GitHubLabel
}

// GitHubUser represents a GitHub user.
type GitHubUser struct {
	Login string `json:"login"`
}

// GitHubLabel represents a GitHub label.
type GitHubLabel struct {
	Name string `json:"name"`
}

// GitHubComment represents a comment on an issue or PR.
type GitHubComment struct {
	ID        int64      `json:"id"`
	Body      string     `json:"body"`
	User      GitHubUser `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	IssueURL  string     `json:"issue_url,omitempty"`
	PRURL     string     `json:"pull_request_url,omitempty"`
	HTMLURL   string     `json:"html_url"`
}

// BoardItem represents an item on a GitHub project board.
type BoardItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Content BoardContent
}

// BoardContent represents the content of a board item.
type BoardContent struct {
	Type       string `json:"type"` // "Issue" or "PullRequest"
	Repository string `json:"repository"`
	Number     int    `json:"number"`
}

// ItemDetails contains detailed information about an issue or PR.
type ItemDetails struct {
	Ref      string           // GitHub reference (e.g., "org/repo#123")
	Title    string           // Issue/PR title
	Author   string           // Author login
	Body     string           // Issue/PR body
	IsPR     bool             // Whether this is a PR
	State    string           // open/closed
	Comments []CommentSummary // Recent comments
	// PR-specific fields
	Files     []PRFile   // Changed files (PRs only)
	Reviews   []PRReview // Reviews (PRs only)
	Additions int        // Lines added (PRs only)
	Deletions int        // Lines deleted (PRs only)
	Commits   int        // Number of commits (PRs only)
	Labels    []string   // Label names
	CreatedAt time.Time  // Creation time
}

// CommentSummary contains summarized comment information.
type CommentSummary struct {
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// PRFile represents a file changed in a PR.
type PRFile struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// PRReview represents a PR review.
type PRReview struct {
	Author string `json:"author"`
	State  string `json:"state"`
	Body   string `json:"body"`
}

// DigestItem represents an item for digest generation.
type DigestItem struct {
	Ref          string   // GitHub reference (e.g., "org/repo#123")
	Number       int      // Issue/PR number
	Title        string   // Item title
	Author       string   // Author login
	IsPR         bool     // Whether this is a PR
	State        string   // open/closed/merged
	Merged       bool     // Whether PR was merged
	HTMLURL      string   // URL to the item
	CreatedAt    string   // ISO timestamp
	UpdatedAt    string   // ISO timestamp
	Contributors []string // List of contributor logins
	Body         string   // Full body text (for --verbose mode)
	Summary      string   // LLM-generated summary (for --verbose mode)
}

// TakehomeSummary maps GitHub refs to their take-home summaries.
type TakehomeSummary map[string]string

// Config represents config.json settings.
type Config struct {
	Paths ConfigPaths `json:"paths"`
}

// ConfigPaths contains path settings from config.json.
type ConfigPaths struct {
	Code    string `json:"code"`
	Writing string `json:"writing"`
}
