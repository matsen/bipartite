"""Main CLI dispatcher for fc."""

from __future__ import annotations

import argparse
import sys

from fc_cli.shared.config import validate_nexus_directory


def main():
    """Main entry point for the fc CLI."""
    validate_nexus_directory()

    parser = argparse.ArgumentParser(
        prog="flowc",
        description="Flow-central CLI for managing GitHub activity and project boards",
    )
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Checkin subcommand
    checkin_parser = subparsers.add_parser(
        "checkin", help="Check in on GitHub activity across tracked repos"
    )
    checkin_parser.add_argument("--since", help="Time period (e.g., 2d, 12h, 1w)")
    checkin_parser.add_argument("--repo", help="Check single repo only")
    checkin_parser.add_argument(
        "--category", help="Check repos in category only (writing, code)"
    )
    checkin_parser.add_argument(
        "--summarize",
        action="store_true",
        help="Generate LLM take-home summaries for each item (uses claude CLI)",
    )
    checkin_parser.add_argument(
        "--all",
        action="store_true",
        help="Show all activity (disable ball-in-my-court filtering)",
    )

    # Board subcommand
    board_parser = subparsers.add_parser("board", help="Manage GitHub project boards")
    board_subparsers = board_parser.add_subparsers(
        dest="board_command", help="Board commands"
    )

    # board list
    list_parser = board_subparsers.add_parser("list", help="List board items by status")
    list_parser.add_argument(
        "--status", help="Filter by status (blocked, next, active, done)"
    )
    list_parser.add_argument("--label", help="Filter by label")
    list_parser.add_argument("--json", action="store_true", help="Output as JSON")
    list_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # board add
    add_parser = board_subparsers.add_parser("add", help="Add issue to board")
    add_parser.add_argument("issue", type=int, help="Issue number to add")
    add_parser.add_argument("--status", help="Initial status (blocked, next, active)")
    add_parser.add_argument("--label", help="Label to apply")
    add_parser.add_argument(
        "--repo", help="Repository (org/repo format, required)"
    )
    add_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # board move
    move_parser = board_subparsers.add_parser(
        "move", help="Move item to different status"
    )
    move_parser.add_argument("issue", type=int, help="Issue number to move")
    move_parser.add_argument("--status", required=True, help="New status")
    move_parser.add_argument(
        "--repo", help="Repository (org/repo format, required)"
    )
    move_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # board remove
    remove_parser = board_subparsers.add_parser(
        "remove", help="Remove issue from board"
    )
    remove_parser.add_argument("issue", type=int, help="Issue number to remove")
    remove_parser.add_argument(
        "--repo", help="Repository (org/repo format, required)"
    )
    remove_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # board sync
    sync_parser = board_subparsers.add_parser(
        "sync", help="Sync board with beads (report mismatches)"
    )
    sync_parser.add_argument("--fix", action="store_true", help="Auto-fix mismatches")
    sync_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # board refresh-cache
    refresh_parser = board_subparsers.add_parser(
        "refresh-cache", help="Refresh cached board metadata"
    )
    refresh_parser.add_argument(
        "--board",
        help="Board to use (owner/number, e.g., matsengrp/30). Defaults to first board in sources.json",
    )

    # Spawn subcommand (replaces issue, handles both issues and PRs)
    spawn_parser = subparsers.add_parser(
        "spawn", help="Spawn tmux window for GitHub issue or PR review"
    )
    spawn_parser.add_argument(
        "ref",
        help="GitHub reference: org/repo#number or full URL (e.g., matsengrp/repo#166 or https://github.com/org/repo/pull/42)",
    )
    spawn_parser.add_argument(
        "--prompt",
        help="Custom prompt to use instead of default review prompt; project context is prepended automatically",
    )

    # Issue subcommand (deprecated, kept for backwards compatibility)
    issue_parser = subparsers.add_parser(
        "issue",
        help="[Deprecated: use 'spawn'] Spawn tmux window for GitHub issue review",
    )
    issue_parser.add_argument(
        "issue_ref",
        help="Issue reference in org/repo#number format (e.g., matsengrp/dasm2-experiments#166)",
    )

    # Digest subcommand
    digest_parser = subparsers.add_parser(
        "digest", help="Generate and post activity digest to Slack"
    )
    digest_parser.add_argument(
        "--channel",
        required=True,
        help="Channel whose repos to scan (e.g., dasm2, loris)",
    )
    digest_parser.add_argument(
        "--since",
        default="1w",
        help="Time period to summarize (e.g., 1w, 2d, 12h). Default: 1w",
    )
    digest_parser.add_argument(
        "--post-to",
        help="Override destination channel (e.g., --post-to scratch for testing)",
    )
    digest_parser.add_argument(
        "--repos",
        help="Override repos to scan (comma-separated, e.g., matsengrp/repo1,matsengrp/repo2)",
    )

    # Tree subcommand
    tree_parser = subparsers.add_parser(
        "tree", help="Generate interactive HTML tree view of beads issues"
    )
    tree_parser.add_argument(
        "--since",
        help="Highlight beads created after this date (YYYY-MM-DD or ISO format)",
    )
    tree_parser.add_argument(
        "-o", "--output",
        help="Output file path (default: stdout)",
    )
    tree_parser.add_argument(
        "--open",
        action="store_true",
        help="Open in browser after generating",
    )

    args = parser.parse_args()

    if args.command is None:
        parser.print_help()
        sys.exit(1)

    if args.command == "checkin":
        from fc_cli.checkin.cli import run_checkin

        run_checkin(args)

    elif args.command == "board":
        if args.board_command is None:
            board_parser.print_help()
            sys.exit(1)

        from fc_cli.board.cli import run_board

        run_board(args)

    elif args.command == "spawn":
        from fc_cli.issue import run_spawn

        sys.exit(run_spawn(args))

    elif args.command == "issue":
        from fc_cli.issue import run_issue

        sys.exit(run_issue(args))

    elif args.command == "digest":
        from fc_cli.digest.cli import run_digest

        sys.exit(run_digest(args))

    elif args.command == "tree":
        from fc_cli.tree.cli import run_tree

        sys.exit(run_tree(args))


if __name__ == "__main__":
    main()
