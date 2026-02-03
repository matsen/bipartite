// Package flow provides functionality for GitHub activity management.
// It implements the flowc commands (checkin, board, spawn, digest, tree)
// as part of the bip CLI.
package flow

import "time"

// Sources represents the sources.yml configuration file.
type Sources struct {
	Slack   SlackConfig       `yaml:"slack"`   // Slack channel configuration
	Boards  map[string]string `yaml:"boards"`  // "channel" -> "owner/N" board key
	Context map[string]string `yaml:"context"` // repo -> context file path
	Code    []RepoEntry       `yaml:"code"`
	Writing []RepoEntry       `yaml:"writing"`
}

// SlackConfig represents the slack section of sources.yml.
type SlackConfig struct {
	Channels map[string]SlackChannelConfig `yaml:"channels"`
}

// SlackChannelConfig is a configured Slack channel from sources.yml.
type SlackChannelConfig struct {
	ID      string `json:"id" yaml:"id"`
	Purpose string `json:"purpose" yaml:"purpose"`
}

// RepoEntry represents a repository entry in sources.yml.
// It can be either a string (repo name only) or an object with channel info.
type RepoEntry struct {
	Repo    string `yaml:"repo"`
	Channel string `yaml:"channel,omitempty"`
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
	Body         string   // Full body text from GitHub (issue/PR description)
	Summary      string   // AI-generated one-sentence summary of the body
}

// TakehomeSummary maps GitHub refs to their take-home summaries.
type TakehomeSummary map[string]string

// ItemAction represents a single action (comment, close, merge) on a GitHub issue or PR.
// Used by ball-in-court logic to determine who acted last on an item.
//
// ItemAction unifies comments and events into a single type, allowing
// BallInMyCourt to consider both "X commented" and "Y closed/merged" when
// determining if an item requires attention.
type ItemAction struct {
	ItemNumber int       // Issue or PR number
	Actor      string    // GitHub username who performed the action
	Timestamp  time.Time // When the action occurred
}

// Config represents config.yml settings.
type Config struct {
	Paths ConfigPaths `yaml:"paths"`
}

// ConfigPaths contains path settings from config.yml.
type ConfigPaths struct {
	Code    string `yaml:"code"`
	Writing string `yaml:"writing"`
}
