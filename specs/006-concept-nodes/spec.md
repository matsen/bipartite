# Feature Specification: Concept Nodes

**Feature Branch**: `006-concept-nodes`
**Created**: 2026-01-21
**Status**: Draft
**Input**: Extend knowledge graph with concept nodes — named ideas, methods, or phenomena that papers relate to. Enable CLI commands for concept CRUD and paper-concept edge linking.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Manage Concepts (Priority: P1)

A researcher wants to define named concepts (methods, phenomena, ideas) that appear across their paper collection. They create concepts with descriptive names, optional aliases, and descriptions to build a vocabulary of key ideas in their field.

**Why this priority**: Concepts are the foundation — without them, no paper-concept relationships can exist. This is the minimal viable unit that unlocks all other functionality.

**Independent Test**: Can be fully tested by adding a concept, listing it, and verifying persistence. Delivers immediate value by establishing the concept vocabulary.

**Acceptance Scenarios**:

1. **Given** an empty concepts store, **When** user adds a concept with ID, name, aliases, and description, **Then** the concept is persisted and can be retrieved by ID
2. **Given** an existing concept, **When** user updates the concept's name or description, **Then** the changes are persisted
3. **Given** an existing concept with no linked papers, **When** user deletes the concept, **Then** the concept is removed from storage
4. **Given** an existing concept, **When** user lists all concepts, **Then** the concept appears in the output with its name and description

---

### User Story 2 - Link Papers to Concepts (Priority: P2)

A researcher wants to tag papers with the concepts they discuss. They create edges between papers and concepts, specifying the relationship type (introduces, applies, models, critiques, etc.) and an optional summary.

**Why this priority**: Paper-concept edges are the primary value proposition — they enable concept-based queries. Depends on concepts existing (P1).

**Independent Test**: Can be tested by linking a paper to a concept and verifying the edge exists. Requires at least one concept to exist.

**Acceptance Scenarios**:

1. **Given** an existing paper ID and concept ID, **When** user creates an edge with relationship type and summary, **Then** the edge is persisted in the edges store
2. **Given** an existing paper-concept edge, **When** user queries edges for that paper, **Then** the edge appears with its relationship type and summary
3. **Given** a concept ID that does not exist, **When** user attempts to create an edge to it, **Then** the system reports an error indicating the concept was not found
4. **Given** a standard relationship type from the vocabulary, **When** user creates an edge, **Then** the type is accepted without warning
5. **Given** a non-standard relationship type, **When** user creates an edge, **Then** the system accepts the type but may warn about non-standard usage

---

### User Story 3 - Query Papers by Concept (Priority: P2)

A researcher wants to find all papers in their collection that discuss a specific concept. They query by concept ID and see all papers linked to that concept, grouped by relationship type.

**Why this priority**: Same priority as linking — this is the read side of the core value proposition. Both are needed for the feature to be useful.

**Independent Test**: Can be tested by querying papers for a concept after creating edges. Returns meaningful results only when edges exist.

**Acceptance Scenarios**:

1. **Given** a concept with multiple linked papers, **When** user queries papers by concept ID, **Then** all linked papers are returned with their relationship types and summaries
2. **Given** a concept with no linked papers, **When** user queries papers by concept ID, **Then** an empty result is returned (not an error)
3. **Given** papers linked with different relationship types (introduces, applies, models), **When** user queries papers by concept, **Then** results can be filtered or grouped by relationship type

---

### User Story 4 - Query Concepts by Paper (Priority: P3)

A researcher wants to understand what concepts a specific paper relates to. They query by paper ID and see all concepts the paper is linked to, along with the relationship types.

**Why this priority**: Useful for understanding individual papers but less frequently needed than finding papers by concept. Nice-to-have after core functionality works.

**Independent Test**: Can be tested by querying concepts for a paper after creating edges.

**Acceptance Scenarios**:

1. **Given** a paper with multiple linked concepts, **When** user queries concepts by paper ID, **Then** all linked concepts are returned with relationship types
2. **Given** a paper with no linked concepts, **When** user queries concepts by paper ID, **Then** an empty result is returned

---

### User Story 5 - Merge Duplicate Concepts (Priority: P3)

A researcher discovers that two concepts they created are actually the same (e.g., "SHM" and "somatic-hypermutation"). They merge them, updating all edges to point to the surviving concept.

**Why this priority**: Data hygiene feature. Important for long-term usability but not blocking core workflows.

**Independent Test**: Can be tested by creating two concepts, linking papers to both, merging, and verifying all edges now point to the surviving concept.

**Acceptance Scenarios**:

