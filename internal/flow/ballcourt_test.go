package flow

import (
	"strconv"
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

	// Helper to create an action
	makeAction := func(author string, itemNumber int) ItemAction {
		return ItemAction{
			Actor:      author,
			ItemNumber: itemNumber,
			Timestamp:  now,
		}
	}

	tests := []struct {
		name       string
		itemAuthor string
		actions    []ItemAction
		expected   bool
		reason     string
	}{
		{
			name:       "their item, no actions",
			itemAuthor: them,
			actions:    nil,
			expected:   true,
			reason:     "needs review",
		},
		{
			name:       "their item, I acted last",
			itemAuthor: them,
			actions:    []ItemAction{makeAction(me, 1)},
			expected:   false,
			reason:     "waiting for their reply",
		},
		{
			name:       "their item, they acted last",
			itemAuthor: them,
			actions:    []ItemAction{makeAction(me, 1), makeAction(them, 1)},
			expected:   true,
			reason:     "they pinged again",
		},
		{
			name:       "my item, no actions",
			itemAuthor: me,
			actions:    nil,
			expected:   false,
			reason:     "waiting for feedback",
		},
		{
			name:       "my item, they acted last",
			itemAuthor: me,
			actions:    []ItemAction{makeAction(them, 1)},
			expected:   true,
			reason:     "they replied",
		},
		{
			name:       "my item, I acted last",
			itemAuthor: me,
			actions:    []ItemAction{makeAction(them, 1), makeAction(me, 1)},
			expected:   false,
			reason:     "waiting for their reply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := makeItem(tt.itemAuthor, 1)
			got := BallInMyCourt(item, tt.actions, me)
			if got != tt.expected {
				t.Errorf("BallInMyCourt() = %v, want %v (%s)", got, tt.expected, tt.reason)
			}
		})
	}
}

func TestBallInMyCourtScenarios(t *testing.T) {
	me := "matsen"
	now := time.Now()

	// Helper to create an action with timestamp
	makeAction := func(author string, itemNumber int, when time.Time) ItemAction {
		return ItemAction{
			Actor:      author,
			ItemNumber: itemNumber,
			Timestamp:  when,
		}
	}

	// Scenario 1: Someone commented Saturday on an old issue you created months ago.
	// -> Show: they replied to your item
	t.Run("someone replied to my old issue", func(t *testing.T) {
		item := GitHubItem{
			Number:    1,
			User:      GitHubUser{Login: me},
			CreatedAt: now.Add(-90 * 24 * time.Hour), // Created months ago
			UpdatedAt: now,
		}
		actions := []ItemAction{
			makeAction("colleague", 1, now.Add(-2*24*time.Hour)), // Commented Saturday
		}
		if !BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction(me, 2, now.Add(-2*24*time.Hour)),
		}
		if BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction(me, 5, now),
		}
		if BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction(me, 6, now.Add(-1*time.Hour)),
			makeAction("colleague", 6, now),
		}
		if !BallInMyCourt(item, actions, me) {
			t.Error("Expected true: they replied to my review")
		}
	})
}

func TestBallInMyCourtWithPRReviews(t *testing.T) {
	me := "matsen"
	now := time.Now()

	// Helper to create an action
	makeAction := func(author string, itemNumber int, when time.Time) ItemAction {
		return ItemAction{
			Actor:      author,
			ItemNumber: itemNumber,
			Timestamp:  when,
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
		actions := []ItemAction{
			makeAction(me, 107, now),
		}
		if BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction(me, 107, now.Add(-1*time.Hour)),
			makeAction("colleague", 107, now),
		}
		if !BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction("reviewer", 50, now),
		}
		if !BallInMyCourt(item, actions, me) {
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
		actions := []ItemAction{
			makeAction(me, 200, now.Add(-5*time.Minute)),
			makeAction(me, 200, now),
		}
		if BallInMyCourt(item, actions, me) {
			t.Error("Expected false: all activity is mine (review + inline)")
		}
	})
}

