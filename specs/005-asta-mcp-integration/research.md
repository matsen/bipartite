# Research: ASTA MCP Integration

**Phase 0 Output** | **Date**: 2026-01-20

## MCP Protocol Details

### Decision: Use JSON-RPC 2.0 over HTTP

**Rationale**: MCP (Model Context Protocol) uses JSON-RPC 2.0 for all communication. ASTA exposes an HTTP endpoint at `https://asta-tools.allen.ai/mcp/v1` that accepts standard JSON-RPC tool invocation requests.

**Alternatives Considered**:
- MCP client library: Rejected per Constitution VI (Simplicity) - adds unnecessary dependency
- Direct REST API: Not available - ASTA only exposes MCP interface

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "<tool-name>",
    "arguments": {
      "<param>": "<value>"
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "<JSON-encoded-result>"
      }
    ]
  }
}
```

**Note**: The actual result data is JSON-encoded within the `text` field and must be parsed.

## ASTA Tool Inventory

Based on the MCP tools available from ASTA, the following tools map to our CLI commands:

| CLI Command | MCP Tool | Key Parameters |
|-------------|----------|----------------|
| `bp asta search` | `search_papers_by_relevance` | `keyword`, `fields`, `limit`, `publication_date_range`, `venues` |
| `bp asta snippet` | `snippet_search` | `query`, `limit`, `venues`, `paper_ids`, `inserted_before` |
| `bp asta paper` | `get_paper` | `paper_id`, `fields` |
| `bp asta citations` | `get_citations` | `paper_id`, `fields`, `limit`, `publication_date_range` |
| `bp asta references` | N/A (use get_paper with references field) | - |
| `bp asta author` | `search_authors_by_name` | `name`, `fields`, `limit` |
| `bp asta author-papers` | `get_author_papers` | `author_id`, `paper_fields`, `limit`, `publication_date_range` |

### Additional Tools Available

- `get_paper_batch`: Get multiple papers by IDs
- `search_paper_by_title`: Search by exact title (vs keyword relevance)

## Authentication

### Decision: x-api-key header from ASTA_API_KEY env var

**Rationale**: ASTA uses API key authentication via the `x-api-key` HTTP header. Following existing patterns (`S2_API_KEY`), we use `ASTA_API_KEY` in `.env`.

**Implementation**: Load via `godotenv.Load()` in the asta parent command init, same as s2 commands.

## Rate Limiting

### Decision: 10 req/sec using golang.org/x/time/rate

**Rationale**: ASTA documentation specifies 10 requests/second per endpoint. Using the same rate limiter pattern as the existing S2 client.

**Implementation**:
```go
limiter := rate.NewLimiter(rate.Limit(10.0), 1) // 10 req/sec, burst of 1
```

## Field Defaults

### Paper Fields

Default fields to request for paper lookups:
```
title,abstract,authors,year,venue,publicationDate,url,citationCount,referenceCount,isOpenAccess,fieldsOfStudy
```

### Author Fields

Default fields for author searches:
```
name,url,affiliations,paperCount,citationCount,hIndex
```

## Paper ID Formats

ASTA supports the same paper ID formats as Semantic Scholar:

| Format | Example |
|--------|---------|
| S2 ID | `649def34f8be52c8b66281af98ae884c09aef38b` |
| CorpusId | `CorpusId:215416146` |
| DOI | `DOI:10.18653/v1/N18-3011` |
| ArXiv | `ARXIV:2106.15928` |
| PubMed | `PMID:19872477` |
| PMC | `PMCID:2323736` |
| URL | `URL:https://arxiv.org/abs/2106.15928v1` |

**Implementation**: Reuse existing `s2.ParsePaperID()` function from `internal/s2/parser.go`.

## Date Range Format

Publication date ranges use the format `<startDate>:<endDate>` with dates in `YYYY-MM-DD` format:

| Example | Meaning |
|---------|---------|
| `2019-03-05` | On March 5th, 2019 |
| `2019-03` | During March 2019 |
| `2019` | During 2019 |
| `2016-03-05:2020-06-06` | Between dates |
| `1981-08-25:` | On or after date |
| `:2015-01` | Before or on date |
| `2015:2020` | Between years |

## Error Handling

### Decision: Map MCP errors to CLI exit codes

| Error Type | Exit Code | Constant |
|------------|-----------|----------|
| Not found | 1 | `ExitASTANotFound` |
| Auth failure | 2 | `ExitASTAAuthError` |
| API/Rate limit | 3 | `ExitASTAAPIError` |

**Rationale**: Follows existing S2 exit code pattern for consistency.

## Sources

- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP JSON-RPC Usage](https://milvus.io/ai-quick-reference/how-is-jsonrpc-used-in-the-model-context-protocol)
- [MCP GitHub Repository](https://github.com/modelcontextprotocol/modelcontextprotocol)
