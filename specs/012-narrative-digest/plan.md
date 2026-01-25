# Implementation Plan: Narrative Digest

**Branch**: `012-narrative-digest` | **Date**: 2026-01-25 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/012-narrative-digest/spec.md`

## Summary

Implement a `/bip.narrative` slash command that generates thematic, prose-style narrative digests from GitHub activity. This is a two-part feature:
1. **Prerequisite**: Change `bip digest` default to preview-only (currently posts to Slack by default)
2. **Main feature**: Create slash command that uses `bip digest` output, reads channel config from nexus, and generates themed markdown using Claude Code's LLM

The implementation uses the slash command pattern exclusively for LLM generation—no LLM calls from Go code.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage)
**Storage**: JSONL (source of truth) + SQLite (ephemeral query layer)
**Testing**: go test (unit, integration)
**Target Platform**: macOS, Linux
**Project Type**: Single CLI project
**Performance Goals**: N/A (human-interactive command)
**Constraints**: No LLM calls from Go code; use slash command pattern
**Scale/Scope**: Single-user CLI tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | CLI is primary interface; slash command for agent interaction |
| II. Git-Versionable Architecture | ✅ PASS | Output is markdown files in nexus; config is markdown; no new database tables |
| III. Fail-Fast Philosophy | ✅ PASS | Missing config file produces helpful error; malformed config rejected |
| IV. Real Testing | ✅ PASS | Tests will use real fixture data from nexus narrative configs |
| V. Clean Architecture | ✅ PASS | Clear separation: Go CLI (data fetching) vs slash command (LLM generation) |
| VI. Simplicity | ✅ PASS | Minimal changes: flag inversion + new slash command; no new Go LLM code |

**All gates pass. Proceeding to Phase 0.**

## Project Structure

### Documentation (this feature)

```text
specs/012-narrative-digest/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (if applicable)
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bip/
└── digest.go            # MODIFY: Add --post flag, change default behavior

.claude/commands/
├── bip.digest.md        # MODIFY: Update to match new CLI behavior
└── bip.narrative.md     # NEW: Slash command for narrative generation

internal/flow/
├── llm.go               # MODIFY: Add --verbose summarization support (optional)
└── gh.go                # Existing: GitHub API helpers

# Narrative config/output in nexus (separate repo)
nexus/narrative/
├── preferences.md       # EXISTING: Shared defaults
├── dasm2.md             # EXISTING: Channel config example
└── dasm2/
    └── YYYY-MM-DD.md    # OUTPUT: Generated digests
```

**Structure Decision**: Single project structure; primary changes are flag modification in existing file and new slash command file.

## Constitution Check (Post-Design)

*Re-evaluated after Phase 1 design completion.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ PASS | CLI commands for data; slash command for generation |
| II. Git-Versionable Architecture | ✅ PASS | All config/output in markdown; no database changes |
| III. Fail-Fast Philosophy | ✅ PASS | Errors on missing config, no activity, malformed input |
| IV. Real Testing | ✅ PASS | Will test with real nexus configs; integration tests for digest |
| V. Clean Architecture | ✅ PASS | Go handles data; slash command handles LLM; clear separation |
| VI. Simplicity | ✅ PASS | Breaking change (--post required) per constitution principle |

**All gates pass post-design.**

## Complexity Tracking

> No Constitution Check violations requiring justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
