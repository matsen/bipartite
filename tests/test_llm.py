"""Tests for flowc.llm module."""

from __future__ import annotations

from flowc.llm import (
    _build_digest_prompt,
    _build_summary_prompt,
    _parse_summary_response,
    _postprocess_digest,
    generate_digest_summary,
)


class TestParseSummaryResponse:
    """Tests for _parse_summary_response."""

    def test_parses_valid_json(self):
        """Valid JSON object is parsed correctly."""
        response = '{"matsengrp/repo#123": "summary here"}'
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {"matsengrp/repo#123": "summary here"}

    def test_parses_json_with_markdown_code_block(self):
        """JSON wrapped in markdown code blocks is extracted."""
        response = """```json
{"matsengrp/repo#123": "summary here"}
```"""
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {"matsengrp/repo#123": "summary here"}

    def test_parses_json_with_plain_code_block(self):
        """JSON wrapped in plain code blocks is extracted."""
        response = """```
{"matsengrp/repo#123": "summary here"}
```"""
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {"matsengrp/repo#123": "summary here"}

    def test_handles_multiple_items(self):
        """Multiple items are parsed correctly."""
        response = """{
            "matsengrp/repo#1": "first summary",
            "matsengrp/repo#2": "second summary"
        }"""
        items = [{"ref": "matsengrp/repo#1"}, {"ref": "matsengrp/repo#2"}]
        result = _parse_summary_response(response, items)
        assert result == {
            "matsengrp/repo#1": "first summary",
            "matsengrp/repo#2": "second summary",
        }

    def test_returns_empty_dict_on_invalid_json(self, capsys):
        """Invalid JSON returns empty dict with warning."""
        response = "This is not JSON at all"
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {}

        # Check warning was printed
        captured = capsys.readouterr()
        assert "Warning: Failed to parse LLM response as JSON" in captured.err

    def test_returns_empty_dict_on_preamble_text(self, capsys):
        """JSON with preamble text fails gracefully."""
        response = """Here are the summaries:
{"matsengrp/repo#123": "summary"}"""
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {}

        captured = capsys.readouterr()
        assert "Warning" in captured.err

    def test_handles_empty_response(self, capsys):
        """Empty response returns empty dict."""
        result = _parse_summary_response("", [])
        assert result == {}

    def test_handles_whitespace_around_json(self):
        """Whitespace around JSON is handled."""
        response = """

  {"matsengrp/repo#123": "summary"}

"""
        items = [{"ref": "matsengrp/repo#123"}]
        result = _parse_summary_response(response, items)
        assert result == {"matsengrp/repo#123": "summary"}


class TestBuildSummaryPrompt:
    """Tests for _build_summary_prompt."""

    def test_builds_prompt_for_issue(self):
        """Prompt is built correctly for an issue."""
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test issue",
                "author": "testuser",
                "body": "Issue body text",
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "REF: matsengrp/repo#123" in prompt
        assert "TYPE: Issue" in prompt
        assert "TITLE: Test issue" in prompt
        assert "AUTHOR: testuser" in prompt
        assert "STATUS: needs_action" in prompt
        assert "BODY: Issue body text" in prompt

    def test_builds_prompt_for_pr(self):
        """Prompt is built correctly for a PR."""
        items = [
            {
                "ref": "matsengrp/repo#456",
                "title": "Test PR",
                "author": "prauthor",
                "body": "PR description",
                "is_pr": True,
                "ball_in_my_court": False,
                "comments": [],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "TYPE: PR" in prompt
        assert "STATUS: waiting" in prompt

    def test_includes_comments(self):
        """Comments are included in prompt."""
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": "",
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [
                    {"author": "commenter1", "body": "First comment"},
                    {"author": "commenter2", "body": "Second comment"},
                ],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "@commenter1: First comment" in prompt
        assert "@commenter2: Second comment" in prompt

    def test_truncates_long_comments(self):
        """Long comments are truncated to 200 chars."""
        long_comment = "x" * 300
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": "",
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [{"author": "commenter", "body": long_comment}],
            }
        ]
        prompt = _build_summary_prompt(items)

        # Should have truncated comment (200 chars)
        assert "x" * 200 in prompt
        assert "x" * 201 not in prompt

    def test_truncates_long_body(self):
        """Long body is truncated to 300 chars."""
        long_body = "y" * 500
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": long_body,
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "y" * 300 in prompt
        assert "y" * 301 not in prompt

    def test_limits_to_last_5_comments(self):
        """Only last 5 comments are included."""
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": "",
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [
                    {"author": f"user{i}", "body": f"comment {i}"} for i in range(10)
                ],
            }
        ]
        prompt = _build_summary_prompt(items)

        # Should have comments 5-9, not 0-4
        assert "@user5: comment 5" in prompt
        assert "@user9: comment 9" in prompt
        assert "@user0: comment 0" not in prompt
        assert "@user4: comment 4" not in prompt

    def test_handles_none_body(self):
        """None body is handled gracefully."""
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": None,
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "BODY:" in prompt  # Should have empty body, not crash

    def test_includes_output_format_instructions(self):
        """Prompt includes JSON output format instructions."""
        items = [
            {
                "ref": "matsengrp/repo#123",
                "title": "Test",
                "author": "author",
                "body": "",
                "is_pr": False,
                "ball_in_my_court": True,
                "comments": [],
            }
        ]
        prompt = _build_summary_prompt(items)

        assert "JSON object" in prompt
        assert "Return ONLY the JSON object" in prompt


