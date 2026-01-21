# Bip Workflows

Detailed instructions for common bip workflows.

## Find Papers (bip-find)

Search for papers in the local library and return Google Drive PDF paths.

### Query Parsing

Parse user queries to identify:
- **Author names**: "Schmidler", "Mathews and Schmidler"
- **Years/ranges**: "2025", "recent", "last 2 years"
- **Topics/keywords**: "importance sampling", "B cell"
- **Combinations**: "Schmidler papers from 2025 about phylogenetics"

### Search Strategy

1. **Search the local library**:
   ```bash
   cd /Users/matsen/re/bipartite
   ./bip search "<constructed query>"
   ```

2. **For topic-heavy queries**, also try semantic search:
   ```bash
   ./bip semantic "<topic>"
   ```

3. **Filter results** by author/year criteria from the query.

### Present Results

Display results numbered, showing:
- Title
- Authors
- Year

### Handle Selection

- **Single paper** (e.g., "3"): Return its PDF path
- **Multiple papers** (e.g., "2, 4, 5" or "all"): Return all PDF paths
- **Refine search**: Help narrow down if requested

### Return PDF Paths

Combine:
- Root: `/Users/matsen/Google Drive/My Drive/Paperpile`
- Plus `pdf_path` from `./bip get <id>`

### Example Interactions

- "Schmidler" -> list all Schmidler papers, user picks subset
- "importance sampling 2025" -> papers matching both criteria
- "recent MCMC papers" -> semantic search, filtered to last 2 years

---

## If Paper NOT in Local Library

Use ASTA MCP tools (or `bip asta` commands) to search broader literature:

### Search by Title/Keyword

```bash
./bip asta search "phylogenetic inference" --human
```

Or via MCP tools:
```
mcp__asta__search_papers_by_relevance
mcp__asta__search_paper_by_title
```

### Get Verbatim Quotes (for provenance)

```bash
./bip asta snippet "exact phrase to find"
```

Or via MCP:
```
mcp__asta__snippet_search with query like "exact phrase to find"
```

### Trace Citations

```bash
./bip asta citations DOI:10.1093/sysbio/syy032
./bip asta references DOI:10.1093/sysbio/syy032
```

Or via MCP:
```
mcp__asta__get_citations
mcp__asta__get_paper (with references field)
```

### Get Paper Details

```bash
./bip asta paper DOI:10.1093/sysbio/syy032 --human
```

Or via MCP:
```
mcp__asta__get_paper with fields "title,authors,year,abstract,venue,url"
```

This is useful for:
- Finding papers not in the local library
- Tracing citation chains to establish provenance
- Getting direct quotes as evidence

---

## Update Library (bip-update)

Import references from a Paperpile export.

### Steps

1. **Find the most recent Paperpile export**:
   ```bash
   ls -t ~/Downloads/Paperpile*.json | head -1
   ```

2. **Confirm with user** which file to use (show filename and date).

3. **Run the import**:
   ```bash
   cd /Users/matsen/re/bipartite
   ./bip import --format paperpile "<path>"
   ```

4. **Report results**: Show new/updated/unchanged counts.

5. **Ask about cleanup**: Offer to remove the import file from Downloads if user wants.

---

## Explore Literature

For open-ended literature exploration without adding to your collection.

### Topic Discovery

```bash
# Search by keyword relevance
./bip asta search "variational inference" --limit 30 --human

# Filter by year
./bip asta search "deep learning phylogenetics" --year 2023:2025 --human
```

### Citation Network Exploration

```bash
# Find papers citing a foundational paper
./bip asta citations DOI:10.1093/sysbio/syy032 --limit 50 --human

# Find what a paper builds on
./bip asta references DOI:10.1093/sysbio/syy032 --human
```

### Author Exploration

```bash
# Find an author
./bip asta author "Frederick Matsen" --human

# Get their papers (use author ID from previous result)
./bip asta author-papers 145666442 --human
```

### Add Papers to Collection

When you find papers worth keeping:
```bash
./bip s2 add DOI:10.1093/sysbio/syy032
```

---

## Find Literature Gaps

Identify papers cited by your collection but not in it.

```bash
./bip s2 gaps --human
```

Review the gaps and add interesting papers:
```bash
./bip s2 add DOI:10.xxxx/yyyy
```
