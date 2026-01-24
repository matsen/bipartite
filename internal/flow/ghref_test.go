package flow

import "testing"

func TestParseGitHubRef(t *testing.T) {
	tests := []struct {
		input    string
		wantRepo string
		wantNum  int
		wantType string
		wantNil  bool
	}{
		// Hash format
		{"matsengrp/dasm2-experiments#166", "matsengrp/dasm2-experiments", 166, "", false},
		{"org/repo-v2#42", "org/repo-v2", 42, "", false},
		{"org/my#repo#123", "org/my#repo", 123, "", false}, // Uses LAST #

		// URL format
		{"https://github.com/org/repo/issues/42", "org/repo", 42, "issue", false},
		{"https://github.com/org/repo/pull/123", "org/repo", 123, "pr", false},
		{"github.com/org/repo/issues/10", "org/repo", 10, "issue", false},          // No https
		{"https://www.github.com/org/repo/pull/5", "org/repo", 5, "pr", false},     // www
		{"https://github.com/org/repo/issues/99/", "org/repo", 99, "issue", false}, // Trailing slash

		// Invalid hash formats
		{"org/repo123", "", 0, "", true},  // No #
		{"repo#123", "", 0, "", true},     // No org/
		{"#123", "", 0, "", true},         // No org/
		{"org/repo#abc", "", 0, "", true}, // Non-numeric
		{"org/repo#", "", 0, "", true},    // Empty number
		{"org/repo#0", "", 0, "", true},   // Zero
		{"org/repo#-5", "", 0, "", true},  // Negative

		// Invalid URLs
		{"https://github.com/org/repo", "", 0, "", true},             // No issue/pr path
		{"https://github.com/org/repo/commits/abc", "", 0, "", true}, // Wrong path
		{"https://gitlab.com/org/repo/issues/1", "", 0, "", true},    // Wrong domain
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseGitHubRef(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseGitHubRef(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Errorf("ParseGitHubRef(%q) = nil, want non-nil", tt.input)
				return
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("ParseGitHubRef(%q).Repo = %q, want %q", tt.input, got.Repo, tt.wantRepo)
			}
			if got.Number != tt.wantNum {
				t.Errorf("ParseGitHubRef(%q).Number = %d, want %d", tt.input, got.Number, tt.wantNum)
			}
			if got.ItemType != tt.wantType {
				t.Errorf("ParseGitHubRef(%q).ItemType = %q, want %q", tt.input, got.ItemType, tt.wantType)
			}
		})
	}
}

func TestGitHubURL(t *testing.T) {
	tests := []struct {
		repo     string
		number   int
		itemType string
		expected string
	}{
		{"org/repo", 123, "issue", "https://github.com/org/repo/issues/123"},
		{"org/repo", 456, "pr", "https://github.com/org/repo/pull/456"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := GitHubURL(tt.repo, tt.number, tt.itemType)
			if got != tt.expected {
				t.Errorf("GitHubURL(%q, %d, %q) = %q, want %q", tt.repo, tt.number, tt.itemType, got, tt.expected)
			}
		})
	}
}
