# Implementation Plan: flowc Go Port

**Branch**: `30-flowc-go-port` | **Date**: 2026-01-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-flowc-go-port/spec.md`

## Summary

Port the flowc Python CLI to Go, integrating it directly into the bip CLI (e.g., `bip checkin`, `bip board`). The goal is a single Go binary with no Python dependencies, maintaining exact behavioral parity with the existing 110-test Python implementation.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), existing bip infrastructure
**Storage**: JSONL (sources.yml, .beads/issues.jsonl) - read-only for this feature
**Testing**: go test with table-driven tests, porting all 110 Python tests
**Target Platform**: macOS and Linux
**Project Type**: Single project (extends existing bip CLI)
**Performance Goals**: CLI startup <100ms (already achieved by Go)
**Constraints**: Must pass all 110 existing Python tests, exact ball-in-court logic match
**Scale/Scope**: 6 commands (checkin, board, spawn, digest, tree, plus board subcommands)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | CLI-first, JSON output supported |
| II. Git-Versionable | ✅ PASS | Reads existing JSONL files, no new persistent state |
| III. Fail-Fast | ✅ PASS | Clear errors for missing sources.yml, invalid refs |
| IV. Real Testing | ✅ PASS | Porting real Python tests with real fixture data |
| V. Clean Architecture | ✅ PASS | Follows existing bip patterns |
| VI. Simplicity | ✅ PASS | Uses gh CLI wrapper (existing), minimal deps |

**External Dependencies**:
- `gh` CLI for GitHub API (already assumed by existing flowc)
- `tmux` for spawn command (already assumed)
- `claude` CLI for LLM integration (already assumed)

All external dependencies are runtime tools, not library dependencies - consistent with constitution.

## Project Structure

### Documentation (this feature)

```text
specs/010-flowc-go-port/
├── plan.md              # This file
├── research.md          # Existing behavior documentation (complete)
├── spec.md              # Feature specification (complete)
└── tasks.md             # Task breakdown (exists)
```

### Source Code (repository root)

```text
cmd/bip/
├── checkin.go           # bip flow checkin command
├── board.go             # bip flow board commands
├── spawn.go             # bip flow spawn command
├── digest.go            # bip flow digest command
└── tree.go              # bip flow tree command

internal/flow/
├── types.go             # Shared types (Sources, Bead, RepoEntry)
├── config.go            # sources.yml parsing
├── config_test.go       # Config tests (12 from Python)
├── beads.go             # .beads/issues.jsonl loading
├── duration.go          # Duration parsing (2d, 12h, 1w)
├── duration_test.go     # Duration tests (9 from Python)
├── ghref.go             # GitHub reference parsing
├── ghref_test.go        # GitHub ref tests (34 from Python)
├── time.go              # Relative time formatting
├── time_test.go         # Time tests
├── ballcourt.go         # Ball-in-court filtering logic
├── ballcourt_test.go    # Ball-in-court tests (20 from Python)
├── gh.go                # GitHub API via gh CLI
├── llm.go               # LLM prompt building and response parsing
├── slack.go             # Slack webhook integration
├── board/
│   ├── cache.go         # Board metadata caching
│   └── api.go           # Board GraphQL operations
├── checkin/
│   └── activity.go      # Activity fetching
├── digest/
│   └── activity.go      # Digest activity by channel
├── spawn/
│   └── tmux.go          # Tmux window management
└── tree/
    └── tree.go          # HTML tree generation
```

**Structure Decision**: Extends existing bip CLI structure. Core logic in `internal/flow/` with command-specific subdirectories for complex operations. Flat structure for shared utilities (duration, ghref, time, ballcourt).

## Complexity Tracking

No constitution violations to justify.

## Research Summary

Research phase complete - see [research.md](./research.md) for:
- Ball-in-court truth table and edge cases
- Duration parsing valid/invalid formats
- GitHub reference parsing rules (hash format, URL format)
- Relative time formatting boundaries
- Comment/file/review truncation rules
- LLM prompt structure and response parsing
- All 110 test case behaviors documented

## Implementation Phases

### Phase 1: Core Infrastructure (T001-T006)
Shared types, config parsing, beads loading, utility functions (duration, ghref, time).

### Phase 2: GitHub API (T007-T009)
gh CLI wrapper for REST/GraphQL, ball-in-court filtering implementation.

### Phase 3: Commands (T010-T042)
Implement commands in priority order: checkin (P1), board (P1/P2), spawn (P2), digest (P2), tree (P3).

### Phase 4: Testing (T046-T051)
Port all 110 Python tests to Go table-driven tests.

### Phase 5: Polish (T052-T055)
Integration tests, README update, skill consolidation, Python removal.
