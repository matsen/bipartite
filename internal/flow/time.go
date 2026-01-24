package flow

import (
	"fmt"
	"time"
)

// FormatRelativeTime formats a time as relative to now.
// Examples: "just now", "5 minutes ago", "2 hours ago", "3 days ago"
func FormatRelativeTime(t time.Time) string {
	now := time.Now().UTC()
	if t.Location() != time.UTC {
		t = t.UTC()
	}

	delta := now.Sub(t)

	// Handle future timestamps
	if delta < 0 {
		return "in the future"
	}

	// Less than 1 minute
	if delta < time.Minute {
		return "just now"
	}

	// Minutes
	minutes := int(delta.Minutes())
	if minutes < 60 {
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	// Hours
	hours := int(delta.Hours())
	if hours < 24 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	// Days
	days := int(delta.Hours() / 24)
	if days < 30 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	// Months
	months := days / 30
	if months < 12 {
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	// Years
	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// FormatTimeAgo formats a time as a short relative string (e.g., "2d ago", "5h ago").
func FormatTimeAgo(t time.Time) string {
	now := time.Now().UTC()
	if t.Location() != time.UTC {
		t = t.UTC()
	}

	delta := now.Sub(t)
	if delta < 0 {
		return "future"
	}

	if delta.Hours() >= 24 {
		days := int(delta.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}

	if delta.Hours() >= 1 {
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	}

	return fmt.Sprintf("%dm ago", int(delta.Minutes()))
}

// ParseGitHubTimestamp parses a GitHub API timestamp (ISO 8601 with Z suffix).
func ParseGitHubTimestamp(s string) (time.Time, error) {
	// GitHub uses RFC3339 format
	return time.Parse(time.RFC3339, s)
}
