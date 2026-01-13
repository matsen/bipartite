# Feature Specification: Core Reference Manager

**Feature Branch**: `001-core-reference-manager`
**Created**: 2026-01-12
**Status**: Draft
**Input**: Phase I: Core reference manager - bp CLI with Paperpile JSON import, JSONL storage, ephemeral query layer, BibTeX export, and PDF opening via configured folder path

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Import References from Paperpile (Priority: P1)

A researcher exports their library from Paperpile as JSON and imports it into bipartite to create a searchable, agent-accessible reference collection. This is the foundation—without import, nothing else works.

**Why this priority**: Import is the entry point for all data. No other feature is useful without references in the system.

**Independent Test**: Export a Paperpile JSON file, run the import command, verify references are stored and queryable.

**Acceptance Scenarios**:

1. **Given** a new bipartite repository, **When** user runs import with a Paperpile JSON export, **Then** all references are stored with their metadata (title, authors, DOI, abstract, venue, dates, PDF paths).

2. **Given** an existing repository with references, **When** user imports an updated Paperpile export containing the same papers with updated metadata, **Then** existing entries are updated (matched by DOI) and new entries are added.

3. **Given** a Paperpile export with papers lacking DOIs, **When** user imports, **Then** papers are stored using the Paperpile citekey as identifier and flagged as missing DOI.

4. **Given** a Paperpile export with multiple PDF attachments per paper, **When** user imports, **Then** the main PDF is stored separately from supplementary PDFs.

---

### User Story 2 - Search and Query References (Priority: P2)

A researcher or agent searches their reference collection to find relevant papers. Results are returned in structured JSON format for agent consumption, with human-readable format available.

**Why this priority**: Search enables finding papers, which is essential for all downstream workflows (opening, exporting, citing).

**Independent Test**: Import some references, run search queries, verify correct papers are returned with expected metadata.

**Acceptance Scenarios**:

1. **Given** a repository with imported references, **When** user searches by keyword, **Then** matching papers are returned with their full metadata.

2. **Given** a repository with references, **When** user requests a specific paper by ID, **Then** the paper's complete metadata is returned.

3. **Given** a repository with references, **When** user lists all papers, **Then** all papers are returned in a consistent, parseable format.

4. **Given** any query command, **When** user requests JSON output, **Then** output is valid JSON suitable for piping to other tools.

---

### User Story 3 - Open PDFs for Reading (Priority: P3)

A researcher or agent wants to open a paper's main PDF in the system viewer. This is a core agent use case—agents can direct humans to read specific papers.

**Why this priority**: PDF access is a key design goal. Agents opening papers for humans is a primary use case.

**Independent Test**: Configure PDF path, import references with PDF paths, run open command, verify PDF opens in viewer.

**Acceptance Scenarios**:

1. **Given** a configured PDF folder path and a paper with a linked PDF, **When** user runs open command with paper ID, **Then** the PDF opens in the system's configured viewer.

2. **Given** a paper with supplementary PDFs, **When** user requests to open supplements, **Then** the correct supplementary PDF opens.

3. **Given** a paper whose PDF file does not exist at the expected path, **When** user runs open command, **Then** a clear error message explains the missing file and expected location.

---

### User Story 4 - Export to BibTeX (Priority: P4)

A researcher writing a paper exports references to BibTeX format for use with LaTeX. They can export their entire collection or specific papers by ID.

**Why this priority**: BibTeX export completes the academic writing workflow—import from reference manager, work with bipartite, export for LaTeX.

**Independent Test**: Import references, export to BibTeX, verify output is valid BibTeX that LaTeX can process.

**Acceptance Scenarios**:

1. **Given** a repository with references, **When** user exports all papers to BibTeX, **Then** valid BibTeX entries are produced for each paper.

2. **Given** a repository with references, **When** user exports specific papers by ID, **Then** only those papers appear in the BibTeX output.

3. **Given** papers with varying metadata completeness, **When** exported to BibTeX, **Then** entries include all available fields and omit unavailable fields gracefully.

---

### User Story 5 - Initialize and Configure Repository (Priority: P5)