func TestBallInMyCourtWithEvents(t *testing.T) {
	me := "matsen"
	now := time.Now()

	makeAction := func(author string, itemNumber int, when time.Time) ItemAction {
		return ItemAction{
			ItemNumber: itemNumber,
			Actor:      author,
			Timestamp:  when,
		}
	}

	// Scenario: Their PR, someone commented, then I merged it.
	// -> Hide: I acted last (merge counts as action)
	t.Run("their PR, I merged after their comment", func(t *testing.T) {
		item := GitHubItem{
			Number: 32,
			User:   GitHubUser{Login: "colleague"},
			IsPR:   true,
			State:  "closed",
		}
		actions := []ItemAction{
			makeAction("colleague", 32, now.Add(-2*time.Hour)), // their comment
			makeAction(me, 32, now),                            // I merged
		}
		if BallInMyCourt(item, actions, me) {
			t.Error("Expected false: I merged, ball is not in my court")
		}
	})

	// Scenario: Their issue, someone commented, then I closed it.
	// -> Hide: I acted last (close counts as action)
	t.Run("their issue, I closed after their comment", func(t *testing.T) {
		item := GitHubItem{
			Number: 38,
			User:   GitHubUser{Login: "colleague"},
			State:  "closed",
		}
		actions := []ItemAction{
			makeAction("other", 38, now.Add(-1*time.Hour)), // someone's comment
			makeAction(me, 38, now),                        // I closed
		}
		if BallInMyCourt(item, actions, me) {
			t.Error("Expected false: I closed, ball is not in my court")
		}
	})

	// Scenario: My issue, someone closed it (e.g., duplicate).
	// -> Show: they acted on my item
	t.Run("my issue, someone else closed it", func(t *testing.T) {
		item := GitHubItem{
			Number: 100,
			User:   GitHubUser{Login: me},
			State:  "closed",
		}
		actions := []ItemAction{
			makeAction("maintainer", 100, now), // they closed
		}
		if !BallInMyCourt(item, actions, me) {
			t.Error("Expected true: someone else closed my issue")
		}
	})

	// Scenario: Their PR, I commented, then they commented, then I merged.
	// -> Hide: I acted last despite back-and-forth
	t.Run("back and forth then I merged", func(t *testing.T) {
		item := GitHubItem{
			Number: 50,
			User:   GitHubUser{Login: "colleague"},
			IsPR:   true,
			State:  "closed",
		}
		actions := []ItemAction{
			makeAction(me, 50, now.Add(-3*time.Hour)),          // my review
			makeAction("colleague", 50, now.Add(-2*time.Hour)), // their response
			makeAction(me, 50, now),                            // I merged
		}
		if BallInMyCourt(item, actions, me) {
			t.Error("Expected false: I merged last")
		}
	})

	// Scenario: No actions have occurred on someone else's item.
	// -> Show: Ball is in my court by default (I should respond)
	t.Run("no actions on their item", func(t *testing.T) {
		item := GitHubItem{
			Number: 99,
			User:   GitHubUser{Login: "colleague"},
		}
		if !BallInMyCourt(item, nil, me) {
			t.Error("Expected true: no actions means ball is in my court")
		}
		if !BallInMyCourt(item, []ItemAction{}, me) {
			t.Error("Expected true: empty actions means ball is in my court")
		}
	})

	// Scenario: Actions are provided out of chronological order (e.g., from different API calls).
	// -> Sorting should identify my merge as the most recent action.
	t.Run("actions processed in chronological order despite slice order", func(t *testing.T) {
		item := GitHubItem{
			Number: 77,
			User:   GitHubUser{Login: "colleague"},
			IsPR:   true,
		}
		// Out-of-order in slice: merge is most recent but appears second
		actions := []ItemAction{
			makeAction("colleague", 77, now.Add(-3*time.Hour)), // their comment
			makeAction(me, 77, now),                            // I merged (latest)
			makeAction("other", 77, now.Add(-1*time.Hour)),     // review comment
		}
		if BallInMyCourt(item, actions, me) {
			t.Error("Expected false: my merge was most recent action")
		}
	})
}

