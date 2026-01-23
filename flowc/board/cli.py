"""CLI handler for board subcommand."""

from __future__ import annotations

import argparse
import json
import sys

from flowc.board.api import (
    add_issue_to_board,
    list_board_items,
    move_item,
    refresh_board_cache,
    remove_issue_from_board,
)
from flowc.board.sync import print_sync_report, sync_board_with_beads
from flowc.shared.config import get_default_board


def get_board_key(args: argparse.Namespace) -> str:
    """Get board key from args or default."""
    if hasattr(args, "board") and args.board:
        return args.board

    default = get_default_board()
    if default:
        return default

    print(
        "Error: No board specified and no default board in sources.json",
        file=sys.stderr,
    )
    sys.exit(1)


def run_board(args: argparse.Namespace):
    """Run the board command."""
    if args.board_command == "list":
        handle_list(args)
    elif args.board_command == "add":
        handle_add(args)
    elif args.board_command == "move":
        handle_move(args)
    elif args.board_command == "remove":
        handle_remove(args)
    elif args.board_command == "sync":
        handle_sync(args)
    elif args.board_command == "refresh-cache":
        handle_refresh_cache(args)
    else:
        print(f"Unknown board command: {args.board_command}", file=sys.stderr)
        sys.exit(1)


def handle_list(args: argparse.Namespace):
    """Handle 'fc board list' command."""
    board_key = get_board_key(args)
    items = list_board_items(board_key)

    if not items:
        print(f"No items found on board {board_key}")
        return

    # Filter by status if specified
    if args.status:
        status_filter = args.status.lower()
        items = [i for i in items if i.get("status", "").lower() == status_filter]

    # Filter by label if specified
    if args.label:
        label_filter = args.label.lower()
        items = [
            i
            for i in items
            if any(label_filter in lbl.lower() for lbl in i.get("labels", []))
        ]

    if args.json:
        print(json.dumps(items, indent=2))
        return

    # Group by status
    by_status: dict[str, list] = {}
    for item in items:
        status = item.get("status", "No Status")
        by_status.setdefault(status, []).append(item)

    print(f"## Board: {board_key}\n")

    # Print in preferred order
    status_order = ["Blocked", "Next", "Active", "Done"]
    printed_statuses = set()

    for status in status_order:
        if status in by_status:
            print_status_group(status, by_status[status])
            printed_statuses.add(status)

    # Print any remaining statuses
    for status, status_items in by_status.items():
        if status not in printed_statuses:
            print_status_group(status, status_items)


def print_status_group(status: str, items: list[dict]):
    """Print a group of items for a status."""
    print(f"### {status} ({len(items)})")
    for item in items:
        content = item.get("content", {})
        item_type = content.get("type", "Unknown")
        number = content.get("number", "")
        repo = content.get("repository", "")
        title = item.get("title", "Untitled")

        if item_type == "Issue":
            print(f"  #{number}: {title}")
            if repo:
                repo_short = repo.split("/")[-1]
                print(
                    f"       [{repo_short}] https://github.com/{repo}/issues/{number}"
                )
        elif item_type == "DraftIssue":
            print(f"  [Draft] {title}")
        else:
            print(f"  {title}")

    print()


def handle_add(args: argparse.Namespace):
    """Handle 'fc board add' command."""
    board_key = get_board_key(args)
    if not args.repo:
        print("Error: --repo is required", file=sys.stderr)
        sys.exit(1)

    success, message = add_issue_to_board(
        board_key,
        args.issue,
        repo=args.repo,
        status=args.status,
        label=args.label,
    )

    if success:
        print(message)
    else:
        print(f"Error: {message}", file=sys.stderr)
        sys.exit(1)


def handle_move(args: argparse.Namespace):
    """Handle 'fc board move' command."""
    board_key = get_board_key(args)
    if not args.repo:
        print("Error: --repo is required", file=sys.stderr)
        sys.exit(1)

    success, message = move_item(
        board_key,
        args.issue,
        args.status,
        repo=args.repo,
    )

    if success:
        print(f"Moved #{args.issue} to '{args.status}'")
    else:
        print(f"Error: {message}", file=sys.stderr)
        sys.exit(1)


def handle_remove(args: argparse.Namespace):
    """Handle 'fc board remove' command."""
    board_key = get_board_key(args)
    if not args.repo:
        print("Error: --repo is required", file=sys.stderr)
        sys.exit(1)

    success, message = remove_issue_from_board(
        board_key,
        args.issue,
        repo=args.repo,
    )

    if success:
        print(message)
    else:
        print(f"Error: {message}", file=sys.stderr)
        sys.exit(1)


def handle_sync(args: argparse.Namespace):
    """Handle 'fc board sync' command."""
    board_key = get_board_key(args)

    result = sync_board_with_beads(board_key, fix=args.fix)
    print_sync_report(result, board_key)


def handle_refresh_cache(args: argparse.Namespace):
    """Handle 'fc board refresh-cache' command."""
    board_key = get_board_key(args)

    success, message = refresh_board_cache(board_key)
    if success:
        print(message)
    else:
        print(f"Error: {message}", file=sys.stderr)
        sys.exit(1)
