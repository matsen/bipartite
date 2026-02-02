# Feature Specification: Projects and Repos

**Feature Branch**: `011-repo-nodes`
**Created**: 2026-01-23
**Status**: Draft
**Input**: Extend the knowledge graph to include projects (first-class nodes representing ongoing work) and repos (GitHub repositories belonging to projects). Concepts connect to projects (not repos), enforcing a disciplined graph structure where concepts bridge the literature and the researcher's active work.

## Motivation

The bipartite vision (from VISION.md) describes a graph with:
- **One side**: The researcher's world (notes, code, artifacts, concepts)
- **Other side**: The academic literature (papers, citations, authors)

Currently, bip supports papers and concepts but has no way to represent the "researcher's world" — the ongoing projects that concepts inform. This feature closes that gap.

### Architecture: Concepts as the Bridge

The graph structure enforces discipline through concept-mediated connections:

```
papers ←——→ concepts ←——→ projects
  (literature)    (ideas)    (your work)
                                 │
                                 ├── repo: code
                                 └── repo: manuscript
```

**No direct paper↔project edges.** If you want to connect a paper to a project, you must name the concept that bridges them. This forces clarity about *why* the paper matters to the project.

For example, instead of "project X is informed by paper Y", you must articulate:
- Paper Y **introduces** concept Z
- Project X **implements** concept Z

This discipline yields a cleaner, more reusable knowledge graph.

### Projects vs Repos

- **Project**: A first-class node representing a logical unit of work (e.g., "dasm2"). Edges connect concepts to projects.
- **Repo**: A GitHub repository belonging to a project. Repos are tracked for metadata (URL, description, topics) but edges do NOT connect directly to repos.

A typical project has:
- One or more code repos
- One or more writing repos (manuscripts in progress)

### Writing Repos vs Papers

- A **writing repo** represents a manuscript in progress (not yet published) — belongs to a project
- A **paper** represents published work (in refs.jsonl) — a separate node type

These are different lifecycle stages, not two nodes for the same thing. Once a manuscript is published, add it to refs.jsonl as a paper; the writing repo can remain as historical artifact or be removed.

### Use Cases

1. **Concept-to-project traceability**: "Which projects implement variational inference?"
2. **Literature gap identification**: "My project applies concept X — what papers introduce X?"
3. **Lab portfolio view**: "Show me all projects in our group and their connected concepts"
4. **Writing support**: "What concepts does my project discuss? Am I missing key papers?"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a Project (Priority: P1)

A researcher wants to create a project representing an ongoing unit of work. The project is a first-class node that concepts can link to.

**Why this priority**: Projects are the foundation — without them, no concept↔project edges can exist.

**Independent Test**: Can be fully tested by adding a project, listing it, and verifying persistence.

**Acceptance Scenarios**:

1. **Given** a project ID and name, **When** user creates a project, **Then** the project node is persisted
2. **Given** an existing project ID, **When** user attempts to create another with the same ID, **Then** the system reports a conflict
3. **Given** a project ID, **When** user retrieves the project, **Then** all stored metadata is returned including name, description, and linked repos

---

### User Story 2 - Add Repos to a Project (Priority: P1)

A researcher wants to register GitHub repositories belonging to a project. The system fetches metadata from GitHub and associates the repo with the project.

**Why this priority**: Repos provide the concrete GitHub links and metadata for projects.

**Independent Test**: Can be tested by adding a repo to a project and verifying the association.

**Acceptance Scenarios**:

1. **Given** a valid GitHub URL and project ID, **When** user adds a repo, **Then** the system fetches repo metadata and creates a repo node linked to the project
2. **Given** org/repo shorthand (e.g., `matsen/dasm2`), **When** user adds a repo, **Then** the system resolves it to the full GitHub URL and fetches metadata
3. **Given** a non-GitHub repo, **When** user adds a repo with `--manual` flag, **Then** the system creates a repo node with user-provided metadata
4. **Given** a project, **When** user lists its repos, **Then** all repos belonging to that project are returned
5. **Given** an existing repo with the same GitHub URL, **When** user attempts to add it again, **Then** the system reports a conflict

---

### User Story 3 - Link Concepts to Projects (Priority: P1)

A researcher wants to document which concepts their project implements, applies, or studies. This is the primary way to connect projects to the literature — through concepts.

**Why this priority**: This is the core value — concept↔project edges bridge literature and work.

