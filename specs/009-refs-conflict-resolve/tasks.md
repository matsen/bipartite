# Tasks: Domain-Aware Conflict Resolution

**Input**: Design documents from `/specs/009-refs-conflict-resolve/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Tests ARE included in this task list for thorough validation of conflict resolution logic.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Source code**: `cmd/bip/`, `internal/` at repository root
- **Tests**: Co-located with source (`*_test.go` files)
- **Test fixtures**: `testdata/conflict/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create internal/conflict package directory structure
- [ ] T002 [P] Create internal/conflict/types.go with ConflictRegion, PaperMatch, FieldConflict, ResolutionPlan, ParseError types
- [ ] T003 [P] Add ResolveResult, UnresolvedInfo, ResolveOp types to cmd/bip/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**Critical**: No user story work can begin until this phase is complete

- [ ] T004 Create testdata/conflict/ directory for test fixtures
- [ ] T005 [P] Create test fixture testdata/conflict/simple_ours_better.jsonl (ours has more fields)
- [ ] T006 [P] Create test fixture testdata/conflict/simple_theirs_better.jsonl (theirs has more fields)
- [ ] T007 [P] Create test fixture testdata/conflict/complementary_merge.jsonl (non-overlapping fields)
- [ ] T008 [P] Create test fixture testdata/conflict/true_conflict.jsonl (same field, different values)
- [ ] T009 [P] Create test fixture testdata/conflict/multiple_papers.jsonl (multiple conflict regions)
- [ ] T010 [P] Create test fixture testdata/conflict/malformed_markers.jsonl (invalid conflict markers)
- [ ] T011 [P] Create test fixture testdata/conflict/no_conflicts.jsonl (clean file without markers)
- [ ] T012 Implement conflict marker parser in internal/conflict/parser.go (state machine: NORMAL, IN_OURS, IN_THEIRS)
- [ ] T013 Write parser tests in internal/conflict/parser_test.go (table-driven, use fixtures)
- [ ] T014 Implement paper matching by DOI/ID in internal/conflict/matcher.go
- [ ] T015 Write matcher tests in internal/conflict/matcher_test.go

**Checkpoint**: Foundation ready - parser and matcher functional with tests passing

---

## Phase 3: User Story 1 - Auto-Resolve Simple Metadata Conflicts (Priority: P1)

**Goal**: Enable automatic resolution of conflicts where one version has more complete metadata

**Independent Test**: Create refs.jsonl with git conflict markers, run `bip resolve`, verify output contains merged paper with most complete metadata

### Tests for User Story 1

- [ ] T016 [P] [US1] Write resolver tests for completeness comparison in internal/conflict/resolver_test.go
- [ ] T017 [P] [US1] Write resolver tests for complementary metadata merging in internal/conflict/resolver_test.go
- [ ] T018 [P] [US1] Write resolver tests for different-paper preservation in internal/conflict/resolver_test.go

### Implementation for User Story 1

- [ ] T019 [US1] Implement completeness scoring (priority field comparison: Abstract > Authors > Venue > Published > DOI) in internal/conflict/resolver.go
- [ ] T020 [US1] Implement metadata merging logic (field-by-field, union for slices, most-specific for dates) in internal/conflict/resolver.go
- [ ] T021 [US1] Implement paper-only-on-one-side handling (ActionAddOurs, ActionAddTheirs) in internal/conflict/resolver.go
- [ ] T022 [US1] Implement author list comparison (longer wins, same length different = conflict) in internal/conflict/resolver.go
- [ ] T023 [US1] Create resolve command scaffolding in cmd/bip/resolve.go with cobra command definition
- [ ] T024 [US1] Implement resolve command main logic (read file, parse conflicts, resolve, write output) in cmd/bip/resolve.go
- [ ] T025 [US1] Implement JSON and --human output formatting in cmd/bip/resolve.go
- [ ] T026 [US1] Write integration tests for bip resolve in cmd/bip/resolve_test.go

**Checkpoint**: User Story 1 complete - basic `bip resolve` auto-resolves completeness conflicts

---

## Phase 4: User Story 2 - Preview Conflict Resolution (Priority: P1)

**Goal**: Allow users to preview what `bip resolve` would do without modifying files

**Independent Test**: Run `bip resolve --dry-run` and verify output shows conflicts detected and proposed resolutions without modifying files

### Tests for User Story 2

- [ ] T027 [P] [US2] Write dry-run test cases (file unchanged, output shows plan) in cmd/bip/resolve_test.go
- [ ] T028 [P] [US2] Write no-conflicts detection test in cmd/bip/resolve_test.go

### Implementation for User Story 2

- [ ] T029 [US2] Add --dry-run flag to resolve command in cmd/bip/resolve.go
- [ ] T030 [US2] Implement dry-run output formatting (list conflicts with proposed resolutions) in cmd/bip/resolve.go
- [ ] T031 [US2] Implement no-conflicts-detected message in cmd/bip/resolve.go
- [ ] T032 [US2] Implement unresolvable conflicts preview (shows which need interactive) in cmd/bip/resolve.go

