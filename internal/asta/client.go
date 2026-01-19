package asta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	// BaseURL is the Semantic Scholar API base URL.
	BaseURL = "https://api.semanticscholar.org/graph/v1"

	// DefaultFields are the fields requested by default for paper lookups.
	DefaultFields = "paperId,externalIds,title,abstract,authors,year,venue,publicationDate,citationCount,referenceCount,isOpenAccess,fieldsOfStudy"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// UnauthenticatedRateLimit is the rate limit without an API key (100 req / 5 min).
	UnauthenticatedRateLimit = 100.0 / 300.0 // ~0.33 req/sec

	// AuthenticatedRateLimit is the rate limit with an API key (1 req/sec).
	AuthenticatedRateLimit = 1.0
)

// Client is a rate-limited HTTP client for the Semantic Scholar API.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	apiKey     string
	baseURL    string
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

// NewClient creates a new Semantic Scholar API client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		baseURL:    BaseURL,
	}

	// Check for API key in environment
	if key := os.Getenv("S2_API_KEY"); key != "" {
		c.apiKey = key
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Set rate limiter based on authentication
	if c.apiKey != "" {
		c.limiter = rate.NewLimiter(rate.Limit(AuthenticatedRateLimit), 1)
	} else {
		c.limiter = rate.NewLimiter(rate.Limit(UnauthenticatedRateLimit), 1)
	}

	return c
}

// GetPaper fetches a paper by its identifier.
func (c *Client) GetPaper(ctx context.Context, paperID string) (*S2Paper, error) {
	endpoint := fmt.Sprintf("%s/paper/%s", c.baseURL, url.PathEscape(paperID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add fields parameter
	q := req.URL.Query()
	q.Set("fields", DefaultFields)
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var paper S2Paper
	if err := json.NewDecoder(resp.Body).Decode(&paper); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &paper, nil
}

// GetCitations fetches papers that cite the given paper.
func (c *Client) GetCitations(ctx context.Context, paperID string, limit int) (*CitationsResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	endpoint := fmt.Sprintf("%s/paper/%s/citations", c.baseURL, url.PathEscape(paperID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	q := req.URL.Query()
	q.Set("fields", "paperId,externalIds,title,authors,year,venue")
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result CitationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// GetReferences fetches papers referenced by the given paper.
func (c *Client) GetReferences(ctx context.Context, paperID string, limit int) (*ReferencesResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	endpoint := fmt.Sprintf("%s/paper/%s/references", c.baseURL, url.PathEscape(paperID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	q := req.URL.Query()
	q.Set("fields", "paperId,externalIds,title,authors,year,venue")
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result ReferencesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// SearchByTitle searches for papers by title.
func (c *Client) SearchByTitle(ctx context.Context, title string, limit int) (*SearchResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	endpoint := fmt.Sprintf("%s/paper/search", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	q := req.URL.Query()
	q.Set("query", title)
	q.Set("fields", DefaultFields)
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// do executes an HTTP request with rate limiting.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Set common headers
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}

	return resp, nil
}

// checkResponse checks for API errors.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	msg := string(body)
	if msg == "" {
		msg = resp.Status
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    strings.TrimSpace(msg),
	}

	// Parse Retry-After header for rate limits
	if resp.StatusCode == 429 {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				apiErr.RetryAfter = secs
			}
		}
		return fmt.Errorf("%w: %v", ErrRateLimited, apiErr)
	}

	return apiErr
}
