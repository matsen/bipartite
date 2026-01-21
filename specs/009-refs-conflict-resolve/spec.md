# Feature Specification: Domain-Aware Conflict Resolution

**Feature Branch**: `009-refs-conflict-resolve`
**Created**: 2026-01-21
**Status**: Draft
**Input**: User description: "Implement bip resolve command for domain-aware conflict resolution in refs.jsonl. Git sees JSON blobs; bip knows: doi is a unique identifier we can match on, one version might have an abstract the other lacks, a paper with more metadata is probably better, author lists can be merged if one is more complete."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Auto-Resolve Simple Metadata Conflicts (Priority: P1)

Alice pulls from the shared repository and gets a merge conflict in refs.jsonl. On one side, her agent added a paper with just DOI and title. On the other side, a collaborator added the same paper (same DOI) with a full abstract and complete author list. Alice wants bip to automatically resolve this by keeping the more complete version.

**Why this priority**: This is the most common conflict scenario - same paper added independently with different completeness levels. Manual JSON wrangling is error-prone and tedious.

**Independent Test**: Can be fully tested by creating a refs.jsonl with git conflict markers, running `bip resolve`, and verifying the output contains the merged paper with the most complete metadata.

**Acceptance Scenarios**:

1. **Given** refs.jsonl with conflict markers containing the same paper (matching DOI) on both sides where one has more fields, **When** user runs `bip resolve`, **Then** the file is resolved with the more complete version
2. **Given** refs.jsonl with conflict markers where the same paper has complementary metadata (ours has abstract, theirs has venue), **When** user runs `bip resolve`, **Then** the file is resolved with fields merged from both versions
3. **Given** refs.jsonl with conflict markers containing different papers (different DOIs) on each side, **When** user runs `bip resolve`, **Then** both papers are included in the resolved output

---

### User Story 2 - Preview Conflict Resolution (Priority: P1)

Before making changes, Ben wants to see what `bip resolve` would do. He runs a dry-run to understand the conflicts and verify the resolution strategy before applying it.

**Why this priority**: Equally critical - users need confidence in what will change before modifying their library. This prevents unexpected data loss.

**Independent Test**: Can be fully tested by running `bip resolve --dry-run` and verifying output shows conflicts detected and proposed resolutions without modifying any files.

**Acceptance Scenarios**:

1. **Given** refs.jsonl with conflict markers, **When** user runs `bip resolve --dry-run`, **Then** output shows detected conflicts and proposed resolution without modifying the file
2. **Given** refs.jsonl with no conflicts, **When** user runs `bip resolve --dry-run`, **Then** output indicates no conflicts detected
3. **Given** refs.jsonl with unresolvable conflicts, **When** user runs `bip resolve --dry-run`, **Then** output shows which conflicts require interactive resolution

---

### User Story 3 - Interactive Resolution for True Conflicts (Priority: P2)

Carla has a merge conflict where the same paper has different values for the same field on each side - the abstract was edited differently by two collaborators. Auto-resolution cannot safely choose. Carla uses interactive mode to review and select which version to keep for each conflicting field.

**Why this priority**: Less common than auto-resolvable conflicts, but essential for completeness. Without this, some conflicts would require manual file editing.

**Independent Test**: Can be fully tested by creating a conflict with true field-level conflicts, running `bip resolve --interactive`, and verifying prompts appear for each unresolvable field.

**Acceptance Scenarios**:

1. **Given** a paper conflict where both sides have different non-empty abstracts, **When** user runs `bip resolve --interactive`, **Then** user is prompted to choose which abstract to keep
2. **Given** multiple papers with field conflicts, **When** user runs `bip resolve --interactive`, **Then** user can resolve each conflict one by one
3. **Given** a mix of auto-resolvable and true conflicts, **When** user runs `bip resolve --interactive`, **Then** auto-resolvable conflicts are handled automatically and prompts appear only for true conflicts

---

### Edge Cases

