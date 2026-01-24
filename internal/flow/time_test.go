package flow

import (
	"testing"
	"time"
)

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2 hours ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1 day ago"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"1 month ago", now.Add(-45 * 24 * time.Hour), "1 month ago"},
		{"3 months ago", now.Add(-90 * 24 * time.Hour), "3 months ago"},
		{"1 year ago", now.Add(-400 * 24 * time.Hour), "1 year ago"},
		{"2 years ago", now.Add(-800 * 24 * time.Hour), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRelativeTime(tt.time)
			if got != tt.expected {
				t.Errorf("FormatRelativeTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		time     time.Time
		expected string
	}{
		{now.Add(-30 * time.Minute), "30m ago"},
		{now.Add(-5 * time.Hour), "5h ago"},
		{now.Add(-3 * 24 * time.Hour), "3d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatTimeAgo(tt.time)
			if got != tt.expected {
				t.Errorf("FormatTimeAgo() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseGitHubTimestamp(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2026-01-20T10:30:00Z", false},
		{"2026-01-20T10:30:00+00:00", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseGitHubTimestamp(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ParseGitHubTimestamp(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParseGitHubTimestamp(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}
