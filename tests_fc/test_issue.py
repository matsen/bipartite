"""Tests for fc_cli/issue.py parsing and formatting functions.

Run with: pytest tests/test_issue.py -v
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

from fc_cli.issue import (
    format_comments,
    format_pr_files,
    format_relative_time,
    format_reviews,
    parse_github_ref,
)
from fc_cli.shared.config import extract_repo_name

# =============================================================================
# parse_github_ref Tests
# =============================================================================


def test_parse_github_ref_valid_hash_format():
    """Test parsing valid org/repo#number format."""
    assert parse_github_ref("matsengrp/dasm2-experiments#166") == (
        "matsengrp/dasm2-experiments",
        166,
        None,  # type unknown, needs detection
    )


def test_parse_github_ref_repo_with_hyphens_numbers():
    """Test parsing repos with hyphens and numbers in name."""
    assert parse_github_ref("org/repo-v2#42") == ("org/repo-v2", 42, None)
    assert parse_github_ref("matsengrp/dnsm-experiments-1#81") == (
        "matsengrp/dnsm-experiments-1",
        81,
        None,
    )


def test_parse_github_ref_no_hash():
    """Test that missing # returns None (unless URL)."""
    assert parse_github_ref("org/repo123") is None
    assert parse_github_ref("invalid") is None


def test_parse_github_ref_missing_org():
    """Test that missing org/ prefix returns None."""
    assert parse_github_ref("repo#123") is None
    assert parse_github_ref("#123") is None


def test_parse_github_ref_invalid_number():
    """Test that non-numeric or invalid numbers return None."""
    assert parse_github_ref("org/repo#abc") is None
    assert parse_github_ref("org/repo#") is None
    assert parse_github_ref("org/repo#0") is None
    assert parse_github_ref("org/repo#-5") is None


def test_parse_github_ref_uses_last_hash():
    """Test that parsing uses the last # (for repos with # in name somehow)."""
    # Edge case: if there were ever a # before the number
    result = parse_github_ref("org/my#repo#123")
    assert result == ("org/my#repo", 123, None)


# =============================================================================
# parse_github_ref URL Tests
# =============================================================================


def test_parse_github_ref_issue_url():
    """Test parsing GitHub issue URLs."""
    result = parse_github_ref("https://github.com/matsengrp/repo/issues/42")
    assert result == ("matsengrp/repo", 42, "issue")


def test_parse_github_ref_pr_url():
    """Test parsing GitHub PR URLs."""
    result = parse_github_ref("https://github.com/matsengrp/repo/pull/123")
    assert result == ("matsengrp/repo", 123, "pr")


def test_parse_github_ref_url_without_https():
    """Test parsing URLs without https:// prefix."""
    result = parse_github_ref("github.com/org/repo/issues/10")
    assert result == ("org/repo", 10, "issue")

    result = parse_github_ref("github.com/org/repo/pull/20")
    assert result == ("org/repo", 20, "pr")


def test_parse_github_ref_url_with_www():
    """Test parsing URLs with www. prefix."""
    result = parse_github_ref("https://www.github.com/org/repo/pull/5")
    assert result == ("org/repo", 5, "pr")


def test_parse_github_ref_url_with_trailing_slash():
    """Test parsing URLs with trailing slash."""
    result = parse_github_ref("https://github.com/org/repo/issues/99/")
    assert result == ("org/repo", 99, "issue")


def test_parse_github_ref_invalid_url():
    """Test that invalid URLs return None."""
    assert parse_github_ref("https://github.com/org/repo") is None
    assert parse_github_ref("https://github.com/org/repo/commits/abc") is None
    assert parse_github_ref("https://gitlab.com/org/repo/issues/1") is None


# =============================================================================
# extract_repo_name Tests
# =============================================================================


def test_extract_repo_name_basic():
    """Test basic repo name extraction."""
    assert extract_repo_name("matsengrp/dasm2-experiments") == "dasm2-experiments"
    assert extract_repo_name("org/repo") == "repo"


