# Implementation Plan: Concept Nodes

**Branch**: `006-concept-nodes` | **Date**: 2026-01-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-concept-nodes/spec.md`

## Summary

Extend bipartite's knowledge graph with concept nodes — named ideas, methods, or phenomena that papers relate to. This requires a new `concepts.jsonl` file for persistence, a new domain type (`internal/concept/`), SQLite indexing, and a full set of CLI commands (`bip concept *`) for CRUD, querying, and merging concepts. Paper-concept edges use the existing edges.jsonl format with validation that target concepts exist.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage)
**Storage**: JSONL (concepts.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`)
**Testing**: go test with real fixtures
**Target Platform**: macOS, Linux (CLI)
**Project Type**: Single project (CLI tool)
**Performance Goals**: Concept CRUD < 2s, concept queries < 3s for 10k papers (per SC-001/002)
**Constraints**: CLI must feel instant (<100ms startup), fail-fast on errors
**Scale/Scope**: Personal knowledge graph (thousands of papers, hundreds of concepts)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check (Phase 0)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | PASS | CLI-first, JSON default, `--human` flag |
| II. Git-Versionable | PASS | JSONL source of truth, SQLite ephemeral |
| III. Fail-Fast | PASS | Validate IDs, reject invalid input |
| IV. Real Testing | PASS | Real fixture data, no mocks |
| V. Clean Architecture | PASS | Separate domain/storage/CLI layers |
| VI. Simplicity | PASS | Minimal new code, reuse existing patterns |

### Post-Design Re-Check (Phase 1)

| Principle | Status | Verification |
|-----------|--------|--------------|
| I. Agent-First Design | PASS | All 10 CLI commands output JSON by default, `--human` optional. Exit codes defined for scripting. |
| II. Git-Versionable | PASS | concepts.jsonl is new source of truth. SQLite concepts/concepts_fts tables are ephemeral, rebuilt by `bip rebuild`. |
| III. Fail-Fast | PASS | ID validation regex `^[a-z0-9][a-z0-9_-]*$`. Delete blocked without `--force` when edges exist. Edge validation checks both refs and concepts. |
| IV. Real Testing | PASS | Test fixtures in testdata/concepts/. Integration tests use real JSONL I/O. No mocks planned. |
| V. Clean Architecture | PASS | Domain type (internal/concept), storage layer (concepts_jsonl.go, concepts_sqlite.go), CLI layer (cmd/bip/concept.go). Names: `Concept`, `ValidateForCreate()`, `ConceptsPath()`. |
| VI. Simplicity | PASS | No new dependencies. Reuses edge.Edge for paper-concept edges. No `target_type` field added (runtime lookup instead). No transaction wrapper (follows existing rebuild pattern). |

**All gates pass. No violations requiring justification.**

## Project Structure

### Documentation (this feature)

```text
specs/006-concept-nodes/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI contract)
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── concept/
│   └── concept.go       # NEW: Concept domain type + validation
├── storage/
│   ├── concepts_jsonl.go    # NEW: JSONL read/write for concepts
│   └── concepts_sqlite.go   # NEW: SQLite schema + queries for concepts
├── edge/
│   └── edge.go          # MODIFY: Add concept edge type detection
└── config/
    └── config.go        # MODIFY: Add ConceptsFile constant

cmd/bip/
├── concept.go           # NEW: bip concept * commands
├── main.go              # MODIFY: Register concept command
├── edge.go              # MODIFY: Validate concept targets
└── rebuild.go           # MODIFY: Include concepts in rebuild

testdata/
└── concepts/            # NEW: Test fixtures
    ├── test-concepts.jsonl
    └── test-paper-concept-edges.jsonl
```

**Structure Decision**: Follow existing single-project layout. New files mirror edge pattern (edge.go → concept.go, edges_jsonl.go → concepts_jsonl.go, etc.).

## Complexity Tracking

> No violations to justify — design follows existing patterns.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none) | — | — |
