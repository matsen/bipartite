package flow

import (
	"testing"
	"time"
)

func TestBallInMyCourt(t *testing.T) {
	me := "me"
	them := "them"
	now := time.Now()

	// Helper to create an item
	makeItem := func(author string, number int) GitHubItem {
		return GitHubItem{
			Number:    number,
			User:      GitHubUser{Login: author},
			UpdatedAt: now,
		}
	}

	// Helper to create a comment
	makeComment := func(author string, itemNumber int) GitHubComment {
		return GitHubComment{
			User:      GitHubUser{Login: author},
			IssueURL:  "https://api.github.com/repos/org/repo/issues/" + itoa(itemNumber),
			UpdatedAt: now,
		}
	}

	tests := []struct {
		name       string
		itemAuthor string
		comments   []GitHubComment
		expected   bool
		reason     string
	}{
		{
			name:       "their item, no comments",
			itemAuthor: them,
			comments:   nil,
			expected:   true,
			reason:     "needs review",
		},
		{
			name:       "their item, I commented last",
			itemAuthor: them,
			comments:   []GitHubComment{makeComment(me, 1)},
			expected:   false,
			reason:     "waiting for their reply",
		},
		{
			name:       "their item, they commented last",
			itemAuthor: them,
			comments:   []GitHubComment{makeComment(me, 1), makeComment(them, 1)},
			expected:   true,
			reason:     "they pinged again",
		},
		{
			name:       "my item, no comments",
			itemAuthor: me,
			comments:   nil,
			expected:   false,
			reason:     "waiting for feedback",
		},
		{
			name:       "my item, they commented last",
			itemAuthor: me,
			comments:   []GitHubComment{makeComment(them, 1)},
			expected:   true,
			reason:     "they replied",
		},
		{
			name:       "my item, I commented last",
			itemAuthor: me,
			comments:   []GitHubComment{makeComment(them, 1), makeComment(me, 1)},
			expected:   false,
			reason:     "waiting for their reply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := makeItem(tt.itemAuthor, 1)
			got := BallInMyCourt(item, tt.comments, me)
			if got != tt.expected {
				t.Errorf("BallInMyCourt() = %v, want %v (%s)", got, tt.expected, tt.reason)
			}
		})
	}
}

