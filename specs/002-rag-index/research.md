# Research: RAG Index for Semantic Search

**Feature**: 002-rag-index
**Date**: 2026-01-12

## Technology Decisions

### 1. Embedding Generation

**Decision**: Use Ollama with `all-MiniLM-L6-v2` model (local, offline-first)

**Rationale**:
- **Offline-first**: After initial model download, runs entirely locally
- **Zero cost**: No API fees for embedding 6,000+ papers
- **Privacy**: Paper abstracts stay on user's machine
- **CLI philosophy alignment**: No network dependency during normal operation
- **Quality sufficient**: 84% on semantic similarity benchmarks (adequate for paper discovery)
- **Fast**: ~100-200 abstracts/second on modern CPU

**Model Details**:
- Model: `all-minilm:l6-v2` (also known as `all-MiniLM-L6-v2`)
- Dimensions: 384 (compact, efficient for storage and search)
- Size: ~22MB on disk
- One-time setup: `ollama pull all-minilm:l6-v2`

**Alternatives Considered**:
- **OpenAI text-embedding-3-small**: Higher quality (93%) but requires API key, has cost (~$0.02 per 6000 papers), and network dependency
- **Voyage AI**: Similar to OpenAI, restrictive free tier (3 req/min)
- **nomic-embed-text**: Good quality (71%) but larger model (600MB)
- **ONNX Runtime local**: Complex setup, CGO dependencies, Ollama is simpler

**Implementation**:
```go
// HTTP API to Ollama (localhost:11434)
POST /api/embeddings
{"model": "all-minilm:l6-v2", "prompt": "abstract text..."}
Response: {"embedding": [0.1, 0.2, ...]}
```

