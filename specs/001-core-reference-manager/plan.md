# Implementation Plan: Core Reference Manager

**Branch**: `001-core-reference-manager` | **Date**: 2026-01-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-core-reference-manager/spec.md`

## Summary

Build the foundational `bp` CLI tool in Go that enables researchers and AI agents to manage academic references. The system uses JSONL as the source of truth with an ephemeral SQLite database for queries, imports from Paperpile JSON exports, exports to BibTeX, and opens PDFs from a configured folder path.

## Technical Context

**Language/Version**: Go 1.21+ (latest stable)
**Primary Dependencies**:
- `spf13/cobra` - CLI framework (de facto standard for Go CLIs)
- `modernc.org/sqlite` - pure Go SQLite (no CGO, easy cross-compilation)
**Storage**: JSONL source of truth + ephemeral SQLite for queries
**Testing**: Go's built-in `testing` package with real fixture data
**Target Platform**: macOS and Linux (darwin/amd64, darwin/arm64, linux/amd64)
**Project Type**: Single CLI application
**Performance Goals**:
- CLI startup: <100ms
- Import 1000 papers: <30s
- Search queries: <500ms for 10k papers
- PDF open: <1s to launch viewer
**Constraints**:
- No CGO (pure Go for easy cross-compilation)
- Embeddable SQLite (no separate server)
- Single binary distribution
**Scale/Scope**: Up to 10,000 papers per repository (researcher's personal library)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Agent-First Design
- [x] **CLI is primary interface**: `bp` command with subcommands
- [x] **Structured output by default**: JSON output for all queries
- [x] **Human-readable alternative**: `--human` or similar flag
- [x] **No MCP required**: Direct bash interaction
- [x] **Composable**: Works with pipes, beads orchestration

### II. Git-Versionable Architecture
- [x] **JSONL source of truth**: `refs.jsonl` stores all references
- [x] **Ephemeral query layer**: SQLite rebuilt via `bip rebuild`
- [x] **Database gitignored**: `cache/refs.db` in `.gitignore`
- [x] **Clean merges**: JSONL format supports append-style merges
- [x] **Self-contained repos**: Each bipartite repo is standalone

### III. Fail-Fast Philosophy
- [x] **No silent defaults**: Missing config errors immediately
- [x] **Clear error messages**: What went wrong + what was expected
- [x] **No silent swallowing**: All failures visible
- [x] **Invalid input rejected**: Not silently corrected
- [x] **Missing files error**: Clear errors for missing PDFs

### IV. Real Testing (Agentic TDD)
- [x] **No fake mocks**: Real Paperpile export fixtures from `_ignore/`
- [x] **Edge case coverage**: Missing DOIs, multiple attachments, partial dates
- [x] **TDD cycle**: Agent writes test → implements → iterates
- [x] **Integration tests**: Real file I/O and database operations

### V. Clean Architecture
- [x] **Single responsibility**: Separate modules for import, query, export
- [x] **Dependency inversion**: Storage interface abstracts JSONL/SQLite
- [x] **Meaningful names**: `paper_doi` not `id`, `has_abstract` not `abstract`
- [x] **No generic names**: No Manager, Handler, Utils

### VI. Simplicity
- [x] **Minimal dependencies**: Standard library + pure Go SQLite
- [x] **Fast startup**: Go compiled binary
- [x] **No premature abstraction**: Build for current needs
- [x] **Delete unused code**: No commented-out code

### Technology Constraints
- [x] **CLI responsiveness**: Go provides <100ms startup
- [x] **Embeddable database**: Pure Go SQLite (modernc.org/sqlite)
- [x] **Data portability**: JSONL + BibTeX are human-readable
- [x] **Platform support**: macOS + Linux via Go cross-compilation

**Gate Status**: PASS - No violations requiring justification

## Project Structure

### Documentation (this feature)

```text
specs/001-core-reference-manager/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI contract)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── bp/
    └── main.go          # CLI entry point

internal/
├── config/              # Repository configuration
│   ├── config.go
│   └── config_test.go
├── importer/            # Import from external formats
│   ├── paperpile.go
│   └── paperpile_test.go
├── reference/           # Core reference domain
│   ├── reference.go     # Reference struct and methods
│   └── author.go        # Author struct
├── storage/             # Data persistence
│   ├── jsonl.go         # JSONL read/write
│   ├── sqlite.go        # SQLite query layer
│   └── storage_test.go
├── query/               # Search and retrieval
│   ├── search.go
│   └── search_test.go
├── export/              # Export formats
│   ├── bibtex.go
│   └── bibtex_test.go
└── pdf/                 # PDF path resolution and opening
    ├── opener.go
    └── opener_test.go

testdata/                # Test fixtures (subset of _ignore/)
├── paperpile_sample.json
└── expected_outputs/

go.mod
go.sum
```

**Structure Decision**: Single CLI application with `internal/` packages organized by domain responsibility. The `cmd/bip/` pattern follows Go conventions for CLI tools.

## Complexity Tracking

> No violations requiring justification - design follows constitution principles.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | -          | -                                   |

---

## Post-Design Constitution Re-Check

*Re-evaluated after Phase 1 design completion.*

**Status**: PASS

All constitution principles verified against design artifacts:

- **Agent-First**: CLI contract defines JSON-default output with `--human` flag (contracts/cli.md)
- **Git-Versionable**: JSONL format with ephemeral SQLite documented (data-model.md)
- **Fail-Fast**: Exit codes and explicit error messages defined (contracts/cli.md)
- **Real Testing**: Fixture strategy with real Paperpile data documented (quickstart.md)
- **Clean Architecture**: Internal packages organized by domain responsibility (plan.md)
- **Simplicity**: Minimal dependencies - only standard library + pure Go SQLite (research.md)

No additional violations introduced during design phase.
