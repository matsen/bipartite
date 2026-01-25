# Tasks: Narrative Digest

**Input**: Design documents from `/specs/012-narrative-digest/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md

**Tests**: Tests are not explicitly requested. Standard `go test ./...` coverage applies.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US0, US1, US2, US3)
- Include exact file paths in descriptions

## User Stories Summary

| Story | Priority | Title |
|-------|----------|-------|
| US0 | P0 (Prerequisite) | Safe Digest Default |
| US1 | P1 | Generate Narrative Digest |
| US2 | P1 | Theme Classification with Viral/Antibody Paragraphs |
| US3 | P1 | Hybrid Formatting |

Note: US1-US3 are all P1 and will be delivered together via the slash command. US0 must complete first.

---

## Phase 1: Setup

**Purpose**: No new infrastructure needed. Existing codebase already has required structure.

- [X] T001 Verify `claude` CLI is available for Haiku summarization (run `which claude`)
- [X] T002 Verify nexus narrative directory structure exists (`ls ~/re/nexus/narrative/`)

**Checkpoint**: Prerequisites verified.

---

## Phase 2: User Story 0 - Safe Digest Default (Priority: P0 - Prerequisite)

**Goal**: Change `bip digest` to preview-only by default; require `--post` flag to actually post to Slack.

**Independent Test**: Run `bip digest --channel foo` and verify it shows preview but does NOT post.

### Implementation for User Story 0

- [X] T003 [US0] Add `--post` flag to digest command in cmd/bip/digest.go
- [X] T004 [US0] Remove `--dry-run` flag from digest command in cmd/bip/digest.go
- [X] T005 [US0] Invert posting logic: default to preview-only, post only when `--post` is set in cmd/bip/digest.go
- [X] T006 [US0] Update command description and help text in cmd/bip/digest.go
- [X] T007 [US0] Update `/bip.digest` slash command to match new CLI behavior in .claude/commands/bip.digest.md

**Checkpoint**: `bip digest` now requires `--post` to actually send to Slack. Breaking change is intentional per constitution.

---

## Phase 3: User Story 0 Extension - Verbose Mode (Priority: P0)

**Goal**: Add `--verbose` flag to fetch PR/issue bodies and summarize using Claude Haiku.

**Independent Test**: Run `bip digest --channel foo --verbose` and verify summaries appear in output.

### Implementation for Verbose Mode

- [X] T008 [US0] Add `Body` and `Summary` fields to DigestItem struct in internal/flow/types.go
- [X] T009 [US0] Add `--verbose` flag to digest command in cmd/bip/digest.go
- [X] T010 [US0] Implement body fetching via `gh api` in cmd/bip/digest.go (modify fetchChannelActivity)
- [X] T011 [US0] Implement parallel Haiku summarization using bounded goroutines in internal/flow/llm.go
- [X] T012 [US0] Integrate summarization into digest output when `--verbose` is set in cmd/bip/digest.go
- [X] T013 [US0] Add fail-fast error handling if Claude CLI fails during summarization in internal/flow/llm.go

**Checkpoint**: `bip digest --verbose` fetches bodies and includes Haiku-generated summaries.

---

## Phase 4: User Stories 1-3 - Narrative Slash Command (Priority: P1)

**Goal**: Create `/bip.narrative` slash command that generates thematic, prose-style digests.

**Independent Test**: Run `/bip.narrative dasm2` from Claude Code and verify `narrative/dasm2/YYYY-MM-DD.md` is created.

Note: US1, US2, US3 are all implemented by the slash command. The LLM handles theme classification and hybrid formatting based on the config files and prompt.

### Slash Command Implementation

- [X] T014 [P] [US1] Create skill directory structure at .claude/skills/bip.narrative/
- [X] T015 [US1] Create main skill file at .claude/skills/bip.narrative/SKILL.md with:
  - Argument parsing for channel and flags (--since, --verbose)
  - Step 1: Run `bip digest --channel {channel} --since {since} [--verbose]`
  - Step 2: Read `narrative/preferences.md` (shared defaults)
  - Step 3: Read `narrative/{channel}.md` (themes, repo context)
  - Step 4: Construct prompt with raw activity, themes, and preferences
  - Step 5: Generate narrative using Claude Code's LLM
  - Step 6: Write output to `narrative/{channel}/YYYY-MM-DD.md`
  - Error handling for missing config files (FR-010, FR-023)
  - No-activity handling (FR-022)
- [X] T016 [US1] Symlink skill to global Claude skills directory (~/.claude/skills/)
- [X] T017 [US2] Include theme classification instructions in SKILL.md prompt template
- [X] T018 [US2] Include subheading instructions in SKILL.md (read from channel config's "Project-Specific Preferences" section; viral/antibody is one example, but subheadings are user-specified per channel)
- [X] T019 [US3] Include hybrid formatting instructions (bullets vs prose) in SKILL.md
- [X] T020 [US3] Include status prefix rules (In progress:, Open:) in SKILL.md
- [X] T021 [US1] Include "Looking Ahead" section generation rules in SKILL.md

**Checkpoint**: `/bip.narrative {channel}` generates complete themed narrative digest.

---

## Phase 5: Polish & Documentation

**Purpose**: Documentation updates and final validation

- [X] T022 [P] Update CLAUDE.md to document `/bip.narrative` alongside other bip commands
- [X] T023 [P] Update README.md if user-facing command documentation exists
- [X] T024 Run quickstart.md validation: test basic usage from nexus directory
- [X] T025 Verify all acceptance scenarios from spec.md:
  - US0: Preview by default, --post required to post
  - US1: Markdown file created at correct path
  - US2: Theme classification respects config
  - US3: Hybrid formatting applied correctly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - verify prerequisites
- **US0 Core (Phase 2)**: Depends on Setup - change default behavior
- **US0 Verbose (Phase 3)**: Depends on Phase 2 - adds --verbose
- **US1-3 Slash Command (Phase 4)**: Depends on Phase 3 - uses updated digest output
- **Polish (Phase 5)**: Depends on Phase 4 - documentation and validation

### Critical Path

```
T001-T002 (Setup)
    ↓
