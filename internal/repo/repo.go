// Package repo defines the core domain types for repository nodes.
package repo

import (
	"errors"
	"regexp"
)

// Repo represents a GitHub repository belonging to a project.
type Repo struct {
	ID          string   `json:"id"`                    // Required: unique identifier
	Project     string   `json:"project"`               // Required: project ID this repo belongs to
	Type        string   `json:"type"`                  // Required: "github" or "manual"
	Name        string   `json:"name"`                  // Required: display name
	GitHubURL   string   `json:"github_url,omitempty"`  // Required if type=github
	Description string   `json:"description,omitempty"` // From GitHub or user-provided
	Topics      []string `json:"topics,omitempty"`      // From GitHub or user-provided
	Language    string   `json:"language,omitempty"`    // From GitHub
	CreatedAt   string   `json:"created_at,omitempty"`  // RFC3339, auto-set
	UpdatedAt   string   `json:"updated_at,omitempty"`  // RFC3339, auto-set
}

// RepoType constants.
const (
	TypeGitHub = "github"
	TypeManual = "manual"
)

// IDPattern is the regex pattern for valid repo IDs.
// Must start with alphanumeric, followed by alphanumeric, hyphens, or underscores.
var IDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Validation errors.
var (
	ErrEmptyID            = errors.New("id is required")
	ErrInvalidID          = errors.New("id must match pattern: lowercase alphanumeric, hyphens, underscores; must start with alphanumeric")
	ErrEmptyProject       = errors.New("project is required")
	ErrEmptyName          = errors.New("name is required")
	ErrInvalidType        = errors.New("type must be 'github' or 'manual'")
	ErrMissingGitHubURL   = errors.New("github_url is required for github type repos")
	ErrDuplicateID        = errors.New("repo with this id already exists")
	ErrDuplicateGitHubURL = errors.New("repo with this github_url already exists")
	ErrRepoNotFound       = errors.New("repo not found")
	ErrProjectNotFound    = errors.New("project not found")
	ErrNoGitHubURL        = errors.New("repo is manual type (no github_url to refresh)")
)

// ValidateForCreate validates a repo for creation.
// Returns an error if any required field is missing or invalid.
func (r *Repo) ValidateForCreate() error {
	if r.ID == "" {
		return ErrEmptyID
	}
	if !IDPattern.MatchString(r.ID) {
		return ErrInvalidID
	}
	if r.Project == "" {
		return ErrEmptyProject
	}
	if r.Name == "" {
		return ErrEmptyName
	}
	if r.Type != TypeGitHub && r.Type != TypeManual {
		return ErrInvalidType
	}
	if r.Type == TypeGitHub && r.GitHubURL == "" {
		return ErrMissingGitHubURL
	}
	return nil
}

// ValidateID validates just the ID field (useful for lookup operations).
func ValidateID(id string) error {
	if id == "" {
		return ErrEmptyID
	}
	if !IDPattern.MatchString(id) {
		return ErrInvalidID
	}
	return nil
}
