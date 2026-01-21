# Quickstart: Concept Nodes

**Feature Branch**: `006-concept-nodes`

## Getting Started

After building bipartite with concept node support, here's how to use the new features.

### 1. Build

```bash
go build -o bip ./cmd/bip
```

### 2. Create Concepts

Define key concepts in your research domain:

```bash
# Add a concept with name, aliases, and description
./bip concept add somatic-hypermutation \
  --name "Somatic Hypermutation" \
  --aliases "SHM,shm" \
  --description "Process by which B cells diversify antibody genes through point mutations"

# Add more concepts
./bip concept add variational-inference \
  --name "Variational Inference" \
  --aliases "VI" \
  --description "Approximation method for Bayesian inference"

./bip concept add phylogenetics \
  --name "Phylogenetics" \
  --description "Study of evolutionary relationships among biological entities"
```

### 3. Link Papers to Concepts

Create edges from papers to concepts:

```bash
# Paper introduces a concept
./bip edge add Halpern1998-yc somatic-hypermutation introduces \
  --summary "Foundational paper describing somatic hypermutation mechanism"

# Paper applies a concept/method
./bip edge add Matsen2025-oj variational-inference applies \
  --summary "Uses VI for posterior approximation in phylogenetic models"

# Paper models a phenomenon
./bip edge add McCoy2022-bd somatic-hypermutation models \
  --summary "Creates computational model of SHM targeting patterns"
```

### 4. Query Papers by Concept

Find all papers related to a concept:

```bash
# All papers for a concept
./bip concept papers somatic-hypermutation

# Filter by relationship type
./bip concept papers somatic-hypermutation --type introduces

# Human-readable output
./bip concept papers variational-inference --human
```

### 5. Query Concepts by Paper

See what concepts a paper relates to:

```bash
./bip paper concepts Matsen2025-oj --human
```

### 6. Manage Concepts

```bash
# List all concepts
./bip concept list --human

# Get a specific concept
./bip concept get somatic-hypermutation --human

# Update a concept
./bip concept update somatic-hypermutation \
  --description "Updated description with more detail"

# Delete a concept (will warn if papers are linked)
./bip concept delete unused-concept

# Force delete (removes linked edges)
./bip concept delete old-concept --force
```

### 7. Merge Duplicate Concepts

If you discover two concepts are the same:

```bash
# Merge 'shm' into 'somatic-hypermutation'
# - All edges from 'shm' move to 'somatic-hypermutation'
# - Aliases from 'shm' are added to 'somatic-hypermutation'
# - 'shm' is deleted
./bip concept merge shm somatic-hypermutation --human
```

### 8. Rebuild Index

After modifying JSONL files directly, rebuild the SQLite index:

```bash
./bip rebuild
# Output includes concept count: {"status":"rebuilt","references":150,"edges":45,"concepts":12}
```

---

## Standard Relationship Types

Use these relationship types for paper-concept edges (from `relationship-types.json`):

| Type | When to Use |
|------|-------------|
| `introduces` | Paper first presents or defines this concept |
| `applies` | Paper uses concept as a tool or method |
| `models` | Paper creates computational/mathematical model of phenomenon |
| `evaluates-with` | Paper uses concept for evaluation or benchmarking |
| `critiques` | Paper identifies limitations or problems with concept |
| `extends` | Paper builds upon or extends the concept |

Custom types are allowed (you'll see a warning but the edge will be created).

---

## Example Workflow

Building a knowledge graph for antibody research:

```bash
# 1. Create domain concepts
./bip concept add bcr-sequencing --name "BCR Sequencing" --aliases "B cell receptor sequencing"
./bip concept add clonal-expansion --name "Clonal Expansion"
./bip concept add affinity-maturation --name "Affinity Maturation"

# 2. Link papers as you read them
./bip edge add Georgiou2014-xz bcr-sequencing introduces \
  --summary "Reviews high-throughput BCR sequencing methods"

./bip edge add Matsen2025-oj bcr-sequencing applies \
  --summary "Uses BCR-seq data for phylogenetic inference"

# 3. Query to find related work
./bip concept papers bcr-sequencing --human

# 4. Find what concepts a new paper touches
./bip paper concepts NewPaper2026-ab --human
```

---

## Output Formats

All commands default to JSON for scripting:

```bash
# JSON (default)
./bip concept list | jq '.concepts[].id'

# Human-readable
./bip concept list --human
```
