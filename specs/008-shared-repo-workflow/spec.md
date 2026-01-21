# Feature Specification: Shared Repository Workflow Commands

**Feature Branch**: `008-shared-repo-workflow`
**Created**: 2026-01-21
**Status**: Draft
**Input**: User description: "Commands needed to support teams sharing a paper library via git. With matsengrp/bip-papers set up as a shared repository, several command enhancements would streamline the collaborative workflow."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Open Multiple Papers for Review (Priority: P1)

Alice's agent adds papers to the shared repository. Alice wants to quickly open and verify that the papers are relevant to her research before committing time to read them in detail.

**Why this priority**: This is the most common collaborative workflow - reviewing papers added by others (or agents) requires quick multi-paper access. Without this, users must manually look up and open each paper one by one.

**Independent Test**: Can be fully tested by running `bip open` with multiple paper IDs and verifying all PDFs open in the default viewer. Delivers immediate value for paper review workflows.

**Acceptance Scenarios**:

1. **Given** a repository with papers P1, P2, P3, **When** user runs `bip open P1 P2 P3`, **Then** all three papers open in the default PDF viewer
2. **Given** a repository with 10 papers, **When** user runs `bip open --recent 3`, **Then** the 3 most recently added papers open
3. **Given** a repository with papers added across commits, **When** user runs `bip open --since abc123`, **Then** all papers added after commit abc123 open

---

### User Story 2 - Track New Papers from Collaborators (Priority: P1)

Bernadetta pulls updates from the shared repository and wants to see what papers were added by her collaborators since she last checked, so she can stay current with the team's literature collection.

**Why this priority**: Equally critical to multi-paper open - this is the entry point for the collaborative workflow. Users need to discover what changed before they can act on it.

**Independent Test**: Can be fully tested by running `bip new` or `bip diff` commands and verifying the output lists the correct papers. Delivers immediate value for staying synchronized with collaborators.

**Acceptance Scenarios**:

1. **Given** uncommitted changes in refs.jsonl, **When** user runs `bip diff`, **Then** output shows papers added and removed since last commit
2. **Given** a repository with history, **When** user runs `bip new --since abc123`, **Then** output lists papers added after that commit with metadata
3. **Given** papers added over the past week, **When** user runs `bip new --days 7`, **Then** output lists papers added within the last 7 days

---

### User Story 3 - Export Specific Papers to BibTeX (Priority: P2)

An agent determines that a specific paper should be cited in a manuscript. The agent needs to export just that paper's BibTeX entry and optionally append it to an existing .bib file without duplicating entries.

**Why this priority**: Important for manuscript writing workflow but depends on users first being able to discover and review papers (P1 features). Single-paper export is more targeted than batch export.

**Independent Test**: Can be fully tested by running `bip export --bibtex <id>` and verifying correct BibTeX output. Append mode can be tested by checking the .bib file contains the entry without duplicates.

**Acceptance Scenarios**:

1. **Given** a paper with ID "abc123" in the library, **When** user runs `bip export --bibtex abc123`, **Then** BibTeX entry for that paper is output
2. **Given** multiple papers, **When** user runs `bip export --bibtex id1 id2 id3`, **Then** BibTeX entries for all specified papers are output
3. **Given** an existing .bib file with entries, **When** user runs `bip export --bibtex --append refs.bib newpaper`, **Then** the new entry is appended and duplicates are not created

---

### Edge Cases

- What happens when `bip open` is called with an ID that has no associated PDF file? → Covered by FR-004: show error, continue opening available PDFs
- How does `bip diff` handle merge commits or rebased history?
- What happens when `bip new --since` references a non-existent commit? → Exit with error message indicating commit not found
- How does `bip export --bibtex --append` handle a paper that already exists in the .bib file (deduplication)? → Match by DOI (primary), fall back to citation key
- What happens when `bip open --recent N` is called but fewer than N papers exist? → Open all available papers (no error)
- How does `bip new --days` handle timezones and papers added at midnight boundaries? → Use UTC for all date calculations

