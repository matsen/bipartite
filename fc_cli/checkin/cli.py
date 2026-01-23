"""CLI handler for checkin subcommand."""

from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone

from fc_cli.checkin.activity import (
    ball_in_my_court,
    fetch_all_activity,
    get_comment_item_number,
)
from fc_cli.checkin.board_check import check_boards, print_board_changes
from fc_cli.shared.config import (
    GITHUB_USER,
    SOURCES_FILE,
    load_beads,
    load_boards,
    load_repos,
    load_state,
    save_state,
)
from fc_cli.shared.github import fetch_item_details


def parse_duration(s: str) -> timedelta:
    """Parse duration like '2d', '12h', '1w'."""
    unit = s[-1]
    val = int(s[:-1])
    if unit == "d":
        return timedelta(days=val)
    elif unit == "h":
        return timedelta(hours=val)
    elif unit == "w":
        return timedelta(weeks=val)
    else:
        raise ValueError(f"Unknown duration unit: {unit}")


def parse_github_timestamp(timestamp: str) -> datetime:
    """Parse GitHub API timestamp (ISO 8601 with Z suffix)."""
    return datetime.fromisoformat(timestamp.replace("Z", "+00:00"))


def determine_since_time(args: argparse.Namespace, state: dict) -> datetime:
    """Determine the 'since' cutoff time from args or state."""
    if args.since:
        return datetime.now(timezone.utc) - parse_duration(args.since)
    elif state.get("last_checkin"):
        return parse_github_timestamp(state["last_checkin"])
    else:
        return datetime.now(timezone.utc) - timedelta(days=1)


def get_repos_to_check(args: argparse.Namespace) -> list[str]:
    """Get list of repos to check based on args."""
    import json

    if args.repo:
        return [args.repo]

    if args.category:
        with open(SOURCES_FILE) as f:
            data = json.load(f)
        return data.get(args.category, [])

    return load_repos()


def print_summary(
    total_issues: int,
    total_prs: int,
    total_comments: int,
    repo_count: int,
    board_changes: dict | None,
):
    """Print final summary line."""
    print("---")
    print(
        f"Total: {total_issues} issues, {total_prs} PRs, {total_comments} comments across {repo_count} repos"
    )

    if board_changes:
        total_new_drafts = sum(len(c["new_drafts"]) for c in board_changes.values())
        total_removed = sum(len(c["removed_drafts"]) for c in board_changes.values())
        total_orphans = sum(len(c["orphan_issues"]) for c in board_changes.values())
        if total_new_drafts or total_removed or total_orphans:
            print(
                f"Boards: {total_new_drafts} new drafts, {total_removed} removed, {total_orphans} orphan issues"
            )


def collect_items_for_summary(activity: dict[str, dict]) -> list[tuple[str, dict]]:
    """Collect all issues and PRs from activity for summarization.

    Args:
        activity: Activity dict from fetch_all_activity.

    Returns:
        List of (repo, item) tuples.
    """
    items = []
    for repo, data in activity.items():
        for issue in data.get("issues", []):
            items.append((repo, issue))
        for pr in data.get("prs", []):
            items.append((repo, pr))
    return items


def filter_comments_for_item(comments: list[dict], number: int) -> list[dict]:
    """Filter and transform activity comments for a specific item.

    Args:
        comments: Raw comments from activity (GitHub API format).
        number: Issue/PR number to filter for.

    Returns:
        List of comment dicts in the format expected by fetch_item_details.
    """
    filtered = []
    for c in comments:
        if get_comment_item_number(c) == number:
            filtered.append(
                {
                    "author": c.get("user", {}).get("login", "unknown"),
                    "body": c.get("body", ""),
                    "created_at": c.get("created_at", ""),
                }
            )
    return filtered


def generate_summaries(
    activity: dict[str, dict], github_user: str = GITHUB_USER
) -> dict[str, str]:
    """Generate take-home summaries for all items in activity.

    Args:
        activity: Activity dict from fetch_all_activity.
        github_user: Current user's GitHub login.

    Returns:
        Dict mapping "repo#number" -> summary string.
    """
    from fc_cli.llm import generate_take_home_summaries

    items_to_summarize = []
    items_list = collect_items_for_summary(activity)

    if not items_list:
        return {}

    print(f"\nFetching details for {len(items_list)} items...")

    for repo, item in items_list:
        number = item["number"]
        all_comments = activity[repo].get("comments", [])

        # Filter comments for this specific item (reuse already-fetched data)
        item_comments = filter_comments_for_item(all_comments, number)

        # Fetch issue details, passing pre-filtered comments to avoid redundant API call
        details = fetch_item_details(repo, number, comments=item_comments)
        if not details:
            continue

        # Add ball_in_my_court status (uses all_comments for correct logic)
        details["ball_in_my_court"] = ball_in_my_court(item, all_comments, github_user)
        items_to_summarize.append(details)

    if not items_to_summarize:
        return {}

    print(f"Generating summaries for {len(items_to_summarize)} items...")
    return generate_take_home_summaries(items_to_summarize)


def print_take_home_summaries(summaries: dict[str, str]):
    """Print take-home summaries in a readable format."""
    if not summaries:
        print("\nNo summaries generated.")
        return

    print("\n" + "=" * 60)
    print("TAKE-HOME SUMMARIES")
    print("=" * 60)

    for ref, summary in summaries.items():
        print(f"  {ref}: {summary}")


def run_checkin(args: argparse.Namespace):
    """Run the checkin command."""
    state = load_state()
    since = determine_since_time(args, state)
    print(f"Checking activity since {since.strftime('%Y-%m-%d %H:%M')} UTC\n")

    repos = get_repos_to_check(args)

    # Filter by ball-in-my-court unless --all is specified
    filter_user = None if args.all else GITHUB_USER
    activity, counts = fetch_all_activity(repos, since, github_user=filter_user)

    boards = load_boards()
    board_changes = {}
    if boards:
        beads = load_beads()
        board_changes = check_boards(boards, beads)
        print_board_changes(board_changes)

    print_summary(
        counts["issues"],
        counts["prs"],
        counts["comments"],
        len(repos),
        board_changes if boards else None,
    )

    # Generate and print take-home summaries if requested
    if getattr(args, "summarize", False) and activity:
        summaries = generate_summaries(activity)
        print_take_home_summaries(summaries)

    state["last_checkin"] = datetime.now(timezone.utc).isoformat()
    save_state(state)
    print("\nUpdated .last-checkin.json")
