"""Board-to-beads sync functionality."""

from __future__ import annotations

import re

from flowc.board.api import add_issue_to_board, list_board_items
from flowc.shared.config import load_beads

GITHUB_REF_PATTERN = re.compile(r"GitHub:\s*([^#\s]+)#(\d+)")


def get_p0_beads_with_github_refs() -> list[dict]:
    """Get all P0 beads that have GitHub issue references.

    Returns list of dicts with keys: bead_id, title, repo, issue_number.
    """
    beads = load_beads()
    p0_beads = []

    for bead in beads:
        # Check if P0 (priority stored as integer field)
        if bead.get("priority") != 0:
            continue

        # Extract GitHub reference from description
        desc = bead.get("description", "")
        github_match = GITHUB_REF_PATTERN.search(desc)
        if not github_match:
            continue

        p0_beads.append(
            {
                "bead_id": bead.get("id", ""),
                "title": bead.get("title", ""),
                "repo": github_match.group(1),
                "issue_number": int(github_match.group(2)),
            }
        )

    return p0_beads


def get_board_issues(board_key: str) -> list[dict]:
    """Get all issues on a board.

    Returns list of dicts with keys: item_id, title, repo, issue_number, status.
    """
    items = list_board_items(board_key)
    issues = []

    for item in items:
        content = item.get("content", {})
        if content.get("type") != "Issue":
            continue

        repo = content.get("repository", "")
        number = content.get("number")
        if not repo or not number:
            continue

        issues.append(
            {
                "item_id": item.get("id"),
                "title": item.get("title", ""),
                "repo": repo,
                "issue_number": number,
                "status": item.get("status", ""),
            }
        )

    return issues


def sync_board_with_beads(board_key: str, fix: bool = False) -> dict:
    """Compare P0 beads with board items and report/fix mismatches.

    Args:
        board_key: Board to sync (e.g., 'matsengrp/30').
        fix: If True, automatically fix mismatches.

    Returns:
        Dict with keys: missing_from_board, not_in_beads, fixes_applied.
    """
    p0_beads = get_p0_beads_with_github_refs()
    board_issues = get_board_issues(board_key)

    # Build sets for comparison
    p0_refs = {(b["repo"], b["issue_number"]) for b in p0_beads}
    board_refs = {(i["repo"], i["issue_number"]) for i in board_issues}

    # P0 beads not on board
    missing_from_board = []
    for bead in p0_beads:
        ref = (bead["repo"], bead["issue_number"])
        if ref not in board_refs:
            missing_from_board.append(bead)

    # Board issues not in P0 beads
    not_in_beads = []
    for issue in board_issues:
        ref = (issue["repo"], issue["issue_number"])
        if ref not in p0_refs:
            not_in_beads.append(issue)

    fixes_applied = []

    if fix:
        # Add missing P0s to board
        for bead in missing_from_board:
            success, msg = add_issue_to_board(
                board_key,
                bead["issue_number"],
                repo=bead["repo"],
                status="next",  # Default to Next column
            )
            if success:
                fixes_applied.append(f"Added #{bead['issue_number']} to board")
            else:
                fixes_applied.append(f"Failed to add #{bead['issue_number']}: {msg}")

        # Remove non-P0s from board (optional, commented out for safety)
        # for issue in not_in_beads:
        #     success, msg = remove_issue_from_board(
        #         board_key, issue["issue_number"], repo=issue["repo"]
        #     )
        #     if success:
        #         fixes_applied.append(f"Removed #{issue['issue_number']} from board")

    return {
        "missing_from_board": missing_from_board,
        "not_in_beads": not_in_beads,
        "fixes_applied": fixes_applied,
    }


def print_sync_report(result: dict, board_key: str):
    """Print sync report to stdout."""
    missing = result["missing_from_board"]
    not_in_beads = result["not_in_beads"]
    fixes = result.get("fixes_applied", [])

    print(f"## Board Sync: {board_key}\n")

    if missing:
        print(f"**P0 beads not on board** ({len(missing)}):")
        for bead in missing:
            print(f"  - {bead['repo']}#{bead['issue_number']}: {bead['title'][:50]}")
            print(f"    Bead: {bead['bead_id']}")
        print()

    if not_in_beads:
        print(f"**Board issues without P0 bead** ({len(not_in_beads)}):")
        for issue in not_in_beads:
            print(f"  - {issue['repo']}#{issue['issue_number']}: {issue['title'][:50]}")
            print(f"    Status: {issue['status']}")
        print()

    if fixes:
        print("**Fixes applied:**")
        for fix in fixes:
            print(f"  - {fix}")
        print()

    if not missing and not not_in_beads:
        print("Board and P0 beads are in sync!")