**Checkpoint**: User Story 2 complete - `bip resolve --dry-run` previews all operations

---

## Phase 5: User Story 3 - Interactive Resolution for True Conflicts (Priority: P2)

**Goal**: Enable interactive prompts for conflicts that cannot be auto-resolved

**Independent Test**: Create conflict with true field-level conflicts, run `bip resolve --interactive`, verify prompts appear for each unresolvable field

### Tests for User Story 3

- [ ] T033 [P] [US3] Write test fixture testdata/conflict/interactive_needed.jsonl (multiple true conflicts)
- [ ] T034 [P] [US3] Write interactive prompt unit tests (mock stdin) in internal/conflict/interactive_test.go

### Implementation for User Story 3

- [ ] T035 [US3] Implement interactive prompt handler in internal/conflict/interactive.go (numbered options [1/2], input validation)
- [ ] T036 [US3] Add --interactive flag to resolve command in cmd/bip/resolve.go
- [ ] T037 [US3] Implement interactive resolution workflow (auto-resolve what possible, prompt for rest) in cmd/bip/resolve.go
- [ ] T038 [US3] Implement progress indication ("Resolving conflict 2 of 5...") in cmd/bip/resolve.go
- [ ] T039 [US3] Implement exit code 1 when unresolvable conflicts exist without --interactive in cmd/bip/resolve.go
- [ ] T040 [US3] Write interactive integration tests in cmd/bip/resolve_test.go

**Checkpoint**: User Story 3 complete - `bip resolve --interactive` handles all conflict types

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T041 [P] Implement exit code 3 for malformed conflict markers in cmd/bip/resolve.go
- [ ] T042 [P] Add actionable error messages with line numbers for parse errors in internal/conflict/parser.go
- [ ] T043 [P] Handle edge case: refs.jsonl doesn't exist or is empty in cmd/bip/resolve.go
- [ ] T044 [P] Handle edge case: paper has no DOI on either side (match by ID fallback) in internal/conflict/matcher.go
- [ ] T045 Register resolve command in cmd/bip/root.go
- [ ] T046 Run quickstart.md validation scenarios manually
- [ ] T047 [P] Verify all tests pass with go test ./...
- [ ] T048 [P] Run go fmt ./... and go vet ./...

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 priority and can proceed in parallel after Foundation
  - US3 (P2) can start after Foundation but may benefit from US1 completion
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Core resolution logic
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Can develop in parallel with US1
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Builds on US1 resolution logic

### Within Each User Story

- Tests written first (ensures they fail before implementation)
- Types before logic
- Internal packages before command layer
- Core implementation before integration

### Parallel Opportunities

Setup phase:
- T002 and T003 can run in parallel (different files)

Foundational phase:
- T005-T011 (all fixtures) can run in parallel
- After fixtures: T012-T013 (parser) and T014-T015 (matcher) can run in parallel

User Story 1:
- T016, T017, T018 (tests) can run in parallel

User Story 2:
- T027, T028 (tests) can run in parallel

User Story 3:
- T033, T034 (fixtures and tests) can run in parallel

Polish:
- T041, T042, T043, T044 can run in parallel (different concerns)
- T047, T048 can run in parallel

---

## Parallel Example: Foundational Phase

```bash
# Launch all fixtures in parallel:
Task: "Create test fixture testdata/conflict/simple_ours_better.jsonl"
Task: "Create test fixture testdata/conflict/simple_theirs_better.jsonl"
Task: "Create test fixture testdata/conflict/complementary_merge.jsonl"
Task: "Create test fixture testdata/conflict/true_conflict.jsonl"
Task: "Create test fixture testdata/conflict/multiple_papers.jsonl"
Task: "Create test fixture testdata/conflict/malformed_markers.jsonl"
Task: "Create test fixture testdata/conflict/no_conflicts.jsonl"

# Then launch parser and matcher in parallel:
Task: "Implement conflict marker parser in internal/conflict/parser.go"
Task: "Implement paper matching by DOI/ID in internal/conflict/matcher.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (types definition)
2. Complete Phase 2: Foundational (parser, matcher, fixtures)
3. Complete Phase 3: User Story 1 (auto-resolve)
4. Complete Phase 4: User Story 2 (dry-run)
5. **STOP and VALIDATE**: Test basic resolution and preview independently
6. This covers the most common use cases

### Full Feature

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Basic resolution works
3. Add User Story 2 → Preview capability
4. Add User Story 3 → Interactive mode for edge cases
5. Polish → Error handling, edge cases, documentation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests use table-driven approach with real fixtures (per constitution)
- Commit after each task or logical group
- Exit codes: 0 (success), 1 (unresolvable conflicts), 3 (data/parse error)
