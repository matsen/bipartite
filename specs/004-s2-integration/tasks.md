# Tasks: Semantic Scholar (S2) Integration

**Input**: Design documents from `/specs/004-s2-integration/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/cli.md

**Tests**: Tests are not explicitly requested in this specification. If TDD is desired, add test tasks before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- **CLI commands**: `cmd/bip/`
- **Internal packages**: `internal/`
- **Test fixtures**: `testdata/`
- **Integration tests**: `tests/integration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the internal/s2 package structure and shared types

- [X] T001 Create internal/s2/ package directory structure
- [X] T002 [P] Create types.go with S2Paper, PaperIdentifier, and API response types in internal/s2/types.go
- [X] T003 [P] Create parser.go with paper identifier parsing (DOI:, ARXIV:, PMID:, etc.) in internal/s2/parser.go
- [X] T004 [P] Create mapper.go with S2Paper to Reference mapping in internal/s2/mapper.go
- [X] T005 Create client.go with rate-limited HTTP client for S2 API in internal/s2/client.go
- [X] T006 Create test fixtures directory and sample S2 API responses in testdata/asta/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core S2 API client infrastructure that ALL commands depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T007 Implement rate limiter using golang.org/x/time/rate in internal/s2/client.go
- [X] T008 Implement S2 API paper lookup method (GET /paper/{id}) in internal/s2/client.go
- [X] T009 Implement S2 API citations method (GET /paper/{id}/citations) in internal/s2/client.go
- [X] T010 Implement S2 API references method (GET /paper/{id}/references) in internal/s2/client.go
- [X] T011 Implement S2 API search by title method (GET /paper/search) in internal/s2/client.go
- [X] T012 Add author name parsing (split on last space, handle suffixes) in internal/s2/mapper.go
- [X] T013 Add error types for S2 API (NotFound, RateLimited, NetworkError) in internal/s2/errors.go
- [X] T014 Create bp s2 parent command with --human flag in cmd/bip/asta.go
- [X] T015 Add local paper lookup helper (resolve local ID to DOI/S2 ID) in internal/s2/local.go

**Checkpoint**: Foundation ready - S2 client works, parent command exists, user story implementation can begin

---

## Phase 3: User Story 1 - Add Paper by DOI (Priority: P1) MVP

**Goal**: Researchers can add papers to their collection using DOI, arXiv ID, or other S2-supported identifiers

**Independent Test**: Run `./bip s2 add DOI:10.1093/sysbio/syy032`, verify paper is added to refs.jsonl with metadata from S2

### Implementation for User Story 1

- [X] T016 [US1] Create s2_add.go command skeleton with flags (--update, --link, --human) in cmd/bip/s2_add.go
- [X] T017 [US1] Implement paper fetch from S2 API in asta add command in cmd/bip/s2_add.go
- [X] T018 [US1] Implement duplicate detection (check if DOI already exists locally) in cmd/bip/s2_add.go
- [X] T019 [US1] Implement paper mapping from S2 response to Reference schema in cmd/bip/s2_add.go
- [X] T020 [US1] Implement append to refs.jsonl using existing storage package in cmd/bip/s2_add.go
- [X] T021 [US1] Implement --update flag to refresh metadata for existing papers in cmd/bip/s2_add.go
- [X] T022 [US1] Implement --link flag to set pdf_path on added paper in cmd/bip/s2_add.go
- [X] T023 [US1] Implement JSON output (default) and --human flag output formatting in cmd/bip/s2_add.go
- [X] T024 [US1] Add exit codes per CLI contract (0=success, 1=not found, 2=duplicate, 3=API error) in cmd/bip/s2_add.go
- [X] T025 [US1] Add test fixture for paper response in testdata/asta/paper_response.json

**Checkpoint**: `bip s2 add DOI:...` works independently - core MVP complete

---

## Phase 4: User Story 2 - Add Paper from PDF (Priority: P2)

**Goal**: Researchers can add papers by providing a PDF file, with automatic DOI extraction

**Independent Test**: Run `./bip s2 add-pdf testdata/sample.pdf`, verify DOI is extracted and paper is added

### Implementation for User Story 2

- [X] T026 [P] [US2] Create internal/pdf/doi.go with PDF DOI extraction using ledongthuc/pdf library
- [X] T027 [P] [US2] Implement DOI regex pattern matching (10.\d{4,9}/[-._;()/:A-Z0-9]+) in internal/pdf/doi.go
- [X] T028 [US2] Create s2_addpdf.go command skeleton with flags (--link, --human) in cmd/bip/s2_addpdf.go
- [X] T029 [US2] Implement PDF text extraction (first 2 pages) in cmd/bip/s2_addpdf.go
- [X] T030 [US2] Implement DOI extraction and fallback to title-based search in cmd/bip/s2_addpdf.go
- [X] T031 [US2] Implement title extraction for fallback S2 search in cmd/bip/s2_addpdf.go
- [X] T032 [US2] Handle multiple title matches (prompt for selection or error in non-interactive) in cmd/bip/s2_addpdf.go
- [X] T033 [US2] Implement --link flag to automatically set pdf_path in cmd/bip/s2_addpdf.go
- [X] T034 [US2] Add JSON and human-readable output formatting in cmd/bip/s2_addpdf.go
- [X] T035 [US2] Add exit codes per CLI contract in cmd/bip/s2_addpdf.go
- [ ] T036 [P] [US2] Add test PDF with embedded DOI in testdata/asta/sample_with_doi.pdf

