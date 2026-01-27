# Tasks: URL Output and Clipboard Support

**Input**: Design documents from `/specs/015-url-clipboard/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested in specification - test tasks omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go CLI project**: `cmd/bip/`, `internal/` at repository root
- Uses spf13/cobra for CLI, modernc.org/sqlite for storage

---

## Phase 1: Setup

**Purpose**: Project structure verification and new package scaffolding

- [x] T001 Create internal/clipboard/ package directory structure

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T002 [P] Add external ID fields (PMID, PMCID, ArXivID, S2ID) to Reference struct in internal/reference/reference.go
- [x] T003 [P] Add external ID columns (pmid, pmcid, arxiv_id, s2_id) to SQLite schema in internal/storage/sqlite.go
- [x] T004 Update SQLite insert/query logic to handle external ID fields in internal/storage/sqlite.go

**Checkpoint**: Foundation ready - external ID storage is complete, user story implementation can now begin

---

## Phase 3: User Story 4 - External IDs Populated on Import (Priority: P2, but blocks US3)

**Goal**: When papers are imported via S2, external identifiers are automatically stored

**Independent Test**: Import a paper via `bip s2 add` and verify external IDs appear in refs.jsonl

**Why this order**: Although P2, this blocks User Story 3 (alternative URL formats). Must complete before US3.

### Implementation for User Story 4

- [x] T005 [US4] Update S2 mapper to extract and populate PMID from ExternalIDs.PubMed in internal/s2/mapper.go
- [x] T006 [US4] Update S2 mapper to extract and populate PMCID from ExternalIDs.PubMedCentral in internal/s2/mapper.go
- [x] T007 [US4] Update S2 mapper to extract and populate ArXivID from ExternalIDs.ArXiv in internal/s2/mapper.go
- [x] T008 [US4] Update S2 mapper to populate S2ID from PaperID in internal/s2/mapper.go
- [x] T009 [US4] Verify S2Paper struct includes ExternalIDs fields in internal/s2/types.go

**Checkpoint**: Papers imported via S2 now have external IDs stored

---

## Phase 4: User Story 1 - Get DOI URL for a Reference (Priority: P1) :dart: MVP

**Goal**: User can get the DOI link for any paper in their library

**Independent Test**: Run `bip url <ref-id>` and verify correct DOI URL is output

### Implementation for User Story 1

- [x] T010 [US1] Create url command scaffold with cobra (Args, RunE, flags) in cmd/bip/url.go
- [x] T011 [US1] Define URLResult struct for JSON output in cmd/bip/url.go
- [x] T012 [US1] Implement DOI URL generation function (https://doi.org/{doi}) in cmd/bip/url.go
- [x] T013 [US1] Implement reference lookup by ID using existing storage query in cmd/bip/url.go
- [x] T014 [US1] Implement JSON output format (--json flag, default) in cmd/bip/url.go
- [x] T015 [US1] Implement human-readable output (-H flag, URL to stdout) in cmd/bip/url.go
- [x] T016 [US1] Handle error case: reference not found in cmd/bip/url.go
- [x] T017 [US1] Handle error case: no DOI available for reference in cmd/bip/url.go
- [x] T018 [US1] Register url command in cmd/bip/root.go

**Checkpoint**: `bip url <ref-id>` outputs DOI URL - MVP complete

---

## Phase 5: User Story 2 - Copy URL to Clipboard (Priority: P1)

**Goal**: User can copy a reference URL directly to their clipboard

**Independent Test**: Run `bip url <ref-id> --copy` and verify URL is in system clipboard

### Implementation for User Story 2

- [x] T019 [P] [US2] Implement clipboard package with Copy function signature in internal/clipboard/clipboard.go
- [x] T020 [US2] Implement platform detection using runtime.GOOS in internal/clipboard/clipboard.go
- [x] T021 [US2] Implement macOS clipboard support using pbcopy in internal/clipboard/clipboard.go
- [x] T022 [US2] Implement Linux clipboard support with xclip/xsel fallback in internal/clipboard/clipboard.go
- [x] T023 [US2] Define ErrClipboardUnavailable error and IsAvailable function in internal/clipboard/clipboard.go
- [x] T024 [US2] Add --copy flag to url command in cmd/bip/url.go
- [x] T025 [US2] Integrate clipboard Copy call with url command in cmd/bip/url.go
- [x] T026 [US2] Implement graceful fallback when clipboard unavailable (warning to stderr) in cmd/bip/url.go
- [x] T027 [US2] Update URLResult struct to include Copied field in cmd/bip/url.go

**Checkpoint**: `bip url <ref-id> --copy` copies URL to clipboard with graceful fallback

---

## Phase 6: User Story 3 - Get Alternative URL Formats (Priority: P2)

**Goal**: User can get PubMed, PMC, arXiv, or Semantic Scholar URL instead of DOI

**Independent Test**: Run `bip url <ref-id> --pubmed` and verify correct PubMed URL is output

### Implementation for User Story 3

- [x] T028 [P] [US3] Add --pubmed flag to url command in cmd/bip/url.go
- [x] T029 [P] [US3] Add --pmc flag to url command in cmd/bip/url.go
- [x] T030 [P] [US3] Add --arxiv flag to url command in cmd/bip/url.go
- [x] T031 [P] [US3] Add --s2 flag to url command in cmd/bip/url.go
- [x] T032 [US3] Implement mutual exclusivity check for format flags in cmd/bip/url.go
- [x] T033 [US3] Implement PubMed URL generation (https://pubmed.ncbi.nlm.nih.gov/{pmid}/) in cmd/bip/url.go
- [x] T034 [US3] Implement PMC URL generation (https://www.ncbi.nlm.nih.gov/pmc/articles/{pmcid}/) in cmd/bip/url.go
- [x] T035 [US3] Implement arXiv URL generation (https://arxiv.org/abs/{arxiv_id}) in cmd/bip/url.go
- [x] T036 [US3] Implement S2 URL generation (https://www.semanticscholar.org/paper/{s2_id}) in cmd/bip/url.go
- [x] T037 [US3] Handle error case: requested ID type not available for reference in cmd/bip/url.go
- [x] T038 [US3] Update URLResult.Format field to reflect selected format in cmd/bip/url.go

**Checkpoint**: All URL formats working with proper error handling

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final validation

- [x] T039 [P] Update README.md with bip url command documentation
- [x] T040 [P] Update /bip skill in .claude/skills/bip/ with url command guidance
- [x] T041 Run quickstart.md validation scenarios manually
- [x] T042 Verify all functional requirements (FR-001 through FR-013) from spec.md are met

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 4 (Phase 3)**: Depends on Foundational - BLOCKS User Story 3
- **User Story 1 (Phase 4)**: Depends on Foundational only (MVP)
- **User Story 2 (Phase 5)**: Depends on User Story 1 (adds --copy to existing command)
- **User Story 3 (Phase 6)**: Depends on User Story 4 (needs external IDs populated)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1: Setup
    |
Phase 2: Foundational (schema changes)
    |
    +---> Phase 3: US4 (S2 mapper) ---> Phase 6: US3 (alt formats)
    |
    +---> Phase 4: US1 (DOI URL) ---> Phase 5: US2 (clipboard)
                                              |
                                      Phase 7: Polish
```

