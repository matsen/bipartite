# Tasks: Knowledge Graph

**Input**: Design documents from `/specs/003-knowledge-graph/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/cli.md ✓, quickstart.md ✓

**Tests**: Tests will be included following existing project patterns (unit tests alongside implementation).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md project structure:
- **CLI commands**: `cmd/bp/`
- **Domain types**: `internal/edge/`
- **Storage**: `internal/storage/`
- **Integration tests**: `tests/integration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the edge domain type and basic package structure

- [ ] T001 Create internal/edge/ package directory structure
- [ ] T002 [P] Define Edge domain type with JSON tags in internal/edge/edge.go
- [ ] T003 [P] Add Edge validation methods (ValidateForCreate) in internal/edge/edge.go

**Checkpoint**: Edge domain type ready for storage and CLI layers

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core storage infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Implement JSONL read/write for edges in internal/storage/edges_jsonl.go
- [ ] T005 Add unit tests for edges JSONL operations in internal/storage/edges_jsonl_test.go
- [ ] T006 [P] Implement SQLite edge index schema in internal/storage/edges_sqlite.go
- [ ] T007 [P] Add unit tests for edges SQLite operations in internal/storage/edges_sqlite_test.go
- [ ] T008 Create bp edge parent command with --json flag support in cmd/bp/edge.go
- [ ] T009 Register edge command in cmd/bp/main.go
- [ ] T010 Extend bp rebuild to rebuild edge index from edges.jsonl in cmd/bp/rebuild.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Add Edges from External Tool (Priority: P1) 🎯 MVP

**Goal**: Enable external tools to add edges via `bp edge add` and `bp edge import` commands

**Independent Test**: Can be tested by running `bp edge add` with edge data and verifying the edge is stored and retrievable via JSONL inspection

### Implementation for User Story 1

- [ ] T011 [US1] Implement paper ID existence validation helper in internal/storage/edges_jsonl.go
- [ ] T012 [US1] Implement bp edge add subcommand with --source, --target, --type, --summary flags in cmd/bp/edge.go
- [ ] T013 [US1] Add edge upsert logic (update existing edge if same source/target/type) in internal/storage/edges_jsonl.go
- [ ] T014 [US1] Implement human-readable output for edge add in cmd/bp/edge.go
- [ ] T015 [US1] Implement JSON output for edge add when --json flag provided in cmd/bp/edge.go
- [ ] T016 [US1] Add unit tests for edge add command in cmd/bp/edge_test.go
- [ ] T017 [P] [US1] Implement bp edge import subcommand for JSONL bulk import in cmd/bp/edge.go
- [ ] T018 [US1] Add import progress reporting (added/updated/skipped counts) in cmd/bp/edge.go
- [ ] T019 [US1] Add error handling for missing source/target papers during import in cmd/bp/edge.go
- [ ] T020 [US1] Add unit tests for edge import command in cmd/bp/edge_test.go
- [ ] T021 [US1] Create integration test for edge add/import workflow in tests/integration/edge_test.go

**Checkpoint**: At this point, external tools can add edges to the knowledge graph. User Story 1 should be fully functional and testable independently.

---

## Phase 4: User Story 2 - Query Edges for a Paper (Priority: P2)

**Goal**: Enable researchers to list all edges connected to a specific paper with direction filtering

**Independent Test**: Can be tested by adding edges, then querying with `bp edge list <paper-id>` and verifying correct results

### Implementation for User Story 2

- [ ] T022 [US2] Implement SQLite query for outgoing edges (by source_id) in internal/storage/edges_sqlite.go
- [ ] T023 [US2] Implement SQLite query for incoming edges (by target_id) in internal/storage/edges_sqlite.go
- [ ] T024 [US2] Implement bp edge list subcommand with paper-id argument in cmd/bp/edge.go
- [ ] T025 [US2] Add --incoming flag to filter for edges where paper is target in cmd/bp/edge.go
- [ ] T026 [US2] Add --all flag to show both incoming and outgoing edges in cmd/bp/edge.go
- [ ] T027 [US2] Implement human-readable output with direction indicators in cmd/bp/edge.go
- [ ] T028 [US2] Implement JSON output for edge list when --json flag provided in cmd/bp/edge.go
- [ ] T029 [US2] Handle case where paper has no edges (empty result with message) in cmd/bp/edge.go
- [ ] T030 [US2] Add unit tests for edge list command in cmd/bp/edge_test.go
- [ ] T031 [US2] Add integration test for edge list workflow in tests/integration/edge_test.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently. Researchers can add and query edges.

---