**Independent Test**: Can be tested by linking a concept to a project and querying the edge.

**Acceptance Scenarios**:

1. **Given** an existing concept ID and project ID, **When** user creates an edge, **Then** the edge is persisted with relationship type and summary
2. **Given** a concept, **When** user queries projects that implement it, **Then** all concept→project edges are returned
3. **Given** a project, **When** user queries concepts it relates to, **Then** all relevant edges are returned
4. **Given** a project linked to concept X, and concept X linked to paper Y, **When** user queries papers relevant to the project, **Then** paper Y is returned (transitive query through concepts)

---

### User Story 4 - Refresh Repo Metadata from GitHub (Priority: P3)

A researcher wants to update a repo's metadata after changes on GitHub (new description, topics, etc.).

**Why this priority**: Nice-to-have maintenance feature.

**Independent Test**: Can be tested by modifying repo on GitHub, running refresh, and verifying changes.

**Acceptance Scenarios**:

1. **Given** a repo linked to GitHub, **When** user runs refresh, **Then** metadata is re-fetched and updated
2. **Given** a manual repo (no GitHub link), **When** user runs refresh, **Then** the system reports no remote source to refresh from

---

### User Story 5 - View Project Graph Neighborhood (Priority: P2)

A researcher wants to see all concepts connected to a project, and transitively, all papers that discuss those concepts.

**Why this priority**: Valuable synthesis view for understanding the intellectual foundations of work.

**Independent Test**: Can be tested by querying a project's full neighborhood.

**Acceptance Scenarios**:

1. **Given** a project with linked concepts, **When** user queries project graph, **Then** all connected concepts are returned
2. **Given** a project linked to concepts, **When** user queries transitive papers, **Then** all papers linked to those concepts are returned
3. **Given** the neighborhood data, **When** displayed, **Then** the concept layer is clearly shown as the bridge between papers and the project

---

### Edge Cases

- What happens when GitHub API rate limits are hit? System should cache aggressively and provide helpful error message with retry guidance.
- What happens when a GitHub repo is deleted or made private? System should keep the repo node but mark it as "unreachable" on next refresh attempt.
- What happens when user tries to delete a project with linked edges or repos? System should warn, show counts, and require `--force` flag.
- What happens with GitHub Enterprise or GitLab repos? Initially out of scope; system should accept manual repos for these.
- What happens when project ID conflicts with paper or concept ID? System should validate uniqueness across all node types.
- What happens when user tries to create a paper↔project edge? System should reject it and explain that connections must go through concepts.

## Data Model

### Project Node Schema

Projects are stored in `projects.jsonl`:

```jsonl
{"id":"dasm2","name":"DASM2","description":"Distance-based antibody sequence modeling","created_at":"2026-01-23T10:00:00Z","updated_at":"2026-01-23T10:00:00Z"}
{"id":"phylo-review","name":"Phylogenetics Review","description":"Review paper on modern phylogenetic methods","created_at":"2026-01-23T11:00:00Z","updated_at":"2026-01-23T11:00:00Z"}
{"id":"bipartite","name":"Bipartite","description":"Agent-first academic reference manager","created_at":"2026-01-23T10:00:00Z","updated_at":"2026-01-23T10:00:00Z"}
```

**Required fields**:
- `id`: Unique identifier (lowercase alphanumeric, hyphens, underscores)
- `name`: Display name

**Optional fields**:
- `description`: Project description
- `created_at`: Auto-populated timestamp when the project node was created
- `updated_at`: Auto-populated timestamp when the project node was last modified

### Repo Node Schema

Repos are stored in `repos.jsonl`:

```jsonl
{"id":"dasm2-code","project":"dasm2","type":"github","github_url":"https://github.com/matsen/dasm2","name":"dasm2","description":"Distance-based antibody sequence modeling","topics":["antibodies","ml"],"language":"Python","created_at":"2026-01-23T10:00:00Z","updated_at":"2026-01-23T10:00:00Z"}
{"id":"dasm2-paper","project":"dasm2","type":"github","github_url":"https://github.com/matsen/dasm2-paper","name":"dasm2-paper","description":"Manuscript for DASM2 methods paper","topics":["manuscript"],"language":"LaTeX","created_at":"2026-01-23T11:00:00Z","updated_at":"2026-01-23T11:00:00Z"}
{"id":"bipartite-code","project":"bipartite","type":"github","github_url":"https://github.com/matsen/bipartite","name":"bipartite","description":"Agent-first academic reference manager","topics":["reference-manager","cli"],"language":"Go","created_at":"2026-01-23T10:00:00Z","updated_at":"2026-01-23T10:00:00Z"}
```

