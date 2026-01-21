# CLI Contract: S2 Commands

**Feature**: 004-s2-integration
**Date**: 2026-01-19

## Overview

All S2 commands are subcommands of `bip s2`. Output is JSON by default (agent-first design), with `--human` flag for readable output.

## Commands

### bp s2 add

Add a paper to the collection by fetching metadata from Semantic Scholar.

**Synopsis**:
```bash
bp s2 add <paper-id> [--update] [--link <pdf-path>] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| paper-id | yes | Paper identifier (DOI:..., ARXIV:..., PMID:..., or S2 ID) |
| --update, -u | no | Update metadata if paper already exists |
| --link, -l | no | Set pdf_path to the given file path |
| --human | no | Human-readable output (default: JSON) |

**Behavior**:
- Queries Semantic Scholar for the paper
- If paper exists locally (by DOI), reports duplicate unless `--update`
- Maps S2 response to Reference schema
- Appends to refs.jsonl
- Rebuilds SQLite index

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success (added or updated) |
| 1 | Paper not found in Semantic Scholar |
| 2 | Paper already exists (without --update) |
| 3 | API error (rate limit, network) |

**Output (JSON)**:
```json
{
  "action": "added",
  "paper": {
    "id": "Smith2025-ab",
    "doi": "10.1038/s41586-025-00001-0",
    "title": "A Novel Approach to...",
    "authors": [{"first": "John", "last": "Smith"}],
    "year": 2025,
    "venue": "Nature"
  }
}
```

**Output (human)**:
```
Added: Smith2025-ab
  Title: A Novel Approach to...
  Authors: John Smith
  Year: 2025
  Venue: Nature
```

---

### bp s2 add-pdf

Add a paper by extracting DOI from a PDF and fetching metadata.

**Synopsis**:
```bash
bp s2 add-pdf <pdf-path> [--link] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| pdf-path | yes | Path to PDF file |
| --link | no | Set pdf_path to the PDF (default: do not link) |
| --human | no | Human-readable output |

**Behavior**:
- Extracts DOI from PDF (text search for DOI pattern)
- If DOI found, proceeds as `bip s2 add DOI:...`
- If no DOI, attempts title extraction and S2 title search
- If multiple matches, prompts for selection (or errors in non-interactive)

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | DOI extraction failed and no title match |
| 2 | Multiple matches, user selection required |
| 3 | API error |

**Output (JSON)**:
```json
{
  "action": "added",
  "doi_source": "extracted",
  "paper": { ... }
}
```

---

### bp s2 lookup

Query Semantic Scholar for paper information without adding to collection.

**Synopsis**:
```bash
bp s2 lookup <paper-id> [--fields <field-list>] [--exists] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| paper-id | yes | Paper identifier |
| --fields, -f | no | Comma-separated fields to return (default: all) |
| --exists, -e | no | Include whether paper exists in local collection |
| --human | no | Human-readable output |

**Available Fields**:
- `title`, `authors`, `abstract`, `year`, `venue`, `doi`
- `citationCount`, `referenceCount`, `fieldsOfStudy`, `isOpenAccess`

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Paper not found |
| 3 | API error |

**Output (JSON)**:
```json
{
  "paperId": "649def34f8be52c8b66281af98ae884c09aef38b",
  "doi": "10.1038/s41586-025-00001-0",
  "title": "A Novel Approach to...",
  "authors": [{"name": "John Smith", "authorId": "12345"}],
  "year": 2025,
  "venue": "Nature",
  "citationCount": 42,
  "referenceCount": 35,
  "existsLocally": true
}
```

---

### bp s2 citations

Find papers that cite a given paper.

**Synopsis**:
```bash
bp s2 citations <paper-id> [--local-only] [--limit N] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| paper-id | yes | Local paper ID or S2 identifier |
| --local-only | no | Only show citations in local collection |
| --limit, -n | no | Maximum results (default: 50) |
| --human | no | Human-readable output |

**Behavior**:
- If paper-id is a local ID, looks up DOI/S2 ID from refs.jsonl
- Queries S2 for citing papers
- Checks each result against local collection

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Paper not found (locally or in S2) |
| 3 | API error |

**Output (JSON)**:
```json
{
  "paper_id": "Smith2025-ab",
  "citations": [
    {
      "paperId": "abc123",
      "doi": "10.1234/example",
      "title": "Building on Smith's Work...",
      "year": 2026,
      "existsLocally": false,
      "localId": null
    },
    {
      "paperId": "def456",
      "doi": "10.5678/another",
      "title": "Further Extensions...",
      "year": 2026,
      "existsLocally": true,
      "localId": "Jones2026-xy"
    }
  ],
  "total": 2
}
```

