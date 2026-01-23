"""Board API with GraphQL fallback for managing GitHub project boards."""

from __future__ import annotations

import json
import subprocess
import sys
from typing import Any

from flowc.board.cache import (
    get_board_cache,
    get_cached_item_id,
    invalidate_cache,
    set_board_cache,
    set_cached_item_id,
)
from flowc.shared.github import get_issue_node_id, gh_graphql


def parse_board_key(board_key: str) -> tuple[str, str]:
    """Parse board key like 'matsengrp/30' into (owner, number)."""
    parts = board_key.split("/")
    if len(parts) != 2:
        raise ValueError(f"Invalid board key: {board_key} (expected owner/number)")
    return parts[0], parts[1]


def fetch_project_id(owner: str, project_num: str) -> str | None:
    """Fetch the GraphQL node ID for a project."""
    # Try org first
    query = """
    query($owner: String!, $number: Int!) {
      organization(login: $owner) {
        projectV2(number: $number) {
          id
        }
      }
    }
    """
    result = gh_graphql(query, {"owner": owner, "number": project_num})
    project_id = (
        result.get("data", {}).get("organization", {}).get("projectV2", {}).get("id")
    )
    if project_id:
        return project_id

    # Try user
    query = """
    query($owner: String!, $number: Int!) {
      user(login: $owner) {
        projectV2(number: $number) {
          id
        }
      }
    }
    """
    result = gh_graphql(query, {"owner": owner, "number": project_num})
    return result.get("data", {}).get("user", {}).get("projectV2", {}).get("id")


def fetch_project_fields(project_id: str) -> dict[str, Any]:
    """Fetch field IDs and option values for a project."""
    query = """
    query($projectId: ID!) {
      node(id: $projectId) {
        ... on ProjectV2 {
          fields(first: 20) {
            nodes {
              ... on ProjectV2Field {
                id
                name
              }
              ... on ProjectV2SingleSelectField {
                id
                name
                options {
                  id
                  name
                }
              }
            }
          }
        }
      }
    }
    """
    result = gh_graphql(query, {"projectId": project_id})
    fields_data = (
        result.get("data", {}).get("node", {}).get("fields", {}).get("nodes", [])
    )

    fields = {}
    status_options = {}

    for field in fields_data:
        field_name = field.get("name", "")
        field_id = field.get("id", "")

        if field_name and field_id:
            fields[field_name] = field_id

        if field_name == "Status" and "options" in field:
            for opt in field["options"]:
                status_options[opt["name"].lower()] = opt["id"]

    return {
        "field_ids": fields,
        "status_options": status_options,
    }


def ensure_board_metadata(
    board_key: str, force_refresh: bool = False
) -> dict[str, Any]:
    """Ensure board metadata is cached and return it."""
    if not force_refresh:
        cached = get_board_cache(board_key)
        if cached and "project_id" in cached:
            return cached

    owner, project_num = parse_board_key(board_key)

    project_id = fetch_project_id(owner, project_num)
    if not project_id:
        raise ValueError(f"Could not find project {board_key}")

    fields_data = fetch_project_fields(project_id)

    cache_data = {
        "project_id": project_id,
        "owner": owner,
        "project_num": project_num,
        **fields_data,
        "item_ids": {},
    }

    set_board_cache(board_key, cache_data)
    return cache_data


