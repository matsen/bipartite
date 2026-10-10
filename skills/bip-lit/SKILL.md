---
name: bip-lit
description: Unified guidance for using the bipartite reference library CLI. Use when searching for papers, managing the library, or exploring literature via S2/ASTA.
---

# Bip Reference Library

The library (~6000 papers) lives in the nexus repo (`nexus_path` in `~/.config/bip/config.yml`); most relevant papers are already there.
`bip <cmd> --help` is the reference for flags, ID formats, and output modes; this skill carries only what the help can't.
An argument (`/bip-lit <query>`) is the query for the steps below.

## Find a paper, answer a question about one

1. **Search locally first**, always with `--human` (the default JSON is easy to mis-scan):
   ```bash
   bip search "BME Felsenstein likelihood" -a "Gascuel" --human
   ```
   If it errors (e.g. `no such column`), run `bip rebuild` and retry; the SQLite layer is rebuilt from JSONL.
2. **Found it → read the PDF**, not the abstract or S2/ASTA metadata.
   The path is `bip config pdf-root --human` (per machine; `$BIP_PDF_ROOT` overrides) plus `pdf_path` from `bip get <id> --human`.
   - Text: pdf-navigator MCP — `search_pdf_text` to find the page, then `read_pdf_text`. Keeps column order and inline math, and is searchable.
   - Figures, panels, rendered equations, tables: built-in `Read` on the 1–3 pages that hold them. It renders pages as images, so never the whole paper.
3. **Before concluding it's absent**, run an exact-match check (`-a "LastName"` or `--doi`) rather than another keyword permutation.
   A plain "not found" is trustworthy; a search that reports truncation (`showing M`) is not proof of absence.
4. **Not found, or the search needs to be wider → ASTA**, without asking: `bip asta search` or `mcp__asta__*`.
   Broaden by dropping terms and trying synonyms, then trace citations of a related paper (`bip asta citations <id>`).

### Writing queries

`bip search` is exact-token AND matching, unstemmed, so one inflected or half-remembered word zeroes out a query.
Keep queries short and use distinctive method names (`"WEIGHBOR"`, `"FASTME"`), not descriptions.
Put people in `-a`, not the query: `"Tim Yu Bloom"` matches those tokens in any field, while `-a "Yu" -a "Bloom"` matches last names.

## S2, ASTA, NCBI

- `bip asta` for exploration: read-only, 10 req/s, and the only one with snippet search.
- `bip s2` for lookups and `bip s2 gaps` (papers your collection cites but lacks); 1 req/s.
- `bip ncbi` for PMCIDs (e.g. NIH RPPR); S2 and ASTA don't return them reliably. NCBI only knows papers in PMC, so a missing PMCID doesn't mean a missing paper.

`bip asta snippet` often times out ("context deadline exceeded"): use `--limit 5`, or `mcp__asta__snippet_search` with a small limit, retry once, then fall back to `bip asta search`.

## Adding to the library

Give the user the DOI and ask them to add it to Paperpile; `bip s2 add` changes the library and needs the user's word.
Once they've imported it, `git pull --ff-only` in the nexus repo and `bip rebuild`.
To import a Paperpile export, run `/bip-lit-import`.

A just-imported PDF can be missing because the Linux rclone mount's directory cache is stale: `systemctl --user kill -s HUP <mount service>` (on pax, `rclone-gdrive.service`).
If `rclone lsf "<remote>:<dir>"` doesn't list it either, Paperpile hasn't synced it to Drive yet.

## Opening an S2 paper page in a browser

Resolve the ID to the 40-char S2 paper ID and open the website URL.
The `api.semanticscholar.org` redirect often gives the browser JSON, and `semanticscholar.org/paper/CorpusID:...` 404s.

```bash
pid=$(curl -s "https://api.semanticscholar.org/graph/v1/paper/DOI:10.1093/sysbio/syy032?fields=title" \
  | sed -n 's/.*"paperId": *"\([^"]*\)".*/\1/p')   # or CorpusId:236964352
open -a "Google Chrome" "https://www.semanticscholar.org/paper/$pid"
```
