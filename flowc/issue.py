"""Spawn tmux window for GitHub issue or PR review."""

from __future__ import annotations

import json
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Literal

from flowc.checkin.spawn import (
    create_tmux_window,
    is_in_tmux,
    tmux_window_exists,
)
from flowc.shared.config import (
    MAX_COMMENTS_DISPLAY,
    extract_repo_name,
    get_repo_context_path,
    get_repo_local_path,
    load_config,
)

# Type alias for GitHub item types
GHItemType = Literal["issue", "pr"]

# Cache for current GitHub user
_github_user: str | None = None


def get_github_user() -> str | None:
    """Get the current GitHub user's login name."""
    global _github_user
    if _github_user is not None:
        return _github_user

    result = subprocess.run(
        ["gh", "api", "user", "--jq", ".login"],
        capture_output=True,
        text=True,
    )
    if result.returncode == 0:
        _github_user = result.stdout.strip()
        return _github_user
    return None


def parse_github_ref(arg: str) -> tuple[str, int, GHItemType | None] | None:
    """Parse GitHub reference (URL or org/repo#number).

    Returns (org/repo, number, type_hint) tuple or None if invalid.
    type_hint is "issue", "pr", or None (needs detection).

    Supported formats:
        - org/repo#123 (type unknown, needs detection)
        - https://github.com/org/repo/issues/123
        - https://github.com/org/repo/pull/123
        - github.com/org/repo/issues/123 (without scheme)

    Examples:
        >>> parse_github_ref("matsengrp/dasm2-experiments#166")
        ('matsengrp/dasm2-experiments', 166, None)
        >>> parse_github_ref("https://github.com/matsengrp/repo/pull/42")
        ('matsengrp/repo', 42, 'pr')
        >>> parse_github_ref("https://github.com/org/repo/issues/10")
        ('org/repo', 10, 'issue')
    """
    # Try URL format first
    # Matches: (https://)?(www.)?github.com/org/repo/(issues|pull)/number
    url_pattern = (
        r"^(?:https?://)?(?:www\.)?github\.com/([^/]+/[^/]+)/(issues|pull)/(\d+)/?$"
    )
    url_match = re.match(url_pattern, arg)
    if url_match:
        org_repo = url_match.group(1)
        item_type: GHItemType = "issue" if url_match.group(2) == "issues" else "pr"
        number = int(url_match.group(3))
        return (org_repo, number, item_type)

    # Try org/repo#number format
    if "#" not in arg:
        return None

    # Split on last # to handle repos like dnsm-experiments-1
    last_hash = arg.rfind("#")
    org_repo = arg[:last_hash]
    number_str = arg[last_hash + 1 :]

    if not org_repo or "/" not in org_repo:
        return None

    try:
        number = int(number_str)
        if number <= 0:
            return None
    except ValueError:
        return None

    return (org_repo, number, None)  # Type unknown, needs detection


