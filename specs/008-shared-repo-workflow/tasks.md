# Tasks: Shared Repository Workflow Commands

**Input**: Design documents from `/specs/008-shared-repo-workflow/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: No test tasks included (tests not explicitly requested in specification).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **CLI commands**: `cmd/bip/`
- **Internal packages**: `internal/`
- **Test fixtures**: `testdata/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the git integration package and response types that all commands will use

- [X] T001 Create internal/git/ package directory structure
- [X] T002 [P] Define response types (OpenMultipleResult, DiffResult, NewPapersResult, ExportResult) in cmd/bip/types.go
- [X] T003 [P] Define internal types (BibTeXIndex, GitDiff) in internal/git/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core git integration infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Implement git repository detection (find repo root, check if in git repo) in internal/git/git.go
- [X] T005 Implement commit validation (verify commit exists, resolve refs like HEAD~N) in internal/git/git.go
- [X] T006 Implement JSONL snapshot retrieval from git (git show <commit>:.bipartite/refs.jsonl) in internal/git/git.go
- [X] T007 Implement GitDiff functions (DiffWorkingTree, DiffSince) in internal/git/diff.go
- [X] T008 Implement git log parsing for commit history (commits touching refs.jsonl) in internal/git/log.go
- [X] T009 Create shared git helper functions for CLI commands in cmd/bip/git_helpers.go

**Checkpoint**: Git integration ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Open Multiple Papers for Review (Priority: P1)

**Goal**: Allow users to open multiple papers by ID, with `--recent N` and `--since <commit>` flags

**Independent Test**: Run `bip open` with multiple paper IDs and verify all PDFs open in the default viewer

### Implementation for User Story 1

- [X] T010 [US1] Modify cmd/bip/open.go to accept multiple positional ID arguments
- [X] T011 [US1] Add --recent N flag to cmd/bip/open.go (open N most recently added papers)
- [X] T012 [US1] Add --since <commit> flag to cmd/bip/open.go (open papers added after commit)
- [X] T013 [US1] Implement mutual exclusivity validation (IDs vs --recent vs --since) in cmd/bip/open.go
- [X] T014 [US1] Implement multi-paper opening loop with error collection (FR-004: continue on missing PDF) in cmd/bip/open.go
- [X] T015 [US1] Implement JSON output (OpenMultipleResult) for multi-paper open in cmd/bip/open.go
- [X] T016 [US1] Implement --human output formatting for multi-paper open in cmd/bip/open.go
- [X] T017 [US1] Add actionable error messages per contracts/cli.md (commit not found, PDF not found, pdf_root not set) in cmd/bip/open.go

**Checkpoint**: User Story 1 complete - users can open multiple papers by ID, --recent, or --since

---

## Phase 4: User Story 2 - Track New Papers from Collaborators (Priority: P1)

**Goal**: Provide `bip diff` and `bip new` commands to show papers added/removed since commits

**Independent Test**: Run `bip diff` or `bip new --since` and verify output lists correct papers

### Implementation for User Story 2

- [X] T018 [P] [US2] Create cmd/bip/diff.go with cobra command skeleton and --human flag
- [X] T019 [P] [US2] Create cmd/bip/new.go with cobra command skeleton, --since, --days, --human flags
- [X] T020 [US2] Implement bip diff logic using DiffWorkingTree() in cmd/bip/diff.go
- [X] T021 [US2] Implement JSON output (DiffResult) for bip diff in cmd/bip/diff.go
- [X] T022 [US2] Implement --human output formatting for bip diff in cmd/bip/diff.go
- [X] T023 [US2] Implement bip new --since logic using DiffSince() in cmd/bip/new.go
- [X] T024 [US2] Implement bip new --days logic (papers added within N days UTC) in cmd/bip/new.go
- [X] T025 [US2] Implement mutual exclusivity validation (--since vs --days) in cmd/bip/new.go
- [X] T026 [US2] Implement JSON output (NewPapersResult) for bip new in cmd/bip/new.go
- [X] T027 [US2] Implement --human output formatting for bip new in cmd/bip/new.go
- [X] T028 [US2] Add actionable error messages per contracts/cli.md (commit not found, not in git repo) in cmd/bip/diff.go and cmd/bip/new.go
- [X] T029 [US2] Register diff and new commands in cmd/bip/root.go

