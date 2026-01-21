# Data Model: Core Reference Manager

**Feature**: 001-core-reference-manager
**Date**: 2026-01-12

## Overview

The data model consists of three main entities stored in two persistence layers:
- **Source of Truth**: JSONL file (`refs.jsonl`) - human-readable, git-versionable
- **Query Layer**: SQLite database (`refs.db`) - ephemeral, rebuilt from JSONL

## Entities

### Reference

The core entity representing an academic paper or article.

```go
type Reference struct {
    // Identity
    ID  string `json:"id"`  // Internal stable identifier (from citekey)
    DOI string `json:"doi"` // Digital Object Identifier (primary deduplication key)

    // Metadata
    Title    string   `json:"title"`
    Authors  []Author `json:"authors"`
    Abstract string   `json:"abstract"`
    Venue    string   `json:"venue"` // Journal, conference, or preprint server

    // Publication Date
    Published PublicationDate `json:"published"`

    // File Paths (relative to configured PDF root)
    PDFPath         string   `json:"pdf_path"`
    SupplementPaths []string `json:"supplement_paths,omitempty"`

    // Import Tracking
    Source ImportSource `json:"source"`

    // Relationships
    Supersedes string `json:"supersedes,omitempty"` // DOI of paper this replaces
}
```

**Field Constraints**:

| Field | Required | Constraints |
|-------|----------|-------------|
| `id` | Yes | Non-empty, unique within repository |
| `doi` | No | Valid DOI format when present |
| `title` | Yes | Non-empty |
| `authors` | Yes | At least one author |
| `abstract` | No | May be empty string |
| `venue` | No | May be empty string |
| `published.year` | Yes | Integer > 1900 |
| `published.month` | No | Integer 1-12 when present |
| `published.day` | No | Integer 1-31 when present |
| `pdf_path` | No | Relative path when present |
| `supplement_paths` | No | Array of relative paths |
| `source.type` | Yes | One of: paperpile, zotero, mendeley, manual, asta |
| `source.id` | No | Original ID from source system |
| `supersedes` | No | Valid DOI when present |

---

### Author

Represents a paper author with optional ORCID identifier.

```go
type Author struct {
    First string `json:"first"`         // First/given name(s)
    Last  string `json:"last"`          // Last/family name
    ORCID string `json:"orcid,omitempty"` // ORCID identifier (without URL prefix)
}
```

**Field Constraints**:

| Field | Required | Constraints |
|-------|----------|-------------|
| `first` | No | May be empty (single-name authors) |
| `last` | Yes | Non-empty |
| `orcid` | No | Format: `0000-0000-0000-0000` when present |

---

### PublicationDate

Structured date allowing partial information (year-only common for older papers).

```go
type PublicationDate struct {
    Year  int `json:"year"`
    Month int `json:"month,omitempty"` // 1-12, 0 if unknown
    Day   int `json:"day,omitempty"`   // 1-31, 0 if unknown
}
```

**Validation Rules**:
- Year is always required and must be > 1900
- Month, if present, must be 1-12
- Day, if present, must be valid for the given month/year

---

### ImportSource

Tracks where a reference was imported from for re-import matching.

```go
type ImportSource struct {
    Type string `json:"type"` // paperpile, zotero, mendeley, manual, asta
    ID   string `json:"id"`   // Original ID from source system
}
```

**Source Types**:
- `paperpile`: Imported from Paperpile JSON export
- `zotero`: Imported from Zotero (future)
- `mendeley`: Imported from Mendeley (future)
- `manual`: Manually added via CLI
- `asta`: Fetched from ASTA/Semantic Scholar (future)

---

### Config

Repository configuration stored in `.bipartite/config.json`.

```go
type Config struct {
    PDFRoot   string `json:"pdf_root"`   // Absolute path to PDF folder
    PDFReader string `json:"pdf_reader"` // Reader preference: system, skim, zathura, etc.
}
```

**Field Constraints**:

| Field | Required | Constraints |
|-------|----------|-------------|
| `pdf_root` | No | Must exist as directory when set |
| `pdf_reader` | No | Defaults to "system" |

---

## Storage Schema

### JSONL Format (refs.jsonl)

One reference per line, complete JSON object:

```jsonl
{"id":"Ahn2026-rs","doi":"10.64898/2026.01.05.697808","title":"Influenza hemagglutinin...","authors":[{"first":"Jenny J","last":"Ahn","orcid":"0009-0000-3912-7162"}],"abstract":"Abstract Hemagglutinins...","venue":"bioRxiv","published":{"year":2026,"month":1,"day":6},"pdf_path":"All Papers/A/Ahn et al. 2026 - Influenza....pdf","source":{"type":"paperpile","id":"2773420d-4009-0be9-920f-d674f7f86794"}}
```

