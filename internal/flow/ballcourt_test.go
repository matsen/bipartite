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

func TestBallInMyCourtStrict(t *testing.T) {
	me := "jgallowa07"
	now := time.Now()

	t.Run("known-answer mix", func(t *testing.T) {
		// 5 items from different authors, various involvement signals.
		items := []GitHubItem{
			{ // #1: assigned to me
				Number:    1,
				User:      GitHubUser{Login: "alice"},
				Assignees: []GitHubUser{{Login: me}},
				UpdatedAt: now,
			},
			{ // #2: mentioned in body
				Number:    2,
				User:      GitHubUser{Login: "bob"},
				Body:      "hey @jgallowa07, can you look at this?",
				UpdatedAt: now,
			},
			{ // #3: I previously commented
				Number:    3,
				User:      GitHubUser{Login: "carol"},
				UpdatedAt: now,
			},
			{ // #4: requested reviewer
				Number:             4,
				User:               GitHubUser{Login: "dave"},
				IsPR:               true,
				RequestedReviewers: []GitHubUser{{Login: me}},
				UpdatedAt:          now,
			},
			{ // #5: no connection
				Number:    5,
				User:      GitHubUser{Login: "eve"},
				UpdatedAt: now,
			},
		}

		inv := Involvement{
			Commenters: map[int][]string{
				3: {"carol", me, "frank"}, // I'm a past commenter on #3
				5: {"eve", "frank"},       // not me
			},
		}

		for _, n := range []int{1, 2, 3, 4} {
			if !BallInMyCourtStrict(items[n-1], nil, me, inv) {
				t.Errorf("item #%d: expected ball-in-my-court (involvement signal present)", n)
			}
		}
		if BallInMyCourtStrict(items[4], nil, me, inv) {
			t.Error("item #5: expected NOT ball-in-my-court (no connection)")
		}
	})

	t.Run("their item no actions no involvement -> false", func(t *testing.T) {
		// Regression for issue #123: EPIC #506 in dasm2-experiments authored by
		// matsen with no assignees, no comments, no mentions of jgallowa07.
		// Broad filter would show this; strict filter must drop it.
		item := GitHubItem{
			Number: 506,
			User:   GitHubUser{Login: "matsen"},
		}
		if BallInMyCourtStrict(item, nil, me, Involvement{}) {
			t.Error("Expected false: teammate item with no involvement should be dropped by strict filter")
		}
	})

	t.Run("two-person repo regression", func(t *testing.T) {
		// In a small repo where the user is assigned to or a requested reviewer
		// on every teammate item, the strict filter matches the broad filter.
		// No regression for small-team workflows.
		items := []GitHubItem{
			{Number: 1, User: GitHubUser{Login: "colleague"}, Assignees: []GitHubUser{{Login: me}}},
			{Number: 2, User: GitHubUser{Login: "colleague"}, IsPR: true, RequestedReviewers: []GitHubUser{{Login: me}}},
			{Number: 3, User: GitHubUser{Login: me}}, // my own, no actions
		}
		for _, item := range items {
			broad := BallInMyCourt(item, nil, me)
			strict := BallInMyCourtStrict(item, nil, me, Involvement{})
			if broad != strict {
				t.Errorf("item #%d: broad=%v strict=%v; expected agreement when user is connected to every teammate item", item.Number, broad, strict)
			}
		}
	})

	t.Run("stale engagement still counts", func(t *testing.T) {
		// Their item, no actions in window, I commented 6 months ago.
		// Should still show as ball-in-court.
		item := GitHubItem{
			Number: 42,
			User:   GitHubUser{Login: "colleague"},
		}
		inv := Involvement{
			Commenters: map[int][]string{42: {"colleague", me}},
		}
		if !BallInMyCourtStrict(item, nil, me, inv) {
			t.Error("Expected true: past engagement should keep ball-in-court even without window actions")
		}
	})

	t.Run("strict matches base when actions present", func(t *testing.T) {
		// When there are actions in the window, strict and base behave identically:
		// last actor decides, involvement is irrelevant.
		item := GitHubItem{
			Number: 7,
			User:   GitHubUser{Login: "colleague"},
		}
		actions := []ItemAction{
			{Actor: "colleague", ItemNumber: 7, Timestamp: now.Add(-1 * time.Hour)},
			{Actor: me, ItemNumber: 7, Timestamp: now},
		}
		// I acted last -> false in both filters
		if BallInMyCourtStrict(item, actions, me, Involvement{}) {
			t.Error("Expected false: I acted last")
		}
		if BallInMyCourt(item, actions, me) != BallInMyCourtStrict(item, actions, me, Involvement{}) {
			t.Error("Strict and base should agree when actions exist")
		}
	})

	t.Run("my item no actions still false under strict", func(t *testing.T) {
		// Involvement does not matter for my own items.
		item := GitHubItem{
			Number:    10,
			User:      GitHubUser{Login: me},
			Assignees: []GitHubUser{{Login: me}},
		}
		if BallInMyCourtStrict(item, nil, me, Involvement{}) {
			t.Error("Expected false: my own item with no actions is waiting for feedback")
		}
	})

	t.Run("mention in code block does not count", func(t *testing.T) {
		item := GitHubItem{
			Number: 20,
			User:   GitHubUser{Login: "colleague"},
			Body:   "here is some code:\n```\n@jgallowa07 wrote this\n```\nthat's it",
		}
		if BallInMyCourtStrict(item, nil, me, Involvement{}) {
			t.Error("Expected false: @mention inside fenced code block should not count")
		}
	})

	t.Run("mention in inline code does not count", func(t *testing.T) {
		item := GitHubItem{
			Number: 21,
			User:   GitHubUser{Login: "colleague"},
			Body:   "the handle `@jgallowa07` is formatted as code, not a mention",
		}
		if BallInMyCourtStrict(item, nil, me, Involvement{}) {
			t.Error("Expected false: @mention inside inline code should not count")
		}
	})

	t.Run("mention as substring of longer handle does not count", func(t *testing.T) {
		item := GitHubItem{
			Number: 22,
			User:   GitHubUser{Login: "colleague"},
			Body:   "cc @jgallowa07-alt please",
		}
		if BallInMyCourtStrict(item, nil, me, Involvement{}) {
			t.Error("Expected false: @jgallowa07-alt is a different handle, not a mention of jgallowa07")
		}
	})
}