**Checkpoint**: `bip s2 add-pdf` works independently - PDF workflow complete

---

## Phase 5: User Story 3 - Lookup Paper Info (Priority: P2)

**Goal**: Agents and researchers can query S2 for paper information without adding to the collection

**Independent Test**: Run `./bip s2 lookup DOI:10.1093/sysbio/syy032`, verify JSON output with paper metadata

### Implementation for User Story 3

- [X] T037 [US3] Create s2_lookup.go command skeleton with flags (--fields, --exists, --human) in cmd/bip/s2_lookup.go
- [X] T038 [US3] Implement paper lookup from S2 API in cmd/bip/s2_lookup.go
- [X] T039 [US3] Implement --fields flag to select which fields to return in cmd/bip/s2_lookup.go
- [X] T040 [US3] Implement --exists flag to check if paper is in local collection in cmd/bip/s2_lookup.go
- [X] T041 [US3] Add JSON and human-readable output formatting in cmd/bip/s2_lookup.go
- [X] T042 [US3] Add exit codes per CLI contract in cmd/bip/s2_lookup.go

**Checkpoint**: `bip s2 lookup` works independently - exploration tool complete

---

## Phase 6: User Story 4 - Find Citing Papers (Priority: P3)

**Goal**: Researchers can find papers that cite a paper in their collection (forward citation tracking)

**Independent Test**: Run `./bip s2 citations <paper-id>`, verify list of citing papers is returned

### Implementation for User Story 4

- [X] T043 [US4] Create s2_citations.go command skeleton with flags (--local-only, --limit, --human) in cmd/bip/s2_citations.go
- [X] T044 [US4] Implement paper ID resolution (local ID to S2 ID) in cmd/bip/s2_citations.go
- [X] T045 [US4] Implement citations fetch from S2 API in cmd/bip/s2_citations.go
- [X] T046 [US4] Implement local collection check for each citation result in cmd/bip/s2_citations.go
- [X] T047 [US4] Implement --local-only flag to filter to papers in collection in cmd/bip/s2_citations.go
- [X] T048 [US4] Implement --limit flag for pagination in cmd/bip/s2_citations.go
- [X] T049 [US4] Add JSON and human-readable output formatting in cmd/bip/s2_citations.go
- [X] T050 [US4] Add exit codes per CLI contract in cmd/bip/s2_citations.go
- [X] T051 [P] [US4] Add test fixture for citations response in testdata/asta/citations_response.json

**Checkpoint**: `bip s2 citations` works independently - forward citation tracking complete

---

## Phase 7: User Story 5 - Find Referenced Papers (Priority: P3)

**Goal**: Researchers can find papers referenced by a paper in their collection (backward exploration)

**Independent Test**: Run `./bip s2 references <paper-id>`, verify list of referenced papers is returned

### Implementation for User Story 5

- [X] T052 [US5] Create s2_references.go command skeleton with flags (--missing, --limit, --human) in cmd/bip/s2_references.go
- [X] T053 [US5] Implement paper ID resolution (local ID to S2 ID) in cmd/bip/s2_references.go
- [X] T054 [US5] Implement references fetch from S2 API in cmd/bip/s2_references.go
- [X] T055 [US5] Implement local collection check for each reference result in cmd/bip/s2_references.go
- [X] T056 [US5] Implement --missing flag to filter to papers NOT in collection in cmd/bip/s2_references.go
- [X] T057 [US5] Implement --limit flag for pagination in cmd/bip/s2_references.go
- [X] T058 [US5] Add JSON and human-readable output formatting in cmd/bip/s2_references.go
- [X] T059 [US5] Add exit codes per CLI contract in cmd/bip/s2_references.go
- [X] T060 [P] [US5] Add test fixture for references response in testdata/asta/references_response.json

**Checkpoint**: `bip s2 references` works independently - backward exploration complete

---

## Phase 8: User Story 6 - Discover Literature Gaps (Priority: P4)

**Goal**: Researchers can discover important papers they might be missing based on citation analysis

**Independent Test**: Run `./bip s2 gaps`, verify list of highly-cited-but-missing papers is returned

### Implementation for User Story 6

