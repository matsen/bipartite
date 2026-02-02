# Feature Specification: flowc Go Port

**Feature Branch**: `010-flowc-go-port`
**Created**: 2026-01-23
**Status**: Draft
**Input**: Port the flowc Python CLI to Go, integrating it into the bip CLI. The goal is a single Go binary with no Python dependencies.

## Overview

flowc is a CLI for managing GitHub activity and project boards, centered around a "nexus" directory containing:
- `sources.yml` - Repository list, board mappings, and channel configuration
- `.beads/issues.jsonl` - Local issue tracker (beads) with priorities and GitHub references
- `config.yml` - Local path configuration (code directory, writing directory)
- `context/` - Project context files for issue review

The port will add these commands directly to bip (e.g., `bip checkin`, `bip board`, `bip spawn`).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Check In on GitHub Activity (Priority: P1)

A researcher starts their day and wants to see what needs attention across all their tracked repositories. They want to focus only on items requiring their action ("ball in my court"), not items where they're waiting for others.

**Why this priority**: This is the primary daily workflow - understanding what needs attention across multiple repos.

**Independent Test**: Run `bip checkin` and verify it shows issues/PRs where the user needs to take action.

**Acceptance Scenarios**:

1. **Given** repos with active issues/PRs, **When** user runs `bip checkin`, **Then** only items requiring user action are shown (ball-in-my-court filtering)
2. **Given** an issue I created with no comments, **When** I run checkin, **Then** it is hidden (waiting for feedback)
3. **Given** an issue someone else created with no comments, **When** I run checkin, **Then** it is shown (needs my review)
4. **Given** their issue where I commented last, **When** I run checkin, **Then** it is hidden (waiting for their reply)
5. **Given** my issue where they commented last, **When** I run checkin, **Then** it is shown (they replied)
6. **Given** `--all` flag, **When** I run checkin, **Then** all activity is shown regardless of ball-in-my-court status

---

### User Story 2 - Sync Board with Beads (Priority: P1)

A researcher uses GitHub project boards to track priorities and wants to ensure their board reflects their P0 beads (highest priority items). They want to see mismatches between beads and the board.

**Why this priority**: Board sync is critical for maintaining alignment between local planning (beads) and GitHub project management.

**Independent Test**: Run `bip board sync` and verify it correctly matches P0 beads with board issues via `GitHub: org/repo#N` pattern in bead descriptions.

**Acceptance Scenarios**:

1. **Given** P0 beads with GitHub references not on the board, **When** user runs `bip board sync`, **Then** these are listed as "P0 beads not on board"
2. **Given** board issues without matching P0 beads, **When** user runs `bip board sync`, **Then** these are listed as "Board issues without P0 bead"
3. **Given** P0 beads with GitHub references that ARE on the board, **When** user runs sync, **Then** these are not listed in either category
4. **Given** `--fix` flag, **When** user runs sync, **Then** missing P0s are automatically added to the board

---

### User Story 3 - Spawn Review Session (Priority: P2)

A researcher wants to review a specific GitHub issue or PR with full context. They want to spawn a tmux window with the issue loaded and relevant project context.

**Why this priority**: Important for deep work on specific issues, but less frequent than daily checkin.

**Independent Test**: Run `bip spawn org/repo#123` and verify tmux window is created with correct context.

**Acceptance Scenarios**:

1. **Given** a valid issue reference `org/repo#123`, **When** user runs spawn, **Then** a tmux window is created with the issue context
2. **Given** a GitHub URL, **When** user runs spawn, **Then** URL is parsed and tmux window is created
3. **Given** a repo with context defined in sources.yml, **When** user runs spawn, **Then** project context is prepended to the prompt
4. **Given** `--prompt` without issue reference, **When** user runs spawn, **Then** an adhoc tmux window is created with the prompt as context

---

### User Story 4 - Generate Activity Digest (Priority: P2)

A researcher wants to generate a summary of activity across repos in a channel (e.g., "dasm2") and optionally post it to Slack.

**Why this priority**: Useful for team communication but less critical than individual workflow.

**Independent Test**: Run `bip digest --channel dasm2 --since 1w` and verify summary is generated.

**Acceptance Scenarios**:

1. **Given** repos with activity, **When** user runs `bip digest --channel dasm2`, **Then** an LLM-generated summary is produced
2. **Given** a Slack webhook configured, **When** user runs digest with `--post-to`, **Then** summary is posted to Slack
3. **Given** date range `--since 1w`, **When** user runs digest, **Then** only activity from the last week is included

---

### User Story 5 - View Beads Tree (Priority: P3)

