"""Configuration and path management for flowc."""

from __future__ import annotations

import json
import sys
from pathlib import Path

# Paths relative to current working directory (nexus)
ROOT = Path.cwd()
SOURCES_FILE = ROOT / "sources.json"
BEADS_FILE = ROOT / "issues.jsonl"
STATE_FILE = ROOT / ".last-checkin.json"
CACHE_FILE = ROOT / ".flow-cache.json"

# Configuration
DEFAULT_DISPLAY_LIMIT = 10
COMMENT_PREVIEW_LENGTH = 80
CACHE_TTL_HOURS = 24
MAX_COMMENTS_DISPLAY = 10


def validate_nexus_directory():
    """Check that we're in a valid nexus directory.

    Exits with error if sources.json is not found in current directory.
    """
    if not SOURCES_FILE.exists():
        print(
            "Error: sources.json not found in current directory.\n"
            "Run flowc from your nexus directory (e.g., ~/re/nexus).",
            file=sys.stderr,
        )
        sys.exit(1)


def load_config() -> dict:
    """Load config.json from cwd.

    Returns:
        Config dict with paths and other settings.
        Returns defaults if config.json doesn't exist.
    """
    config_file = ROOT / "config.json"
    if config_file.exists():
        with open(config_file) as f:
            return json.load(f)
    return {"paths": {"code": "~/re", "writing": "~/writing"}}


def extract_repo_name(org_repo: str) -> str:
    """Extract repository name from org/repo string.

    Args:
        org_repo: Full repository reference like 'matsengrp/dasm2-experiments'

    Returns:
        Repository name like 'dasm2-experiments'

    Examples:
        >>> extract_repo_name("matsengrp/dasm2-experiments")
        'dasm2-experiments'
        >>> extract_repo_name("org/repo-v2")
        'repo-v2'
    """
    return org_repo.split("/")[-1]


def load_sources() -> dict:
    """Load the full sources.json data."""
    with open(SOURCES_FILE) as f:
        return json.load(f)


def normalize_repo_entry(entry: str | dict) -> str:
    """Extract repo name from a repo entry (string or object).

    Args:
        entry: Either a repo string or a dict with 'repo' key.

    Returns:
        The repo name string.
    """
    if isinstance(entry, dict):
        return entry["repo"]
    return entry


def load_repos() -> list[str]:
    """Load all repos from sources.json.

    Returns:
        List of repo names like 'matsengrp/dasm2-experiments'.

    Raises:
        ValueError: If sources.json is invalid or contains no repos.
    """
    try:
        data = load_sources()
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON in {SOURCES_FILE}: {e}")

    repos = []
    for category_name, category_data in data.items():
        if category_name in ("boards", "context"):
            continue
        if isinstance(category_data, list):
            for entry in category_data:
                repos.append(normalize_repo_entry(entry))

    if not repos:
        raise ValueError(f"No repos found in {SOURCES_FILE}")

    return repos


def load_repos_by_channel(channel: str) -> list[str]:
    """Load repos that belong to a specific channel.

    Args:
        channel: The channel name to filter by.

    Returns:
        List of repo names that have the specified channel.
    """
    data = load_sources()
    repos = []
    for category_name, category_data in data.items():
        if category_name in ("boards", "context"):
            continue
        if isinstance(category_data, list):
            for entry in category_data:
                if isinstance(entry, dict) and entry.get("channel") == channel:
                    repos.append(entry["repo"])
    return repos


def list_channels() -> list[str]:
    """List all unique channels defined in sources.json.

    Returns:
        Sorted list of channel names.
    """
    data = load_sources()
    channels = set()
    for category_name, category_data in data.items():
        if category_name in ("boards", "context"):
            continue
        if isinstance(category_data, list):
            for entry in category_data:
                if isinstance(entry, dict) and entry.get("channel"):
                    channels.add(entry["channel"])
    return sorted(channels)


def load_boards() -> dict[str, str]:
    """Load boards mapping from sources.json. Returns {org/num: bead_id}."""
    data = load_sources()
    return data.get("boards", {})


def get_default_board() -> str | None:
    """Get the first board from sources.json as the default."""
    boards = load_boards()
    if boards:
        return next(iter(boards.keys()))
    return None


def load_beads() -> list[dict]:
    """Load beads from JSONL file."""
    beads = []
    if BEADS_FILE.exists():
        with open(BEADS_FILE) as f:
            for line in f:
                if line.strip():
                    beads.append(json.loads(line))
    return beads


def load_state() -> dict:
    """Load last check-in state."""
    if STATE_FILE.exists():
        with open(STATE_FILE) as f:
            return json.load(f)
    return {}


def save_state(state: dict):
    """Save check-in state atomically."""
    temp_file = STATE_FILE.with_suffix(".tmp")
    with open(temp_file, "w") as f:
        json.dump(state, f, indent=2)
    temp_file.replace(STATE_FILE)


def _repo_in_category(repo: str, category_data: list) -> bool:
    """Check if repo is in category list (handles string and object entries)."""
    for entry in category_data:
        if normalize_repo_entry(entry) == repo:
            return True
    return False


def get_repo_local_path(repo: str) -> Path | None:
    """Map a GitHub repo (org/name) to local path based on sources.json category.

    Uses config.json paths for code and writing directories.

    Returns:
        Path to local clone, or None if not found in sources.json.
    """
    sources = load_sources()
    config = load_config()
    paths = config.get("paths", {})
    repo_name = extract_repo_name(repo)

    if _repo_in_category(repo, sources.get("writing", [])):
        writing_path = Path(paths.get("writing", "~/writing")).expanduser()
        return writing_path / repo_name
    elif _repo_in_category(repo, sources.get("code", [])):
        code_path = Path(paths.get("code", "~/re")).expanduser()
        return code_path / repo_name
    return None


def get_repo_context_path(repo: str) -> Path | None:
    """Get the context file path for a repo if one is defined.

    Args:
        repo: Full repository reference like 'matsengrp/dasm2-experiments'

    Returns:
        Absolute path to context file, or None if no context defined.
    """
    sources = load_sources()
    context_map = sources.get("context", {})
    relative_path = context_map.get(repo)
    if relative_path:
        return ROOT / relative_path
    return None