- [X] T061 [US6] Create s2_gaps.go command skeleton with flags (--min-citations, --limit, --human) in cmd/bip/s2_gaps.go
- [X] T062 [US6] Implement collection loading (all papers with DOIs) in cmd/bip/s2_gaps.go
- [X] T063 [US6] Implement batched references fetch for all collection papers in cmd/bip/s2_gaps.go
- [X] T064 [US6] Implement in-memory citation aggregation (count papers cited by multiple local papers) in cmd/bip/s2_gaps.go
- [X] T065 [US6] Implement gap filtering (cited by >= min_citations AND not in collection) in cmd/bip/s2_gaps.go
- [X] T066 [US6] Implement ranking by citation count within collection in cmd/bip/s2_gaps.go
- [X] T067 [US6] Implement progress indication for long-running batch operations in cmd/bip/s2_gaps.go
- [X] T068 [US6] Implement --min-citations flag (default: 2) in cmd/bip/s2_gaps.go
- [X] T069 [US6] Implement --limit flag (default: 20) in cmd/bip/s2_gaps.go
- [X] T070 [US6] Add JSON and human-readable output formatting (show which local papers cite each gap) in cmd/bip/s2_gaps.go
- [X] T071 [US6] Add exit codes per CLI contract in cmd/bip/s2_gaps.go

**Checkpoint**: `bip s2 gaps` works independently - proactive gap discovery complete

---

## Phase 9: User Story 7 - Link Preprint to Published Version (Priority: P4)

**Goal**: Researchers can detect and link preprints to their published versions using the `supersedes` field

**Independent Test**: Run `./bip s2 link-published`, verify preprints with published versions are detected

### Implementation for User Story 7

- [X] T072 [US7] Create s2_linkpub.go command skeleton with flags (--auto, --human) in cmd/bip/s2_linkpub.go
- [X] T073 [US7] Implement preprint detection (check venue for bioRxiv/medRxiv/arXiv) in cmd/bip/s2_linkpub.go
- [X] T074 [US7] Implement collection scan for preprints without supersedes set in cmd/bip/s2_linkpub.go
- [X] T075 [US7] Implement S2 title search to find published version in cmd/bip/s2_linkpub.go
- [X] T076 [US7] Implement published version matching (same title, non-preprint venue, same authors) in cmd/bip/s2_linkpub.go
- [X] T077 [US7] Implement confirmation prompt for linking (unless --auto) in cmd/bip/s2_linkpub.go
- [X] T078 [US7] Implement supersedes field update in refs.jsonl in cmd/bip/s2_linkpub.go
- [X] T079 [US7] Implement --auto flag for automatic linking without confirmation in cmd/bip/s2_linkpub.go
- [X] T080 [US7] Add JSON and human-readable output formatting in cmd/bip/s2_linkpub.go
- [X] T081 [US7] Add exit codes per CLI contract in cmd/bip/s2_linkpub.go

**Checkpoint**: `bip s2 link-published` works independently - preprint management complete

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T082 [P] Add integration test for asta add command in tests/integration/s2_test.go
- [ ] T083 [P] Add integration test for asta citations/references commands in tests/integration/s2_test.go
- [X] T084 Validate all commands against CLI contract in contracts/cli.md
- [ ] T085 Run quickstart.md validation (all examples work)
- [X] T086 Code review and cleanup across internal/s2/ package
- [X] T087 Ensure rate limiting is consistent across all commands

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 -> P2 -> P3 -> P4)
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Uses US1 add logic internally
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 6 (P4)**: Can start after Foundational (Phase 2) - Uses US5 references logic internally
- **User Story 7 (P4)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Command skeleton before implementation
- Core implementation before flags
- Flags before output formatting
- Output formatting before exit codes

### Parallel Opportunities

- Setup phase: T002, T003, T004 can run in parallel (different files)
- Foundational phase: T008, T009, T010, T011 can run in parallel (after T007 rate limiter)
- Once Foundational completes, all user stories can start in parallel
- Within US2: T026, T027 can run in parallel (different files)
- Within US4: T051 can run in parallel with implementation
- Within US5: T060 can run in parallel with implementation
- Polish phase: T082, T083 can run in parallel (different test files)

---

## Parallel Example: Foundational Phase

```bash
# After T007 (rate limiter) completes, launch API methods in parallel:
Task: "Implement S2 API paper lookup method in internal/s2/client.go"
Task: "Implement S2 API citations method in internal/s2/client.go"
Task: "Implement S2 API references method in internal/s2/client.go"
Task: "Implement S2 API search by title method in internal/s2/client.go"
```

## Parallel Example: User Story 1

```bash
# T016-T025 are sequential (same file, dependencies)
# But US1 can run in parallel with US3-US7 (different commands)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Add Paper by DOI)
4. **STOP and VALIDATE**: Test `bip s2 add DOI:...` independently
5. Deploy/demo if ready - basic paper addition works!

### Incremental Delivery

1. Complete Setup + Foundational -> Foundation ready
2. Add User Story 1 -> Test independently -> Deploy/Demo (MVP!)
3. Add User Story 2 (PDF) -> Test independently -> Deploy/Demo
4. Add User Story 3 (Lookup) -> Test independently -> Deploy/Demo
5. Add User Stories 4+5 (Citations/References) -> Test independently -> Deploy/Demo
6. Add User Stories 6+7 (Gaps/Linking) -> Test independently -> Deploy/Demo
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Stories 1, 2 (Add commands)
   - Developer B: User Stories 3, 4, 5 (Query commands)
   - Developer C: User Stories 6, 7 (Discovery commands)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- JSON output is default (agent-first design), --human flag for readable output
- Rate limiting is critical - respect S2 limits (100 req/5 min unauthenticated)
