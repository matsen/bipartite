package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultOllamaURL is the default Ollama API endpoint.
	DefaultOllamaURL = "http://localhost:11434"

	// DefaultModel is the default embedding model.
	DefaultModel = "all-minilm:l6-v2"

	// DefaultDimensions is the expected output dimensions for all-minilm.
	DefaultDimensions = 384

	// DefaultTimeout is the timeout for embedding requests.
	DefaultTimeout = 30 * time.Second

	// apiPathTags is the Ollama API endpoint for listing models.
	apiPathTags = "/api/tags"

	// apiPathEmbeddings is the Ollama API endpoint for generating embeddings.
	apiPathEmbeddings = "/api/embeddings"
)

// OllamaProvider generates embeddings using the Ollama API.
type OllamaProvider struct {
	baseURL    string
	model      string
	dimensions int
	client     *http.Client
}

// OllamaOption configures an OllamaProvider.
type OllamaOption func(*OllamaProvider)

// WithBaseURL sets the Ollama API base URL.
func WithBaseURL(url string) OllamaOption {
	return func(p *OllamaProvider) {
		p.baseURL = url
	}
}

// WithModel sets the embedding model.
func WithModel(model string) OllamaOption {
	return func(p *OllamaProvider) {
		p.model = model
	}
}

// WithDimensions sets the expected vector dimensions.
func WithDimensions(dims int) OllamaOption {
	return func(p *OllamaProvider) {
		p.dimensions = dims
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) OllamaOption {
	return func(p *OllamaProvider) {
		p.client.Timeout = timeout
	}
}

// NewOllamaProvider creates a new Ollama embedding provider.
func NewOllamaProvider(opts ...OllamaOption) *OllamaProvider {
	p := &OllamaProvider{
		baseURL:    DefaultOllamaURL,
		model:      DefaultModel,
		dimensions: DefaultDimensions,
		client:     &http.Client{Timeout: DefaultTimeout},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// doGet performs a GET request to the specified path and returns the response.
// The caller is responsible for closing the response body.
func (p *OllamaProvider) doGet(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// formatErrorBody reads and formats the response body for error messages.
func formatErrorBody(body io.Reader) string {
	respBody, err := io.ReadAll(body)
	if err != nil {
		return fmt.Sprintf("(failed to read response body: %v)", err)
	}
	return string(respBody)
}

// Embed generates an embedding for the given text.
func (p *OllamaProvider) Embed(ctx context.Context, text string) (Embedding, error) {
	reqBody := ollamaEmbedRequest{
		Model:  p.model,
		Prompt: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Embedding{}, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+apiPathEmbeddings, bytes.NewReader(body))
	if err != nil {
		return Embedding{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Embedding{}, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Embedding{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, formatErrorBody(resp.Body))
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Embedding{}, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Embedding) != p.dimensions {
		return Embedding{}, fmt.Errorf("unexpected embedding dimensions: got %d, want %d", len(result.Embedding), p.dimensions)
	}

	return Embedding{Vector: result.Embedding}, nil
}

// ModelName returns the name of the embedding model.
func (p *OllamaProvider) ModelName() string {
	return p.model
}

// Dimensions returns the expected vector dimensions.
func (p *OllamaProvider) Dimensions() int {
	return p.dimensions
}

// IsAvailable checks if Ollama is running and accessible.
func (p *OllamaProvider) IsAvailable(ctx context.Context) error {
	resp, err := p.doGet(ctx, apiPathTags)
	if err != nil {
		return fmt.Errorf("ollama is not running: %w", err)
	}
	resp.Body.Close()
	return nil
}

// HasModel checks if the required model is available in Ollama.
func (p *OllamaProvider) HasModel(ctx context.Context) (bool, error) {
	resp, err := p.doGet(ctx, apiPathTags)
	if err != nil {
		return false, fmt.Errorf("checking models: %w", err)
	}
	defer resp.Body.Close()

	var result ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decoding response: %w", err)
	}

	for _, m := range result.Models {
		if m.Name == p.model {
			return true, nil
		}
	}

	return false, nil
}

// ollamaEmbedRequest is the request body for the Ollama embeddings API.
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the response from the Ollama embeddings API.
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// ollamaTagsResponse is the response from the Ollama tags API.
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

// ollamaModel represents a model in the Ollama tags response.
type ollamaModel struct {
	Name string `json:"name"`
}
