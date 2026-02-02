# CLI Contract: bip concept

**Feature Branch**: `006-concept-nodes`
**Date**: 2026-01-21

## Overview

All `bip concept` commands follow bipartite CLI conventions:
- Default output: JSON (machine-readable)
- Human output: `--human` flag
- Exit codes: 0 (success), 1 (general error), 2 (data error), 3 (validation error)

---

## Commands

### bip concept add

Create a new concept.

**Usage**:
```bash
bip concept add <id> --name <name> [--aliases <alias1,alias2>] [--description <desc>] [--human]
```

**Arguments**:
- `<id>`: Concept ID (required, positional). Must match `^[a-z0-9][a-z0-9_-]*$`

**Flags**:
- `--name, -n <string>`: Display name (required)
- `--aliases, -a <string>`: Comma-separated aliases (optional)
- `--description, -d <string>`: Description text (optional)
- `--human`: Human-readable output

**Exit Codes**:
- `0`: Concept created successfully
- `1`: General error (file I/O, etc.)
- `3`: Validation error (invalid ID, duplicate ID)

**JSON Output** (success):
```json
{
  "status": "created",
  "concept": {
    "id": "somatic-hypermutation",
    "name": "Somatic Hypermutation",
    "aliases": ["SHM"],
    "description": "Process by which B cells diversify antibody genes"
  }
}
```

**Human Output** (success):
```
Created concept: somatic-hypermutation
  Name: Somatic Hypermutation
  Aliases: SHM
  Description: Process by which B cells diversify antibody genes
```

**Error Output** (duplicate ID):
```json
{
  "error": "concept with id 'somatic-hypermutation' already exists"
}
```

---

### bip concept get

Retrieve a concept by ID.

**Usage**:
```bash
bip concept get <id> [--human]
```

**Arguments**:
- `<id>`: Concept ID (required, positional)

**Exit Codes**:
- `0`: Concept found
- `2`: Concept not found

**JSON Output** (success):
```json
{
  "id": "somatic-hypermutation",
  "name": "Somatic Hypermutation",
  "aliases": ["SHM"],
  "description": "Process by which B cells diversify antibody genes"
}
```

**Human Output** (success):
```
somatic-hypermutation
  Name: Somatic Hypermutation
  Aliases: SHM
  Description: Process by which B cells diversify antibody genes
```

---

### bip concept list

List all concepts.

**Usage**:
```bash
bip concept list [--human]
```

**Exit Codes**:
- `0`: Success (even if empty)

**JSON Output**:
```json
{
  "concepts": [
    {
      "id": "phylogenetics",
      "name": "Phylogenetics",
      "aliases": [],
      "description": "Study of evolutionary relationships"
    },
    {
      "id": "somatic-hypermutation",
      "name": "Somatic Hypermutation",
      "aliases": ["SHM"],
      "description": "Process by which B cells diversify antibody genes"
    }
  ],
  "count": 2
}
```

**Human Output**:
```
phylogenetics
  Name: Phylogenetics

somatic-hypermutation
  Name: Somatic Hypermutation
  Aliases: SHM

Total: 2 concepts
```

---

### bip concept update

Update an existing concept.

**Usage**:
```bash
bip concept update <id> [--name <name>] [--aliases <alias1,alias2>] [--description <desc>] [--human]
```

**Arguments**:
- `<id>`: Concept ID (required, positional)

**Flags**:
- `--name, -n <string>`: New display name (optional)
- `--aliases, -a <string>`: Comma-separated aliases (optional, replaces existing)
- `--description, -d <string>`: New description (optional)
- `--human`: Human-readable output

**Exit Codes**:
- `0`: Concept updated
- `2`: Concept not found
- `3`: No update flags provided

**JSON Output** (success):
```json
{
  "status": "updated",
  "concept": {
    "id": "somatic-hypermutation",
    "name": "Somatic Hypermutation (SHM)",
    "aliases": ["SHM", "somatic hypermutation"],
    "description": "Updated description"
  }
}
```

---

### bip concept delete

Delete a concept.

**Usage**:
```bash
bip concept delete <id> [--force] [--human]
```

**Arguments**:
- `<id>`: Concept ID (required, positional)

**Flags**:
- `--force, -f`: Delete even if edges exist (required when edges exist)
- `--human`: Human-readable output

**Exit Codes**:
- `0`: Concept deleted
- `2`: Concept not found
- `3`: Concept has linked edges (use --force)

