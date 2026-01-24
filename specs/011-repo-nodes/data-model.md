# Data Model: Projects and Repos

**Feature**: 011-repo-nodes | **Date**: 2026-01-24

## Entities

### Project

Represents a logical unit of research work (e.g., a paper being written, a software tool).

```go
// internal/project/project.go
type Project struct {
    ID          string `json:"id"`                    // Required: lowercase alphanumeric + hyphens/underscores
    Name        string `json:"name"`                  // Required: human-readable display name
    Description string `json:"description,omitempty"` // Optional
    CreatedAt   string `json:"created_at,omitempty"`  // RFC3339, auto-set on create
    UpdatedAt   string `json:"updated_at,omitempty"`  // RFC3339, auto-set on update
}
```

**Validation Rules**:
- `ID`: non-empty, matches `^[a-z0-9][a-z0-9_-]*$`
- `ID`: globally unique across papers, concepts, projects (fail on collision)
- `Name`: non-empty

**Storage**: `.bipartite/projects.jsonl`

### Repo

Represents a GitHub repository belonging to a project.

```go
// internal/repo/repo.go
type Repo struct {
    ID          string   `json:"id"`                    // Required: unique identifier
    Project     string   `json:"project"`               // Required: project ID this repo belongs to
    Type        string   `json:"type"`                  // Required: "github" or "manual"
    Name        string   `json:"name"`                  // Required: display name
    GitHubURL   string   `json:"github_url,omitempty"`  // Required if type=github
    Description string   `json:"description,omitempty"` // From GitHub or user-provided
    Topics      []string `json:"topics,omitempty"`      // From GitHub or user-provided
    Language    string   `json:"language,omitempty"`    // From GitHub
    CreatedAt   string   `json:"created_at,omitempty"`  // RFC3339, auto-set
    UpdatedAt   string   `json:"updated_at,omitempty"`  // RFC3339, auto-set
}
```

**Validation Rules**:
- `ID`: non-empty, matches `^[a-z0-9][a-z0-9_-]*$`
- `Project`: must reference existing project
- `Type`: must be "github" or "manual"
- `GitHubURL`: required if type="github", must be valid GitHub URL
- `GitHubURL`: globally unique (one-project-per-repo constraint)
- `Name`: non-empty

**Storage**: `.bipartite/repos.jsonl`

### Edge (Extended)

Existing edge schema, extended to support concept↔project edges.

```go
// internal/edge/edge.go (existing, extended validation)
type Edge struct {
    SourceID         string `json:"source_id"`          // paper ID, concept:id, or project:id
    TargetID         string `json:"target_id"`          // paper ID, concept:id, or project:id
    RelationshipType string `json:"relationship_type"`
    Summary          string `json:"summary"`
    CreatedAt        string `json:"created_at,omitempty"`
}
```

**New Relationship Types** (concept↔project):
- `implemented-in`: Concept is implemented in this project
- `applied-in`: Concept is applied/used in this project
- `studied-by`: Concept is studied/investigated by this project
- `introduces`: Project introduces or defines this concept
- `refines`: Project refines understanding of this concept

**Validation Rules** (new):
- Reject edges where source or target has `repo:` prefix
- Reject edges between unprefixed paper ID and `project:` prefix
- Accept edges between `concept:` and `project:` prefixes (either direction)

## Relationships

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Knowledge Graph                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌─────────┐         ┌──────────┐         ┌──────────┐             │
│   │  Papers │ ←─────→ │ Concepts │ ←─────→ │ Projects │             │
│   │ (DOI)   │  edges  │(concept:)│  edges  │(project:)│             │
│   └─────────┘         └──────────┘         └──────────┘             │
│                                                  │                   │
│                                           ┌──────┴──────┐            │
│                                           │    Repos    │            │
│                                           │  (repo:)    │            │
│                                           │ (no edges)  │            │
│                                           └─────────────┘            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

Legend:
  ←─────→  Edges (stored in edges.jsonl)
  ────────  Containment (repo.project field references project.id)
```

### Cardinality

| Relationship | Cardinality | Notes |
|--------------|-------------|-------|
| Project → Repo | 1:N | A project can have many repos |
| Repo → Project | N:1 | A repo belongs to exactly one project |
| Project ↔ Concept | N:M | Many-to-many via edges |
| Concept ↔ Paper | N:M | Many-to-many via edges (existing) |
| Paper ↔ Paper | N:M | Many-to-many via edges (existing) |

## SQLite Schema Extensions

### New Tables

```sql
-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT,
    updated_at TEXT
);

-- Repos table
CREATE TABLE IF NOT EXISTS repos (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL REFERENCES projects(id),
    type TEXT NOT NULL CHECK (type IN ('github', 'manual')),
    name TEXT NOT NULL,
    github_url TEXT,
    description TEXT,
    topics TEXT,      -- JSON array
    language TEXT,
    created_at TEXT,
    updated_at TEXT,
    UNIQUE(github_url)  -- Enforce one-project-per-repo
);

CREATE INDEX IF NOT EXISTS idx_repos_project ON repos(project);
CREATE INDEX IF NOT EXISTS idx_repos_github_url ON repos(github_url);
```

### Existing Table (edges) - No Schema Change

Edge validation happens in application code, not SQL constraints.

## State Transitions

### Project Lifecycle

```
[create] → ACTIVE → [delete] → DELETED
              │
              └─→ [update] → ACTIVE
```

- **Create**: Validates ID uniqueness, sets timestamps
- **Update**: Updates fields, sets `updated_at`
- **Delete**: Cascade-deletes repos and edges involving this project

### Repo Lifecycle

```
[add] → ACTIVE → [delete] → DELETED
           │
           ├─→ [update] → ACTIVE
           └─→ [refresh] → ACTIVE (updated metadata)
```

- **Add**: Validates project exists, fetches GitHub metadata (if type=github)
- **Refresh**: Re-fetches GitHub metadata, updates fields
- **Delete**: Removes repo only (no cascade)

## ID Format Examples

| Entity | ID Format | Example |
|--------|-----------|---------|
| Paper | DOI or S2 ID (unprefixed) | `10.1038/s41586-021-03819-2` |
| Concept | `concept:` + slug | `concept:variational-inference` |
| Project | `project:` + slug | `project:dasm2` |
| Repo | `repo:` + slug | `repo:dasm2-code` |

**Note**: In JSONL storage, projects and repos store the slug without prefix (e.g., `"id": "dasm2"`). The prefix is added when referencing in edges.
