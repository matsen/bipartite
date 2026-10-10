---
name: bip-lit
description: Unified guidance for using the bipartite reference library CLI. Use when searching for papers, managing the library, or exploring literature via S2/ASTA.
---

# Bip Reference Library

A CLI tool for managing academic references with local storage and external paper search.

**Repository**: Configured via `nexus_path` in `~/.config/bip/config.yml`

**Issues**: https://github.com/matsen/bipartite/issues

## Local-First, Paper-First Policy

**Every search starts in the local library.
When it comes up short or the search needs to be wider, go on to ASTA or S2 without asking.**

**When answering questions about papers, READ THE ACTUAL PAPER PDF.**
Do not rely on abstracts, S2 metadata, or ASTA when the paper is in the local library.
Reach for the right reader for the job:

- **Text and search → pdf-navigator MCP (MuPDF).**
  Use `search_pdf_text` to jump to the relevant pages, then `read_pdf_text` / `read_pdf_page`.
  MuPDF keeps reading order across columns and inline math intact, and the text is searchable.
  This is the default.
- **Figures, panels, rendered equations, tables → built-in `Read` with a narrow `pages` range.**
  The built-in reader renders pages as *images* (token-heavy, not searchable), so use it only for the 1–3 pages that hold the visual you need — never the whole paper.

The nexus library has ~6000 papers.
Most relevant papers are already there.

### Required Search Order

1. **Local search FIRST** (always do this):
   ```bash
   bip search -a "LastName" "keyword" --human
   ```
**Always use `--human`** — the default JSON output is verbose and easy to mis-scan.

2. **If `bip search` fails** (e.g., schema error), rebuild the database:
   ```bash
   bip rebuild
   ```
Then retry the search.

3. **If found locally, READ THE PAPER** to answer the question:
   ```bash
   bip get <id> --human   # Get PDF path
   ```
Then use `mcp__pdf-navigator__search_pdf_text` (jump to the page) and `mcp__pdf-navigator__read_pdf_text` to find the answer directly in the paper.
For a figure or equation you need to *see*, use the built-in `Read` tool on a narrow `pages` range instead.
Get the PDF base path from `bip config pdf-root --human` rather than hardcoding it — it differs per machine (a Google Drive path on macOS, an rclone mount such as `~/gdrive/Paperpile` on Linux) and `$BIP_PDF_ROOT` may override the configured value.

4. **Before concluding a paper is absent, try an exact-match check** — `bip search -a "LastName"` or `bip search --doi "..."` — rather than another keyword permutation.
   `bip search` reports when results are truncated (`Found N references (showing M; ...)`), so a plain "not found" is trustworthy, but a truncated keyword search on its own is not proof of absence.

5. **If not found locally, or the search needs to be wider**, use ASTA (`bip asta` or `mcp__asta__*`).

**DO NOT** rely on abstracts or S2 metadata when you have access to the actual paper PDF.

## Argument Handling

When invoked with arguments like `/bip-lit find <query>` or `/bip-lit <query>`:

1. **Always search local library first** with `bip search "<query>" --human`
2. If local search fails with an error, rebuild the database and retry
3. **If found locally and answering a question, read the paper PDF** using pdf-navigator tools
4. **If local options come up short**, search externally with ASTA
5. For title searches, use the full title; for topic searches, use key terms

## Commands

`bip <cmd> --help` is the reference for every flag, ID format, and output mode; this skill carries only what the help can't.

## Search Strategy

### Query Formulation Tips

**Keep queries short and specific** - long conceptual queries are a correctness hazard, not just slow: `bip search` uses exact-token AND matching, so one inflected or half-remembered word (e.g. plural vs. singular) can zero out an otherwise-matching query:
- Bad: `"correlation between BME criterion and Felsenstein likelihood around correct tree"`
- Good: `"BME Felsenstein likelihood phylogeny"` or `"Bruno WEIGHBOR likelihood"`

**Use --author flag instead of embedding names in query** - Precise last name matching:
- Good: `bip search -a "Yu" -a "Bloom" --year 2022:` (exact last name match)
- Good: `bip search -a "Tim Yu" -a "Bloom"` (first prefix + exact last name)
- Bad: `bip search "Tim Yu Bloom"` (keyword search matches tokens anywhere, not author names)

**Use specific method/algorithm names**:
- `"WEIGHBOR"`, `"FASTME"`, `"neighbor joining"` rather than general descriptions

### Systematic Search Workflow

For finding a specific paper or result:

1. **Local library first** (fastest, already curated):
   ```bash
   # Use flags for author/year filtering (most reliable)
   bip search -a "AuthorName" --year 2020: --human
   bip search "topic" -a "Author" --human

   # Or plain keyword search (use -a for authors when possible)
   bip search "distinctive title words" --human
   ```

2. **If found, read the paper** to get authoritative answers:
   ```bash
   bip get <id> --human  # Get PDF path
   # Then use pdf-navigator to search/read the PDF
   ```

3. **External keyword search** (if not found locally):
   ```bash
   bip asta search "AuthorName keyword1 keyword2" --limit 20 --human
   ```

4. **Broaden if needed** - remove author, try synonyms:
   ```bash
   bip asta search "minimum evolution likelihood" --human
   bip asta search "distance method maximum likelihood phylogeny" --human
   ```

5. **Citation tracing** - if you find a related paper, check what cites it:
   ```bash
   bip asta citations DOI:10.xxxx/yyyy --limit 50 --human
   ```

6. **MCP tools directly** - for more control over fields and filters:
   ```
   mcp__asta__search_papers_by_relevance with specific date ranges
   mcp__asta__get_citations with publication_date_range filter
   ```

