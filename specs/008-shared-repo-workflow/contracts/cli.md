# CLI Contract: Shared Repository Workflow Commands

## Overview

This document specifies the CLI interface for the shared repository workflow commands. All commands follow existing bip conventions:
- JSON output by default
- `--human` flag for readable output
- Exit codes per `cmd/bip/exitcodes.go`

---

## bip open (Enhanced)

### Synopsis
```
bip open <id>... [flags]
bip open --recent N [flags]
bip open --since <commit> [flags]
```

### Description
Open one or more papers' PDFs in the configured viewer. Supports opening multiple papers by ID, the N most recently added papers, or papers added since a specific git commit.

### Arguments
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `<id>...` | string | Conditional | One or more paper IDs (mutually exclusive with --recent/--since) |

### Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--recent` | int | 0 | Open the N most recently added papers |
| `--since` | string | "" | Open papers added after this git commit |
| `--supplement` | int | 0 | Open Nth supplementary PDF (1-indexed) |
| `--human` | bool | false | Human-readable output |

### Mutual Exclusivity
- Positional IDs, `--recent`, and `--since` are mutually exclusive
- `--supplement` only valid with single ID

### Exit Codes
| Code | Condition |
|------|-----------|
| 0 | At least one PDF opened successfully |
| 1 | No papers found, all PDFs missing, or other error |
| 2 | Configuration error (pdf_root not set) |

### JSON Output
```json
{
  "opened": [
    {"id": "Smith2024-ab", "path": "/full/path/to/paper.pdf"}
  ],
  "errors": [
    {"id": "Jones2023-xy", "error": "PDF not found: papers/jones2023.pdf"}
  ]
}
```

### Human Output
```
Opening 3 papers:
  ✓ Smith2024-ab: papers/smith2024.pdf
  ✓ Lee2024-cd: papers/lee2024.pdf
  ✗ Jones2023-xy: PDF not found
```

### Examples
```bash
# Open specific papers
bip open Smith2024-ab Jones2023-xy Lee2024-cd

# Open 5 most recent papers
bip open --recent 5

# Open papers added since last pull
bip open --since HEAD~3

# Open papers since a specific commit
bip open --since abc123f
```

---

## bip diff

### Synopsis
```
bip diff [flags]
```

### Description
Show papers added or removed in the working tree compared to the last commit. Useful for reviewing uncommitted changes before committing.

### Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--human` | bool | false | Human-readable output |

### Exit Codes
| Code | Condition |
|------|-----------|
| 0 | Success (even if no changes) |
| 1 | Error reading refs or git state |

### JSON Output
```json
{
  "added": [
    {"id": "Smith2024-ab", "title": "A New Method...", "authors": "Smith, Jones", "year": 2024}
  ],
  "removed": [
    {"id": "Old2020-xy", "title": "Deprecated Paper", "authors": "Old", "year": 2020}
  ]
}
```

### Human Output
```
Changes since last commit:

Added (2):
  + Smith2024-ab: A New Method for... (Smith, Jones, 2024)
  + Lee2024-cd: Deep Learning in... (Lee et al., 2024)

Removed (1):
  - Old2020-xy: Deprecated Paper (Old, 2020)
```

### Examples
```bash
# Show uncommitted changes (JSON)
bip diff

# Show uncommitted changes (human-readable)
bip diff --human
```

---

## bip new

### Synopsis
```
bip new --since <commit> [flags]
bip new --days N [flags]
```

### Description
List papers added to the repository since a specific commit or within the last N days. Useful for tracking new additions from collaborators after pulling updates.

### Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | "" | List papers added after this git commit |
| `--days` | int | 0 | List papers added within last N days (UTC) |
| `--human` | bool | false | Human-readable output |

### Mutual Exclusivity
- `--since` and `--days` are mutually exclusive
- One of them must be provided

### Exit Codes
| Code | Condition |
|------|-----------|
| 0 | Success (even if no new papers) |
| 1 | Invalid commit reference or other error |

