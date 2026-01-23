"""Tests for ball-in-my-court filtering logic."""

import pytest

from fc_cli.checkin.activity import (
    ball_in_my_court,
    filter_by_ball_in_court,
    filter_comments_by_items,
)


def make_item(number: int, author: str) -> dict:
    """Create a minimal issue/PR dict."""
    return {"number": number, "user": {"login": author}}


def make_comment(item_number: int, author: str, updated_at: str) -> dict:
    """Create a minimal comment dict."""
    return {
        "issue_url": f"https://api.github.com/repos/org/repo/issues/{item_number}",
        "user": {"login": author},
        "updated_at": updated_at,
    }


class TestBallInMyCourt:
    """Tests for ball_in_my_court() truth table and scenarios."""

    # Truth table tests
    def test_their_item_no_comments_shows(self):
        """Their item, no comments -> True (need to review)."""
        item = make_item(1, "other_user")
        assert ball_in_my_court(item, [], "matsen") is True

    def test_their_item_i_commented_last_hides(self):
        """Their item, I commented last -> False (waiting for their reply)."""
        item = make_item(1, "other_user")
        comments = [make_comment(1, "matsen", "2024-01-15T10:00:00Z")]
        assert ball_in_my_court(item, comments, "matsen") is False

    def test_their_item_they_commented_last_shows(self):
        """Their item, they commented last -> True (they pinged again)."""
        item = make_item(1, "other_user")
        comments = [make_comment(1, "other_user", "2024-01-15T10:00:00Z")]
        assert ball_in_my_court(item, comments, "matsen") is True

    def test_my_item_no_comments_hides(self):
        """My item, no comments -> False (waiting for feedback)."""
        item = make_item(1, "matsen")
        assert ball_in_my_court(item, [], "matsen") is False

    def test_my_item_they_commented_last_shows(self):
        """My item, they commented last -> True (they replied)."""
        item = make_item(1, "matsen")
        comments = [make_comment(1, "other_user", "2024-01-15T10:00:00Z")]
        assert ball_in_my_court(item, comments, "matsen") is True

    def test_my_item_i_commented_last_hides(self):
        """My item, I commented last -> False (waiting for their reply)."""
        item = make_item(1, "matsen")
        comments = [make_comment(1, "matsen", "2024-01-15T10:00:00Z")]
        assert ball_in_my_court(item, comments, "matsen") is False

    # Scenario tests from docstring
    def test_scenario_1_someone_commented_on_old_issue(self):
        """Scenario 1: Someone commented Saturday on an old issue you created.

        -> Show: they replied to your item, ball is in your court.
        """
        item = make_item(42, "matsen")  # My old issue
        comments = [make_comment(42, "colleague", "2024-01-13T14:00:00Z")]  # Saturday
        assert ball_in_my_court(item, comments, "matsen") is True

    def test_scenario_2_i_added_context_to_my_issue(self):
        """Scenario 2: You added a comment Saturday to your own issue.

        -> Hide: you commented last, waiting for their reply.
        """
        item = make_item(42, "matsen")
        comments = [make_comment(42, "matsen", "2024-01-13T14:00:00Z")]
        assert ball_in_my_court(item, comments, "matsen") is False

    def test_scenario_3_someone_opened_pr_no_comments(self):
        """Scenario 3: Someone opened a new PR Friday, no comments yet.

        -> Show: their item needs your review.
        """
        item = make_item(99, "contributor")
        assert ball_in_my_court(item, [], "matsen") is True

    def test_scenario_4_i_opened_pr_no_comments(self):
        """Scenario 4: You opened a PR Friday, no comments yet.

        -> Hide: waiting for feedback on your item.
        """
        item = make_item(99, "matsen")
        assert ball_in_my_court(item, [], "matsen") is False

    def test_scenario_5_i_reviewed_their_pr(self):
        """Scenario 5: You reviewed someone's PR and left comments.

        -> Hide: you commented last, waiting for them to address.
        """
        item = make_item(50, "contributor")  # Their PR
        comments = [make_comment(50, "matsen", "2024-01-15T10:00:00Z")]  # My review
        assert ball_in_my_court(item, comments, "matsen") is False

    def test_scenario_6_they_replied_to_my_review(self):
        """Scenario 6: Someone replied to your review on their PR.

        -> Show: they responded, ball is in your court.
        """
        item = make_item(50, "contributor")  # Their PR
        comments = [
            make_comment(50, "matsen", "2024-01-15T10:00:00Z"),  # My review
            make_comment(50, "contributor", "2024-01-15T14:00:00Z"),  # Their reply
        ]
        assert ball_in_my_court(item, comments, "matsen") is True

    # Edge cases
    def test_multiple_comments_on_my_item_uses_last(self):
        """With multiple comments on my item, the last commenter determines visibility."""
        item = make_item(1, "matsen")
        comments = [
            make_comment(1, "other_user", "2024-01-15T08:00:00Z"),
            make_comment(1, "matsen", "2024-01-15T09:00:00Z"),
            make_comment(1, "other_user", "2024-01-15T10:00:00Z"),  # Last
        ]
        assert ball_in_my_court(item, comments, "matsen") is True

    def test_multiple_comments_on_their_item_uses_last(self):
        """With multiple comments on their item, the last commenter determines visibility."""
        item = make_item(1, "other_user")
        # Back-and-forth review: I commented, they replied, I commented again
        comments = [
            make_comment(1, "matsen", "2024-01-15T08:00:00Z"),
            make_comment(1, "other_user", "2024-01-15T09:00:00Z"),
            make_comment(1, "matsen", "2024-01-15T10:00:00Z"),  # Last (me)
        ]
        # I commented last -> hide (waiting for them)
        assert ball_in_my_court(item, comments, "matsen") is False

    def test_multiple_comments_on_their_item_they_last(self):
        """Their item, they commented last after back-and-forth -> show."""
        item = make_item(1, "other_user")
        comments = [
            make_comment(1, "matsen", "2024-01-15T08:00:00Z"),
            make_comment(1, "other_user", "2024-01-15T09:00:00Z"),  # Last (them)
        ]
        assert ball_in_my_court(item, comments, "matsen") is True

    def test_comments_on_other_items_ignored(self):
        """Comments on different items don't affect this item's visibility."""
        item = make_item(1, "matsen")
        comments = [
            make_comment(2, "other_user", "2024-01-15T10:00:00Z"),  # Different item
            make_comment(3, "other_user", "2024-01-15T11:00:00Z"),  # Different item
        ]
        # My item with no comments on it -> hide
        assert ball_in_my_court(item, comments, "matsen") is False


