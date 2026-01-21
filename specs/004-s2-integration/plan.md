# Implementation Plan: Semantic Scholar (S2) Integration

**Branch**: `004-s2-integration` | **Date**: 2026-01-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-s2-integration/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Integrate Semantic Scholar API (ASTA) into bip artite to enable adding papers by DOI/PDF, exploring citation graphs, discovering literature gaps, and linking preprints to published versions. Uses direct HTTP client with Go standard library (no MCP dependency).

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage), net/http (API calls), pdfcpu (PDF parsing)
**Storage**: JSONL (refs.jsonl) + ephemeral SQLite + in-memory LRU cache for API responses
**Testing**: go test (existing pattern)
**Target Platform**: macOS, Linux (CLI binary)
**Project Type**: Single project (existing cmd/bip structure)
**Performance Goals**: Add paper <3s, citation queries <5s, gap discovery <60s for 1000 papers
**Constraints**: Respect S2 rate limits (100 req/5 min unauthenticated), cache aggressively
**Scale/Scope**: Personal reference collections (100-5000 papers typical)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | All commands via CLI, JSON output by default, `--human` flag for readable output |
| II. Git-Versionable Architecture | ✅ PASS | Data stored in JSONL (refs.jsonl), cache in .bip artite/cache/ (gitignored) |
| III. Fail-Fast Philosophy | ✅ PASS | API errors, rate limits, missing DOIs all produce clear errors |
| IV. Real Testing (Agentic TDD) | ✅ PASS | Tests with real S2 API fixtures (recorded responses) |
| V. Clean Architecture | ✅ PASS | New `internal/s2/` package, clear separation from existing code |
| VI. Simplicity | ✅ PASS | Standard library net/http, minimal deps (pdfcpu only for PDF), no MCP server |

**Technology Constraints Check**:
- CLI Responsiveness: ✅ Go compiled binary, <100ms startup
- Embeddable: ✅ SQLite (existing), no daemon processes, in-memory cache
- Data Portability: ✅ JSONL for persistent data, JSON for API cache
- Platform Support: ✅ macOS and Linux (existing)

## Project Structure

### Documentation (this feature)

```text
specs/004-s2-integration/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/bip/
├── asta.go              # bp s2 subcommand (add, add-pdf, lookup, citations, references, gaps, link-published)
├── ...                  # Existing commands unchanged

internal/
├── asta/
│   ├── client.go        # HTTP client for Semantic Scholar API
│   ├── client_test.go   # Tests with recorded fixtures
│   ├── types.go         # S2Paper, PaperIdentifier, etc.
│   ├── cache.go         # In-memory LRU cache with optional persistence
│   └── ratelimit.go     # Rate limit tracker
├── pdf/
│   ├── opener.go        # Existing PDF opener
│   └── doi_extractor.go # New: DOI extraction from PDF text (pdfcpu)
├── reference/
│   └── reference.go     # Existing (may need Source.Metadata field)
└── ...                  # Existing packages unchanged

testdata/
├── asta/                # Recorded S2 API responses for tests
│   ├── paper_by_doi.json
│   ├── citations.json
│   └── ...
└── ...                  # Existing test fixtures
```

**Structure Decision**: Single project with existing Go module structure. New `internal/s2/` package contains all Semantic Scholar client code. PDF DOI extraction added to existing `internal/pdf/` package.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
