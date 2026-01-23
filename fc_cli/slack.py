"""Slack webhook integration for fc_cli."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

# Try to load .env file if python-dotenv is available
try:
    from dotenv import load_dotenv

    load_dotenv(Path(__file__).parent.parent / ".env")
except ImportError:
    pass


def get_webhook_url(channel: str) -> str | None:
    """Get Slack webhook URL for a channel from environment.

    Looks for SLACK_WEBHOOK_<CHANNEL> environment variable.

    Args:
        channel: Channel name (e.g., 'dasm2', 'test').

    Returns:
        Webhook URL or None if not configured.
    """
    env_var = f"SLACK_WEBHOOK_{channel.upper()}"
    return os.environ.get(env_var)


def post_to_slack(webhook_url: str, message: str) -> bool:
    """Post a message to Slack via webhook.

    Args:
        webhook_url: The Slack webhook URL.
        message: The message text (supports Slack mrkdwn formatting).

    Returns:
        True if successful, False otherwise.
    """
    payload = {"text": message}
    data = json.dumps(payload).encode("utf-8")

    req = urllib.request.Request(
        webhook_url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return response.status == 200
    except urllib.error.HTTPError as e:
        print(f"Slack API error: {e.code} {e.reason}", file=sys.stderr)
        return False
    except urllib.error.URLError as e:
        print(f"Network error posting to Slack: {e.reason}", file=sys.stderr)
        return False
    except Exception as e:
        print(f"Error posting to Slack: {e}", file=sys.stderr)
        return False


def send_digest(channel: str, message: str) -> bool:
    """Send a digest message to a Slack channel.

    Args:
        channel: Channel name (must have SLACK_WEBHOOK_<CHANNEL> configured).
        message: The digest message to send.

    Returns:
        True if successful, False otherwise.
    """
    webhook_url = get_webhook_url(channel)
    if not webhook_url:
        print(
            f"Error: No webhook configured for channel '{channel}'.\n"
            f"Set SLACK_WEBHOOK_{channel.upper()} in .env file.",
            file=sys.stderr,
        )
        return False

    return post_to_slack(webhook_url, message)
