"""GitHub API wrapper using gh CLI."""

from __future__ import annotations

import json
import subprocess
import sys
from datetime import datetime


def gh_api(endpoint: str) -> list | dict:
    """Call GitHub API via gh CLI.

    Args:
        endpoint: GitHub API endpoint path.

    Returns:
        Parsed JSON response (list or dict), or empty list on error.
    """
    result = subprocess.run(
        ["gh", "api", endpoint, "--paginate"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(
            f"  Error calling 'gh api {endpoint}': {result.stderr.strip()}",
            file=sys.stderr,
        )
        return []

    output = result.stdout.strip()
    if not output:
        return []

    try:
        return json.loads(output)
    except json.JSONDecodeError as e:
        items = []
        for line_num, line in enumerate(output.split("\n"), 1):
            if line.strip():
                try:
                    parsed = json.loads(line)
                    if isinstance(parsed, list):
                        items.extend(parsed)
                    else:
                        items.append(parsed)
                except json.JSONDecodeError:
                    print(
                        f"Warning: Failed to parse line {line_num} from 'gh api {endpoint}'",
                        file=sys.stderr,
                    )
        if not items:
            print(
                f"Error: Could not parse JSON from 'gh api {endpoint}': {e}",
                file=sys.stderr,
            )
        return items


def gh_graphql(query: str, variables: dict | None = None) -> dict:
    """Execute a GraphQL query via gh CLI.

    Args:
        query: GraphQL query string.
        variables: Optional variables dict.

    Returns:
        Parsed JSON response, or empty dict on error.
    """
    cmd = ["gh", "api", "graphql", "-f", f"query={query}"]
    if variables:
        for key, value in variables.items():
            cmd.extend(["-f", f"{key}={value}"])

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"GraphQL error: {result.stderr.strip()}", file=sys.stderr)
        return {}

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return {}


def fetch_issues(repo: str, since: datetime) -> list[dict]:
    """Fetch issues updated since given time."""
    since_str = since.strftime("%Y-%m-%dT%H:%M:%SZ")
    endpoint = f"/repos/{repo}/issues?state=all&since={since_str}&sort=updated&direction=desc&per_page=100"
    return gh_api(endpoint)


def fetch_comments(repo: str, since: datetime) -> list[dict]:
    """Fetch issue comments since given time."""
    since_str = since.strftime("%Y-%m-%dT%H:%M:%SZ")
    endpoint = f"/repos/{repo}/issues/comments?since={since_str}&sort=updated&direction=desc&per_page=100"
    return gh_api(endpoint)


def fetch_pr_comments(repo: str, since: datetime) -> list[dict]:
    """Fetch PR review comments since given time."""
    since_str = since.strftime("%Y-%m-%dT%H:%M:%SZ")
    endpoint = f"/repos/{repo}/pulls/comments?since={since_str}&sort=updated&direction=desc&per_page=100"
    return gh_api(endpoint)


def fetch_issue(repo: str, number: int) -> dict | None:
    """Fetch a single issue by number.

    Returns issue dict or None if not found.
    """
    result = gh_api(f"/repos/{repo}/issues/{number}")
    if isinstance(result, dict) and result.get("number"):
        return result
    return None


def get_issue_node_id(repo: str, number: int) -> str | None:
    """Get the GraphQL node ID for an issue.

    Returns node_id string or None if not found.
    """
    issue = fetch_issue(repo, number)
    if issue:
        return issue.get("node_id")
    return None


def fetch_item_comments(repo: str, number: int, limit: int = 10) -> list[dict]:
    """Fetch recent comments for an issue or PR.

    Args:
        repo: Repository in org/name format.
        number: Issue or PR number.
        limit: Maximum number of comments to fetch.

    Returns:
        List of comment dicts with 'author' and 'body' keys.
    """
    endpoint = f"/repos/{repo}/issues/{number}/comments?per_page={limit}"
    raw_comments = gh_api(endpoint)
    if not isinstance(raw_comments, list):
        return []

    comments = []
    for c in raw_comments[-limit:]:  # Take most recent
        comments.append(
            {
                "author": c.get("user", {}).get("login", "unknown"),
                "body": c.get("body", ""),
                "created_at": c.get("created_at", ""),
            }
        )
    return comments


def fetch_pr_reviewers(repo: str, number: int) -> list[str]:
    """Fetch reviewers for a PR.

    Args:
        repo: Repository in org/name format.
        number: PR number.

    Returns:
        List of reviewer login names.
    """
    endpoint = f"/repos/{repo}/pulls/{number}/reviews"
    reviews = gh_api(endpoint)
    if not isinstance(reviews, list):
        return []

    # Deduplicate reviewers (same person may leave multiple reviews)
    reviewers = set()
    for review in reviews:
        user = review.get("user", {})
        if user and user.get("login"):
            reviewers.add(user["login"])
    return list(reviewers)


def fetch_item_commenters(repo: str, number: int) -> list[str]:
    """Fetch commenters for an issue or PR.

    Args:
        repo: Repository in org/name format.
        number: Issue or PR number.

    Returns:
        List of commenter login names.
    """
    endpoint = f"/repos/{repo}/issues/{number}/comments"
    comments = gh_api(endpoint)
    if not isinstance(comments, list):
        return []

    commenters = set()
    for comment in comments:
        user = comment.get("user", {})
        if user and user.get("login"):
            commenters.add(user["login"])
    return list(commenters)


def fetch_item_details(
    repo: str, number: int, comments: list[dict] | None = None
) -> dict | None:
    """Fetch issue/PR details including recent comments.

    This is a reusable function for getting item context for LLM processing.

    Args:
        repo: Repository in org/name format.
        number: Issue or PR number.
        comments: Optional pre-fetched comments to avoid redundant API calls.
            If provided, these are used directly instead of fetching.

    Returns:
        Dict with item details, or None if not found:
        - ref: GitHub reference (e.g., "org/repo#123")
        - title: Issue/PR title
        - author: Author login
        - body: Issue/PR body
        - is_pr: Whether this is a PR
        - state: open/closed
        - comments: List of recent comments
    """
    issue = fetch_issue(repo, number)
    if not issue:
        return None

    if comments is None:
        comments = fetch_item_comments(repo, number)

    return {
        "ref": f"{repo}#{number}",
        "title": issue.get("title", ""),
        "author": issue.get("user", {}).get("login", "unknown"),
        "body": issue.get("body", ""),
        "is_pr": "pull_request" in issue,
        "state": issue.get("state", "unknown"),
        "comments": comments,
    }
