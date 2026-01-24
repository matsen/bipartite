package flow

import (
	"strings"
	"testing"
)

func TestParseSummaryResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		wantKeys []string
	}{
		{
			name:     "valid JSON",
			response: `{"matsengrp/repo#123": "summary here"}`,
			wantErr:  false,
			wantKeys: []string{"matsengrp/repo#123"},
		},
		{
			name: "JSON in markdown code block",
			response: "```json\n" +
				`{"matsengrp/repo#123": "summary here"}` + "\n```",
			wantErr:  false,
			wantKeys: []string{"matsengrp/repo#123"},
		},
		{
			name: "JSON in plain code block",
			response: "```\n" +
				`{"matsengrp/repo#123": "summary here"}` + "\n```",
			wantErr:  false,
			wantKeys: []string{"matsengrp/repo#123"},
		},
		{
			name: "multiple items",
			response: `{
				"matsengrp/repo#1": "first summary",
				"matsengrp/repo#2": "second summary"
			}`,
			wantErr:  false,
			wantKeys: []string{"matsengrp/repo#1", "matsengrp/repo#2"},
		},
		{
			name:     "invalid JSON",
			response: "This is not JSON at all",
			wantErr:  true,
		},
		{
			name: "JSON with preamble",
			response: `Here are the summaries:
{"matsengrp/repo#123": "summary"}`,
			wantErr: true,
		},
		{
			name:     "empty response",
			response: "",
			wantErr:  true,
		},
		{
			name: "whitespace around JSON",
			response: `

  {"matsengrp/repo#123": "summary"}

`,
			wantErr:  false,
			wantKeys: []string{"matsengrp/repo#123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSummaryResponse(tt.response)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSummaryResponse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseSummaryResponse() unexpected error: %v", err)
				return
			}
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("parseSummaryResponse() missing key %q", key)
				}
			}
		})
	}
}

func TestExtractFromCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "plain code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no closing fence",
			input:    "```json\n{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFromCodeBlock(tt.input)
			if got != tt.expected {
				t.Errorf("extractFromCodeBlock() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildSummaryPrompt(t *testing.T) {
	t.Run("builds prompt for issue", func(t *testing.T) {
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#123",
				Title:  "Test issue",
				Author: "testuser",
				Body:   "Issue body text",
				IsPR:   false,
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, "REF: matsengrp/repo#123") {
			t.Error("missing REF")
		}
		if !strings.Contains(prompt, "TYPE: Issue") {
			t.Error("missing TYPE: Issue")
		}
		if !strings.Contains(prompt, "TITLE: Test issue") {
			t.Error("missing TITLE")
		}
		if !strings.Contains(prompt, "AUTHOR: testuser") {
			t.Error("missing AUTHOR")
		}
		if !strings.Contains(prompt, "BODY: Issue body text") {
			t.Error("missing BODY")
		}
	})

	t.Run("builds prompt for PR", func(t *testing.T) {
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#456",
				Title:  "Test PR",
				Author: "prauthor",
				Body:   "PR description",
				IsPR:   true,
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, "TYPE: PR") {
			t.Error("missing TYPE: PR")
		}
	})

	t.Run("includes comments", func(t *testing.T) {
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#123",
				Title:  "Test",
				Author: "author",
				IsPR:   false,
				Comments: []CommentSummary{
					{Author: "commenter1", Body: "First comment"},
					{Author: "commenter2", Body: "Second comment"},
				},
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, "@commenter1: First comment") {
			t.Error("missing first comment")
		}
		if !strings.Contains(prompt, "@commenter2: Second comment") {
			t.Error("missing second comment")
		}
	})

	t.Run("truncates long comments to 200 chars", func(t *testing.T) {
		longComment := strings.Repeat("x", 300)
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#123",
				Title:  "Test",
				Author: "author",
				IsPR:   false,
				Comments: []CommentSummary{
					{Author: "commenter", Body: longComment},
				},
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, strings.Repeat("x", 200)) {
			t.Error("should contain truncated comment")
		}
		if strings.Contains(prompt, strings.Repeat("x", 201)) {
			t.Error("should not contain more than 200 x's")
		}
	})

	t.Run("truncates long body to 300 chars", func(t *testing.T) {
		longBody := strings.Repeat("y", 500)
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#123",
				Title:  "Test",
				Author: "author",
				Body:   longBody,
				IsPR:   false,
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, strings.Repeat("y", 300)) {
			t.Error("should contain truncated body")
		}
		if strings.Contains(prompt, strings.Repeat("y", 301)) {
			t.Error("should not contain more than 300 y's")
		}
	})

	t.Run("limits to last 5 comments", func(t *testing.T) {
		var comments []CommentSummary
		for i := 0; i < 10; i++ {
			comments = append(comments, CommentSummary{
				Author: "user" + itoa(i),
				Body:   "comment " + itoa(i),
			})
		}
		items := []ItemDetails{
			{
				Ref:      "matsengrp/repo#123",
				Title:    "Test",
				Author:   "author",
				IsPR:     false,
				Comments: comments,
			},
		}
		prompt := buildSummaryPrompt(items)

		// Should have comments 5-9, not 0-4
		if !strings.Contains(prompt, "@user5: comment 5") {
			t.Error("should contain user5's comment")
		}
		if !strings.Contains(prompt, "@user9: comment 9") {
			t.Error("should contain user9's comment")
		}
		if strings.Contains(prompt, "@user0: comment 0") {
			t.Error("should NOT contain user0's comment")
		}
		if strings.Contains(prompt, "@user4: comment 4") {
			t.Error("should NOT contain user4's comment")
		}
	})

	t.Run("includes output format instructions", func(t *testing.T) {
		items := []ItemDetails{
			{
				Ref:    "matsengrp/repo#123",
				Title:  "Test",
				Author: "author",
				IsPR:   false,
			},
		}
		prompt := buildSummaryPrompt(items)

		if !strings.Contains(prompt, "JSON object") {
			t.Error("should mention JSON object")
		}
		if !strings.Contains(prompt, "Return ONLY the JSON object") {
			t.Error("should have JSON-only instruction")
		}
	})
}

