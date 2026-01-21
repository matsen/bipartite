package concept

import (
	"testing"
)

func TestValidateForCreate(t *testing.T) {
	tests := []struct {
		name    string
		concept Concept
		wantErr error
	}{
		{
			name: "valid concept",
			concept: Concept{
				ID:          "somatic-hypermutation",
				Name:        "Somatic Hypermutation",
				Aliases:     []string{"SHM"},
				Description: "A process",
			},
			wantErr: nil,
		},
		{
			name: "valid with underscores",
			concept: Concept{
				ID:   "bcr_sequencing",
				Name: "BCR Sequencing",
			},
			wantErr: nil,
		},
		{
			name: "valid numeric start",
			concept: Concept{
				ID:   "5ht-receptor",
				Name: "5-HT Receptor",
			},
			wantErr: nil,
		},
		{
			name: "empty id",
			concept: Concept{
				ID:   "",
				Name: "Test",
			},
			wantErr: ErrEmptyID,
		},
		{
			name: "invalid id - uppercase",
			concept: Concept{
				ID:   "SHM",
				Name: "Test",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "invalid id - starts with hyphen",
			concept: Concept{
				ID:   "-hypermutation",
				Name: "Test",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "invalid id - starts with underscore",
			concept: Concept{
				ID:   "_hypermutation",
				Name: "Test",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "invalid id - contains space",
			concept: Concept{
				ID:   "somatic hypermutation",
				Name: "Test",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "invalid id - contains dots",
			concept: Concept{
				ID:   "somatic.hypermutation",
				Name: "Test",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "empty name",
			concept: Concept{
				ID:   "test",
				Name: "",
			},
			wantErr: ErrEmptyName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.concept.ValidateForCreate()
			if err != tt.wantErr {
				t.Errorf("ValidateForCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr error
	}{
		{"somatic-hypermutation", nil},
		{"bcr_sequencing", nil},
		{"5ht-receptor", nil},
		{"phylogenetics", nil},
		{"", ErrEmptyID},
		{"SHM", ErrInvalidID},
		{"-hyphen-start", ErrInvalidID},
		{"_underscore-start", ErrInvalidID},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := ValidateID(tt.id)
			if err != tt.wantErr {
				t.Errorf("ValidateID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestMergeAliases(t *testing.T) {
	target := &Concept{
		ID:      "somatic-hypermutation",
		Name:    "Somatic Hypermutation",
		Aliases: []string{"SHM"},
	}

	source := &Concept{
		ID:      "shm",
		Name:    "SHM Process",
		Aliases: []string{"hypermutation", "SHM"}, // SHM is duplicate
	}

	added := target.MergeAliases(source)

	// Should have added "hypermutation" and "SHM Process" (source name)
	// Should NOT have added "SHM" (already exists)
	if len(added) != 2 {
		t.Errorf("MergeAliases() added %d aliases, want 2: %v", len(added), added)
	}

	expectedAliases := map[string]bool{
		"SHM":           true,
		"hypermutation": true,
		"SHM Process":   true,
	}

	for _, a := range target.Aliases {
		if !expectedAliases[a] {
			t.Errorf("MergeAliases() unexpected alias %q", a)
		}
	}
}
