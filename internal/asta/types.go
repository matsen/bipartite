// Package asta provides a client for the ASTA MCP (Model Context Protocol) API.
package asta

// MCP Protocol Types (JSON-RPC 2.0)

// MCPRequest is the JSON-RPC 2.0 request envelope for MCP tool calls.
type MCPRequest struct {
	JSONRPC string    `json:"jsonrpc"` // Always "2.0"
	ID      int       `json:"id"`      // Request correlation ID
	Method  string    `json:"method"`  // Always "tools/call"
	Params  MCPParams `json:"params"`  // Tool invocation parameters
}

// MCPParams contains tool invocation parameters.
type MCPParams struct {
	Name      string         `json:"name"`      // Tool name (e.g., "search_papers_by_relevance")
	Arguments map[string]any `json:"arguments"` // Tool-specific arguments
}

// MCPResponse is the JSON-RPC 2.0 response envelope.
type MCPResponse struct {
	JSONRPC string     `json:"jsonrpc"`          // Always "2.0"
	ID      int        `json:"id"`               // Matching request ID
	Result  *MCPResult `json:"result,omitempty"` // Success result
	Error   *MCPError  `json:"error,omitempty"`  // Error result
}

// MCPResult contains successful tool results.
type MCPResult struct {
	Content []MCPContent `json:"content"` // Array of content blocks
}

// MCPContent represents a content block in the result.
type MCPContent struct {
	Type string `json:"type"` // Content type ("text")
	Text string `json:"text"` // JSON-encoded tool output
}

// MCPError represents an MCP error response.
type MCPError struct {
	Code    int    `json:"code"`    // Error code
	Message string `json:"message"` // Error message
}

// Domain Types

// ASTAPaper represents a paper returned from ASTA searches.
type ASTAPaper struct {
	PaperID         string       `json:"paperId"`
	Title           string       `json:"title"`
	Abstract        string       `json:"abstract,omitempty"`
	Authors         []ASTAAuthor `json:"authors,omitempty"`
	Year            int          `json:"year,omitempty"`
	Venue           string       `json:"venue,omitempty"`
	PublicationDate string       `json:"publicationDate,omitempty"` // YYYY-MM-DD format
	URL             string       `json:"url,omitempty"`
	CitationCount   int          `json:"citationCount,omitempty"`
	ReferenceCount  int          `json:"referenceCount,omitempty"`
	IsOpenAccess    bool         `json:"isOpenAccess,omitempty"`
	FieldsOfStudy   []string     `json:"fieldsOfStudy,omitempty"`
}

// ASTAAuthor represents author information from ASTA.
type ASTAAuthor struct {
	AuthorID      string   `json:"authorId,omitempty"`
	Name          string   `json:"name"`
	URL           string   `json:"url,omitempty"`
	Affiliations  []string `json:"affiliations,omitempty"`
	PaperCount    int      `json:"paperCount,omitempty"`
	CitationCount int      `json:"citationCount,omitempty"`
	HIndex        int      `json:"hIndex,omitempty"`
}

// ASTASnippet represents a text snippet from snippet search.
type ASTASnippet struct {
	Snippet string           `json:"snippet"`
	Score   float64          `json:"score,omitempty"`
	Paper   ASTAPaperSummary `json:"paper"`
}

// ASTAPaperSummary provides minimal paper info for snippet context.
type ASTAPaperSummary struct {
	PaperID string       `json:"paperId"`
	Title   string       `json:"title"`
	Authors []ASTAAuthor `json:"authors,omitempty"`
	Year    int          `json:"year,omitempty"`
}

// ASTACitation represents a citation result.
type ASTACitation struct {
	CitingPaper ASTAPaper `json:"citingPaper"`
}

// Response types for CLI output

// SearchResponse is the response for paper search.
type SearchResponse struct {
	Total  int         `json:"total"`
	Papers []ASTAPaper `json:"papers"`
}

// SnippetResponse is the response for snippet search.
type SnippetResponse struct {
	Snippets []ASTASnippet `json:"snippets"`
}

// CitationsResponse is the response for citations lookup.
type CitationsResponse struct {
	PaperID       string      `json:"paperId"`
	CitationCount int         `json:"citationCount"`
	Citations     []ASTAPaper `json:"citations"`
}

// ReferencesResponse is the response for references lookup.
type ReferencesResponse struct {
	PaperID        string      `json:"paperId"`
	ReferenceCount int         `json:"referenceCount"`
	References     []ASTAPaper `json:"references"`
}

// AuthorsResponse is the response for author search.
type AuthorsResponse struct {
	Authors []ASTAAuthor `json:"authors"`
}

// AuthorPapersResponse is the response for author papers lookup.
type AuthorPapersResponse struct {
	AuthorID string      `json:"authorId"`
	Name     string      `json:"name"`
	Papers   []ASTAPaper `json:"papers"`
}
