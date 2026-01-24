# /bip.board

Manage GitHub project boards.

## Instructions

```bash
bip board list                              # List items by status
bip board list --status blocked             # Filter by status
bip board add 123 --repo org/repo           # Add issue to board
bip board move 123 --repo org/repo --status done  # Move item
bip board remove 123 --repo org/repo        # Remove from board
bip board sync                              # Check beads sync
bip board sync --fix                        # Auto-add missing P0 beads
```

## Subcommands

### list
Show board items grouped by status (blocked, next, active, done).

Options:
- `--status STATUS` — Filter by status
- `--json` — Output as JSON
- `--board owner/number` — Specify board

### add
Add an issue to the board.

Options:
- `--repo org/repo` — Repository (required)
- `--status STATUS` — Initial status

### move
Move an item to a different status.

Options:
- `--repo org/repo` — Repository (required)
- `--status STATUS` — New status (required)

### remove
Remove an issue from the board.

Options:
- `--repo org/repo` — Repository (required)

### sync
Compare board state with P0 beads and report mismatches.

Options:
- `--fix` — Auto-fix mismatches (adds missing P0 beads to board)
