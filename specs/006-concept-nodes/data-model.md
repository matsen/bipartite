# Data Model: Concept Nodes

**Feature Branch**: `006-concept-nodes`
**Date**: 2026-01-21

## Entities

### Concept

A named idea, method, or phenomenon that papers can relate to.

```go
// internal/concept/concept.go
type Concept struct {
    ID          string   `json:"id"`          // Required, unique, lowercase alphanumeric + hyphens/underscores
    Name        string   `json:"name"`        // Required, human-readable display name
    Aliases     []string `json:"aliases"`     // Optional, alternative names (e.g., ["SHM"] for somatic-hypermutation)
    Description string   `json:"description"` // Optional, longer explanation
}
```

**Validation Rules**:
- `ID`: Required, non-empty, matches `^[a-z0-9][a-z0-9_-]*$`
- `Name`: Required, non-empty
- `Aliases`: Optional, may be empty slice or omitted
- `Description`: Optional, may be empty string or omitted

**JSONL Example** (`concepts.jsonl`):
```json
{"id": "somatic-hypermutation", "name": "Somatic Hypermutation", "aliases": ["SHM"], "description": "Process by which B cells diversify antibody genes through point mutations"}
{"id": "variational-inference", "name": "Variational Inference", "aliases": ["VI"], "description": "Approximation method for Bayesian inference"}
{"id": "phylogenetics", "name": "Phylogenetics", "aliases": [], "description": "Study of evolutionary relationships among biological entities"}
```

---

### Paper-Concept Edge

A directed relationship from a paper to a concept. Uses existing `edge.Edge` type — no new struct needed.

```go
// internal/edge/edge.go (existing)
type Edge struct {
    SourceID         string `json:"source_id"`         // Paper ID (e.g., "Matsen2025-oj")
    TargetID         string `json:"target_id"`         // Concept ID (e.g., "somatic-hypermutation")
    RelationshipType string `json:"relationship_type"` // e.g., "introduces", "applies", "models"
    Summary          string `json:"summary"`           // Required explanation
    CreatedAt        string `json:"created_at"`        // RFC3339 timestamp
}
```

**Validation Rules**:
- `SourceID`: Must exist in refs.jsonl
- `TargetID`: Must exist in concepts.jsonl (for paper-concept edges)
- `RelationshipType`: Required, non-empty. Warn if not in `relationship-types.json` paper-concept list
- `Summary`: Required, non-empty

**Standard Relationship Types** (paper-concept, from `relationship-types.json`):
| Type | Description |
|------|-------------|
| `introduces` | Paper first presents this concept |
| `applies` | Paper uses concept as a tool/method |
| `models` | Paper creates computational/mathematical model of this phenomenon |
| `evaluates-with` | Paper uses this for evaluation/benchmarking |
| `critiques` | Paper identifies limitations of concept |
| `extends` | Paper builds upon concept |

**JSONL Example** (stored in existing `edges.jsonl`):
```json
{"source_id": "Matsen2025-oj", "target_id": "variational-inference", "relationship_type": "applies", "summary": "Uses VI for posterior approximation in phylogenetic models", "created_at": "2026-01-21T10:00:00Z"}
{"source_id": "Halpern1998-yc", "target_id": "somatic-hypermutation", "relationship_type": "introduces", "summary": "Foundational paper describing somatic hypermutation mechanism", "created_at": "2026-01-21T10:05:00Z"}
```

---

## SQLite Schema

### concepts table

```sql
CREATE TABLE IF NOT EXISTS concepts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    aliases_json TEXT,  -- JSON array as string, e.g., '["SHM","shm"]'
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_concepts_name ON concepts(name);
```

### concepts_fts (Full-Text Search)

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS concepts_fts USING fts5(
    id,
    name,
    aliases_text,   -- Space-joined aliases for searching
    description
);
```

**Rebuild Process**:
1. Read concepts.jsonl
2. Delete all from `concepts` table
3. Delete all from `concepts_fts` table
4. Insert into `concepts` table
5. Insert into `concepts_fts` table (with `aliases_text = strings.Join(aliases, " ")`)

### edges table (existing, no changes)

The existing edges table already supports paper-concept edges:
```sql
CREATE TABLE IF NOT EXISTS edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    summary TEXT NOT NULL,
    created_at TEXT,
    PRIMARY KEY (source_id, target_id, relationship_type)
);
```

---

## State Transitions

### Concept Lifecycle

```
[Not Exists] --add--> [Active] --delete--> [Deleted]
                         |
                         +--update--> [Active]
                         |
                         +--merge(into)--> [Deleted] (edges transferred)
```

### Paper-Concept Edge Lifecycle

```
[Not Exists] --add--> [Active] --delete--> [Deleted]
                         |
                         +--update--> [Active]
```

---

## Relationships Diagram

```
+----------------+          +------------------+
|     Paper      |          |     Concept      |
|----------------|          |------------------|
| id (PK)        |          | id (PK)          |
| doi            |    N:M   | name             |
| title          |<-------->| aliases          |
| abstract       |   via    | description      |
| ...            |  edges   |                  |
+----------------+          +------------------+
         |                           ^
         |                           |
         v                           |
+-------------------------------------------+
|                 Edge                       |
|-------------------------------------------|
| source_id (FK->Paper) + target_id + type  |
| target_id (FK->Paper OR Concept)          |
| relationship_type                          |
| summary                                    |
| created_at                                 |
+-------------------------------------------+
```

**Key Points**:
- Papers have many-to-many relationship with Concepts via edges
- Same edge table stores both paper-paper and paper-concept edges
- Target type determined by runtime lookup (ID exists in refs vs concepts)

---

## ID Namespace

| Entity | ID Pattern | Examples |
|--------|------------|----------|
| Paper | `Author####-xx` (Paperpile format) | `Matsen2025-oj`, `Halpern1998-yc` |
| Concept | `^[a-z0-9][a-z0-9_-]*$` (lowercase) | `somatic-hypermutation`, `bcr_sequencing` |

**Collision Risk**: Low — Paperpile IDs contain uppercase and specific year/suffix patterns that don't match concept ID pattern.

---

## File Locations

| Data | File Path |
|------|-----------|
| Concepts (source of truth) | `.bipartite/concepts.jsonl` |
| Edges (both types) | `.bipartite/edges.jsonl` |
| SQLite index | `.bipartite/cache/refs.db` |

---

## Query Patterns

### Get papers by concept
```sql
SELECT e.source_id, e.relationship_type, e.summary
FROM edges e
WHERE e.target_id = ?
ORDER BY e.relationship_type, e.source_id
```

### Get concepts by paper
```sql
SELECT e.target_id, e.relationship_type, e.summary
FROM edges e
WHERE e.source_id = ?
  AND e.target_id IN (SELECT id FROM concepts)
ORDER BY e.relationship_type, e.target_id
```

### Search concepts by text
```sql
SELECT c.id, c.name, c.aliases_json, c.description
FROM concepts_fts fts
JOIN concepts c ON fts.id = c.id
WHERE concepts_fts MATCH ?
ORDER BY rank
```

### Count edges for concept (for delete warning)
```sql
SELECT COUNT(*) FROM edges WHERE target_id = ?
```
