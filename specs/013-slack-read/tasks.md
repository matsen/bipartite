# Tasks: Slack Channel Reading

**Input**: Design documents from `/specs/013-slack-read/`
**Prerequisites**: plan.md, spec.md

**Tests**: Integration tests are specified in plan.md (tests/slack_integration_test.go). Unit tests in internal/flow/slack_test.go.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

This project uses:
- **CLI commands**: `cmd/bip/`
- **Internal packages**: `internal/flow/`
- **Integration tests**: `tests/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Extend existing Slack module with reading infrastructure

- [x] T001 Add SlackClient struct and constructor in internal/flow/slack.go
- [x] T002 [P] Add Message, HistoryResponse, Period data types in internal/flow/slack.go
- [x] T003 [P] Add ChannelConfig, ChannelsResponse, ChannelInfo data types in internal/flow/slack.go
- [x] T004 Implement user cache load/save functions (loadUserCache, saveUserCache) in internal/flow/slack.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core Slack API interaction that MUST be complete before user story commands

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement GetUsers() API call for user ID to name mapping in internal/flow/slack.go
- [x] T006 Implement GetChannelHistory() API call for message fetching in internal/flow/slack.go
- [x] T007 [P] Add LoadChannelConfig() to read channels from sources.json in internal/flow/slack.go
- [x] T008 Create parent `bip slack` command in cmd/bip/slack.go (following s2.go pattern)

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Fetch Channel History (Priority: P1) 🎯 MVP

**Goal**: Users can fetch recent messages from a Slack channel with time filtering and output format options

**Independent Test**: Run `bip slack history <channel>` against a configured channel and verify messages are returned with user attribution and timestamps

### Implementation for User Story 1

- [x] T009 [US1] Implement `bip slack history <channel>` subcommand in cmd/bip/slack_history.go
- [x] T010 [US1] Add --days flag (default: 14) for time period filtering in cmd/bip/slack_history.go
- [x] T011 [US1] Add --since flag (YYYY-MM-DD) that overrides --days in cmd/bip/slack_history.go
- [x] T012 [US1] Add --limit flag (default: 100) for message count limit in cmd/bip/slack_history.go
- [x] T013 [US1] Add --human flag for human-readable markdown output in cmd/bip/slack_history.go
- [x] T014 [US1] Implement JSON output format (default) with channel, period, messages in cmd/bip/slack_history.go
- [x] T015 [US1] Implement human-readable markdown output with headers per user/date in cmd/bip/slack_history.go
- [x] T016 [US1] Add error handling for channel not in configuration (exit code 2) in cmd/bip/slack_history.go
- [x] T017 [US1] Add error handling for bot not member of channel (exit code 3) in cmd/bip/slack_history.go
- [x] T018 [US1] Add error handling for missing slack_bot_token (exit code 1) in cmd/bip/slack_history.go

**Checkpoint**: User Story 1 should be fully functional - can fetch history from configured channels

---

## Phase 4: User Story 2 - List Available Channels (Priority: P2)

**Goal**: Users can discover which Slack channels are configured and available for querying

**Independent Test**: Run `bip slack channels` and verify the configured channels are listed with their IDs and purposes

### Implementation for User Story 2

- [x] T019 [US2] Implement `bip slack channels` subcommand in cmd/bip/slack_channels.go
- [x] T020 [US2] Add --human flag for human-readable table output in cmd/bip/slack_channels.go
- [x] T021 [US2] Implement JSON output format (default) with channels array in cmd/bip/slack_channels.go
- [x] T022 [US2] Implement human-readable table format output in cmd/bip/slack_channels.go

**Checkpoint**: User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Agent-Driven Goal Analysis (Priority: P3)

**Goal**: AI agents can programmatically fetch and parse goal/retrospective data for automated analysis

**Independent Test**: Agent script calls `bip slack history` for configured channels and successfully parses JSON output

### Implementation for User Story 3

- [x] T023 [US3] Verify JSON output includes all required fields (timestamp, user, date, text) in internal/flow/slack.go
- [x] T024 [US3] Add integration test for agent workflow parsing in tests/slack_integration_test.go

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Testing, validation, and documentation

- [x] T025 [P] Create unit tests for SlackClient methods in internal/flow/slack_test.go
- [x] T026 [P] Create integration tests against real Slack workspace in tests/slack_integration_test.go
- [x] T027 Verify all error conditions produce actionable error messages (SC-002)
- [x] T028 Verify JSON output is valid and parseable by standard tools (SC-003)
- [x] T029 Update README.md with new `bip slack` commands

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can proceed in priority order (P1 → P2 → P3)
  - US3 depends on US1 being complete (verifies its output format)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Independent of US1
- **User Story 3 (P3)**: Depends on US1 being complete (validates US1's JSON output for agent use)

### Within Each User Story

- Core implementation before error handling
- JSON output before human-readable output
- All functionality before moving to next priority

### Parallel Opportunities

- T002 and T003 can run in parallel (different data types)
- T007 can run in parallel with T005/T006 (reads config, doesn't depend on API)
- T025 and T026 can run in parallel (different test files)

---

## Parallel Example: Phase 1 Setup

```bash
# Launch data type definitions together:
Task: "Add Message, HistoryResponse, Period data types in internal/flow/slack.go"
Task: "Add ChannelConfig, ChannelsResponse, ChannelInfo data types in internal/flow/slack.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test `bip slack history <channel>` independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test `bip slack history` → MVP ready!
3. Add User Story 2 → Test `bip slack channels` → Discovery added
4. Add User Story 3 → Validate agent workflow → Feature complete
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files or independent sections, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Sources.json configuration format defined in plan.md Phase 0 R3
