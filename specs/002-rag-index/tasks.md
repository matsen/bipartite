# Tasks: RAG Index for Semantic Search

**Input**: Design documents from `/specs/002-rag-index/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md

**Tests**: No test tasks included (not explicitly requested in spec).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions (from plan.md)

- CLI commands: `cmd/bp/`
- Embedding package: `internal/embedding/`
- Semantic package: `internal/semantic/`
- Test fixtures: `testdata/abstracts/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and new package structure

- [X] T001 Create embedding package directory structure at internal/embedding/
- [X] T002 Create semantic package directory structure at internal/semantic/
- [X] T003 [P] Create test fixtures directory at testdata/abstracts/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core embedding and storage infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Embedding Infrastructure

- [X] T004 Define Embedding types and interfaces in internal/embedding/embedding.go
- [X] T005 Define Provider interface in internal/embedding/provider.go
- [X] T006 Implement Ollama provider in internal/embedding/ollama.go (HTTP client for localhost:11434)
- [X] T007 [P] Implement Ollama availability check (GET /api/tags) in internal/embedding/ollama.go
- [X] T008 [P] Implement model existence check in internal/embedding/ollama.go

### Semantic Index Infrastructure

- [X] T009 Define SemanticIndex struct and types in internal/semantic/types.go
- [X] T010 Implement cosine similarity function in internal/semantic/search.go
- [X] T011 Implement GOB serialization (save/load) for SemanticIndex in internal/semantic/index.go
- [X] T012 Add embedding_metadata table schema to internal/storage/schema.go

### Test Fixtures

- [X] T013 [P] Create phylogenetics test fixture in testdata/abstracts/phylogenetics.json
- [X] T014 [P] Create ML methods test fixture in testdata/abstracts/ml_methods.json
- [X] T015 [P] Create no-abstract test fixture in testdata/abstracts/no_abstract.json

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Semantic Search by Concept (Priority: P1) 🎯 MVP

**Goal**: Enable semantic search via `bp semantic <query>` command with similarity scoring

**Independent Test**: Import references with abstracts, build the semantic index, run a conceptual query, verify that results include semantically related papers that wouldn't match keyword search.

### Implementation for User Story 1

- [X] T016 [US1] Implement index build logic (iterate papers, call embedding provider, populate SemanticIndex) in internal/semantic/builder.go
- [X] T017 [US1] Implement progress reporting for index build (to stderr) in internal/semantic/builder.go
- [X] T018 [US1] Implement semantic search function (query embedding + cosine similarity ranking) in internal/semantic/search.go
- [X] T019 [US1] Implement threshold filtering and limit in internal/semantic/search.go
- [X] T020 [US1] Create bp index build command with --no-progress and --human flags in cmd/bp/index.go
- [X] T021 [US1] Create bp semantic command with --limit, --threshold, --human flags in cmd/bp/semantic.go
- [X] T022 [US1] Implement JSON output format for bp semantic per contracts/cli.md
- [X] T023 [US1] Implement human-readable output format for bp semantic per contracts/cli.md
- [X] T024 [US1] Add error handling: empty query, index not found, Ollama not available in cmd/bp/semantic.go
- [X] T025 [US1] Wire index build command to main.go and register subcommand in cmd/bp/index.go
- [X] T026 [US1] Wire semantic command to main.go and register subcommand in cmd/bp/semantic.go

**Checkpoint**: User Story 1 complete - semantic search works with index build

---

## Phase 4: User Story 2 - Find Similar Papers (Priority: P2)

**Goal**: Enable finding similar papers via `bp similar <id>` command

**Independent Test**: Import references, build index, select a paper, run similar-papers command, verify results are topically related to the source paper.

### Implementation for User Story 2

- [X] T027 [US2] Implement find-similar function (lookup paper embedding, compute similarities) in internal/semantic/search.go
- [X] T028 [US2] Create bp similar command with --limit and --human flags in cmd/bp/similar.go
- [X] T029 [US2] Implement JSON output format for bp similar per contracts/cli.md
- [X] T030 [US2] Implement human-readable output format for bp similar per contracts/cli.md
- [X] T031 [US2] Add error handling: paper not found, paper has no abstract, index not found in cmd/bp/similar.go
- [X] T032 [US2] Wire similar command to main.go and register subcommand in cmd/bp/similar.go

**Checkpoint**: User Stories 1 AND 2 work independently

---