T003-T007 (US0 Core: Safe Default)
    ↓
T008-T013 (US0 Verbose Mode)
    ↓
T014-T021 (US1-3: Slash Command)
    ↓
T022-T025 (Polish)
```

### Parallel Opportunities

Within Phase 2 (US0 Core):
- T003-T006 are sequential (same file: cmd/bip/digest.go)
- T007 can run in parallel with T003-T006 (different file)

Within Phase 3 (US0 Verbose):
- T008 (types.go) and T011 (llm.go) can run in parallel
- T009-T010, T012 depend on T008
- T013 depends on T011

Within Phase 4 (Slash Command):
- T014 (directory) must complete first
- T015-T021 are all edits to SKILL.md - sequential within the file
- T016 (symlink) can run after T014

Within Phase 5 (Polish):
- T022-T023 can run in parallel
- T024-T025 depend on all prior phases

---

## Parallel Example: Phase 2 + Phase 3 Setup

```bash
# Can run in parallel (different files):
Task: "Add Body and Summary fields to DigestItem in internal/flow/types.go"
Task: "Implement parallel Haiku summarization in internal/flow/llm.go"
Task: "Update /bip.digest slash command in .claude/commands/bip.digest.md"
```

---

## Implementation Strategy

### MVP First (US0 Only)

1. Complete Phase 1: Setup verification
2. Complete Phase 2: US0 Core (safe default)
3. **STOP and VALIDATE**: Test `bip digest --channel foo` shows preview only
4. Deploy/merge if ready (breaking change documented)

### Full Feature Delivery

1. Complete Phase 1: Setup
2. Complete Phase 2: US0 Core → Test preview-only default
3. Complete Phase 3: US0 Verbose → Test `--verbose` flag
4. Complete Phase 4: US1-3 Slash Command → Test `/bip.narrative`
5. Complete Phase 5: Polish → Final validation

### File Summary

| File | Action | Phase |
|------|--------|-------|
| cmd/bip/digest.go | MODIFY | 2, 3 |
| internal/flow/types.go | MODIFY | 3 |
| internal/flow/llm.go | MODIFY | 3 |
| .claude/commands/bip.digest.md | MODIFY | 2 |
| .claude/skills/bip.narrative/SKILL.md | NEW | 4 |
| CLAUDE.md | MODIFY | 5 |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US0 is the prerequisite; US1-3 are delivered together via the slash command
- No LLM calls from Go code (slash command pattern for generation)
- Breaking change (--post required) is intentional per constitution