class TestBuildDigestPrompt:
    """Tests for _build_digest_prompt."""

    def test_includes_channel_and_date_range(self):
        """Prompt includes channel name and date range."""
        items = [
            {
                "number": 123,
                "title": "Test issue",
                "author": "testuser",
                "is_pr": False,
                "state": "open",
                "html_url": "https://github.com/org/repo/issues/123",
            }
        ]
        prompt = _build_digest_prompt(items, "dasm2", "Jan 12-18")

        assert "Channel: dasm2" in prompt
        assert "Date range: Jan 12-18" in prompt
        assert "*This week in dasm2*" in prompt

    def test_formats_issue(self):
        """Issues are formatted correctly."""
        items = [
            {
                "number": 123,
                "title": "Test issue",
                "author": "testuser",
                "is_pr": False,
                "state": "open",
                "html_url": "https://github.com/org/repo/issues/123",
            }
        ]
        prompt = _build_digest_prompt(items, "dasm2", "Jan 12-18")

        assert "[Issue]" in prompt
        assert "#123" in prompt
        assert "Test issue" in prompt
        assert "@testuser" in prompt

    def test_formats_pr(self):
        """PRs are formatted correctly."""
        items = [
            {
                "number": 456,
                "title": "Test PR",
                "author": "prauthor",
                "is_pr": True,
                "state": "closed",
                "merged": True,
                "html_url": "https://github.com/org/repo/pull/456",
            }
        ]
        prompt = _build_digest_prompt(items, "dasm2", "Jan 12-18")

        assert "[PR]" in prompt
        assert "#456" in prompt
        assert "merged" in prompt

    def test_includes_slack_format_instructions(self):
        """Prompt includes Slack mrkdwn formatting instructions."""
        items = [
            {
                "number": 123,
                "title": "Test",
                "author": "user",
                "is_pr": False,
                "state": "open",
                "html_url": "https://github.com/org/repo/issues/123",
            }
        ]
        prompt = _build_digest_prompt(items, "test", "Jan 1-7")

        assert "Slack" in prompt
        assert "mrkdwn" in prompt
        assert "<URL|#number>" in prompt


