# Tasks: Core Reference Manager

**Input**: Design documents from `/specs/001-core-reference-manager/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **CLI entry**: `cmd/bip/`
- **Internal packages**: `internal/{config,importer,reference,storage,query,export,pdf}/`
- **Test fixtures**: `testdata/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create directory structure per plan.md: cmd/bip/, internal/{config,importer,reference,storage,query,export,pdf}/, testdata/
- [X] T002 Initialize Go module with `go mod init github.com/matsen/bip artite`
- [X] T003 [P] Add spf13/cobra dependency via `go get github.com/spf13/cobra`
- [X] T004 [P] Add modernc.org/sqlite dependency via `go get modernc.org/sqlite`
- [X] T005 [P] Extract test fixtures from _ignore/paperpile-export-jan-12.json to testdata/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 [P] Create Reference struct with JSON tags in internal/reference/reference.go
- [X] T007 [P] Create Author struct in internal/reference/author.go
- [X] T008 [P] Create PublicationDate struct in internal/reference/reference.go
- [X] T009 [P] Create ImportSource struct in internal/reference/reference.go
- [X] T010 [P] Create Config struct with PDFRoot and PDFReader fields in internal/config/config.go
- [X] T011 Implement config Load/Save methods in internal/config/config.go
- [X] T012 Implement JSONL ReadAll function in internal/storage/jsonl.go
- [X] T013 Implement JSONL Append function in internal/storage/jsonl.go
- [X] T014 Create CLI root command with --human and --version flags in cmd/bip/main.go
- [X] T015 [P] Create JSON/human output helper functions in cmd/bip/output.go
- [X] T016 [P] Define exit codes as constants in cmd/bip/exitcodes.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 5 - Initialize and Configure Repository (Priority: P5)

**Goal**: Researcher sets up a new bip artite repository, configuring the PDF folder path

**Independent Test**: Run init in empty directory, configure PDF path, verify configuration persists

**Why first**: All other user stories require an initialized repository to exist

### Implementation for User Story 5

- [X] T017 [US5] Implement bp init command creating .bip artite/ structure in cmd/bip/init.go
- [X] T018 [US5] Add repository detection (check for existing .bip artite/) in internal/config/config.go
- [X] T019 [US5] Implement bp config get/set command in cmd/bip/config.go
- [X] T020 [US5] Add path validation (verify directory exists) in internal/config/config.go
- [X] T021 [US5] Add error for init in already-initialized directory (exit code 1)

**Checkpoint**: At this point, users can initialize repositories and configure PDF paths

---

## Phase 4: User Story 1 - Import References from Paperpile (Priority: P1)

**Goal**: Researcher imports Paperpile JSON export to create searchable reference collection

**Independent Test**: Export Paperpile JSON, run import command, verify references are stored and queryable

### Implementation for User Story 1

- [X] T022 [P] [US1] Parse Paperpile JSON array structure in internal/importer/paperpile.go
- [X] T023 [P] [US1] Map Paperpile fields to Reference struct (see research.md mapping) in internal/importer/paperpile.go
- [X] T024 [US1] Extract main PDF vs supplementary PDFs from attachments in internal/importer/paperpile.go
- [X] T025 [US1] Implement DOI-based deduplication in internal/storage/jsonl.go
- [X] T026 [US1] Implement ID collision handling with suffix (e.g., Ahn2026-rs-2) in internal/storage/jsonl.go
- [X] T027 [US1] Handle papers without DOI (use citekey as ID) in internal/importer/paperpile.go
- [X] T028 [US1] Implement bp import --format paperpile command in cmd/bip/import.go
- [X] T029 [US1] Add --dry-run flag showing what would be imported in cmd/bip/import.go
- [X] T030 [US1] Add import statistics output (imported/updated/skipped counts) in cmd/bip/import.go

**Checkpoint**: At this point, User Story 1 is fully functional - researchers can import papers

---

## Phase 5: User Story 6 - Rebuild Query Layer (Priority: P6)

**Goal**: After git pull or corruption, researcher rebuilds the SQLite query layer from JSONL source

**Independent Test**: Modify source data file, run rebuild, verify query layer reflects changes

**Why before US2**: Search/query commands depend on SQLite being built

### Implementation for User Story 6

- [X] T031 [US6] Create SQLite schema (references table, indexes) in internal/storage/sqlite.go
- [X] T032 [US6] Implement FTS5 virtual table for full-text search in internal/storage/sqlite.go
- [X] T033 [US6] Create FTS5 triggers to keep search index in sync in internal/storage/sqlite.go
- [X] T034 [US6] Implement RebuildFromJSONL function in internal/storage/sqlite.go
- [X] T035 [US6] Implement bp rebuild command in cmd/bip/rebuild.go
- [X] T036 [US6] Add rebuild statistics output (reference count) in cmd/bip/rebuild.go

**Checkpoint**: At this point, User Story 6 is functional - query layer can be rebuilt

---

## Phase 6: User Story 2 - Search and Query References (Priority: P2)

**Goal**: Researcher/agent searches reference collection with structured JSON output

**Independent Test**: Import references, run search queries, verify correct papers returned with metadata

### Implementation for User Story 2

- [X] T037 [P] [US2] Implement keyword search using FTS5 in internal/storage/sqlite.go
- [X] T038 [P] [US2] Implement field-specific search (author:, title:) in internal/storage/sqlite.go
- [X] T039 [US2] Implement bp search command with --limit flag in cmd/bip/search.go
- [X] T040 [P] [US2] Implement GetByID function in internal/storage/sqlite.go
- [X] T041 [US2] Implement bp get command in cmd/bip/get.go
- [X] T042 [P] [US2] Implement ListAll function in internal/storage/sqlite.go
- [X] T043 [US2] Implement bp list command with --limit flag in cmd/bip/list.go
- [X] T044 [US2] Ensure empty search results return empty array (not error) in cmd/bip/search.go

