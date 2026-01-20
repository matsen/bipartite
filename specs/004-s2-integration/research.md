# Research: Semantic Scholar (S2) Integration

**Feature**: 004-s2-integration
**Date**: 2026-01-19

## Research Questions

1. What are the Semantic Scholar API endpoints and rate limits?
2. How do we extract DOIs from PDF files in Go?
3. What's the best pattern for HTTP clients with rate limiting in Go?
4. How do we map S2 author names to first/last name format?

---

## 1. Semantic Scholar API

### Decision: Use S2 Academic Graph API v1

**Rationale**: Well-documented, stable API with generous free tier. Supports all required operations (paper lookup, citations, references).

### Endpoints

| Operation | Endpoint | Method |
|-----------|----------|--------|
| Get paper | `/paper/{paper_id}` | GET |
| Get paper batch | `/paper/batch` | POST |
| Get citations | `/paper/{paper_id}/citations` | GET |
| Get references | `/paper/{paper_id}/references` | GET |
| Search by keyword | `/paper/search` | GET |
| Search by title | `/paper/search?query={title}` | GET |

**Base URL**: `https://api.semanticscholar.org/graph/v1`

**Note**: An MCP server is also available at `https://asta-tools.allen.ai/mcp/v1` for Claude Code integration. The `bp` CLI uses direct HTTP calls for standalone operation.

### Paper ID Formats

All endpoints accept these ID formats:
- `DOI:10.1038/nature12373`
- `ARXIV:2106.15928`
- `PMID:19872477`
- `PMCID:2323736`
- `CorpusId:215416146`
- Raw S2 paper ID (40 hex chars)
- `URL:https://arxiv.org/abs/2106.15928`

### Rate Limits

| Tier | Limit | Notes |
|------|-------|-------|
| Unauthenticated | 100 requests / 5 minutes | Fallback if no key |
| API Key | 1 request / second sustained | **Available in .env** |

**Implementation**: Use token bucket rate limiter. With API key, can sustain 1 req/sec. Key stored in `.env` as `S2_API_KEY`.

### Response Fields

Default fields are minimal. Use `fields` parameter to request additional data:

```
?fields=paperId,externalIds,title,authors,abstract,year,venue,publicationDate,citationCount,referenceCount
```

### Alternatives Considered

- **CrossRef API**: Good for DOI resolution but no citation graph
- **OpenAlex**: Newer, open data, but less mature API
- **Google Scholar**: No API, scraping prohibited

---

## 2. PDF DOI Extraction

### Decision: Text-based regex extraction using pdftotext or Go PDF library

**Rationale**: DOIs follow a predictable pattern. Most academic PDFs contain the DOI in text form (header, footer, or first page).

### DOI Pattern

```regex
10\.\d{4,}/[^\s]+
```

More precise:
```regex
10\.\d{4,9}/[-._;()/:A-Z0-9]+
```

### Implementation Options

| Option | Pros | Cons |
|--------|------|------|
| Shell out to `pdftotext` | Reliable, handles most PDFs | External dependency |
| `github.com/ledongthuc/pdf` | Pure Go, no dependencies | May struggle with some PDFs |
| `github.com/pdfcpu/pdfcpu` | Full-featured PDF library | Heavy dependency |

**Decision**: Use `ledongthuc/pdf` for pure Go implementation. Fall back to title-based search if DOI extraction fails.

### Extraction Strategy

1. Extract text from first 2 pages (DOI usually in header/footer)
2. Search for DOI pattern
3. If multiple DOIs found, prefer the one matching common publisher prefixes
4. If no DOI, extract title (first large text block) and search S2 by title

### Alternatives Considered

- **PDF metadata fields**: Unreliable, often empty
- **CrossRef title search**: Could supplement S2 title search
- **Barcode/QR scanning**: Over-engineered for this use case

---

## 3. HTTP Client with Rate Limiting

### Decision: Standard library net/http with custom rate-limited transport

**Rationale**: No external dependencies needed. Go's standard library is sufficient.

### Implementation Pattern

```go
type RateLimitedClient struct {
    client  *http.Client
    limiter *rate.Limiter
}

func (c *RateLimitedClient) Do(req *http.Request) (*http.Response, error) {
    if err := c.limiter.Wait(req.Context()); err != nil {
        return nil, err
    }
    return c.client.Do(req)
}
```

Uses `golang.org/x/time/rate` for token bucket limiter (already in Go extended stdlib).

