# Tasks: bip scout — Remote Server Availability

**Input**: Design documents from `/specs/016-bip-scout/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Included — plan.md specifies unit tests for parsing/config and integration tests for SSH.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependencies, and shared type definitions

- [X] T001 Add `golang.org/x/crypto/ssh` and `gopkg.in/yaml.v3` dependencies via `go get`
- [X] T002 Create shared type definitions (ScoutConfig, ServerEntry, SSHConfig, Server, ScoutResult, ServerStatus, ServerMetrics, GPUInfo) in `internal/scout/types.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config loading and SSH connectivity — MUST complete before any user story

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 [P] Implement YAML config loading, validation, and pattern expansion (`{NN..MM}` brace expansion) in `internal/scout/config.go`
- [X] T004 [P] Implement config unit tests with fixture YAML (valid configs, invalid configs, pattern expansion edge cases) in `internal/scout/config_test.go`
- [X] T005 [P] Implement SSH agent auth discovery (SSH_AUTH_SOCK), ProxyJump dialing (jump host → target), session management with connect timeout, and actionable error messages (FR-015) in `internal/scout/ssh.go`
- [X] T006 [P] Implement SSH unit tests (agent auth detection, error message formatting) in `internal/scout/ssh_test.go`
- [X] T007 [P] Implement remote command execution: build combined command string with `___SCOUT_DELIM___` separators, execute in single SSH session, split output, and parse each metric (CPU float, memory float, load avg 3-tuple, GPU util ints, GPU mem int pairs) in `internal/scout/metrics.go`
- [X] T008 [P] Implement metrics parser tests with real command output fixtures (normal output, missing nvidia-smi, unexpected formats, empty output) in `internal/scout/metrics_test.go`

**Checkpoint**: Foundation ready — config loading, SSH connection, and metrics parsing all functional and tested

---

## Phase 3: User Story 1 — Check All Servers (Priority: P1) 🎯 MVP

**Goal**: `bip scout` checks all configured servers via SSH in parallel and outputs JSON with status, CPU, memory, load, and GPU data

**Independent Test**: Run `bip scout` from a nexus directory with valid `servers.yml` and verify structured JSON output

### Tests for User Story 1

- [X] T009 [P] [US1] Add test for parallel server checking with bounded concurrency (semaphore of 5) and result aggregation in `internal/scout/metrics_test.go`

### Implementation for User Story 1

- [X] T010 [US1] Implement parallel server orchestration: fan-out with bounded semaphore (max 5 concurrent), collect ServerStatus results, assemble ScoutResult in `internal/scout/metrics.go`
- [X] T011 [US1] Create Cobra command `bip scout` with `--server` and `--human` flags, config loading from CWD `servers.yml`, exit code handling (0/1/2 per cli.md contract), and JSON output (pretty-printed, 2-space indent) in `cmd/bip/scout.go`
- [X] T012 [US1] Wire scout command into root command in `cmd/bip/main.go`

**Checkpoint**: `bip scout` outputs JSON for all servers — US1 fully functional

---

## Phase 4: User Story 4 — Server Configuration via YAML (Priority: P1)

**Goal**: Users define servers and SSH params in `servers.yml` with pattern expansion

**Independent Test**: Create `servers.yml` with `beetle{01..05}` pattern, run `bip scout`, verify all 5 expanded servers appear

**Note**: The config implementation is in Phase 2 (T003/T004). This phase covers integration validation and edge case handling.

### Implementation for User Story 4

- [X] T013 [P] [US4] Add config integration tests: validate pattern expansion produces correct server list, missing `servers.yml` exits with code 2 and helpful error, malformed YAML exits with code 2, server entry with neither name nor pattern is rejected, proxy_jump and connect_timeout flow through to SSH in `internal/scout/config_test.go`

**Checkpoint**: Config loading fully validated — patterns expand, errors are clear

---

## Phase 5: User Story 2 — Human-Readable Table Output (Priority: P2)

**Goal**: `bip scout --human` displays a terminal-friendly table with aligned columns

**Independent Test**: Run `bip scout --human` and verify aligned table with Server, Status, CPU, Memory, Load Avg, GPU Usage, GPU Memory columns

