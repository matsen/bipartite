---
name: bip
description: Unified guidance for using the bipartite reference library CLI. Use when searching for papers, managing the library, or exploring literature via S2/ASTA.
---

# Bip Reference Library

A CLI tool for managing academic references with local storage and external paper search.

**Repository**: `/Users/matsen/re/bipartite`
**PDF Storage**: `/Users/matsen/Google Drive/My Drive/Paperpile`

## Quick Reference

| Task | Command |
|------|---------|
| Search local library | `./bip search "query"` |
| Semantic search | `./bip semantic "query"` |
| Get paper details | `./bip get <id>` |
| Add paper to collection | `./bip s2 add DOI:10.1234/...` |
| Find literature gaps | `./bip s2 gaps` |
| Fast paper search (external) | `./bip asta search "query"` |
| Find text snippets | `./bip asta snippet "query"` |
| Create concept | `./bip concept add <id> --name "Name"` |
| Link paper to concept | `./bip edge add -s <paper> -t <concept> -r <type> -m "summary"` |
| Papers for concept | `./bip concept papers <concept-id>` |
| Concepts for paper | `./bip paper concepts <paper-id>` |

## S2 vs ASTA: When to Use Which

Both access Semantic Scholar's paper database but through different APIs:

| Use Case | Command | Why |
|----------|---------|-----|
| Add paper to collection | `bip s2 add` | Only S2 can modify local library |
| Find literature gaps | `bip s2 gaps` | Analyzes your collection |
| Explore without adding | `bip asta *` | Faster, read-only |
| Find text snippets in papers | `bip asta snippet` | Unique to ASTA |
| Fast paper search | `bip asta search` | 10x faster rate limit |
| Get citations/references | Either works | ASTA is faster |

**Rule of thumb**: Use `bip asta` for exploration, `bip s2` when you want to modify your library.

See [api-guide.md](api-guide.md) for detailed comparison.

## Common Workflows

### Find a Paper

1. **Search local library first**:
   ```bash
   ./bip search "Schmidler phylogenetics"
   # or for topic-heavy queries:
   ./bip semantic "importance sampling MCMC"
   ```

2. **Get PDF path** for a result:
   ```bash
   ./bip get <id>
   # pdf_path field + "/Users/matsen/Google Drive/My Drive/Paperpile"
   ```

3. **If not in library**, search externally:
   ```bash
   ./bip asta search "phylogenetic inference"
   ```

### Update Library from Paperpile

1. Export from Paperpile (JSON format) to ~/Downloads
2. Find the export file:
   ```bash
   ls -t ~/Downloads/Paperpile*.json | head -1
   ```
3. Import:
   ```bash
   ./bip import --format paperpile "<path>"
   ```
4. Optionally delete the export file after confirming success

### Explore Literature

1. **Search by topic**:
   ```bash
   ./bip asta search "variational inference phylogenetics" --limit 20
   ```

2. **Find specific text passages**:
   ```bash
   ./bip asta snippet "Bayesian phylogenetic inference"
   ```

3. **Trace citations**:
   ```bash
   ./bip asta citations DOI:10.1093/sysbio/syy032
   ./bip asta references DOI:10.1093/sysbio/syy032
   ```

4. **Add interesting papers** to your collection:
   ```bash
   ./bip s2 add DOI:10.1093/sysbio/syy032
   ```

See [workflows.md](workflows.md) for detailed workflow instructions.

## Output Format

All commands output JSON by default. Add `--human` for readable format:

```bash
./bip asta search "phylogenetics" --human
./bip s2 lookup DOI:10.1234/example --human
```

## Paper ID Formats

Both S2 and ASTA accept these identifier formats:
- `DOI:10.1093/sysbio/syy032`
- `ARXIV:2106.15928`
- `PMID:19872477`
- `CorpusId:215416146`
- Raw Semantic Scholar ID (40-char hex)

## Concept Nodes (Knowledge Graph)

Build a knowledge graph by creating concepts and linking papers to them.

### Create Concepts

```bash
# Add a concept with name, aliases, and description
./bip concept add somatic-hypermutation \
  --name "Somatic Hypermutation" \
  --aliases "SHM,shm" \
  --description "Process by which B cells diversify antibody genes"

# List all concepts
./bip concept list --human

# Get a specific concept
./bip concept get somatic-hypermutation --human
```

### Link Papers to Concepts

```bash
# Use flags: -s (source paper), -t (target concept), -r (relationship type), -m (summary)
./bip edge add -s Halpern1998-yc -t mutation-selection-model -r introduces \
  -m "Foundational paper defining the mutation-selection model"

./bip edge add -s Yaari2013-dg -t somatic-hypermutation -r models \
  -m "Introduces S5F model for SHM targeting"
```

### Standard Relationship Types

| Type | When to Use |
|------|-------------|
| `introduces` | Paper first presents or defines this concept |
| `applies` | Paper uses concept as a tool or method |
| `models` | Paper creates computational/mathematical model |
| `evaluates-with` | Paper uses concept for evaluation/benchmarking |
| `critiques` | Paper identifies limitations or problems |
| `extends` | Paper builds upon or extends the concept |

### Query the Knowledge Graph

```bash
# Find all papers linked to a concept
./bip concept papers somatic-hypermutation --human

# Filter by relationship type
./bip concept papers somatic-hypermutation --type introduces

# Find what concepts a paper relates to
./bip paper concepts Halpern1998-yc --human
```

### Manage Concepts

```bash
# Update a concept
./bip concept update somatic-hypermutation --description "Updated description"

# Delete a concept (warns if papers linked)
./bip concept delete unused-concept

# Force delete (removes linked edges too)
./bip concept delete old-concept --force

# Merge duplicate concepts
./bip concept merge shm somatic-hypermutation --human
```