**Required fields**:
- `id`: Unique identifier (lowercase alphanumeric, hyphens, underscores)
- `project`: ID of the project this repo belongs to (required — every repo must belong to a project)
- `type`: `github` or `manual`
- `name`: Display name

**Optional fields**:
- `github_url`: Full GitHub URL (required if type=github)
- `description`: Repo description
- `topics`: Array of topic tags (from GitHub or user-defined)
- `language`: Primary programming language (from GitHub)
- `created_at`: Auto-populated timestamp when the repo node was created
- `updated_at`: Auto-populated timestamp when the repo node was last modified

### Edge Relationship Types

New relationship types for `relationship-types.json`:

```json
{
  "concept-project": [
    {"type": "implemented-in", "description": "Concept is implemented in this project"},
    {"type": "applied-in", "description": "Concept is applied/used in this project"},
    {"type": "studied-by", "description": "Concept is studied/investigated by this project"}
  ],
  "project-concept": [
    {"type": "introduces", "description": "Project introduces or defines this concept"},
    {"type": "refines", "description": "Project refines understanding of this concept"}
  ]
}
```

**Note**: There are NO `paper-project`, `project-paper`, `paper-repo`, or `concept-repo` relationship types. Papers connect to projects ONLY through concepts. Repos have no edges — they are metadata children of projects.

### Node Type Disambiguation

Since edges reference nodes by ID and we now have three edge-capable node types (papers, concepts, projects), we need a disambiguation strategy.

**Strategy**: Type-prefixed IDs for unambiguous identification.

All node IDs include a type prefix:
- Papers: `paper:<id>` (e.g., `paper:10.1038/s41586-021-03819-2`)
- Concepts: `concept:<id>` (e.g., `concept:variational-inference`)
- Projects: `project:<id>` (e.g., `project:dasm2`)
- Repos: `repo:<id>` (e.g., `repo:dasm2-code`)

This enables:
- Validation without loading all node stores
- Batch schema updates for a given type
- Clear provenance in edge definitions

Edge schema uses prefixed IDs directly:

```json
{"source_id": "concept:variational-inference", "target_id": "project:dasm2", ...}
```

The `source_type`/`target_type` fields become redundant but MAY be retained for query optimization.

### Updated Edge Schema

```jsonl
{"source_id":"concept:variational-inference","target_id":"project:dasm2","relationship_type":"implemented-in","summary":"DASM2 uses variational inference for the latent space model","created_at":"2026-01-23T12:00:00Z"}
```

### Graph Structure

The enforced graph structure:

```
┌─────────┐         ┌──────────┐         ┌──────────┐
│  Papers │ ←─────→ │ Concepts │ ←─────→ │ Projects │
└─────────┘         └──────────┘         └──────────┘
                                              │
                                         ┌────┴────┐
                                         │  Repos  │
                                         │ (no edges)
                                         └─────────┘

Papers ↔ Concepts: introduces, applies, models, evaluates-with, critiques, extends
Concepts ↔ Projects: implemented-in, applied-in, studied-by, introduces, refines
Projects → Repos: containment (not an edge — repos have a `project` field)
```

Transitive queries (e.g., "papers relevant to project X") traverse through concepts.

## Requirements *(mandatory)*

### Functional Requirements

**Project CRUD**:
- **FR-001**: System MUST allow users to create projects with ID, name, and optional description
- **FR-002**: System MUST persist projects to `projects.jsonl` in the data directory
- **FR-003**: System MUST allow users to retrieve a project by its ID
- **FR-004**: System MUST allow users to list all projects
- **FR-005**: System MUST allow users to update a project's metadata
- **FR-006**: System MUST allow users to delete a project, which cascade-deletes all repos belonging to that project and all edges involving the project

**Project ID Validation**:
- **FR-007**: Project IDs MUST be non-empty strings containing only lowercase alphanumeric characters, hyphens, and underscores
- **FR-008**: Project IDs MUST be unique within the projects store
- **FR-009**: Project IDs MUST NOT conflict with existing paper IDs or concept IDs

