# Quickstart: Shared Repository Workflow Commands

## Prerequisites

- Go 1.25.5+ installed
- Git available on PATH
- Existing bipartite repository with papers
- PDF root configured (`bip config pdf-root`)

## Build

```bash
go build -o bip ./cmd/bip
```

## Common Workflows

### 1. Review Papers After Pulling Updates

After `git pull` from a shared repository, see what's new:

```bash
# See what changed (uncommitted)
./bip diff --human

# See papers added since your last commit
./bip new --since HEAD~1 --human

# Open all new papers to review
./bip open --since HEAD~1
```

### 2. Quick-Review Recently Added Papers

Your agent added papers overnight. Open the most recent ones:

```bash
# Open 5 most recent papers
./bip open --recent 5

# Or see what was added in last 3 days
./bip new --days 3 --human
```

### 3. Export Paper for Manuscript Citation

Your agent identified a paper to cite. Export it to your .bib file:

```bash
# Export single paper to stdout
./bip export --bibtex Smith2024-ab

# Append to existing .bib file (won't duplicate)
./bip export --bibtex --append ~/paper/refs.bib Smith2024-ab Lee2024-cd
```

### 4. Check Uncommitted Changes Before Commit

Before committing changes to the shared repository:

```bash
# See what will be committed
./bip diff --human

# Review the papers being added
./bip open Smith2024-ab Lee2024-cd
```

## Testing Commands

### Test `bip open` with Multiple Papers

```bash
# Requires: at least 2 papers in library with PDFs
./bip open $(./bip list | jq -r '.[0:2][].id')

# Expected: 2 PDF viewers open
```

### Test `bip diff`

```bash
# Add a paper, don't commit
./bip s2 add DOI:10.1234/example

# Show uncommitted changes
./bip diff --human

# Expected: Shows 1 paper added
```

### Test `bip new --since`

```bash
# Get current commit
BEFORE=$(git rev-parse HEAD)

# Add and commit a paper
./bip s2 add DOI:10.1234/example
git add .bipartite/refs.jsonl
git commit -m "Add test paper"

# Show papers since before
./bip new --since $BEFORE --human

# Expected: Shows 1 paper added
```

### Test `bip export --bibtex --append`

```bash
# Create empty .bib file
echo "" > /tmp/test.bib

# Export a paper
./bip export --bibtex --append /tmp/test.bib Smith2024-ab

# Export same paper again (should skip)
./bip export --bibtex --append /tmp/test.bib Smith2024-ab

# Expected: Second call shows skipped=1
```

## Edge Cases to Test

1. **Missing PDF**: `bip open` with paper that has no PDF
2. **Non-existent commit**: `bip new --since xyz789` (should error)
3. **Empty diff**: `bip diff` when no changes (should show empty arrays)
4. **Duplicate detection**: `bip export --append` with existing DOI

## JSON vs Human Output

All commands default to JSON for agent integration:

```bash
# JSON (default)
./bip diff
# {"added": [...], "removed": [...]}

# Human-readable
./bip diff --human
# Added (2):
#   + Smith2024-ab: ...
```
