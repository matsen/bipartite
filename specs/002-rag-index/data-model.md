# Data Model: RAG Index for Semantic Search

**Feature**: 002-rag-index
**Date**: 2026-01-12

## Overview

Phase II adds semantic search via vector embeddings. This extends the existing Phase I data model without modifying the source-of-truth (refs.jsonl). All semantic index data is ephemeral and stored in a gitignored cache.

## Entities

### Embedding

A vector representation of a paper's abstract.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| paper_id | string | Reference to paper in refs.jsonl | PRIMARY KEY, NOT NULL |
| vector | float32[384] | Embedding vector | NOT NULL, fixed dimension |
| model_name | string | Embedding model identifier | NOT NULL |
| indexed_at | timestamp | When embedding was created | NOT NULL |
| abstract_hash | string | SHA256 of abstract text | NOT NULL |

**Relationships**:
- `paper_id` → `Reference.id` (from Phase I)

**Notes**:
- `abstract_hash` enables detecting when abstracts change (future incremental rebuild)
- `model_name` tracks which model produced the embedding (important for reproducibility)
- Vector dimension (384) is determined by the embedding model

### SemanticIndex (Collection)

Metadata about the semantic index as a whole.

| Field | Type | Description |
|-------|------|-------------|
| model_name | string | Embedding model used for all vectors |
| model_dimensions | int | Vector dimension (384 for all-MiniLM) |
| created_at | timestamp | When index was built |
| paper_count | int | Number of papers indexed |
| skipped_count | int | Papers skipped (no abstract) |
| build_duration_ms | int | Time taken to build |

**Storage**: Embedded in the GOB-serialized SemanticIndex struct.

## Storage Schema

### GOB Index File (semantic.gob)

The vector index is stored as a GOB-encoded Go struct:

```go
// SemanticIndex is persisted to .bipartite/cache/semantic.gob
type SemanticIndex struct {
    ModelName      string               `json:"model_name"`      // e.g., "all-minilm:l6-v2"
    Dimensions     int                  `json:"dimensions"`      // 384
    CreatedAt      time.Time            `json:"created_at"`
    PaperCount     int                  `json:"paper_count"`
    SkippedCount   int                  `json:"skipped_count"`
    BuildDurationMs int64               `json:"build_duration_ms"`
    Embeddings     map[string][]float32 `json:"-"`               // paper_id -> vector (not in JSON output)
}
```

### Metadata Table (in refs.db)

Embedding metadata is stored in the existing refs.db for staleness detection:

```sql
-- Track embedding metadata (added to existing refs.db schema)
CREATE TABLE IF NOT EXISTS embedding_metadata (
    paper_id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    indexed_at INTEGER NOT NULL,
    abstract_hash TEXT NOT NULL
);
```

### File Structure

```
.bipartite/
├── refs.jsonl           # Source of truth (Phase I, unchanged)
├── config.json          # Configuration (Phase I, unchanged)
└── cache/
    ├── refs.db          # Query cache (Phase I) + embedding_metadata table
    └── semantic.gob     # NEW: Vector index (Phase II, GOB-encoded)
```

**Why GOB over JSON/SQLite**: GOB is Go's native binary format—compact, fast to serialize/deserialize, and handles `[]float32` efficiently. For 6000 papers at 384 dimensions, the index is ~9MB (vs ~50MB for JSON).

## Data Flow

### Index Build

```
refs.jsonl → Read papers with abstracts
          → Generate embeddings (Ollama)
          → Store in SemanticIndex struct
          → Serialize to semantic.gob
          → Write metadata to refs.db
```

### Semantic Search

```
Query text → Generate query embedding (Ollama)
          → Load SemanticIndex from semantic.gob (lazy, cached)
          → Brute-force cosine similarity search
          → Return ranked paper IDs
          → Fetch full paper data from refs.db
          → Return combined results
```

### Find Similar Papers

```
Paper ID → Load SemanticIndex from semantic.gob (lazy, cached)
        → Fetch paper embedding from index
        → Brute-force cosine similarity search
        → Return ranked paper IDs (excluding source)
        → Fetch full paper data from refs.db
        → Return combined results
```

## Validation Rules

### Embedding Generation

1. Paper MUST have non-empty abstract to be indexed
2. Abstract MUST be at least 50 characters to be indexed
   - Shorter abstracts are skipped with a warning (counted in skipped_count)
   - Rationale: Very short text produces low-quality embeddings
3. Embedding vector MUST have exactly 384 dimensions
4. Ollama MUST be running and have model pulled

### Search Operations

1. Query MUST be non-empty string
2. Query embedding MUST match index model dimensions
3. Limit MUST be positive integer (default: 10, max: 100)
4. Threshold MUST be in range [0.0, 1.0] (default: 0.5)

### Index Health

1. All papers in semantic.gob MUST exist in refs.jsonl
2. Model name MUST be consistent across all embeddings
3. Vector dimensions MUST match model specification (384 for all-minilm:l6-v2)

## State Transitions

### Semantic Index States

```
┌─────────────┐
│  Not Built  │ ← Initial state, no semantic.gob exists
└─────────────┘
       │
       │ bp index build
       ▼
┌─────────────┐
│   Building  │ ← Generating embeddings, progress shown
└─────────────┘
       │
       │ Success
       ▼
┌─────────────┐
│    Ready    │ ← Search commands work
└─────────────┘
       │
       │ refs.jsonl changed (git pull, re-import)
       ▼
┌─────────────┐
│    Stale    │ ← Index may be out of sync
└─────────────┘
       │
       │ bp index build (rebuild)
       ▼
┌─────────────┐
│    Ready    │
└─────────────┘
```

**Note**: "Stale" state is not automatically detected in Phase II. Users run `bip index check` to verify, or `bip index build` to ensure freshness.

## Relationship to Phase I

| Phase I Entity | Phase II Relationship |
|----------------|----------------------|
| Reference | Embedding.paper_id → Reference.id |
| Reference.abstract | Source text for embedding generation |
| refs.jsonl | Source of truth, read-only for Phase II |
| refs.db | Used to fetch full paper data for search results |

**Key Principle**: Phase II does not modify Phase I data. The semantic index is purely additive and ephemeral.