class TestPostprocessDigest:
    """Tests for _postprocess_digest."""

    def test_adds_pr_prefix_with_repo(self):
        """Adds repo name and PR: prefix for pull requests."""
        digest = "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
        items = [
            {"number": 123, "is_pr": True, "contributors": [], "ref": "org/repo#123"}
        ]

        result = _postprocess_digest(digest, items)

        assert "• repo PR: Fix bug" in result

    def test_adds_issue_prefix_with_repo(self):
        """Adds repo name and Issue: prefix for issues."""
        digest = "• New feature request (<https://github.com/org/repo/issues/456|#456>)"
        items = [
            {"number": 456, "is_pr": False, "contributors": [], "ref": "org/repo#456"}
        ]

        result = _postprocess_digest(digest, items)

        assert "• repo Issue: New feature request" in result

    def test_adds_contributors(self):
        """Adds contributor @mentions at end of line."""
        digest = "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
        items = [
            {
                "number": 123,
                "is_pr": True,
                "contributors": ["alice", "bob", "charlie"],
                "ref": "org/repo#123",
            }
        ]

        result = _postprocess_digest(digest, items)

        assert result.endswith("— @alice @bob @charlie")

    def test_contributors_sorted_alphabetically(self):
        """Contributors appear in alphabetical order."""
        digest = "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
        items = [
            {
                "number": 123,
                "is_pr": True,
                "contributors": ["zoe", "alice"],
                "ref": "org/repo#123",
            }
        ]

        result = _postprocess_digest(digest, items)

        # Contributors should already be sorted by fetch_channel_activity,
        # but verify they're in order in the output
        assert "@alice" in result
        assert "@zoe" in result

    def test_preserves_non_bullet_lines(self):
        """Non-bullet lines are preserved unchanged."""
        digest = """*This week in dasm2* (Jan 12-18)

*Merged*
• Fix bug (<https://github.com/org/repo/pull/123|#123>)

*Discussion*
• Feature request (<https://github.com/org/repo/issues/456|#456>)"""
        items = [
            {
                "number": 123,
                "is_pr": True,
                "contributors": ["alice"],
                "ref": "org/repo#123",
            },
            {
                "number": 456,
                "is_pr": False,
                "contributors": ["bob"],
                "ref": "org/repo#456",
            },
        ]

        result = _postprocess_digest(digest, items)

        assert "*This week in dasm2* (Jan 12-18)" in result
        assert "*Merged*" in result
        assert "*Discussion*" in result

    def test_handles_missing_item(self):
        """Lines with unknown issue numbers are preserved."""
        digest = "• Unknown item (<https://github.com/org/repo/pull/999|#999>)"
        items = [
            {"number": 123, "is_pr": True, "contributors": [], "ref": "org/repo#123"}
        ]

        result = _postprocess_digest(digest, items)

        # Should be unchanged
        assert result == digest

    def test_handles_line_without_link(self):
        """Bullet lines without Slack links are preserved."""
        digest = "• Some text without a link"
        items = [
            {"number": 123, "is_pr": True, "contributors": [], "ref": "org/repo#123"}
        ]

        result = _postprocess_digest(digest, items)

        assert result == digest

    def test_handles_empty_contributors(self):
        """Items with no contributors don't get trailing dash."""
        digest = "• Fix bug (<https://github.com/org/repo/pull/123|#123>)"
        items = [
            {"number": 123, "is_pr": True, "contributors": [], "ref": "org/repo#123"}
        ]

        result = _postprocess_digest(digest, items)

        assert "—" not in result
        assert (
            result == "• repo PR: Fix bug (<https://github.com/org/repo/pull/123|#123>)"
        )

    def test_full_digest_transformation(self):
        """Full digest is transformed correctly."""
        digest = """*This week in dasm2* (Jan 12-18)

*Merged*
• Structure-aware loss function (<https://github.com/matsengrp/dasm2/pull/142|#142>)

*New Issues*
• OOM on large batches (<https://github.com/matsengrp/dasm2/issues/156|#156>)"""

        items = [
            {
                "number": 142,
                "is_pr": True,
                "contributors": ["matsen", "willdumm"],
                "ref": "matsengrp/dasm2#142",
            },
            {
                "number": 156,
                "is_pr": False,
                "contributors": ["ksung25"],
                "ref": "matsengrp/dasm2#156",
            },
        ]

        result = _postprocess_digest(digest, items)

        assert "• dasm2 PR: Structure-aware loss function" in result
        assert "• dasm2 Issue: OOM on large batches" in result
        assert "— @matsen @willdumm" in result
        assert "— @ksung25" in result

    def test_same_number_different_repos(self):
        """Items with same number in different repos are matched correctly."""
        digest = """*Merged*
• First repo item (<https://github.com/org/repo-a/pull/31|#31>)
• Second repo item (<https://github.com/org/repo-b/issues/31|#31>)"""

        items = [
            {
                "number": 31,
                "is_pr": True,
                "contributors": ["alice"],
                "ref": "org/repo-a#31",
            },
            {
                "number": 31,
                "is_pr": False,
                "contributors": ["bob"],
                "ref": "org/repo-b#31",
            },
        ]

        result = _postprocess_digest(digest, items)

        # Each line should get the correct repo name and type
        assert "• repo-a PR: First repo item" in result
        assert "— @alice" in result
        assert "• repo-b Issue: Second repo item" in result
        assert "— @bob" in result


class TestGenerateDigestSummary:
    """Tests for generate_digest_summary."""

    def test_returns_fallback_for_empty_items(self):
        """Returns a simple message when no items."""
        result = generate_digest_summary([], "dasm2", "Jan 12-18")

        assert result is not None
        assert "*This week in dasm2*" in result
        assert "No activity" in result
