# Data Model: URL Output and Clipboard Support

**Feature**: 015-url-clipboard
**Date**: 2026-01-27

## Entity Changes

### Reference (Modified)

The Reference struct gains four new optional fields for external identifiers.

**New Fields**:

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| PMID | string | `pmid` | PubMed ID (omitted if empty) |
| PMCID | string | `pmcid` | PubMed Central ID (omitted if empty) |
| ArXivID | string | `arxiv_id` | arXiv identifier (omitted if empty) |
| S2ID | string | `s2_id` | Semantic Scholar paper ID (omitted if empty) |

**Full Struct Definition** (after modification):

```go
// Reference represents an academic paper or article.
type Reference struct {
    // Identity
    ID  string `json:"id"`
    DOI string `json:"doi"`

    // Metadata
    Title    string   `json:"title"`
    Authors  []Author `json:"authors"`
    Abstract string   `json:"abstract"`
    Venue    string   `json:"venue"`

    // Publication Date
    Published PublicationDate `json:"published"`

    // File Paths
    PDFPath         string   `json:"pdf_path"`
    SupplementPaths []string `json:"supplement_paths,omitempty"`

    // Import Tracking
    Source ImportSource `json:"source"`

    // Relationships
    Supersedes string `json:"supersedes,omitempty"`

    // External Identifiers (NEW)
    PMID    string `json:"pmid,omitempty"`
    PMCID   string `json:"pmcid,omitempty"`
    ArXivID string `json:"arxiv_id,omitempty"`
    S2ID    string `json:"s2_id,omitempty"`
}
```

### JSONL Format

Example reference with external IDs:

```json
{
  "id": "Smith2024-ab",
  "doi": "10.1234/example",
  "title": "Example Paper Title",
  "authors": [{"first": "John", "last": "Smith"}],
  "abstract": "This is the abstract...",
  "venue": "Nature",
  "published": {"year": 2024, "month": 3, "day": 15},
  "pdf_path": "papers/Smith2024-ab.pdf",
  "source": {"type": "s2", "id": "649def34f8be52c8b66281af98ae884c09aef38b"},
  "pmid": "12345678",
  "pmcid": "PMC1234567",
  "arxiv_id": "2106.15928",
  "s2_id": "649def34f8be52c8b66281af98ae884c09aef38b"
}
```

**Note**: Only populated fields appear in output (omitempty). A paper without PubMed will not have `pmid` key.

## SQLite Schema Changes

### refs Table (Modified)

Add four nullable text columns:

```sql
ALTER TABLE refs ADD COLUMN pmid TEXT;
ALTER TABLE refs ADD COLUMN pmcid TEXT;
ALTER TABLE refs ADD COLUMN arxiv_id TEXT;
ALTER TABLE refs ADD COLUMN s2_id TEXT;
```

**Note**: Since SQLite is ephemeral (rebuilt from JSONL), the actual implementation modifies the `CREATE TABLE` statement in `createSchema()`.

### Updated Schema

```sql
CREATE TABLE IF NOT EXISTS refs (
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
    authors_json TEXT NOT NULL,
    supplement_paths_json TEXT,
    -- NEW columns
    pmid TEXT,
    pmcid TEXT,
    arxiv_id TEXT,
    s2_id TEXT
);
```

## S2 API Mapping

The S2 API already returns external IDs in the `ExternalIDs` struct. The mapper update extracts these:

| S2 API Field | Reference Field |
|--------------|-----------------|
| `ExternalIDs.PubMed` | `PMID` |
| `ExternalIDs.PubMedCentral` | `PMCID` |
| `ExternalIDs.ArXiv` | `ArXivID` |
| `PaperID` | `S2ID` |

**Mapping Logic**:
```go
func MapS2ToReference(paper S2Paper) reference.Reference {
    ref := reference.Reference{
        // ... existing mapping ...

        // External IDs
        PMID:    paper.ExternalIDs.PubMed,
        PMCID:   paper.ExternalIDs.PubMedCentral,
        ArXivID: paper.ExternalIDs.ArXiv,
        S2ID:    paper.PaperID,
    }
    return ref
}
```

## URL Result Type

New type for JSON output from `bip url`:

```go
// URLResult is the JSON output for bip url command.
type URLResult struct {
    URL    string `json:"url"`
    Format string `json:"format"` // doi, pubmed, pmc, arxiv, s2
    Copied bool   `json:"copied"` // true if --copy succeeded
}
```

## Validation Rules

1. **External IDs are optional**: Papers imported before this feature or from non-S2 sources will not have external IDs
2. **Missing ID = error**: If user requests `--pubmed` but reference has no PMID, return clear error
3. **DOI is default**: If no format flag specified, use DOI; if no DOI exists, error
4. **No empty strings**: Empty strings should be omitted from JSONL (omitempty), not stored as `"pmid": ""`
