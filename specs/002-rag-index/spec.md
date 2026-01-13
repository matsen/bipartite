# Feature Specification: RAG Index for Semantic Search

**Feature Branch**: `002-rag-index`
**Created**: 2026-01-12
**Status**: Draft
**Input**: Phase II RAG Index - semantic search over paper abstracts using vector embeddings

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Semantic Search by Concept (Priority: P1)

A researcher or agent wants to find papers related to a concept, even when papers don't contain the exact keywords. For example, searching "methods for inferring evolutionary trees" should find papers about "phylogenetics" and "tree reconstruction" even if they don't use the exact query terms.

**Why this priority**: This is the core value proposition of RAG—finding conceptually related papers that keyword search would miss.

**Independent Test**: Import references with abstracts, build the semantic index, run a conceptual query, verify that results include semantically related papers that wouldn't match keyword search.

**Acceptance Scenarios**:

1. **Given** a repository with indexed abstracts, **When** user searches "methods for inferring evolutionary trees", **Then** papers about phylogenetics and tree reconstruction are returned, even without exact keyword matches.

2. **Given** a repository with indexed abstracts, **When** user searches "machine learning for protein structure", **Then** papers about deep learning, neural networks, and structural biology are returned based on semantic similarity.

3. **Given** a repository with indexed abstracts, **When** user runs semantic search, **Then** results include a relevance score indicating how closely each paper matches the query concept.

4. **Given** a semantic search, **When** user requests JSON output, **Then** output includes paper metadata plus similarity scores in a structured format.

---

### User Story 2 - Find Similar Papers (Priority: P2)

A researcher has found a useful paper and wants to discover other papers in their collection that cover similar topics. This enables literature exploration: "I liked this paper, what else do I have like it?"

**Why this priority**: Finding similar papers is a natural follow-up to semantic search and leverages the same embedding infrastructure.

**Independent Test**: Import references, build index, select a paper, run similar-papers command, verify results are topically related to the source paper.

**Acceptance Scenarios**:

1. **Given** a repository with indexed abstracts and a paper ID, **When** user requests similar papers, **Then** papers with similar abstracts are returned, ranked by similarity.

2. **Given** a paper about MCMC methods in phylogenetics, **When** user requests similar papers, **Then** other papers about Bayesian inference, phylogenetics, or sampling methods are returned.

3. **Given** a request for similar papers, **When** user specifies a limit, **Then** only the top N most similar papers are returned.

---

### User Story 3 - Rebuild Semantic Index (Priority: P3)

After importing new papers or pulling changes from git, the researcher rebuilds the semantic index to include new abstracts in the vector search.

**Why this priority**: Index rebuild is a maintenance operation but essential for keeping semantic search current with the reference collection.

**Independent Test**: Import new papers, run index rebuild, verify new papers appear in semantic search results.

**Acceptance Scenarios**:

1. **Given** a repository where new papers have been imported, **When** user rebuilds the semantic index, **Then** the index includes embeddings for all papers with abstracts.

2. **Given** a corrupted or missing semantic index, **When** user rebuilds, **Then** the index is recreated from the source data.

3. **Given** a rebuild operation, **When** complete, **Then** statistics are reported (papers indexed, papers skipped due to missing abstracts).

---

### User Story 4 - Check Index Health (Priority: P4)

A researcher wants to verify the semantic index is complete and healthy—that all papers with abstracts are indexed and the index is queryable.

**Why this priority**: Diagnostic capability helps troubleshoot when semantic search isn't returning expected results.

**Independent Test**: Build index, run check command, verify it reports index status and any gaps.

**Acceptance Scenarios**:

1. **Given** a repository with a semantic index, **When** user runs index check, **Then** system reports: total papers, papers with abstracts, papers indexed, papers missing from index.

2. **Given** papers that have abstracts but are not indexed, **When** user runs check, **Then** those papers are listed as needing indexing.

---

### Edge Cases

- What happens when searching with an empty query? (Error with clear message, not empty results)
- What happens when a paper has no abstract? (Paper is excluded from semantic index; keyword search still works)
- What happens when the semantic index doesn't exist? (Error directing user to build it first)
- What happens when the embedding service is unavailable? (Error with clear message about the failure)
- What happens when a query returns no results above the similarity threshold? (Empty result set with message, not error)
- What happens when finding similar papers for a paper with no abstract? (Error explaining the paper has no abstract to compare)

## Requirements *(mandatory)*

### Functional Requirements

**Semantic Search**

- **FR-001**: System MUST support semantic search via `bp semantic <query>` command
- **FR-002**: System MUST return papers ranked by semantic similarity to the query
- **FR-003**: System MUST include similarity scores in search results
- **FR-004**: System MUST support `--limit N` flag to control number of results (default: 10)
- **FR-005**: System MUST support `--threshold T` flag to filter by minimum similarity score
- **FR-006**: System MUST output JSON by default, with `--human` flag for readable format

**Similar Papers**

- **FR-007**: System MUST find similar papers via `bp similar <id>` command
- **FR-008**: System MUST return papers ranked by similarity to the specified paper's abstract
- **FR-009**: System MUST support `--limit N` flag for similar papers (default: 10)
- **FR-010**: System MUST error clearly if the source paper has no abstract

**Index Management**

- **FR-011**: System MUST build/rebuild semantic index via `bp index build` command
- **FR-012**: System MUST report progress during index building (papers processed)
- **FR-013**: System MUST report statistics on completion (total indexed, skipped)
- **FR-014**: System MUST skip papers without abstracts during indexing (with count reported)
- **FR-015**: System MUST check index health via `bp index check` command
- **FR-016**: System MUST store semantic index in a location that can be gitignored (ephemeral, rebuildable)

**Architecture Constraints**

- **FR-017**: System MUST NOT require a separate server process for semantic search
- **FR-018**: System MUST rebuild the semantic index entirely from JSONL source data
- **FR-019**: System MUST work offline after initial index build (embeddings cached locally)

### Key Entities

- **Embedding**: A vector representation of a paper's abstract. Key attributes: paper ID, vector (list of floats), model identifier used to generate it.

- **Semantic Index**: Collection of embeddings for all indexed papers. Attributes: embedding model identifier, creation timestamp, paper count.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Semantic search returns results in under 1 second for collections up to 10,000 papers
- **SC-002**: Users find relevant papers that keyword search misses in 8 out of 10 conceptual queries
- **SC-003**: Index build completes in under 5 minutes for 6,000 papers (current collection size)
- **SC-004**: Similar-papers command returns topically related papers for 90% of test cases
- **SC-005**: Index can be fully rebuilt from JSONL source without external dependencies (except embedding model)
- **SC-006**: All semantic search commands produce valid, parseable JSON when requested
- **SC-007**: CLI startup time remains under 200ms even with semantic index loaded

## Clarifications

### Session 2026-01-12

- Q: Should index building require local-only (no network) or allow external API? → A: Defer to implementation; current spec wording is sufficient.

## Assumptions

- Papers with abstracts (94% of current collection) are the target for semantic indexing
- Embedding generation may require an external API or local model; the choice is an implementation detail
- The semantic index is ephemeral and can be rebuilt from source data at any time
- Similarity thresholds and ranking algorithms are implementation details to be tuned
- Target collection size is up to 10,000 papers (same as Phase I)
- Users accept that semantic search quality depends on abstract quality and embedding model