A researcher wants to visualize their beads hierarchy as an interactive HTML tree, with optional highlighting of recently created items.

**Why this priority**: Visualization is helpful but not critical path for daily work.

**Independent Test**: Run `bip tree --open` and verify HTML tree opens in browser.

**Acceptance Scenarios**:

1. **Given** beads in `.beads/issues.jsonl`, **When** user runs tree, **Then** HTML is generated showing hierarchical structure
2. **Given** `--since 2026-01-20`, **When** user runs tree, **Then** beads created after that date are highlighted
3. **Given** `--open` flag, **When** user runs tree, **Then** HTML opens in default browser
4. **Given** beads with `GitHub: org/repo#N` in description, **When** user runs tree, **Then** items link to GitHub issues

---

### User Story 6 - Manage Board Items (Priority: P2)

A researcher wants to add, move, or remove issues from their GitHub project board via CLI.

**Why this priority**: Essential for board management workflow.

**Acceptance Scenarios**:

1. **Given** an issue number and repo, **When** user runs `bip board add 123 --repo org/repo`, **Then** issue is added to the board
2. **Given** an issue on the board, **When** user runs `bip board move 123 --status active`, **Then** issue is moved to the "active" column
3. **Given** an issue on the board, **When** user runs `bip board remove 123`, **Then** issue is removed from the board
4. **Given** a board, **When** user runs `bip board list`, **Then** all board items are listed by status

---

## Requirements *(mandatory)*

### Functional Requirements

#### Configuration (sources.yml)

- **FR-001**: System MUST read repository list from `sources.yml` in current directory
- **FR-002**: System MUST support repos as either strings (`"org/repo"`) or objects (`{"repo": "org/repo", "channel": "dasm2"}`)
- **FR-003**: System MUST read board mappings from `sources.yml` `boards` key
- **FR-004**: System MUST read project context paths from `sources.yml` `context` key
- **FR-005**: System MUST validate nexus directory (error if sources.yml not found)

#### Beads Integration

- **FR-006**: System MUST read beads from `.beads/issues.jsonl`
- **FR-007**: System MUST identify P0 beads by `priority` field value of 0 (integer, not text)
- **FR-008**: System MUST extract GitHub references from bead descriptions using pattern `GitHub: org/repo#N`

#### Checkin Command

- **FR-009**: System MUST fetch recent issues and PRs from tracked repos via GitHub API
- **FR-010**: System MUST implement ball-in-my-court filtering (see Ball-in-Court Logic below)
- **FR-011**: System MUST support `--since` flag for time-based filtering (e.g., 2d, 12h, 1w)
- **FR-012**: System MUST support `--repo` flag to filter to single repo
- **FR-013**: System MUST support `--category` flag to filter by sources.yml category
- **FR-014**: System MUST support `--all` flag to disable ball-in-my-court filtering
- **FR-015**: System MUST support `--summarize` flag to generate LLM summaries

#### Board Commands

- **FR-016**: System MUST provide `board list` with `--status` and `--label` filters
- **FR-017**: System MUST provide `board add <issue> --repo <repo> [--status <status>]`
- **FR-018**: System MUST provide `board move <issue> --status <status>`
- **FR-019**: System MUST provide `board remove <issue>`
- **FR-020**: System MUST provide `board sync` to compare P0 beads with board items
- **FR-021**: System MUST provide `board sync --fix` to auto-add missing P0s to board
- **FR-022**: System MUST provide `board refresh-cache` to refresh cached board metadata

#### Spawn Command

- **FR-023**: System MUST parse GitHub references in `org/repo#N` format
- **FR-024**: System MUST parse GitHub URLs (issues and pull requests)
- **FR-025**: System MUST determine issue type (issue vs PR) from URL or API lookup
- **FR-026**: System MUST spawn tmux window with appropriate context
- **FR-027**: System MUST prepend project context if defined for the repo
- **FR-028**: System MUST support `--prompt` flag for custom prompt override
- **FR-028b**: System MUST allow `--prompt` without issue reference for adhoc sessions (window named `adhoc-YYYY-MM-DD-HHMMSS`)

#### Digest Command

- **FR-029**: System MUST fetch activity for repos in specified channel
- **FR-030**: System MUST support `--since` duration (default 1w)
- **FR-031**: System MUST generate LLM summary in Slack mrkdwn format
- **FR-032**: System MUST postprocess digest to add repo names and contributor @mentions
- **FR-033**: System MUST post to Slack if webhook configured (config: `slack_webhooks.{channel}`)
- **FR-034**: System MUST support `--post-to` to override destination channel