### Within Each User Story

- Types/structs before functions
- Core implementation before error handling
- Flags registered before mutual exclusivity logic
- Command complete before moving to dependent story

### Parallel Opportunities

**Phase 2 (Foundational)**:
- T002 and T003 can run in parallel (different files)
- T004 depends on T002 and T003

**Phase 5 (US2)**:
- T019 (clipboard package) can start immediately in Phase 5

**Phase 6 (US3)**:
- T028, T029, T030, T031 can all run in parallel (adding flags)
- T033-T038 depend on flags being added

**Phase 7 (Polish)**:
- T039 and T040 can run in parallel (different files)

---

## Parallel Example: Phase 6 (User Story 3)

```bash
# Launch all format flag implementations together:
Task T028: "Add --pubmed flag to url command in cmd/bip/url.go"
Task T029: "Add --pmc flag to url command in cmd/bip/url.go"
Task T030: "Add --arxiv flag to url command in cmd/bip/url.go"
Task T031: "Add --s2 flag to url command in cmd/bip/url.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (schema changes)
3. Complete Phase 4: User Story 1 (DOI URL)
4. **STOP and VALIDATE**: Test `bip url <ref-id>` with existing references
5. User can now get DOI URLs - deploy if ready

### Incremental Delivery

1. Setup + Foundational → Schema ready
2. Add US1 (DOI URL) → `bip url` works → **MVP deployed**
3. Add US2 (clipboard) → `bip url --copy` works → Deploy
4. Add US4 (S2 mapper) → New imports have external IDs
5. Add US3 (alt formats) → `bip url --pubmed` etc works → Deploy
6. Polish → Documentation complete → Feature complete

### Suggested MVP Scope

**User Story 1** delivers minimal viable functionality:
- Get DOI URL for any reference
- DOI is universal identifier (every paper has one)

Add **User Story 2** for high-value increment:
- Clipboard copy eliminates manual selection
- Core value proposition of the feature

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- User Story 4 is P2 but implemented early because US3 depends on it
- Schema changes require `rm .bipartite/cache/refs.db && bip rebuild`
- Clipboard gracefully falls back when unavailable (no hard error)
- URL goes to stdout, messages to stderr (composable for piping)
- Commit after each task or logical group
