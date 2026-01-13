# Feature Specification: Knowledge Graph (Phase III-a: Paper Edges)

**Feature Branch**: `003-knowledge-graph`
**Created**: 2026-01-13
**Status**: Draft
**Phase**: III-a (Paper Edges) — see VISION.md for Phase III-b (Concept Nodes)
**Input**: User description: "Phase III Knowledge Graph - directed edges with relational summaries, edge storage and query in bp, external edge generation via tex-to-edges Claude skill"

## Clarifications

### Session 2026-01-13

- Q: What happens to edges when their referenced paper is deleted? → A: Preserve edges (mark endpoint as missing); `bp groom` can flag orphans

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add Edges from External Tool (Priority: P1)

An external tool (such as a Claude Code skill analyzing a manuscript) generates edges describing relationships between papers. The researcher imports these edges into bipartite for persistent storage and later querying.

**Why this priority**: This is the primary data ingestion path. Without the ability to add edges, no other functionality is useful. The tex-to-edges skill is the first planned edge generator.

**Independent Test**: Can be fully tested by running `bp edge add` with edge data and verifying the edge is stored and retrievable.

**Acceptance Scenarios**:

1. **Given** a bipartite repo with papers imported, **When** an external tool calls `bp edge add` with source paper ID, target paper ID, relationship type, and summary, **Then** the edge is stored in the knowledge graph
2. **Given** edge data in JSONL format, **When** the user runs `bp edge import edges.jsonl`, **Then** all edges are added to the knowledge graph
3. **Given** an edge referencing a paper ID that doesn't exist, **When** attempting to add the edge, **Then** the system reports an error identifying the missing paper

---

### User Story 2 - Query Edges for a Paper (Priority: P2)

A researcher or agent wants to understand how a specific paper relates to other papers in their collection. They query the knowledge graph to find all edges involving that paper.

**Why this priority**: After ingestion, querying is the core value proposition. Understanding relationships is why the knowledge graph exists.

**Independent Test**: Can be tested by adding edges, then querying for edges connected to a specific paper and verifying correct results.

**Acceptance Scenarios**:

1. **Given** a paper with edges to other papers, **When** the user runs `bp edge list <paper-id>`, **Then** all edges where the paper is the source are displayed with relationship types and summaries
2. **Given** a paper with edges from other papers pointing to it, **When** the user runs `bp edge list <paper-id> --incoming`, **Then** all edges where the paper is the target are displayed
3. **Given** a paper with both outgoing and incoming edges, **When** the user runs `bp edge list <paper-id> --all`, **Then** both directions are displayed with clear indication of direction

---

### User Story 3 - Search Edges by Relationship Type (Priority: P3)

A researcher wants to find all papers that extend or contradict a particular line of research. They filter edges by relationship type to explore the academic discourse.

**Why this priority**: Enables more sophisticated exploration of the knowledge graph beyond simple adjacency queries.

**Independent Test**: Can be tested by adding edges with different relationship types, then filtering by type and verifying correct filtering.

**Acceptance Scenarios**:

1. **Given** edges with various relationship types (cites, extends, contradicts), **When** the user runs `bp edge search --type extends`, **Then** only edges with relationship type "extends" are returned
2. **Given** edges with various relationship types, **When** the user runs `bp edge search --type contradicts --json`, **Then** results are returned in JSON format suitable for agent consumption

---

### User Story 4 - Export Edges (Priority: P4)

A researcher wants to share their knowledge graph annotations or back them up. They export edges to JSONL format.

**Why this priority**: Enables portability and backup, but less critical than core add/query functionality.

**Independent Test**: Can be tested by adding edges, exporting to JSONL, and verifying the export contains all edges with correct data.

**Acceptance Scenarios**:

1. **Given** a knowledge graph with edges, **When** the user runs `bp edge export`, **Then** all edges are written to stdout in JSONL format
2. **Given** edges for specific papers, **When** the user runs `bp edge export --paper <id>`, **Then** only edges involving that paper are exported

---

### Edge Cases

