# Quickstart: Projects and Repos

**Feature**: 011-repo-nodes | **Date**: 2026-01-24

## Overview

Projects and repos extend the bipartite knowledge graph to represent your ongoing work:

```
papers ←──→ concepts ←──→ projects
  (literature)  (ideas)    (your work)
                              │
                              └── repos (GitHub links)
```

## Workflow Example

### 1. Create a Project

```bash
bip project add dasm2 --name "DASM2" --description "Distance-based antibody sequence modeling"
```

### 2. Add Repos to the Project

```bash
# From GitHub URL
bip repo add https://github.com/matsen/dasm2 --project dasm2

# Or shorthand
bip repo add matsen/dasm2-paper --project dasm2
```

### 3. Link Concepts to the Project

```bash
# First, ensure the concept exists
bip concept add variational-inference --name "Variational Inference"

# Then create the edge (concept → project)
bip edge add \
  --source concept:variational-inference \
  --target project:dasm2 \
  --type implemented-in \
  --summary "DASM2 uses VI for the latent space model"
```

### 4. Query the Graph

```bash
# What concepts does my project use?
bip project concepts dasm2

# What papers are relevant to my project? (via concepts)
bip project papers dasm2

# What repos belong to my project?
bip project repos dasm2
```

## Key Concepts

### Type-Prefixed IDs

When creating edges involving projects or concepts, use type prefixes:

| Node Type | Prefix | Example |
|-----------|--------|---------|
| Paper | (none) | `10.1038/s41586-021-03819-2` |
| Concept | `concept:` | `concept:variational-inference` |
| Project | `project:` | `project:dasm2` |
| Repo | `repo:` | `repo:dasm2-code` (no edges allowed) |

### Graph Constraints

- **No direct paper↔project edges**: Connections must go through concepts
- **Repos have no edges**: Repos are metadata, not edge endpoints
- **One project per repo**: A GitHub URL can only belong to one project

## Common Tasks

### View All Projects

```bash
bip project list
```

### Refresh GitHub Metadata

```bash
bip repo refresh dasm2-code
```

### Delete a Project (and its repos)

```bash
# Check first
bip project repos dasm2
bip project concepts dasm2

# Delete (will fail if edges exist)
bip project delete dasm2

# Force delete (removes repos and edges)
bip project delete dasm2 --force
```

### Add a Manual Repo (non-GitHub)

```bash
bip repo add --manual \
  --project dasm2 \
  --id dasm2-internal \
  --name "Internal Tools" \
  --description "Private internal tooling"
```

## Integration with Existing Workflows

### Finding Papers for Your Project

1. Identify concepts your project uses
2. Query papers that introduce/apply those concepts
3. The concept layer explains *why* the paper matters

```bash
# Step 1: What concepts does my project use?
bip project concepts dasm2

# Step 2: What papers introduce variational inference?
bip concept papers variational-inference --type introduces
```

### Validating Your Graph

```bash
# Check for orphaned edges, invalid references
bip check
```

### Rebuilding the Index

After manually editing JSONL files:

```bash
bip rebuild
```
