"""LLM infrastructure for fc_cli using claude CLI."""

from __future__ import annotations

import json
import re
import subprocess
import sys


def call_claude(prompt: str, model: str = "haiku") -> str | None:
    """Call claude CLI with the given prompt.

    Args:
        prompt: The prompt to send to the model.
        model: Model to use (haiku, sonnet, opus). Defaults to haiku.

    Returns:
        The model's response text, or None on error.
    """
    try:
        result = subprocess.run(
            ["claude", "--model", model, "-p", prompt],
            capture_output=True,
            text=True,
            timeout=120,  # 2 minute timeout
        )
        if result.returncode != 0:
            print(f"claude CLI error: {result.stderr.strip()}", file=sys.stderr)
            return None
        return result.stdout.strip()
    except FileNotFoundError:
        print("Error: 'claude' CLI not found. Install Claude Code.", file=sys.stderr)
        return None
    except subprocess.TimeoutExpired:
        print("Error: claude CLI timed out after 120s", file=sys.stderr)
        return None
    except Exception as e:
        print(f"Error calling claude CLI: {e}", file=sys.stderr)
        return None


def generate_take_home_summaries(items: list[dict]) -> dict[str, str]:
    """Generate take-home summaries for a batch of GitHub items.

    Args:
        items: List of item dicts, each containing:
            - ref: GitHub reference (e.g., "matsengrp/repo#123")
            - title: Issue/PR title
            - author: Author login
            - body: Issue/PR body (may be truncated)
            - comments: List of recent comments with author and body
            - is_pr: Whether this is a PR
            - ball_in_my_court: Whether action is expected from user

    Returns:
        Dict mapping ref -> take-home summary string.
    """
    if not items:
        return {}

    # Build the prompt
    prompt = _build_summary_prompt(items)

    # Call claude
    response = call_claude(prompt, model="haiku")
    if not response:
        return {}

    # Parse the response
    return _parse_summary_response(response, items)


def _build_summary_prompt(items: list[dict]) -> str:
    """Build the prompt for take-home summary generation."""
    items_text = []
    for item in items:
        item_type = "PR" if item.get("is_pr") else "Issue"
        ball_status = "needs_action" if item.get("ball_in_my_court") else "waiting"

        # Format comments
        comments_text = ""
        if item.get("comments"):
            comment_lines = []
            for c in item["comments"][-5:]:  # Last 5 comments
                author = c.get("author", "unknown")
                body = c.get("body", "")[:200]  # Truncate long comments
                comment_lines.append(f"    @{author}: {body}")
            comments_text = "\n".join(comment_lines)

        body_preview = (item.get("body") or "")[:300]

        items_text.append(f"""
---
REF: {item["ref"]}
TYPE: {item_type}
TITLE: {item["title"]}
AUTHOR: {item.get("author", "unknown")}
STATUS: {ball_status}
BODY: {body_preview}
RECENT_COMMENTS:
{comments_text}
---""")

    all_items = "\n".join(items_text)

    return f"""You are helping triage GitHub activity. For each item below, provide a brief take-home summary (1 short sentence) that tells the user what happened and whether they need to act.

Focus on:
- What's the current state/what happened?
- Does the user need to do anything?
- If waiting, what are they waiting for?

Examples of good summaries:
- "Will responded to your review - ready for re-review"
- "David acknowledged suggestion - no action needed"
- "Kevin asked about data format - decision needed"
- "New issue from Hugh about flu data - needs triage"
- "CI failed on your PR - needs fix"
- "Merged successfully - no action"

Output format: Return a JSON object mapping each REF to its summary.
Example: {{"matsengrp/repo#123": "summary here", "matsengrp/repo#456": "another summary"}}

Items to summarize:
{all_items}

Return ONLY the JSON object, no other text."""


def _parse_summary_response(response: str, items: list[dict]) -> dict[str, str]:
    """Parse the LLM response into a ref -> summary mapping."""
    # Try to extract JSON from the response
    try:
        # Handle case where response might have markdown code blocks
        text = response.strip()
        if text.startswith("```"):
            # Remove code block markers
            lines = text.split("\n")
            text = "\n".join(lines[1:-1] if lines[-1].strip() == "```" else lines[1:])

        return json.loads(text)
    except json.JSONDecodeError as e:
        print(
            f"Warning: Failed to parse LLM response as JSON: {e}",
            file=sys.stderr,
        )
        # Show truncated response for debugging
        preview = response[:200] + "..." if len(response) > 200 else response
        print(f"Response preview: {preview}", file=sys.stderr)
        return {}


