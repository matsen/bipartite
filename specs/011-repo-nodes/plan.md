# Implementation Plan: Projects and Repos as Knowledge Graph Nodes

**Branch**: `011-repo-nodes` | **Date**: 2026-01-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/011-repo-nodes/spec.md`

## Summary

Extend the bipartite knowledge graph to include **projects** (first-class nodes representing ongoing work) and **repos** (GitHub repositories belonging to projects). Projects connect to concepts (not directly to papers), enforcing the discipline that connections must go through named ideas. This completes the "bipartite" vision: papers ↔ concepts ↔ projects.

**Technical Approach**:
- Add `internal/project/` and `internal/repo/` domain packages following the existing `internal/concept/` pattern
- Add `projects.jsonl` and `repos.jsonl` storage (parallel to `concepts.jsonl`)
- Extend edge validation to support concept↔project edges while rejecting paper↔project and *↔repo edges
- Add type-prefixed IDs (`project:`, `repo:`) for new nodes; existing paper/concept IDs remain unprefixed for backward compatibility
- GitHub metadata fetching via REST API (no auth required for public repos)

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage)
**Storage**: JSONL (source of truth) + SQLite (ephemeral query layer, rebuilt via `bip rebuild`)
**Testing**: `go test ./...` with real fixtures in `testdata/`
**Target Platform**: macOS, Linux (CLI tool)
**Project Type**: Single CLI application
**Performance Goals**: Local CRUD operations <100ms, GitHub fetch <10s timeout
**Constraints**: No daemon processes, embeddable SQLite only, JSON output by default
**Scale/Scope**: Dozens of projects/repos per bipartite repository (researcher-scale)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | CLI commands with `--json` flag, composable with pipes |
| II. Git-Versionable Architecture | ✅ PASS | JSONL source of truth, SQLite ephemeral |
| III. Fail-Fast Philosophy | ✅ PASS | Reject invalid edges immediately, fail on ID collision |
| IV. Real Testing (Agentic TDD) | ✅ PASS | Will use real fixtures, no mocks |
| V. Clean Architecture | ✅ PASS | Domain packages (`internal/project/`, `internal/repo/`) with clear responsibilities |
| VI. Simplicity | ✅ PASS | No migration script (manual), separate JSONL files (simple), fail on collision (no auto-suffix) |

**Clarifications Applied** (from user):
- All 3 phases in scope
- No `bip migrate` command; existing edges updated manually in nexus
- Use separate `projects.jsonl`/`repos.jsonl` (simpler than sources.json integration)
- ID collision → error (no auto-suffix)

## Project Structure

### Documentation (this feature)

```text
specs/011-repo-nodes/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI interface specs)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/bip/
├── project.go           # NEW: bip project add/get/list/update/delete/repos/concepts/papers
├── repo.go              # NEW: bip repo add/get/list/update/delete/refresh
├── edge.go              # MODIFY: add type validation, reject paper↔project and *↔repo
├── check.go             # MODIFY: validate project/repo constraints
└── rebuild.go           # MODIFY: include projects/repos in SQLite rebuild

internal/
├── project/
│   └── project.go       # NEW: Project domain type + validation
├── repo/
│   └── repo.go          # NEW: Repo domain type + validation
├── github/
│   └── client.go        # NEW: GitHub API client for repo metadata
├── config/
│   └── config.go        # MODIFY: add ProjectsFile, ReposFile constants
└── storage/
    ├── projects_jsonl.go    # NEW: JSONL read/write for projects
    ├── projects_sqlite.go   # NEW: SQLite operations for projects
    ├── repos_jsonl.go       # NEW: JSONL read/write for repos
    ├── repos_sqlite.go      # NEW: SQLite operations for repos
    └── edges_sqlite.go      # MODIFY: transitive queries (project→concepts→papers)

tests/
└── (go test in each package)

testdata/
├── projects/            # NEW: test fixtures
└── repos/               # NEW: test fixtures
```

**Structure Decision**: Single CLI application following existing patterns. New domain types in `internal/`, new CLI commands in `cmd/bip/`, JSONL+SQLite storage following existing concepts/edges patterns.

## Complexity Tracking

No constitution violations requiring justification.
