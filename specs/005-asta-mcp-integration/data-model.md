# Data Model: ASTA MCP Integration

**Phase 1 Output** | **Date**: 2026-01-20

## Entities

### MCPRequest

MCP JSON-RPC request envelope.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| jsonrpc | string | Yes | Always "2.0" |
| id | int | Yes | Request correlation ID |
| method | string | Yes | Always "tools/call" |
| params | MCPParams | Yes | Tool invocation parameters |

### MCPParams

Tool invocation parameters.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Tool name (e.g., "search_papers_by_relevance") |
| arguments | map[string]any | Yes | Tool-specific arguments |

### MCPResponse

MCP JSON-RPC response envelope.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| jsonrpc | string | Yes | Always "2.0" |
| id | int | Yes | Matching request ID |
| result | MCPResult | No | Success result |
| error | MCPError | No | Error result |

### MCPResult

Successful tool result.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | []MCPContent | Yes | Array of content blocks |

### MCPContent

Content block in result.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| type | string | Yes | Content type ("text") |
| text | string | Yes | JSON-encoded tool output |

### MCPError

Error response.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| code | int | Yes | Error code |
| message | string | Yes | Error message |

---

## Domain Types

### ASTAPaper

Paper returned from ASTA searches.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| PaperID | string | `paperId` | Semantic Scholar paper ID |
| Title | string | `title` | Paper title |
| Abstract | string | `abstract` | Paper abstract |
| Authors | []ASTAAuthor | `authors` | Author list |
| Year | int | `year` | Publication year |
| Venue | string | `venue` | Publication venue |
| PublicationDate | string | `publicationDate` | YYYY-MM-DD format |
| URL | string | `url` | S2 URL |
| CitationCount | int | `citationCount` | Number of citations |
| ReferenceCount | int | `referenceCount` | Number of references |
| IsOpenAccess | bool | `isOpenAccess` | Open access flag |
| FieldsOfStudy | []string | `fieldsOfStudy` | Research fields |

### ASTAAuthor

Author information.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| AuthorID | string | `authorId` | S2 author ID |
| Name | string | `name` | Display name |
| URL | string | `url` | S2 profile URL |
| Affiliations | []string | `affiliations` | Institutional affiliations |
| PaperCount | int | `paperCount` | Total papers |
| CitationCount | int | `citationCount` | Total citations |
| HIndex | int | `hIndex` | h-index |

### ASTASnippet

Text snippet from snippet search.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| Snippet | string | `snippet` | Matched text |
| Score | float64 | `score` | Relevance score |
| Paper | ASTAPaperSummary | - | Paper context |

### ASTAPaperSummary

Minimal paper info for snippet context.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| PaperID | string | `paperId` | S2 paper ID |
| Title | string | `title` | Paper title |
| Authors | []ASTAAuthor | `authors` | Author list |
| Year | int | `year` | Publication year |

### ASTACitation

Citation result (paper citing another).

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| CitingPaper | ASTAPaper | `citingPaper` | The citing paper |

---

## Relationships

```
MCPRequest 1:1 MCPResponse (via id)
ASTAPaper *:* ASTAAuthor (many-to-many)
ASTASnippet *:1 ASTAPaper (snippet belongs to paper)
ASTACitation *:1 ASTAPaper (citation references paper)
```

---

## Validation Rules

| Entity | Field | Rule |
|--------|-------|------|
| MCPRequest | jsonrpc | Must be "2.0" |
| MCPRequest | method | Must be "tools/call" |
| MCPRequest | id | Must be positive integer |
| ASTAPaper | PaperID | Non-empty string |
| ASTAAuthor | AuthorID | Non-empty string for author-papers lookup |
