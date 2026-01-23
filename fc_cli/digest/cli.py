"""CLI handler for digest subcommand."""

from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone

from fc_cli.llm import generate_digest_summary
from fc_cli.shared.config import list_channels, load_repos_by_channel
from fc_cli.shared.github import (
    fetch_issues,
    fetch_item_commenters,
    fetch_pr_reviewers,
)
from fc_cli.slack import get_webhook_url, send_digest


def parse_duration(s: str) -> timedelta:
    """Parse duration like '2d', '12h', '1w'."""
    if len(s) < 2:
        raise ValueError(f"Invalid duration format: {s}")
    unit = s[-1]
    try:
        val = int(s[:-1])
    except ValueError:
        raise ValueError(f"Invalid duration format: {s}")
    if unit == "d":
        return timedelta(days=val)
    elif unit == "h":
        return timedelta(hours=val)
    elif unit == "w":
        return timedelta(weeks=val)
    else:
        raise ValueError(f"Unknown duration unit: {unit}")


def format_date_range(since: datetime, until: datetime) -> str:
    """Format a date range for display (e.g., 'Jan 12-18')."""
    if since.month == until.month:
        return f"{since.strftime('%b')} {since.day}-{until.day}"
    return f"{since.strftime('%b %d')}-{until.strftime('%b %d')}"


def fetch_channel_activity(repos: list[str], since: datetime) -> list[dict]:
    """Fetch all issues and PRs for repos since the given time.

    Args:
        repos: List of repo names.
        since: Cutoff time for activity.

    Returns:
        List of item dicts with standardized fields including contributors.
    """
    items = []
    for repo in repos:
        all_items = fetch_issues(repo, since)
        for item in all_items:
            is_pr = "pull_request" in item
            number = item["number"]
            author = item.get("user", {}).get("login", "unknown")

            # Collect contributors: author + commenters + reviewers (for PRs)
            contributors = {author}
            commenters = fetch_item_commenters(repo, number)
            contributors.update(commenters)

            if is_pr:
                reviewers = fetch_pr_reviewers(repo, number)
                contributors.update(reviewers)

            # Sort alphabetically, remove 'unknown' if present
            contributors.discard("unknown")
            sorted_contributors = sorted(contributors, key=str.lower)

            items.append(
                {
                    "ref": f"{repo}#{item['number']}",
                    "number": number,
                    "title": item["title"],
                    "author": author,
                    "is_pr": is_pr,
                    "state": item.get("state", "open"),
                    "merged": is_pr and item.get("pull_request", {}).get("merged_at"),
                    "html_url": item.get("html_url", ""),
                    "created_at": item.get("created_at", ""),
                    "updated_at": item.get("updated_at", ""),
                    "contributors": sorted_contributors,
                }
            )
    return items


def run_digest(args: argparse.Namespace):
    """Run the digest command."""
    channel = args.channel
    post_to = getattr(args, "post_to", None) or channel
    repos_override = getattr(args, "repos", None)

    # Get repos to scan
    if repos_override:
        repos = [r.strip() for r in repos_override.split(",")]
    else:
        repos = load_repos_by_channel(channel)

    # Validate we have repos
    if not repos:
        available_channels = list_channels()
        if not available_channels:
            print("No channels configured in sources.json.")
            print("Add 'channel' field to repos in the 'code' section.")
            return 1
        print(f"No repos configured for channel '{channel}'.")
        print(f"Available channels: {', '.join(available_channels)}")
        print("Or use --repos to specify repos directly.")
        return 1

    # Check webhook is configured for destination
    webhook_url = get_webhook_url(post_to)
    if not webhook_url:
        print(f"No webhook configured for channel '{post_to}'.")
        print(f"Set SLACK_WEBHOOK_{post_to.upper()} in .env file.")
        return 1

    # Determine time range
    duration = parse_duration(args.since)
    until = datetime.now(timezone.utc)
    since = until - duration
    date_range = format_date_range(since, until)

    print(f"Generating digest for #{channel} ({date_range})...")
    if post_to != channel:
        print(f"(posting to #{post_to})")

    print(f"Scanning {len(repos)} repos...")
    items = fetch_channel_activity(repos, since)
    print(f"Found {len(items)} items")

    # Generate summary
    print("Generating summary...")
    message = generate_digest_summary(items, channel, date_range)

    if not message:
        print("Failed to generate summary")
        return 1

    # Print preview
    print("\n" + "=" * 60)
    print("DIGEST PREVIEW")
    print("=" * 60)
    print(message)
    print("=" * 60 + "\n")

    # Post to Slack
    print(f"Posting to #{post_to}...")
    if send_digest(post_to, message):
        print("Posted successfully!")
        return 0
    else:
        print("Failed to post to Slack")
        return 1
