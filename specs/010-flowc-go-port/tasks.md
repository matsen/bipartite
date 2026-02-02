# Tasks: flowc Go Port

**Feature Branch**: `30-flowc-go-port`
**Status**: Complete

## Phase 1: Setup & Core Infrastructure

- [X] T001 Create internal/flow/ package structure with shared types in internal/flow/types.go
- [X] T002 Implement sources.yml parsing in internal/flow/config.go (FR-001 to FR-005)
- [X] T003 Implement beads.jsonl loading in internal/flow/beads.go (FR-006 to FR-008)
- [X] T004 Implement duration parsing in internal/flow/duration.go (FR-045, FR-046)
- [X] T005 Implement GitHub reference parsing in internal/flow/ghref.go (FR-047 to FR-050)
- [X] T006 Implement relative time formatting in internal/flow/time.go (FR-054)

## Phase 2: GitHub API Integration

- [X] T007 Implement gh CLI wrapper for REST API calls in internal/flow/gh.go
- [X] T008 Implement GraphQL API support via gh CLI in internal/flow/gh.go
- [X] T009 Implement ball-in-my-court filtering logic in internal/flow/ballcourt.go (FR-041 to FR-044)

## Phase 3: Checkin Command (User Story 1 - P1)

- [X] T010 [US1] Create cmd/bip/checkin.go with cobra command structure
- [X] T011 [US1] Implement activity fetching in internal/flow/checkin/activity.go (FR-009)
- [X] T012 [US1] Implement ball-in-my-court filtering integration (FR-010)
- [X] T013 [US1] Implement --since, --repo, --category, --all flags (FR-011 to FR-014)
- [X] T014 [US1] Implement --summarize flag with LLM integration (FR-015)

## Phase 4: Board Commands (User Story 2 & 6 - P1/P2)

- [X] T015 [US2] Create cmd/bip/board.go with subcommand structure
- [X] T016 [US2] Implement board cache in internal/flow/board/cache.go (FR-022)
- [X] T017 [US2] Implement board API in internal/flow/board/api.go
- [X] T018 [US6] Implement board list command (FR-016)
- [X] T019 [US6] Implement board add command (FR-017)
- [X] T020 [US6] Implement board move command (FR-018)
- [X] T021 [US6] Implement board remove command (FR-019)
- [X] T022 [US2] Implement board sync command (FR-020, FR-021)
- [X] T023 [US2] Implement board refresh-cache command (FR-022)

## Phase 5: Spawn Command (User Story 3 - P2)

- [X] T024 [US3] Create cmd/bip/spawn.go with cobra command structure
- [X] T025 [US3] Implement tmux utilities in internal/flow/spawn/tmux.go (FR-026)
- [X] T026 [US3] Implement issue/PR fetching and type detection (FR-023 to FR-025)
- [X] T027 [US3] Implement context loading from sources.yml (FR-027)
- [X] T028 [US3] Implement --prompt flag (FR-028)
- [X] T029 [US3] Implement prompt building for issues and PRs

## Phase 6: Digest Command (User Story 4 - P2)

- [X] T030 [US4] Create cmd/bip/digest.go with cobra command structure
- [X] T031 [US4] Implement activity fetching by channel in internal/flow/digest/activity.go (FR-029)
- [X] T032 [US4] Implement --since duration parsing (FR-030)
- [X] T033 [US4] Implement LLM digest generation in internal/flow/llm.go (FR-031, FR-055 to FR-057)
- [X] T034 [US4] Implement digest postprocessing (FR-032)
- [X] T035 [US4] Implement Slack webhook posting in internal/flow/slack.go (FR-033)
- [X] T036 [US4] Implement --post-to flag (FR-034)

## Phase 7: Tree Command (User Story 5 - P3)

- [X] T037 [US5] Create cmd/bip/tree.go with cobra command structure
- [X] T038 [US5] Implement tree building from beads hierarchy in internal/flow/tree/tree.go
- [X] T039 [US5] Implement HTML generation (FR-035, FR-039, FR-040)
- [X] T040 [US5] Implement --since highlighting (FR-036)
- [X] T041 [US5] Implement --output flag (FR-037)
- [X] T042 [US5] Implement --open flag with browser launch (FR-038)

## Phase 8: Output Formatting

- [X] T043 Implement comment formatting with truncation (FR-052)
- [X] T044 Implement PR files formatting with truncation (FR-053)
- [X] T045 Implement JSON output support where applicable (FR-051)

## Phase 9: Testing

- [X] T046 Port test_config.py tests to internal/flow/config_test.go (12 tests)
- [X] T047 Port test_activity.py tests to internal/flow/ballcourt_test.go (20 tests)
- [X] T048 Port test_digest.py tests to internal/flow/duration_test.go (9 tests)
- [X] T049 Port test_issue.py tests to internal/flow/ghref_test.go and time_test.go (34 tests)
- [X] T050 Port test_llm.py tests to internal/flow/llm_test.go (31 tests)
- [X] T051 Port test_slack.py tests to internal/flow/slack_test.go (4 tests)

## Phase 10: Polish & Integration

- [X] T052 Add integration tests for full command flows
- [X] T053 Update README.md with new bip commands documentation
- [X] T054 Consolidate skills: merge flowc skill into bip skill
- [X] T055 Remove Python flowc code and dependencies

## Dependencies

```
T001 → T002, T003, T004, T005, T006 (shared types needed first)
T007, T008 → T009 (GitHub API needed for ball-in-court)
T002, T003, T009 → T010-T014 (checkin needs config, beads, filtering)
T002, T007, T008 → T015-T023 (board needs config and GitHub APIs)
T002, T005, T007 → T024-T029 (spawn needs config, ghref, GitHub)
T002, T004, T007, T033 → T030-T036 (digest needs config, duration, GitHub, LLM)
T003 → T037-T042 (tree needs beads loading)
```

## Parallel Execution Opportunities

- T002, T003, T004, T005, T006 can run in parallel (after T001)
- T007, T008 can run in parallel
- T018-T021 (board subcommands) can run in parallel after T017
- T046-T051 (test porting) can run in parallel

## Success Criteria

- [X] All 110 Python tests have equivalent passing Go tests
- [X] CLI interface backward compatible: `bip checkin` works like `flowc checkin`
- [X] Ball-in-my-court logic matches Python implementation exactly
- [X] Single Go binary with no Python dependencies
