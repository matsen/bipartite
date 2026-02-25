package flow

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CommentsToActions converts GitHubComments to ItemActions.
// Skips malformed entries (missing item number or actor).
//
// Note: We use UpdatedAt instead of CreatedAt because substantive comment edits
// (adding information, answering questions) should count as new actions. This
// means cosmetic edits (typo fixes) may incorrectly flip ball-in-court status,
// but this is preferable to ignoring real follow-up information.
func CommentsToActions(comments []GitHubComment) []ItemAction {
	actions := make([]ItemAction, 0, len(comments))
	for _, c := range comments {
		itemNum := getCommentItemNumber(c)
		if itemNum == 0 {
			// Skip malformed comment (no item URL)
			continue
		}
		if c.User.Login == "" {
			// Skip comments from deleted users
			continue
		}
		actions = append(actions, ItemAction{
			ItemNumber: itemNum,
			Actor:      c.User.Login,
			Timestamp:  c.UpdatedAt,
		})
	}
	return actions
}

// EventsToActions converts GitHubEvents to ItemActions.
// Skips malformed entries (missing item number or actor).
//
// Note: We use CreatedAt (not UpdatedAt) because GitHub events are immutable.
// Unlike comments which can be edited, a close or merge event represents a
// single point-in-time action that cannot be modified after the fact.
func EventsToActions(events []GitHubEvent) []ItemAction {
	actions := make([]ItemAction, 0, len(events))
	for _, e := range events {
		if e.Issue.Number == 0 {
			// Skip malformed event
			continue
		}
		if e.Actor.Login == "" {
			// Skip events from deleted users
			continue
		}
		actions = append(actions, ItemAction{
			ItemNumber: e.Issue.Number,
			Actor:      e.Actor.Login,
			Timestamp:  e.CreatedAt,
		})
	}
	return actions
}

// BallInMyCourt determines if the user needs to act on an item.
//
// Truth table:
//
//	Their item, no actions        -> true  (need to review)
//	Their item, they acted last   -> true  (they pinged again)
//	Their item, I acted last      -> false (waiting for their reply)
//	My item, no actions           -> false (waiting for feedback)
//	My item, they acted last      -> true  (they replied)
//	My item, I acted last         -> false (waiting for their reply)
//
// Actions include comments, close events, and merge events.
func BallInMyCourt(item GitHubItem, actions []ItemAction, githubUser string) bool {
	author := item.User.Login
	isMyItem := author == githubUser

	// Filter actions to only those on this item
	itemActions := filterActionsForItem(actions, item.Number)

	if len(itemActions) == 0 {
		// No actions: show their items (need review), hide mine (waiting for feedback)
		return !isMyItem
	}

	// Has actions: show if last actor is not me (they're waiting for my response)
	sortActionsByTime(itemActions)
	lastActor := itemActions[len(itemActions)-1].Actor

	return lastActor != "" && lastActor != githubUser
}

// filterActionsForItem returns actions that belong to the given item number.
func filterActionsForItem(actions []ItemAction, itemNumber int) []ItemAction {
	var filtered []ItemAction
	for _, a := range actions {
		if a.ItemNumber == itemNumber {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// getCommentItemNumber extracts the issue/PR number a comment belongs to.
func getCommentItemNumber(comment GitHubComment) int {
	url := comment.IssueURL
	if url == "" {
		url = comment.PRURL
	}
	if url == "" {
		return 0
	}

	// Extract number from URL like ".../issues/123"
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return 0
	}
	numStr := parts[len(parts)-1]
	n, _ := strconv.Atoi(numStr)
	return n
}

// sortActionsByTime sorts actions by timestamp.
func sortActionsByTime(actions []ItemAction) {
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Timestamp.Before(actions[j].Timestamp)
	})
}

// FilterByBallInCourt filters items to only those where ball is in user's court.
func FilterByBallInCourt(items []GitHubItem, actions []ItemAction, githubUser string) []GitHubItem {
	var filtered []GitHubItem
	for _, item := range items {
		if BallInMyCourt(item, actions, githubUser) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// EnrichActionsWithLastComments fetches the last issue-thread comment for items
// that have no actions in the current window. This prevents ball-in-court from
// falling through to the author-based default when the user's last comment
// predates the since window.
//
// Note: only issue-thread comments (/issues/{n}/comments) are considered here.
// PR review submissions and inline review comments are handled separately
// via FetchPRReviewsAsComments.
func EnrichActionsWithLastComments(repo string, items []GitHubItem, existingActions []ItemAction) []ItemAction {
	// Build set of item numbers that already have actions
	hasActions := make(map[int]bool)
	for _, a := range existingActions {
		hasActions[a.ItemNumber] = true
	}

	var enriched []ItemAction
	for _, item := range items {
		if hasActions[item.Number] {
			continue
		}

		comment, err := FetchLastItemComment(repo, item.Number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch last comment for %s#%d: %v\n", repo, item.Number, err)
			continue
		}
		if comment == nil {
			continue
		}
		if comment.User.Login == "" {
			continue
		}

		itemNum := getCommentItemNumber(*comment)
		if itemNum == 0 {
			// Should not occur unless the API returns a comment without issue_url;
			// fall back to the known item number.
			itemNum = item.Number
		}
		enriched = append(enriched, ItemAction{
			ItemNumber: itemNum,
			Actor:      comment.User.Login,
			Timestamp:  comment.UpdatedAt,
		})
	}

	return enriched
}

// FilterCommentsByItems returns comments that belong to the given items.
func FilterCommentsByItems(comments []GitHubComment, items []GitHubItem) []GitHubComment {
	itemNumbers := make(map[int]bool)
	for _, item := range items {
		itemNumbers[item.Number] = true
	}

	var filtered []GitHubComment
	for _, c := range comments {
		if itemNumbers[getCommentItemNumber(c)] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
