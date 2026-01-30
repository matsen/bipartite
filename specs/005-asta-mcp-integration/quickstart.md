# Quickstart: ASTA MCP Integration

**Phase 1 Output** | **Date**: 2026-01-20

## Setup

### 1. Configure API Key

Add your ASTA API key to `~/.config/bip/config.json`:

```json
{
  "asta_api_key": "your-api-key-here"
}
```

### 2. Verify Installation

```bash
./bip asta --help
```

---

## Common Workflows

### Find Papers on a Topic

```bash
# Search for papers (JSON output for agents)
./bip asta search "phylogenetic inference"

# Human-readable output
./bip asta search "phylogenetic inference" --human

# Limit results and filter by year
./bip asta search "SARS-CoV-2" --limit 10 --year 2020:2024
```

### Find Specific Text in Papers

ASTA's unique snippet search finds exact text passages:

```bash
# Find papers mentioning a specific method
./bip asta snippet "variational inference phylogenetics" --human

# Search within specific papers
./bip asta snippet "mutation rate" --papers "DOI:10.1093/sysbio/syy032,ARXIV:2106.15928"
```

### Look Up Paper Details

```bash
# Get full paper info by DOI
./bip asta paper DOI:10.1093/sysbio/syy032

# Get by arXiv ID
./bip asta paper ARXIV:2106.15928 --human

# Get only specific fields
./bip asta paper DOI:10.1038/nature12373 --fields title,authors,citationCount
```

### Explore Citation Network

```bash
# Who cites this paper?
./bip asta citations DOI:10.1093/sysbio/syy032 --limit 20 --human

# What does this paper cite?
./bip asta references DOI:10.1093/sysbio/syy032 --human

# Filter citations by year
./bip asta citations DOI:10.1038/nature12373 --year 2020: --limit 50
```

### Find Authors and Their Work

```bash
# Search for an author
./bip asta author "Frederick Matsen" --human

# Get their papers (use author ID from search results)
./bip asta author-papers 1234567 --limit 50 --human

# Filter by publication year
./bip asta author-papers 1234567 --year 2020:2024
```

---

## Agent Integration

### JSON Output Parsing

All commands output JSON by default for agent consumption:

```bash
# Parse with jq
./bip asta search "machine learning" | jq '.papers[].title'

# Count results
./bip asta search "deep learning" | jq '.total'

# Extract paper IDs for further processing
./bip asta search "neural networks" | jq -r '.papers[].paperId'
```

### Combining with S2 Commands

ASTA is read-only. To add papers to your collection, pipe to S2:

```bash
# Find papers with ASTA, add to collection with S2
./bip asta search "phylogenetics bayesian" | \
  jq -r '.papers[0].paperId' | \
  xargs -I {} ./bip s2 add {}
```

### Error Handling

Check exit codes for scripting:

```bash
./bip asta paper DOI:10.1234/nonexistent
echo $?  # Returns 1 for not found

if ./bip asta paper DOI:10.1038/nature12373 > /dev/null 2>&1; then
  echo "Paper exists"
fi
```

---

## Differences from S2 Commands

| Feature | `bip s2` | `bip asta` |
|---------|---------|-----------|
| Snippet search | No | Yes |
| Add to collection | Yes | No |
| Rate limit | 1 req/sec | 10 req/sec |
| API key | `s2_api_key` | `asta_api_key` |

Use ASTA for **exploration** (search, snippets, citations).
Use S2 for **collection management** (add, update, link).
