// Package github provides a client for fetching repository metadata from the GitHub API.
package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Client is a GitHub API client for fetching repository metadata.
type Client struct {
	httpClient *http.Client
	token      string
}

// RepoMetadata contains metadata fetched from the GitHub API.
type RepoMetadata struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	HTMLURL     string   `json:"html_url"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// Errors.
var (
	ErrInvalidURL   = errors.New("invalid GitHub URL format")
	ErrRepoNotFound = errors.New("repository not found (404)")
	ErrRateLimited  = errors.New("GitHub API rate limit exceeded")
	ErrUnauthorized = errors.New("GitHub API authentication failed")
	ErrAPIError     = errors.New("GitHub API error")
	ErrNetworkError = errors.New("network error connecting to GitHub")
)

// NewClient creates a new GitHub API client.
// It reads GITHUB_TOKEN from the environment for authenticated requests.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		token: os.Getenv("GITHUB_TOKEN"),
	}
}

// urlPatterns for parsing GitHub URLs.
var (
	// Matches: https://github.com/owner/repo, https://github.com/owner/repo.git, github.com/owner/repo
	fullURLPattern = regexp.MustCompile(`^(?:https?://)?github\.com/([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+?)(?:\.git)?$`)
	// Matches: owner/repo
	shorthandPattern = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)$`)
)

// ParseGitHubURL parses a GitHub URL or org/repo shorthand and returns (owner, repo).
// Supported formats:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - github.com/owner/repo
//   - owner/repo
func ParseGitHubURL(input string) (owner, repo string, err error) {
	input = strings.TrimSpace(input)

	// Try full URL pattern first
	if matches := fullURLPattern.FindStringSubmatch(input); matches != nil {
		return matches[1], matches[2], nil
	}

	// Try shorthand pattern
	if matches := shorthandPattern.FindStringSubmatch(input); matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", ErrInvalidURL
}

// NormalizeGitHubURL normalizes a GitHub URL input to the canonical https form.
func NormalizeGitHubURL(input string) (string, error) {
	owner, repo, err := ParseGitHubURL(input)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/%s/%s", owner, repo), nil
}

// DeriveRepoID derives a default repo ID from a GitHub URL.
// Returns the lowercased repo name (e.g., "matsen/Bipartite" -> "bipartite").
func DeriveRepoID(input string) (string, error) {
	_, repo, err := ParseGitHubURL(input)
	if err != nil {
		return "", err
	}
	return strings.ToLower(repo), nil
}

// FetchRepoMetadata fetches repository metadata from the GitHub API.
func (c *Client) FetchRepoMetadata(urlOrShorthand string) (*RepoMetadata, error) {
	owner, repo, err := ParseGitHubURL(urlOrShorthand)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}

	// Set required headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "bipartite-cli")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Success
	case http.StatusNotFound:
		return nil, ErrRepoNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, ErrRateLimited
		}
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	default:
		return nil, fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
	}

	var meta RepoMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("%w: decoding response: %v", ErrAPIError, err)
	}

	return &meta, nil
}