def list_board_items(board_key: str) -> list[dict]:
    """List all items on a board using gh CLI."""
    owner, project_num = parse_board_key(board_key)

    cmd = [
        "gh",
        "project",
        "item-list",
        project_num,
        "--owner",
        owner,
        "--format",
        "json",
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Error listing board items: {result.stderr.strip()}", file=sys.stderr)
        return []

    try:
        data = json.loads(result.stdout)
        return data.get("items", [])
    except json.JSONDecodeError:
        return []


def add_issue_to_board(
    board_key: str,
    issue_number: int,
    repo: str,
    status: str | None = None,
    label: str | None = None,
) -> tuple[bool, str]:
    """Add an issue to a board.

    Returns (success, message).
    """
    owner, project_num = parse_board_key(board_key)

    # Try gh project item-add first
    cmd = [
        "gh",
        "project",
        "item-add",
        project_num,
        "--owner",
        owner,
        "--url",
        f"https://github.com/{repo}/issues/{issue_number}",
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)

    item_id = None
    if result.returncode == 0 and result.stdout.strip():
        # gh project item-add returns the item ID on success
        item_id = result.stdout.strip()
        set_cached_item_id(board_key, issue_number, repo, item_id)
    else:
        # Fall back to GraphQL
        item_id = add_issue_via_graphql(board_key, issue_number, repo)
        if not item_id:
            return False, f"Failed to add issue #{issue_number} to board"

    # Apply status if specified
    if status:
        success, msg = set_item_status(board_key, item_id, status)
        if not success:
            return False, f"Added to board but failed to set status: {msg}"

    # TODO: Apply label if specified (requires label field lookup)

    return True, f"Added #{issue_number} to board" + (
        f" with status '{status}'" if status else ""
    )


def add_issue_via_graphql(board_key: str, issue_number: int, repo: str) -> str | None:
    """Add an issue to a board using GraphQL API."""
    metadata = ensure_board_metadata(board_key)
    project_id = metadata["project_id"]

    # Get the issue's node ID
    issue_node_id = get_issue_node_id(repo, issue_number)
    if not issue_node_id:
        print(f"Could not find issue #{issue_number} in {repo}", file=sys.stderr)
        return None

    mutation = """
    mutation($projectId: ID!, $contentId: ID!) {
      addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
        item {
          id
        }
      }
    }
    """
    result = gh_graphql(mutation, {"projectId": project_id, "contentId": issue_node_id})

    item_id = (
        result.get("data", {}).get("addProjectV2ItemById", {}).get("item", {}).get("id")
    )

    if item_id:
        set_cached_item_id(board_key, issue_number, repo, item_id)
        return item_id

    return None


def get_item_id(board_key: str, issue_number: int, repo: str) -> str | None:
    """Get the board item ID for an issue, fetching if necessary."""
    # Check cache first
    item_id = get_cached_item_id(board_key, issue_number, repo)
    if item_id:
        return item_id

    # Search board items
    items = list_board_items(board_key)
    for item in items:
        content = item.get("content", {})
        if content.get("repository") == repo and content.get("number") == issue_number:
            item_id = item.get("id")
            if item_id:
                set_cached_item_id(board_key, issue_number, repo, item_id)
                return item_id

    return None


def set_item_status(board_key: str, item_id: str, status: str) -> tuple[bool, str]:
    """Set the status of a board item.

    Returns (success, message).
    """
    metadata = ensure_board_metadata(board_key)
    project_id = metadata["project_id"]
    status_field_id = metadata.get("field_ids", {}).get("Status")
    status_options = metadata.get("status_options", {})

    if not status_field_id:
        return False, "Status field not found on board"

    status_lower = status.lower()
    option_id = status_options.get(status_lower)
    if not option_id:
        available = ", ".join(status_options.keys())
        return False, f"Unknown status '{status}'. Available: {available}"

    mutation = """
    mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
      updateProjectV2ItemFieldValue(input: {
        projectId: $projectId
        itemId: $itemId
        fieldId: $fieldId
        value: {singleSelectOptionId: $optionId}
      }) {
        projectV2Item {
          id
        }
      }
    }
    """
    result = gh_graphql(
        mutation,
        {
            "projectId": project_id,
            "itemId": item_id,
            "fieldId": status_field_id,
            "optionId": option_id,
        },
    )

    if result.get("data", {}).get("updateProjectV2ItemFieldValue"):
        return True, f"Status set to '{status}'"
    return False, "Failed to update status"


def move_item(
    board_key: str,
    issue_number: int,
    status: str,
    repo: str,
) -> tuple[bool, str]:
    """Move a board item to a new status.

    Returns (success, message).
    """
    item_id = get_item_id(board_key, issue_number, repo)
    if not item_id:
        return False, f"Issue #{issue_number} not found on board"

    return set_item_status(board_key, item_id, status)


def remove_issue_from_board(
    board_key: str,
    issue_number: int,
    repo: str,
) -> tuple[bool, str]:
    """Remove an issue from a board.

    Returns (success, message).
    """
    item_id = get_item_id(board_key, issue_number, repo)
    if not item_id:
        return False, f"Issue #{issue_number} not found on board"

    metadata = ensure_board_metadata(board_key)
    project_id = metadata["project_id"]

    mutation = """
    mutation($projectId: ID!, $itemId: ID!) {
      deleteProjectV2Item(input: {projectId: $projectId, itemId: $itemId}) {
        deletedItemId
      }
    }
    """
    result = gh_graphql(mutation, {"projectId": project_id, "itemId": item_id})

    if result.get("data", {}).get("deleteProjectV2Item", {}).get("deletedItemId"):
        # Remove from cache
        invalidate_cache(board_key)
        return True, f"Removed #{issue_number} from board"

    return False, f"Failed to remove #{issue_number} from board"


def refresh_board_cache(board_key: str) -> tuple[bool, str]:
    """Force refresh the board cache.

    Returns (success, message).
    """
    try:
        invalidate_cache(board_key)
        metadata = ensure_board_metadata(board_key, force_refresh=True)
        status_count = len(metadata.get("status_options", {}))
        return True, f"Cache refreshed. Found {status_count} status options."
    except Exception as e:
        return False, f"Failed to refresh cache: {e}"