func TestBuildDigestPrompt(t *testing.T) {
	t.Run("includes channel and date range", func(t *testing.T) {
		items := []DigestItem{
			{
				Number:  123,
				Title:   "Test issue",
				Author:  "testuser",
				IsPR:    false,
				State:   "open",
				HTMLURL: "https://github.com/org/repo/issues/123",
			},
		}
		prompt := buildDigestPrompt(items, "dasm2", "Jan 12-18")

		if !strings.Contains(prompt, "Channel: dasm2") {
			t.Error("missing Channel")
		}
		if !strings.Contains(prompt, "Date range: Jan 12-18") {
			t.Error("missing Date range")
		}
		if !strings.Contains(prompt, "*This week in dasm2*") {
			t.Error("missing This week in header")
		}
	})

	t.Run("formats issue", func(t *testing.T) {
		items := []DigestItem{
			{
				Number:  123,
				Title:   "Test issue",
				Author:  "testuser",
				IsPR:    false,
				State:   "open",
				HTMLURL: "https://github.com/org/repo/issues/123",
			},
		}
		prompt := buildDigestPrompt(items, "dasm2", "Jan 12-18")

		if !strings.Contains(prompt, "[Issue]") {
			t.Error("missing [Issue]")
		}
		if !strings.Contains(prompt, "#123") {
			t.Error("missing #123")
		}
		if !strings.Contains(prompt, "Test issue") {
			t.Error("missing title")
		}
		if !strings.Contains(prompt, "@testuser") {
			t.Error("missing author")
		}
	})

	t.Run("formats PR", func(t *testing.T) {
		items := []DigestItem{
			{
				Number:  456,
				Title:   "Test PR",
				Author:  "prauthor",
				IsPR:    true,
				State:   "closed",
				Merged:  true,
				HTMLURL: "https://github.com/org/repo/pull/456",
			},
		}
		prompt := buildDigestPrompt(items, "dasm2", "Jan 12-18")

		if !strings.Contains(prompt, "[PR]") {
			t.Error("missing [PR]")
		}
		if !strings.Contains(prompt, "#456") {
			t.Error("missing #456")
		}
		if !strings.Contains(prompt, "merged") {
			t.Error("missing merged state")
		}
	})

	t.Run("includes Slack format instructions", func(t *testing.T) {
		items := []DigestItem{
			{
				Number:  123,
				Title:   "Test",
				Author:  "user",
				IsPR:    false,
				State:   "open",
				HTMLURL: "https://github.com/org/repo/issues/123",
			},
		}
		prompt := buildDigestPrompt(items, "test", "Jan 1-7")

		if !strings.Contains(prompt, "Slack") {
			t.Error("missing Slack")
		}
		if !strings.Contains(prompt, "mrkdwn") {
			t.Error("missing mrkdwn")
		}
		if !strings.Contains(prompt, "<URL|#number>") {
			t.Error("missing URL format instruction")
		}
	})
}

