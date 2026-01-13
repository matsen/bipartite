package embedding

import "testing"

func TestEmbedding_Dimensions(t *testing.T) {
	tests := []struct {
		name     string
		vector   []float32
		expected int
	}{
		{
			name:     "384 dimensions",
			vector:   make([]float32, 384),
			expected: 384,
		},
		{
			name:     "empty vector",
			vector:   []float32{},
			expected: 0,
		},
		{
			name:     "small vector",
			vector:   []float32{1.0, 2.0, 3.0},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb := Embedding{Vector: tt.vector}
			if got := emb.Dimensions(); got != tt.expected {
				t.Errorf("Dimensions() = %d, want %d", got, tt.expected)
			}
		})
	}
}
