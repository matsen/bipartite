"""Board checking for checkin (report board/beads sync status)."""

from __future__ import annotations

import json
import subprocess
import sys

from flowc.checkin.activity import (
    collect_all_github_refs,
    parse_board_marker,
)


def fetch_board_items(owner: str, project_num: str) -> list[dict]:
    """Fetch all items from a GitHub project board."""
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
        err = result.stderr.strip()
        print(
            f"  Error fetching board 'gh project item-list {project_num}': {err}",
            file=sys.stderr,
        )
        return []
    try:
        data = json.loads(result.stdout)
        return data.get("items", [])
    except json.JSONDecodeError:
        return []


def check_boards(boards: dict[str, str], beads: list[dict]) -> dict:
    """Compare board items with beads.

    Handles two types of board items:
    - DraftIssue: tracked via 'Board: org/num#itemID' markers
    - Issue: tracked via 'GitHub: org/repo#num' references (skips closed/Done)

    Returns dict with 'new_drafts', 'removed_drafts', 'orphan_issues' per board.
    """
    bead_by_board: dict[str, dict[str, dict]] = {}
    for bead in beads:
        desc = bead.get("description", "")
        marker = parse_board_marker(desc)
        if marker:
            board_key, item_id = marker
            if board_key not in bead_by_board:
                bead_by_board[board_key] = {}
            bead_by_board[board_key][item_id] = bead

    github_refs = collect_all_github_refs(beads)

    results = {}
    for board_key, parent_bead_id in boards.items():
        parts = board_key.split("/")
        if len(parts) != 2:
            continue
        owner, project_num = parts

        board_items = fetch_board_items(owner, project_num)

        draft_items = []
        issue_items = []
        for item in board_items:
            content = item.get("content", {})
            item_type = content.get("type", "")
            if item_type == "DraftIssue":
                draft_items.append(item)
            elif item_type == "Issue":
                issue_items.append(item)

        beads_for_board = bead_by_board.get(board_key, {})
        draft_item_ids = {item["id"]: item for item in draft_items}

        new_drafts = [
            item
            for item_id, item in draft_item_ids.items()
            if item_id not in beads_for_board
        ]

        removed_drafts = [
            bead
            for item_id, bead in beads_for_board.items()
            if item_id not in draft_item_ids
        ]

        orphan_issues = []
        for item in issue_items:
            status = item.get("status", "")
            if status.lower() == "done":
                continue

            content = item.get("content", {})
            repo = content.get("repository", "")
            number = content.get("number")
            if repo and number:
                ref = f"{repo}#{number}"
                if ref not in github_refs:
                    orphan_issues.append(item)

        if new_drafts or removed_drafts or orphan_issues:
            results[board_key] = {
                "parent_bead": parent_bead_id,
                "new_drafts": new_drafts,
                "removed_drafts": removed_drafts,
                "orphan_issues": orphan_issues,
            }

    return results


def print_board_changes(board_changes: dict):
    """Print board sync status."""
    if not board_changes:
        return

    print("## Project Boards\n")
    for board_key, changes in board_changes.items():
        parent = changes["parent_bead"]
        new_drafts = changes["new_drafts"]
        removed_drafts = changes["removed_drafts"]
        orphan_issues = changes["orphan_issues"]

        print(f"### {board_key} -> {parent}")

        if new_drafts:
            print(f"\n**New draft issues** ({len(new_drafts)}):")
            for item in new_drafts:
                print(f"  + {item.get('title', 'Untitled')}")
                print(f"    Board: {board_key}#{item['id']}")

        if removed_drafts:
            print(f"\n**Drafts removed from board** ({len(removed_drafts)}):")
            for bead in removed_drafts:
                print(f"  - {bead['id']}: {bead.get('title', 'Untitled')}")

        if orphan_issues:
            print(f"\n**Issues not in beads** ({len(orphan_issues)}):")
            for item in orphan_issues:
                content = item.get("content", {})
                repo = content.get("repository", "")
                number = content.get("number", "")
                url = content.get("url", "")
                print(f"  ! {item.get('title', 'Untitled')}")
                print(f"    GitHub: {repo}#{number}")
                if url:
                    print(f"    {url}")

        print()