**Checkpoint**: User Story 2 complete - users can track new papers from collaborators

---

## Phase 5: User Story 3 - Export Specific Papers to BibTeX (Priority: P2)

**Goal**: Support `bip export --bibtex <id>...` and `--append` mode with deduplication

**Independent Test**: Run `bip export --bibtex <id>` and verify correct BibTeX output; test append mode for deduplication

### Implementation for User Story 3

- [X] T030 [US3] Implement BibTeXIndex type and ParseBibTeXFile() for deduplication in internal/export/bibtex_parse.go
- [X] T031 [US3] Add HasEntry() method (match by DOI primary, citation key fallback) to BibTeXIndex in internal/export/bibtex_parse.go
- [X] T032 [US3] Modify cmd/bip/export.go to accept multiple positional ID arguments
- [X] T033 [US3] Add --append <file> flag to cmd/bip/export.go
- [X] T034 [US3] Implement single/multi-paper BibTeX export (IDs to stdout) in cmd/bip/export.go
- [X] T035 [US3] Implement --append mode with BibTeXIndex deduplication in cmd/bip/export.go
- [X] T036 [US3] Implement JSON output (ExportResult) for --append mode in cmd/bip/export.go
- [X] T037 [US3] Add actionable error messages per contracts/cli.md (unknown key, file read/write errors) in cmd/bip/export.go

**Checkpoint**: User Story 3 complete - users can export specific papers to BibTeX with optional append/deduplication

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation, documentation, and final quality checks

- [X] T038 Run all quickstart.md test scenarios to validate command behavior
- [X] T039 [P] Update CLAUDE.md with new commands documentation
- [X] T040 [P] Update README.md with new command usage examples
- [X] T041 Verify all commands follow CLI patterns (JSON default, --human flag, exit codes)
- [X] T042 Run go vet ./... and go fmt ./... for code quality

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 priority and can proceed in parallel
  - US3 (P2) can proceed in parallel with US1/US2 or after them
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Uses git log parsing for --recent
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Uses DiffWorkingTree and DiffSince
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Independent of US1/US2, only needs existing export infrastructure

### Within Each User Story

- Command skeleton before implementation logic
- Core implementation before output formatting
- JSON output before --human output
- Error messages after main implementation

### Parallel Opportunities

- T002 and T003 (Setup) can run in parallel (different files)
- T018 and T019 (US2 command skeletons) can run in parallel (different files)
- T039 and T040 (Polish docs) can run in parallel (different files)
- All three user stories can be worked on in parallel after Foundational phase

---

## Parallel Example: User Story 2

```bash
# Launch command skeletons together:
Task: "Create cmd/bip/diff.go with cobra command skeleton and --human flag"
Task: "Create cmd/bip/new.go with cobra command skeleton, --since, --days, --human flags"

# Then proceed with implementation sequentially within each command
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Open Multiple Papers)
4. Complete Phase 4: User Story 2 (Track New Papers)
5. **STOP and VALIDATE**: Test both P1 stories independently
6. Deploy/demo collaborative workflow (open + track new papers)

### Incremental Delivery

1. Complete Setup + Foundational -> Git integration ready
2. Add User Story 1 -> Test independently -> Multi-paper open works
3. Add User Story 2 -> Test independently -> Diff/new commands work
4. Add User Story 3 -> Test independently -> BibTeX export enhanced
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (open.go modifications)
   - Developer B: User Story 2 (new diff.go and new.go)
   - Developer C: User Story 3 (export.go modifications + bibtex_parse.go)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- FR references map to spec.md functional requirements
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
