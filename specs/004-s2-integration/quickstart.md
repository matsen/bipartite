# Quickstart: Semantic Scholar (S2) Integration

**Feature**: 004-s2-integration
**Date**: 2026-01-19

## Prerequisites

- Bipartite repo initialized (`bip init`)
- Optional: S2 API key in `.env` as `S2_API_KEY` (higher rate limits)

## Basic Usage

### Add a Paper by DOI

```bash
# Add a paper using its DOI
bp s2 add DOI:10.1038/nature12373

# Output (JSON by default):
# {"action":"added","paper":{"id":"Kucsko2013-th","doi":"10.1038/nature12373","title":"Nanometre-scale thermometry..."}}

# Human-readable output
bp s2 add DOI:10.1093/sysbio/syy032 --human
# Output:
# Added: Zhang2018-vi
#   Title: A Variational Approach to Bayesian Phylogenetic Inference
#   Authors: Cheng Zhang, Frederick A. Matsen IV
#   Year: 2018
#   Venue: Systematic Biology
```

### Add a Paper from PDF

```bash
# Extract DOI from PDF and add
bp s2 add-pdf ~/Downloads/paper.pdf

# Link the PDF to the reference
bp s2 add-pdf ~/Downloads/paper.pdf --link
```

### Lookup Paper Info (without adding)

```bash
# Check a paper's info before adding
bp s2 lookup DOI:10.1038/nature12373

# Check specific fields
bp s2 lookup DOI:10.1038/nature12373 --fields title,citationCount,year

# Check if you already have it
bp s2 lookup DOI:10.1038/nature12373 --exists
```

## Citation Exploration

### Find Papers That Cite a Paper

```bash
# See who cited a paper in your collection
bp s2 citations Zhang2018-vi

# Only show citations you already have
bp s2 citations Zhang2018-vi --local-only

# Limit results
bp s2 citations Zhang2018-vi --limit 10 --human
```

### Find Papers Referenced by a Paper

```bash
# See what a paper cites
bp s2 references Zhang2018-vi

# Show only references you're missing
bp s2 references Zhang2018-vi --missing

# Human-readable
bp s2 references Zhang2018-vi --human
```

## Literature Discovery

### Find Literature Gaps

```bash
# Find papers cited by multiple papers in your collection
bp s2 gaps

# Only show papers cited by 3+ of your papers
bp s2 gaps --min-citations 3

# Human-readable with context
bp s2 gaps --human
# Output:
# Literature gaps (cited by 2+ papers in your collection):
#
#   Felsenstein 1981 - Maximum likelihood methods (Nature)
#     Cited by 15 papers in your collection:
#       - Zhang2018-vi, Smith2020-ab, ...
```

### Link Preprints to Published Versions

```bash
# Find preprints with published versions
bp s2 link-published --human

# Auto-link without confirmation
bp s2 link-published --auto
```

## Agent Usage (JSON Output)

All commands output JSON by default for agent consumption:

```bash
# Add and capture result
result=$(bp s2 add DOI:10.1234/example)
paper_id=$(echo "$result" | jq -r '.paper.id')

# Chain operations
bp s2 references "$paper_id" | jq '.references[] | select(.existsLocally == false)'
```

## Example Workflow: Building a Literature Review

```bash
# 1. Start with a key paper
bp s2 add DOI:10.1093/sysbio/syy032

# 2. Find what it cites
bp s2 references Zhang2018-vi --missing --human
# Shows 35 references you don't have

# 3. Add important references
bp s2 add DOI:10.1093/molbev/msx149
bp s2 add DOI:10.1093/bioinformatics/btx396

# 4. Find papers that cite your key paper
bp s2 citations Zhang2018-vi --human
# Shows recent work building on this paper

# 5. Discover gaps across your collection
bp s2 gaps --min-citations 2 --human
# Shows foundational papers you might be missing
```

## Configuration

### API Key (Optional)

For higher rate limits (1 req/sec vs 100 req/5min):

```bash
# Add to .env (already gitignored)
echo "S2_API_KEY=your_key_here" >> .env
```

### Rate Limiting

The system automatically respects S2 rate limits:
- Without API key: ~1 request per 3 seconds
- With API key: ~1 request per second

Long operations (like `bip s2 gaps`) show progress.

## Troubleshooting

### "Paper not found"

```bash
# Try different identifier formats
bp s2 lookup DOI:10.1234/example      # DOI
bp s2 lookup ARXIV:2106.15928         # arXiv
bp s2 lookup PMID:19872477            # PubMed
```

### "Rate limited"

Wait a few minutes or add an API key to `.env`.

### PDF DOI extraction fails

```bash
# Fall back to manual DOI entry
bp s2 add DOI:10.1234/example --link ~/path/to/paper.pdf
```