### JSON Output
```json
{
  "papers": [
    {
      "id": "Smith2024-ab",
      "title": "A New Method for Phylogenetic Analysis",
      "authors": "Smith, Jones",
      "year": 2024,
      "commit_sha": "abc123f"
    }
  ],
  "since_ref": "def456a",
  "total_count": 1
}
```

### Human Output
```
Papers added since def456a:

  Smith2024-ab: A New Method for Phylogenetic Analysis
    Smith, Jones (2024)
    Added in commit abc123f

  Lee2024-cd: Deep Learning Approaches to...
    Lee, Kim, Park (2024)
    Added in commit abc123f

2 papers added
```

### Examples
```bash
# Papers since a specific commit
bip new --since abc123f

# Papers added in last 7 days
bip new --days 7

# Papers since last week (human-readable)
bip new --days 7 --human
```

---

## bip export --bibtex (Enhanced)

### Synopsis
```
bip export --bibtex [<id>...] [flags]
bip export --bibtex --append <file> [<id>...] [flags]
```

### Description
Export papers to BibTeX format. Can export all papers, specific papers by ID, or append to an existing .bib file with automatic deduplication.

### Arguments
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `<id>...` | string | No | Paper IDs to export (default: all papers) |

### Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--bibtex` | bool | false | Export to BibTeX format (required) |
| `--append` | string | "" | Append to existing .bib file (with deduplication) |
| `--keys` | string | "" | Legacy: comma-separated IDs (deprecated, use positional args) |

### Exit Codes
| Code | Condition |
|------|-----------|
| 0 | Success |
| 1 | Paper not found, file error, or other error |

### Output Behavior
- **Without --append**: BibTeX written to stdout (no JSON wrapper)
- **With --append**: JSON response with statistics

### JSON Output (with --append)
```json
{
  "exported": 3,
  "skipped": 1,
  "skipped_ids": ["Smith2024-ab"],
  "output_path": "/path/to/refs.bib"
}
```

### BibTeX Output (without --append)
```bibtex
@article{Smith2024-ab,
  author = {Smith, John and Jones, Jane},
  title = {A New Method for Phylogenetic Analysis},
  journal = {Systematic Biology},
  year = {2024},
  doi = {10.1093/sysbio/example},
}

@article{Lee2024-cd,
  ...
}
```

### Deduplication Rules (--append mode)
1. Match by DOI (primary): Skip if DOI exists in .bib file
2. Match by citation key (fallback): Skip if key exists and entry has no DOI
3. Entries without DOI in library are deduplicated by key only

### Examples
```bash
# Export all papers to stdout
bip export --bibtex > refs.bib

# Export specific papers
bip export --bibtex Smith2024-ab Lee2024-cd

# Append to existing file with deduplication
bip export --bibtex --append refs.bib Smith2024-ab Lee2024-cd

# Export single paper for citation
bip export --bibtex Smith2024-ab
```

---

## Error Messages

All error messages follow FR-019 (actionable with resolution hints).

### bip open
```
error: reference not found: Smith2024-ab
  Hint: Use 'bip list' to see available references

error: pdf_root not configured
  Hint: Use 'bip config pdf-root /path/to/pdfs' to set the PDF directory

error: commit not found: xyz789
  Hint: Verify the commit exists with 'git log --oneline'
```

### bip diff
```
error: not in a git repository
  Hint: Initialize with 'git init' or navigate to a git repository

error: refs.jsonl not tracked by git
  Hint: Run 'git add .bipartite/refs.jsonl' to track the file
```

### bip new
```
error: commit not found: xyz789
  Hint: Verify the commit exists with 'git log --oneline'

error: --since or --days flag required
  Hint: Use 'bip new --since <commit>' or 'bip new --days N'
```

### bip export
```
error: unknown key: Smith2024-ab
  Hint: Use 'bip list' to see available references

error: cannot read file: refs.bib
  Hint: Check file exists and has read permissions

error: cannot write to file: refs.bib
  Hint: Check file has write permissions
```