A researcher sets up a new bipartite repository, configuring the path to their PDF folder (e.g., Paperpile's Google Drive sync folder).

**Why this priority**: Initialization is required once per repository. Other features depend on configuration but init is a one-time operation.

**Independent Test**: Run init in an empty directory, configure PDF path, verify configuration is persisted.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** user runs init command, **Then** a bipartite repository structure is created with necessary files and folders.

2. **Given** an initialized repository, **When** user configures PDF folder path, **Then** the path is stored and used for resolving paper PDF locations.

3. **Given** a directory that is already a bipartite repository, **When** user runs init, **Then** the system refuses with a clear error (no silent overwrite).

---

### User Story 6 - Rebuild Query Layer (Priority: P6)

After pulling changes from git (e.g., collaborator added papers), the researcher rebuilds the query layer from the source-of-truth data file.

**Why this priority**: Rebuild enables the git-based collaboration workflow. Lower priority because it's a maintenance operation, not daily use.

**Independent Test**: Modify the source data file, run rebuild, verify query layer reflects the changes.

**Acceptance Scenarios**:

1. **Given** a repository where the source data has changed (e.g., after git pull), **When** user runs rebuild, **Then** the query layer is reconstructed from the source data.

2. **Given** a corrupted or missing query layer, **When** user runs rebuild, **Then** the query layer is recreated successfully.

---

### Edge Cases

- What happens when importing a paper with a DOI that matches an existing entry but different citekey? (Update metadata, keep existing ID)
- What happens when two papers in an import have the same citekey but different DOIs? (Suffix the second ID, e.g., `Author2026-ab-2`)
- What happens when PDF path configuration points to a non-existent directory? (Error immediately on config, not silently on later use)
- What happens when search returns no results? (Empty result set, not an error)
- What happens when exporting a paper with minimal metadata to BibTeX? (Include available fields, produce valid BibTeX)
- What happens when the source data file is malformed? (Error with line number and description, not silent failure)

## Requirements *(mandatory)*

### Functional Requirements

**Initialization & Configuration**

- **FR-001**: System MUST initialize a repository structure in the current directory via `bp init`
- **FR-002**: System MUST store configuration (PDF folder path, PDF reader preference) via `bp config`
- **FR-003**: System MUST fail immediately if initializing an already-initialized directory
- **FR-004**: System MUST fail immediately if configured PDF path does not exist

**Import**

- **FR-005**: System MUST import references from Paperpile JSON export format
- **FR-006**: System MUST extract and store: title, authors (with ORCID if present), DOI, abstract, venue, publication date, citekey, PDF paths
- **FR-007**: System MUST use DOI as the primary key for deduplication on re-import
- **FR-008**: System MUST preserve existing internal IDs when updating papers by DOI match
- **FR-009**: System MUST generate unique IDs for new papers using source citekey, with suffix if collision occurs
- **FR-010**: System MUST distinguish main PDF from supplementary PDFs in Paperpile attachments
- **FR-011**: System MUST track import source (type: paperpile, id: paperpile's internal ID)

**Query**

- **FR-012**: System MUST support keyword search across title, authors, and abstract via `bp search`
- **FR-013**: System MUST retrieve a single paper by ID via `bp get <id>`
- **FR-014**: System MUST list all papers via `bp list`
- **FR-015**: System MUST output JSON by default for all query commands
- **FR-016**: System MUST support human-readable output format via flag

**PDF Access**

- **FR-017**: System MUST open a paper's main PDF via `bp open <id>`
- **FR-018**: System MUST resolve PDF paths relative to the configured PDF folder
- **FR-019**: System MUST support opening supplementary PDFs
- **FR-020**: System MUST fail with clear error if PDF file not found

**Export**

- **FR-021**: System MUST export all papers to BibTeX via `bp export --bibtex`
- **FR-022**: System MUST export specific papers by ID via `bp export --bibtex --keys id1,id2`
- **FR-023**: System MUST produce valid classic BibTeX format (not BibLaTeX)

**Data Integrity**

- **FR-024**: System MUST store all reference data in a human-readable, git-mergeable format
- **FR-025**: System MUST rebuild query layer from source data via `bp rebuild`
- **FR-026**: System MUST validate source data integrity via `bp check`

### Key Entities

- **Reference**: A paper or article with metadata. Key attributes: internal ID (stable, derived from citekey), DOI (primary deduplication key), title, authors (list with first/last/ORCID), abstract, venue, publication date (year/month/day), main PDF path, supplementary PDF paths, import source (type + external ID).

- **Author**: A person who wrote a paper. Key attributes: first name, last name, ORCID (optional).

- **Configuration**: Repository settings. Key attributes: PDF folder root path, PDF reader preference (for page targeting).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can import a 1000-paper Paperpile export in under 30 seconds
- **SC-002**: Search queries return results in under 500ms for collections up to 10,000 papers
- **SC-003**: Opening a PDF completes in under 1 second (time to launch viewer)
- **SC-004**: CLI startup time is under 100ms (feels instant)
- **SC-005**: Re-importing an unchanged export produces no modifications to stored data
- **SC-006**: All CLI commands produce valid, parseable JSON when requested
- **SC-007**: BibTeX export produces files that compile without errors in standard LaTeX
- **SC-008**: Repository data survives git clone/pull/push cycles without corruption
- **SC-009**: Agents can complete search-open-paper workflow without human intervention

## Assumptions

- Users have an existing Paperpile account and can export their library as JSON
- Users have PDFs synced to a local folder (e.g., via Google Drive for Paperpile)
- Target collection size is up to 10,000 papers (researcher's personal library, not institutional scale)
- Primary users are researchers and AI agents working on academic writing tasks
- macOS and Linux are the target platforms; Windows is not a priority