func TestPostprocessDigest(t *testing.T) {
	t.Run("adds PR prefix with repo", func(t *testing.T) {
		digest := "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true},
		}
		result := postprocessDigest(digest, items)

		if !strings.Contains(result, "• repo PR: Fix bug") {
			t.Errorf("got %q, want PR prefix", result)
		}
	})

	t.Run("adds Issue prefix with repo", func(t *testing.T) {
		digest := "• New feature request (<https://github.com/org/repo/issues/456|#456>)"
		items := []DigestItem{
			{Ref: "org/repo#456", Number: 456, IsPR: false},
		}
		result := postprocessDigest(digest, items)

		if !strings.Contains(result, "• repo Issue: New feature request") {
			t.Errorf("got %q, want Issue prefix", result)
		}
	})

	t.Run("adds contributors", func(t *testing.T) {
		digest := "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true, Contributors: []string{"alice", "bob", "charlie"}},
		}
		result := postprocessDigest(digest, items)

		if !strings.HasSuffix(result, "— @alice @bob @charlie") {
			t.Errorf("got %q, want contributors suffix", result)
		}
	})

	t.Run("preserves non-bullet lines", func(t *testing.T) {
		digest := `*This week in dasm2* (Jan 12-18)

*Merged*
• Fix bug (<https://github.com/org/repo/pull/123|#123>)

*Discussion*
• Feature request (<https://github.com/org/repo/issues/456|#456>)`
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true, Contributors: []string{"alice"}},
			{Ref: "org/repo#456", Number: 456, IsPR: false, Contributors: []string{"bob"}},
		}
		result := postprocessDigest(digest, items)

		if !strings.Contains(result, "*This week in dasm2* (Jan 12-18)") {
			t.Error("should preserve header")
		}
		if !strings.Contains(result, "*Merged*") {
			t.Error("should preserve Merged section")
		}
		if !strings.Contains(result, "*Discussion*") {
			t.Error("should preserve Discussion section")
		}
	})

	t.Run("handles missing item", func(t *testing.T) {
		digest := "• Unknown item (<https://github.com/org/repo/pull/999|#999>)"
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true},
		}
		result := postprocessDigest(digest, items)

		if result != digest {
			t.Errorf("got %q, want unchanged %q", result, digest)
		}
	})

	t.Run("handles line without link", func(t *testing.T) {
		digest := "• Some text without a link"
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true},
		}
		result := postprocessDigest(digest, items)

		if result != digest {
			t.Errorf("got %q, want unchanged %q", result, digest)
		}
	})

	t.Run("handles empty contributors", func(t *testing.T) {
		digest := "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
		items := []DigestItem{
			{Ref: "org/repo#123", Number: 123, IsPR: true, Contributors: []string{}},
		}
		result := postprocessDigest(digest, items)

		if strings.Contains(result, "—") {
			t.Errorf("got %q, should not have dash for empty contributors", result)
		}
	})

	t.Run("same number different repos", func(t *testing.T) {
		digest := `*Merged*
• First repo item (<https://github.com/org/repo-a/pull/31|#31>)
• Second repo item (<https://github.com/org/repo-b/issues/31|#31>)`
		items := []DigestItem{
			{Ref: "org/repo-a#31", Number: 31, IsPR: true, Contributors: []string{"alice"}},
			{Ref: "org/repo-b#31", Number: 31, IsPR: false, Contributors: []string{"bob"}},
		}
		result := postprocessDigest(digest, items)

		if !strings.Contains(result, "• repo-a PR: First repo item") {
			t.Errorf("wrong repo-a line: %s", result)
		}
		if !strings.Contains(result, "• repo-b Issue: Second repo item") {
			t.Errorf("wrong repo-b line: %s", result)
		}
	})
}

func TestGenerateDigestSummaryEmpty(t *testing.T) {
	result, err := GenerateDigestSummary(nil, "dasm2", "Jan 12-18")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "*This week in dasm2*") {
		t.Error("should contain header")
	}
	if !strings.Contains(result, "No activity") {
		t.Error("should contain No activity message")
	}
}
