# Tasks: Generic JSONL + SQLite Store Abstraction

**Input**: Design documents from `/specs/014-jsonl-sqlite-store/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested. Tests will be added organically following existing project patterns.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md, this extends the existing bip CLI:
- Core library: `internal/store/`
- CLI commands: `cmd/bip/`
- Test fixtures: `testdata/stores/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and test fixture creation

- [X] T001 Create internal/store/ package directory structure
- [X] T002 [P] Create testdata/stores/ directory with valid_schema.json fixture
- [X] T003 [P] Create testdata/stores/invalid_schema_no_primary.json fixture
- [X] T004 [P] Create testdata/stores/sample_records.jsonl fixture with 5-10 sample records

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and schema system that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define FieldType constants and Field struct in internal/store/schema.go
- [X] T006 Define Schema struct with Name and Fields in internal/store/schema.go
- [X] T007 Implement ParseSchema() to load and parse JSON schema file in internal/store/schema.go
- [X] T008 Implement Schema.Validate() with primary key check, type validation, enum rules in internal/store/schema.go
- [X] T009 [P] Define StoreRegistry and StoreConfig types in internal/store/registry.go
- [X] T010 Implement LoadRegistry() and SaveRegistry() for .bipartite/stores.json in internal/store/registry.go
- [X] T011 Define Store struct with Name, Schema, Dir, paths in internal/store/store.go
- [X] T012 Implement NewStore() constructor that derives JSONL and DB paths in internal/store/store.go
- [X] T013 Create `bip store` parent command group in cmd/bip/store.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Define and Initialize a Custom Store (Priority: P1) 🎯 MVP

**Goal**: Users can create a new store with a JSON schema, generating empty JSONL and SQLite files

**Independent Test**: Run `bip store init test_store --schema testdata/stores/valid_schema.json` and verify files created

### Implementation for User Story 1

- [X] T014 [US1] Implement GenerateDDL() to create SQLite CREATE TABLE statement from schema in internal/store/sqlite.go
- [X] T015 [US1] Implement GenerateIndexDDL() for fields with index:true in internal/store/sqlite.go
- [X] T016 [US1] Implement GenerateFTS5DDL() for fields with fts:true in internal/store/sqlite.go
- [X] T017 [US1] Implement CreateMetaTable() for _meta table (hash, last_sync) in internal/store/sqlite.go
- [X] T018 [US1] Implement Store.Init() to create empty JSONL, SQLite with schema, register in stores.json in internal/store/store.go
- [X] T019 [US1] Implement `bip store init` command with --schema and --dir flags in cmd/bip/store_init.go
- [X] T020 [US1] Add JSON output support to store init command in cmd/bip/store_init.go
- [X] T021 [US1] Add schema validation error messages (missing primary, invalid type) in cmd/bip/store_init.go

**Checkpoint**: User Story 1 complete - can create stores with schemas

---

## Phase 4: User Story 3 - Sync JSONL to SQLite (Priority: P1)

**Goal**: Users can rebuild SQLite index from JSONL source of truth

**Independent Test**: Manually add records to JSONL, run `bip store sync`, verify records queryable

**Note**: Implementing sync before append/query because sync is needed to test query functionality

### Implementation for User Story 3

- [X] T022 [US3] Implement ComputeJSONLHash() using SHA256 in internal/store/jsonl.go
- [X] T023 [US3] Implement GetStoredHash() and SetStoredHash() for _meta table in internal/store/sqlite.go
- [X] T024 [US3] Implement Store.NeedsSync() comparing JSONL hash to stored hash in internal/store/store.go
- [X] T025 [US3] Implement ReadAllRecords() to read all JSONL lines into []Record in internal/store/jsonl.go
- [X] T026 [US3] Implement Store.Sync() with full rebuild: clear tables, insert all, update hash in internal/store/store.go
- [X] T027 [US3] Implement `bip store sync` command with single store and --all flag in cmd/bip/store_sync.go
- [X] T028 [US3] Add JSON output support and "skipped" vs "rebuilt" status in cmd/bip/store_sync.go