### Snippet Search Caveats

The `bip asta snippet` command can be **slow and unreliable** (timeouts are common).
Alternatives:
- Use keyword search first to find candidate papers
- Use MCP `mcp__asta__snippet_search` directly with smaller limits
- If snippet times out, fall back to `bip asta search`

## S2 vs ASTA vs NCBI: When to Use Which

S2 and ASTA both access Semantic Scholar; NCBI is a separate ID-resolution service:

| Use Case | Command | Why |
|----------|---------|-----|
| Find literature gaps | `bip s2 gaps` | Analyzes your collection |
| Explore without adding | `bip asta *` | Faster, read-only |
| Find text snippets in papers | `bip asta snippet` | Unique to ASTA |
| Fast paper search | `bip asta search` | 10x faster rate limit |
| Get citations/references | Either S2 or ASTA | ASTA is faster |
| Backfill PMCIDs (e.g., for NIH RPPR) | `bip ncbi backfill` | NCBI is the canonical source; S2/ASTA do not return PMCIDs reliably |

**Rule of thumb**: Use `bip asta` for exploration, `bip s2` for lookups and collection gaps, `bip ncbi` for authoritative PMCID resolution.
NCBI only knows PMCIDs for papers actually in PMC — absence is not a signal that the paper is missing.

## Common Workflows

### Find a Paper / Answer a Question About a Paper

1. **Search local library first**:
   ```bash
   bip search "Schmidler phylogenetics" --human
   ```

2. **Get PDF path** for a result:
   ```bash
   bip get <id> --human
   # full path = `bip config pdf-root --human` + the pdf_path field
   ```

3. **Read the actual paper** to answer questions:
   ```bash
   # Text: search to the relevant page (MuPDF — searchable, clean reading order)
   mcp__pdf-navigator__search_pdf_text(file_path, "phage display")
   mcp__pdf-navigator__read_pdf_text(file_path, 2, 3)
   ```
For a **figure, panel, or rendered equation** you need to *see*, use the built-in `Read` tool on a narrow page range instead (it renders pages as images — token-heavy, so read only the page(s) with the visual):
   ```
   Read(file_path, pages="4")
   ```
**Always prefer reading the paper over relying on abstracts or external metadata.**

4. **If not in library**, search externally:
   ```bash
   bip asta search "phylogenetic inference"
   ```

### Update Library from Paperpile

Run `/bip-lit-import`.

### Want a Paper in the Library?

Give the user its DOI and ask them to add it to Paperpile. Once they have imported it, `git pull --ff-only` in the nexus repo (`nexus_path` in `~/.config/bip/config.yml`) and run `bip rebuild` to see it.

## Opening a paper page in the browser

The reliable way to open an S2 paper page in Chrome is to resolve the identifier to the 40-char paper ID, then open the **website** URL.
The `https://api.semanticscholar.org/...` redirect form is NOT reliable — Chrome often gets a JSON/non-navigable response instead of the rendered page.

```bash
# From a DOI: resolve to paperId, then open the website page
doi="10.1093/sysbio/syy032"
pid=$(curl -s "https://api.semanticscholar.org/graph/v1/paper/DOI:$doi?fields=title" \
  | sed -n 's/.*"paperId": *"\([^"]*\)".*/\1/p')
open -a "Google Chrome" "https://www.semanticscholar.org/paper/$pid"

# If you already have the 40-char SHA paper ID, open it directly:
open -a "Google Chrome" "https://www.semanticscholar.org/paper/<sha>"
```

Note: the bare `https://www.semanticscholar.org/paper/CorpusID:...` form does NOT resolve (404s) — you must use the SHA paper ID.
To resolve a CorpusId instead of a DOI, swap `DOI:$doi` above for `CorpusId:236964352`.

## Troubleshooting

### Snippet Search Timeouts

`bip asta snippet` frequently times out with "context deadline exceeded".
Workarounds:

1. **Reduce limit**: `--limit 5` instead of default
2. **Use MCP directly**: `mcp__asta__snippet_search` with small limit
3. **Fall back to keyword search**: `bip asta search` is more reliable
4. **Retry once** - sometimes it's transient

### No Results Found

If searches return nothing relevant:

1. **Check spelling** of author names and technical terms
2. **Simplify query** - fewer terms, more common synonyms
3. **Try both local and external**:
   ```bash
   bip search "topic" --human # local
   bip asta search "topic"   # external
   ```
4. **Check date filters** - paper may be too old/new for range

### Paper Not Found by ID

If `bip get <id>` or `bip asta paper <id>` fails:

1. **Verify ID format**: `DOI:10.xxxx/yyyy` (include prefix)
2. **Try alternate IDs**: Same paper may have DOI, PMID, arXiv ID
3. **Search by title instead**: `bip asta search "exact paper title"`

### PDF Missing After an Import

On a Linux rclone mount, a just-imported PDF can be absent because the mount's directory cache is stale.
Flush it with `systemctl --user kill -s HUP <mount service>` (on pax, `rclone-gdrive.service`).
If the file is still missing and `rclone lsf "<remote>:<dir>"` doesn't list it either, Paperpile hasn't synced it to Drive yet.

### SQL Schema Errors

If you see errors like `no such column: pmid` or similar schema mismatches:

```bash
bip rebuild
```

The SQLite database is ephemeral and rebuilt from the JSONL source of truth.
Schema changes require deleting and rebuilding.

### Slow Performance

- `bip s2` commands are rate-limited to 1 req/sec
- Use `bip asta` for bulk exploration (10 req/sec)
- Run searches in parallel when independent
