---
name: tex-edges
description: Add knowledge graph edges from TeX paper citations. Use when connecting papers cited in manuscripts to the bip knowledge graph.
---

# Adding Edges from TeX Paper Citations

Workflow for extracting citations from TeX manuscripts and adding them as knowledge graph edges.

## Quick Start

```bash
/tex-edges ~/writing/my-paper-tex
```

## Workflow

### 1. Explore the TeX repo with a subagent

Use a parallel Explore subagent to understand the paper's topic, key concepts, and related projects.

### 2. Extract citation keys from .tex files

Grep the `.tex` files for all citation commands (`\cite`, `\citep`, `\citet`, `\citealp`, etc.):

```bash
grep -roh '\\cite[a-zA-Z]*{[^}]*}' ~/writing/paper-tex/*.tex \
  | tr ',' '\n' | sed 's/.*{//;s/}//;s/^ *//' | sort | uniq -c | sort -rn
```

The `uniq -c | sort -rn` gives citation frequency — the most-cited papers are likely the most important ones to connect.

### 3. Check citation keys against library

Citation keys (e.g., `Kim2020-ip`, `Aldous1996-fk`) usually match bip paper IDs directly:

```bash
cd ~/re/nexus
for key in Key1 Key2 Key3; do
  echo -n "$key: "
  bip get "$key" --human 2>/dev/null | head -1 || echo "NOT FOUND"
done
```

**When a key is NOT FOUND**, the paper may still be in the library under a different key suffix (e.g., `Sethna2019-lv` in bib vs `Sethna2019-at` in library). Search by title or keyword:

```bash
bip search "OLGA"  # search by tool/method name or keyword
```

If found under a different key, **ask the user** if they want to fix the bib and tex files to use the library's key. If yes, update both `main.bib` (the `@ARTICLE{...}` key) and all `\cite{...}` references in `.tex` files.

For papers truly not in library, add via `bip s2 add "DOI:..."`. If rate-limited, note for later.

### 4. Identify concepts

Look at the highly-cited papers and the subagent analysis. Create concept nodes for:
- Named algorithms or methods (e.g., "beta-splitting", "F-matrices")
- Biological processes (e.g., "affinity-maturation")
- Modeling frameworks (e.g., "mutation-selection-model")

Only create concepts for things that **bridge multiple papers** — a concept that only one paper touches isn't pulling its weight in the graph.

```bash
bip concept add concept-id --name "Name" --aliases "a,b" --description "..."
```

### 5. Add edges

```bash
# Paper → concept
bip edge add -s PaperID -t concept:concept-id -r introduces -m "Summary"

# Concept → project (if relevant)
bip edge add -s concept:concept-id -t project:project-id -r applied-in -m "Summary"
```

**Paper → concept relationship types**:

| Type | Use |
|------|-----|
| `introduces` | Paper first presents or defines this concept |
| `extends` | Paper builds upon or generalizes the concept |
| `applies` | Paper uses concept as a tool or method |
| `models` | Paper provides computational/mathematical model |
| `evaluates-with` | Paper uses concept for evaluation/benchmarking |
| `critiques` | Paper identifies limitations or problems |

**Concept → project relationship types**: `applied-in`, `relevant-to`

### 6. Rebuild visualization

```bash
cd ~/re/nexus && make viz
open -a "Google Chrome" viz/knowledge-graph.html
```

## Example Session

```bash
# 1. Get citation keys ranked by frequency
grep -roh '\\cite[a-zA-Z]*{[^}]*}' ~/writing/my-method-tex/*.tex \
  | tr ',' '\n' | sed 's/.*{//;s/}//;s/^ *//' | sort | uniq -c | sort -rn

# 2. Check top keys against library
cd ~/re/nexus
for key in Smith2020-ab Jones2019-cd Brown2021-ef; do
  echo -n "$key: "
  bip get "$key" --human 2>/dev/null | head -1 || echo "$key: NOT FOUND"
done

# 3. Add missing papers
bip s2 add "DOI:10.1234/missing-paper"

# 4. Create concept if it bridges papers
bip concept add my-method --name "My Method" --description "Novel approach for X"

# 5. Add edges
bip edge add -s MyPaper2025-xx -t concept:my-method -r introduces -m "Introduces the method"
bip edge add -s Smith2020-ab -t concept:my-method -r applies -m "Earlier application"
bip edge add -s concept:my-method -t project:my-project -r applied-in -m "Core method"

# 6. Visualize
make viz && open -a "Google Chrome" viz/knowledge-graph.html
```

## Tips

- **Citation keys ≈ paper IDs** — check directly first, but key suffixes can differ
- **Key mismatch?** Search by keyword (`bip search "OLGA"`), then fix bib+tex to match the library key
- **Citation frequency** from grep helps prioritize which papers matter most
- **Use subagents** for exploring TeX repos — keeps main context clean
- **Create concepts conservatively** — only for named methods that bridge papers
- **Bib files are usually unnecessary** — the citation keys give you direct library access
- **Batch edge additions** when possible — less back-and-forth