**JSON Output** (blocked by edges):
```json
{
  "error": "concept 'somatic-hypermutation' has 5 linked edges; use --force to delete anyway",
  "edge_count": 5
}
```

**JSON Output** (success with --force):
```json
{
  "status": "deleted",
  "id": "somatic-hypermutation",
  "edges_removed": 5
}
```

---

### bip concept papers

Query papers linked to a concept.

**Usage**:
```bash
bip concept papers <concept-id> [--type <relationship-type>] [--human]
```

**Arguments**:
- `<concept-id>`: Concept ID (required, positional)

**Flags**:
- `--type, -t <string>`: Filter by relationship type (optional)
- `--human`: Human-readable output

**Exit Codes**:
- `0`: Success (even if no papers)
- `2`: Concept not found

**JSON Output**:
```json
{
  "concept_id": "variational-inference",
  "papers": [
    {
      "paper_id": "Matsen2025-oj",
      "relationship_type": "applies",
      "summary": "Uses VI for posterior approximation"
    },
    {
      "paper_id": "Blei2003-kj",
      "relationship_type": "introduces",
      "summary": "Foundational paper on VI methods"
    }
  ],
  "count": 2
}
```

**Human Output**:
```
Papers linked to: variational-inference

[introduces]
  Blei2003-kj: Foundational paper on VI methods

[applies]
  Matsen2025-oj: Uses VI for posterior approximation

Total: 2 papers
```

---

### bip concept merge

Merge one concept into another.

**Usage**:
```bash
bip concept merge <source-id> <target-id> [--human]
```

**Arguments**:
- `<source-id>`: Concept to merge FROM (will be deleted)
- `<target-id>`: Concept to merge INTO (will survive)

**Exit Codes**:
- `0`: Merge successful
- `2`: Source or target concept not found
- `3`: Source and target are the same

**JSON Output**:
```json
{
  "status": "merged",
  "source_id": "shm",
  "target_id": "somatic-hypermutation",
  "edges_updated": 3,
  "aliases_added": ["SHM"],
  "duplicates_removed": 1
}
```

**Human Output**:
```
Merged 'shm' into 'somatic-hypermutation'
  Edges updated: 3
  Aliases added: SHM
  Duplicate edges removed: 1
```

---

### bip paper concepts

Query concepts linked to a paper.

**Usage**:
```bash
bip paper concepts <paper-id> [--type <relationship-type>] [--human]
```

**Arguments**:
- `<paper-id>`: Paper ID (required, positional)

**Flags**:
- `--type, -t <string>`: Filter by relationship type (optional)
- `--human`: Human-readable output

**Exit Codes**:
- `0`: Success (even if no concepts)
- `2`: Paper not found

**JSON Output**:
```json
{
  "paper_id": "Matsen2025-oj",
  "concepts": [
    {
      "concept_id": "variational-inference",
      "relationship_type": "applies",
      "summary": "Uses VI for posterior approximation"
    },
    {
      "concept_id": "phylogenetics",
      "relationship_type": "applies",
      "summary": "Applies to phylogenetic inference problems"
    }
  ],
  "count": 2
}
```

**Human Output**:
```
Concepts for paper: Matsen2025-oj

[applies]
  variational-inference: Uses VI for posterior approximation
  phylogenetics: Applies to phylogenetic inference problems

Total: 2 concepts
```

---

## Integration with Existing Commands

### bip edge add

**Existing behavior**: Validates source and target against refs.jsonl

**New behavior**: Validates target against BOTH refs.jsonl AND concepts.jsonl
- If target found in refs → paper-paper edge
- If target found in concepts → paper-concept edge
- If target found in neither → error

**Warning for non-standard relationship types**:
- If target is concept and relationship_type not in `relationship-types.json` paper-concept list → warning (not error)

### bip rebuild

**Existing behavior**: Rebuilds refs + edges SQLite tables

**New behavior**: Also rebuilds concepts + concepts_fts tables
- Output includes concept count

**JSON Output**:
```json
{
  "status": "rebuilt",
  "references": 150,
  "edges": 45,
  "concepts": 12
}
```

---

## Shared Flags

All commands support:
- `--human`: Human-readable output (default: JSON)

Commands that modify data also respect:
- Repository discovery: Uses `nexus_path` from `~/.config/bip/config.yml`