## Requirements *(mandatory)*

### Functional Requirements

#### Open Multiple Papers (bip open)

- **FR-001**: System MUST support opening multiple papers by ID in a single command (`bip open <id1> <id2> ...`)
- **FR-002**: System MUST support `--recent N` flag to open the N most recently added papers (recency determined by git commit timestamp)
- **FR-003**: System MUST support `--since <commit>` flag to open papers added after a specific git commit
- **FR-004**: System MUST gracefully handle missing PDF files with a clear error message while still opening available PDFs
- **FR-005**: System MUST open PDFs using the system default PDF viewer

#### Track What's New (bip diff, bip new)

- **FR-006**: System MUST provide `bip diff` command showing papers added/removed since last commit
- **FR-007**: System MUST provide `bip new --since <commit>` to list papers added after a git commit
- **FR-008**: System MUST provide `bip new --days N` to list papers added within the last N days
- **FR-009**: System MUST output JSON by default with `--human` flag for readable output (consistent with existing commands)
- **FR-010**: System MUST display paper metadata (title, authors, year, ID) in the output

#### BibTeX Export (bip export --bibtex)

- **FR-011**: System MUST support `bip export --bibtex <id>` to export a single paper's BibTeX entry
- **FR-012**: System MUST support multiple IDs (`bip export --bibtex <id1> <id2> ...`)
- **FR-013**: System MUST support `--append <file>` flag to append entries to an existing .bib file
- **FR-014**: System MUST deduplicate entries when appending (match by DOI if present, fall back to citation key match for entries without DOI)
- **FR-015**: System MUST generate valid BibTeX with appropriate entry types (@article, @inproceedings, etc.)

#### Cross-Cutting Requirements

- **FR-016**: All commands MUST follow existing CLI patterns (JSON default, `--human` for readable output)
- **FR-017**: Multi-ID arguments MUST use consistent parsing across all commands
- **FR-018**: `--since <commit>` MUST accept any valid git commit reference (SHA, branch name, tag, HEAD~N)
- **FR-019**: Error messages MUST be actionable and indicate how to resolve the issue

### Key Entities

- **Paper Reference**: Existing entity from refs.jsonl - contains ID, title, authors, year, DOI, PDF path
- **Git Commit**: External entity - represents a point in repository history for `--since` filtering
- **BibTeX Entry**: Output format - contains citation key, entry type, and bibliographic fields

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can open 5 papers simultaneously in under 3 seconds
- **SC-002**: Users can identify new papers from collaborators within 10 seconds of pulling updates
- **SC-003**: Exporting a single paper's BibTeX completes in under 1 second
- **SC-004**: 100% of exported BibTeX entries are valid and parseable by standard tools
- **SC-005**: Append mode correctly deduplicates 100% of existing entries
- **SC-006**: All commands produce consistent output format (JSON default, human-readable with flag)

## Assumptions

- The repository uses git for version control and refs.jsonl is tracked in git
- Papers have associated PDF files stored locally (path in refs.jsonl)
- The system's default PDF viewer can handle multiple files opened in sequence
- BibTeX citation keys can be derived from paper metadata (author-year or DOI-based)
- Git is available on the system PATH for commit-based filtering

## Out of Scope

- Remote PDF fetching (papers must already have local PDFs)
- BibTeX style customization (uses standard format)
- Conflict resolution for shared repository merges (see separate issue #18)
- Real-time synchronization or push notifications for new papers

## Clarifications

### Session 2026-01-21

- Q: How is "recently added" determined for `--recent N` flag? → A: Based on git commit timestamp (when the paper's entry was committed)
- Q: How are BibTeX entries matched for deduplication when appending? → A: Match by DOI (primary), fall back to citation key match for entries without DOI
- Q: What happens when `--since` references a non-existent commit? → A: Exit with error message indicating commit not found
- Q: What happens when `--recent N` requests more papers than exist? → A: Open all available papers (no error)
- Q: How does `--days N` handle timezones? → A: Use UTC for all date calculations