**Design Decisions**:
- No pretty-printing (one line per record for clean git diffs)
- Fields ordered consistently (id first, then doi, then metadata)
- Empty arrays omitted (`supplement_paths` only if non-empty)
- Empty strings included (explicit `"abstract":""` vs missing field)

---

### SQLite Schema (refs.db)

Ephemeral database rebuilt from JSONL via `bip rebuild`.

```sql
-- Main references table
CREATE TABLE references (
    id TEXT PRIMARY KEY,
    doi TEXT,
    title TEXT NOT NULL,
    abstract TEXT,
    venue TEXT,
    pub_year INTEGER NOT NULL,
    pub_month INTEGER,
    pub_day INTEGER,
    pdf_path TEXT,
    source_type TEXT NOT NULL,
    source_id TEXT,
    supersedes TEXT,
    -- Full JSON for complex fields
    authors_json TEXT NOT NULL,
    supplement_paths_json TEXT
);

-- Index for DOI lookups (deduplication)
CREATE UNIQUE INDEX idx_references_doi ON references(doi) WHERE doi IS NOT NULL;

-- Full-text search on title, abstract, authors
CREATE VIRTUAL TABLE references_fts USING fts5(
    id,
    title,
    abstract,
    authors_text,  -- "Jenny J Ahn, Timothy C Yu, ..."
    content='references',
    content_rowid='rowid'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER references_ai AFTER INSERT ON references BEGIN
    INSERT INTO references_fts(rowid, id, title, abstract, authors_text)
    VALUES (new.rowid, new.id, new.title, new.abstract,
            (SELECT group_concat(json_extract(value, '$.first') || ' ' || json_extract(value, '$.last'), ', ')
             FROM json_each(new.authors_json)));
END;
```

**Why SQLite for Queries**:
- Fast full-text search via FTS5
- DOI index for O(1) deduplication checks
- Standard SQL for flexible queries
- Ephemeral: corrupted? Just rebuild

---

## State Transitions

### Reference Lifecycle

```
[Import] → [Stored] → [Updated via re-import] → [Exported]
                   ↘ [Marked as superseded]
```

### Import Deduplication Flow

```
                    ┌─────────────────────────────────────┐
                    │        Incoming Reference           │
                    └─────────────────┬───────────────────┘
                                      │
                              Has DOI?│
                         ┌────────────┴────────────┐
                         │ Yes                     │ No
                         ▼                         ▼
                 ┌───────────────┐         ┌──────────────┐
                 │ DOI in DB?    │         │ ID in DB?    │
                 └───────┬───────┘         └──────┬───────┘
                    Yes  │  No                Yes │  No
                    ▼    ▼                   ▼    ▼
             ┌────────┐ ┌────────┐    ┌────────┐ ┌────────┐
             │ Update │ │ Insert │    │ Suffix │ │ Insert │
             │existing│ │  new   │    │  ID    │ │  new   │
             └────────┘ └────────┘    └────────┘ └────────┘
```

---

## Relationships

### Reference → Reference (Supersedes)

A reference may supersede another (e.g., published paper supersedes preprint).

```
Reference A (published)
    └── supersedes: "10.xxx/preprint" → Reference B (preprint)
```

**Query**: "Find the canonical version of this paper"
```sql
SELECT r.* FROM references r
WHERE r.doi = (
    SELECT supersedes FROM references WHERE id = ?
)
```

### Reference → Authors (One-to-Many)

Authors are embedded in the reference as a JSON array. No separate authors table.

**Rationale**: Authors are not independently queryable entities in Phase I. If author-centric queries become important (Phase III knowledge graph), we can extract them.

---

## Validation Rules

### On Import

1. **Required fields present**: id, title, authors (≥1), published.year, source.type
2. **DOI format**: If present, matches pattern `10.\d{4,}/.*`
3. **Date validity**: Year > 1900, month 1-12, day 1-31 and valid for month
4. **ORCID format**: If present, matches `\d{4}-\d{4}-\d{4}-\d{4}`

### On Config Update

1. **PDF root exists**: Path must be a valid directory
2. **PDF reader valid**: Must be one of: system, skim, zathura, evince, okular

### On PDF Open

1. **Config exists**: PDF root must be configured
2. **File exists**: Combined path must point to existing file