## Phase 5: User Story 3 - Rebuild Semantic Index (Priority: P3)

**Goal**: Enable full index rebuild with statistics reporting via `bp index build`

**Independent Test**: Import new papers, run index rebuild, verify new papers appear in semantic search results.

### Implementation for User Story 3

- [X] T033 [US3] Implement skip logic for papers without abstracts (with count tracking) in internal/semantic/builder.go
- [X] T034 [US3] Implement skip logic for abstracts <50 characters (with warning) in internal/semantic/builder.go
- [X] T035 [US3] Implement build statistics collection (indexed, skipped, duration, size) in internal/semantic/builder.go
- [X] T036 [US3] Implement JSON output for bp index build per contracts/cli.md
- [X] T037 [US3] Implement human-readable output for bp index build per contracts/cli.md
- [X] T038 [US3] Store embedding metadata (paper_id, model_name, indexed_at, abstract_hash) in refs.db

**Checkpoint**: User Stories 1, 2, AND 3 work independently

---

## Phase 6: User Story 4 - Check Index Health (Priority: P4)

**Goal**: Enable index health verification via `bp index check` command

**Independent Test**: Build index, run check command, verify it reports index status and any gaps.

### Implementation for User Story 4

- [X] T039 [US4] Implement index health check logic (compare indexed vs papers with abstracts) in cmd/bp/index.go
- [X] T040 [US4] Implement missing papers detection in cmd/bp/index.go
- [X] T041 [US4] Create bp index check command with --human flag in cmd/bp/index.go
- [X] T042 [US4] Implement JSON output for bp index check per contracts/cli.md
- [X] T043 [US4] Implement human-readable output for bp index check per contracts/cli.md
- [X] T044 [US4] Add exit codes per contracts/cli.md (0=healthy, 2=not found, 6=stale)

**Checkpoint**: All user stories complete and independently functional

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, error handling consistency, and edge cases

- [X] T045 [P] Verify all exit codes match contracts/cli.md specification across all commands
- [X] T046 [P] Verify all error messages match contracts/cli.md standard errors
- [X] T047 Implement lazy loading for SemanticIndex (load only when needed) in internal/semantic/index.go
- [X] T048 Verify index file is in .bipartite/cache/semantic.gob (gitignored location)
- [X] T049 Run quickstart.md validation workflow manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can proceed sequentially in priority order (P1 → P2 → P3 → P4)
  - US2, US3, US4 each add incremental functionality to US1's foundation
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Core semantic search
- **User Story 2 (P2)**: Can start after US1 - Reuses embedding lookup and similarity search
- **User Story 3 (P3)**: Can start after US1 - Enhances index build with statistics
- **User Story 4 (P4)**: Can start after US3 - Uses index metadata for health check

### Within Each User Story

- Internal package work before CLI commands
- Core logic before output formatting
- Happy path before error handling

### Parallel Opportunities

Within **Phase 2 (Foundational)**:
```
Parallel Group A (Types):
- T004: Embedding types
- T009: SemanticIndex types

Parallel Group B (After types):
- T005, T006, T007, T008: Ollama provider (sequential within)
- T010, T011: Semantic search/index (sequential within)

Parallel Group C (Test fixtures - independent):
- T013: phylogenetics.json
- T014: ml_methods.json
- T015: no_abstract.json
```

Within **Phase 3 (US1)**:
```
Sequential flow:
T016 → T017 (index build) → T018 → T019 (search) → T020 → T021 (CLI) → T022, T023 (output) → T024 (errors) → T025, T026 (wiring)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test semantic search end-to-end
5. Demo: `bp index build` → `bp semantic "phylogenetics"`

### Incremental Delivery

1. **Setup + Foundational** → Foundation ready
2. **Add User Story 1** → `bp semantic` works → Demo (MVP!)
3. **Add User Story 2** → `bp similar` works → Demo
4. **Add User Story 3** → `bp index build` with statistics → Demo
5. **Add User Story 4** → `bp index check` works → Demo
6. Each story adds value without breaking previous stories

---

## Notes

- All commands output JSON by default, `--human` for readable format (Phase I pattern)
- Index stored in `.bipartite/cache/semantic.gob` (gitignored, ephemeral)
- Embedding model: `nomic-embed-text` via Ollama (768 dimensions, 8K context)
- Ollama must be running for index build and semantic search
- Papers without abstracts are skipped during indexing (tracked in statistics)
- Minimum abstract length: 50 characters
