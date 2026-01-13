# Research: Knowledge Graph

**Feature**: 003-knowledge-graph
**Date**: 2026-01-13

## Overview

This document captures research decisions for the Knowledge Graph feature. Since the feature extends existing bipartite patterns with no new dependencies, research focuses on design decisions rather than technology evaluation.

## Decision 1: Edge Storage Format

**Decision**: Store edges in `edges.jsonl` alongside existing `refs.jsonl`

**Rationale**:
- Follows established pattern from Phase I (refs.jsonl)
- JSONL is append-friendly for git merges
- Human-readable for debugging
- Can be processed line-by-line for large files

**Alternatives Considered**:
- Single combined file (refs + edges): Rejected - different entity types, harder to reason about
- Separate SQLite database: Rejected - violates git-versionable principle
- Graph-specific format (GraphML, etc.): Rejected - adds complexity, less human-readable

## Decision 2: Edge Identity

**Decision**: Edge uniqueness defined by (source_id, target_id, relationship_type) tuple

**Rationale**:
- Allows multiple relationship types between same paper pair (A cites B, A extends B)
- Prevents duplicate edges of same type
- Simple composite key, easy to implement

**Alternatives Considered**:
- UUID per edge: Rejected - makes deduplication harder, no semantic meaning
- (source, target) only: Rejected - too restrictive, can't model multiple relationships
- Include summary in identity: Rejected - summaries should be updatable without creating new edge

## Decision 3: SQLite Index Schema

**Decision**: Create `edges` table with indexes on source_id, target_id, and relationship_type

**Rationale**:
- Supports all query patterns: list by paper, search by type
- Ephemeral index rebuilt from JSONL on `bp rebuild`
- Consistent with existing refs SQLite pattern

**Schema**:
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

## Decision 4: Orphaned Edge Handling

**Decision**: Preserve edges when referenced papers are deleted; `bp groom` flags orphans

**Rationale**:
- Edges represent valuable relationship information
- Cascade deletion could lose data that's hard to recreate
- `bp groom` already exists for maintenance tasks
- Consistent with git-mergeable philosophy (don't auto-delete)

**Alternatives Considered**:
- Cascade delete: Rejected - too destructive, violates data preservation
- Prevent paper deletion: Rejected - too restrictive, paper cleanup is valid
- Immediate warning on query: Rejected - noisy, groom is better

## Decision 5: CLI Command Structure

**Decision**: `bp edge <subcommand>` pattern with add, import, list, search, export

**Rationale**:
- Groups related functionality under single parent command
- Consistent with Unix conventions (git, docker patterns)
- Clear separation from existing paper commands
- Allows future expansion (delete, update, etc.)

**Alternatives Considered**:
- Top-level commands (`bp edge-add`): Rejected - clutters help, poor discoverability
- Combined with paper commands: Rejected - edges are distinct entity type

## Decision 6: No New Dependencies

**Decision**: Implement using existing Go stdlib + modernc.org/sqlite

**Rationale**:
- Constitution principle VI (Simplicity) requires minimal dependencies
- Existing stack is sufficient for all requirements
- No graph-specific operations that would benefit from a graph library
- Queries are simple (by source, by target, by type)

**Alternatives Considered**:
- Graph library (e.g., gonum/graph): Rejected - overkill for edge storage/query
- Separate graph database: Rejected - violates embeddable constraint

## Open Questions (None)

All technical questions resolved. Design follows existing patterns with clear decisions.