#### Tree Command

- **FR-035**: System MUST generate interactive HTML tree from beads hierarchy
- **FR-036**: System MUST support `--since` for highlighting recently created beads
- **FR-037**: System MUST support `--output` to specify output file path
- **FR-038**: System MUST support `--open` to open in browser
- **FR-039**: System MUST render GitHub links for beads with `GitHub:` references
- **FR-040**: System MUST include keyboard shortcuts (c=collapse, e=expand)

### Ball-in-Court Logic

The ball-in-my-court filter determines whether an item needs the user's attention:

| Item Author | Last Commenter | Visible? | Reason |
|-------------|----------------|----------|--------|
| Them | (none) | Yes | Needs review |
| Them | Me | No | Waiting for their reply |
| Them | Them | Yes | They pinged again |
| Me | (none) | No | Waiting for feedback |
| Me | Them | Yes | They replied |
| Me | Me | No | Waiting for their reply |

- **FR-041**: Comments from the user's own items that they authored last → hidden
- **FR-042**: Comments on others' items where user commented last → hidden
- **FR-043**: New items from others with no comments → visible
- **FR-044**: Items where others commented last → visible

### Duration Parsing

- **FR-045**: System MUST parse duration strings: `Nd` (days), `Nh` (hours), `Nw` (weeks)
- **FR-046**: System MUST reject invalid formats with clear error message

### GitHub Reference Parsing

- **FR-047**: System MUST parse `org/repo#N` format
- **FR-048**: System MUST parse GitHub URLs: `https://github.com/org/repo/issues/N`
- **FR-049**: System MUST parse GitHub URLs: `https://github.com/org/repo/pull/N`
- **FR-050**: System MUST handle URLs with/without https://, with/without www, with trailing slash

### Output Formatting

- **FR-051**: System MUST support JSON output where applicable
- **FR-052**: Comments MUST be truncated to last 10, with "(N total, showing last 10)" header
- **FR-053**: PR files MUST be truncated to first 20, with "(N total, showing first 20)" header
- **FR-054**: Relative time formatting: "just now", "N minutes/hours/days/months/years ago"

### LLM Integration

- **FR-055**: System MUST build prompts with truncated content (body: 300 chars, comments: 200 chars each, last 5)
- **FR-056**: System MUST parse JSON response from LLM, handling markdown code blocks
- **FR-057**: System MUST fall back gracefully if LLM response is invalid JSON

## Success Criteria *(mandatory)*

- **SC-001**: All 110 existing Python tests have equivalent Go tests that pass
- **SC-002**: No Python dependencies remain - single Go binary
- **SC-003**: CLI interface is backward compatible (`bip checkin` works like `flowc checkin`)
- **SC-004**: Ball-in-my-court logic matches Python implementation exactly
- **SC-005**: Duration parsing handles all documented formats
- **SC-006**: GitHub reference parsing handles all URL variants from tests

## Assumptions

- GitHub CLI (`gh`) is available for API interactions
- tmux is available for spawn command
- claude CLI is available for LLM summarization
- User's GitHub username can be determined via `gh auth status`

## Out of Scope

- The `issue` command (deprecated, replaced by `spawn`)
- Paperpile-specific integrations
- Real-time notifications

## Technical Notes

### Go Package Structure (Suggested)

```
cmd/bip/
  flow.go           # flow subcommand dispatcher
internal/flow/
  checkin/          # checkin command
  board/            # board commands
  spawn/            # spawn command
  digest/           # digest command
  tree/             # tree command
  config/           # sources.yml, beads loading
  github/           # GitHub API interactions
  llm/              # LLM prompt building and response parsing
```

### Key Data Structures

```go
// From sources.yml
type Sources struct {
    Boards  map[string]string `json:"boards"`  // "org/N" -> bead_id
    Context map[string]string `json:"context"` // repo -> context file path
    Code    []RepoEntry       `json:"code"`
    Writing []RepoEntry       `json:"writing"`
}

type RepoEntry struct {
    Repo    string `json:"repo"`
    Channel string `json:"channel,omitempty"`
}

// From .beads/issues.jsonl
type Bead struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"`
    Priority    int    `json:"priority"`
    IssueType   string `json:"issue_type"`
    CreatedAt   string `json:"created_at"`
}
```

## Migration Strategy

1. Implement core data loading (sources.yml, beads) first
2. Port ball-in-my-court logic with comprehensive tests
3. Implement commands in priority order: checkin, board sync, spawn, digest, tree
4. Run Python and Go implementations side-by-side for validation
5. Remove Python code once Go implementation passes all tests