- What happens when refs.jsonl has no conflict markers? -> Exit with message indicating no conflicts detected
- What happens when a paper has no DOI on either side? -> Match by ID field as fallback; if no match possible, include both
- What happens when conflict markers are malformed or nested? -> Exit with error describing the parsing issue
- How are author lists merged when both sides have authors but lists differ? -> Prefer the longer/more complete author list; if same length with different content, treat as true conflict
- What happens when `bip resolve` runs without `--interactive` and encounters unresolvable conflicts? -> Exit with error listing the unresolvable conflicts and suggesting `--interactive` flag
- What happens if refs.jsonl doesn't exist or is empty? -> Exit with message indicating no refs.jsonl found

## Requirements *(mandatory)*

### Functional Requirements

#### Conflict Detection

- **FR-001**: System MUST detect git conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`) in refs.jsonl
- **FR-002**: System MUST parse both "ours" and "theirs" versions of conflicted content as valid JSONL
- **FR-003**: System MUST identify same-paper conflicts by matching on DOI (primary) or ID (fallback)
- **FR-004**: System MUST distinguish between auto-resolvable conflicts (one side has more data) and true conflicts (both sides have different non-empty values)

#### Auto-Resolution Strategy

- **FR-005**: System MUST prefer the paper version with more non-empty fields when resolving same-paper conflicts
- **FR-006**: System MUST merge complementary metadata when one version has a field the other lacks
- **FR-007**: System MUST include papers that appear only on one side of the conflict (new additions)
- **FR-008**: System MUST preserve all unique papers (different DOIs) from both sides

#### Command Interface

- **FR-009**: System MUST provide `bip resolve` command that auto-resolves conflicts and writes the result
- **FR-010**: System MUST provide `--dry-run` flag that shows proposed resolution without modifying files
- **FR-011**: System MUST provide `--interactive` flag for prompting on unresolvable conflicts
- **FR-011a**: Interactive prompts MUST use numbered options (e.g., "[1] ours [2] theirs") with user typing the number to select
- **FR-012**: System MUST exit with error when encountering unresolvable conflicts without `--interactive` flag

#### Output Format

- **FR-013**: System MUST output JSON by default with `--human` flag for readable output (consistent with existing commands)
- **FR-014**: Dry-run output MUST list each conflict with its proposed resolution and reasoning
- **FR-015**: Resolution summary MUST include counts of: papers merged, papers added from ours, papers added from theirs, fields requiring interactive input

### Key Entities

- **Conflict Region**: A section of refs.jsonl bounded by git conflict markers, containing "ours" and "theirs" versions
- **Paper Match**: Two paper records (one from each side) identified as representing the same paper via DOI or ID
- **Field Conflict**: A specific field where both sides have different non-empty values
- **Resolution Decision**: The chosen value for a field or paper, with source (ours/theirs/merged)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can resolve typical metadata conflicts (completeness differences) without manual editing
- **SC-002**: Dry-run accurately predicts 100% of changes that would be made by actual resolution
- **SC-003**: No data loss occurs during auto-resolution (all unique papers preserved)
- **SC-004**: Users can resolve a 10-paper conflict in under 30 seconds using auto-resolution
- **SC-005**: Interactive mode requires user input only for true conflicts (both sides non-empty and different)

## Assumptions

- The repository uses git for version control and refs.jsonl experiences merge conflicts
- Conflict markers follow standard git format (`<<<<<<<`, `=======`, `>>>>>>>`)
- DOI is the authoritative identifier for matching papers across conflict sides
- Papers without DOI can be matched by their ID field
- More non-empty fields indicates a more complete/better paper record
- Users have terminal access for interactive prompts

## Out of Scope

- Resolving conflicts in edges.jsonl or concepts.jsonl (future enhancement)
- Semantic conflict resolution (e.g., "this abstract is better written")
- Three-way merge with common ancestor
- Automatic git staging of resolved file
- Network-based metadata enrichment during resolution

## Clarifications

### Session 2026-01-21

- Q: What interaction method should interactive mode use for prompts? → A: Numbered prompts (e.g., "[1] ours [2] theirs" - user types 1 or 2)