func TestBodyMentionsUser(t *testing.T) {
	cases := []struct {
		name string
		body string
		user string
		want bool
	}{
		{"simple mention", "hey @alice", "alice", true},
		{"mention at start", "@alice please review", "alice", true},
		{"mention after punctuation", "cc: @alice, thanks", "alice", true},
		{"case insensitive", "cc @Alice", "alice", true},
		{"no mention", "alice is the author", "alice", false},
		{"email-like not a mention", "alice@example.com", "alice", false},
		{"longer handle not a match", "cc @alice-bot", "alice", false},
		{"underscore continuation not a match", "cc @alice_bot", "alice", false},
		{"substring prefix not a match", "cc @alicia", "alice", false},
		{"inside fenced block", "```\n@alice\n```", "alice", false},
		{"inside inline code", "use `@alice` as handle", "alice", false},
		{"mixed: mention + code", "`@alice` means @alice", "alice", true},
		{"empty body", "", "alice", false},
		{"empty user", "hey @alice", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bodyMentionsUser(c.body, c.user); got != c.want {
				t.Errorf("bodyMentionsUser(%q, %q) = %v, want %v", c.body, c.user, got, c.want)
			}
		})
	}
}

func TestHasInvolvement(t *testing.T) {
	me := "me"
	t.Run("assignee", func(t *testing.T) {
		item := GitHubItem{Assignees: []GitHubUser{{Login: me}}}
		if !HasInvolvement(item, me, Involvement{}) {
			t.Error("expected true: assignee")
		}
	})
	t.Run("requested reviewer", func(t *testing.T) {
		item := GitHubItem{RequestedReviewers: []GitHubUser{{Login: me}}}
		if !HasInvolvement(item, me, Involvement{}) {
			t.Error("expected true: requested reviewer")
		}
	})
	t.Run("mention", func(t *testing.T) {
		item := GitHubItem{Body: "cc @me here"}
		if !HasInvolvement(item, me, Involvement{}) {
			t.Error("expected true: mention")
		}
	})
	t.Run("past commenter", func(t *testing.T) {
		item := GitHubItem{Number: 1}
		inv := Involvement{Commenters: map[int][]string{1: {"other", me}}}
		if !HasInvolvement(item, me, inv) {
			t.Error("expected true: past commenter")
		}
	})
	t.Run("no signals", func(t *testing.T) {
		item := GitHubItem{Number: 1, Body: "nothing here"}
		inv := Involvement{Commenters: map[int][]string{1: {"other"}}}
		if HasInvolvement(item, me, inv) {
			t.Error("expected false: no involvement signals")
		}
	})
	t.Run("empty user", func(t *testing.T) {
		item := GitHubItem{Assignees: []GitHubUser{{Login: ""}}}
		if HasInvolvement(item, "", Involvement{}) {
			t.Error("expected false: empty user should not match empty assignee login")
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