### Tests for User Story 2

- [X] T014 [P] [US2] Add table formatting tests: online server with GPUs (aggregated avg util, summed memory), online server without GPUs (dashes), offline server (all dashes), column alignment with varying widths in `internal/scout/format_test.go`

### Implementation for User Story 2

- [X] T015 [US2] Implement human-readable table formatter: column headers, left-aligned server/status, right-aligned numeric columns, GPU aggregation (average utilization with count, summed memory used/total with percentage), dash placeholders for offline/no-GPU in `internal/scout/format.go`
- [X] T016 [US2] Wire `--human` flag to table formatter in `cmd/bip/scout.go` (dispatch to format.go when flag is set)

**Checkpoint**: Both JSON and human table output work — US2 independently testable

---

## Phase 6: User Story 3 — Check a Single Server (Priority: P2)

**Goal**: `bip scout --server beetle01` checks only the named server

**Independent Test**: Run `bip scout --server beetle01` and verify JSON for only that server

### Implementation for User Story 3

- [X] T017 [US3] Implement `--server` filtering: match flag value against expanded server names, exit code 2 with error if no match, pass single-server list to orchestrator in `cmd/bip/scout.go`
- [X] T018 [US3] Add test for `--server` flag: valid name returns single-server result, unknown name produces clear error in `internal/scout/config_test.go`

**Checkpoint**: All three output modes work: all-JSON, all-human, single-server

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Error handling edge cases, documentation, and final validation

- [X] T019 [P] Handle edge cases: nvidia-smi not installed on has_gpu server (null GPU metrics, server still online), unparseable metric output (partial results with error indication), all servers offline (full list with all offline), zero-pattern expansion (config error) in `internal/scout/metrics.go` and `internal/scout/config.go`
- [X] T020 [P] Run `go vet ./...` and `go fmt ./...` to verify code quality
- [X] T021 [P] Update README.md with `bip scout` command documentation and example output
- [X] T022 Run quickstart.md validation: build binary, create test `servers.yml`, execute `bip scout`, `bip scout --human`, `bip scout --server <name>`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — core JSON output
- **US4 (Phase 4)**: Config validation; can run in parallel with US1 (tests only, implementation already in Phase 2)
- **US2 (Phase 5)**: Depends on Phase 3 (needs ScoutResult to format)
- **US3 (Phase 6)**: Depends on Phase 3 (needs server list filtering before orchestration)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational — the core feature
- **US4 (P1)**: Config already built in Foundational; this phase adds integration tests — can parallel with US1
- **US2 (P2)**: Depends on US1 (formats the same ScoutResult)
- **US3 (P2)**: Depends on US1 (filters server list before calling same orchestrator)

### Within Each User Story

- Tests written alongside implementation (no strict TDD ordering specified)
- Types → Config → SSH → Metrics → Orchestration → Command → Formatting
- Foundation must complete before story work

### Parallel Opportunities

**Phase 2 (all [P])**:
```
T003 (config.go) | T005 (ssh.go) | T007 (metrics.go)  — different files
T004 (config_test) | T006 (ssh_test) | T008 (metrics_test) — different files
```

**Phase 3-6 partial parallelism**:
```
T009 (US1 test) | T013 (US4 config integration tests) | T014 (US2 format tests) — different files
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 4 Only)

1. Complete Phase 1: Setup (dependencies + types)
2. Complete Phase 2: Foundational (config, SSH, metrics)
3. Complete Phase 3: US1 — `bip scout` with JSON output
4. Complete Phase 4: US4 — Config validation
5. **STOP and VALIDATE**: `bip scout` produces correct JSON for all configured servers

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → `bip scout` works with JSON → MVP!
3. Add US4 → Config edge cases validated
4. Add US2 → `--human` table output works
5. Add US3 → `--server` filtering works
6. Polish → Edge cases, docs, quickstart validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- All SSH connections route through optional ProxyJump host
- Bounded semaphore (5) prevents connection flooding
- Single SSH session per server (FR-009)
- `servers.yml` lives in nexus directory (CWD), discovered same as `sources.yml`
- Exit codes: 0 (success), 1 (system error), 2 (config error) per cli.md contract
