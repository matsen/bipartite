# /board

Manage GitHub project boards.

## Instructions

```bash
flowc board list                              # List items by status
flowc board list --status blocked             # Filter by status
flowc board add 123 --repo org/repo           # Add issue to board
flowc board move 123 --repo org/repo --status done  # Move item
flowc board remove 123 --repo org/repo        # Remove from board
flowc board sync                              # Check beads sync
flowc board refresh-cache                     # Refresh board metadata
```

## Subcommands

### list
Show board items grouped by status (blocked, next, active, done).

Options:
- `--status STATUS` — Filter by status
- `--label LABEL` — Filter by label
- `--json` — Output as JSON
- `--board owner/number` — Specify board

### add
Add an issue to the board.

Options:
- `--repo org/repo` — Repository (required)
- `--status STATUS` — Initial status
- `--label LABEL` — Label to apply

### move
Move an item to a different status.

Options:
- `--repo org/repo` — Repository (required)
- `--status STATUS` — New status (required)

### sync
Compare board state with beads and report mismatches.

Options:
- `--fix` — Auto-fix mismatches