1. **Given** two concepts where one should be merged into the other, **When** user executes merge, **Then** all edges pointing to the old concept are updated to point to the new concept
2. **Given** a merge operation, **When** complete, **Then** the old concept is deleted
3. **Given** aliases on the old concept, **When** merge completes, **Then** those aliases are added to the surviving concept

---

### Edge Cases

- What happens when user tries to delete a concept that has linked papers? System should warn, show edge count, and require `--force` flag. With `--force`, both the concept AND all edges pointing to it are deleted.
- What happens when user creates a concept with an ID that already exists? System should report an error.
- What happens when the concept ID contains invalid characters? System should validate IDs (alphanumeric, hyphens, underscores only).
- What happens when user merges a concept into itself? System should report an error.

## Requirements *(mandatory)*

### Functional Requirements

**Concept CRUD**:
- **FR-001**: System MUST allow users to create concepts with required ID and name, optional aliases (list of strings), and optional description
- **FR-002**: System MUST persist concepts to a JSONL file (`concepts.jsonl`) in the data directory
- **FR-003**: System MUST allow users to retrieve a concept by its ID
- **FR-004**: System MUST allow users to list all concepts with name and description
- **FR-005**: System MUST allow users to update a concept's name, aliases, or description
- **FR-006**: System MUST allow users to delete a concept (with safeguards if edges exist)

**Concept ID Validation**:
- **FR-007**: Concept IDs MUST be non-empty strings containing only lowercase alphanumeric characters, hyphens, and underscores
- **FR-008**: Concept IDs MUST be unique within the concepts store

**Paper-Concept Edges**:
- **FR-009**: System MUST allow users to create edges from papers to concepts with source_id (paper), target_id (concept), relationship_type, and optional summary
- **FR-010**: System MUST validate that the source paper exists in refs.jsonl before creating an edge
- **FR-011**: System MUST validate that the target concept exists in concepts.jsonl before creating an edge
- **FR-012**: System MUST accept any relationship type string but should warn for non-standard types not in relationship-types.json
- **FR-013**: Paper-concept edges MUST be stored in the existing edges.jsonl file (same as paper-paper edges)

**Querying**:
- **FR-014**: System MUST allow querying all papers linked to a given concept
- **FR-015**: System MUST allow querying all concepts linked to a given paper
- **FR-016**: Query results MUST include relationship type and summary for each edge

**Concept Merging**:
- **FR-017**: System MUST allow merging one concept into another, updating all edges
- **FR-018**: After merge, the source concept MUST be deleted
- **FR-019**: Aliases from the merged concept SHOULD be added to the surviving concept

**CLI Interface**:
- **FR-020**: System MUST provide `bip concept add` command for creating concepts
- **FR-021**: System MUST provide `bip concept get` command for retrieving a concept by ID
- **FR-022**: System MUST provide `bip concept list` command for listing all concepts
- **FR-023**: System MUST provide `bip concept update` command for modifying concepts
- **FR-024**: System MUST provide `bip concept delete` command for removing concepts
- **FR-025**: System MUST provide `bip concept papers` command for querying papers by concept
- **FR-026**: System MUST provide `bip paper concepts` command for querying concepts by paper
- **FR-027**: System MUST provide `bip concept merge` command for merging concepts
- **FR-028**: All concept commands MUST support `--human` flag for human-readable output (default is JSON)

**Index Rebuild**:
- **FR-029**: The `bip rebuild` command MUST include concepts in the SQLite index for efficient querying

### Key Entities

- **Concept**: A named idea, method, or phenomenon with unique ID, display name, optional aliases, and optional description. Concepts exist independently of papers.
- **Paper-Concept Edge**: A directed relationship from a paper to a concept, typed with a relationship verb (introduces, applies, models, etc.) and optional summary text.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can add a concept and retrieve it in under 2 seconds
- **SC-002**: Users can query papers by concept and receive results in under 3 seconds for collections up to 10,000 papers
- **SC-003**: All concept CRUD operations complete without data corruption (verified by round-trip tests)
- **SC-004**: Users can successfully merge concepts and verify all edges were updated
- **SC-005**: The concept workflow (add concept, link papers, query) can be completed in under 5 CLI commands

## Assumptions

- Paper IDs from Paperpile (e.g., `Halpern1998-yc`) and concept IDs (e.g., `somatic-hypermutation`) are unlikely to collide. No namespace prefix is needed initially.
- Users are comfortable with command-line interfaces for managing their knowledge graph.
- The existing edges.jsonl format can accommodate paper-concept edges without schema changes (target_id can reference either paper or concept).
- Standard relationship types are documented in relationship-types.json; users can reference this for guidance.

## Out of Scope

- Concept-to-concept edges (taxonomic relationships like "is-a" or "extends")
- Automatic concept extraction from paper text
- Concept embeddings or semantic search over concepts
- GUI or web interface for concept management