**Output (human)**:
```
Papers citing Smith2025-ab:

  [NOT IN COLLECTION] Building on Smith's Work... (2026)
    DOI: 10.1234/example

  [IN COLLECTION: Jones2026-xy] Further Extensions... (2026)
    DOI: 10.5678/another

Total: 2 citations (1 in collection)
```

---

### bp s2 references

Find papers referenced by a given paper.

**Synopsis**:
```bash
bp s2 references <paper-id> [--missing] [--limit N] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| paper-id | yes | Local paper ID or S2 identifier |
| --missing | no | Only show references NOT in local collection |
| --limit, -n | no | Maximum results (default: 100) |
| --human | no | Human-readable output |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Paper not found |
| 3 | API error |

**Output (JSON)**:
```json
{
  "paper_id": "Smith2025-ab",
  "references": [
    {
      "paperId": "xyz789",
      "doi": "10.1111/foundation",
      "title": "Foundational Theory...",
      "year": 2020,
      "existsLocally": true,
      "localId": "Foundation2020-zz"
    }
  ],
  "total": 35,
  "inCollection": 12,
  "missing": 23
}
```

---

### bp s2 gaps

Discover literature gaps - highly cited papers not in your collection.

**Synopsis**:
```bash
bp s2 gaps [--min-citations N] [--limit N] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| --min-citations, -m | no | Minimum citation count within collection (default: 2) |
| --limit, -n | no | Maximum results (default: 20) |
| --human | no | Human-readable output |

**Behavior**:
- For each paper in collection, fetches references from S2
- Aggregates: papers cited by multiple local papers but not in collection
- Ranks by citation count within collection

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 3 | API error |

**Output (JSON)**:
```json
{
  "gaps": [
    {
      "paperId": "classic123",
      "doi": "10.1000/classic",
      "title": "The Classic Paper Everyone Cites",
      "year": 2010,
      "citedByLocal": ["Smith2025-ab", "Jones2024-cd", "Brown2023-ef"],
      "citationCountLocal": 3
    }
  ],
  "total": 15,
  "analyzed_papers": 150
}
```

**Output (human)**:
```
Literature gaps (cited by 2+ papers in your collection):

  The Classic Paper Everyone Cites (2010)
    DOI: 10.1000/classic
    Cited by 3 papers in your collection:
      - Smith2025-ab
      - Jones2024-cd
      - Brown2023-ef

Found 15 gaps after analyzing 150 papers.
```

---

### bp s2 link-published

Find and link preprints to their published versions.

**Synopsis**:
```bash
bp s2 link-published [--auto] [--human]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| --auto | no | Automatically link without confirmation |
| --human | no | Human-readable output |

**Behavior**:
- Finds papers with bioRxiv/medRxiv/arXiv in venue
- For each, queries S2 for published version
- If found and `--auto`, sets `supersedes` field
- If found and no `--auto`, prompts for confirmation

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 3 | API error |

**Output (JSON)**:
```json
{
  "linked": [
    {
      "preprint_id": "Smith2025-bioRxiv",
      "preprint_doi": "10.1101/2025.01.01.000001",
      "published_doi": "10.1038/s41586-025-00001-0",
      "published_venue": "Nature"
    }
  ],
  "no_published_found": ["OtherPreprint2025-ab"],
  "already_linked": ["LinkedPaper2024-cd"],
  "total_preprints": 10
}
```

**Output (human)**:
```
Scanning 10 preprints for published versions...

  Smith2025-bioRxiv (10.1101/2025.01.01.000001)
    → Published in Nature: 10.1038/s41586-025-00001-0
    Linked!

  OtherPreprint2025-ab
    → No published version found

Summary: 1 linked, 5 no published version, 4 already linked
```

---

## Common Patterns

### Paper Identifier Resolution

All commands accepting `<paper-id>` follow this resolution order:

1. If matches local ID in refs.jsonl → use that paper's DOI/S2 ID
2. If starts with `DOI:`, `ARXIV:`, `PMID:`, etc. → query S2 directly
3. If looks like raw S2 ID (40 hex chars) → query S2 directly

### Rate Limiting

All S2 API calls respect rate limits:
- Unauthenticated: 100 requests / 5 minutes
- Commands report progress for long operations
- `--no-cache` flag available to bypass response cache

### Error Messages

API errors include actionable guidance:

```json
{
  "error": "rate_limited",
  "message": "Semantic Scholar rate limit exceeded",
  "suggestion": "Wait 5 minutes or reduce request frequency",
  "retry_after": 300
}
```

```json
{
  "error": "not_found",
  "message": "Paper not found in Semantic Scholar",
  "paper_id": "DOI:10.1234/nonexistent",
  "suggestion": "Verify the DOI is correct"
}
```
