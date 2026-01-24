# Tasks: Projects and Repos as Knowledge Graph Nodes

**Input**: Design documents from `/specs/011-repo-nodes/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Included per Constitution principle IV (Real Testing with fixtures)

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- CLI commands: `cmd/bip/`
- Domain packages: `internal/project/`, `internal/repo/`, `internal/github/`
- Storage: `internal/storage/`
- Config: `internal/config/`
- Test fixtures: `testdata/projects/`, `testdata/repos/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and config updates

- [X] T001 Add ProjectsFile and ReposFile constants in internal/config/config.go
- [X] T002 [P] Add ProjectsPath() and ReposPath() functions in internal/config/config.go
- [X] T003 [P] Create testdata/projects/ directory with test fixtures
- [X] T004 [P] Create testdata/repos/ directory with test fixtures

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Domain types and storage layer that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Domain Types

- [X] T005 [P] Create internal/project/ package directory
- [X] T006 [P] Create internal/repo/ package directory
- [X] T007 [P] Create internal/github/ package directory
- [X] T008 Implement Project struct and validation in internal/project/project.go
- [X] T009 [P] Implement Repo struct and validation in internal/repo/repo.go
- [X] T010 [P] Implement GitHub API client in internal/github/client.go

### Storage Layer (JSONL)

- [X] T011 [P] Implement ReadAllProjects() in internal/storage/projects_jsonl.go
- [X] T012 [P] Implement AppendProject(), WriteAllProjects() in internal/storage/projects_jsonl.go
- [X] T013 [P] Implement FindProjectByID(), DeleteProjectFromSlice() in internal/storage/projects_jsonl.go
- [X] T014 [P] Implement ReadAllRepos() in internal/storage/repos_jsonl.go
- [X] T015 [P] Implement AppendRepo(), WriteAllRepos() in internal/storage/repos_jsonl.go
- [X] T016 [P] Implement FindRepoByID(), FindRepoByGitHubURL() in internal/storage/repos_jsonl.go

### Storage Layer (SQLite)

- [X] T017 [P] Add projects table schema in internal/storage/sqlite.go
- [X] T018 [P] Add repos table schema in internal/storage/sqlite.go
- [X] T019 Implement RebuildProjectsFromJSONL() in internal/storage/projects_sqlite.go
- [X] T020 [P] Implement RebuildReposFromJSONL() in internal/storage/repos_sqlite.go
- [X] T021 [P] Implement GetProjectByID(), GetAllProjects() in internal/storage/projects_sqlite.go
- [X] T022 [P] Implement GetRepoByID(), GetAllRepos(), GetReposByProject() in internal/storage/repos_sqlite.go

### Unit Tests for Foundational

- [X] T023 [P] Write tests for Project validation in internal/project/project_test.go
- [X] T024 [P] Write tests for Repo validation in internal/repo/repo_test.go
- [X] T025 [P] Write tests for GitHub client in internal/github/client_test.go
- [X] T026 [P] Write tests for projects JSONL in internal/storage/projects_jsonl_test.go
- [X] T027 [P] Write tests for repos JSONL in internal/storage/repos_jsonl_test.go
- [X] T028 [P] Write tests for projects SQLite in internal/storage/projects_sqlite_test.go
- [X] T029 [P] Write tests for repos SQLite in internal/storage/repos_sqlite_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Create a Project (Priority: P1) 🎯 MVP

**Goal**: Researcher can create, view, list, update, and delete projects

**Independent Test**: `bip project add test-proj --name "Test" && bip project get test-proj && bip project delete test-proj`

### Tests for User Story 1

- [X] T030 [P] [US1] Integration test for project CRUD in tests/integration/project_test.go

### Implementation for User Story 1

- [X] T031 [US1] Implement `bip project add` command in cmd/bip/project.go
- [X] T032 [US1] Implement `bip project get` command in cmd/bip/project.go
- [X] T033 [US1] Implement `bip project list` command in cmd/bip/project.go
- [X] T034 [US1] Implement `bip project update` command in cmd/bip/project.go
- [X] T035 [US1] Implement `bip project delete` command in cmd/bip/project.go
- [X] T036 [US1] Add global ID collision check (project ID vs paper/concept IDs) in cmd/bip/project.go
- [X] T037 [US1] Add human-readable output formatting for project commands in cmd/bip/project.go

**Checkpoint**: User Story 1 complete - can create/manage projects independently

---

## Phase 4: User Story 2 - Add Repos to a Project (Priority: P1)

**Goal**: Researcher can add GitHub repos to projects, with auto-fetched metadata

**Independent Test**: `bip repo add matsen/bipartite --project test-proj && bip repo get bipartite && bip project repos test-proj`