**Checkpoint**: User Story 3 complete - can sync JSONL to SQLite

---

## Phase 5: User Story 2 - Append and Query Records (Priority: P1)

**Goal**: Users can append records and query them via SQL

**Independent Test**: Run `bip store append`, then `bip store sync`, then `bip store query` to retrieve records

### Implementation for User Story 2

- [X] T029 [US2] Implement ValidateRecord() checking types, enums, required primary key in internal/store/schema.go
- [X] T030 [US2] Implement CheckDuplicatePrimaryKey() scanning existing JSONL in internal/store/jsonl.go
- [X] T031 [US2] Implement AppendRecord() with atomic append to JSONL in internal/store/jsonl.go
- [X] T032 [US2] Implement Store.Append() combining validation and atomic write in internal/store/store.go
- [X] T033 [US2] Implement `bip store append` command with JSON arg, --file, --stdin in cmd/bip/store_append.go
- [X] T034 [US2] Add validation error messages (enum, type, duplicate key) in cmd/bip/store_append.go
- [X] T035 [US2] Implement Store.Query() executing SQL against SQLite in internal/store/store.go
- [X] T036 [US2] Implement PrepareFTSQuery() for FTS5 search escaping in internal/store/sqlite.go
- [X] T037 [US2] Implement `bip store query` command with SQL argument in cmd/bip/store_query.go
- [X] T038 [US2] Add output format flags: --json, --csv, --jsonl, --human in cmd/bip/store_query.go
- [X] T039 [US2] Add "store not synced" warning when DB is stale in cmd/bip/store_query.go

**Checkpoint**: User Stories 1, 2, 3 complete - Core MVP functional

---

## Phase 6: User Story 4 - Delete Records (Priority: P2)

**Goal**: Users can delete records by ID or WHERE clause

**Independent Test**: Append records, delete by ID, verify removed from JSONL and queries

### Implementation for User Story 4

- [X] T040 [US4] Implement WriteAllRecords() for atomic JSONL rewrite (temp file + rename) in internal/store/jsonl.go
- [X] T041 [US4] Implement Store.DeleteByID() removing record and triggering sync in internal/store/store.go
- [X] T042 [US4] Implement Store.DeleteWhere() using SQL to find matching IDs in internal/store/store.go
- [X] T043 [US4] Implement `bip store delete` command with ID arg and --where flag in cmd/bip/store_delete.go
- [X] T044 [US4] Add JSON output and error handling in cmd/bip/store_delete.go

**Checkpoint**: User Story 4 complete - delete functionality works

---

## Phase 7: User Story 5 - List and Inspect Stores (Priority: P2)

**Goal**: Users can list all stores and view detailed info about a specific store

**Independent Test**: Create multiple stores, run `bip store list`, run `bip store info <name>`

### Implementation for User Story 5

- [X] T045 [US5] Implement Store.Count() returning record count from JSONL in internal/store/store.go
- [X] T046 [US5] Implement Store.Info() returning sync status, file sizes, schema details in internal/store/store.go
- [X] T047 [US5] Implement ListStores() loading registry and gathering info for each in internal/store/registry.go
- [X] T048 [US5] Implement `bip store list` command with table output in cmd/bip/store_list.go
- [X] T049 [US5] Implement `bip store info` command with detailed output in cmd/bip/store_info.go
- [X] T050 [US5] Add JSON output support to both list and info commands

**Checkpoint**: User Story 5 complete - store management commands work

---

## Phase 8: User Story 6 - Cross-Store Queries (Priority: P3)

**Goal**: Users can JOIN data across multiple stores in a single query

**Independent Test**: Create two stores, run `bip store query --cross "SELECT ... JOIN ..."`

### Implementation for User Story 6