def test_extract_repo_name_with_special_chars():
    """Test repo names with hyphens and numbers."""
    assert extract_repo_name("matsengrp/dnsm-experiments-1") == "dnsm-experiments-1"
    assert extract_repo_name("org/repo-v2") == "repo-v2"


# =============================================================================
# format_relative_time Tests
# =============================================================================


def test_format_relative_time_just_now():
    """Test formatting times less than a minute ago."""
    now = datetime.now(timezone.utc)
    iso = now.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "just now"


def test_format_relative_time_minutes():
    """Test formatting times in minutes."""
    now = datetime.now(timezone.utc)
    five_min_ago = now - timedelta(minutes=5)
    iso = five_min_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "5 minutes ago"

    one_min_ago = now - timedelta(minutes=1)
    iso = one_min_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "1 minute ago"


def test_format_relative_time_hours():
    """Test formatting times in hours."""
    now = datetime.now(timezone.utc)
    two_hours_ago = now - timedelta(hours=2)
    iso = two_hours_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "2 hours ago"

    one_hour_ago = now - timedelta(hours=1)
    iso = one_hour_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "1 hour ago"


def test_format_relative_time_days():
    """Test formatting times in days."""
    now = datetime.now(timezone.utc)
    three_days_ago = now - timedelta(days=3)
    iso = three_days_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "3 days ago"

    one_day_ago = now - timedelta(days=1)
    iso = one_day_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "1 day ago"


def test_format_relative_time_months():
    """Test formatting times in months (approximate)."""
    now = datetime.now(timezone.utc)
    # 45 days = ~1.5 months, should show 1 month
    forty_five_days_ago = now - timedelta(days=45)
    iso = forty_five_days_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "1 month ago"

    # 90 days = 3 months
    ninety_days_ago = now - timedelta(days=90)
    iso = ninety_days_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "3 months ago"


def test_format_relative_time_years():
    """Test formatting times in years."""
    now = datetime.now(timezone.utc)
    # 400 days = ~1.1 years
    four_hundred_days_ago = now - timedelta(days=400)
    iso = four_hundred_days_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "1 year ago"

    # 800 days = ~2.2 years
    eight_hundred_days_ago = now - timedelta(days=800)
    iso = eight_hundred_days_ago.isoformat().replace("+00:00", "Z")
    assert format_relative_time(iso) == "2 years ago"


# =============================================================================
# format_comments Tests
# =============================================================================


def test_format_comments_empty():
    """Test formatting empty comments list."""
    assert format_comments([]) == "(No comments)"


def test_format_comments_single():
    """Test formatting a single comment."""
    comments = [
        {
            "author": {"login": "alice"},
            "createdAt": "2024-01-15T10:00:00Z",
            "body": "This is a comment",
        }
    ]
    result = format_comments(comments)
    assert "(1 total)" in result
    assert "@alice" in result
    assert "This is a comment" in result


def test_format_comments_multiple():
    """Test formatting multiple comments."""
    comments = [
        {
            "author": {"login": "alice"},
            "createdAt": "2024-01-15T10:00:00Z",
            "body": "First comment",
        },
        {
            "author": {"login": "bob"},
            "createdAt": "2024-01-16T10:00:00Z",
            "body": "Second comment",
        },
    ]
    result = format_comments(comments)
    assert "(2 total)" in result
    assert "@alice" in result
    assert "@bob" in result
    assert "First comment" in result
    assert "Second comment" in result


def test_format_comments_truncates_at_limit():
    """Test that comments are truncated to MAX_COMMENTS_DISPLAY."""
    # Create 15 comments with distinct names
    comments = [
        {
            "author": {"login": f"commenter_{i:02d}"},
            "createdAt": f"2024-01-{i:02d}T10:00:00Z",
            "body": f"Comment number {i}",
        }
        for i in range(1, 16)
    ]
    result = format_comments(comments)
    assert "(15 total, showing last 10)" in result
    # First 5 should NOT be included (comments 1-5)
    assert "@commenter_01" not in result
    assert "@commenter_05" not in result
    # Last 10 should be included (comments 6-15)
    assert "@commenter_06" in result
    assert "@commenter_15" in result


