# Tasks: ASTA MCP Integration

**Input**: Design documents from `/specs/005-asta-mcp-integration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Package**: `internal/asta/` for ASTA client code
- **Commands**: `cmd/bip/` for CLI commands

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the ASTA package structure and core types

- [X] T001 Create internal/asta/ package directory structure
- [X] T002 [P] Create MCP types (MCPRequest, MCPResponse, MCPParams, MCPResult, MCPContent, MCPError) in internal/asta/types.go
- [X] T003 [P] Create domain types (ASTAPaper, ASTAAuthor, ASTASnippet, ASTACitation) in internal/asta/types.go
- [X] T004 [P] Create error types (ErrNotFound, ErrAuthError, ErrRateLimited, ErrAPIError) in internal/asta/errors.go
- [X] T005 Create MCP HTTP client with rate limiting in internal/asta/client.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the parent command and shared infrastructure for all ASTA commands

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Add ASTA exit codes (ExitASTANotFound=1, ExitASTAAuthError=2, ExitASTAAPIError=3) to cmd/bip/exitcodes.go
- [X] T007 Create parent `bip asta` command with --human flag and godotenv loading in cmd/bip/asta.go
- [X] T008 Create ASTA helper functions (output formatting, error handling) in cmd/bip/asta_helpers.go

**Checkpoint**: Foundation ready - `bip asta --help` should work

---

## Phase 3: User Story 1 - Search Papers by Keyword (Priority: P1) 🎯 MVP

**Goal**: Enable researchers to find papers on a topic using keyword search

**Independent Test**: Run `bip asta search "phylogenetic inference"`, verify papers are returned with title, authors, year, venue

### Implementation for User Story 1

- [X] T009 [US1] Add SearchPapers method to client calling search_papers_by_relevance MCP tool in internal/asta/client.go
- [X] T010 [US1] Create `bip asta search` command with --limit, --year, --venue flags in cmd/bip/asta_search.go
- [X] T011 [US1] Implement JSON and human output formatting for search results in cmd/bip/asta_search.go

**Checkpoint**: `bip asta search "machine learning" --human` returns papers

---

## Phase 4: User Story 2 - Search Paper Snippets (Priority: P1)

**Goal**: Enable researchers to find specific text passages within papers (ASTA's unique feature)

**Independent Test**: Run `bip asta snippet "variational inference phylogenetics"`, verify text snippets are returned with paper context

### Implementation for User Story 2

- [X] T012 [US2] Add SnippetSearch method to client calling snippet_search MCP tool in internal/asta/client.go
- [X] T013 [US2] Create `bip asta snippet` command with --limit, --venue, --papers flags in cmd/bip/asta_snippet.go
- [X] T014 [US2] Implement JSON and human output formatting for snippet results in cmd/bip/asta_snippet.go

**Checkpoint**: `bip asta snippet "mutation rate" --human` returns snippets with paper context

---

## Phase 5: User Story 3 - Get Paper Details (Priority: P2)

**Goal**: Enable researchers to look up detailed information about a specific paper

**Independent Test**: Run `bip asta paper DOI:10.1093/sysbio/syy032`, verify paper metadata is returned

### Implementation for User Story 3

- [X] T015 [US3] Add GetPaper method to client calling get_paper MCP tool in internal/asta/client.go
- [X] T016 [US3] Create `bip asta paper` command with --fields flag in cmd/bip/asta_paper.go
- [X] T017 [US3] Implement JSON and human output formatting for paper details in cmd/bip/asta_paper.go

**Checkpoint**: `bip asta paper DOI:10.1038/nature12373 --human` returns paper details

---

## Phase 6: User Story 4 - Get Citations/References (Priority: P2)

**Goal**: Enable researchers to explore a paper's citation network

**Independent Test**: Run `bip asta citations DOI:10.1093/sysbio/syy032`, verify citing papers are returned

### Implementation for User Story 4

- [X] T018 [US4] Add GetCitations method to client calling get_citations MCP tool in internal/asta/client.go
- [X] T019 [US4] Create `bip asta citations` command with --limit, --year flags in cmd/bip/asta_citations.go
- [X] T020 [US4] Implement JSON and human output formatting for citations in cmd/bip/asta_citations.go
- [X] T021 [US4] Create `bip asta references` command with --limit flag in cmd/bip/asta_references.go
- [X] T022 [US4] Implement JSON and human output formatting for references in cmd/bip/asta_references.go

**Checkpoint**: `bip asta citations DOI:10.1038/nature12373 --human` and `bip asta references DOI:10.1038/nature12373 --human` both work

---

## Phase 7: User Story 5 - Search Authors (Priority: P3)

**Goal**: Enable researchers to find an author and their publications

**Independent Test**: Run `bip asta author "Frederick Matsen"`, verify author info is returned

### Implementation for User Story 5

- [X] T023 [US5] Add SearchAuthors method to client calling search_authors_by_name MCP tool in internal/asta/client.go
- [X] T024 [US5] Add GetAuthorPapers method to client calling get_author_papers MCP tool in internal/asta/client.go
- [X] T025 [US5] Create `bip asta author` command with --limit flag in cmd/bip/asta_author.go
- [X] T026 [US5] Implement JSON and human output formatting for author results in cmd/bip/asta_author.go
- [X] T027 [US5] Create `bip asta author-papers` command with --limit, --year flags in cmd/bip/asta_author_papers.go
- [X] T028 [US5] Implement JSON and human output formatting for author papers in cmd/bip/asta_author_papers.go

**Checkpoint**: `bip asta author "Frederick Matsen" --human` and `bip asta author-papers <id> --human` both work

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation, and cleanup

- [X] T029 Update README.md with ASTA commands documentation
- [X] T030 Update CLAUDE.md with ASTA notes under Commands section
- [X] T031 Run `go build -o bip ./cmd/bip` and verify all commands work
- [X] T032 Run `go vet ./...` and `go fmt ./...` to ensure code quality
- [X] T033 Manual test: Run quickstart.md examples to validate all workflows

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Phase 2 completion
  - US1 and US2 are both P1 priority, can proceed in parallel
  - US3 and US4 are both P2 priority, can proceed in parallel (after US1/US2 or independently)
  - US5 is P3 priority, can proceed after Phase 2
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2 - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Phase 2 - No dependencies on other stories
- **User Story 3 (P2)**: Can start after Phase 2 - No dependencies on other stories
- **User Story 4 (P2)**: Can start after Phase 2 - No dependencies on other stories
- **User Story 5 (P3)**: Can start after Phase 2 - No dependencies on other stories

### Within Each User Story

- Client method before CLI command
- CLI command before output formatting (often same file)
- Complete story before moving to next priority

### Parallel Opportunities

- T002, T003, T004 can run in parallel (different sections of types.go, or split into separate files)
- All user stories can start in parallel after Phase 2 (if team capacity allows)
- T018-T022 (citations/references) can split between two developers

---

## Parallel Example: Setup Phase

```bash
# Launch all type definitions together:
Task: "Create MCP types in internal/asta/types.go"
Task: "Create domain types in internal/asta/types.go"
Task: "Create error types in internal/asta/errors.go"
```

## Parallel Example: User Stories (after Phase 2)

```bash
# Developer A works on US1 (search):
Task: "T009 Add SearchPapers method"
Task: "T010 Create bp asta search command"
Task: "T011 Implement search output formatting"

# Developer B works on US2 (snippets) in parallel:
Task: "T012 Add SnippetSearch method"
Task: "T013 Create bp asta snippet command"
Task: "T014 Implement snippet output formatting"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (types, errors, client)
2. Complete Phase 2: Foundational (parent command, helpers)
3. Complete Phase 3: User Story 1 (search papers)
4. **STOP and VALIDATE**: Test `bip asta search "phylogenetics" --human`
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 (search) → Test → Demo (MVP!)
3. Add US2 (snippets) → Test → Demo (P1 complete)
4. Add US3 (paper) + US4 (citations) → Test → Demo (P2 complete)
5. Add US5 (authors) → Test → Demo (Full feature)
6. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Reuse existing s2.ParsePaperID() for paper ID parsing
- Follow existing S2 command patterns for consistency
