package github

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		// Full HTTPS URLs
		{
			name:      "https url",
			input:     "https://github.com/matsen/bipartite",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		{
			name:      "https url with .git",
			input:     "https://github.com/matsen/bipartite.git",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		{
			name:      "http url",
			input:     "http://github.com/matsen/bipartite",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		// Without protocol
		{
			name:      "without protocol",
			input:     "github.com/matsen/bipartite",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		{
			name:      "without protocol with .git",
			input:     "github.com/matsen/bipartite.git",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		// Shorthand
		{
			name:      "shorthand",
			input:     "matsen/bipartite",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		{
			name:      "shorthand with hyphen",
			input:     "matsen/dasm2-paper",
			wantOwner: "matsen",
			wantRepo:  "dasm2-paper",
			wantErr:   false,
		},
		{
			name:      "shorthand with underscore",
			input:     "matsen/dasm2_code",
			wantOwner: "matsen",
			wantRepo:  "dasm2_code",
			wantErr:   false,
		},
		// With whitespace
		{
			name:      "with leading/trailing whitespace",
			input:     "  matsen/bipartite  ",
			wantOwner: "matsen",
			wantRepo:  "bipartite",
			wantErr:   false,
		},
		// Invalid inputs
		{
			name:    "no slash",
			input:   "matsen",
			wantErr: true,
		},
		{
			name:    "too many slashes in shorthand",
			input:   "matsen/bipartite/extra",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "just slash",
			input:   "/",
			wantErr: true,
		},
		{
			name:    "gitlab url",
			input:   "https://gitlab.com/matsen/bipartite",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitHubURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGitHubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("ParseGitHubURL() owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("ParseGitHubURL() repo = %v, want %v", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestNormalizeGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "https url",
			input: "https://github.com/matsen/bipartite",
			want:  "https://github.com/matsen/bipartite",
		},
		{
			name:  "shorthand",
			input: "matsen/bipartite",
			want:  "https://github.com/matsen/bipartite",
		},
		{
			name:  "without protocol",
			input: "github.com/matsen/bipartite",
			want:  "https://github.com/matsen/bipartite",
		},
		{
			name:  "with .git",
			input: "https://github.com/matsen/bipartite.git",
			want:  "https://github.com/matsen/bipartite",
		},
		{
			name:    "invalid",
			input:   "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeGitHubURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeGitHubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeGitHubURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeriveRepoID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "lowercase repo",
			input: "matsen/bipartite",
			want:  "bipartite",
		},
		{
			name:  "mixed case repo",
			input: "matsen/Bipartite",
			want:  "bipartite",
		},
		{
			name:  "full url",
			input: "https://github.com/matsen/DASM2",
			want:  "dasm2",
		},
		{
			name:  "with hyphen",
			input: "matsen/dasm2-paper",
			want:  "dasm2-paper",
		},
		{
			name:    "invalid",
			input:   "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveRepoID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveRepoID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeriveRepoID() = %v, want %v", got, tt.want)
			}
		})
	}
}
