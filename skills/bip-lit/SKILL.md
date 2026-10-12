---
name: bip-lit
description: Unified guidance for using the bipartite reference library CLI. Use when searching for papers, reading or summarizing one (shared notes in nexus papers/), managing the library, or exploring literature via S2/ASTA.
---

# Bip Reference Library

The library (~6000 papers) lives in the nexus repo (`nexus_path` in `~/.config/bip/config.yml`); most relevant papers are already there.
`bip <cmd> --help` is the reference for flags, ID formats, and output modes; this skill carries only what the help can't.
An argument (`/bip-lit <query>`, or `/bip-lit find <query>` with `find` dropped) is the query for the steps below.

## Find a paper, answer a question about one

1. **Search locally first**, always with `--human` (the default JSON is easy to mis-scan):
   ```bash
   bip search "BME Felsenstein likelihood" -a "Gascuel" --human
   ```
   If it errors (e.g. `no such column`), run `bip rebuild` and retry; the SQLite layer is rebuilt from JSONL.
2. **Found it → read the PDF**, not the abstract or S2/ASTA metadata.
   The path is `bip config pdf-root --human` (per machine; `$BIP_PDF_ROOT` overrides) plus the `PDF:` field of `bip get <id> --human`.
   Read the `Supplements:` it lists too: methods numbers (data size, training budget, per-condition tables) are often only there.
   A listed PDF missing from disk is usually the Linux rclone mount's stale directory cache: `systemctl --user kill -s HUP <mount service>` (on pax, `rclone-gdrive.service`), then look again; an `Input/output error` on the first read after a HUP is transient, so retry once.
   If `rclone lsf "<remote>:<dir>"` doesn't list it either, or the paper isn't in the library, read a legal open-access copy: `bip s2 lookup <id> --fields openAccessPdf` gives its URL and status, and an arXiv preprint is at `https://arxiv.org/pdf/<id>`.
   Note which version you read (published, author manuscript, preprint) and cite its sections rather than journal pages; this is for reading only, and Paperpile stays the library of record.
   Don't use a lookup service that needs the user's email address (Unpaywall does); S2 already carries the same pointer.
   Fall back to the abstract only if neither exists, say the note is abstract-only, and give the user the DOI link to add the paper.
   - Text: pdf-navigator MCP — `search_pdf_text` to find the page, then `read_pdf_text`. Keeps column order and inline math, and is searchable.
   - Figures, panels, rendered equations, tables: built-in `Read` on the 1–3 pages that hold them. It renders pages as images, so never the whole paper.
   If `bip get` shows `Paper notes:`, read that shared nexus file first and go to the PDF only for what it doesn't answer.
   Whatever you learn from the paper goes into `nexus/papers/<id>.md` (create it if absent; format in `papers/README.md` there), not only into your project's notes; commit and push nexus.
3. **Before concluding it's absent**, run an exact-match check (`-a "LastName"` or `--doi`) rather than another keyword permutation.
   No truncation notice (`showing M`) means you saw every match for those tokens, not that the paper is absent.
4. **Not found, or the search needs to be wider → ASTA**, without asking: `bip asta search` or `mcp__asta__*`.
   Broaden by dropping terms and trying synonyms, then trace citations of a related paper (`bip asta citations <id>`).

### Writing queries

`bip search` is exact-token AND matching, unstemmed, so one inflected or half-remembered word zeroes out a query.
Keep queries short and use distinctive method names (`"WEIGHBOR"`, `"FASTME"`), not descriptions.
In `bip search`, put people in `-a`, not the query: `"Tim Yu Bloom"` matches those tokens in any field, while `-a "Yu" -a "Bloom"` matches last names.

## S2, ASTA, NCBI

- `bip asta` for exploration: read-only, 10 req/s, and the only one with snippet search.
- `bip s2` for lookups and `bip s2 gaps` (papers your collection cites but lacks); 1 req/s with the key, ~0.33 without.
- `bip ncbi` for PMCIDs (e.g. NIH RPPR); S2 and ASTA don't return them reliably. NCBI only knows papers in PMC, so a missing PMCID doesn't mean a missing paper.

`bip asta snippet` often times out ("context deadline exceeded"): use `--limit 5`, or `mcp__asta__snippet_search` with a small limit, retry once, then fall back to `bip asta search`.

## Adding to the library

Give the user the DOI as a `https://doi.org/<doi>` link, never bare, and ask them to add it to Paperpile; `bip s2 add` changes the library and needs the user's word.
Once they've imported it, `git pull --ff-only` in the nexus repo and `bip rebuild`.
To import a Paperpile export, run `/bip-lit-import`.

## Opening an S2 paper page in a browser

Resolve the ID to the 40-char S2 paper ID and open the website URL.
The `api.semanticscholar.org` redirect often gives the browser JSON, and `semanticscholar.org/paper/CorpusID:...` 404s.

```bash
pid=$(curl -s "https://api.semanticscholar.org/graph/v1/paper/DOI:10.1093/sysbio/syy032?fields=title" \
  | sed -n 's/.*"paperId": *"\([^"]*\)".*/\1/p')   # or CorpusId:236964352
xdg-open "https://www.semanticscholar.org/paper/$pid"   # macOS: open
```