- [X] T051 [US6] Implement AttachAllStores() using SQLite ATTACH DATABASE in internal/store/sqlite.go
- [X] T052 [US6] Implement QueryCross() attaching stores, executing query, detaching in internal/store/store.go
- [X] T053 [US6] Add --cross flag to `bip store query` command in cmd/bip/store_query.go
- [X] T054 [US6] Handle table aliasing for cross-store queries in cmd/bip/store_query.go

**Checkpoint**: User Story 6 complete - cross-store queries work

---

## Phase 9: User Story 7 - Migrate Existing Stores (Priority: P3)

**Goal**: Existing refs, concepts, edges stores use the generic store abstraction

**Independent Test**: Run `bip rebuild` and verify it uses store sync for all stores

### Implementation for User Story 7

- [ ] T055 [P] [US7] Create .bipartite/schemas/refs.json matching current refs schema
- [ ] T056 [P] [US7] Create .bipartite/schemas/concepts.json matching current concepts schema
- [ ] T057 [P] [US7] Create .bipartite/schemas/edges.json matching current edges schema
- [ ] T058 [US7] Register refs, concepts, edges in .bipartite/stores.json
- [ ] T059 [US7] Update `bip rebuild` command to call `store sync --all` in cmd/bip/rebuild.go
- [ ] T060 [US7] Verify existing internal/storage/ code still works during transition

**Checkpoint**: User Story 7 complete - migration path established

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup and validation

- [X] T061 Run `go fmt ./...` and `go vet ./...` on all new code
- [X] T062 Add doc comments to all exported functions in internal/store/
- [X] T063 Run quickstart.md validation (manual test of documented workflow)
- [X] T064 Update README.md with new `bip store` command documentation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - creates stores
- **User Story 3 (P1)**: Depends on US1 - syncs stores that exist
- **User Story 2 (P1)**: Depends on US1 and US3 - append needs init, query needs sync
- **User Story 4 (P2)**: Depends on US2 - delete needs records to exist
- **User Story 5 (P2)**: Depends on US1 - list/info needs stores to exist
- **User Story 6 (P3)**: Depends on US2 - cross-query needs queryable stores
- **User Story 7 (P3)**: Depends on US1-3 - migration needs core store working

### Within Each User Story

- SQLite DDL before Store.Init() (US1)
- JSONL hash before Store.Sync() (US3)
- Validation before Store.Append() (US2)
- Library code before CLI commands (all)

### Parallel Opportunities

- T002, T003, T004 can run in parallel (different fixture files)
- T009 can run in parallel with T005-T008 (different files)
- T055, T056, T057 can run in parallel (different schema files)

---

## Parallel Example: Foundational Phase

```bash
# Launch schema tasks sequentially (same file):
Task T005: "Define FieldType constants and Field struct in internal/store/schema.go"
Task T006: "Define Schema struct in internal/store/schema.go"
Task T007: "Implement ParseSchema() in internal/store/schema.go"
Task T008: "Implement Schema.Validate() in internal/store/schema.go"

# In parallel with registry tasks (different file):
Task T009: "Define StoreRegistry and StoreConfig types in internal/store/registry.go"
Task T010: "Implement LoadRegistry() and SaveRegistry() in internal/store/registry.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1, 2, 3)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (init)
4. Complete Phase 4: User Story 3 (sync)
5. Complete Phase 5: User Story 2 (append/query)
6. **STOP and VALIDATE**: Full end-to-end workflow works
7. Demo: `bip store init` → `bip store append` → `bip store sync` → `bip store query`

### Incremental Delivery

1. Setup + Foundational → Core types ready
2. Add US1 → Can create stores (minimal value)
3. Add US3 → Can sync stores (enables queries)
4. Add US2 → Can append and query (MVP complete!)
5. Add US4 → Can delete records
6. Add US5 → Can manage stores
7. Add US6 → Can cross-query
8. Add US7 → Migration complete

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1, US3, US2 order is intentional: init → sync → append/query dependency chain
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
