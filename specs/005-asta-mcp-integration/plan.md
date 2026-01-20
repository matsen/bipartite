# Implementation Plan: ASTA MCP Integration

**Branch**: `005-asta-mcp-integration` | **Date**: 2026-01-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-asta-mcp-integration/spec.md`

## Summary

Implement CLI wrappers for Allen AI's ASTA (Academic Search Tool API) MCP service, providing read-only paper search, snippet search, and citation exploration capabilities that complement the existing `bp s2` commands.

## Technical Context

**Language/Version**: Go 1.25.5 (matches existing codebase)
**Primary Dependencies**: spf13/cobra (CLI), golang.org/x/time/rate (rate limiting), joho/godotenv (env loading)
**Storage**: N/A (ASTA is read-only external API, no local persistence)
**Testing**: go test with real API calls (following Agentic TDD)
**Target Platform**: macOS and Linux CLI
**Project Type**: Single project (CLI extension)
**Performance Goals**: 10 req/sec (ASTA rate limit)
**Constraints**: Must not exceed ASTA rate limits; clear auth errors for missing API key
**Scale/Scope**: 7 new CLI commands under `bp asta` parent

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✓ PASS | JSON default output, --human flag, bash-composable CLI |
| II. Git-Versionable Architecture | ✓ PASS | No local state; ASTA is read-only external API |
| III. Fail-Fast Philosophy | ✓ PASS | Clear errors for missing ASTA_API_KEY, API failures |
| IV. Real Testing | ✓ PASS | Will use real ASTA API calls with test queries |
| V. Clean Architecture | ✓ PASS | internal/asta package with clear separation |
| VI. Simplicity | ✓ PASS | Direct HTTP calls, no MCP client library overhead |

**Gate Result**: PASS - No violations requiring justification

## Project Structure

### Documentation (this feature)

```text
specs/005-asta-mcp-integration/
├── plan.md              # This file
├── research.md          # Phase 0 output (MCP protocol details)
├── data-model.md        # Phase 1 output (ASTA types)
├── contracts/           # Phase 1 output (API contracts)
├── quickstart.md        # Phase 1 output (usage examples)
└── tasks.md             # Phase 2 output (implementation tasks)
```

### Source Code (repository root)

```text
internal/
├── asta/                # New package for ASTA MCP client
│   ├── client.go        # MCP HTTP client with rate limiting
│   ├── types.go         # Request/response types
│   └── errors.go        # Error types

cmd/bp/
├── asta.go              # Parent command with --human flag
├── asta_search.go       # bp asta search <query>
├── asta_snippet.go      # bp asta snippet <query>
├── asta_paper.go        # bp asta paper <id>
├── asta_citations.go    # bp asta citations <id>
├── asta_references.go   # bp asta references <id>
├── asta_author.go       # bp asta author <name>
└── asta_author_papers.go # bp asta author-papers <id>
```

**Structure Decision**: Follows existing `internal/s2/` and `cmd/bp/s2*.go` patterns for consistency. No new test directories needed—tests will be in standard `*_test.go` files.

## Complexity Tracking

> No violations requiring justification—all constitution checks pass.
