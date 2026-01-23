"""GitHub activity fetching and display."""

from __future__ import annotations

import re
from datetime import datetime, timezone

from fc_cli.shared.config import COMMENT_PREVIEW_LENGTH, DEFAULT_DISPLAY_LIMIT
from fc_cli.shared.github import fetch_comments, fetch_issues, fetch_pr_comments

# Compiled regex patterns
GITHUB_REF_PATTERN = re.compile(r"GitHub:\s*([^#\s]+#\d+)")
GITHUB_REF_PARTS_PATTERN = re.compile(r"GitHub:\s*([^#\s]+)#(\d+)")
BOARD_MARKER_PATTERN = re.compile(r"Board:\s*([^#\s]+)#(\S+)")
LINKED_ISSUE_PATTERN = re.compile(r"(?:Fixes|Closes|Resolves)\s*#(\d+)", re.IGNORECASE)


def parse_github_timestamp(timestamp: str) -> datetime:
    """Parse GitHub API timestamp (ISO 8601 with Z suffix)."""
    return datetime.fromisoformat(timestamp.replace("Z", "+00:00"))


def format_time_ago(dt: datetime) -> str:
    """Format datetime as relative time."""
    now = datetime.now(timezone.utc)
    delta = now - dt
    if delta.days > 0:
        return f"{delta.days}d ago"
    hours = delta.seconds // 3600
    if hours > 0:
        return f"{hours}h ago"
    minutes = delta.seconds // 60
    return f"{minutes}m ago"


def sanitize_repo_name(repo: str) -> str:
    """Extract and sanitize repo name for use in bead IDs."""
    repo_name = repo.split("/")[-1]
    return re.sub(r"[^a-zA-Z0-9_-]", "-", repo_name)


def extract_github_refs_from_description(desc: str) -> list[str]:
    """Extract all GitHub issue references from a single description.

    Returns list of 'org/repo#num' strings.
    """
    return GITHUB_REF_PATTERN.findall(desc)


def collect_all_github_refs(beads: list[dict]) -> set[str]:
    """Build set of all GitHub references across all beads.

    Returns set of 'org/repo#num' strings.
    """
    refs = set()
    for bead in beads:
        desc = bead.get("description", "")
        refs.update(extract_github_refs_from_description(desc))
    return refs


def parse_board_marker(desc: str) -> tuple[str, str] | None:
    """Extract board info from description like 'Board: matsengrp/28#PVTI_xxx'.

    Returns (board_key, item_id) or None.
    """
    match = BOARD_MARKER_PATTERN.search(desc)
    if match:
        return match.group(1), match.group(2)
    return None


def extract_linked_issues(pr_body: str | None) -> list[int]:
    """Extract issue numbers linked in a PR body via 'Fixes/Closes/Resolves #N'.

    Args:
        pr_body: The body text of a pull request.

    Returns:
        List of issue numbers found in the body.
    """
    if not pr_body:
        return []
    return [int(m) for m in LINKED_ISSUE_PATTERN.findall(pr_body)]


def get_comment_item_number(comment: dict) -> int | None:
    """Extract the issue/PR number a comment belongs to.

    Handles both regular issue comments and PR review comments.
    """
    if "issue_url" in comment:
        return int(comment["issue_url"].split("/")[-1])
    if "pull_request_url" in comment:
        return int(comment["pull_request_url"].split("/")[-1])
    return None


def ball_in_my_court(item: dict, comments: list[dict], github_user: str) -> bool:
    """Determine if the user needs to act on this item.

    Truth table (tested in tests/test_activity.py):
        Their item, no comments       -> True  (need to review)
        Their item, they commented    -> True  (they pinged again)
        Their item, I commented last  -> False (waiting for their reply)
        My item, no comments          -> False (waiting for feedback)
        My item, they commented last  -> True  (they replied)
        My item, I commented last     -> False (waiting for their reply)

    Example scenarios (for user running `flowc checkin --since 3d` on Monday),
    also tested in tests/test_activity.py:

        1. Someone commented Saturday on an old issue you created months ago.
           -> Show: they replied to your item, ball is in your court.

        2. You added a comment Saturday to your own issue (adding context).
           -> Hide: you commented last, waiting for their reply.

        3. Someone opened a new PR Friday, no comments yet.
           -> Show: their item needs your review.

        4. You opened a PR Friday, no comments yet.
           -> Hide: waiting for feedback on your item.

        5. You reviewed someone's PR and left comments.
           -> Hide: you commented last, waiting for them to address.

        6. Someone replied to your review on their PR.
           -> Show: they responded, ball is in your court.
    """
    author = item["user"]["login"]
    is_my_item = author == github_user

    item_number = item["number"]
    item_comments = [c for c in comments if get_comment_item_number(c) == item_number]

    if not item_comments:
        # No comments: show their items (need review), hide mine (waiting for feedback)
        return not is_my_item

    # Has comments: show if last commenter is not me (they're waiting for my response)
    item_comments.sort(key=lambda c: c.get("updated_at", c.get("created_at", "")))
    last_commenter = item_comments[-1].get("user", {}).get("login", "")

    return last_commenter and last_commenter != github_user


