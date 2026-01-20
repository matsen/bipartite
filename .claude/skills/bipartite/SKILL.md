---
name: bipartite
description: Unified guidance for using the bipartite reference library CLI. Use when searching for papers, managing the library, or exploring literature via S2/ASTA.
---

# Bipartite Reference Library

A CLI tool for managing academic references with local storage and external paper search.

**Repository**: `/Users/matsen/re/bipartite`
**PDF Storage**: `/Users/matsen/Google Drive/My Drive/Paperpile`

## Quick Reference

| Task | Command |
|------|---------|
| Search local library | `./bp search "query"` |
| Semantic search | `./bp semantic "query"` |
| Get paper details | `./bp get <id>` |
| Add paper to collection | `./bp s2 add DOI:10.1234/...` |
| Find literature gaps | `./bp s2 gaps` |
| Fast paper search (external) | `./bp asta search "query"` |
| Find text snippets | `./bp asta snippet "query"` |

## S2 vs ASTA: When to Use Which

Both access Semantic Scholar's paper database but through different APIs:

| Use Case | Command | Why |
|----------|---------|-----|
| Add paper to collection | `bp s2 add` | Only S2 can modify local library |
| Find literature gaps | `bp s2 gaps` | Analyzes your collection |
| Explore without adding | `bp asta *` | Faster, read-only |
| Find text snippets in papers | `bp asta snippet` | Unique to ASTA |
| Fast paper search | `bp asta search` | 10x faster rate limit |
| Get citations/references | Either works | ASTA is faster |

**Rule of thumb**: Use `bp asta` for exploration, `bp s2` when you want to modify your library.

See [api-guide.md](api-guide.md) for detailed comparison.

## Common Workflows

### Find a Paper

1. **Search local library first**:
   ```bash
   ./bp search "Schmidler phylogenetics"
   # or for topic-heavy queries:
   ./bp semantic "importance sampling MCMC"
   ```

2. **Get PDF path** for a result:
   ```bash
   ./bp get <id>
   # pdf_path field + "/Users/matsen/Google Drive/My Drive/Paperpile"
   ```

3. **If not in library**, search externally:
   ```bash
   ./bp asta search "phylogenetic inference"
   ```

### Update Library from Paperpile

1. Export from Paperpile (JSON format) to ~/Downloads
2. Find the export file:
   ```bash
   ls -t ~/Downloads/Paperpile*.json | head -1
   ```
3. Import:
   ```bash
   ./bp import --format paperpile "<path>"
   ```
4. Optionally delete the export file after confirming success

### Explore Literature

1. **Search by topic**:
   ```bash
   ./bp asta search "variational inference phylogenetics" --limit 20
   ```

2. **Find specific text passages**:
   ```bash
   ./bp asta snippet "Bayesian phylogenetic inference"
   ```

3. **Trace citations**:
   ```bash
   ./bp asta citations DOI:10.1093/sysbio/syy032
   ./bp asta references DOI:10.1093/sysbio/syy032
   ```

4. **Add interesting papers** to your collection:
   ```bash
   ./bp s2 add DOI:10.1093/sysbio/syy032
   ```

See [workflows.md](workflows.md) for detailed workflow instructions.

## Output Format

All commands output JSON by default. Add `--human` for readable format:

```bash
./bp asta search "phylogenetics" --human
./bp s2 lookup DOI:10.1234/example --human
```

## Paper ID Formats

Both S2 and ASTA accept these identifier formats:
- `DOI:10.1093/sysbio/syy032`
- `ARXIV:2106.15928`
- `PMID:19872477`
- `CorpusId:215416146`
- Raw Semantic Scholar ID (40-char hex)