**Checkpoint**: At this point, User Stories 1, 2, 5, and 6 are all functional

---

## Phase 7: User Story 3 - Open PDFs for Reading (Priority: P3)

**Goal**: Researcher/agent opens paper's PDF in system viewer

**Independent Test**: Configure PDF path, import references with PDF paths, run open command, verify PDF opens

### Implementation for User Story 3

- [X] T045 [P] [US3] Implement PDF path resolution (join pdf_root + relative path) in internal/pdf/opener.go
- [X] T046 [US3] Implement platform-specific open command (open on macOS, xdg-open on Linux) in internal/pdf/opener.go
- [X] T047 [US3] Add configurable PDF reader support (skim, zathura, evince, okular) in internal/pdf/opener.go
- [X] T048 [US3] Implement bp open command in cmd/bip/open.go
- [X] T049 [US3] Add --supplement N flag for opening supplementary PDFs in cmd/bip/open.go
- [X] T050 [US3] Add clear error messages for missing PDF files in cmd/bip/open.go

**Checkpoint**: At this point, User Story 3 is functional - PDFs can be opened

---

## Phase 8: User Story 4 - Export to BibTeX (Priority: P4)

**Goal**: Researcher exports references to BibTeX for LaTeX documents

**Independent Test**: Import references, export to BibTeX, verify output is valid BibTeX

### Implementation for User Story 4

- [X] T051 [P] [US4] Implement BibTeX entry type mapping (article, inproceedings, book) in internal/export/bibtex.go
- [X] T052 [US4] Implement BibTeX field formatting (author format: "Last, First and Last, First") in internal/export/bibtex.go
- [X] T053 [US4] Implement LaTeX special character escaping (& % $ # _ { } ~ ^) in internal/export/bibtex.go
- [X] T054 [US4] Implement bp export --bibtex command in cmd/bip/export.go
- [X] T055 [US4] Add --keys filter for exporting specific references in cmd/bip/export.go
- [X] T056 [US4] Handle papers with minimal metadata gracefully in internal/export/bibtex.go

**Checkpoint**: At this point, all 6 user stories are functional

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T057 [P] Implement bp check command for repository integrity in cmd/bip/check.go
- [X] T058 [P] Add duplicate DOI detection to check command in cmd/bip/check.go
- [X] T059 [P] Add missing PDF detection to check command in cmd/bip/check.go
- [X] T060 Improve human-readable output formatting across all commands
- [X] T061 [P] Add shell completion generation (bash, zsh, fish) via cobra
- [X] T062 Run full integration test with _ignore/paperpile-export-jan-12.json
- [X] T063 Validate all commands against contracts/cli.md specification
- [X] T064 Run quickstart.md validation workflow

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **US5 Init/Config (Phase 3)**: Depends on Foundational - other stories need initialized repo
- **US1 Import (Phase 4)**: Depends on US5 - needs repo to import into
- **US6 Rebuild (Phase 5)**: Depends on US1 - needs data to build query layer from
- **US2 Search/Query (Phase 6)**: Depends on US6 - needs SQLite query layer
- **US3 Open PDFs (Phase 7)**: Depends on US1 - needs references with PDF paths
- **US4 Export (Phase 8)**: Depends on US1 - needs references to export
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```
US5 (Init/Config) ─┐
                   ├──► US1 (Import) ─┬──► US6 (Rebuild) ──► US2 (Search/Query)
                   │                  ├──► US3 (Open PDFs)
                   │                  └──► US4 (Export)
```

### Within Each User Story

- Models/types before storage operations
- Storage operations before CLI commands
- Core implementation before flags/options
- Story complete before moving to next phase

### Parallel Opportunities

**Phase 1 (Setup)**:
- T003, T004, T005 can run in parallel

**Phase 2 (Foundational)**:
- T006, T007, T008, T009, T010 can run in parallel (different files)
- T015, T016 can run in parallel

**Phase 4 (US1 Import)**:
- T022, T023 can run in parallel (both in paperpile.go but different functions)

**Phase 6 (US2 Search/Query)**:
- T037, T038 can run in parallel
- T040, T042 can run in parallel

---

## Parallel Example: Foundational Phase

```bash
# Launch all type definitions together:
Task: "Create Reference struct in internal/reference/reference.go"
Task: "Create Author struct in internal/reference/author.go"
Task: "Create Config struct in internal/config/config.go"

# Launch output helpers together:
Task: "Create JSON/human output helper functions in cmd/bip/output.go"
Task: "Define exit codes as constants in cmd/bip/exitcodes.go"
```

---

## Implementation Strategy

### MVP First (User Stories 5, 1, 6, 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: US5 Init/Config
4. Complete Phase 4: US1 Import
5. Complete Phase 5: US6 Rebuild
6. Complete Phase 6: US2 Search/Query
7. **STOP and VALIDATE**: Test import → rebuild → search workflow
8. Deploy/demo if ready - this is a useful CLI!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US5 Init/Config → Can create repositories
3. Add US1 Import → Can import papers (**MVP!**)
4. Add US6 Rebuild + US2 Search → Can find papers
5. Add US3 Open PDFs → Can open papers
6. Add US4 Export → Can export to BibTeX
7. Each story adds value without breaking previous stories

### Key Validation Points

After completing MVP (through Phase 6):
```bash
./bip init
./bip config pdf-root ~/Google\ Drive/Paperpile
./bip import --format paperpile _ignore/paperpile-export-jan-12.json
./bip rebuild
./bip search "phylogenetics"
./bip get Ahn2026-rs
```

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable after its dependencies
- Tests not included per template guidance (not explicitly requested in spec)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