- What happens when adding a duplicate edge (same source, target, and type)? System should update the summary rather than create a duplicate.
- What happens when deleting a paper that has edges? Edges are preserved with the endpoint marked as missing; `bp groom` flags orphaned edges for review.
- How does the system handle edges with very long summaries? Summaries should be stored in full with no truncation.
- What happens when querying a paper with no edges? Return empty result set with appropriate message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST store directed edges with: source node ID, target node ID, relationship type, and relational summary
- **FR-002**: System MUST support relationship types: "cites", "extends", "contradicts", "implements", "applies-to", "builds-on"
- **FR-003**: System MUST allow custom relationship types beyond the predefined set
- **FR-004**: System MUST provide `bp edge add` command to add individual edges
- **FR-005**: System MUST provide `bp edge import` command to bulk import edges from JSONL
- **FR-006**: System MUST provide `bp edge list <paper-id>` command to list edges for a paper
- **FR-007**: System MUST provide `bp edge search` command to filter edges by relationship type
- **FR-008**: System MUST provide `bp edge export` command to export edges to JSONL
- **FR-009**: System MUST validate that source and target node IDs exist before adding an edge (fail-fast)
- **FR-010**: System MUST store edges in JSONL format (following bipartite's git-mergeable philosophy)
- **FR-011**: System MUST rebuild edge index from JSONL on `bp rebuild`
- **FR-012**: System MUST output JSON format when `--json` flag is provided (agent-first design)
- **FR-013**: System MUST support edges between papers (paper → paper relationships)
- **FR-014**: System MUST handle edge updates (same source, target, type) by replacing the existing summary
- **FR-015**: System MUST preserve edges when referenced papers are deleted, marking the endpoint as missing
- **FR-016**: System MUST detect orphaned edges (with missing endpoints) during `bp groom` operations

### Key Entities

- **Edge**: A directed relationship between two nodes. Contains: source ID, target ID, relationship type (string), relational summary (prose text describing the relationship from source's perspective toward target), and optional metadata.
- **Node**: Currently limited to papers (existing in refs.jsonl). Future phases may add concepts and artifacts as node types.
- **Relationship Type**: A categorization of the edge's semantic meaning. Predefined types exist but custom types are allowed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Researchers can add edges from external tools in under 1 second per edge
- **SC-002**: Edge queries return results in under 500ms for collections with up to 10,000 edges
- **SC-003**: The knowledge graph data remains git-mergeable (JSONL format, no binary blobs)
- **SC-004**: Agents can programmatically add and query edges using JSON input/output
- **SC-005**: Edge data survives `bp rebuild` without loss (ephemeral index, persistent JSONL)
- **SC-006**: A tex-to-edges workflow can generate and import 100 edges in under 30 seconds

## Assumptions

- Node types are limited to papers in this phase. Concepts and artifacts (mentioned in VISION.md) are deferred to a future iteration.
- The tex-to-edges Claude skill is developed separately and uses `bp edge add` or `bp edge import` to store results.
- Edge direction follows the convention: source "relates-to" target (e.g., Paper A "extends" Paper B means A extends B).
- Summaries are free-form prose with no length limit.
- The predefined relationship types are suggestions; the system is permissive and allows any string as a type.

## Dependencies

- Phase I (001-core-reference-manager): Papers must exist in refs.jsonl before edges can reference them
- Phase II (002-rag-index): Not a hard dependency, but RAG search could enhance edge discovery in future iterations

## Test Data

Test fixtures are located in `testdata/edges/`:

- **refs-subset.jsonl**: 15 papers from the DASM manuscript ecosystem (subset of refs.jsonl for self-contained testing)
- **test-edges.jsonl**: 20 realistic edges representing relationships between antibody language model papers

### Fixture Contents

The test edges cover all relationship types:
- `builds-on` (5): e.g., DASM builds on thrifty mutation model
- `cites` (6): e.g., DASM cites SHM deep learning work
- `contradicts` (3): e.g., DASM shows AntiBERTy conflates mutation with selection
- `extends` (4): e.g., DASM extends BERT-style masked modeling
- `applies-to` (2): e.g., DASM evaluated on FLAb benchmark

### Usage

- **Unit tests**: Use `testdata/edges/refs-subset.jsonl` + `testdata/edges/test-edges.jsonl` for self-contained tests
- **Integration tests**: Can use full `.bipartite/refs.jsonl` with `testdata/edges/test-edges.jsonl`

## Out of Scope

- Concept nodes and artifact nodes (future phase)
- Automatic edge generation within bp (external tools handle this)
- Graph visualization (CLI-focused, agents/external tools can visualize)
- Edge validation against external sources (ASTA integration is Phase IV)
- Discovery tracking (Phase V in VISION.md)
