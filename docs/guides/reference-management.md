# Reference Management

Bipartite replaces GUI reference managers with a CLI that both humans and agents can use. Your paper library lives in git-versionable JSONL, not a proprietary database.

## Getting Started

Start from the [nexus-template](https://github.com/matsen/nexus-template), then configure and import:

```bash
bip config pdf-root ~/Google\ Drive/My\ Drive/Paperpile
bip import --format paperpile ~/Downloads/export.json
bip rebuild
```

`bip rebuild` builds the SQLite query index from the JSONL source files. Run it after pulling changes or if the database gets corrupted.

## Searching

```bash
bip search "phylogenetics"
bip search "author:Matsen"
bip search "title:influenza" --limit 10
```

Keyword search queries title, abstract, and authors. Use `author:` or `title:` prefixes to narrow scope.

### Semantic Search

For conceptual queries that go beyond keyword matching:

```bash
bip index build                  # Build the semantic index (requires Ollama)
bip semantic "methods for tree inference"
bip similar Zhang2018-vi         # Find papers similar to a specific paper
```

Semantic search uses local embeddings via Ollama to find related papers even without exact word matches.

## Working with Papers

```bash
bip get Smith2024-ab             # Fetch metadata as JSON
bip get Smith2024-ab --human     # Human-readable output
bip open Smith2024-ab            # Open PDF in configured viewer
bip open --recent 5              # Open the 5 most recently added papers
bip open --since HEAD~3          # Open papers added in last 3 commits
bip url Smith2024-ab             # Get DOI URL
bip url Smith2024-ab --copy      # Copy URL to clipboard
bip url Smith2024-ab --arxiv     # Get arXiv URL instead
```

`bip open` supports supplementary PDFs with `--supplement N`.

`bip url` can output DOI, PubMed, PubMed Central, arXiv, or Semantic Scholar URLs.

## Adding Papers via Semantic Scholar

The `bip s2` commands fetch metadata from Semantic Scholar's Academic Graph API:

```bash
bip s2 add DOI:10.1038/nature12373
bip s2 add ARXIV:2106.15928 --link ~/papers/paper.pdf
bip s2 lookup DOI:10.1093/sysbio/syy032   # Preview without adding
```

Supported ID formats: `DOI:`, `ARXIV:`, `PMID:`, `PMCID:`, `CorpusId:`, or a raw S2 paper ID.

Use `--link` to associate a local PDF path when adding. Use `--update` to refresh metadata for a paper already in the collection.

## Exploring Citations

```bash
bip s2 citations Zhang2018-vi              # Who cites this paper?
bip s2 citations Zhang2018-vi --local-only # Only show citing papers already in your collection
bip s2 references Zhang2018-vi             # What does this paper cite?
bip s2 references Zhang2018-vi --missing   # Only references NOT in your collection
bip s2 gaps                                # Highly-cited papers you're missing
bip s2 gaps --min-citations 3              # Require at least 3 local citations
```

`bip s2 gaps` is particularly useful: it finds papers cited by multiple papers in your collection that you haven't added yet — likely foundational work you should know about.

## Exporting

```bash
bip export --bibtex                                    # Export all papers
bip export --bibtex Smith2024-ab Lee2024-cd            # Export specific papers
bip export --bibtex --append refs.bib Smith2024-ab     # Append with deduplication
```

## Collaboration

The library is designed for multi-user workflows via git:

```bash
git pull                  # Get collaborator changes
bip rebuild               # Rebuild index from updated JSONL
bip new --days 7          # What's been added recently?
bip new --since HEAD~3    # Papers added in last 3 commits
bip diff                  # Uncommitted additions/removals
```

When merges produce conflicts in `refs.jsonl`:

```bash
bip resolve --dry-run     # Preview resolution
bip resolve               # Auto-resolve using paper metadata
bip resolve --interactive # Prompt for true conflicts
```

`bip resolve` understands that DOI is a unique identifier, that longer author lists are better, and that more complete metadata should win.

## Maintenance

```bash
bip dedupe --dry-run      # Find duplicates by source ID
bip dedupe --merge        # Merge duplicates, keeping first and updating edges
bip check                 # Verify repository integrity
```

## Agent Usage

All commands output JSON by default. Agents call them via bash — no MCP server needed:

```bash
bip search "variational inference" --limit 5
bip get Smith2024-ab
bip s2 add DOI:10.1038/s41586-024-07487-w
```

Add `--human` for human-readable output in any command.
