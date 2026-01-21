# Data Model: Knowledge Graph Visualization

**Feature**: 007-knowledge-graph-viz
**Date**: 2026-01-21

## Overview

This feature does not introduce new persistent data. It reads from existing data sources (SQLite rebuilt from JSONL) and produces ephemeral HTML output.

## Input Data Sources

### 1. References (existing)

**Source**: `refs` table in SQLite (from `refs.jsonl`)

**Fields Used**:
| Field | Type | Used For |
|-------|------|----------|
| id | string | Node ID |
| title | string | Tooltip display |
| authors | []Author | Tooltip display |
| published.year | int | Tooltip display |

### 2. Concepts (existing)

**Source**: `concepts` table in SQLite (from `concepts.jsonl`)

**Fields Used**:
| Field | Type | Used For |
|-------|------|----------|
| id | string | Node ID |
| name | string | Node label, tooltip |
| aliases | []string | Tooltip display |
| description | string | Tooltip display |

### 3. Edges (existing)

**Source**: `edges` table in SQLite (from `edges.jsonl`)

**Fields Used**:
| Field | Type | Used For |
|-------|------|----------|
| source_id | string | Edge source (paper ID) |
| target_id | string | Edge target (concept ID) |
| relationship_type | string | Edge color, tooltip |
| summary | string | Tooltip display |

## Output Data Structures

### GraphData (internal/viz)

The in-memory representation extracted from SQLite for rendering.

```go
// GraphData contains all data needed to render the visualization.
type GraphData struct {
    Nodes []Node
    Edges []Edge
}

// Node represents a paper or concept in the graph.
type Node struct {
    ID          string   `json:"id"`
    Type        string   `json:"type"`        // "paper" or "concept"
    Label       string   `json:"label"`       // Display label
    // Tooltip data (paper-specific)
    Title       string   `json:"title,omitempty"`
    Authors     string   `json:"authors,omitempty"`  // Formatted string
    Year        int      `json:"year,omitempty"`
    // Tooltip data (concept-specific)
    Name        string   `json:"name,omitempty"`
    Aliases     []string `json:"aliases,omitempty"`
    Description string   `json:"description,omitempty"`
    // Sizing
    ConnectionCount int  `json:"connectionCount"` // For concept node sizing
}

// Edge represents a paper-concept relationship.
type Edge struct {
    Source           string `json:"source"`
    Target           string `json:"target"`
    RelationshipType string `json:"relationshipType"`
    Summary          string `json:"summary"`
}
```

### CytoscapeElements (JSON output)

The Cytoscape.js-compatible format embedded in HTML.

```json
{
  "nodes": [
    {
      "data": {
        "id": "Halpern1998-yc",
        "type": "paper",
        "label": "Halpern1998-yc",
        "title": "Evolutionary distances for protein-coding sequences...",
        "authors": "Halpern AL, Bruno WJ",
        "year": 1998
      }
    },
    {
      "data": {
        "id": "mutation-selection-model",
        "type": "concept",
        "label": "Mutation-Selection Model",
        "name": "Mutation-Selection Model",
        "aliases": ["MutSel", "mutation-selection balance"],
        "description": "A model combining mutation rates with selection...",
        "connectionCount": 5
      }
    }
  ],
  "edges": [
    {
      "data": {
        "source": "Halpern1998-yc",
        "target": "mutation-selection-model",
        "relationshipType": "introduces",
        "summary": "Introduces the mutation-selection model for..."
      }
    }
  ]
}
```

## Data Extraction Query

### Papers with Concept Edges

Only papers that have edges to concepts are included (not all papers).

```sql
SELECT DISTINCT r.id, r.title, r.authors_json, r.pub_year
FROM refs r
INNER JOIN edges e ON r.id = e.source_id
INNER JOIN concepts c ON e.target_id = c.id
```

### Concepts

All concepts are included, with connection counts.

```sql
SELECT c.id, c.name, c.aliases_json, c.description,
       (SELECT COUNT(*) FROM edges WHERE target_id = c.id) as connection_count
FROM concepts c
```

### Edges (Paper to Concept only)

Only edges where target is a concept.

```sql
SELECT e.source_id, e.target_id, e.relationship_type, e.summary
FROM edges e
WHERE e.target_id IN (SELECT id FROM concepts)
```

## Validation Rules

1. **Node ID uniqueness**: Paper IDs and concept IDs must not collide (enforced by source data)
2. **Edge endpoints exist**: All edge source_ids must exist in refs, all target_ids in concepts
3. **Empty graph**: Valid state - render empty state message instead of graph

## State Transitions

N/A - This feature produces stateless output. Each `bip viz` invocation reads current database state and generates HTML.
