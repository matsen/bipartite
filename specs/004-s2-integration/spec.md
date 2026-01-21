# Feature Specification: Semantic Scholar (S2) Integration

**Feature Branch**: `004-s2-integration`
**Created**: 2026-01-19
**Status**: Draft
**Phase**: IV (per VISION.md)
**Input**: User description: "Phase IV Semantic Scholar (S2) Integration - connect to broader academic graph, add papers from DOI/PDF, find related papers, enrich local data"

## Overview

ASTA (Semantic Scholar API) provides access to the broader academic graph. This phase connects bipartite to that graph, enabling:

- Adding papers by DOI or from PDF (metadata fetch)
- Finding papers that cite or are cited by papers in your collection
- Discovering literature gaps (highly-cited papers you don't have)
- Auto-detecting preprint→published relationships

Note: Direct DOI fetching from publishers returns 403s. Semantic Scholar provides a working API.

## Clarifications

### Session 2026-01-19

- Q: How should S2 fields not in current refs.jsonl schema be handled? → A: Map overlapping fields + store S2-specific fields in a nested `source.metadata` object
- Q: How should duplicates be detected for papers without DOIs? → A: Use S2 paper ID as secondary dedup key
- Q: Should ASTA use existing MCP server or direct HTTP client? → A: Direct HTTP client using standard library net/http (simpler, no MCP dependency)
- Q: Which approach for PDF DOI extraction in pure Go? → A: Use pdfcpu library for text extraction, regex for DOI pattern matching
- Q: Where should API response cache be stored? → A: In-memory LRU cache with optional persistence to .bipartite/cache/s2-cache.json

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add Paper by DOI (Priority: P1)

A researcher finds a paper they want to add to their collection. They have the DOI (from a citation, webpage, or the paper itself). They want to add it to bipartite with full metadata without manually entering anything.

**Why this priority**: This is the most common use case - quickly adding a paper when you have its identifier. It's the foundation for other features (PDF add uses this after DOI extraction).

**Independent Test**: Run `bip s2 add DOI:10.1234/example`, verify paper is added to refs.jsonl with metadata from Semantic Scholar.

**Acceptance Scenarios**:

1. **Given** a valid DOI, **When** user runs `bip s2 add DOI:10.1234/example`, **Then** the paper is added to refs.jsonl with title, authors, abstract, year, and venue from Semantic Scholar
2. **Given** a DOI for a paper already in the collection (matched by DOI), **When** user runs `bip s2 add`, **Then** the system reports the paper already exists and optionally updates metadata with `--update` flag
3. **Given** a DOI not found in Semantic Scholar, **When** user runs `bip s2 add`, **Then** the system reports a clear error that the DOI was not found
4. **Given** a paper added via S2, **Then** the `source.type` field is set to `"s2"` and `source.id` contains the Semantic Scholar paper ID

---

### User Story 2 - Add Paper from PDF (Priority: P2)

A researcher has a PDF and wants to add it to their collection. The system extracts the DOI from the PDF and fetches metadata from Semantic Scholar.

**Why this priority**: PDFs are how researchers often encounter papers. Automating the DOI extraction → metadata fetch flow is high value.

**Independent Test**: Run `bip s2 add-pdf paper.pdf`, verify DOI is extracted and paper is added with S2 metadata.

**Acceptance Scenarios**:

1. **Given** a PDF with an embedded DOI, **When** user runs `bip s2 add-pdf paper.pdf`, **Then** the DOI is extracted and metadata is fetched from Semantic Scholar
2. **Given** a PDF without an embedded DOI, **When** user runs `bip s2 add-pdf`, **Then** the system attempts title-based search and prompts for confirmation if a match is found
3. **Given** a PDF where no match can be found, **When** user runs `bip s2 add-pdf`, **Then** the system reports failure with suggestions (manual DOI entry, check the PDF)
4. **Given** a successful add, **When** `--link` flag is provided, **Then** the `pdf_path` field is populated with the path to the PDF

---

### User Story 3 - Lookup Paper Info (Priority: P2)

An agent or researcher wants to query Semantic Scholar for paper information without adding it to the collection. This is useful for exploration and decision-making.

**Why this priority**: Agents need to inspect papers before deciding whether to add them. This supports the agent-first design.

**Independent Test**: Run `bip s2 lookup DOI:10.1234/example`, verify JSON output with paper metadata.

**Acceptance Scenarios**:

1. **Given** a valid paper identifier, **When** user runs `bip s2 lookup <id>`, **Then** paper metadata is returned as JSON (title, authors, abstract, year, venue, citation count, references count)
2. **Given** a paper ID, **When** user runs `bip s2 lookup <id> --fields abstract,citations`, **Then** only requested fields are returned
3. **Given** the `--exists` flag, **When** user runs `bip s2 lookup <id> --exists`, **Then** output includes whether the paper exists in the local collection

---

### User Story 4 - Find Papers Citing a Paper (Priority: P3)

A researcher wants to see what papers cite a paper in their collection. This helps with forward citation tracking - "who built on this work?"

**Why this priority**: Citation tracking is core to literature exploration. Finding citing papers helps researchers stay current.

**Independent Test**: Run `bip s2 citations <paper-id>`, verify list of citing papers is returned.

**Acceptance Scenarios**:

1. **Given** a paper in the collection, **When** user runs `bip s2 citations <paper-id>`, **Then** papers citing it are returned from Semantic Scholar
2. **Given** citations results, **When** `--local-only` flag is used, **Then** only citations that are also in the local collection are shown
3. **Given** citations results, **Then** each result indicates whether it exists in the local collection
4. **Given** a limit flag, **When** user runs `bip s2 citations <id> --limit 20`, **Then** at most 20 citations are returned

---

### User Story 5 - Find Papers Referenced by a Paper (Priority: P3)

A researcher wants to see what papers a given paper cites. This helps with backward exploration - "what did this work build on?"

**Why this priority**: Understanding a paper's foundation is essential for literature review. References show intellectual lineage.

**Independent Test**: Run `bip s2 references <paper-id>`, verify list of referenced papers is returned.

**Acceptance Scenarios**:

1. **Given** a paper in the collection, **When** user runs `bip s2 references <paper-id>`, **Then** papers it references are returned from Semantic Scholar
2. **Given** references results, **Then** each result indicates whether it exists in the local collection
3. **Given** the `--missing` flag, **When** user runs `bip s2 references <id> --missing`, **Then** only references NOT in the local collection are shown

---

### User Story 6 - Discover Literature Gaps (Priority: P4)

A researcher wants to find important papers they might be missing. The system analyzes citations across their collection to find frequently-cited papers they don't have.

**Why this priority**: Proactive gap discovery is powerful but depends on other features working first.

**Independent Test**: Run `bip s2 gaps`, verify list of highly-cited-but-missing papers is returned.

**Acceptance Scenarios**:

1. **Given** a collection with papers, **When** user runs `bip s2 gaps`, **Then** papers cited by multiple papers in the collection (but not in the collection) are listed
2. **Given** gap results, **Then** results are ranked by citation count within the collection (how many of your papers cite it)
3. **Given** a threshold flag, **When** user runs `bip s2 gaps --min-citations 3`, **Then** only papers cited by at least 3 papers in the collection are shown
4. **Given** gap results, **Then** each result shows which papers in your collection cite it

---

### User Story 7 - Link Preprint to Published Version (Priority: P4)

A researcher has a preprint in their collection and the published version now exists. The system detects this and offers to link them via the `supersedes` field.

**Why this priority**: Keeps the collection clean and shows paper evolution. Less urgent than core add/query.

**Independent Test**: Run `bip s2 link-published`, verify preprints with published versions are detected and `supersedes` is populated.

**Acceptance Scenarios**:

1. **Given** a preprint in the collection, **When** user runs `bip s2 link-published`, **Then** Semantic Scholar is queried for the published version
2. **Given** a published version is found, **When** detected, **Then** the system reports the match and offers to set `supersedes` on the preprint
3. **Given** `--auto` flag, **When** matches are found, **Then** `supersedes` is automatically set without confirmation
4. **Given** a paper already has `supersedes` set, **Then** it is skipped

---

### Edge Cases

- What happens when Semantic Scholar is unreachable? Clear error message with retry suggestion.
- What happens when rate limited? Respectful backoff with progress indication.
- What happens when a paper has multiple DOIs (preprint + published)? Accept any, link to canonical S2 ID.
- What happens when PDF DOI extraction fails but title search finds multiple matches? Present options, require user selection.
- What happens when adding a paper that already exists by DOI? Report duplicate, offer `--update` to refresh metadata.
- What happens when the S2 API returns incomplete data (missing abstract)? Add paper with available data, warn about missing fields.

## Requirements *(mandatory)*

### Functional Requirements

**Paper Addition**

- **FR-001**: System MUST add papers via `bip s2 add <paper-id>` where paper-id supports DOI, arXiv ID, PMID, and S2 ID formats
- **FR-002**: System MUST extract DOI from PDF via `bip s2 add-pdf <path>`
- **FR-003**: System MUST fetch metadata from Semantic Scholar including: title, authors, abstract, year, venue, DOI
- **FR-004**: System MUST detect duplicate papers by DOI before adding; for papers without DOI, use S2 paper ID as secondary dedup key
- **FR-005**: System MUST support `--update` flag to refresh metadata for existing papers
- **FR-006**: System MUST set `source.type` to `"s2"` and `source.id` to S2 paper ID for added papers
- **FR-007**: System MUST support `--link <pdf-path>` to associate a PDF with the added paper

**Paper Lookup**

- **FR-008**: System MUST lookup paper info via `bip s2 lookup <paper-id>` without adding
- **FR-009**: System MUST support `--fields` flag to select which fields to return
- **FR-010**: System MUST support `--exists` flag to check if paper is in local collection

**Citation Exploration**

- **FR-011**: System MUST find citing papers via `bip s2 citations <paper-id>`
- **FR-012**: System MUST find referenced papers via `bip s2 references <paper-id>`
- **FR-013**: System MUST indicate whether each result exists in the local collection
- **FR-014**: System MUST support `--local-only` flag to filter to papers in collection
- **FR-015**: System MUST support `--missing` flag to filter to papers NOT in collection
- **FR-016**: System MUST support `--limit N` flag for pagination

**Gap Discovery**

- **FR-017**: System MUST find literature gaps via `bip s2 gaps`
- **FR-018**: System MUST rank gaps by citation count within the collection
- **FR-019**: System MUST support `--min-citations N` threshold filter
- **FR-020**: System MUST show which local papers cite each gap

**Preprint Linking**

- **FR-021**: System MUST detect preprint→published relationships via `bip s2 link-published`
- **FR-022**: System MUST populate `supersedes` field when linking
- **FR-023**: System MUST support `--auto` flag for automatic linking without confirmation

**Output & Integration**

- **FR-024**: System MUST output JSON by default (agent-first design)
- **FR-025**: System MUST support `--human` flag for human-readable output
- **FR-026**: System MUST integrate with `bip rebuild` (no additional rebuild needed for S2 data)

### Non-Functional Requirements

- **NFR-001**: API calls MUST respect Semantic Scholar rate limits (100 requests/5 minutes for unauthenticated)
- **NFR-002**: System MUST cache API responses using in-memory LRU cache with optional persistence to `.bipartite/cache/s2-cache.json`
- **NFR-003**: System MUST provide clear progress indication for batch operations
- **NFR-004**: System MUST handle network failures gracefully with actionable error messages

### Key Entities

- **S2Paper**: Semantic Scholar paper representation. Fields: paperId (S2 ID), externalIds (DOI, arXiv, PMID), title, authors, abstract, year, venue, citationCount, referenceCount, fieldsOfStudy, isOpenAccess, publicationDate.

- **PaperIdentifier**: A reference to a paper in various formats. Supported: `DOI:10.xxx`, `ARXIV:xxxx.xxxxx`, `PMID:xxxxxxxx`, `S2:xxxxxxxxx`, or raw S2 paper ID.

- **S2 Field Mapping**: When adding papers via S2, overlapping fields (title, authors, year, abstract, venue, DOI) map directly to refs.jsonl schema. S2-specific fields (citationCount, referenceCount, fieldsOfStudy, isOpenAccess, publicationDate) are stored in `source.metadata` object alongside `source.type: "s2"` and `source.id: <S2 paperId>`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Adding a paper by DOI completes in under 3 seconds
- **SC-002**: PDF DOI extraction succeeds for 80%+ of academic PDFs with embedded DOIs
- **SC-003**: Citation/reference queries return results in under 5 seconds
- **SC-004**: Gap discovery for a 1000-paper collection completes in under 60 seconds
- **SC-005**: All commands produce valid, parseable JSON when requested
- **SC-006**: System respects rate limits and never triggers S2 API blocks

## Assumptions

- Semantic Scholar API remains available and free for reasonable use (current: 100 req/5 min unauthenticated)
- Most academic papers have DOIs that S2 can resolve
- PDF DOI extraction is best-effort; some PDFs won't have extractable DOIs
- The `supersedes` field already exists in the data model (from Phase I)
- Papers added via S2 follow the same JSONL format as papers from other sources

## Technical Decisions

- **HTTP Client**: Direct HTTP using Go standard library `net/http` (no MCP server dependency)
- **PDF Parsing**: Use `pdfcpu` library for text extraction, regex pattern matching for DOI detection
- **Caching**: In-memory LRU cache for API responses; optional JSON persistence for cross-session caching

## Dependencies

- **Phase I (001-core-reference-manager)**: refs.jsonl schema, basic CLI infrastructure
- **Phase III-a (003-knowledge-graph)**: Edge storage could be enhanced with ASTA citation data (optional integration)

## Out of Scope

- Full-text PDF analysis beyond DOI extraction
- Automatic PDF download (users manage their own PDFs)
- Author disambiguation beyond what S2 provides
- Citation graph visualization (external tools can visualize)
- Bulk import of entire S2 search results (add papers one at a time or via explicit list)

## Future Considerations

- **Edge Generation**: ASTA citation data could auto-generate edges for the knowledge graph
- **Author Tracking**: Track specific authors and alert when they publish new papers
- **Venue Filtering**: Filter citations/references by venue (e.g., only top conferences)
- **API Key Support**: Support S2 API keys for higher rate limits
