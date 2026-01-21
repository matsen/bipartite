# Phase III-b: Concept Nodes

**Status**: Vision Document (pre-spec)
**Depends on**: 003-knowledge-graph (paper edges)

## Overview

Extend the knowledge graph with **concept nodes** — named ideas, methods, or phenomena that papers relate to. This enables queries like:
- "What papers discuss somatic hypermutation?"
- "What concepts does this paper introduce vs. apply?"
- "Find papers that critique masked language models"

## Scope

**In scope (this phase)**:
- Concept node storage (`concepts.jsonl`)
- Paper ↔ concept edges (extend existing `edges.jsonl`)
- CLI commands for concept CRUD and querying

**Out of scope (deferred)**:
- Concept ↔ concept edges (e.g., "variational inference extends probabilistic inference")
- Automatic concept extraction (external tools handle this)

## Data Model

### Concept Node

```jsonl
{"id":"somatic-hypermutation","name":"somatic hypermutation","aliases":["SHM"],"description":"Enzymatic process introducing mutations into BCR-coding DNA"}
{"id":"mutation-selection-model","name":"mutation-selection model","aliases":["MutSel"],"description":"Framework separating neutral mutation probability from selection factors"}
{"id":"masked-language-model","name":"masked language model","aliases":["MLM","BERT-style"],"description":"Training objective predicting masked tokens from surrounding context"}
{"id":"transformer-encoder","name":"transformer encoder","aliases":[],"description":"Neural network architecture using self-attention for sequence modeling"}
{"id":"deep-mutational-scanning","name":"deep mutational scanning","aliases":["DMS"],"description":"High-throughput assay measuring effects of all single mutations"}
{"id":"affinity-maturation","name":"affinity maturation","aliases":[],"description":"Selection process in germinal centers improving antibody binding"}
```

### Paper → Concept Edges

Standard relationship types are documented in `/relationship-types.json`. Bip accepts any string, but tools should use standard types by default. Extensions are encouraged when justified — add them to the vocabulary file with a clear description before use.

```jsonl
{"source_id":"Halpern1998-yc","target_id":"mutation-selection-model","relationship_type":"introduces","summary":"Introduces codon-level mutation-selection framework with site-specific amino acid frequencies"}
{"source_id":"Sung2025-hz","target_id":"somatic-hypermutation","relationship_type":"models","summary":"Creates wide-context neural network model of SHM trained on neutrally-evolving out-of-frame data"}
{"source_id":"Olsen2022-fp","target_id":"masked-language-model","relationship_type":"applies","summary":"Uses masked objective to predict antibody sequences, learning germline and mutation patterns"}
{"source_id":"Devlin2018-bd","target_id":"masked-language-model","relationship_type":"introduces","summary":"Introduces bidirectional masked pre-training for language understanding"}
{"source_id":"Devlin2018-bd","target_id":"transformer-encoder","relationship_type":"applies","summary":"Uses transformer encoder architecture for bidirectional context"}
{"source_id":"Chungyoun2024-fc","target_id":"deep-mutational-scanning","relationship_type":"evaluates-with","summary":"Curates DMS datasets as benchmarks for antibody fitness prediction"}
{"source_id":"Adams2016-ja","target_id":"deep-mutational-scanning","relationship_type":"extends","summary":"Extends DMS with titration curves to measure sequence-affinity landscapes"}
{"source_id":"Lin2023-wd","target_id":"transformer-encoder","relationship_type":"applies","summary":"Scales transformer to 15B parameters for evolutionary protein structure prediction"}
```

## User Story

**As a researcher**, I want to tag papers with the concepts they discuss, so that I can:
1. Find all papers in my collection that discuss a specific concept
2. Understand what concepts a paper introduces vs. merely applies
3. Build a map of which methods/ideas are used across my literature

### Example Workflow

```bash
# Add a concept
bip concept add somatic-hypermutation \
  --name "somatic hypermutation" \
  --alias "SHM" \
  --description "Enzymatic process introducing mutations into BCR-coding DNA"

# Link paper to concept
bip edge add --source Sung2025-hz --target somatic-hypermutation \
  --type models \
  --summary "Creates wide-context neural network model of SHM"

# Find all papers discussing a concept
bip concept papers somatic-hypermutation

# Find all concepts a paper relates to
bip paper concepts Sung2025-hz

# List all concepts
bip concept list
```

## Design Considerations

### Node ID Namespacing

Paper IDs come from Paperpile (e.g., `Halpern1998-yc`). Concept IDs are user-defined slugs (e.g., `somatic-hypermutation`). These namespaces are unlikely to collide, but we could add a prefix (`concept:somatic-hypermutation`) if needed.

### Edge Storage

Paper-concept edges use the same `edges.jsonl` as paper-paper edges. The `target_id` field can reference either a paper ID or a concept ID. Validation checks that the target exists in either `refs.jsonl` or `concepts.jsonl`.

### Concept Merging

If two concepts are later discovered to be the same, we need a merge operation:
```bash
bip concept merge old-concept-id new-concept-id
```
This updates all edges pointing to `old-concept-id` to point to `new-concept-id`.

## Future Extensions

- **Concept → concept edges**: Taxonomic relationships like "variational inference is-a probabilistic inference"
- **Concept extraction tool**: Claude skill to analyze abstracts and suggest concepts
- **Concept embeddings**: Semantic search over concepts using RAG index