def filter_by_ball_in_court(
    items: list[dict], comments: list[dict], github_user: str
) -> list[dict]:
    """Filter items to only those where ball is in user's court.

    Args:
        items: List of issues or PRs.
        comments: All comments for the repo (used to determine last commenter).
        github_user: Current user's GitHub login.

    Returns:
        Filtered list of items requiring user action.
    """
    return [item for item in items if ball_in_my_court(item, comments, github_user)]


def filter_comments_by_items(comments: list[dict], items: list[dict]) -> list[dict]:
    """Filter comments to only those belonging to the given items.

    Args:
        comments: All comments for the repo.
        items: List of issues/PRs to keep comments for.

    Returns:
        Comments that belong to items in the list.
    """
    item_numbers = {item["number"] for item in items}
    return [c for c in comments if get_comment_item_number(c) in item_numbers]


def fetch_all_activity(
    repos: list[str], since: datetime, github_user: str | None = None
) -> tuple[dict[str, dict], dict[str, int]]:
    """Fetch issues, PRs, and comments for all repos.

    Args:
        repos: List of repo names to fetch.
        since: Cutoff time for activity.
        github_user: If provided, filter to items where ball is in user's court.
            If None, show all activity (no filtering).

    Returns:
        Tuple of (activity_dict, counts_dict).
        activity_dict: {repo: {'issues': [...], 'prs': [...]}}
        counts_dict: {'issues': N, 'prs': M, 'comments': K}
    """
    activity: dict[str, dict] = {}
    total_issues = 0
    total_prs = 0
    total_comments = 0

    for repo in repos:
        all_items = fetch_issues(repo, since)
        issue_comments = fetch_comments(repo, since)
        pr_comments = fetch_pr_comments(repo, since)

        new_issues = [i for i in all_items if not i.get("pull_request")]
        new_prs = [i for i in all_items if i.get("pull_request")]
        all_comments = issue_comments + pr_comments

        # Apply ball-in-my-court filtering if github_user is provided
        if github_user:
            new_issues = filter_by_ball_in_court(new_issues, all_comments, github_user)
            new_prs = filter_by_ball_in_court(new_prs, all_comments, github_user)
            # Filter comments to only those on items we're showing
            all_items_shown = new_issues + new_prs
            all_comments = filter_comments_by_items(all_comments, all_items_shown)

        if new_issues or new_prs:
            activity[repo] = {
                "issues": new_issues,
                "prs": new_prs,
                "comments": all_comments,
            }

        if not new_issues and not new_prs and not all_comments:
            continue

        print(f"## {repo}")
        total_issues += len(new_issues)
        total_prs += len(new_prs)
        total_comments += len(all_comments)

        print_items(new_issues, "issue", since)
        print_items(new_prs, "pr", since)
        print_comments(all_comments)
        print()

    return activity, {
        "issues": total_issues,
        "prs": total_prs,
        "comments": total_comments,
    }


def print_items(items: list[dict], item_type: str, since: datetime):
    """Print formatted list of issues or PRs."""
    if not items:
        return

    type_label = "Pull Requests" if item_type == "pr" else "Issues"
    print(f"\n### {type_label} ({len(items)})")

    for item in items[:DEFAULT_DISPLAY_LIMIT]:
        updated = parse_github_timestamp(item["updated_at"])
        created = parse_github_timestamp(item["created_at"])
        is_new = created > since
        marker = "NEW" if is_new else "upd"
        print(f"  [{marker}] {item['html_url']} - {item['title']} ({format_time_ago(updated)})")

    if len(items) > DEFAULT_DISPLAY_LIMIT:
        print(f"  ... and {len(items) - DEFAULT_DISPLAY_LIMIT} more")


def print_comments(comments: list[dict]):
    """Print formatted comment activity grouped by item."""
    if not comments:
        return

    print(f"\n### Comments ({len(comments)})")

    by_item: dict[str, list] = {}
    for c in comments:
        if "issue_url" in c:
            item_num = c["issue_url"].split("/")[-1]
        elif "pull_request_url" in c:
            item_num = c["pull_request_url"].split("/")[-1]
        else:
            continue
        by_item.setdefault(item_num, []).append(c)

    for item_num, item_comments in list(by_item.items())[:DEFAULT_DISPLAY_LIMIT]:
        # Get the URL from the first comment to build the issue/PR link
        first_comment = item_comments[0]
        item_url = first_comment["html_url"].split("#")[0]  # Strip comment anchor
        print(f"  {item_url}: {len(item_comments)} new comment(s)")
        for c in item_comments[:3]:
            updated = parse_github_timestamp(c["updated_at"])
            user = c["user"]["login"]
            body_preview = (
                c["body"][:COMMENT_PREVIEW_LENGTH].replace("\n", " ")
                if c["body"]
                else ""
            )
            print(f"    @{user} ({format_time_ago(updated)}): {body_preview}...")
            print(f"      {c['html_url']}")

    if len(by_item) > DEFAULT_DISPLAY_LIMIT:
        print(
            f"  ... and {len(by_item) - DEFAULT_DISPLAY_LIMIT} more items with comments"
        )
