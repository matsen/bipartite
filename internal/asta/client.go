package asta

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"golang.org/x/time/rate"
)

const (
	// BaseURL is the ASTA MCP API base URL.
	BaseURL = "https://asta-tools.allen.ai/mcp/v1"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 60 * time.Second

	// RateLimit is 10 requests per second per ASTA documentation.
	RateLimit = 10.0

	// DefaultPaperFields are the fields requested by default for paper lookups.
	DefaultPaperFields = "title,abstract,authors,year,venue,publicationDate,url,citationCount,referenceCount,isOpenAccess,fieldsOfStudy"

	// DefaultAuthorFields are the fields requested by default for author lookups.
	DefaultAuthorFields = "name,url,affiliations,paperCount,citationCount,hIndex"

	// Default limits for various search operations.
	DefaultSearchLimit       = 50
	DefaultSnippetLimit      = 20
	DefaultCitationsLimit    = 100
	DefaultReferencesLimit   = 100
	DefaultAuthorSearchLimit = 10
	DefaultAuthorPapersLimit = 100
)

// Client is a rate-limited HTTP client for the ASTA MCP API.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	apiKey     string
	baseURL    string
	requestID  atomic.Int32
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithAPIKey sets the API key for authenticated requests.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL sets a custom base URL (for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new ASTA MCP API client.
func NewClient(opts ...ClientOption) *Client {
	// Use a longer timeout for SSE streaming - the server sends pings every 15s
	// and may take a while to process requests
	c := &Client{
		httpClient: &http.Client{Timeout: 3 * time.Minute},
		limiter:    rate.NewLimiter(rate.Limit(RateLimit), 1),
		baseURL:    BaseURL,
	}

	// Check for API key in environment and global config
	if key := config.GetASTAAPIKey(); key != "" {
		c.apiKey = key
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// parseSSEResponse extracts text content from an SSE/MCP response stream.
func parseSSEResponse(body io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var allTextContent []string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip ping events, empty lines, and event type lines
		if strings.HasPrefix(line, ": ping") || line == "" || strings.HasPrefix(line, "event:") {
			continue
		}

		// Look for data: lines containing JSON
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			var mcpResp MCPResponse
			if err := json.Unmarshal([]byte(data), &mcpResp); err != nil {
				continue
			}

			if mcpResp.Error != nil {
				return nil, &APIError{
					StatusCode: mcpResp.Error.Code,
					Code:       "mcp_error",
					Message:    mcpResp.Error.Message,
				}
			}

			if mcpResp.Result != nil {
				for _, content := range mcpResp.Result.Content {
					if content.Type == "text" && content.Text != "" {
						allTextContent = append(allTextContent, content.Text)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SSE stream: %w", err)
	}

	return allTextContent, nil
}

// combineStreamingResults combines multiple streaming responses into a single JSON result.
func combineStreamingResults(textContent []string) ([]byte, error) {
	if len(textContent) == 0 {
		return nil, fmt.Errorf("%w: no content received", ErrInvalidResponse)
	}

	if len(textContent) == 1 {
		return []byte(textContent[0]), nil
	}

	// Multiple responses indicate streaming results - combine into array
	var combined strings.Builder
	combined.WriteString(`{"result":[`)
	for i, text := range textContent {
		if i > 0 {
			combined.WriteString(",")
		}
		combined.WriteString(text)
	}
	combined.WriteString("]}")
	return []byte(combined.String()), nil
}

// checkHTTPErrors returns an error if the HTTP response indicates a problem.
func checkHTTPErrors(resp *http.Response) error {
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("%w: status %d", ErrAuthError, resp.StatusCode)
	}
	if resp.StatusCode == 429 {
		return fmt.Errorf("%w: status %d", ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       "api_error",
			Message:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}
	return nil
}

// callTool executes an MCP tool call and returns the raw JSON result.
func (c *Client) callTool(ctx context.Context, toolName string, args map[string]any) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	reqID := int(c.requestID.Add(1))
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  "tools/call",
		Params: MCPParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPErrors(resp); err != nil {
		return nil, err
	}

	textContent, err := parseSSEResponse(resp.Body)
	if err != nil {
		return nil, err
	}

	return combineStreamingResults(textContent)
}

// SearchPapers searches for papers by keyword relevance.
func (c *Client) SearchPapers(ctx context.Context, keyword string, limit int, dateRange, venues string) (*SearchResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	args := map[string]any{
		"keyword": keyword,
		"fields":  DefaultPaperFields,
		"limit":   limit,
	}
	if dateRange != "" {
		args["publication_date_range"] = dateRange
	}
	if venues != "" {
		args["venues"] = venues
	}

	result, err := c.callTool(ctx, "search_papers_by_relevance", args)
	if err != nil {
		return nil, err
	}

	return parseSearchPapersResult(result)
}

// unwrapStreamedList parses an MCP-streamed list result, accommodating the
// three shapes the server may emit:
//
//  1. wrapped: {"result": [...]} — produced by combineStreamingResults when
//     more than one SSE chunk arrived.
//  2. bare array: [...] — when the upstream tool returns an array directly.
//  3. bare single element: {...} — produced by combineStreamingResults when
//     exactly one SSE chunk arrived (e.g., limit=1, see issue #134).
//
// isValid distinguishes a real bare element from an empty/error object so
// stage 3 doesn't silently swallow malformed responses. label is used to
// build the error message and should match the historical wording for the
// caller (e.g., "search results", "authors", "citations"); tests assert on
// these strings, so keep them stable.
func unwrapStreamedList[T any](result []byte, isValid func(T) bool, label string) ([]T, error) {
	var wrapper struct {
		Result []T `json:"result"`
	}
	wrapErr := json.Unmarshal(result, &wrapper)
	if wrapErr == nil && wrapper.Result != nil {
		return wrapper.Result, nil
	}

	// Stage 2: bare array. JSON `null` also parses cleanly into a nil slice;
	// treat that as "no array shape" rather than success-with-zero so a
	// stray null body doesn't silently look like an empty result set.
	var arr []T
	arrErr := json.Unmarshal(result, &arr)
	if arrErr == nil && arr != nil {
		return arr, nil
	}

	var single T
	if err := json.Unmarshal(result, &single); err == nil && isValid(single) {
		return []T{single}, nil
	}

	switch {
	case wrapErr != nil:
		return nil, fmt.Errorf("%w: parsing %s: %v", ErrInvalidResponse, label, wrapErr)
	case arrErr != nil:
		return nil, fmt.Errorf("%w: parsing %s as array: %v", ErrInvalidResponse, label, arrErr)
	default:
		// Both stages 1 and 2 parsed without error but produced no list
		// (e.g., bare `null` or `{"result":null}`).
		return nil, fmt.Errorf("%w: parsing %s: empty result", ErrInvalidResponse, label)
	}
}

// parseSearchPapersResult parses the raw bytes from a paper-search tool call.
func parseSearchPapersResult(result []byte) (*SearchResponse, error) {
	papers, err := unwrapStreamedList(result, func(p ASTAPaper) bool { return p.PaperID != "" }, "search results")
	if err != nil {
		return nil, err
	}
	return &SearchResponse{Total: len(papers), Papers: papers}, nil
}

// SnippetSearch searches for text snippets within papers.
func (c *Client) SnippetSearch(ctx context.Context, query string, limit int, venues, paperIDs string) (*SnippetResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	args := map[string]any{
		"query": query,
		"limit": limit,
	}
	if venues != "" {
		args["venues"] = venues
	}
	if paperIDs != "" {
		args["paper_ids"] = paperIDs
	}

	result, err := c.callTool(ctx, "snippet_search", args)
	if err != nil {
		return nil, err
	}

	// Parse snippet search response - the text content is {"data": [...]}
	// Try direct format first, then wrapped format for compatibility
	type snippetData struct {
		Score float64 `json:"score"`
		Paper struct {
			CorpusID string   `json:"corpusId"`
			Title    string   `json:"title"`
			Authors  []string `json:"authors"`
		} `json:"paper"`
		Snippet struct {
			Text string `json:"text"`
		} `json:"snippet"`
	}

	var data []snippetData

	// Try direct {"data": [...]} format
	var direct struct {
		Data []snippetData `json:"data"`
	}
	if err := json.Unmarshal(result, &direct); err != nil {
		return nil, fmt.Errorf("%w: parsing snippet results: %v", ErrInvalidResponse, err)
	}
	if direct.Data != nil {
		data = direct.Data
	} else {
		// Try wrapped {"result": {"data": [...]}} format
		var wrapper struct {
			Result struct {
				Data []snippetData `json:"data"`
			} `json:"result"`
		}
		if err := json.Unmarshal(result, &wrapper); err != nil {
			return nil, fmt.Errorf("%w: parsing snippet results: %v", ErrInvalidResponse, err)
		}
		data = wrapper.Result.Data
	}

	snippets := make([]ASTASnippet, len(data))
	for i, r := range data {
		authors := make([]ASTAAuthor, len(r.Paper.Authors))
		for j, name := range r.Paper.Authors {
			authors[j] = ASTAAuthor{Name: name}
		}
		snippets[i] = ASTASnippet{
			Snippet: r.Snippet.Text,
			Score:   r.Score,
			Paper: ASTAPaperSummary{
				PaperID: r.Paper.CorpusID,
				Title:   r.Paper.Title,
				Authors: authors,
			},
		}
	}

	return &SnippetResponse{Snippets: snippets}, nil
}

// GetPaper fetches a paper by its identifier.
func (c *Client) GetPaper(ctx context.Context, paperID string, fields string) (*ASTAPaper, error) {
	if fields == "" {
		fields = DefaultPaperFields
	}

	args := map[string]any{
		"paper_id": paperID,
		"fields":   fields,
	}

	result, err := c.callTool(ctx, "get_paper", args)
	if err != nil {
		return nil, err
	}

	var paper ASTAPaper
	if err := json.Unmarshal(result, &paper); err != nil {
		return nil, fmt.Errorf("%w: parsing paper: %v", ErrInvalidResponse, err)
	}

	if paper.PaperID == "" {
		return nil, ErrNotFound
	}

	return &paper, nil
}

// GetCitations fetches papers that cite the given paper.
func (c *Client) GetCitations(ctx context.Context, paperID string, limit int, dateRange string) (*CitationsResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	args := map[string]any{
		"paper_id": paperID,
		"fields":   "title,authors,year,venue,citationCount",
		"limit":    limit,
	}
	if dateRange != "" {
		args["publication_date_range"] = dateRange
	}

	result, err := c.callTool(ctx, "get_citations", args)
	if err != nil {
		return nil, err
	}

	return parseCitationsResult(result, paperID)
}

// citationEntry matches a single ASTA citation: {"citingPaper": {...}}.
type citationEntry struct {
	CitingPaper ASTAPaper `json:"citingPaper"`
}

// parseCitationsResult parses the raw bytes from a citations tool call.
func parseCitationsResult(result []byte, paperID string) (*CitationsResponse, error) {
	entries, err := unwrapStreamedList(result, func(e citationEntry) bool { return e.CitingPaper.PaperID != "" }, "citations")
	if err != nil {
		return nil, err
	}
	citations := make([]ASTAPaper, len(entries))
	for i, e := range entries {
		citations[i] = e.CitingPaper
	}
	return &CitationsResponse{
		PaperID:       paperID,
		CitationCount: len(citations),
		Citations:     citations,
	}, nil
}

// GetReferences fetches papers referenced by the given paper.
// Note: ASTA doesn't have a direct references endpoint, so we use get_paper with references field.
func (c *Client) GetReferences(ctx context.Context, paperID string, limit int) (*ReferencesResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	args := map[string]any{
		"paper_id": paperID,
		"fields":   "references,references.title,references.authors,references.year,references.venue",
	}

	result, err := c.callTool(ctx, "get_paper", args)
	if err != nil {
		return nil, err
	}

	// Parse paper with references
	var paper struct {
		PaperID    string      `json:"paperId"`
		References []ASTAPaper `json:"references"`
	}
	if err := json.Unmarshal(result, &paper); err != nil {
		return nil, fmt.Errorf("%w: parsing references: %v", ErrInvalidResponse, err)
	}

	refs := paper.References
	if len(refs) > limit {
		refs = refs[:limit]
	}

	return &ReferencesResponse{
		PaperID:        paperID,
		ReferenceCount: len(paper.References),
		References:     refs,
	}, nil
}

// SearchAuthors searches for authors by name.
func (c *Client) SearchAuthors(ctx context.Context, name string, limit int) (*AuthorsResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	args := map[string]any{
		"name":   name,
		"fields": DefaultAuthorFields,
		"limit":  limit,
	}

	result, err := c.callTool(ctx, "search_authors_by_name", args)
	if err != nil {
		return nil, err
	}

	return parseSearchAuthorsResult(result)
}

// parseSearchAuthorsResult parses the raw bytes from an author-search tool call.
func parseSearchAuthorsResult(result []byte) (*AuthorsResponse, error) {
	// AuthorID is omitempty in ASTAAuthor (an unresolved author may have only a
	// name); Name is the field that's always present on a real author record.
	authors, err := unwrapStreamedList(result, func(a ASTAAuthor) bool { return a.Name != "" }, "authors")
	if err != nil {
		return nil, err
	}
	return &AuthorsResponse{Authors: authors}, nil
}

// GetAuthorPapers fetches papers by an author.
func (c *Client) GetAuthorPapers(ctx context.Context, authorID string, limit int, dateRange string) (*AuthorPapersResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	args := map[string]any{
		"author_id":    authorID,
		"paper_fields": "title,year,venue,citationCount",
		"limit":        limit,
	}
	if dateRange != "" {
		args["publication_date_range"] = dateRange
	}

	result, err := c.callTool(ctx, "get_author_papers", args)
	if err != nil {
		return nil, err
	}

	return parseAuthorPapersResult(result, authorID)
}

// parseAuthorPapersResult parses the raw bytes from a get_author_papers tool call.
func parseAuthorPapersResult(result []byte, authorID string) (*AuthorPapersResponse, error) {
	papers, err := unwrapStreamedList(result, func(p ASTAPaper) bool { return p.PaperID != "" }, "author papers")
	if err != nil {
		return nil, err
	}
	return &AuthorPapersResponse{AuthorID: authorID, Papers: papers}, nil
}