def generate_digest_summary(
    items: list[dict], channel: str, date_range: str
) -> str | None:
    """Generate a digest summary for a channel's activity.

    Args:
        items: List of item dicts, each containing:
            - ref: GitHub reference (e.g., "matsengrp/repo#123")
            - title: Issue/PR title
            - author: Author login
            - is_pr: Whether this is a PR
            - state: open/closed/merged
            - html_url: URL to the item
            - contributors: List of contributor logins (optional)
        channel: Channel name for the header.
        date_range: Human-readable date range (e.g., "Jan 12-18").

    Returns:
        Formatted digest message for Slack, or None on error.
    """
    if not items:
        return f"*This week in {channel}* ({date_range})\n\nNo activity this period."

    prompt = _build_digest_prompt(items, channel, date_range)
    response = call_claude(prompt, model="haiku")

    if response:
        response = _postprocess_digest(response, items)

    return response


def _postprocess_digest(digest: str, items: list[dict]) -> str:
    """Add PR:/Issue: prefixes and @mentions to digest lines.

    Args:
        digest: Raw digest text from LLM.
        items: List of item dicts with number, is_pr, contributors, and ref.

    Returns:
        Digest with prefixes and @mentions added.
    """
    # Build lookup by ref (org/repo#number) for exact matching
    item_lookup: dict[str, dict] = {}
    for item in items:
        ref = item.get("ref", "")
        if ref:
            item_lookup[ref] = item

    # Pattern to extract repo and number from Slack link URL
    # e.g., <https://github.com/matsengrp/dasm-epistasis-experiments/pull/31|#31>
    url_pattern = re.compile(
        r"<https://github\.com/([^/]+/[^/]+)/(?:pull|issues)/(\d+)\|#\d+>"
    )

    lines = digest.split("\n")
    result_lines = []

    for line in lines:
        if not line.startswith("•"):
            result_lines.append(line)
            continue

        # Extract repo and number from URL in the line
        match = url_pattern.search(line)
        if not match:
            result_lines.append(line)
            continue

        repo_full = match.group(1)  # e.g., "matsengrp/dasm-epistasis-experiments"
        number = match.group(2)  # e.g., "31"
        ref = f"{repo_full}#{number}"

        item = item_lookup.get(ref)
        if not item:
            result_lines.append(line)
            continue

        # Extract repo name from ref (e.g., "matsengrp/mat-pcp#36" -> "mat-pcp")
        repo_name = repo_full.split("/")[-1] if "/" in repo_full else ""

        # Add repo and type prefix after bullet
        type_prefix = "PR:" if item.get("is_pr") else "Issue:"
        prefix = f"{repo_name} {type_prefix}" if repo_name else type_prefix
        line = line.replace("• ", f"• {prefix} ", 1)

        # Add contributors at the end
        contributors = item.get("contributors", [])
        if contributors:
            mentions = " ".join(f"@{c}" for c in contributors)
            line = f"{line} — {mentions}"

        result_lines.append(line)

    return "\n".join(result_lines)


def _build_digest_prompt(items: list[dict], channel: str, date_range: str) -> str:
    """Build the prompt for digest summary generation."""
    items_text = []
    for item in items:
        item_type = "PR" if item.get("is_pr") else "Issue"
        state = item.get("state", "open")
        if item.get("merged"):
            state = "merged"

        items_text.append(
            f"- [{item_type}] #{item['number']}: {item['title']} "
            f"(by @{item.get('author', 'unknown')}, {state}) "
            f"URL: {item.get('html_url', '')}"
        )

    all_items = "\n".join(items_text)

    return f"""You are writing a weekly digest for a team Slack channel. Summarize the following GitHub activity as a concise bullet-list message.

Channel: {channel}
Date range: {date_range}

Activity to summarize:
{all_items}

Format the output as a Slack message using mrkdwn:
- Start with: *This week in {channel}* ({date_range})
- Use bullet points (•) for each item
- Categorize by: Merged PRs, New issues, Active discussions
- Include Slack-style links: <URL|#number> or <URL|title>
- Keep it concise - one line per item
- Skip categories with no items

Example output:
*This week in dasm2* (Jan 12-18)

*Merged*
• Structure-aware loss function (<https://github.com/...|#142>)

*New Issues*
• OOM on large batches (<https://github.com/...|#156>)

*Discussion*
• Dataset versioning approach (<https://github.com/...|#148>)

Return ONLY the formatted Slack message, no other text."""