func TestBallInMyCourtScenarios(t *testing.T) {
	me := "matsen"
	now := time.Now()

	// Scenario 1: Someone commented Saturday on an old issue you created months ago.
	// -> Show: they replied to your item
	t.Run("someone replied to my old issue", func(t *testing.T) {
		item := GitHubItem{
			Number:    1,
			User:      GitHubUser{Login: me},
			CreatedAt: now.Add(-90 * 24 * time.Hour), // Created months ago
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			{
				User:      GitHubUser{Login: "colleague"},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/1",
				UpdatedAt: now.Add(-2 * 24 * time.Hour), // Commented Saturday
			},
		}
		if !BallInMyCourt(item, comments, me) {
			t.Error("Expected true: they replied to my item")
		}
	})

	// Scenario 2: You added a comment Saturday to your own issue (adding context).
	// -> Hide: you commented last
	t.Run("I commented on my own issue", func(t *testing.T) {
		item := GitHubItem{
			Number:    2,
			User:      GitHubUser{Login: me},
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			{
				User:      GitHubUser{Login: me},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/2",
				UpdatedAt: now.Add(-2 * 24 * time.Hour),
			},
		}
		if BallInMyCourt(item, comments, me) {
			t.Error("Expected false: I commented last")
		}
	})

	// Scenario 3: Someone opened a new PR Friday, no comments yet.
	// -> Show: their item needs your review
	t.Run("new PR from them, no comments", func(t *testing.T) {
		item := GitHubItem{
			Number:    3,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		if !BallInMyCourt(item, nil, me) {
			t.Error("Expected true: their PR needs review")
		}
	})

	// Scenario 4: You opened a PR Friday, no comments yet.
	// -> Hide: waiting for feedback
	t.Run("my new PR, no comments", func(t *testing.T) {
		item := GitHubItem{
			Number:    4,
			User:      GitHubUser{Login: me},
			IsPR:      true,
			UpdatedAt: now,
		}
		if BallInMyCourt(item, nil, me) {
			t.Error("Expected false: waiting for feedback on my PR")
		}
	})

	// Scenario 5: You reviewed someone's PR and left comments.
	// -> Hide: you commented last
	t.Run("I reviewed their PR", func(t *testing.T) {
		item := GitHubItem{
			Number:    5,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			{
				User:      GitHubUser{Login: me},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/5",
				UpdatedAt: now,
			},
		}
		if BallInMyCourt(item, comments, me) {
			t.Error("Expected false: I reviewed, waiting for them to address")
		}
	})

	// Scenario 6: Someone replied to your review on their PR.
	// -> Show: they responded
	t.Run("they replied to my review", func(t *testing.T) {
		item := GitHubItem{
			Number:    6,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			{
				User:      GitHubUser{Login: me},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/6",
				UpdatedAt: now.Add(-1 * time.Hour),
			},
			{
				User:      GitHubUser{Login: "colleague"},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/6",
				UpdatedAt: now,
			},
		}
		if !BallInMyCourt(item, comments, me) {
			t.Error("Expected true: they replied to my review")
		}
	})
}

func TestBallInMyCourtWithPRReviews(t *testing.T) {
	me := "matsen"
	now := time.Now()

	// Helper: create a GitHubComment that mimics a PR review
	// (same shape as what FetchPRReviewsAsComments returns)
	makeReviewComment := func(author string, itemNumber int, when time.Time) GitHubComment {
		return GitHubComment{
			User:      GitHubUser{Login: author},
			IssueURL:  "https://api.github.com/repos/org/repo/issues/" + itoa(itemNumber),
			UpdatedAt: when,
			CreatedAt: when,
		}
	}

	// Scenario: Their PR, I submitted a review (approve/request-changes).
	// No inline comments, just the review itself.
	// -> Hide: I acted last via the review.
	t.Run("their PR, I reviewed via approve", func(t *testing.T) {
		item := GitHubItem{
			Number:    107,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			makeReviewComment(me, 107, now),
		}
		if BallInMyCourt(item, comments, me) {
			t.Error("Expected false: I reviewed, waiting for them to address")
		}
	})

	// Scenario: Their PR, I reviewed, then they pushed new commits
	// and re-requested review (their comment is last).
	// -> Show: they acted after my review.
	t.Run("their PR, they responded after my review", func(t *testing.T) {
		item := GitHubItem{
			Number:    107,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			makeReviewComment(me, 107, now.Add(-1*time.Hour)),
			makeReviewComment("colleague", 107, now),
		}
		if !BallInMyCourt(item, comments, me) {
			t.Error("Expected true: they responded after my review")
		}
	})

	// Scenario: My PR, someone approved it.
	// -> Show: they acted (approval is an action).
	t.Run("my PR, someone approved", func(t *testing.T) {
		item := GitHubItem{
			Number:    50,
			User:      GitHubUser{Login: me},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			makeReviewComment("reviewer", 50, now),
		}
		if !BallInMyCourt(item, comments, me) {
			t.Error("Expected true: reviewer approved my PR")
		}
	})

	// Scenario: Their PR, I reviewed, then also left inline comments.
	// Review and inline comments both by me — still hide.
	t.Run("their PR, my review + inline comments", func(t *testing.T) {
		item := GitHubItem{
			Number:    200,
			User:      GitHubUser{Login: "colleague"},
			IsPR:      true,
			UpdatedAt: now,
		}
		comments := []GitHubComment{
			makeReviewComment(me, 200, now.Add(-5*time.Minute)),
			{
				User:      GitHubUser{Login: me},
				PRURL:     "https://api.github.com/repos/org/repo/pulls/200",
				UpdatedAt: now,
			},
		}
		if BallInMyCourt(item, comments, me) {
			t.Error("Expected false: all activity is mine (review + inline)")
		}
	})
}

func TestFilterByBallInCourt(t *testing.T) {
	me := "me"
	now := time.Now()

	items := []GitHubItem{
		{Number: 1, User: GitHubUser{Login: "them"}, UpdatedAt: now}, // Show: their item
		{Number: 2, User: GitHubUser{Login: me}, UpdatedAt: now},     // Hide: my item, no comments
		{Number: 3, User: GitHubUser{Login: "them"}, UpdatedAt: now}, // Hide: I commented last
	}

	comments := []GitHubComment{
		{User: GitHubUser{Login: me}, IssueURL: "https://api.github.com/repos/org/repo/issues/3", UpdatedAt: now},
	}

	filtered := FilterByBallInCourt(items, comments, me)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 item, got %d", len(filtered))
		return
	}

	if filtered[0].Number != 1 {
		t.Errorf("Expected item #1, got #%d", filtered[0].Number)
	}
}

// Simple int to string for test URLs
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
