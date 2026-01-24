package flow

import (
	"regexp"
	"strconv"
	"strings"
)

// Patterns for parsing GitHub references.
var (
	// Matches: (https://)?(www.)?github.com/org/repo/(issues|pull)/number
	urlPattern = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/([^/]+/[^/]+)/(issues|pull)/(\d+)/?$`)
)

// ParseGitHubRef parses a GitHub reference (URL or org/repo#number).
// Returns nil if the input is invalid.
//
// Supported formats:
//   - org/repo#123 (type unknown, needs detection)
//   - https://github.com/org/repo/issues/123
//   - https://github.com/org/repo/pull/123
//   - github.com/org/repo/issues/123 (without scheme)
//   - https://www.github.com/org/repo/pull/5 (with www)
func ParseGitHubRef(arg string) *GitHubRef {
	arg = strings.TrimSpace(arg)

	// Try URL format first
	if matches := urlPattern.FindStringSubmatch(arg); matches != nil {
		number, _ := strconv.Atoi(matches[3])
		itemType := "issue"
		if matches[2] == "pull" {
			itemType = "pr"
		}
		return &GitHubRef{
			Repo:     matches[1],
			Number:   number,
			ItemType: itemType,
		}
	}

	// Try org/repo#number format
	return parseHashFormat(arg)
}

// parseHashFormat parses the org/repo#number format.
func parseHashFormat(arg string) *GitHubRef {
	// Must contain #
	if !strings.Contains(arg, "#") {
		return nil
	}

	// Split on last # to handle repos like "org/my#repo#123"
	lastHash := strings.LastIndex(arg, "#")
	orgRepo := arg[:lastHash]
	numberStr := arg[lastHash+1:]

	// Validate org/repo format
	if orgRepo == "" || !strings.Contains(orgRepo, "/") {
		return nil
	}

	// Validate number
	if numberStr == "" {
		return nil
	}

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		return nil
	}

	return &GitHubRef{
		Repo:     orgRepo,
		Number:   number,
		ItemType: "", // Unknown, needs detection via API
	}
}

// GitHubURL constructs a GitHub URL for an issue or PR.
func GitHubURL(orgRepo string, number int, itemType string) string {
	resource := "issues"
	if itemType == "pr" {
		resource = "pull"
	}
	return "https://github.com/" + orgRepo + "/" + resource + "/" + strconv.Itoa(number)
}
