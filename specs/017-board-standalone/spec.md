# Feature Specification: Standalone Board Management

**Feature Branch**: `82-refactor-board-standalone`
**Created**: 2026-01-31
**Status**: Implemented
**Input**: GitHub issue #82 — Refactor bip board to standalone board management tool

## Motivation

Previously `bip board` was coupled to beads via sync logic that enforced P0 beads ↔ board alignment. But:

- Boards show "what's actively being worked on" (tactical)
- Beads track "research questions and goals" (strategic)
- These aren't the same set

Many board items are tactical (bug fixes, refactors, active PRs) and don't need beads. Creating beads just to get something on the board adds friction.

## Changes

### 1. New boards mapping format in sources.yml

**Before** (board → bead_id):
```json
"boards": {
  "matsengrp/30": "flow-dasm"
}
```

**After** (channel → board):
```json
"boards": {
  "dasm2": "matsengrp/30",
  "loris": "matsengrp/29"
}
```

### 2. Simplified `bip board add` command

**Before**:
```bash
bip board add 207 --repo matsengrp/dasm2-experiments
```

**After**:
```bash
bip board add dasm2-experiments#207    # Auto-resolves board from channel
bip board add repo#123 --to matsengrp/30  # Explicit board
```

Resolution: repo → channel (from code/writing array) → board (from boards mapping)

### 3. Multi-board `bip board list`

**Before**: Single board only
**After**: All boards by default, single board if argument provided

```bash
bip board list                  # All boards
bip board list matsengrp/30     # Specific board
```

### 4. Removed sync command

`bip board sync` is removed entirely. Beads and boards are now independent.

## User Scenarios

### User Story 1 — Add Issue to Board (Priority: P1)

A user wants to add a GitHub issue to a project board without specifying which board.

**Acceptance Scenarios**:

1. **Given** a repo with a channel configured in sources.yml, **When** the user runs `bip board add dasm2-experiments#207`, **Then** the issue is added to the board mapped to that channel.
2. **Given** a repo without a channel, **When** the user runs `bip board add myrepo#42`, **Then** an error is shown suggesting `--to` flag.
3. **Given** any repo, **When** the user runs `bip board add myrepo#42 --to matsengrp/30`, **Then** the issue is added to the explicit board.

### User Story 2 — List All Boards (Priority: P1)

A user wants cross-board visibility of all active work.

**Acceptance Scenarios**:

1. **Given** multiple boards configured, **When** the user runs `bip board list`, **Then** all boards are displayed with items grouped by status.
2. **Given** a specific board key, **When** the user runs `bip board list matsengrp/30`, **Then** only that board is displayed.

## Requirements

### Functional Requirements

- **FR-001**: `bip board add <repo#N>` MUST resolve board from repo's channel mapping.
- **FR-002**: `bip board add --to <board>` MUST allow explicit board override.
- **FR-003**: `bip board list` MUST show all boards by default.
- **FR-004**: `bip board list <board>` MUST filter to a single board.
- **FR-005**: `bip board move` and `bip board remove` MUST accept `repo#N` format.
- **FR-006**: Short repo names (without org/) MUST expand to `matsengrp/<repo>`.
- **FR-007**: `bip board sync` MUST be removed.

### Configuration Requirements

- **CR-001**: sources.yml `boards` field MUST map channel names to board keys.
- **CR-002**: Repos MUST have `channel` field in code/writing arrays to use auto-resolution.

## Success Criteria

- **SC-001**: `bip board add dasm2-experiments#207` adds to correct board without explicit --repo or --board flags.
- **SC-002**: `bip board list` shows all configured boards in a single view.
- **SC-003**: All existing board functionality (add, move, remove, refresh-cache) works with new `repo#N` syntax.
- **SC-004**: `bip board sync` command is removed.
