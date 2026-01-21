# Data Model: Knowledge Graph

**Feature**: 003-knowledge-graph
**Date**: 2026-01-13

## Entities

### Edge

A directed relationship between two papers with a relationship type and relational summary.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| source_id | string | yes | ID of the source paper (must exist in refs.jsonl) |
| target_id | string | yes | ID of the target paper (must exist in refs.jsonl) |
| relationship_type | string | yes | Type of relationship (e.g., "cites", "extends", "contradicts") |
| summary | string | yes | Prose describing the relationship from source's perspective toward target |
| created_at | string | no | ISO 8601 timestamp when edge was created |

**Identity**: (source_id, target_id, relationship_type) tuple
**Uniqueness**: No duplicate edges with same identity; updates replace existing summary

### Predefined Relationship Types

These are suggestions, not enforced constraints. Any string is valid.

| Type | Meaning |
|------|---------|
| cites | Source paper references target in bibliography |
| extends | Source builds upon or extends work in target |
| contradicts | Source presents findings that contradict target |
| implements | Source implements methods/algorithms from target |
| applies-to | Source applies methods from target to a new domain |
| builds-on | Source uses target as foundational work |

## JSONL Schema

**File**: `.bipartite/edges.jsonl`

Each line is a JSON object representing one edge:

```json
{"source_id":"Smith2024-ab","target_id":"Jones2023-xy","relationship_type":"extends","summary":"Extends Jones's variational framework to handle non-Euclidean geometries","created_at":"2026-01-13T10:30:00Z"}
```

### Validation Rules

1. **Source paper exists**: `source_id` must match an `id` in refs.jsonl at edge creation time
2. **Target paper exists**: `target_id` must match an `id` in refs.jsonl at edge creation time
3. **Non-empty summary**: `summary` must be non-empty string
4. **Non-empty type**: `relationship_type` must be non-empty string
5. **No self-edges**: `source_id` != `target_id`

### Orphaned Edges

When a paper is deleted from refs.jsonl:
- Edges referencing that paper remain in edges.jsonl
- `bip groom` detects and reports orphaned edges
- User decides whether to remove orphaned edges

## SQLite Index Schema

**File**: `.bipartite/cache/edges.db` (ephemeral, gitignored)

```sql
CREATE TABLE edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    summary TEXT NOT NULL,
    created_at TEXT,
    PRIMARY KEY (source_id, target_id, relationship_type)
);

CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);
CREATE INDEX idx_edges_type ON edges(relationship_type);
```

Rebuilt from edges.jsonl on `bip rebuild`.

## State Transitions

### Edge Lifecycle

```
[Created] --> [Active] --> [Orphaned]
                 |
                 v
            [Updated]
```

| State | Description | Trigger |
|-------|-------------|---------|
| Created | New edge added | `bip edge add` or `bip edge import` |
| Active | Edge references valid papers | Normal state |
| Updated | Summary changed | `bip edge add` with existing (source, target, type) |
| Orphaned | Source or target paper deleted | Paper removal from refs.jsonl |

## Relationships

```
┌─────────────┐       ┌──────────────┐       ┌─────────────┐
│   Paper A   │──────▶│     Edge     │──────▶│   Paper B   │
│ (source)    │       │              │       │ (target)    │
└─────────────┘       │ type: cites  │       └─────────────┘
                      │ summary: ... │
                      └──────────────┘
```

- One paper can have many outgoing edges (as source)
- One paper can have many incoming edges (as target)
- Multiple edges between same paper pair allowed (different relationship types)
- No edge can reference a paper that doesn't exist (at creation time)
