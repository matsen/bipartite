# Implementation Plan: Shared Repository Workflow Commands

**Branch**: `008-shared-repo-workflow` | **Date**: 2026-01-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-shared-repo-workflow/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add three command groups to support teams sharing a paper library via git:
1. **bip open** enhancements: Open multiple papers by ID, with `--recent N` and `--since <commit>` flags
2. **bip diff / bip new**: Track papers added/removed since last commit or a specific commit
3. **bip export --bibtex** enhancements: Export specific papers with optional `--append` mode and deduplication

Technical approach: Extend existing cobra commands with git integration for commit-based filtering, modify PDF opener to support multiple files, and add BibTeX parsing for deduplication during append operations.

## Technical Context

**Language/Version**: Go 1.25.5 (matches existing codebase)
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage), os/exec (git integration)
**Storage**: JSONL (refs.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`) - no schema changes needed
**Testing**: go test with real fixtures from testdata/
**Target Platform**: macOS, Linux (consistent with existing platform support)
**Project Type**: Single CLI application
**Performance Goals**: SC-001: Open 5 papers < 3s, SC-003: Export single paper < 1s
**Constraints**: CLI startup < 100ms (constitution), git must be available on PATH
**Scale/Scope**: Commands for typical research library (100-1000 papers)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | JSON default, --human flag, CLI-only interface |
| II. Git-Versionable Architecture | ✅ PASS | Reads from JSONL/SQLite, git for commit filtering, no new persistent state |
| III. Fail-Fast Philosophy | ✅ PASS | FR-019: Actionable error messages, no silent defaults |
| IV. Real Testing | ✅ PASS | Will use real fixtures, no mocks planned |
| V. Clean Architecture | ✅ PASS | Single responsibility per command, extend existing packages |
| VI. Simplicity | ✅ PASS | Minimal new dependencies (git already assumed), no premature abstraction |

**Technology Constraints**:
- CLI Responsiveness: ✅ Go compiled, instant startup
- Embeddable Over Client-Server: ✅ No new external services (git is called via exec)
- Data Portability: ✅ JSONL source of truth preserved, classic BibTeX output (FR-015)
- Platform Support: ✅ macOS and Linux via existing pdf.Opener

## Project Structure

### Documentation (this feature)

```text
specs/008-shared-repo-workflow/
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
├── open.go              # MODIFY: Multi-paper open, --recent, --since
├── export.go            # MODIFY: Positional IDs, --append mode
├── diff.go              # NEW: bip diff command
├── new.go               # NEW: bip new command
└── git_helpers.go       # NEW: Shared git operations

internal/
├── export/
│   ├── bibtex.go        # MODIFY: Add BibTeX parsing for deduplication
│   └── bibtex_test.go   # MODIFY: Add parse/dedupe tests
├── git/
│   └── git.go           # NEW: Git operations package (refs diff, commit lookup)
├── pdf/
│   └── opener.go        # No changes needed (already supports single file open)
└── storage/
    └── sqlite.go        # MODIFY: Add query for papers by commit timestamp (if needed)

tests/
└── testdata/            # Add fixtures for git diff scenarios, .bib files
```

**Structure Decision**: Follows existing Go CLI structure. New commands in `cmd/bip/`, shared git logic in new `internal/git/` package (consistent with `internal/s2/`, `internal/asta/` pattern). Minimal changes to existing packages.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. All design decisions align with constitution principles.

---

## Post-Design Constitution Re-Check

*Verified after Phase 1 design artifacts completed.*

| Principle | Status | Post-Design Notes |
|-----------|--------|-------------------|
| I. Agent-First Design | ✅ PASS | All commands output JSON by default per contracts/cli.md |
| II. Git-Versionable Architecture | ✅ PASS | No new persistent state; git used read-only for history queries |
| III. Fail-Fast Philosophy | ✅ PASS | Error messages defined in contracts/cli.md with actionable hints |
| IV. Real Testing | ✅ PASS | quickstart.md includes test scenarios with real commands |
| V. Clean Architecture | ✅ PASS | New `internal/git/` package has single responsibility; data-model.md defines clear types |
| VI. Simplicity | ✅ PASS | BibTeX parsing is minimal regex (research.md); no new dependencies added |

**Gate Status**: ✅ PASSED - Ready for Phase 2 task generation

---

## Generated Artifacts

| Artifact | Path | Status |
|----------|------|--------|
| Research | [research.md](research.md) | ✅ Complete |
| Data Model | [data-model.md](data-model.md) | ✅ Complete |
| CLI Contract | [contracts/cli.md](contracts/cli.md) | ✅ Complete |
| Quickstart | [quickstart.md](quickstart.md) | ✅ Complete |
| Tasks | tasks.md | ⏳ Pending (use `/speckit.tasks`) |
