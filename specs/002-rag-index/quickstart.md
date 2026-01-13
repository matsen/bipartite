# Quickstart: RAG Index for Semantic Search

**Feature**: 002-rag-index
**Date**: 2026-01-12

This guide walks through setting up and using semantic search in bipartite.

## Prerequisites

1. **bipartite Phase I complete**: You have an initialized repository with imported papers
2. **Ollama installed**: Install from [ollama.ai](https://ollama.ai)
3. **Embedding model pulled**: Run `ollama pull all-minilm:l6-v2`

## Setup

### 1. Verify Ollama is Running

```bash
# Start Ollama (if not already running)
ollama serve

# In another terminal, verify it's working
curl http://localhost:11434/api/tags
```

### 2. Pull the Embedding Model

```bash
ollama pull all-minilm:l6-v2
```

This downloads the ~22MB model for generating embeddings locally.

### 3. Build the Semantic Index

```bash
# Build index from your imported papers
bp index build

# Expected output:
# Building semantic index...
# [============================] 6235/6235 papers
#
# Build complete:
#   Papers indexed: 5889
#   Papers skipped: 346 (no abstract)
#   Time elapsed: 2m 34s
#   Index size: 12.3 MB
```

## Usage

### Semantic Search

Find papers by concept, not just keywords:

```bash
# Search for papers about a concept
bp semantic "methods for inferring evolutionary trees"

# Limit results
bp semantic "protein folding" --limit 5

# Higher similarity threshold (more relevant results only)
bp semantic "MCMC sampling methods" --threshold 0.7

# Human-readable output
bp semantic "variational inference" --human
```

### Find Similar Papers

Discover papers related to one you know:

```bash
# Find papers similar to a specific paper
bp similar Matsen2025-ab

# Limit results
bp similar Matsen2025-ab --limit 5

# Human-readable
bp similar Matsen2025-ab --human
```

### Check Index Health

Verify your index is up-to-date:

```bash
bp index check

# Expected output (healthy):
# {
#   "status": "healthy",
#   "papers_indexed": 5889,
#   "papers_missing": 0,
#   ...
# }
```

## Workflow Examples

### Research Discovery

```bash
# You're writing about phylogenetics and want related papers
bp semantic "Bayesian inference for phylogenetic trees" --limit 10 --human

# Found an interesting paper, want more like it
bp similar Suchard2024-ab --limit 5 --human

# Export the papers you found for your bibliography
bp export --bibtex --keys Suchard2024-ab,Drummond2023-cd
```

### Agent Workflow

```bash
# Agent finds relevant papers
papers=$(bp semantic "neural network protein structure" --limit 5 | jq -r '.results[].id')

# Agent opens top result for human to read
bp open $(echo "$papers" | head -1)
```

### After Adding New Papers

```bash
# Import new papers from Paperpile
bp import --format paperpile latest-export.json

# Rebuild to include new papers in semantic search
bp index build

# Verify
bp index check
```

## Troubleshooting

### "Ollama is not running"

```bash
# Start Ollama
ollama serve
```

### "Model not found"

```bash
# Pull the embedding model
ollama pull all-minilm:l6-v2
```

### "Semantic index not found"

```bash
# Build the index
bp index build
```

### Index is stale (papers missing)

```bash
# Check what's missing
bp index check --human

# Rebuild to include new papers
bp index build
```

### Search returns unexpected results

Semantic search finds conceptually similar papers, not keyword matches. If results seem off:

1. Try rephrasing your query
2. Use a higher threshold: `--threshold 0.7`
3. Use keyword search for exact terms: `bp search "exact phrase"`

## Performance Notes

- **First search**: ~100-200ms (loading index into memory)
- **Subsequent searches**: ~4ms (index cached in memory)
- **Index build**: ~2-5 minutes for 6000 papers
- **Index size**: ~1.5KB per paper (~9MB for 6000 papers, GOB format)

## Validation Checklist

Use this to verify Phase II is working:

```bash
# 1. Ollama running
curl -s http://localhost:11434/api/tags | jq .models

# 2. Model available
ollama list | grep all-minilm

# 3. Index built
bp index check

# 4. Semantic search works
bp semantic "test query" --limit 1

# 5. Similar papers works
# (use a paper ID from your collection)
bp similar $(bp list --limit 1 | jq -r '.[0].id')
```

All commands should succeed with exit code 0.
