package zotero

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"golang.org/x/time/rate"
)

const (
	// BaseURL is the Zotero Web API base URL.
	BaseURL = "https://api.zotero.org"

	// APIVersion is the Zotero API version to use.
	APIVersion = "3"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// RateLimit is a conservative rate limit for the Zotero API.
	// Zotero doesn't publish exact limits but recommends being polite.
	RateLimit = 5.0 // 5 req/sec

	// MaxItemsPerPage is the maximum items per request.
	MaxItemsPerPage = 100
)

// Client is a rate-limited HTTP client for the Zotero Web API.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	apiKey     string
	userID     string
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

// WithUserID sets the Zotero user ID.
func WithUserID(id string) ClientOption {
	return func(c *Client) {
		c.userID = id
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

// NewClient creates a new Zotero API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		baseURL:    BaseURL,
		limiter:    rate.NewLimiter(rate.Limit(RateLimit), 3),
	}

	// Load from global config
	if key := config.GetZoteroAPIKey(); key != "" {
		c.apiKey = key
	}
	if uid := config.GetZoteroUserID(); uid != "" {
		c.userID = uid
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Validate
	if c.apiKey == "" || c.userID == "" {
		return nil, ErrNotConfigured
	}

	return c, nil
}

// GetItems fetches all items from the user's library.
// Handles pagination automatically.
func (c *Client) GetItems(ctx context.Context) ([]ZoteroItem, error) {
	var allItems []ZoteroItem
	start := 0

	for {
		endpoint := fmt.Sprintf("%s/users/%s/items/top", c.baseURL, c.userID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		q := req.URL.Query()
		q.Set("limit", strconv.Itoa(MaxItemsPerPage))
		q.Set("start", strconv.Itoa(start))
		q.Set("itemType", "-attachment || note") // Exclude attachments and notes
		req.URL.RawQuery = q.Encode()

		resp, err := c.do(req)
		if err != nil {
			return nil, err
		}

		if err := checkResponse(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}

		var items []ZoteroItem
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()

		allItems = append(allItems, items...)

		// Check if there are more pages via Total-Results header
		totalStr := resp.Header.Get("Total-Results")
		if totalStr == "" {
			break
		}
		total, err := strconv.Atoi(totalStr)
		if err != nil || start+len(items) >= total {
			break
		}
		start += len(items)
	}

	return allItems, nil
}

// GetItemsSince fetches items modified since the given library version.
func (c *Client) GetItemsSince(ctx context.Context, sinceVersion int) ([]ZoteroItem, int, error) {
	var allItems []ZoteroItem
	start := 0
	latestVersion := sinceVersion

	for {
		endpoint := fmt.Sprintf("%s/users/%s/items/top", c.baseURL, c.userID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}

		q := req.URL.Query()
		q.Set("limit", strconv.Itoa(MaxItemsPerPage))
		q.Set("start", strconv.Itoa(start))
		q.Set("since", strconv.Itoa(sinceVersion))
		q.Set("itemType", "-attachment || note")
		req.URL.RawQuery = q.Encode()

		resp, err := c.do(req)
		if err != nil {
			return nil, 0, err
		}

		if err := checkResponse(resp); err != nil {
			resp.Body.Close()
			return nil, 0, err
		}

		// Track the latest library version
		if v := resp.Header.Get("Last-Modified-Version"); v != "" {
			if ver, err := strconv.Atoi(v); err == nil && ver > latestVersion {
				latestVersion = ver
			}
		}

		var items []ZoteroItem
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			return nil, 0, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()

		allItems = append(allItems, items...)

		totalStr := resp.Header.Get("Total-Results")
		if totalStr == "" {
			break
		}
		total, err := strconv.Atoi(totalStr)
		if err != nil || start+len(items) >= total {
			break
		}
		start += len(items)
	}

	return allItems, latestVersion, nil
}

// CreateItem creates a new item in the user's library.
func (c *Client) CreateItem(ctx context.Context, item ZoteroItemData) (*ZoteroItem, error) {
	endpoint := fmt.Sprintf("%s/users/%s/items", c.baseURL, c.userID)

	body, err := json.Marshal([]ZoteroItemData{item})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zotero-Write-Token", generateWriteToken())

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result CreateItemResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Check for failures
	for idx, failure := range result.Failed {
		return nil, fmt.Errorf("item creation failed (index %s): %s", idx, failure.Message)
	}

	// Return the created item
	for _, item := range result.Successful {
		return &item, nil
	}

	return nil, fmt.Errorf("no item returned from Zotero API")
}

// GetLibraryVersion returns the current library version number.
func (c *Client) GetLibraryVersion(ctx context.Context) (int, error) {
	endpoint := fmt.Sprintf("%s/users/%s/items/top", c.baseURL, c.userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}

	q := req.URL.Query()
	q.Set("limit", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := c.do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return 0, err
	}

	vStr := resp.Header.Get("Last-Modified-Version")
	if vStr == "" {
		return 0, nil
	}
	return strconv.Atoi(vStr)
}

// do executes an HTTP request with rate limiting and common headers.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req.Header.Set("Zotero-API-Version", APIVersion)
	if c.apiKey != "" {
		req.Header.Set("Zotero-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}

	// Handle Backoff header (soft rate limit)
	if backoff := resp.Header.Get("Backoff"); backoff != "" {
		if secs, err := strconv.Atoi(backoff); err == nil {
			c.limiter.SetLimit(rate.Every(time.Duration(secs) * time.Second))
		}
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

	switch resp.StatusCode {
	case 403:
		return fmt.Errorf("%w: %v", ErrForbidden, apiErr)
	case 404:
		return fmt.Errorf("%w: %v", ErrNotFound, apiErr)
	case 412:
		return fmt.Errorf("%w: %v", ErrConflict, apiErr)
	case 429:
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				apiErr.RetryAfter = secs
			}
		}
		return fmt.Errorf("%w: %v", ErrRateLimited, apiErr)
	}

	return apiErr
}

// generateWriteToken creates a random 32-char hex token for write requests.
func generateWriteToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