## Phase 5: User Story 3 - Search Edges by Relationship Type (Priority: P3)

**Goal**: Enable filtering edges by relationship type to explore specific kinds of relationships

**Independent Test**: Can be tested by adding edges with different types, then filtering with `bp edge search --type <type>` and verifying correct filtering

### Implementation for User Story 3

- [ ] T032 [US3] Implement SQLite query for edges by relationship_type in internal/storage/edges_sqlite.go
- [ ] T033 [US3] Implement bp edge search subcommand with --type flag in cmd/bp/edge.go
- [ ] T034 [US3] Implement human-readable output for search results in cmd/bp/edge.go
- [ ] T035 [US3] Implement JSON output for edge search when --json flag provided in cmd/bp/edge.go
- [ ] T036 [US3] Handle empty search results gracefully in cmd/bp/edge.go
- [ ] T037 [US3] Add unit tests for edge search command in cmd/bp/edge_test.go
- [ ] T038 [US3] Add integration test for edge search workflow in tests/integration/edge_test.go

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently.

---

## Phase 6: User Story 4 - Export Edges (Priority: P4)

**Goal**: Enable exporting edges to JSONL format for backup and sharing

**Independent Test**: Can be tested by adding edges, exporting with `bp edge export`, and verifying the export contains all edges with correct data

### Implementation for User Story 4

- [ ] T039 [US4] Implement SQLite query for all edges in internal/storage/edges_sqlite.go
- [ ] T040 [US4] Implement SQLite query for edges involving a specific paper in internal/storage/edges_sqlite.go
- [ ] T041 [US4] Implement bp edge export subcommand writing to stdout in cmd/bp/edge.go
- [ ] T042 [US4] Add --paper flag to filter export by paper ID in cmd/bp/edge.go
- [ ] T043 [US4] Ensure export format matches import format (round-trip compatibility) in cmd/bp/edge.go
- [ ] T044 [US4] Add unit tests for edge export command in cmd/bp/edge_test.go
- [ ] T045 [US4] Add integration test for export/import round-trip in tests/integration/edge_test.go

**Checkpoint**: All user stories should now be independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Maintenance commands and final integration

- [ ] T046 Extend bp groom to detect and report orphaned edges in cmd/bp/groom.go
- [ ] T047 Add --fix flag to bp groom for orphaned edge removal in cmd/bp/groom.go
- [ ] T048 Extend bp check to verify edge integrity in cmd/bp/check.go
- [ ] T049 Add unit tests for groom edge functionality in cmd/bp/groom_test.go
- [ ] T050 Add unit tests for check edge functionality in cmd/bp/check_test.go
- [ ] T051 Run go fmt ./... and go vet ./... for code quality
- [ ] T052 Validate quickstart.md examples work end-to-end
- [ ] T053 Final integration test covering full workflow in tests/integration/edge_test.go

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P4)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 4 (P4)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Storage layer before CLI commands
- Core implementation before flags/options
- Human output before JSON output
- Unit tests alongside implementation
- Integration tests after all commands work

### Parallel Opportunities

**Phase 1 (Setup)**:
- T002 and T003 can run in parallel

**Phase 2 (Foundational)**:
- T006 and T007 can run in parallel with T004 and T005

**After Foundational**:
- All four user stories can be worked on in parallel by different developers
- Within US1: T017 can start while T012-T016 are in progress

---

## Parallel Example: User Story 1

```bash
# Sequential: Add command first
Task: "Implement bp edge add subcommand in cmd/bp/edge.go"
Task: "Add unit tests for edge add command"

# Then parallel: Import can start
Task: "Implement bp edge import subcommand in cmd/bp/edge.go"
Task: "Add unit tests for edge import command"
```

## Parallel Example: Full Feature

```bash
# After Phase 2 completes, launch all user stories in parallel:
Agent 1: User Story 1 (edge add, edge import)
Agent 2: User Story 2 (edge list)
Agent 3: User Story 3 (edge search)
Agent 4: User Story 4 (edge export)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T010)
3. Complete Phase 3: User Story 1 (T011-T021)
4. **STOP and VALIDATE**: Test edge add/import independently
5. Deploy/demo if ready - external tools can now add edges

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy (MVP!)
3. Add User Story 2 → Test independently → Researchers can query edges
4. Add User Story 3 → Test independently → Relationship type filtering works
5. Add User Story 4 → Test independently → Full export/backup capability
6. Polish phase → Production-ready with maintenance commands

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Exit codes per CLI contract: 0=success, 1=source not found, 2=target not found, 3=invalid args
- All commands support --json flag (agent-first design per constitution)
