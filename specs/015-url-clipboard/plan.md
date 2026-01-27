# Implementation Plan: URL Output and Clipboard Support

**Branch**: `015-url-clipboard` | **Date**: 2026-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/015-url-clipboard/spec.md`

## Summary

Add a `bip url` command to output reference URLs in different formats (DOI, PubMed, PMC, arXiv, Semantic Scholar) with optional clipboard copy support. This requires extending the Reference type with external ID fields and updating the S2 mapper to populate them during import.

## Technical Context

**Language/Version**: Go 1.24.1 (from go.mod)
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite)
**Storage**: JSONL (source of truth) + ephemeral SQLite (rebuilt via `bip rebuild`)
**Testing**: `go test ./...`
**Target Platform**: macOS and Linux (Windows explicitly out of scope)
**Project Type**: single
**Performance Goals**: CLI startup <100ms (constitution requirement)
**Constraints**: No CGO dependencies preferred (constitution: pure Go SQLite)
**Scale/Scope**: Internal tool for reference management

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ Pass | CLI command, JSON output by default, composable |
| II. Git-Versionable | ✅ Pass | External IDs stored in JSONL, schema change rebuilds via `bip rebuild` |
| III. Fail-Fast | ✅ Pass | Clear errors for missing IDs, unavailable clipboard |
| IV. Real Testing | ✅ Pass | Will use real fixture data, no mocks |
| V. Clean Architecture | ✅ Pass | URL generation is pure function, clipboard abstracted |
| VI. Simplicity | ✅ Pass | Minimal implementation, shell-out for clipboard (no CGO) |

## Project Structure

### Documentation (this feature)

```text
specs/015-url-clipboard/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bip/
├── url.go               # New: bip url command implementation

internal/
├── reference/
│   └── reference.go     # Modified: add external ID fields to Reference type
├── s2/
│   └── mapper.go        # Modified: populate external IDs on import
├── storage/
│   ├── sqlite.go        # Modified: add external ID columns to schema
│   └── jsonl.go         # No changes needed (JSON marshaling automatic)
└── clipboard/           # New: clipboard abstraction package
    └── clipboard.go     # Platform detection, shell-out to pbcopy/xclip

tests/
└── (existing test patterns)
```

**Structure Decision**: Follows existing project structure. New `internal/clipboard` package encapsulates platform-specific clipboard access. URL generation logic lives in the command file since it's simple string formatting.

## Constitution Check (Post-Design)

*Re-evaluated after Phase 1 design completion.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ Pass | JSON output by default, URL to stdout for piping, stderr for messages |
| II. Git-Versionable | ✅ Pass | New fields in JSONL, ephemeral SQLite rebuilt via standard workflow |
| III. Fail-Fast | ✅ Pass | Explicit errors for missing IDs, clipboard unavailable warns but continues |
| IV. Real Testing | ✅ Pass | Tests will use references from actual S2 imports |
| V. Clean Architecture | ✅ Pass | Clipboard isolated in internal/clipboard, URL generation is pure |
| VI. Simplicity | ✅ Pass | Shell-out avoids library deps, no backward-compat shims needed |

## Complexity Tracking

No constitution violations - design follows all principles.

## Phase Completion Status

- [x] **Phase 0: Research** - Completed (research.md)
- [x] **Phase 1: Design** - Completed (data-model.md, contracts/, quickstart.md)
- [x] **Phase 2: Tasks** - Completed (tasks.md generated and implemented)