**Error Handling**:
- Check Ollama availability before index build: `GET /api/tags`
- Clear error if Ollama not running: "Ollama is not running. Start it with 'ollama serve' or install from ollama.ai"
- Timeout per abstract: 30 seconds (fail fast, don't hang)

Sources:
- [Ollama Embedding Docs](https://docs.ollama.com/capabilities/embeddings)
- [all-MiniLM-L6-v2 on HuggingFace](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2)

---

### 2. Vector Storage

**Decision**: Pure Go brute-force cosine similarity with GOB-serialized index file

**Rationale**:
- **CGO-free**: Maintains compatibility with existing modernc.org/sqlite (pure Go)
- **Simple**: No external dependencies, easy to understand and debug
- **Fast enough**: ~4ms query latency for 10k vectors at 384 dimensions (well under 1s requirement)
- **Rebuildable**: Simple binary format, trivial to drop and recreate from JSONL

**Critical Constraint**: sqlite-vec does NOT work with modernc.org/sqlite. The [sqlite-vec Go bindings](https://alexgarcia.xyz/sqlite-vec/go.html) only support:
1. mattn/go-sqlite3 (CGO-based)
2. ncruces/go-sqlite3 (WASM-based)

Switching SQLite drivers would require migrating the entire Phase I codebase, which is out of scope.

**Storage Format**:
```go
// SemanticIndex is the persisted index structure
type SemanticIndex struct {
    ModelName   string              // e.g., "all-minilm"
    Dimensions  int                 // 384
    CreatedAt   time.Time
    Embeddings  map[string][]float32 // paper_id -> vector
}

// Persisted as GOB to .bipartite/cache/semantic.gob
```

**Metadata Table** (in existing refs.db):
```sql
-- Track embedding metadata for staleness detection
CREATE TABLE IF NOT EXISTS embedding_metadata (
    paper_id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    indexed_at INTEGER NOT NULL,
    abstract_hash TEXT NOT NULL
);
```

**Storage Location**: `.bipartite/cache/semantic.gob` (gitignored, ephemeral)

**Why Not sqlite-vec**:
- Requires CGO (mattn/go-sqlite3) or driver switch (ncruces/go-sqlite3)
- Would break pure-Go build constraint from Phase I
- Overkill for 10k vectors where brute-force is sub-5ms

**Why Not HNSW (approximate nearest neighbor)**:
- Only beneficial at 100k+ vectors
- Adds complexity for marginal gain at our scale
- Pure Go implementations (coder/hnsw) still need persistence layer

**Performance Verification**:

| Vectors | Dimensions | Brute-Force Query |
|---------|------------|-------------------|
| 10,000 | 384 | ~4ms |
| 10,000 | 768 | ~8ms |
| 10,000 | 1536 | ~15ms |

These times are well within the <1s requirement (SC-001).

**Go Implementation**:
```go
// CosineSimilarity computes similarity between two vectors
func CosineSimilarity(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Search finds top-k similar papers
func (idx *SemanticIndex) Search(query []float32, limit int, threshold float32) []SearchResult {
    results := make([]SearchResult, 0, len(idx.Embeddings))
    for paperID, embedding := range idx.Embeddings {
        sim := CosineSimilarity(query, embedding)
        if sim >= threshold {
            results = append(results, SearchResult{PaperID: paperID, Similarity: sim})
        }
    }
    sort.Slice(results, func(i, j int) bool {
        return results[i].Similarity > results[j].Similarity
    })
    if len(results) > limit {
        results = results[:limit]
    }
    return results
}
```

Sources:
- [sqlite-vec Go bindings limitations](https://alexgarcia.xyz/sqlite-vec/go.html)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)

---

### 3. Similarity Search Algorithm

**Decision**: Cosine similarity computed in pure Go

**Rationale**:
- Standard metric for text embeddings (normalized dot product)
- Simple to implement and understand
- Returns similarity directly (higher = more similar), avoiding distance conversion confusion

**Implementation**: See vector storage section above for `CosineSimilarity` function.

**Threshold Semantics**:
- `--threshold` flag specifies **minimum similarity** (not distance)
- Higher threshold = stricter matching (fewer, more relevant results)
- Range: 0.0 (accept all) to 1.0 (exact match only)

**Recommended Thresholds**:
| Threshold | Meaning | Use Case |
|-----------|---------|----------|
| 0.3 | Loose match | Exploratory search, broad discovery |
| 0.5 | Moderate match | Default, balanced precision/recall |
| 0.7 | Strict match | High-precision queries |

**Default Threshold**: 0.5 (configurable via `--threshold` flag)

---

### 4. Index Rebuild Strategy

**Decision**: Full rebuild on `bp index build`, no incremental updates

**Rationale**:
- **Simplicity**: No tracking of which papers changed
- **Correctness**: Guarantees index matches current JSONL state
- **Acceptable time**: <5 minutes for 6000 papers (per spec SC-003)
- **Matches Phase I pattern**: `bp rebuild` does full SQLite rebuild

**Workflow**:
1. Drop existing semantic.db (if exists)
2. Read all papers from refs.jsonl
3. For each paper with abstract:
   - Generate embedding via Ollama
   - Store in semantic.db
4. Report statistics

**Future Consideration**: If rebuild time becomes problematic (>10 minutes), add incremental mode that:
- Hashes abstracts, only re-embeds changed ones
- This is premature optimization for Phase II

---

### 5. CLI Command Structure

**Decision**: Use subcommand pattern: `bp index build`, `bp index check`

**Rationale**:
- Groups index management under single parent command
- Consistent with standard CLI patterns (git, docker)
- Leaves room for future index operations (`bp index stats`, `bp index prune`)

**Commands**:
```bash
# Index management
bp index build              # Build/rebuild semantic index
bp index build --progress   # Show progress during build
bp index check              # Check index health

# Search commands
bp semantic <query>         # Semantic search
bp semantic <query> --limit 5 --threshold 0.3
bp similar <paper-id>       # Find similar papers
bp similar <paper-id> --limit 10
```

**Alternative Considered**:
- Flat commands: `bp build-index`, `bp check-index` - less organized, clutters help output

---

### 6. Progress Reporting

**Decision**: Streaming progress to stderr, final stats to stdout

**Rationale**:
- Allows piping JSON output while showing progress
- Matches standard Unix conventions
- Users see feedback during long operations

**Implementation**:
```
$ bp index build
Indexing papers...
[=====>                    ] 1234/6000 papers (20%)

Build complete:
- Papers indexed: 5889
- Papers skipped (no abstract): 346
- Time elapsed: 2m 34s
- Index size: 12.3 MB
```

**JSON output** (`--json` flag): Progress suppressed, only final stats as JSON

---

### 7. Error Handling Patterns

**Decision**: Fail fast with actionable messages

**Patterns**:

| Scenario | Error Message |
|----------|---------------|
| Ollama not running | "Error: Ollama is not running. Start it with 'ollama serve' or install from ollama.ai" |
| Model not pulled | "Error: Model 'all-minilm:l6-v2' not found. Run 'ollama pull all-minilm:l6-v2' first" |
| Index not built | "Error: Semantic index not found. Run 'bp index build' first" |
| Paper not found | "Error: Paper 'XYZ-123' not found in database" |
| Paper has no abstract | "Error: Paper 'XYZ-123' has no abstract and cannot be used for similarity search" |
| Empty query | "Error: Search query cannot be empty" |

**Exit Codes**:
- 0: Success
- 1: General error
- 2: Index not found (hint: run `bp index build`)
- 3: Ollama not available

---

### 8. Testing Strategy

**Decision**: Use real abstracts from fixture papers, mock Ollama for unit tests

**Test Fixtures** (in `testdata/abstracts/`):
1. `phylogenetics_papers.json` - Papers about phylogenetics (should cluster)
2. `ml_papers.json` - Papers about machine learning (should cluster)
3. `mixed_papers.json` - Papers from different domains (should not cluster)

**Test Types**:
- **Unit tests**: Mock Ollama HTTP responses, test embedding storage/retrieval
- **Integration tests**: Real Ollama (if available), test end-to-end search quality
- **Contract tests**: Verify CLI output format matches contracts

**Embedding Mock**:
```go
// For unit tests: deterministic fake embeddings based on content hash
func mockEmbedding(text string) []float32 {
    // Hash text to produce consistent fake vector
    // Different texts produce different vectors
    // Similar texts produce similar vectors (via hash similarity)
}
```

---

## Summary

| Decision | Choice | Key Reason |
|----------|--------|------------|
| Embedding Model | Ollama + all-minilm:l6-v2 | Offline-first, zero cost, sufficient quality |
| Vector Storage | Pure Go + GOB file | CGO-free, compatible with modernc.org/sqlite |
| Similarity | Cosine (pure Go) | Standard for text, simple implementation |
| Rebuild | Full rebuild | Simplicity, correctness, acceptable time |
| CLI Structure | Subcommands (bp index ...) | Organized, extensible |
| Progress | Stderr streaming | Unix convention, pipeable |
| Errors | Fail fast + actionable | Constitution compliance |
| Testing | Real fixtures + Ollama mock | Agentic TDD pattern |

## Dependencies Added

```
# New external dependencies
ollama (user-installed, not Go dependency)

# New Go dependencies
(none - pure Go implementation using standard library encoding/gob)
```

**Note**: sqlite-vec was originally considered but is incompatible with modernc.org/sqlite (our existing driver). The pure Go brute-force approach is simpler and fast enough for our scale (10k vectors).