### Tests for User Story 2

- [X] T038 [P] [US2] Integration test for repo CRUD in tests/integration/project_test.go

### Implementation for User Story 2

- [X] T039 [US2] Implement GitHub URL parsing (full URL and org/repo shorthand) in internal/github/client.go
- [X] T040 [US2] Implement `bip repo add` command (GitHub mode) in cmd/bip/repo.go
- [X] T041 [US2] Implement `bip repo add --manual` command in cmd/bip/repo.go
- [X] T042 [US2] Implement `bip repo get` command in cmd/bip/repo.go
- [X] T043 [US2] Implement `bip repo list` and `bip repo list --project` commands in cmd/bip/repo.go
- [X] T044 [US2] Implement `bip repo update` command in cmd/bip/repo.go
- [X] T045 [US2] Implement `bip repo delete` command in cmd/bip/repo.go
- [X] T046 [US2] Implement `bip project repos` command in cmd/bip/project.go
- [X] T047 [US2] Add one-project-per-repo validation (unique GitHub URL) in cmd/bip/repo.go
- [X] T048 [US2] Add human-readable output formatting for repo commands in cmd/bip/repo.go

**Checkpoint**: User Story 2 complete - can add repos with GitHub metadata

---

## Phase 5: User Story 3 - Link Concepts to Projects (Priority: P1)

**Goal**: Researcher can create concept↔project edges, with validation preventing paper↔project and *↔repo edges

**Independent Test**: `bip edge add --source concept:vi --target project:test-proj --type implemented-in --summary "Test"`

### Tests for User Story 3

- [X] T049 [P] [US3] Integration test for concept↔project edges in tests/integration/project_test.go
- [X] T050 [P] [US3] Test rejection of paper↔project edges in tests/integration/project_test.go
- [X] T051 [P] [US3] Test rejection of *↔repo edges in tests/integration/project_test.go

### Implementation for User Story 3

- [X] T052 [US3] Add type prefix parsing (concept:, project:, repo:) in cmd/bip/edge.go
- [X] T053 [US3] Extend validateEdgeEndpoints() to handle projects in cmd/bip/edge.go
- [X] T054 [US3] Add loadProjectIDSet() helper in cmd/bip/edge.go
- [X] T055 [US3] Add validation to reject paper↔project edges in cmd/bip/edge.go
- [X] T056 [US3] Add validation to reject *↔repo edges in cmd/bip/edge.go
- [X] T057 [US3] Add concept-project relationship types to validation in cmd/bip/edge.go
- [X] T058 [US3] Implement `bip project concepts` command in cmd/bip/project.go
- [X] T059 [US3] Add GetConceptsByProject() query in internal/storage/edges_sqlite.go
- [X] T060 [US3] Update `bip edge list` to support --project flag in cmd/bip/edge.go

**Checkpoint**: User Story 3 complete - can link concepts to projects with proper validation

---

## Phase 6: User Story 5 - View Project Graph Neighborhood (Priority: P2)

**Goal**: Researcher can query papers relevant to a project via transitive concept links

**Independent Test**: `bip project papers test-proj` (returns papers linked to concepts linked to project)

### Tests for User Story 5

- [X] T061 [P] [US5] Integration test for transitive paper query in tests/integration/project_test.go

### Implementation for User Story 5

- [X] T062 [US5] Implement GetPapersByProjectTransitive() in internal/storage/edges_sqlite.go
- [X] T063 [US5] Implement `bip project papers` command in cmd/bip/project.go
- [X] T064 [US5] Format output to show concept as "bridge" explanation in cmd/bip/project.go

**Checkpoint**: User Story 5 complete - can view project's full literature neighborhood

---

## Phase 7: User Story 4 - Refresh Repo Metadata (Priority: P3)

**Goal**: Researcher can update repo metadata from GitHub after changes

**Independent Test**: `bip repo refresh bipartite` (re-fetches metadata)

### Tests for User Story 4

- [X] T065 [P] [US4] Integration test for repo refresh in tests/integration/project_test.go (GitHub refresh requires real API)
- [X] T066 [P] [US4] Test error for manual repo refresh in tests/integration/project_test.go

### Implementation for User Story 4

- [X] T067 [US4] Implement `bip repo refresh` command in cmd/bip/repo.go
- [X] T068 [US4] Add error handling for manual repos (no GitHub URL) in cmd/bip/repo.go

**Checkpoint**: User Story 4 complete - can refresh GitHub metadata

---

## Phase 8: Integration (rebuild, check)

**Purpose**: Integrate projects/repos into existing bip commands

### Tests

- [X] T069 [P] Integration test for rebuild with projects/repos in tests/integration/project_test.go
- [X] T070 [P] Integration test for check with project/repo constraints in tests/integration/project_test.go

