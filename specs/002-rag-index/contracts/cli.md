# CLI Contract: RAG Index Commands

**Feature**: 002-rag-index
**Date**: 2026-01-12

This document specifies the CLI interface for Phase II semantic search commands. All commands follow Phase I conventions (JSON default, `--human` flag for readable output).

---

## bp semantic

Search papers by semantic similarity to a query.

### Usage

```bash
bp semantic <query> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| query | Yes | Natural language search query |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --limit | -l | 10 | Maximum number of results |
| --threshold | -t | 0.5 | Minimum similarity threshold (0.0-1.0) |
| --human | -H | false | Human-readable output |

### Output (JSON, default)

```json
{
  "query": "methods for inferring evolutionary trees",
  "results": [
    {
      "id": "Matsen2025-ab",
      "title": "Phylogenetic inference using...",
      "authors": [{"first": "Frederick", "last": "Matsen"}],
      "year": 2025,
      "similarity": 0.87,
      "abstract": "We present a method for..."
    }
  ],
  "total": 5,
  "threshold": 0.5,
  "model": "all-minilm:l6-v2"
}
```

### Output (Human, --human)

```
Search: "methods for inferring evolutionary trees"
Found 5 papers (threshold: 0.5)

1. [0.87] Matsen2025-ab
   Phylogenetic inference using...
   Matsen (2025)

2. [0.82] Suchard2024-cd
   Bayesian phylogenetic methods...
   Suchard et al. (2024)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (results returned, may be empty) |
| 1 | General error |
| 2 | Semantic index not found |
| 3 | Ollama not available |

### Examples

```bash
# Basic search
bp semantic "protein folding prediction"

# Limit results
bp semantic "MCMC sampling" --limit 5

# Higher similarity threshold
bp semantic "neural networks" --threshold 0.7

# Human-readable
bp semantic "variational inference" --human
```

---

## bp similar

Find papers similar to a specific paper.

### Usage

```bash
bp similar <paper-id> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| paper-id | Yes | ID of the source paper |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --limit | -l | 10 | Maximum number of results |
| --human | -H | false | Human-readable output |

### Output (JSON, default)

```json
{
  "source": {
    "id": "Matsen2025-ab",
    "title": "Phylogenetic inference using..."
  },
  "similar": [
    {
      "id": "Suchard2024-cd",
      "title": "Bayesian phylogenetic methods...",
      "authors": [{"first": "Marc", "last": "Suchard"}],
      "year": 2024,
      "similarity": 0.89
    }
  ],
  "total": 10,
  "model": "all-minilm:l6-v2"
}
```

### Output (Human, --human)

```
Papers similar to: Matsen2025-ab
"Phylogenetic inference using..."

1. [0.89] Suchard2024-cd
   Bayesian phylogenetic methods...
   Suchard (2024)

2. [0.85] Drummond2023-ef
   BEAST: A Bayesian framework...
   Drummond et al. (2023)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (paper not found) |
| 2 | Semantic index not found |
| 4 | Paper has no abstract |

### Examples

```bash
# Find similar papers
bp similar Matsen2025-ab

# Limit results
bp similar Matsen2025-ab --limit 5

# Human-readable
bp similar Matsen2025-ab --human
```

---

## bp index build

Build or rebuild the semantic index from paper abstracts.

### Usage

```bash
bp index build [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --no-progress | | false | Suppress progress output (for scripts) |
| --human | -H | false | Human-readable output |

### Output (JSON, default)

```json
{
  "status": "complete",
  "papers_indexed": 5889,
  "papers_skipped": 346,
  "skipped_reason": "no_abstract",
  "duration_seconds": 154.2,
  "model": "all-minilm:l6-v2",
  "index_size_bytes": 12943872
}
```

### Output (Human, --human)

```
Building semantic index...
[============================] 6235/6235 papers

Build complete:
  Papers indexed: 5889
  Papers skipped: 346 (no abstract)
  Time elapsed: 2m 34s
  Index size: 12.3 MB
  Model: all-minilm:l6-v2
```

### Progress Output (stderr)

```
Indexing papers...
[========>                    ] 2000/6235 (32%)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 3 | Ollama not available |
| 5 | Model not found (needs `ollama pull`) |

### Examples

```bash
# Build index (shows progress by default)
bp index build

# Build without progress (for scripts/CI)
bp index build --no-progress

# Human-readable output
bp index build --human
```

---

## bp index check

Check semantic index health and status.

### Usage

```bash
bp index check [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --human | -H | false | Human-readable output |

### Output (JSON, default)

```json
{
  "status": "healthy",
  "papers_total": 6235,
  "papers_with_abstract": 5889,
  "papers_indexed": 5889,
  "papers_missing": 0,
  "model": "all-minilm:l6-v2",
  "index_created": "2026-01-12T14:30:00Z",
  "index_size_bytes": 12943872
}
```

### Output (unhealthy index)

```json
{
  "status": "stale",
  "papers_total": 6300,
  "papers_with_abstract": 5950,
  "papers_indexed": 5889,
  "papers_missing": 61,
  "missing_ids": ["NewPaper2026-ab", "..."],
  "model": "all-minilm:l6-v2",
  "recommendation": "Run 'bp index build' to update the index"
}
```

### Output (Human, --human)

```
Semantic Index Status: healthy

Papers:
  Total in database: 6235
  With abstracts: 5889
  In semantic index: 5889
  Missing from index: 0

Index Info:
  Model: all-minilm:l6-v2
  Created: 2026-01-12 14:30:00
  Size: 12.3 MB
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Healthy |
| 1 | General error |
| 2 | Index not found |
| 6 | Index stale (papers missing) |

### Examples

```bash
# Check index health
bp index check

# Human-readable
bp index check --human

# Use in scripts
bp index check || bp index build
```

---

## Error Messages

Consistent error message format across all commands:

```
Error: <brief description>

<detailed explanation or hint>
```

### Standard Errors

| Condition | Message |
|-----------|---------|
| Index not found | `Error: Semantic index not found\n\nRun 'bp index build' to create the index.` |
| Ollama not running | `Error: Ollama is not running\n\nStart Ollama with 'ollama serve' or install from https://ollama.ai` |
| Model not found | `Error: Embedding model 'all-minilm:l6-v2' not found\n\nRun 'ollama pull all-minilm:l6-v2' to download it.` |
| Paper not found | `Error: Paper 'XYZ-123' not found` |
| Paper no abstract | `Error: Paper 'XYZ-123' has no abstract\n\nSimilarity search requires papers with abstracts.` |
| Empty query | `Error: Search query cannot be empty` |