**Repo CRUD**:
- **FR-010**: System MUST allow users to create repos from GitHub URLs or org/repo shorthand
- **FR-011**: System MUST fetch and store GitHub metadata (name, description, topics, language) when creating GitHub-linked repos
- **FR-012**: System MUST allow users to create manual repos with user-provided metadata
- **FR-013**: System MUST persist repos to `repos.jsonl` in the data directory
- **FR-014**: System MUST require a project ID when creating a repo (every repo belongs to a project)
- **FR-015**: System MUST validate that the project exists before creating a repo
- **FR-016**: System MUST allow users to retrieve a repo by its ID
- **FR-017**: System MUST allow users to list all repos, optionally filtered by project
- **FR-018**: System MUST allow users to update a repo's metadata
- **FR-019**: System MUST allow users to delete a repo
- **FR-020**: System MUST allow users to refresh GitHub metadata for linked repos

**Repo ID Validation**:
- **FR-021**: Repo IDs MUST be non-empty strings containing only lowercase alphanumeric characters, hyphens, and underscores
- **FR-022**: Repo IDs MUST be unique within the repos store
- **FR-023**: System SHOULD derive default repo ID from GitHub repo name (e.g., `matsen/dasm2` → `dasm2`)
- **FR-024**: System MUST allow user to override the derived ID via `--id` flag
- **FR-024a**: A GitHub URL MUST NOT appear in more than one repo entry (enforces one-project-per-repo)

**Concept↔Project Edges**:
- **FR-025**: System MUST allow edges from concepts to projects (concept→project)
- **FR-026**: System MUST allow edges from projects to concepts (project→concept)
- **FR-027**: System MUST NOT allow edges directly between papers and projects
- **FR-028**: System MUST NOT allow edges to or from repos (repos have no edges)
- **FR-029**: System MUST reject attempts to create paper↔project or *↔repo edges with a clear error message
- **FR-030**: Edge schema MUST include source_type and target_type fields to disambiguate node types
- **FR-031**: System MUST validate that source and target nodes exist before creating edges

**Querying**:
- **FR-032**: System MUST allow querying all concepts linked to a given project
- **FR-033**: System MUST allow querying all projects linked to a given concept
- **FR-034**: System MUST support transitive queries: papers relevant to a project (via shared concepts)
- **FR-035**: System MUST support filtering queries by relationship type
- **FR-036**: System MUST support querying a project's full "neighborhood" (concepts + transitive papers)

**CLI Interface — Project Commands**:
- **FR-037**: System MUST provide `bip project add <id> --name <name>` for creating projects
- **FR-038**: System MUST provide `bip project get <id>` for retrieving a project
- **FR-039**: System MUST provide `bip project list` for listing all projects
- **FR-040**: System MUST provide `bip project update <id>` for modifying projects
- **FR-041**: System MUST provide `bip project delete <id>` for removing projects
- **FR-042**: System MUST provide `bip project repos <id>` for listing repos in a project
- **FR-043**: System MUST provide `bip project concepts <id>` for querying linked concepts
- **FR-044**: System MUST provide `bip project papers <id>` for transitive paper query (via concepts)
- **FR-045**: All project commands MUST support `--json` flag for JSON output (default is human-readable)

**CLI Interface — Repo Commands**:
- **FR-047**: System MUST provide `bip repo add <github-url-or-org/repo> --project <id> [--id <id>]` for adding GitHub repos
- **FR-048**: System MUST provide `bip repo add --manual --name <name> --project <id> [--id <id>]` for adding manual repos
- **FR-049**: System MUST provide `bip repo get <id>` for retrieving a repo
- **FR-050**: System MUST provide `bip repo list` for listing all repos
- **FR-051**: System MUST provide `bip repo list --project <id>` for filtering by project
- **FR-052**: System MUST provide `bip repo update <id>` for modifying repos
- **FR-053**: System MUST provide `bip repo delete <id>` for removing repos
- **FR-054**: System MUST provide `bip repo refresh <id>` for updating GitHub metadata
- **FR-055**: All repo commands MUST support `--json` flag for JSON output (default is human-readable)

**Edge Commands (extensions)**:
- **FR-056**: `bip edge add` MUST support `--source-type` and `--target-type` flags
- **FR-057**: System SHOULD infer types when unambiguous (e.g., if source matches a concept ID and target matches a project ID)
- **FR-058**: `bip edge list` MUST support filtering by source/target type
- **FR-059**: `bip edge add` MUST reject edges involving repos with a clear error message

