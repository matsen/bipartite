# Tasks: Concept Nodes

**Input**: Design documents from `/specs/006-concept-nodes/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/cli-concept.md

**Tests**: Written during implementation via TDD (per constitution IV). Test tasks are implicit within each implementation task rather than listed separately.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go CLI tool at repository root
- Source code: `internal/`, `cmd/bip/`
- Test fixtures: `testdata/`

---

## Phase 1: Setup

**Purpose**: Project initialization and configuration updates

- [ ] T001 Add ConceptsFile constant (`concepts.jsonl`) in internal/config/config.go
- [ ] T002 [P] Create test fixture directory testdata/concepts/
- [ ] T003 [P] Create test fixture testdata/concepts/test-concepts.jsonl with sample concepts
- [ ] T004 [P] Create test fixture testdata/concepts/test-paper-concept-edges.jsonl with sample edges

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 Create Concept domain type with ID, Name, Aliases, Description fields in internal/concept/concept.go
- [ ] T006 Implement ValidateForCreate() method with ID regex `^[a-z0-9][a-z0-9_-]*$` in internal/concept/concept.go
- [ ] T007 [P] Implement concepts JSONL read (ReadAllConcepts) in internal/storage/concepts_jsonl.go
- [ ] T008 [P] Implement concepts JSONL write (AppendConcept, WriteAllConcepts) in internal/storage/concepts_jsonl.go
- [ ] T009 [P] Implement concepts JSONL find by ID (FindConceptByID) in internal/storage/concepts_jsonl.go
- [ ] T010 Create SQLite schema for concepts table in internal/storage/concepts_sqlite.go
- [ ] T011 Create SQLite schema for concepts_fts virtual table in internal/storage/concepts_sqlite.go
- [ ] T012 Implement RebuildConceptsFromJSONL function in internal/storage/concepts_sqlite.go
- [ ] T013 Update bip rebuild command to include concepts in cmd/bip/rebuild.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Create and Manage Concepts (Priority: P1)

**Goal**: Researchers can define named concepts (methods, phenomena, ideas) with CRUD operations

**Independent Test**: Can be fully tested by adding a concept, listing it, and verifying persistence

### Implementation for User Story 1

- [ ] T014 [US1] Create concept subcommand root in cmd/bip/concept.go
- [ ] T015 [US1] Register concept command in cmd/bip/main.go
- [ ] T016 [US1] Implement `bip concept add` command with --name, --aliases, --description, --human flags in cmd/bip/concept.go
- [ ] T017 [US1] Implement `bip concept get` command with --human flag in cmd/bip/concept.go
- [ ] T018 [US1] Implement `bip concept list` command with --human flag in cmd/bip/concept.go
- [ ] T019 [US1] Implement `bip concept update` command with --name, --aliases, --description, --human flags in cmd/bip/concept.go
- [ ] T020 [US1] Implement `bip concept delete` command with --force, --human flags in cmd/bip/concept.go
- [ ] T021 [US1] Add delete safeguard: count edges before delete, require --force if edges exist; when --force is used, delete all edges pointing to concept in cmd/bip/concept.go

**Checkpoint**: User Story 1 complete - concept CRUD operations fully functional

---

## Phase 4: User Story 2 - Link Papers to Concepts (Priority: P2)

**Goal**: Researchers can tag papers with concepts via edges with relationship types

**Independent Test**: Can be tested by linking a paper to a concept and verifying the edge exists

### Implementation for User Story 2

- [ ] T022 [US2] Add concept ID loading function (loadConceptIDs) in cmd/bip/edge.go
- [ ] T023 [US2] Extend edge add validation to check target against both refs AND concepts in cmd/bip/edge.go
- [ ] T024 [US2] Add warning for non-standard paper-concept relationship types (check against relationship-types.json) in cmd/bip/edge.go

**Checkpoint**: User Story 2 complete - paper-concept edges can be created with validation

---

## Phase 5: User Story 3 - Query Papers by Concept (Priority: P2)

**Goal**: Researchers can find all papers linked to a specific concept, grouped by relationship type

**Independent Test**: Can be tested by querying papers for a concept after creating edges

### Implementation for User Story 3

- [ ] T025 [US3] Add SQLite query for papers by concept ID in internal/storage/concepts_sqlite.go
- [ ] T026 [US3] Implement `bip concept papers` command with --type, --human flags in cmd/bip/concept.go

**Checkpoint**: User Story 3 complete - can query papers by concept

---

## Phase 6: User Story 4 - Query Concepts by Paper (Priority: P3)

**Goal**: Researchers can see what concepts a specific paper relates to

**Independent Test**: Can be tested by querying concepts for a paper after creating edges

### Implementation for User Story 4

- [ ] T027 [US4] Add SQLite query for concepts by paper ID in internal/storage/concepts_sqlite.go
- [ ] T028 [US4] Add paper subcommand if not exists, then implement `bip paper concepts` command with --type, --human flags in cmd/bip/paper.go

**Checkpoint**: User Story 4 complete - can query concepts by paper

---

## Phase 7: User Story 5 - Merge Duplicate Concepts (Priority: P3)

**Goal**: Researchers can merge duplicate concepts, transferring all edges to the surviving concept

**Independent Test**: Can be tested by creating two concepts, linking papers to both, merging, and verifying edges point to survivor

### Implementation for User Story 5

- [ ] T029 [US5] Implement edge update function to change target_id in internal/storage/edges_jsonl.go
- [ ] T030 [US5] Implement merge logic in cmd/bip/concept.go:
  - Update all edges where target_id = source_concept to point to target_concept
  - Detect duplicate edges (same source_id + target_id + relationship_type) and keep only the one with earlier created_at
  - Add source concept's aliases to target concept's aliases list
- [ ] T031 [US5] Implement `bip concept merge` command with --human flag in cmd/bip/concept.go

**Checkpoint**: User Story 5 complete - concept merge fully functional

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T032 [P] Add concept search by text (FTS) function in internal/storage/concepts_sqlite.go
- [ ] T033 Run quickstart.md validation - verify all documented commands work as specified
- [ ] T034 Verify exit codes match CLI contract (0 success, 1 general error, 2 data error, 3 validation error)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on T001 (config) - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion
- **User Story 2 (Phase 4)**: Depends on Foundational completion; integrates with US1 (needs concepts to exist)
- **User Story 3 (Phase 5)**: Depends on Foundational completion; requires edges to exist (US2)
- **User Story 4 (Phase 6)**: Depends on Foundational completion; requires edges to exist (US2)
- **User Story 5 (Phase 7)**: Depends on Foundational completion; requires US1 (concepts) and US2 (edges)
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Independent - can start after Foundational
- **User Story 2 (P2)**: Logically depends on US1 (concepts must exist to link to)
- **User Story 3 (P2)**: Logically depends on US2 (edges must exist to query)
- **User Story 4 (P3)**: Logically depends on US2 (edges must exist to query)
- **User Story 5 (P3)**: Depends on US1 (needs concepts) and US2 (needs edges to transfer)

### Within Each User Story

- Models/types before storage layer
- Storage layer before CLI commands
- Core implementation before edge cases
- Story complete before moving to next priority

### Parallel Opportunities

**Setup Phase (parallel):**
- T002, T003, T004 can all run in parallel (different files)

**Foundational Phase (parallel after T005-T006):**
- T007, T008, T009 can run in parallel (different functions in same file, but independent)
- T010, T011 can run in parallel (schema creation)

**User Story 1 (sequential):**
- T014-T021 mostly sequential (same file, building up commands)

**Cross-story parallelism:**
- Once Foundational is done, US1 can start
- Once US1 is done, US2 can start
- Once US2 is done, US3 and US4 can run in parallel (different query directions)
- US5 can start after US2

---

## Parallel Example: Setup Phase

```bash
# Launch all test fixture tasks together:
Task: "Create test fixture directory testdata/concepts/"
Task: "Create test fixture testdata/concepts/test-concepts.jsonl"
Task: "Create test fixture testdata/concepts/test-paper-concept-edges.jsonl"
```

## Parallel Example: Foundational Phase

```bash
# After Concept type created (T005-T006), launch storage tasks:
Task: "Implement concepts JSONL read in internal/storage/concepts_jsonl.go"
Task: "Implement concepts JSONL write in internal/storage/concepts_jsonl.go"
Task: "Implement concepts JSONL find by ID in internal/storage/concepts_jsonl.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test `bip concept add`, `get`, `list`, `update`, `delete`
5. Deploy/demo if ready - concepts can be created and managed

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test CRUD → Validate (MVP!)
3. Add User Story 2 → Test edge creation → Validate
4. Add User Story 3 → Test concept->papers query → Validate
5. Add User Story 4 → Test paper->concepts query → Validate
6. Add User Story 5 → Test merge → Validate
7. Each story adds value without breaking previous stories

### Recommended Execution Order (Solo Developer)

1. T001-T004 (Setup)
2. T005-T013 (Foundational)
3. T014-T021 (US1 - CRUD)
4. T022-T024 (US2 - Edge linking)
5. T025-T026 (US3 - Papers by concept query)
6. T027-T028 (US4 - Concepts by paper query)
7. T029-T031 (US5 - Merge)
8. T032-T034 (Polish)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All commands must support `--human` flag for human-readable output (JSON default)
- Exit codes per CLI contract: 0 (success), 1 (general error), 2 (data error), 3 (validation error)