def test_format_comments_exactly_at_limit():
    """Test formatting exactly MAX_COMMENTS_DISPLAY comments."""
    comments = [
        {
            "author": {"login": f"user{i}"},
            "createdAt": f"2024-01-{i:02d}T10:00:00Z",
            "body": f"Comment {i}",
        }
        for i in range(1, 11)  # 10 comments
    ]
    result = format_comments(comments)
    assert "(10 total)" in result
    assert "showing last" not in result  # No truncation message


def test_format_comments_missing_fields():
    """Test formatting comments with missing optional fields."""
    comments = [
        {
            "author": {},  # Missing login
            "createdAt": "",  # Empty timestamp
            "body": "Body only",
        },
        {
            "body": "Minimal comment",
        },
    ]
    result = format_comments(comments)
    assert "@unknown" in result
    assert "Body only" in result
    assert "Minimal comment" in result


# =============================================================================
# format_pr_files Tests
# =============================================================================


def test_format_pr_files_empty():
    """Test formatting empty files list."""
    assert format_pr_files([]) == "(No files changed)"


def test_format_pr_files_single():
    """Test formatting a single changed file."""
    files = [{"path": "src/main.py", "additions": 10, "deletions": 5}]
    result = format_pr_files(files)
    assert "(1 files)" in result
    assert "src/main.py (+10/-5)" in result


def test_format_pr_files_multiple():
    """Test formatting multiple changed files."""
    files = [
        {"path": "src/a.py", "additions": 10, "deletions": 0},
        {"path": "src/b.py", "additions": 0, "deletions": 5},
        {"path": "tests/test_a.py", "additions": 20, "deletions": 10},
    ]
    result = format_pr_files(files)
    assert "(3 files)" in result
    assert "src/a.py (+10/-0)" in result
    assert "src/b.py (+0/-5)" in result
    assert "tests/test_a.py (+20/-10)" in result


def test_format_pr_files_truncates_at_limit():
    """Test that files list is truncated to 20 files."""
    files = [
        {"path": f"file_{i:02d}.py", "additions": i, "deletions": 0} for i in range(25)
    ]
    result = format_pr_files(files)
    assert "(25 total, showing first 20)" in result
    assert "file_00.py" in result
    assert "file_19.py" in result
    assert "file_20.py" not in result


# =============================================================================
# format_reviews Tests
# =============================================================================


def test_format_reviews_empty():
    """Test formatting empty reviews list."""
    assert format_reviews([]) == "(No reviews)"


def test_format_reviews_single():
    """Test formatting a single review."""
    reviews = [{"author": {"login": "reviewer1"}, "state": "APPROVED", "body": "LGTM"}]
    result = format_reviews(reviews)
    assert "(1 total)" in result
    assert "@reviewer1: APPROVED" in result
    assert "LGTM" in result


def test_format_reviews_multiple():
    """Test formatting multiple reviews."""
    reviews = [
        {
            "author": {"login": "alice"},
            "state": "CHANGES_REQUESTED",
            "body": "Please fix X",
        },
        {"author": {"login": "bob"}, "state": "APPROVED", "body": ""},
    ]
    result = format_reviews(reviews)
    assert "(2 total)" in result
    assert "@alice: CHANGES_REQUESTED" in result
    assert "Please fix X" in result
    assert "@bob: APPROVED" in result


def test_format_reviews_long_body_truncated():
    """Test that long review bodies are truncated."""
    reviews = [{"author": {"login": "user"}, "state": "COMMENTED", "body": "x" * 300}]
    result = format_reviews(reviews)
    assert "..." in result
    # Should show 200 chars + "..."
    assert "x" * 200 in result
