# Quickstart: Knowledge Graph

**Feature**: 003-knowledge-graph
**Date**: 2026-01-13

## Prerequisites

- Bipartite repo initialized (`bip init`)
- Papers imported (`bip import`)

## Basic Usage

### Add an Edge

```bash
# Add a relationship between two papers
bp edge add \
  --source Smith2024-ab \
  --target Jones2023-xy \
  --type extends \
  --summary "Extends Jones's variational framework to handle non-Euclidean geometries"
```

### Query Edges

```bash
# List edges from a paper
bp edge list Smith2024-ab

# List edges to a paper (incoming)
bp edge list Smith2024-ab --incoming

# List all edges (both directions)
bp edge list Smith2024-ab --all

# Search by relationship type
bp edge search --type extends
```

### Export Edges

```bash
# Export all edges
bp edge export > my-edges.jsonl

# Export edges for specific paper
bp edge export --paper Smith2024-ab > smith-edges.jsonl
```

## Bulk Import (for tex-to-edges workflow)

```bash
# Import edges from JSONL file
bp edge import edges-from-manuscript.jsonl
```

**Input format** (one JSON object per line):
```json
{"source_id":"manuscript","target_id":"Smith2024-ab","relationship_type":"cites","summary":"Cited for foundational variational methods"}
```

## Agent Usage (JSON output)

All commands support `--json` for structured output:

```bash
# Add edge with JSON response
bp edge add -s A -t B -r cites -m "..." --json

# List edges as JSON
bp edge list Smith2024-ab --json

# Search with JSON output
bp edge search --type extends --json
```

## Maintenance

```bash
# Rebuild edge index (after git pull)
bp rebuild

# Check for orphaned edges
bp groom

# Verify edge integrity
bp check
```

## Example Workflow: tex-to-edges

1. External tool analyzes manuscript and bibliography
2. Generates edges.jsonl with relationships
3. Import into bipartite:

```bash
# Tool output
cat manuscript-edges.jsonl
{"source_id":"my-manuscript","target_id":"Smith2024-ab","relationship_type":"extends","summary":"We extend Smith's method to phylogenetic trees"}
{"source_id":"my-manuscript","target_id":"Jones2023-xy","relationship_type":"cites","summary":"Background on variational inference"}

# Import
bp edge import manuscript-edges.jsonl
# Output: Imported 2 edges (0 updated, 0 skipped)

# Query
bp edge list my-manuscript
# Output: Shows both edges with summaries
```
