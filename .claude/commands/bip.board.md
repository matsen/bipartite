# /bip.board

Manage GitHub project boards. Boards are resolved automatically from repo → channel → board mappings in sources.json.

## Quick Reference

```bash
bip board list                              # List ALL boards by status
bip board list matsengrp/30                 # List specific board
bip board add dasm2-experiments#207         # Add issue (board auto-resolved)
bip board add netam#171 --status "Next"     # Add with initial status
bip board move dasm2-experiments#207 --status done  # Move item
bip board remove netam#171                  # Remove from board
```

## Board Resolution

The board is automatically resolved via channel mappings:
1. Look up repo's channel from `code`/`writing` array in sources.json
2. Look up channel's board from `boards` mapping

Example sources.json:
```json
{
  "boards": {
    "dasm2": "matsengrp/30",
    "loris": "matsengrp/29"
  },
  "code": [
    {"repo": "matsengrp/dasm2-experiments", "channel": "dasm2"},
    {"repo": "matsengrp/loris-experiments", "channel": "loris"}
  ]
}
```

## Subcommands

### list
Show board items grouped by status. Shows ALL boards by default.

```bash
bip board list                  # All boards
bip board list matsengrp/30     # Specific board
bip board list --status "In Progress"  # Filter by status
bip board list --json           # JSON output
```

### add
Add an issue or PR to a board.

```bash
bip board add dasm2-experiments#207    # Auto-resolve board from channel
bip board add repo#123 --to matsengrp/30  # Explicit board
bip board add repo#123 --status "Next"    # Set initial status
```

### move
Move an item to a different status.

```bash
bip board move dasm2-experiments#207 --status done
bip board move repo#123 --status "In Progress" --to matsengrp/30
```

### remove
Remove an issue/PR from a board.

```bash
bip board remove dasm2-experiments#207
bip board remove repo#123 --to matsengrp/30  # Explicit board
```

### refresh-cache
Refresh cached board metadata (status options, field IDs).

```bash
bip board refresh-cache
bip board refresh-cache --board matsengrp/30
```