### Error Handling

| HTTP Status | Handling |
|-------------|----------|
| 200 | Success, parse response |
| 404 | Paper not found, return clear error |
| 429 | Rate limited, wait and retry (up to 3 times) |
| 5xx | Server error, retry with backoff |

### Timeout Strategy

- Connect timeout: 10 seconds
- Read timeout: 30 seconds (large responses)
- Total request timeout: 60 seconds

### Alternatives Considered

- **Third-party HTTP clients (resty, req)**: Unnecessary complexity
- **Circuit breaker pattern**: Over-engineered for CLI tool

---

## 4. Author Name Parsing

### Decision: Split on last space, handle common suffixes

**Rationale**: S2 returns author names as single strings (e.g., "Frederick A. Matsen IV"). We need to split into first/last for the Reference schema.

### Algorithm

```
Input: "Frederick A. Matsen IV"

1. Check for suffixes: Jr, Sr, II, III, IV, PhD, MD, etc.
2. If suffix found, treat as part of last name
3. Split remaining on last space
4. First part → first name, last part → last name

Output: first="Frederick A.", last="Matsen IV"
```

### Edge Cases

| Input | First | Last |
|-------|-------|------|
| `Madonna` | `` | `Madonna` |
| `Jean-Pierre Serre` | `Jean-Pierre` | `Serre` |
| `J. Smith` | `J.` | `Smith` |
| `Robert Smith Jr.` | `Robert` | `Smith Jr.` |
| `王伟` (Chinese) | `` | `王伟` |

### Alternatives Considered

- **S2 author ID lookup**: Could get structured name, but requires extra API call
- **Name parsing libraries**: Over-engineered for this use case
- **ML-based parsing**: Way over-engineered

---

## 5. Preprint Detection

### Decision: Check venue field for preprint servers + S2 publicationTypes

**Rationale**: Preprints are identifiable by venue (bioRxiv, medRxiv, arXiv) or S2 metadata.

### Detection Criteria

```go
func isPreprint(paper Reference) bool {
    venue := strings.ToLower(paper.Venue)
    return strings.Contains(venue, "biorxiv") ||
           strings.Contains(venue, "medrxiv") ||
           strings.Contains(venue, "arxiv")
}
```

### Finding Published Version

S2 API doesn't directly link preprints to published versions. Strategy:

1. Search S2 by title (exact match)
2. Filter results by same authors
3. Filter results by non-preprint venue
4. If match found, that's the published version

### Alternatives Considered

- **DOI resolution to published version**: Preprint DOIs don't redirect
- **CrossRef event data**: Could show relationships but requires extra API

---

## 6. Gap Discovery Algorithm

### Decision: In-memory aggregation with batched API calls

**Rationale**: For collections up to 10,000 papers, we can hold reference data in memory. Batch API calls to respect rate limits.

### Algorithm

```
1. Load all papers with DOIs from collection
2. Batch papers into groups of 500
3. For each batch:
   a. Fetch references for all papers (batch API)
   b. For each reference, increment citation count in gap map
4. Filter gaps: cited by ≥ min_citations AND not in collection
5. Sort by citation count descending
6. Return top N gaps
```

### Performance Estimate

- 1000 papers → 2 batch API calls for paper IDs
- Each paper averages 30 references → 30,000 reference entries to aggregate
- In-memory aggregation: <1 second
- Total time dominated by API calls: ~30 seconds (with rate limiting)

### Alternatives Considered

- **SQLite aggregation**: Requires persisting reference data
- **Streaming approach**: More complex, not needed at this scale

---

## Summary of Decisions

| Question | Decision | Key Rationale |
|----------|----------|---------------|
| S2 API | Academic Graph API v1 | Well-documented, generous free tier |
| PDF DOI | ledongthuc/pdf + regex | Pure Go, no external dependencies |
| HTTP Client | Standard lib + rate.Limiter | Simple, no dependencies |
| Author parsing | Split on last space | Simple, handles most cases |
| Preprint detection | Venue string matching | Reliable, no extra API calls |
| Gap discovery | In-memory aggregation | Sufficient for target scale |

---

## Open Questions (Resolved)

- ~~Should we cache S2 responses?~~ **Yes, optional optimization for later**
- ~~Should we support S2 API keys?~~ **Future enhancement, not MVP**
- ~~How to handle papers without DOIs?~~ **Use S2 paper ID as fallback identifier**
