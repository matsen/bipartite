# Implementation Plan: Knowledge Graph

**Branch**: `003-knowledge-graph` | **Date**: 2026-01-13 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-knowledge-graph/spec.md`

## Summary

Add knowledge graph capabilities to bipartite: directed edges between papers with relationship types and relational summaries. External tools (like tex-to-edges Claude skill) generate edges; bp provides storage, query, and export. Follows existing JSONL + ephemeral SQLite architecture.

## Technical Context

**Language/Version**: Go 1.25.5 (continuing Phase I/II)
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage) - no new dependencies
**Storage**: JSONL (edges.jsonl) + ephemeral SQLite (edge index rebuilt on `bp rebuild`)
**Testing**: go test with real fixture data
**Target Platform**: macOS and Linux (CLI)
**Project Type**: single (CLI tool extending existing `bp` binary)
**Performance Goals**: <1s per edge add, <500ms query for 10k edges
**Constraints**: Ephemeral DB, JSONL source of truth, git-mergeable, no external services

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Agent-First Design | ✅ Pass | All `bp edge` commands support `--json` flag for structured output |
| II. Git-Versionable Architecture | ✅ Pass | edges.jsonl as source of truth, SQLite index ephemeral and gitignored |
| III. Fail-Fast Philosophy | ✅ Pass | Validate paper IDs exist before adding edges; explicit errors for missing papers |
| IV. Real Testing (Agentic TDD) | ✅ Pass | Will use real edge fixtures, integration tests with actual file I/O |
| V. Clean Architecture | ✅ Pass | New `internal/edge/` package follows existing patterns; clear naming |
| VI. Simplicity | ✅ Pass | No new dependencies; extends existing storage patterns |

**Gate Result**: PASS - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/003-knowledge-graph/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli.md           # CLI contract for edge commands
└── tasks.md             # Phase 2 output (speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bp/
├── edge.go              # bp edge subcommand (add, import, list, search, export)
└── ... (existing commands)

internal/
├── edge/
│   ├── edge.go          # Edge domain type
│   └── edge_test.go     # Unit tests
├── storage/
│   ├── edges_jsonl.go   # JSONL read/write for edges
│   ├── edges_jsonl_test.go
│   ├── edges_sqlite.go  # SQLite index for edge queries
│   └── edges_sqlite_test.go
└── ... (existing packages)

tests/
└── integration/
    └── edge_test.go     # Integration tests for edge workflows
```

**Structure Decision**: Extends existing single-project structure. New `internal/edge/` package for domain type, extends `internal/storage/` for persistence. Follows existing patterns from Phase I/II.

## Complexity Tracking

> No violations identified. Design follows existing patterns with no new dependencies.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none) | — | — |