def detect_item_type(org_repo: str, number: int) -> GHItemType | None:
    """Detect whether a GitHub number is an issue or PR.

    Uses gh api to check the issue endpoint - PRs have a 'pull_request' field.

    Returns "issue", "pr", or None if not found/error.
    """
    result = subprocess.run(
        ["gh", "api", f"repos/{org_repo}/issues/{number}", "--jq", ".pull_request"],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        return None

    # If pull_request field exists and is not null, it's a PR
    output = result.stdout.strip()
    if output and output != "null":
        return "pr"
    return "issue"


def fetch_issue(org_repo: str, number: int) -> dict | None:
    """Fetch issue details from GitHub.

    Returns dict with issue data or None on failure.
    """
    result = subprocess.run(
        [
            "gh",
            "issue",
            "view",
            str(number),
            "--repo",
            org_repo,
            "--json",
            "title,body,state,comments,labels,author,createdAt",
        ],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error fetching issue: {result.stderr.strip()}", file=sys.stderr)
        return None

    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        print("Error: Invalid JSON response from gh", file=sys.stderr)
        return None

    if not isinstance(data, dict):
        print("Error: Unexpected response format from gh", file=sys.stderr)
        return None

    return data


def fetch_pr(org_repo: str, number: int) -> dict | None:
    """Fetch PR details from GitHub.

    Returns dict with PR data or None on failure.
    """
    result = subprocess.run(
        [
            "gh",
            "pr",
            "view",
            str(number),
            "--repo",
            org_repo,
            "--json",
            "title,body,state,comments,labels,author,createdAt,files,reviews,additions,deletions,commits",
        ],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error fetching PR: {result.stderr.strip()}", file=sys.stderr)
        return None

    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        print("Error: Invalid JSON response from gh", file=sys.stderr)
        return None

    if not isinstance(data, dict):
        print("Error: Unexpected response format from gh", file=sys.stderr)
        return None

    return data


def format_relative_time(iso_timestamp: str) -> str:
    """Convert ISO timestamp to relative time (e.g., '2 days ago')."""
    dt = datetime.fromisoformat(iso_timestamp.replace("Z", "+00:00"))
    now = datetime.now(timezone.utc)
    delta = now - dt

    # Handle future timestamps (negative delta)
    if delta.total_seconds() < 0:
        return "in the future"

    days = delta.days
    hours = delta.seconds // 3600

    if days > 365:
        years = days // 365
        return f"{years} year{'s' if years != 1 else ''} ago"
    elif days > 30:
        months = days // 30
        return f"{months} month{'s' if months != 1 else ''} ago"
    elif days > 0:
        return f"{days} day{'s' if days != 1 else ''} ago"
    elif hours > 0:
        return f"{hours} hour{'s' if hours != 1 else ''} ago"
    else:
        minutes = delta.seconds // 60
        if minutes > 0:
            return f"{minutes} minute{'s' if minutes != 1 else ''} ago"
        return "just now"


def format_comments(comments: list[dict]) -> str:
    """Format comments for the prompt."""
    if not comments:
        return "(No comments)"

    # Show last N comments if exceeds limit
    limit = MAX_COMMENTS_DISPLAY
    display_comments = comments[-limit:] if len(comments) > limit else comments
    header = (
        f"({len(comments)} total, showing last {limit})"
        if len(comments) > limit
        else f"({len(comments)} total)"
    )

    formatted = [f"## Comments {header}"]
    for comment in display_comments:
        author = comment.get("author", {}).get("login", "unknown")
        created = comment.get("createdAt", "")
        relative_time = format_relative_time(created) if created else ""
        body = comment.get("body", "").strip()

        formatted.append(f"\n@{author} ({relative_time}):\n{body}")

    return "\n".join(formatted)


def build_issue_prompt(org_repo: str, number: int, issue: dict) -> str:
    """Build the review prompt for an issue."""
    title = issue.get("title", "Untitled")
    state = issue.get("state", "unknown")
    author = issue.get("author", {}).get("login", "unknown")
    created = issue.get("createdAt", "")
    body = issue.get("body", "").strip() or "(No description)"
    labels = issue.get("labels", [])
    comments = issue.get("comments", [])

    labels_str = ", ".join(label.get("name", "") for label in labels) or "(none)"
    relative_created = format_relative_time(created) if created else ""
    comments_section = format_comments(comments)

    has_comments = len(comments) > 0

    if has_comments:
        task_section = """Your task:
1. Read the issue and all comments carefully
2. Prepare the user to respond to the latest comment
3. If anything is unclear, explore the codebase to understand it
4. Summarize the discussion and suggest a response

Do NOT make changes, close, or comment on the issue. Analysis only."""
    else:
        task_section = """Your task:
1. Read the issue carefully
2. Summarize what the issue is asking for
3. If anything is unclear from the issue itself, explore the codebase to understand it

Do NOT make changes, close, or comment on the issue. Analysis only."""

    return f"""GitHub issue: {title}
Repository: {org_repo}
URL: https://github.com/{org_repo}/issues/{number}
State: {state}
Author: {author}
Labels: {labels_str}
Created: {relative_created}

## Issue Body
{body}

{comments_section}

---

{task_section}"""


def format_pr_files(files: list[dict]) -> str:
    """Format changed files for the prompt."""
    if not files:
        return "(No files changed)"

    # Limit to first 20 files for readability
    limit = 20
    display_files = files[:limit] if len(files) > limit else files
    header = (
        f"({len(files)} total, showing first {limit})"
        if len(files) > limit
        else f"({len(files)} files)"
    )

    formatted = [f"## Files Changed {header}"]
    for f in display_files:
        path = f.get("path", "unknown")
        additions = f.get("additions", 0)
        deletions = f.get("deletions", 0)
        formatted.append(f"  {path} (+{additions}/-{deletions})")

    return "\n".join(formatted)


def format_reviews(reviews: list[dict]) -> str:
    """Format PR reviews for the prompt."""
    if not reviews:
        return "(No reviews)"

    formatted = [f"## Reviews ({len(reviews)} total)"]
    for review in reviews:
        author = review.get("author", {}).get("login", "unknown")
        state = review.get("state", "unknown")
        body = review.get("body", "").strip()
        formatted.append(f"\n@{author}: {state}")
        if body:
            formatted.append(f"  {body[:200]}{'...' if len(body) > 200 else ''}")

    return "\n".join(formatted)


def user_has_engaged(
    comments: list[dict], reviews: list[dict], username: str | None
) -> bool:
    """Check if the user has already commented or reviewed on this PR."""
    if not username:
        return False

    # Check comments
    for comment in comments:
        if comment.get("author", {}).get("login") == username:
            return True

    # Check reviews
    for review in reviews:
        if review.get("author", {}).get("login") == username:
            return True

    return False


def build_pr_prompt(org_repo: str, number: int, pr: dict) -> str:
    """Build the review prompt for a PR."""
    title = pr.get("title", "Untitled")
    state = pr.get("state", "unknown")
    author = pr.get("author", {}).get("login", "unknown")
    created = pr.get("createdAt", "")
    body = pr.get("body", "").strip() or "(No description)"
    labels = pr.get("labels", [])
    comments = pr.get("comments", [])
    files = pr.get("files", [])
    reviews = pr.get("reviews", [])
    additions = pr.get("additions", 0)
    deletions = pr.get("deletions", 0)
    commits = pr.get("commits", [])

    labels_str = ", ".join(label.get("name", "") for label in labels) or "(none)"
    relative_created = format_relative_time(created) if created else ""
    comments_section = format_comments(comments)
    files_section = format_pr_files(files)
    reviews_section = format_reviews(reviews)

    # Check if user has already engaged with this PR
    current_user = get_github_user()
    already_engaged = user_has_engaged(comments, reviews, current_user)

    if already_engaged:
        task_section = """Your task:
1. Read the PR and all comments/reviews carefully
2. Prepare the user to respond to the latest activity
3. If anything is unclear, explore the codebase to understand it
4. Summarize the discussion and suggest a response

Do NOT approve, merge, comment, or make changes. Analysis only."""
    else:
        task_section = """Your task:
1. Check @CLAUDE.md in this repo for PR review guidelines and follow them
2. If no guidelines exist, review the PR for correctness, style, and potential issues
3. Summarize what the PR does and any concerns
4. Prepare a review for the user

Do NOT approve, merge, comment, or make changes. Analysis only."""

    return f"""GitHub PR: {title}
Repository: {org_repo}
URL: https://github.com/{org_repo}/pull/{number}
State: {state}
Author: {author}
Labels: {labels_str}
Created: {relative_created}
Stats: +{additions}/-{deletions} in {len(commits)} commit(s)

## PR Description
{body}

{files_section}

{reviews_section}

{comments_section}

---

{task_section}"""


def validate_repo_environment(org_repo: str) -> tuple[Path, str] | int:
    """Validate repo exists in sources.json and has a local clone.

    Returns:
        (repo_path, repo_name) tuple on success, or exit code (int) on failure.
    """
    repo_path = get_repo_local_path(org_repo)
    if repo_path is None:
        print(f"Error: Repo {org_repo} not found in sources.json", file=sys.stderr)
        print(
            "Add it to sources.json under 'code' or 'writing' category", file=sys.stderr
        )
        return 1

    repo_name = extract_repo_name(org_repo)
    if not repo_path.exists():
        print(f"Error: Local clone not found at {repo_path}", file=sys.stderr)
        code_path = load_config().get("paths", {}).get("code", "~/re")
        clone_url = f"git@github.com:{org_repo}.git"
        print(
            f"Clone it with: git clone {clone_url} {code_path}/{repo_name}",
            file=sys.stderr,
        )
        return 1

    return (repo_path, repo_name)


def load_repo_context(org_repo: str) -> str | None:
    """Load project context for a repository if available.

    Args:
        org_repo: Repository in org/repo format.

    Returns:
        Project context content or None if not found.
    """
    context_path = get_repo_context_path(org_repo)
    if not context_path or not context_path.exists():
        return None

    try:
        return context_path.read_text()
    except (OSError, UnicodeDecodeError) as e:
        print(
            f"Warning: Could not read context file {context_path}: {e}", file=sys.stderr
        )
        return None


def get_github_url(org_repo: str, number: int, item_type: GHItemType) -> str:
    """Construct GitHub URL for an issue or PR.

    Args:
        org_repo: Repository in org/repo format.
        number: Issue or PR number.
        item_type: "issue" or "pr".

    Returns:
        Full GitHub URL.

    Raises:
        ValueError: If item_type is invalid.
    """
    if item_type == "pr":
        resource = "pull"
    elif item_type == "issue":
        resource = "issues"
    else:
        raise ValueError(f"Invalid item_type: {item_type}. Must be 'pr' or 'issue'")

    return f"https://github.com/{org_repo}/{resource}/{number}"


def prepend_context(content: str, context: str | None) -> str:
    """Prepend project context to content if available.

    Args:
        content: The main content (prompt text).
        context: Optional project context to prepend.

    Returns:
        Content with context prepended, or original content if no context.
    """
    if not context:
        return content
    return f"""## Project Context

{context}

---

{content}"""


def build_custom_prompt(
    org_repo: str,
    number: int,
    item_type: GHItemType,
    custom_prompt: str,
) -> str:
    """Build a custom prompt without context.

    Args:
        org_repo: Repository in org/repo format.
        number: Issue or PR number.
        item_type: "issue" or "pr".
        custom_prompt: User-provided prompt text.

    Returns:
        Custom prompt without context (context should be prepended by caller).
    """
    url = get_github_url(org_repo, number, item_type)
    item_label = "PR" if item_type == "pr" else "Issue"

    return f"""GitHub {item_label}: {org_repo}#{number}
URL: {url}

{custom_prompt}"""


def build_default_prompt(
    org_repo: str,
    number: int,
    item_type: GHItemType,
    data: dict,
) -> str:
    """Build default prompt for an issue or PR without context.

    Args:
        org_repo: Repository in org/repo format.
        number: Issue or PR number.
        item_type: "issue" or "pr".
        data: Fetched issue or PR data.

    Returns:
        Default prompt text without context (context should be prepended by caller).
    """
    if item_type == "pr":
        return build_pr_prompt(org_repo, number, data)
    return build_issue_prompt(org_repo, number, data)


def spawn_review_window(
    org_repo: str,
    repo_name: str,
    number: int,
    data: dict,
    repo_path: Path,
    item_type: GHItemType,
    custom_prompt: str | None = None,
) -> int:
    """Spawn tmux window for issue or PR review.

    Args:
        org_repo: Repository in org/repo format.
        repo_name: Short repository name for window naming.
        number: Issue or PR number.
        data: Fetched issue or PR data.
        repo_path: Local path to repo clone.
        item_type: "issue" or "pr".
        custom_prompt: Optional custom prompt to use instead of default.

    Returns:
        0 on success, 1 on failure.
    """
    window_name = f"{repo_name}#{number}"

    if tmux_window_exists(window_name):
        print(f"Window {window_name} already exists, skipping")
        return 0

    # Build prompt (with or without custom prompt)
    if custom_prompt:
        base_prompt = build_custom_prompt(org_repo, number, item_type, custom_prompt)
    else:
        base_prompt = build_default_prompt(org_repo, number, item_type, data)

    # Add project context if available
    context = load_repo_context(org_repo)
    prompt = prepend_context(base_prompt, context)

    # Create tmux window
    url = get_github_url(org_repo, number, item_type)
    success = create_tmux_window(window_name, repo_path, prompt, url)
    return 0 if success else 1


def run_spawn(args) -> int:
    """Run the spawn command."""
    parsed = parse_github_ref(args.ref)
    if parsed is None:
        print(
            "Error: Invalid format. Expected org/repo#number or GitHub URL",
            file=sys.stderr,
        )
        return 1

    org_repo, number, type_hint = parsed

    # Validate environment
    validation = validate_repo_environment(org_repo)
    if isinstance(validation, int):
        return validation
    repo_path, repo_name = validation

    # Check tmux
    if not is_in_tmux():
        print("Error: Must be running inside tmux", file=sys.stderr)
        return 1

    # Determine item type if not known from URL
    if type_hint is None:
        print(f"Detecting type for {org_repo}#{number}...", file=sys.stderr)
        item_type = detect_item_type(org_repo, number)
        if item_type is None:
            print(f"Error: Could not find issue or PR #{number}", file=sys.stderr)
            return 1
        print(f"  → {item_type}", file=sys.stderr)
    else:
        item_type = type_hint

    # Fetch data based on type
    if item_type == "pr":
        data = fetch_pr(org_repo, number)
    else:
        data = fetch_issue(org_repo, number)

    if data is None:
        return 1

    custom_prompt = getattr(args, "prompt", None)
    return spawn_review_window(
        org_repo, repo_name, number, data, repo_path, item_type, custom_prompt
    )


# Keep old function name for backwards compatibility
def run_issue(args) -> int:
    """Run the issue command (deprecated, use run_spawn)."""
    print(
        "Warning: 'fc issue' is deprecated, use 'fc spawn' instead",
        file=sys.stderr,
    )
    # Adapt old args format to new format
    args.ref = args.issue_ref
    return run_spawn(args)