func TestCommentsToActions(t *testing.T) {
	now := time.Now()

	t.Run("valid comments", func(t *testing.T) {
		comments := []GitHubComment{
			{
				User:      GitHubUser{Login: "alice"},
				IssueURL:  "https://api.github.com/repos/org/repo/issues/1",
				UpdatedAt: now,
			},
			{
				User:      GitHubUser{Login: "bob"},
				PRURL:     "https://api.github.com/repos/org/repo/pulls/2",
				UpdatedAt: now.Add(-1 * time.Hour),
			},
		}

		actions := CommentsToActions(comments)

		if len(actions) != 2 {
			t.Fatalf("Expected 2 actions, got %d", len(actions))
		}
		if actions[0].Actor != "alice" || actions[0].ItemNumber != 1 {
			t.Errorf("Action 0: got actor=%s item=%d", actions[0].Actor, actions[0].ItemNumber)
		}
		if actions[1].Actor != "bob" || actions[1].ItemNumber != 2 {
			t.Errorf("Action 1: got actor=%s item=%d", actions[1].Actor, actions[1].ItemNumber)
		}
	})

	t.Run("skips malformed comments", func(t *testing.T) {
		comments := []GitHubComment{
			{User: GitHubUser{Login: "alice"}, IssueURL: "https://api.github.com/repos/org/repo/issues/1", UpdatedAt: now},
			{User: GitHubUser{Login: ""}, IssueURL: "https://api.github.com/repos/org/repo/issues/2", UpdatedAt: now}, // deleted user
			{User: GitHubUser{Login: "bob"}, IssueURL: "", PRURL: "", UpdatedAt: now},                                 // no URL
		}

		actions := CommentsToActions(comments)

		if len(actions) != 1 {
			t.Fatalf("Expected 1 action (malformed skipped), got %d", len(actions))
		}
		if actions[0].Actor != "alice" {
			t.Errorf("Expected alice, got %s", actions[0].Actor)
		}
	})
}

func TestEventsToActions(t *testing.T) {
	now := time.Now()

	// Helper to create a GitHubEvent
	makeEvent := func(event string, actor string, issueNumber int, when time.Time) GitHubEvent {
		e := GitHubEvent{
			Event:     event,
			Actor:     GitHubUser{Login: actor},
			CreatedAt: when,
		}
		e.Issue.Number = issueNumber
		return e
	}

	t.Run("valid events", func(t *testing.T) {
		events := []GitHubEvent{
			makeEvent("closed", "closer", 10, now),
			makeEvent("merged", "merger", 20, now.Add(-1*time.Hour)),
		}

		actions := EventsToActions(events)

		if len(actions) != 2 {
			t.Fatalf("Expected 2 actions, got %d", len(actions))
		}
		if actions[0].Actor != "closer" || actions[0].ItemNumber != 10 {
			t.Errorf("Action 0: got actor=%s item=%d", actions[0].Actor, actions[0].ItemNumber)
		}
		if actions[1].Actor != "merger" || actions[1].ItemNumber != 20 {
			t.Errorf("Action 1: got actor=%s item=%d", actions[1].Actor, actions[1].ItemNumber)
		}
	})

	t.Run("skips malformed events", func(t *testing.T) {
		events := []GitHubEvent{
			makeEvent("closed", "closer", 10, now),
			makeEvent("closed", "", 11, now),     // deleted user
			makeEvent("closed", "other", 0, now), // no issue number
		}

		actions := EventsToActions(events)

		if len(actions) != 1 {
			t.Fatalf("Expected 1 action (malformed skipped), got %d", len(actions))
		}
		if actions[0].Actor != "closer" {
			t.Errorf("Expected closer, got %s", actions[0].Actor)
		}
	})
}

func TestFilterByBallInCourt(t *testing.T) {
	me := "me"
	now := time.Now()

	items := []GitHubItem{
		{Number: 1, User: GitHubUser{Login: "them"}, UpdatedAt: now}, // Show: their item
		{Number: 2, User: GitHubUser{Login: me}, UpdatedAt: now},     // Hide: my item, no actions
		{Number: 3, User: GitHubUser{Login: "them"}, UpdatedAt: now}, // Hide: I acted last
	}

	actions := []ItemAction{
		{Actor: me, ItemNumber: 3, Timestamp: now},
	}

	filtered := FilterByBallInCourt(items, actions, me)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 item, got %d", len(filtered))
		return
	}

	if filtered[0].Number != 1 {
		t.Errorf("Expected item #1, got #%d", filtered[0].Number)
	}
}

// Keep strconv import for potential future use
var _ = strconv.Itoa
