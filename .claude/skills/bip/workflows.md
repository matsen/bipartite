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
   bip search "<constructed query>"
   ```

2. **For topic-heavy queries**, also try semantic search:
   ```bash
   bip semantic "<topic>"
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
- Plus `pdf_path` from `bip get <id>`

### Example Interactions

- "Schmidler" -> list all Schmidler papers, user picks subset
- "importance sampling 2025" -> papers matching both criteria
- "recent MCMC papers" -> semantic search, filtered to last 2 years

---

## If Paper NOT in Local Library

**⚠️ STOP: Ask the user before proceeding with external search.**

Say: "I couldn't find that paper in the local library. Would you like me to search Semantic Scholar (ASTA)?"

Only after user confirms, use ASTA MCP tools (or `bip asta` commands) to search broader literature:

### Search by Title/Keyword

```bash
bip asta search "phylogenetic inference" --human
```

Or via MCP tools:
```
mcp__asta__search_papers_by_relevance
mcp__asta__search_paper_by_title
```

### Get Verbatim Quotes (for provenance)

```bash
bip asta snippet "exact phrase to find"
```

Or via MCP:
```
mcp__asta__snippet_search with query like "exact phrase to find"
```

### Trace Citations

```bash
bip asta citations DOI:10.1093/sysbio/syy032
bip asta references DOI:10.1093/sysbio/syy032
```

Or via MCP:
```
mcp__asta__get_citations
mcp__asta__get_paper (with references field)
```

### Get Paper Details

```bash
bip asta paper DOI:10.1093/sysbio/syy032 --human
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
   bip import --format paperpile "<path>"
   ```

4. **Report results**: Show new/updated/unchanged counts.

5. **Ask about cleanup**: Offer to remove the import file from Downloads if user wants.

---

## Finding a Specific Paper ("Needle in Haystack")

When you know a paper exists and need to find it:

### Strategy: Narrow to Broad

1. **If you know authors**: Start with author name + key term
   ```bash
   bip search "Felsenstein distance"
   bip asta search "Bruno Halpern neighbor joining" --human
   ```

2. **If you know the method/algorithm**: Use specific names
   ```bash
   bip asta search "WEIGHBOR likelihood" --human
   bip asta search "FASTME balanced minimum evolution" --human
   ```

3. **Broaden progressively**: Remove constraints one at a time
   ```bash
   # Start specific
   bip asta search "Gascuel BME likelihood correlation" --human
   # Remove one term
   bip asta search "Gascuel BME likelihood" --human
   # Try synonyms
   bip asta search "minimum evolution maximum likelihood" --human
   ```

4. **Citation chain**: Find a known related paper, trace its citations
   ```bash
   # Find the foundational paper
   bip asta search "Desper Gascuel BME" --human
   # See what cites it
   bip asta citations DOI:10.1093/molbev/msh049 --limit 50 --human
   ```

### When You Can't Find It

The paper might:
- Use different terminology (try synonyms)
- Be too old to be well-indexed
- Not exist (the result you're looking for may not have been published)
- Be in a thesis/preprint not indexed by Semantic Scholar

---

## Explore Literature

For open-ended literature exploration without adding to your collection.

### Topic Discovery

```bash
# Search by keyword relevance
bip asta search "variational inference" --limit 30 --human

# Filter by year
bip asta search "deep learning phylogenetics" --year 2023:2025 --human
```

### Citation Network Exploration

```bash
# Find papers citing a foundational paper
bip asta citations DOI:10.1093/sysbio/syy032 --limit 50 --human

# Find what a paper builds on
bip asta references DOI:10.1093/sysbio/syy032 --human
```

### Author Exploration

```bash
# Find an author
bip asta author "Frederick Matsen" --human

# Get their papers (use author ID from previous result)
bip asta author-papers 145666442 --human
```

### Add Papers to Collection

When you find papers worth keeping:
```bash
bip s2 add DOI:10.1093/sysbio/syy032
```

---

## Find Literature Gaps

Identify papers cited by your collection but not in it.

```bash
bip s2 gaps --human
```

Review the gaps and add interesting papers:
```bash
bip s2 add DOI:10.xxxx/yyyy
```

---

## Get Paper URLs

Get a URL for any paper in your library and optionally copy it to clipboard.

### Basic URL Retrieval

```bash
# Get DOI URL (default)
bip url Smith2024-ab --human
# Output: https://doi.org/10.1234/example

# Copy to clipboard
bip url Smith2024-ab --copy --human
# Output: https://doi.org/10.1234/example
# Copied to clipboard
```

### Alternative URL Formats

Papers imported via S2 have multiple external identifiers:

```bash
# PubMed URL
bip url Smith2024-ab --pubmed --human

# PubMed Central URL
bip url Smith2024-ab --pmc --human

# arXiv URL
bip url Smith2024-ab --arxiv --human

# Semantic Scholar URL
bip url Smith2024-ab --s2 --human
```

### JSON Output (for Agents)

```bash
# Default JSON output
bip url Smith2024-ab
# {"url":"https://doi.org/10.1234/example","format":"doi","copied":false}

# With copy
bip url Smith2024-ab --copy
# {"url":"https://doi.org/10.1234/example","format":"doi","copied":true}
```

### Pipeline Usage

URL goes to stdout, messages to stderr for easy piping:

```bash
# Copy and open in browser
bip url Smith2024-ab --copy --human | xargs open

# Get multiple URLs
for id in Smith2024-ab Jones2023-cd; do
  bip url "$id" --human
done
```

### Linux Clipboard Support

On Linux, install `xclip` or `xsel` for clipboard support:
```bash
sudo apt install xclip  # or xsel
```

If clipboard is unavailable, the command still outputs the URL with a warning.
