# S2 vs ASTA API Guide

Both `bp s2` and `bp asta` commands access Semantic Scholar's paper database, but through different APIs with different capabilities.

## Quick Comparison

| Feature | `bp s2` | `bp asta` |
|---------|---------|-----------|
| API | Semantic Scholar REST | Allen AI ASTA MCP |
| Rate limit | 1 req/sec | 10 req/sec |
| Snippet search | No | Yes |
| Add to collection | Yes | No (read-only) |
| Auth | S2_API_KEY | ASTA_API_KEY |

## When to Use S2

Use `bp s2` when you need to **modify your local collection**:

```bash
# Add a paper to your collection
./bp s2 add DOI:10.1093/sysbio/syy032

# Look up paper info (slower, 1 req/sec)
./bp s2 lookup DOI:10.1093/sysbio/syy032

# Find citing papers
./bp s2 citations <paper-id>

# Find referenced papers
./bp s2 references <paper-id>

# Find literature gaps (papers cited by collection but not in it)
./bp s2 gaps
```

**Key S2 capabilities**:
- `bp s2 add` - Add papers to your local collection
- `bp s2 gaps` - Analyze your collection for missing papers

## When to Use ASTA

Use `bp asta` for **fast, read-only exploration**:

```bash
# Fast paper search (10x faster rate limit)
./bp asta search "phylogenetic inference" --limit 20 --human

# UNIQUE: Search text snippets within papers
./bp asta snippet "variational inference" --human

# Get paper details
./bp asta paper DOI:10.1093/sysbio/syy032 --human

# Get citations (faster than S2)
./bp asta citations DOI:10.1093/sysbio/syy032 --human

# Get references
./bp asta references DOI:10.1093/sysbio/syy032 --human

# Search for authors
./bp asta author "Frederick Matsen" --human

# Get author's papers
./bp asta author-papers 145666442 --human
```

**Key ASTA capabilities**:
- `bp asta snippet` - Search actual text within papers (unique to ASTA)
- 10x faster rate limit for bulk exploration
- Author search functionality

## ASTA MCP Tools

When using Claude Code, you can also access ASTA directly via MCP tools:

| MCP Tool | Equivalent CLI |
|----------|---------------|
| `mcp__asta__search_papers_by_relevance` | `bp asta search` |
| `mcp__asta__search_paper_by_title` | `bp asta search` (by title) |
| `mcp__asta__snippet_search` | `bp asta snippet` |
| `mcp__asta__get_paper` | `bp asta paper` |
| `mcp__asta__get_citations` | `bp asta citations` |
| `mcp__asta__search_authors_by_name` | `bp asta author` |
| `mcp__asta__get_author_papers` | `bp asta author-papers` |

## Paper ID Formats

Both APIs accept the same identifier formats:

| Format | Example |
|--------|---------|
| DOI | `DOI:10.1093/sysbio/syy032` |
| arXiv | `ARXIV:2106.15928` |
| PubMed | `PMID:19872477` |
| PubMed Central | `PMCID:2323736` |
| Corpus ID | `CorpusId:215416146` |
| Raw S2 ID | `649def34f8be52c8b66281af98ae884c09aef38b` |
| URL | `URL:https://arxiv.org/abs/2106.15928v1` |

## Output Format

Both command families output JSON by default. Add `--human` for readable output:

```bash
./bp s2 lookup DOI:10.1093/sysbio/syy032 --human
./bp asta search "phylogenetics" --human
```

## Environment Variables

```bash
# In .env file
S2_API_KEY=your_s2_api_key      # For bp s2 commands
ASTA_API_KEY=your_asta_api_key  # For bp asta commands
```

## Decision Flowchart

```
Want to modify your collection?
├── Yes → Use bp s2
│   ├── Add paper → bp s2 add
│   └── Find gaps → bp s2 gaps
└── No (read-only exploration)
    ├── Need text snippets? → bp asta snippet
    ├── Bulk search? → bp asta search (faster)
    ├── Author info? → bp asta author
    └── Citations/refs → bp asta (faster)
```
