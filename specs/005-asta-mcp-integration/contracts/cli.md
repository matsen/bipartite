# CLI Contract: ASTA Commands

**Phase 1 Output** | **Date**: 2026-01-20

## Parent Command

### `bip asta`

Parent command for ASTA MCP operations.

```
bp asta [command]
```

**Global Flags** (inherited by all subcommands):
- `--human`: Output human-readable format instead of JSON

---

## Search Commands (P1)

### `bip asta search <query>`

Search papers by keyword relevance.

**Arguments**:
- `query` (required): Search keywords

**Flags**:
- `--limit N` (default: 50): Maximum results
- `--year YYYY:YYYY`: Publication date range
- `--venue STRING`: Filter by venue

**Exit Codes**:
- 0: Success
- 1: Not found (no results)
- 2: Auth error (missing/invalid API key)
- 3: API error (rate limit, network)

**JSON Output** (success):
```json
{
  "total": 42,
  "papers": [
    {
      "paperId": "abc123...",
      "title": "Paper Title",
      "authors": [{"name": "Alice Smith", "authorId": "123"}],
      "year": 2024,
      "venue": "Nature",
      "citationCount": 15,
      "isOpenAccess": true
    }
  ]
}
```

**Human Output** (success):
```
Found 42 papers

1. Paper Title
   Smith A, Jones B (2024) - Nature
   Citations: 15 | Open Access

2. Another Paper
   ...
```

---

### `bip asta snippet <query>`

Search text snippets within papers.

**Arguments**:
- `query` (required): Search text

**Flags**:
- `--limit N` (default: 20): Maximum snippets
- `--venue STRING`: Filter by venue
- `--papers IDs`: Comma-separated paper IDs to search within

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "snippets": [
    {
      "snippet": "...matching text from paper...",
      "score": 0.95,
      "paper": {
        "paperId": "abc123...",
        "title": "Paper Title",
        "authors": [{"name": "Alice Smith"}],
        "year": 2024
      }
    }
  ]
}
```

**Human Output** (success):
```
Found 20 snippets

1. [0.95] Paper Title (Smith 2024)
   "...matching text from paper..."

2. [0.87] Another Paper (Jones 2023)
   "...another matching snippet..."
```

---

## Paper Commands (P2)

### `bip asta paper <paper-id>`

Get paper details by ID.

**Arguments**:
- `paper-id` (required): Paper identifier (DOI:, ARXIV:, PMID:, CorpusId:, or S2 ID)

**Flags**:
- `--fields STRING`: Comma-separated fields to return

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "paperId": "abc123...",
  "title": "Paper Title",
  "abstract": "Paper abstract...",
  "authors": [{"name": "Alice Smith", "authorId": "123"}],
  "year": 2024,
  "venue": "Nature",
  "publicationDate": "2024-03-15",
  "citationCount": 15,
  "referenceCount": 42,
  "isOpenAccess": true,
  "fieldsOfStudy": ["Biology", "Genetics"]
}
```

---

### `bip asta citations <paper-id>`

Get papers that cite this paper.

**Arguments**:
- `paper-id` (required): Paper identifier

**Flags**:
- `--limit N` (default: 100): Maximum results
- `--year YYYY:YYYY`: Filter citing papers by date

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "paperId": "abc123...",
  "citationCount": 150,
  "citations": [
    {
      "paperId": "def456...",
      "title": "Citing Paper",
      "authors": [...],
      "year": 2025
    }
  ]
}
```

---

### `bip asta references <paper-id>`

Get papers referenced by this paper.

**Arguments**:
- `paper-id` (required): Paper identifier

**Flags**:
- `--limit N` (default: 100): Maximum results

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "paperId": "abc123...",
  "referenceCount": 42,
  "references": [
    {
      "paperId": "ghi789...",
      "title": "Referenced Paper",
      "authors": [...],
      "year": 2020
    }
  ]
}
```

---

## Author Commands (P3)

### `bip asta author <name>`

Search for authors by name.

**Arguments**:
- `name` (required): Author name to search

**Flags**:
- `--limit N` (default: 10): Maximum results

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "authors": [
    {
      "authorId": "123",
      "name": "Alice Smith",
      "affiliations": ["MIT"],
      "paperCount": 42,
      "citationCount": 1500,
      "hIndex": 25
    }
  ]
}
```

---

### `bip asta author-papers <author-id>`

Get papers by an author.

**Arguments**:
- `author-id` (required): S2 author ID

**Flags**:
- `--limit N` (default: 100): Maximum results
- `--year YYYY:YYYY`: Filter by publication date

**Exit Codes**: Same as search

**JSON Output** (success):
```json
{
  "authorId": "123",
  "name": "Alice Smith",
  "papers": [
    {
      "paperId": "abc123...",
      "title": "Paper Title",
      "year": 2024,
      "venue": "Nature",
      "citationCount": 15
    }
  ]
}
```

---

## Error Responses

**JSON Format** (all errors):
```json
{
  "error": {
    "code": "not_found",
    "message": "Paper not found",
    "paperId": "DOI:10.1234/invalid"
  }
}
```

**Human Format** (all errors):
```
Error: Paper not found
  Paper ID: DOI:10.1234/invalid
```

**Error Codes**:
- `not_found`: Resource not found
- `auth_error`: Missing or invalid ASTA_API_KEY
- `rate_limited`: Rate limit exceeded
- `api_error`: Other API errors
