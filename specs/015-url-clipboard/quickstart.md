# Quickstart: URL Output and Clipboard Support

**Feature**: 015-url-clipboard
**Date**: 2026-01-27

## Usage Examples

### Get DOI URL (Default)

```bash
# Output DOI URL to stdout
bip url Smith2024-ab
# Output: https://doi.org/10.1234/example
```

### Copy to Clipboard

```bash
# Copy URL to clipboard and print
bip url Smith2024-ab --copy
# stdout: https://doi.org/10.1234/example
# stderr: Copied to clipboard
```

### Alternative URL Formats

```bash
# PubMed
bip url Smith2024-ab --pubmed
# Output: https://pubmed.ncbi.nlm.nih.gov/12345678/

# PubMed Central
bip url Smith2024-ab --pmc
# Output: https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1234567/

# arXiv
bip url Smith2024-ab --arxiv
# Output: https://arxiv.org/abs/2106.15928

# Semantic Scholar
bip url Smith2024-ab --s2
# Output: https://www.semanticscholar.org/paper/649def34f8be52c8b66281af98ae884c09aef38b
```

### JSON Output

```bash
bip url Smith2024-ab --json
# Output: {"url":"https://doi.org/10.1234/example","format":"doi","copied":false}

bip url Smith2024-ab --copy --json
# Output: {"url":"https://doi.org/10.1234/example","format":"doi","copied":true}
```

### Pipeline Usage

```bash
# URL only goes to stdout, composable with other tools
bip url Smith2024-ab --copy | xargs open

# Get multiple URLs
for id in Smith2024-ab Jones2023-cd; do
  bip url "$id"
done
```

## Error Handling

### Reference Not Found

```bash
bip url NonExistent2024-xx
# Error: reference not found: NonExistent2024-xx
```

### Missing ID Type

```bash
bip url Smith2024-ab --pubmed
# Error: no PubMed ID available for Smith2024-ab
```

### Multiple Format Flags

```bash
bip url Smith2024-ab --pubmed --arxiv
# Error: specify only one URL format flag (--pubmed, --pmc, --arxiv, or --s2)
```

### Clipboard Unavailable

```bash
# On headless server without xclip/xsel
bip url Smith2024-ab --copy
# stdout: https://doi.org/10.1234/example
# stderr: Warning: clipboard unavailable (install xclip or xsel on Linux)
```

## Command Reference

```
bip url <ref-id> [flags]

Flags:
  --copy      Copy URL to system clipboard
  --pubmed    Output PubMed URL instead of DOI
  --pmc       Output PubMed Central URL instead of DOI
  --arxiv     Output arXiv URL instead of DOI
  --s2        Output Semantic Scholar URL instead of DOI
  --json      Output as JSON (default behavior unless -H flag)
  -H          Human-readable output
  -h, --help  Help for url

Examples:
  bip url Ahn2026-rs              # Get DOI URL
  bip url Ahn2026-rs --copy       # Copy DOI URL to clipboard
  bip url Ahn2026-rs --pubmed     # Get PubMed URL
```

## Setup Notes

### Linux Clipboard Support

For `--copy` to work on Linux, install one of:

```bash
# Debian/Ubuntu
sudo apt install xclip
# or
sudo apt install xsel

# Fedora
sudo dnf install xclip
```

macOS has built-in clipboard support via `pbcopy`.

### After Schema Change

When updating an existing bipartite installation, rebuild the database:

```bash
rm .bipartite/cache/refs.db
bip rebuild
```

This rebuilds the SQLite index with the new external ID columns.
