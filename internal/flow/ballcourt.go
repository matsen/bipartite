package flow

import (
	"sort"
	"strconv"
	"strings"
)

// BallInMyCourt determines if the user needs to act on an item.
//
// Truth table:
//
//	Their item, no comments       -> true  (need to review)
//	Their item, they commented    -> true  (they pinged again)
//	Their item, I commented last  -> false (waiting for their reply)
//	My item, no comments          -> false (waiting for feedback)
//	My item, they commented last  -> true  (they replied)
//	My item, I commented last     -> false (waiting for their reply)
func BallInMyCourt(item GitHubItem, comments []GitHubComment, githubUser string) bool {
	author := item.User.Login
	isMyItem := author == githubUser

	// Filter comments to only those on this item
	itemComments := filterCommentsForItem(comments, item.Number)

	if len(itemComments) == 0 {
		// No comments: show their items (need review), hide mine (waiting for feedback)
		return !isMyItem
	}

	// Has comments: show if last commenter is not me (they're waiting for my response)
	sortCommentsByTime(itemComments)
	lastCommenter := itemComments[len(itemComments)-1].User.Login

	return lastCommenter != "" && lastCommenter != githubUser
}

// filterCommentsForItem returns comments that belong to the given item number.
func filterCommentsForItem(comments []GitHubComment, itemNumber int) []GitHubComment {
	var filtered []GitHubComment
	for _, c := range comments {
		if getCommentItemNumber(c) == itemNumber {
			filtered = append(filtered, c)
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

// sortCommentsByTime sorts comments by updated_at time.
func sortCommentsByTime(comments []GitHubComment) {
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].UpdatedAt.Before(comments[j].UpdatedAt)
	})
}

// FilterByBallInCourt filters items to only those where ball is in user's court.
func FilterByBallInCourt(items []GitHubItem, comments []GitHubComment, githubUser string) []GitHubItem {
	var filtered []GitHubItem
	for _, item := range items {
		if BallInMyCourt(item, comments, githubUser) {
			filtered = append(filtered, item)
		}
	}
	return filtered
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
