package embedding

import (
	"strings"
	"testing"
	"time"
)

func TestNewOllamaProvider_Defaults(t *testing.T) {
	provider := NewOllamaProvider()

	if provider.baseURL != DefaultOllamaURL {
		t.Errorf("baseURL = %s, want %s", provider.baseURL, DefaultOllamaURL)
	}
	if provider.model != DefaultModel {
		t.Errorf("model = %s, want %s", provider.model, DefaultModel)
	}
	if provider.dimensions != DefaultDimensions {
		t.Errorf("dimensions = %d, want %d", provider.dimensions, DefaultDimensions)
	}
	if provider.client == nil {
		t.Error("client should not be nil")
	}
}

func TestNewOllamaProvider_WithOptions(t *testing.T) {
	customURL := "http://custom:8080"
	customModel := "custom-model"
	customDimensions := 768
	customTimeout := 60 * time.Second

	provider := NewOllamaProvider(
		WithBaseURL(customURL),
		WithModel(customModel),
		WithDimensions(customDimensions),
		WithTimeout(customTimeout),
	)

	if provider.baseURL != customURL {
		t.Errorf("baseURL = %s, want %s", provider.baseURL, customURL)
	}
	if provider.model != customModel {
		t.Errorf("model = %s, want %s", provider.model, customModel)
	}
	if provider.dimensions != customDimensions {
		t.Errorf("dimensions = %d, want %d", provider.dimensions, customDimensions)
	}
	if provider.client.Timeout != customTimeout {
		t.Errorf("timeout = %v, want %v", provider.client.Timeout, customTimeout)
	}
}

func TestOllamaProvider_ModelName(t *testing.T) {
	provider := NewOllamaProvider()
	if provider.ModelName() != DefaultModel {
		t.Errorf("ModelName() = %s, want %s", provider.ModelName(), DefaultModel)
	}

	customModel := "custom-model"
	provider2 := NewOllamaProvider(WithModel(customModel))
	if provider2.ModelName() != customModel {
		t.Errorf("ModelName() = %s, want %s", provider2.ModelName(), customModel)
	}
}

func TestOllamaProvider_Dimensions(t *testing.T) {
	provider := NewOllamaProvider()
	if provider.Dimensions() != DefaultDimensions {
		t.Errorf("Dimensions() = %d, want %d", provider.Dimensions(), DefaultDimensions)
	}

	customDimensions := 768
	provider2 := NewOllamaProvider(WithDimensions(customDimensions))
	if provider2.Dimensions() != customDimensions {
		t.Errorf("Dimensions() = %d, want %d", provider2.Dimensions(), customDimensions)
	}
}

func TestFormatErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple error message",
			input:    "error occurred",
			expected: "error occurred",
		},
		{
			name:     "empty body",
			input:    "",
			expected: "",
		},
		{
			name:     "json error",
			input:    `{"error": "not found"}`,
			expected: `{"error": "not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorBody(strings.NewReader(tt.input))
			if result != tt.expected {
				t.Errorf("formatErrorBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOllamaProvider_ImplementsProvider(t *testing.T) {
	// Compile-time check that OllamaProvider implements Provider interface
	var _ Provider = (*OllamaProvider)(nil)
}
