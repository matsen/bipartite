# Implementation Plan: Generic JSONL + SQLite Store Abstraction

**Branch**: `014-jsonl-sqlite-store` | **Date**: 2026-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/014-jsonl-sqlite-store/spec.md`

## Summary

Add a general-purpose store abstraction where JSONL files are the source of truth and SQLite databases serve as queryable indexes. This generalizes the existing pattern already used by refs, concepts, and edges into a reusable API with JSON schema definitions, automatic SQLite DDL generation, and CLI commands for store management.

## Technical Context

**Language/Version**: Go 1.21+ (matches existing go.mod)
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite)
**Storage**: JSONL (source of truth) + SQLite (ephemeral index)
**Testing**: go test with real fixtures
**Target Platform**: macOS, Linux CLI
**Project Type**: Single project - extends existing `bip` CLI
**Performance Goals**: 1000 records append in <5s, 10k records sync in <10s, FTS query <1s
**Constraints**: <100ms CLI startup, local filesystem only, no CGO
**Scale/Scope**: Thousands to tens of thousands of records per store

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Agent-First Design | PASS | CLI commands with JSON output, composable with pipes |
| II. Git-Versionable Architecture | PASS | JSONL source of truth, SQLite ephemeral/gitignored |
| III. Fail-Fast Philosophy | PASS | Schema validation on append, clear errors for invalid input |
| IV. Real Testing | PASS | Tests use real JSONL fixtures, no mocks |
| V. Clean Architecture | PASS | Single responsibility (schema, JSONL ops, SQLite gen), good naming |
| VI. Simplicity | PASS | Minimal deps (existing sqlite driver), no premature abstraction |

**Gate Result**: PASS - All principles satisfied

## Project Structure

### Documentation (this feature)

```text
specs/014-jsonl-sqlite-store/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # CLI command specifications
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── store/               # NEW: Generic store abstraction
│   ├── schema.go        # Schema parsing and validation
│   ├── schema_test.go
│   ├── jsonl.go         # Generic JSONL operations
│   ├── jsonl_test.go
│   ├── sqlite.go        # SQLite DDL generation and sync
│   ├── sqlite_test.go
│   ├── store.go         # Store type and operations
│   └── store_test.go
├── storage/             # EXISTING: Will be refactored to use generic store
│   └── ...

cmd/bip/
├── store.go             # NEW: `bip store` command group
├── store_init.go        # NEW: `bip store init`
├── store_append.go      # NEW: `bip store append`
├── store_delete.go      # NEW: `bip store delete`
├── store_sync.go        # NEW: `bip store sync`
├── store_query.go       # NEW: `bip store query`
├── store_list.go        # NEW: `bip store list`
├── store_info.go        # NEW: `bip store info`
└── ...

testdata/
└── stores/              # NEW: Test fixtures for store tests
    ├── valid_schema.json
    ├── invalid_schema_no_primary.json
    ├── sample_records.jsonl
    └── ...
```

**Structure Decision**: Extends existing single-project structure. New `internal/store/` package provides generic abstraction. CLI commands added under `cmd/bip/store*.go`. Existing `internal/storage/` will eventually be refactored to use generic store (P3 migration story).

## Complexity Tracking

No constitution violations - no entries needed.
