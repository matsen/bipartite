# Implementation Plan: Domain-Aware Conflict Resolution

**Branch**: `009-refs-conflict-resolve` | **Date**: 2026-01-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/009-refs-conflict-resolve/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement `bip resolve` command for domain-aware conflict resolution in refs.jsonl. Git treats JSON as opaque blobs, but bip understands that DOI is a unique identifier for matching papers, one version may have metadata the other lacks, and more complete records are preferable. The command auto-resolves conflicts where possible, merges complementary metadata, and prompts interactively only for true field-level conflicts.

## Technical Context

**Language/Version**: Go 1.25.5 (matches existing codebase)
**Primary Dependencies**: spf13/cobra (CLI), os/exec (git integration), bufio (user prompts)
**Storage**: JSONL (refs.jsonl) - reads conflicted file, writes resolved version
**Testing**: go test (table-driven tests with fixture files)
**Target Platform**: macOS, Linux (CLI tool)
**Project Type**: Single project - extends existing CLI
**Performance Goals**: Resolve 100+ paper conflicts instantly (<100ms)
**Constraints**: Must handle malformed conflict markers gracefully; no external network calls
**Scale/Scope**: Typical conflicts: 1-50 papers; file size: up to 10MB refs.jsonl

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check (Initial Gate)

| Principle | Requirement | Status | Notes |
|-----------|-------------|--------|-------|
| I. Agent-First Design | CLI primary interface; JSON default output | ✅ PASS | `bip resolve` with JSON output, `--human` flag for readable output |
| II. Git-Versionable | JSONL source of truth; no new persistent state | ✅ PASS | Reads/writes refs.jsonl only; no new database tables |
| III. Fail-Fast | No silent failures; clear error messages | ✅ PASS | Exit with error on malformed markers; suggest `--interactive` for unresolvable |
| IV. Real Testing | Real fixtures; no fake mocks | ✅ PASS | Test with actual conflicted JSONL content |
| V. Clean Architecture | Single responsibility; good naming | ✅ PASS | Separate conflict parsing, paper matching, resolution logic |
| VI. Simplicity | Minimal dependencies; no premature abstraction | ✅ PASS | Uses existing storage package; standard library for prompts |

**Gate Status**: ✅ All principles satisfied - proceeded to Phase 0

### Post-Design Check (Re-evaluation after Phase 1)

| Principle | Post-Design Status | Design Validation |
|-----------|-------------------|-------------------|
| I. Agent-First | ✅ CONFIRMED | ResolveResult JSON struct follows existing patterns; numbered interactive prompts work in terminal/script contexts |
| II. Git-Versionable | ✅ CONFIRMED | No new persistent files; resolved JSONL maintains line-based structure for clean diffs |
| III. Fail-Fast | ✅ CONFIRMED | ParseError struct provides line numbers; exit code 3 for data errors; actionable error messages |
| IV. Real Testing | ✅ CONFIRMED | testdata/conflict/ fixtures cover all scenarios: completeness, merge, true conflict, malformed |
| V. Clean Architecture | ✅ CONFIRMED | internal/conflict package has clear separation: parser.go, matcher.go, resolver.go each single-purpose |
| VI. Simplicity | ✅ CONFIRMED | No external dependencies added; state machine parser is minimal; no over-abstracted interfaces |

**Final Gate Status**: ✅ All principles confirmed after detailed design - ready for task generation

## Project Structure

### Documentation (this feature)

```text
specs/009-refs-conflict-resolve/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/bip/
├── resolve.go           # New command implementation
├── resolve_test.go      # Unit tests for resolve command
└── types.go             # Add ResolveResult, ConflictInfo structs

internal/
├── conflict/            # New package for conflict resolution logic
│   ├── parser.go        # Parse git conflict markers
│   ├── parser_test.go   # Parser tests
│   ├── matcher.go       # Match papers by DOI/ID
│   ├── matcher_test.go  # Matcher tests
│   ├── resolver.go      # Merge/resolution logic
│   └── resolver_test.go # Resolution tests
└── reference/
    └── reference.go     # Existing - no changes needed

testdata/
└── conflict/            # New test fixtures
    ├── simple_ours_better.jsonl
    ├── simple_theirs_better.jsonl
    ├── complementary_merge.jsonl
    ├── true_conflict.jsonl
    ├── multiple_papers.jsonl
    └── malformed_markers.jsonl
```

**Structure Decision**: Extends existing bipartite structure. New `internal/conflict` package separates parsing, matching, and resolution concerns per Clean Architecture principle. Command layer (`cmd/bip/resolve.go`) orchestrates the workflow.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

*No violations - all constitution principles satisfied.*
