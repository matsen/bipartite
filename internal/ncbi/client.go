package ncbi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	// BaseURL is the NCBI PMC ID Converter API base URL.
	BaseURL = "https://pmc.ncbi.nlm.nih.gov/tools/idconv/api/v1/articles/"

	// MaxBatchSize is the maximum IDs per request, per NCBI documentation:
	// "The API service allows for up to 200 IDs in a single request."
	MaxBatchSize = 200

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 60 * time.Second

	// RateLimit is conservative (3 req/sec) for use without an API key, which
	// is NCBI's documented unauthenticated ceiling. The endpoint does not
	// require a key.
	RateLimit = 3.0

	// DefaultTool is the identification string sent in the `tool` param. NCBI
	// requests but does not require this for usage tracking.
	DefaultTool = "bipartite"
)

// IDType identifies which kind of NCBI identifier an input ID is. Required
// because the converter rejects batches that mix types.
type IDType string

const (
	// IDTypeDOI is a Digital Object Identifier (e.g., 10.1038/nature12373).
	IDTypeDOI IDType = "doi"

	// IDTypePMID is a PubMed ID (e.g., 23903748).
	IDTypePMID IDType = "pmid"
)

// Input pairs an ID with its type. Used to batch heterogeneous inputs.
type Input struct {
	Type IDType
	ID   string
}

// Client is a rate-limited HTTP client for the NCBI ID Converter.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	baseURL    string
	tool       string
	email      string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL sets a custom base URL (for testing).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithTool sets the `tool` identification parameter sent with each request.
func WithTool(tool string) ClientOption {
	return func(c *Client) {
		c.tool = tool
	}
}

// WithEmail sets the `email` identification parameter sent with each request.
// NCBI requests this for contact in case of abuse but does not enforce it.
func WithEmail(email string) ClientOption {
	return func(c *Client) {
		c.email = email
	}
}

// NewClient creates a new NCBI ID Converter client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		limiter:    rate.NewLimiter(rate.Limit(RateLimit), 1),
		baseURL:    BaseURL,
		tool:       DefaultTool,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Convert resolves a heterogeneous list of inputs to NCBI records. Inputs are
// grouped by IDType, then batched into chunks of at most MaxBatchSize each.
// Results from all batches are concatenated; callers should match records to
// inputs via Record.RequestedID, not positional index.
//
// A per-record failure (e.g., "not in PMC") is reported in-line as a Record
// with Status="error" and is not an error. A request-wide failure (HTTP error,
// or status="error" with no records) returns an *APIError and aborts; later
// batches are not attempted.
func (c *Client) Convert(ctx context.Context, inputs []Input) ([]Record, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Group by ID type — NCBI rejects mixed-type batches.
	byType := make(map[IDType][]string)
	for _, in := range inputs {
		byType[in.Type] = append(byType[in.Type], in.ID)
	}

	var all []Record
	for idType, ids := range byType {
		for start := 0; start < len(ids); start += MaxBatchSize {
			end := start + MaxBatchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[start:end]
			recs, err := c.convertBatch(ctx, idType, batch)
			if err != nil {
				return nil, err
			}
			all = append(all, recs...)
		}
	}

	return all, nil
}

// convertBatch issues a single request for one batch of same-type IDs.
func (c *Client) convertBatch(ctx context.Context, idType IDType, ids []string) ([]Record, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	q := url.Values{}
	q.Set("format", "json")
	q.Set("idtype", string(idType))
	q.Set("ids", strings.Join(ids, ","))
	if c.tool != "" {
		q.Set("tool", c.tool)
	}
	if c.email != "" {
		q.Set("email", c.email)
	}

	reqURL := c.baseURL + "?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: reading body: %v", ErrNetworkError, err)
	}

	if resp.StatusCode == 429 {
		// Wrap ErrRateLimited so callers can use errors.Is or IsRateLimited
		// to handle this case without inspecting *APIError.
		return nil, fmt.Errorf("%w: HTTP %d (batch: %s)",
			ErrRateLimited, resp.StatusCode, strings.Join(ids, ","))
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "http_error",
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200)),
			BatchIDs:   ids,
		}
	}

	var parsed Response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if parsed.Status != "ok" {
		msg := "unknown error"
		code := "api_error"
		if len(parsed.Errors) > 0 {
			msg = parsed.Errors[0].Message
			code = parsed.Errors[0].Code
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       code,
			Message:    msg,
			BatchIDs:   ids,
		}
	}

	return parsed.Records, nil
}

// truncate clips s to n runes with an ellipsis suffix.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
