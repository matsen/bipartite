"""Cache management for board metadata."""

from __future__ import annotations

import json
import time
from typing import Any

from flowc.shared.config import CACHE_FILE, CACHE_TTL_HOURS


def load_cache() -> dict[str, Any]:
    """Load cache from file."""
    if not CACHE_FILE.exists():
        return {}
    try:
        with open(CACHE_FILE) as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError):
        return {}


def save_cache(cache: dict[str, Any]):
    """Save cache to file atomically."""
    temp_file = CACHE_FILE.with_suffix(".tmp")
    with open(temp_file, "w") as f:
        json.dump(cache, f, indent=2)
    temp_file.replace(CACHE_FILE)


def is_cache_valid(cache: dict[str, Any], board_key: str) -> bool:
    """Check if cache for a board is still valid (not expired)."""
    board_cache = cache.get("boards", {}).get(board_key, {})
    if not board_cache:
        return False

    cached_at = board_cache.get("cached_at", 0)
    max_age = CACHE_TTL_HOURS * 3600
    return (time.time() - cached_at) < max_age


def get_board_cache(board_key: str) -> dict[str, Any] | None:
    """Get cached data for a specific board, or None if not valid."""
    cache = load_cache()
    if is_cache_valid(cache, board_key):
        return cache.get("boards", {}).get(board_key)
    return None


def set_board_cache(board_key: str, data: dict[str, Any]):
    """Set cached data for a specific board."""
    cache = load_cache()
    if "boards" not in cache:
        cache["boards"] = {}

    data["cached_at"] = time.time()
    cache["boards"][board_key] = data
    save_cache(cache)


def get_cached_item_id(board_key: str, issue_number: int, repo: str) -> str | None:
    """Get cached item ID for an issue on a board."""
    board_cache = get_board_cache(board_key)
    if not board_cache:
        return None

    item_ids = board_cache.get("item_ids", {})
    key = f"{repo}#{issue_number}"
    return item_ids.get(key)


def set_cached_item_id(board_key: str, issue_number: int, repo: str, item_id: str):
    """Cache an item ID for an issue on a board."""
    cache = load_cache()
    if "boards" not in cache:
        cache["boards"] = {}
    if board_key not in cache["boards"]:
        cache["boards"][board_key] = {"cached_at": time.time()}

    if "item_ids" not in cache["boards"][board_key]:
        cache["boards"][board_key]["item_ids"] = {}

    key = f"{repo}#{issue_number}"
    cache["boards"][board_key]["item_ids"][key] = item_id
    save_cache(cache)


def invalidate_cache(board_key: str | None = None):
    """Invalidate cache for a specific board, or all boards if None."""
    if board_key is None:
        if CACHE_FILE.exists():
            CACHE_FILE.unlink()
    else:
        cache = load_cache()
        if "boards" in cache and board_key in cache["boards"]:
            del cache["boards"][board_key]
            save_cache(cache)
