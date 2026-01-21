# Implementation Plan: RAG Index for Semantic Search

**Branch**: `002-rag-index` | **Date**: 2026-01-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-rag-index/spec.md`

## Summary

Add semantic search capabilities to bip artite by indexing paper abstracts as vector embeddings. Users can search by concept ("methods for inferring evolutionary trees") and find similar papers, even without exact keyword matches. The index is ephemeral and rebuildable from JSONL source data.

## Technical Context

**Language/Version**: Go 1.21+ (continuing Phase I)
**Primary Dependencies**: Ollama (local embeddings)
**Storage**: GOB-serialized vector index + metadata in refs.db (ephemeral, gitignored)
**Testing**: go test with real abstract fixtures
**Target Platform**: macOS, Linux (same as Phase I)
**Project Type**: Single CLI application (extending existing `bp` binary)
**Performance Goals**: <1s semantic search, <5 min index build for 6000 papers
**Constraints**: No server process, offline-capable after index build, rebuildable from JSONL
**Scale/Scope**: 10,000 papers with abstracts (94% of collection)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | ✅ Pass | CLI commands with JSON output, composable |
| II. Git-Versionable Architecture | ✅ Pass | Semantic index in gitignored cache, rebuildable from JSONL |
| III. Fail-Fast Philosophy | ✅ Pass | Clear errors for missing index, unavailable Ollama |
| IV. Real Testing (Agentic TDD) | ✅ Pass | Real abstract fixtures from Paperpile exports |
| V. Clean Architecture | ✅ Pass | Separate embedding/index/query concerns |
| VI. Simplicity | ✅ Pass | Single embeddable vector store, minimal dependencies |
| Embeddable Over Client-Server | ✅ Pass | Pure Go vector index, no daemon required |
| CLI Responsiveness | ✅ Pass | Index lazy-loaded only for semantic/similar commands |

**Gate Result**: PASS - All principles satisfied or have mitigation plans.

## Project Structure

### Documentation (this feature)

```text
specs/002-rag-index/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI contracts)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bip/
├── main.go              # Existing - add new subcommands
├── semantic.go          # NEW: bp semantic <query>
├── similar.go           # NEW: bp similar <id>
└── index.go             # NEW: bp index build|check

internal/
├── config/              # Existing
├── reference/           # Existing
├── storage/             # Existing
├── embedding/           # NEW: Embedding generation
│   ├── provider.go      # Provider interface
│   ├── ollama.go        # Ollama implementation
│   └── embedding.go     # Embedding types
├── semantic/            # NEW: Semantic search
│   ├── index.go         # Index build/rebuild + GOB persistence
│   ├── search.go        # Cosine similarity search
│   └── types.go         # SemanticIndex, SearchResult types
└── ...

testdata/
├── abstracts/           # NEW: Test abstracts for embedding tests
│   ├── phylogenetics.json
│   ├── ml_methods.json
│   └── no_abstract.json
└── ...
```

**Structure Decision**: Extend existing Phase I structure with new `internal/embedding/` and `internal/semantic/` packages. Keep separation between embedding generation (provider-agnostic) and semantic search (storage/query).

## Complexity Tracking

> No violations requiring justification. Design adheres to all constitution principles.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | - | - |
