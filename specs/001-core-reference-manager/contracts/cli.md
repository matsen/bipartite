# CLI Contract: bp

**Version**: 1.0.0
**Date**: 2026-01-12

## Overview

The `bp` command is the primary interface for bipartite. All commands output JSON by default (agent-first design) with human-readable format available via `--human` flag.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (invalid arguments, runtime failure) |
| 2 | Configuration error (missing config, invalid paths) |
| 3 | Data error (malformed input, validation failure) |

---

## Commands

### bp init

Initialize a new bipartite repository in the current directory.

```
bp init
```

**Creates**:
```
.bipartite/
├── refs.jsonl      # Empty file
├── config.yml     # Default config
└── cache/          # Empty directory (gitignored)
```

**Output** (JSON):
```json
{"status": "initialized", "path": "/path/to/repo"}
```

**Output** (--human):
```
Initialized bipartite repository in /path/to/repo
```

**Errors**:
- Exit 1: Directory already contains `.bipartite/`
- Exit 1: Cannot create directory structure

---

### bp config

Get or set configuration values.

```
bp config                          # Show all config
bp config pdf-root                 # Get specific value
bp config pdf-root /path/to/pdfs   # Set value
bp config pdf-reader skim          # Set PDF reader
```

**Keys**:
- `pdf-root`: Path to PDF folder (e.g., `~/Google Drive/Paperpile`)
- `pdf-reader`: PDF reader preference (`system`, `skim`, `zathura`, `evince`, `okular`)

**Output** (JSON, get all):
```json
{"pdf_root": "/Users/name/Google Drive/Paperpile", "pdf_reader": "skim"}
```

**Output** (JSON, get one):
```json
{"pdf_root": "/Users/name/Google Drive/Paperpile"}
```

**Output** (JSON, set):
```json
{"status": "updated", "key": "pdf_root", "value": "/Users/name/Google Drive/Paperpile"}
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 2: `pdf-root` path does not exist (on set)
- Exit 1: Unknown configuration key

---

### bp import

Import references from an external format.

```
bp import --format paperpile export.json
bp import --format paperpile export.json --dry-run
```

**Flags**:
- `--format` (required): Import format (`paperpile`)
- `--dry-run`: Show what would be imported without writing

**Output** (JSON):
```json
{
  "imported": 150,
  "updated": 23,
  "skipped": 5,
  "errors": []
}
```

**Output** (--dry-run JSON):
```json
{
  "would_import": 150,
  "would_update": 23,
  "would_skip": 5,
  "details": [
    {"id": "Ahn2026-rs", "action": "import", "title": "Influenza..."},
    {"id": "Smith2025-ab", "action": "update", "title": "Machine...", "reason": "doi_match"}
  ]
}
```

**Output** (--human):
```
Importing from Paperpile export...
  Imported: 150 new references
  Updated:  23 existing references (matched by DOI)
  Skipped:  5 (errors below)

Errors:
  - Line 42: Missing required field 'title'
  - Line 108: Invalid DOI format '10.invalid'
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 1: Unknown format
- Exit 1: File not found
- Exit 3: Malformed JSON in input file

---

### bp search

Search references by keyword.

```
bp search "phylogenetics"
bp search "phylogenetics" --limit 10
bp search "author:Matsen"
```

**Flags**:
- `--limit N`: Maximum results (default: 50)
- `--human`: Human-readable output

**Query Syntax**:
- Plain text: Searches title, abstract, and authors
- `author:name`: Search author names only
- `title:text`: Search title only

**Output** (JSON):
```json
[
  {
    "id": "Gao2026-gi",
    "doi": "10.1073/pnas.2510938123",
    "title": "Biological causes and impacts...",
    "authors": [{"first": "Jiansi", "last": "Gao"}, ...],
    "venue": "PNAS",
    "published": {"year": 2026, "month": 1, "day": 9}
  }
]
```

**Output** (--human):
```
Found 3 references:

[1] Gao2026-gi
    Biological causes and impacts of rugged tree landscapes...
    Gao J, Brusselmans M, Carvalho LM, et al.
    PNAS (2026)

[2] Smith2025-ab
    ...
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 0: No results (empty array, not an error)

---

### bp get

Get a single reference by ID.

```
bp get Ahn2026-rs
bp get Ahn2026-rs --human
```

**Output** (JSON):
```json
{
  "id": "Ahn2026-rs",
  "doi": "10.64898/2026.01.05.697808",
  "title": "Influenza hemagglutinin subtypes...",
  "authors": [...],
  "abstract": "Abstract Hemagglutinins...",
  "venue": "bioRxiv",
  "published": {"year": 2026, "month": 1, "day": 6},
  "pdf_path": "All Papers/A/Ahn et al. 2026 - Influenza....pdf",
  "source": {"type": "paperpile", "id": "2773420d-..."}
}
```

**Output** (--human):
```
Ahn2026-rs
══════════════════════════════════════════════════════════════════════

Title:    Influenza hemagglutinin subtypes have different sequence
          constraints despite sharing extremely similar structures

Authors:  Jenny J Ahn, Timothy C Yu, Bernadeta Dadonaite,
          Caelan E Radford, Jesse D Bloom

Venue:    bioRxiv
Date:     2026-01-06
DOI:      10.64898/2026.01.05.697808

Abstract:
  Hemagglutinins (HA) from different influenza A virus subtypes share
  as little as ~40% amino acid identity, yet their protein structure
  and cell entry function are highly conserved...

PDF:      All Papers/A/Ahn et al. 2026 - Influenza....pdf
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 1: Reference ID not found

---

### bp list

List all references.

```
bp list
bp list --limit 100
bp list --human
```

**Flags**:
- `--limit N`: Maximum results (default: all)
- `--human`: Human-readable output

**Output** (JSON):
```json
[
  {"id": "Ahn2026-rs", "title": "...", ...},
  {"id": "Gao2026-gi", "title": "...", ...}
]
```

**Output** (--human):
```
178 references in repository:

  Ahn2026-rs      Influenza hemagglutinin subtypes...
  Gao2026-gi      Biological causes and impacts of...
  ...
```

---

### bp open

Open a paper's PDF in the configured viewer.

```
bp open Ahn2026-rs
bp open Ahn2026-rs --supplement 1
```

**Flags**:
- `--supplement N`: Open Nth supplementary PDF (1-indexed)

**Output** (JSON):
```json
{"status": "opened", "path": "/Users/name/Google Drive/Paperpile/All Papers/A/Ahn et al....pdf"}
```

**Output** (--human):
```
Opening: All Papers/A/Ahn et al. 2026 - Influenza....pdf
```

**Errors**:
- Exit 2: PDF root not configured
- Exit 1: Reference ID not found
- Exit 1: No PDF path for reference
- Exit 1: PDF file not found at expected path
- Exit 1: No supplement at index N

---

### bp export

Export references to BibTeX format.

```
bp export --bibtex
bp export --bibtex --keys Ahn2026-rs,Gao2026-gi
bp export --bibtex > refs.bib
```

**Flags**:
- `--bibtex` (required): Export format
- `--keys id1,id2,...`: Export only specified IDs

**Output** (always text, not JSON):
```bibtex
@article{Ahn2026-rs,
  author = {Ahn, Jenny J and Yu, Timothy C and Dadonaite, Bernadeta and Radford, Caelan E and Bloom, Jesse D},
  title = {Influenza hemagglutinin subtypes have different sequence constraints despite sharing extremely similar structures},
  journal = {bioRxiv},
  year = {2026},
  doi = {10.64898/2026.01.05.697808},
}

@article{Gao2026-gi,
  ...
}
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 1: Unknown key in --keys list

---

### bp rebuild

Rebuild the query layer from source data.

```
bp rebuild
```

**Output** (JSON):
```json
{"status": "rebuilt", "references": 178}
```

**Output** (--human):
```
Rebuilt query database with 178 references
```

**Errors**:
- Exit 2: Not in a bipartite repository
- Exit 3: Malformed refs.jsonl

---

### bp check

Verify repository integrity.

```
bp check
```

**Output** (JSON, success):
```json
{"status": "ok", "references": 178, "issues": []}
```

**Output** (JSON, issues found):
```json
{
  "status": "issues",
  "references": 178,
  "issues": [
    {"type": "missing_pdf", "id": "Smith2025-ab", "expected": "All Papers/S/Smith...pdf"},
    {"type": "duplicate_doi", "ids": ["Paper1", "Paper2"], "doi": "10.1000/xyz"}
  ]
}
```

**Output** (--human):
```
Repository check: 2 issues found

  [WARN] Missing PDF for Smith2025-ab
         Expected: All Papers/S/Smith et al. 2025 - Machine....pdf

  [WARN] Duplicate DOI 10.1000/xyz
         Found in: Paper1, Paper2

178 references checked
```

**Errors**:
- Exit 2: Not in a bipartite repository

---

## Global Flags

Available on all commands:

| Flag | Description |
|------|-------------|
| `--human` | Human-readable output (default: JSON) |
| `--help` | Show help for command |
| `--version` | Show version |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `XDG_CONFIG_HOME` | Override config directory | `~/.config` |
| `NO_COLOR` | Disable colored output | (unset) |

## Global Configuration

Repository location is configured in `~/.config/bip/config.yml`:

```json
{
  "nexus_path": "~/re/nexus"
}
```

The `nexus_path` setting is required and specifies the bipartite repository location.
