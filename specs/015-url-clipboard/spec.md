# Feature Specification: URL Output and Clipboard Support

**Feature Branch**: `015-url-clipboard`
**Created**: 2026-01-27
**Status**: Draft
**Input**: User description: "Add a `bip url` command that outputs reference URLs in different formats (DOI, PubMed, PMC, arXiv, Semantic Scholar) with optional clipboard copy support."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Get DOI URL for a Reference (Priority: P1)

A user wants to quickly get the DOI link for a paper in their library to share with a colleague or paste into a document.

**Why this priority**: DOI is the most universal identifier and the most common use case. Every paper has a DOI, making this the foundational functionality.

**Independent Test**: Can be fully tested by running `bip url <ref-id>` and verifying the correct DOI URL is output.

**Acceptance Scenarios**:

1. **Given** a reference with ID "Smith2024-ab" exists in the library with DOI "10.1234/example", **When** user runs `bip url Smith2024-ab`, **Then** the output is `https://doi.org/10.1234/example`
2. **Given** a reference exists without a DOI, **When** user runs `bip url <ref-id>`, **Then** an error message indicates no DOI is available

---

### User Story 2 - Copy URL to Clipboard (Priority: P1)

A user wants to copy a reference URL directly to their clipboard without manual selection, for immediate pasting elsewhere.

**Why this priority**: Clipboard integration is core to the feature's value proposition - eliminating the copy step saves time and friction.

**Independent Test**: Can be fully tested by running `bip url <ref-id> --copy` and verifying the URL is in the system clipboard.

**Acceptance Scenarios**:

1. **Given** a reference exists with a DOI, **When** user runs `bip url Smith2024-ab --copy`, **Then** the DOI URL is copied to the clipboard, the URL is printed to stdout, and a confirmation message is printed to stderr
2. **Given** clipboard functionality is unavailable (headless server or missing tools), **When** user runs `bip url <ref-id> --copy`, **Then** a warning message indicates clipboard is unavailable and the URL is printed to stdout as fallback

---

### User Story 3 - Get Alternative URL Formats (Priority: P2)

A user wants to get the PubMed, PMC, arXiv, or Semantic Scholar URL for a paper instead of the DOI link.

**Why this priority**: Important for users who need journal-specific or database-specific links, but DOI covers most cases.

**Independent Test**: Can be fully tested by running `bip url <ref-id> --pubmed` and verifying the correct PubMed URL is output.

**Acceptance Scenarios**:

1. **Given** a reference has PMID "12345678", **When** user runs `bip url <ref-id> --pubmed`, **Then** output is `https://pubmed.ncbi.nlm.nih.gov/12345678/`
2. **Given** a reference has PMCID "PMC1234567", **When** user runs `bip url <ref-id> --pmc`, **Then** output is `https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1234567/`
3. **Given** a reference has arXiv ID "2106.15928", **When** user runs `bip url <ref-id> --arxiv`, **Then** output is `https://arxiv.org/abs/2106.15928`
4. **Given** a reference has S2 ID "649def34f8be52c8b66281af98ae884c09aef38b", **When** user runs `bip url <ref-id> --s2`, **Then** output is `https://www.semanticscholar.org/paper/649def34f8be52c8b66281af98ae884c09aef38b`
5. **Given** a reference lacks the requested ID type, **When** user requests that URL format, **Then** an error indicates that ID type is not available for this reference

---

### User Story 4 - External IDs Populated on Import (Priority: P2)

When papers are imported via S2, external identifiers (PMID, PMCID, arXiv, S2 ID) should be automatically stored for later URL generation.

**Why this priority**: Required for Story 3 to work - external IDs must be available before alternative URLs can be generated.

**Independent Test**: Can be tested by importing a paper via `bip s2 add` and verifying the external IDs are stored in refs.jsonl.

**Acceptance Scenarios**:

1. **Given** a paper is imported via S2 that has a PubMed ID, **When** import completes, **Then** the PMID is stored in the reference record
2. **Given** a paper is imported that lacks certain external IDs, **When** import completes, **Then** only available IDs are stored (missing IDs are omitted, not stored as empty)

---

### Edge Cases

- What happens when a reference ID doesn't exist? Display "reference not found" error.
- What happens when multiple URL flags are specified? Error with a clear message asking user to specify only one format flag.
- What happens on systems without clipboard support (SSH sessions, containers)? Fall back to printing URL to stdout with a warning.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `bip url` command that outputs a URL for a given reference ID
- **FR-002**: System MUST output DOI URLs by default (`https://doi.org/{doi}`)
- **FR-003**: System MUST support `--pubmed` flag to output PubMed URLs
- **FR-004**: System MUST support `--pmc` flag to output PubMed Central URLs
- **FR-005**: System MUST support `--arxiv` flag to output arXiv URLs
- **FR-006**: System MUST support `--s2` flag to output Semantic Scholar URLs
- **FR-007**: System MUST support `--copy` flag to copy URL to system clipboard
- **FR-008**: System MUST detect clipboard availability and fall back gracefully when unavailable
- **FR-009**: System MUST work on macOS (pbcopy) and Linux (xclip/xsel) platforms
- **FR-010**: System MUST extend the Reference type to store external IDs (pmid, pmcid, arxiv, s2_id)
- **FR-011**: System MUST populate external IDs when importing papers via S2 API
- **FR-012**: System MUST display clear error messages when requested ID type is unavailable
- **FR-013**: Documentation MUST be updated including the `/bip` skill in `.claude/skills/`

### Key Entities

- **Reference**: Extended with optional flat fields for external identifiers stored directly on the reference object:
  - `pmid` (string, optional): PubMed ID
  - `pmcid` (string, optional): PubMed Central ID
  - `arxiv_id` (string, optional): arXiv identifier
  - `s2_id` (string, optional): Semantic Scholar paper ID

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can retrieve any available URL format for a reference in a single command
- **SC-002**: Users can copy URLs to clipboard without manual text selection
- **SC-003**: Command works consistently across macOS and Linux environments
- **SC-004**: All external IDs available from S2 API are preserved during import
- **SC-005**: Clear feedback is provided when requested URL format is unavailable

## Assumptions

- On Linux, `xclip` or `xsel` may be installed for clipboard support; if missing, `--copy` falls back gracefully with a warning
- The `golang.design/x/clipboard` library provides adequate cross-platform support
- S2 API reliably returns external IDs when available for papers
- Existing references without external IDs will not be automatically backfilled (future enhancement)

## Out of Scope

- **Windows support**: Initial release targets macOS and Linux only. Windows may be added in a future release.

## Installation Notes

For full clipboard support on Linux, install one of:
- `sudo apt install xclip` (Debian/Ubuntu)
- `sudo apt install xsel` (Debian/Ubuntu)
- `sudo dnf install xclip` (Fedora)

macOS clipboard support (`pbcopy`) is built-in and requires no additional installation.

## Clarifications

### Session 2026-01-27

- Q: What happens when multiple URL flags are specified? → A: Error with clear message asking user to specify only one format flag
- Q: How should external IDs be stored in refs.jsonl? → A: Flat fields directly on reference object (pmid, pmcid, arxiv_id, s2_id)
- Q: Should missing clipboard tools on Linux be a hard error or graceful fallback? → A: Graceful fallback with warning, still output URL to stdout
- Q: When --copy succeeds, should URL also be printed to stdout? → A: Yes, URL to stdout and confirmation to stderr (composable for piping)
- Q: Should Windows be supported? → A: No, explicitly out of scope for initial release; added installation notes for Linux clipboard tools
- Q: Should /bip skill be updated? → A: Yes, added FR-013 for documentation updates
