# Data Model: ASTA Integration

**Feature**: 004-asta-integration
**Date**: 2026-01-19

## Overview

ASTA integration uses the existing Reference data model from Phase I. This document describes:
- How Semantic Scholar data maps to the Reference schema
- Optional enrichment fields that could be added
- Caching strategy for API responses

## Semantic Scholar → Reference Mapping

Papers fetched from Semantic Scholar map to the existing Reference schema:

| S2 Field | Reference Field | Notes |
|----------|-----------------|-------|
| `paperId` | `source.id` | S2's internal paper ID |
| `externalIds.DOI` | `doi` | Primary identifier for deduplication |
| `title` | `title` | Direct mapping |
| `authors[].name` | `authors[].first`, `authors[].last` | Split on last space |
| `authors[].authorId` | - | Not stored (S2-specific) |
| `abstract` | `abstract` | Direct mapping, may be null |
| `year` | `published.year` | Direct mapping |
| `publicationDate` | `published.*` | Parse YYYY-MM-DD format |
| `venue` | `venue` | Direct mapping |
| - | `source.type` | Always `"asta"` |
| - | `pdf_path` | Set via `--link` flag if provided |

### Author Name Parsing

Semantic Scholar returns author names as a single string. Parsing strategy:

```
"Frederick A. Matsen IV" → first: "Frederick A.", last: "Matsen IV"
"Madonna" → first: "", last: "Madonna"
"Jean-Pierre Serre" → first: "Jean-Pierre", last: "Serre"
```

Rules:
1. Split on last space
2. Handle suffixes (Jr, Sr, II, III, IV) by including with last name
3. Single-word names → empty first, word as last
4. Hyphenated names → preserve hyphens

### Fields Not Mapped

These S2 fields are available but not stored in the core Reference:

| S2 Field | Reason |
|----------|--------|
| `citationCount` | Changes over time; query live |
| `referenceCount` | Changes over time; query live |
| `fieldsOfStudy` | Could add later; not core metadata |
| `isOpenAccess` | Could add later; not critical |
| `s2FieldsOfStudy` | More granular than needed |
| `influentialCitationCount` | S2-specific metric |
| `tldr` | AI-generated; prefer human abstract |

## Paper Identifier Formats

The system accepts multiple identifier formats for S2 API queries:

| Format | Example | Notes |
|--------|---------|-------|
| DOI | `DOI:10.1038/nature12373` | Most common academic identifier |
| arXiv | `ARXIV:2106.15928` | Preprints |
| PubMed | `PMID:19872477` | Biomedical literature |
| PMC | `PMCID:2323736` | PubMed Central |
| S2 ID | `649def34f8be52c8b66281af98ae884c09aef38b` | Raw S2 paper ID |
| CorpusId | `CorpusId:215416146` | S2 numeric ID |
| URL | `URL:https://arxiv.org/abs/2106.15928` | Supported site URLs |

**Internal Resolution**: When a paper is added, the S2 `paperId` is always stored in `source.id`, regardless of which identifier format was used to fetch it. This ensures consistent lookups.

## Optional Schema Extensions

These fields could be added to Reference in future iterations:

```go
type Reference struct {
    // ... existing fields ...

    // Optional ASTA enrichment (future consideration)
    S2PaperID      string   `json:"s2_paper_id,omitempty"`  // For direct S2 lookups
    FieldsOfStudy  []string `json:"fields_of_study,omitempty"`
    IsOpenAccess   bool     `json:"is_open_access,omitempty"`
}
```

**Decision**: For Phase IV, we store only the core Reference fields. Enrichment fields can be added later without breaking compatibility (JSONL is additive).

## Caching Strategy

### Response Cache (Optional)

To reduce API calls and respect rate limits:

```
.bipartite/cache/
├── refs.db           # Existing SQLite index
├── vectors.gob       # Existing vector index
└── s2_cache.db       # NEW: S2 API response cache (SQLite)
```

**Cache Schema**:

```sql
CREATE TABLE s2_papers (
    paper_id TEXT PRIMARY KEY,  -- S2 paper ID
    response_json TEXT,         -- Full API response
    fetched_at INTEGER,         -- Unix timestamp
    expires_at INTEGER          -- Cache expiry
);

CREATE TABLE s2_citations (
    paper_id TEXT,
    citing_paper_id TEXT,
    fetched_at INTEGER,
    PRIMARY KEY (paper_id, citing_paper_id)
);
```

**Cache Policy**:
- Paper metadata: Cache for 30 days (rarely changes)
- Citations: Cache for 7 days (can change with new publications)
- Force refresh with `--no-cache` flag

**Decision**: Caching is optional for Phase IV MVP. The system should work without caching (just slower and rate-limited). Caching can be added as an optimization.

## Gap Discovery Data Model

For `bp asta gaps`, we need to track citation relationships:

```
Gap = {
    paper_id: "S2:xxxx",           // S2 ID of the missing paper
    title: "...",
    cited_by: ["local_id_1", "local_id_2"],  // Papers in collection that cite it
    citation_count: 2              // Count within collection
}
```

This is computed on-the-fly, not stored. The algorithm:

1. For each paper in collection with a DOI/S2 ID
2. Fetch its references from S2
3. For each reference, check if it exists locally (by DOI)
4. Aggregate: papers cited by multiple local papers but not in collection

## Preprint→Published Linking

The `supersedes` field uses DOI as the link:

```json
{
    "id": "Smith2025-bioRxiv",
    "doi": "10.1101/2025.01.01.000001",
    "title": "My Preprint",
    "supersedes": "10.1038/s41586-025-00001-0",  // DOI of published version
    ...
}
```

**S2 Detection**: Semantic Scholar tracks preprint→published relationships. When querying a preprint, S2 may return `externalIds` with both the preprint DOI and the published DOI, or the paper may have a `publicationTypes` field indicating it's a preprint.

**Linking Flow**:
1. Query S2 for papers in collection where `venue` contains "bioRxiv", "medRxiv", or "arXiv"
2. For each, check if S2 knows about a published version
3. If found, offer to set `supersedes` to the published DOI

## State Transitions

### Paper Addition via ASTA

```
[DOI provided] → [Query S2] → [Paper found?]
                                   │
                     ┌─────────────┴─────────────┐
                     │ Yes                       │ No
                     ▼                           ▼
              [DOI in local?]              [Error: Not found]
                     │
        ┌────────────┴────────────┐
        │ Yes                     │ No
        ▼                         ▼
  [Skip or update]          [Add to refs.jsonl]
  (--update flag)           [Rebuild index]
```

### PDF Addition via ASTA

```
[PDF provided] → [Extract DOI] → [DOI found?]
                                      │
                        ┌─────────────┴─────────────┐
                        │ Yes                       │ No
                        ▼                           ▼
                  [Same as DOI flow]         [Title search]
                                                    │
                                          [Matches found?]
                                                    │
                                   ┌────────────────┴────────────────┐
                                   │ Yes                             │ No
                                   ▼                                 ▼
                            [Prompt for selection]            [Error: Not found]
                                   │
                            [User confirms]
                                   │
                            [Same as DOI flow]
```
