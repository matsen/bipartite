# Feature Specification: ASTA MCP Integration

**Feature Branch**: `005-asta-mcp-integration`
**Created**: 2026-01-20
**Status**: Draft
**Phase**: IV-b
**Input**: CLI wrappers for Allen AI's ASTA (Academic Search Tool API) MCP service

## Overview

ASTA is Allen AI's academic research database, accessed via the Model Context Protocol (MCP). This integration provides CLI commands that call the ASTA MCP endpoint directly, enabling:

- Searching papers by keyword relevance or title
- Searching text snippets within papers
- Looking up paper details, citations, and references
- Finding authors and their publications

This complements the existing `bp s2` commands (Semantic Scholar REST API) by providing access to ASTA's specialized features like snippet search.

## Clarifications

### Session 2026-01-20

- Q: How does ASTA differ from S2? → A: ASTA uses MCP protocol, has snippet search, 10 req/sec rate limit
- Q: Should ASTA commands add papers to collection? → A: No, ASTA is read-only exploration. Use `bp s2 add` to add papers.
- Q: How to authenticate? → A: ASTA_API_KEY in .env, passed via x-api-key header

## ASTA MCP Endpoint

**URL**: `https://asta-tools.allen.ai/mcp/v1`
**Auth**: `x-api-key` header
**Rate Limit**: 10 requests/second per endpoint
**Protocol**: MCP (Model Context Protocol) over HTTP

## User Scenarios & Testing

### User Story 1 - Search Papers by Keyword (Priority: P1)

A researcher wants to find papers on a topic using keyword search.

**Independent Test**: Run `bp asta search "phylogenetic inference"`, verify papers are returned.

**Acceptance Scenarios**:

1. **Given** a keyword query, **When** user runs `bp asta search <query>`, **Then** relevant papers are returned with title, authors, year, venue
2. **Given** search results, **When** `--limit N` flag is used, **Then** at most N papers are returned
3. **Given** search results, **When** `--year 2020:2024` flag is used, **Then** only papers from that date range are returned

---

### User Story 2 - Search Paper Snippets (Priority: P1)

A researcher wants to find specific text passages within papers - ASTA's unique feature.

**Independent Test**: Run `bp asta snippet "variational inference phylogenetics"`, verify text snippets are returned.

**Acceptance Scenarios**:

1. **Given** a query, **When** user runs `bp asta snippet <query>`, **Then** matching text snippets are returned with paper context
2. **Given** snippet results, **Then** each snippet shows the paper it came from (title, authors, year)
3. **Given** `--limit N` flag, **Then** at most N snippets are returned

---

### User Story 3 - Get Paper Details (Priority: P2)

A researcher wants to look up detailed information about a specific paper.

**Independent Test**: Run `bp asta paper DOI:10.1093/sysbio/syy032`, verify paper details are returned.

**Acceptance Scenarios**:

1. **Given** a paper identifier, **When** user runs `bp asta paper <id>`, **Then** paper metadata is returned
2. **Given** `--fields` flag, **Then** only specified fields are returned

---

### User Story 4 - Get Citations/References (Priority: P2)

A researcher wants to explore a paper's citation network.

**Independent Test**: Run `bp asta citations DOI:10.1093/sysbio/syy032`, verify citing papers are returned.

**Acceptance Scenarios**:

1. **Given** a paper ID, **When** user runs `bp asta citations <id>`, **Then** papers citing it are returned
2. **Given** a paper ID, **When** user runs `bp asta references <id>`, **Then** papers it cites are returned
3. **Given** `--limit N` flag, **Then** at most N results are returned

---

### User Story 5 - Search Authors (Priority: P3)

A researcher wants to find an author and their publications.

**Independent Test**: Run `bp asta author "Frederick Matsen"`, verify author info and papers are returned.

**Acceptance Scenarios**:

1. **Given** an author name, **When** user runs `bp asta author <name>`, **Then** matching authors are returned
2. **Given** an author ID, **When** user runs `bp asta author-papers <id>`, **Then** their papers are returned

---

## Requirements

### Functional Requirements

**Search**

- **FR-001**: System MUST search papers via `bp asta search <query>`
- **FR-002**: System MUST search snippets via `bp asta snippet <query>`
- **FR-003**: System MUST support `--limit N` for pagination
- **FR-004**: System MUST support `--year YYYY:YYYY` date range filter
- **FR-005**: System MUST support `--venue` filter

**Paper Lookup**

- **FR-006**: System MUST get paper details via `bp asta paper <id>`
- **FR-007**: System MUST get citations via `bp asta citations <id>`
- **FR-008**: System MUST get references via `bp asta references <id>`
- **FR-009**: Paper IDs MUST support DOI:, ARXIV:, PMID:, CorpusId: formats

**Author Search**

- **FR-010**: System MUST search authors via `bp asta author <name>`
- **FR-011**: System MUST get author's papers via `bp asta author-papers <id>`

**Output**

- **FR-012**: System MUST output JSON by default
- **FR-013**: System MUST support `--human` flag for readable output

### Non-Functional Requirements

- **NFR-001**: System MUST respect ASTA rate limits (10 req/sec)
- **NFR-002**: System MUST read ASTA_API_KEY from .env
- **NFR-003**: System MUST provide clear error messages for auth failures

## Key Entities

- **ASTA API Key**: Stored in .env as `ASTA_API_KEY`
- **MCP Request**: JSON-RPC style request to ASTA endpoint
- **Paper ID formats**: DOI:, ARXIV:, PMID:, CorpusId:, raw S2 ID

## Technical Decisions

- **HTTP Client**: Direct HTTP to MCP endpoint (not using MCP client library)
- **Auth**: x-api-key header from ASTA_API_KEY env var
- **Rate Limiting**: 10 req/sec using golang.org/x/time/rate

## Commands Summary

| Command | Description |
|---------|-------------|
| `bp asta search <query>` | Search papers by keyword relevance |
| `bp asta snippet <query>` | Search text snippets within papers |
| `bp asta paper <id>` | Get paper details |
| `bp asta citations <id>` | Get papers citing this paper |
| `bp asta references <id>` | Get papers this paper cites |
| `bp asta author <name>` | Search for authors |
| `bp asta author-papers <id>` | Get papers by author ID |

## Differences from S2 Commands

| Feature | `bp s2` | `bp asta` |
|---------|---------|-----------|
| API | Semantic Scholar REST | ASTA MCP |
| Rate limit | 1 req/sec | 10 req/sec |
| Snippet search | No | Yes |
| Add to collection | Yes | No (read-only) |
| Auth | S2_API_KEY | ASTA_API_KEY |

## Out of Scope

- Adding papers to collection (use `bp s2 add` for that)
- Caching (ASTA is fast enough, rate limit is generous)
- Batch operations beyond what ASTA provides

## Future Considerations

- Could pipe ASTA search results to `bp s2 add` for bulk import
- Could use snippet search for semantic matching within collection