class TestFilterByBallInCourt:
    """Tests for filter_by_ball_in_court()."""

    def test_filters_to_actionable_items(self):
        """Only items needing action are returned."""
        items = [
            make_item(1, "other_user"),  # Their item -> show
            make_item(2, "matsen"),  # My item, no comments -> hide
            make_item(3, "matsen"),  # My item, they replied -> show
        ]
        comments = [make_comment(3, "other_user", "2024-01-15T10:00:00Z")]

        result = filter_by_ball_in_court(items, comments, "matsen")

        assert len(result) == 2
        assert result[0]["number"] == 1
        assert result[1]["number"] == 3

    def test_empty_items_returns_empty(self):
        """Empty input returns empty output."""
        assert filter_by_ball_in_court([], [], "matsen") == []


class TestFilterCommentsByItems:
    """Tests for filter_comments_by_items()."""

    def test_keeps_comments_on_shown_items(self):
        """Comments on items in the list are kept."""
        items = [make_item(1, "other"), make_item(3, "other")]
        comments = [
            make_comment(1, "user", "2024-01-15T10:00:00Z"),
            make_comment(2, "user", "2024-01-15T10:00:00Z"),  # Item 2 not in list
            make_comment(3, "user", "2024-01-15T10:00:00Z"),
        ]

        result = filter_comments_by_items(comments, items)

        assert len(result) == 2
        assert all(c["issue_url"].endswith("/1") or c["issue_url"].endswith("/3") for c in result)

    def test_empty_items_returns_empty(self):
        """No items means no comments kept."""
        comments = [make_comment(1, "user", "2024-01-15T10:00:00Z")]
        assert filter_comments_by_items(comments, []) == []