### Implementation

- [X] T071 Extend `bip rebuild` to include projects table in cmd/bip/rebuild.go
- [X] T072 Extend `bip rebuild` to include repos table in cmd/bip/rebuild.go
- [X] T073 Extend `bip check` to validate repos reference valid projects in cmd/bip/check.go
- [X] T074 Extend `bip check` to detect orphaned project edges in cmd/bip/check.go
- [X] T075 Extend `bip check` to detect invalid paper↔project or *↔repo edges in cmd/bip/check.go
- [X] T076 Update cascade delete in project delete to remove repos and edges in cmd/bip/project.go

**Checkpoint**: Integration complete - projects/repos work with all existing bip commands

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, final validation

- [X] T077 [P] Update README.md with project/repo command documentation
- [X] T078 [P] Run `go fmt ./...` and `go vet ./...`
- [X] T079 Run full test suite `go test ./...`
- [X] T080 Validate quickstart.md workflow end-to-end
- [X] T081 [P] Add relationship-types.json entries for concept-project edges

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational
- **User Story 2 (Phase 4)**: Depends on Foundational (builds on US1 project existence)
- **User Story 3 (Phase 5)**: Depends on Foundational (needs project ID validation)
- **User Story 5 (Phase 6)**: Depends on US3 (needs concept↔project edges to query)
- **User Story 4 (Phase 7)**: Depends on US2 (needs repos to refresh)
- **Integration (Phase 8)**: Depends on all user stories
- **Polish (Phase 9)**: Depends on all prior phases

### User Story Dependencies

```
                 ┌─────────────────────────────────────────┐
                 │         Foundational (Phase 2)         │
                 └─────────────────┬───────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
         ▼                         ▼                         ▼
   ┌───────────┐           ┌───────────┐           ┌───────────┐
   │   US1     │           │   US2     │           │   US3     │
   │  Project  │           │   Repos   │           │   Edges   │
   │   CRUD    │           │   CRUD    │           │ Validation│
   └─────┬─────┘           └─────┬─────┘           └─────┬─────┘
         │                       │                       │
         │                       ▼                       ▼
         │                 ┌───────────┐           ┌───────────┐
         │                 │   US4     │           │   US5     │
         │                 │  Refresh  │           │ Transitive│
         │                 │  Metadata │           │  Queries  │
         │                 └───────────┘           └───────────┘
         │                       │                       │
         └───────────────────────┴───────────────────────┘
                                 │
                                 ▼
                         ┌───────────────┐
                         │  Integration  │
                         │   (Phase 8)   │
                         └───────────────┘
```

### Parallel Opportunities

**Within Phase 2 (Foundational)**:
- T005, T006, T007 (create directories)
- T008, T009, T010 (domain types)
- T011-T016 (JSONL storage)
- T017-T022 (SQLite storage)
- T023-T029 (unit tests)

**After Foundational**:
- US1, US2, US3 can start in parallel (different files)
- US4 requires US2 completion
- US5 requires US3 completion

---

## Parallel Example: Phase 2 Foundational

```bash
# Launch domain types in parallel:
Task: "Implement Project struct in internal/project/project.go"
Task: "Implement Repo struct in internal/repo/repo.go"
Task: "Implement GitHub client in internal/github/client.go"

# Launch JSONL storage in parallel:
Task: "Implement ReadAllProjects() in internal/storage/projects_jsonl.go"
Task: "Implement ReadAllRepos() in internal/storage/repos_jsonl.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Project CRUD)
4. **STOP and VALIDATE**: `bip project add/get/list/update/delete` all work
5. Deploy/demo - projects work!

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → Projects work → MVP!
3. Add US2 → Repos with GitHub metadata
4. Add US3 → Concept↔project edges with validation
5. Add US5 → Transitive queries (papers via concepts)
6. Add US4 → Metadata refresh
7. Integration → Everything works together
8. Each story adds value without breaking previous

---

## Summary

- **Total tasks**: 81
- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 25 tasks (includes 7 test tasks)
- **Phase 3 (US1 - Project CRUD)**: 8 tasks
- **Phase 4 (US2 - Repos)**: 11 tasks
- **Phase 5 (US3 - Edges)**: 12 tasks
- **Phase 6 (US5 - Transitive)**: 4 tasks
- **Phase 7 (US4 - Refresh)**: 4 tasks
- **Phase 8 (Integration)**: 8 tasks
- **Phase 9 (Polish)**: 5 tasks

**MVP Scope**: Phases 1-3 (37 tasks) → Working project CRUD

**Parallel opportunities**: 47 tasks marked [P]

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Tests included per Constitution IV (Real Testing)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