**Index Rebuild**:
- **FR-060**: The `bip rebuild` command MUST include projects and repos in the SQLite index
- **FR-061**: The `bip check` command MUST validate concept↔project edges
- **FR-062**: The `bip check` command MUST report any paper↔project or *↔repo edges as invalid
- **FR-063**: The `bip check` command MUST verify all repos reference valid projects

### Non-Functional Requirements

- **NFR-001**: GitHub API calls MUST respect rate limits and implement exponential backoff
- **NFR-002**: GitHub metadata SHOULD be cached to minimize API calls
- **NFR-003**: Project and repo operations MUST complete in under 3 seconds for local operations
- **NFR-004**: GitHub metadata fetch MUST timeout after 10 seconds with clear error message
- **NFR-005**: Transitive queries (project→concepts→papers) MUST complete in under 5 seconds for typical graph sizes

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a project and add repos to it in under 10 seconds
- **SC-002**: Users can link a concept to a project and query it back in under 2 seconds
- **SC-003**: Users can query a project's full neighborhood (concepts + transitive papers) in under 5 seconds
- **SC-004**: All project and repo CRUD operations complete without data corruption (verified by round-trip tests)
- **SC-005**: The workflow "create project → add repo → link concept → query transitive papers" completes in under 8 CLI commands
- **SC-006**: Attempts to create direct paper↔project or *↔repo edges are rejected with a helpful error message

## Implementation Notes

### GitHub API Integration

Use the GitHub REST API (no authentication required for public repos, optional for private):

```
GET https://api.github.com/repos/{owner}/{repo}
```

Returns: name, description, topics, language, html_url, created_at, updated_at, etc.

For authenticated access (private repos, higher rate limits), support `github_token` in global config.

### Migration Path

Existing edges use a two-field identity (source_id, target_id). Adding source_type/target_type requires either:

1. **Schema migration**: Add type fields to existing edges (default based on ID lookup)
2. **New edge format**: Support both old and new formats during transition

Recommendation: Schema migration with `bip migrate` command that adds `"source_type":"paper","target_type":"paper"` or `"target_type":"concept"` to existing edges based on ID lookup. Existing edges are all paper↔paper or paper↔concept, so no project edges need migration.

### Transitive Query Implementation

For "papers relevant to project X":
1. Find all concepts linked to project X
2. Find all papers linked to those concepts
3. Return papers with the concept as the "bridge" explanation

This can be implemented as:
- A SQL join across the concepts and edges tables
- Or a two-step query with in-memory join

### Visualization Integration

The existing `bip viz` command (spec 007) should be extended to:
- Include project nodes with distinct styling (different color/shape from papers and concepts)
- Show concept↔project edges
- Optionally show repos as sub-nodes of projects (or in a separate panel)
- Support filtering to show only a project's neighborhood
- Clearly visualize the three-layer structure: papers ↔ concepts ↔ projects

## Assumptions

- GitHub public API provides sufficient metadata for most use cases
- Project IDs are unlikely to collide with paper/concept IDs in practice (but we validate anyway)
- Users have network access for GitHub API calls (or use manual repos for offline work)
- The group primarily uses GitHub; GitLab/Bitbucket support can be added later as manual repos
- Users will be disciplined about creating concepts as the bridge between papers and projects
- Every repo belongs to exactly one project

## Out of Scope

- GitLab, Bitbucket, or other forge integrations (use manual repos)
- Automatic edge extraction from code (e.g., parsing citations in README or docstrings)
- Project-to-project edges (e.g., "project A depends on project B")
- Issue/PR-level granularity (repos are the finest unit tracked)
- Real-time sync with GitHub (manual refresh only)
- Private repo access without user-provided token
- Direct paper↔project edges (by design — must go through concepts)
- Edges to/from repos (repos are metadata, not edge endpoints)

## Open Questions (Resolved)

1. **ID collision handling**: ✅ Fail with error requiring explicit ID change (no auto-suffix).

2. **Bulk import**: ✅ Out of scope for initial implementation.

3. **flowc integration**: ✅ Use separate `projects.jsonl` and `repos.jsonl` files for simplicity. Integration with `sources.yml` deferred to issue #30.

4. **Concept auto-suggestion**: ✅ Not needed - AI agents will suggest concepts based on graph queries.

5. **Repo reassignment**: ✅ Not supported initially. Delete and re-add if needed.
