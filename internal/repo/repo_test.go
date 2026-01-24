package repo

import (
	"testing"
)

func TestRepo_ValidateForCreate(t *testing.T) {
	tests := []struct {
		name    string
		repo    Repo
		wantErr error
	}{
		{
			name: "valid github repo",
			repo: Repo{
				ID:        "dasm2",
				Project:   "dasm2-project",
				Type:      TypeGitHub,
				Name:      "DASM2",
				GitHubURL: "https://github.com/matsen/dasm2",
			},
			wantErr: nil,
		},
		{
			name: "valid manual repo",
			repo: Repo{
				ID:      "internal-tools",
				Project: "dasm2-project",
				Type:    TypeManual,
				Name:    "Internal Tools",
			},
			wantErr: nil,
		},
		{
			name: "valid github repo with all fields",
			repo: Repo{
				ID:          "bipartite",
				Project:     "bipartite-project",
				Type:        TypeGitHub,
				Name:        "bipartite",
				GitHubURL:   "https://github.com/matsen/bipartite",
				Description: "Reference manager",
				Topics:      []string{"cli", "go"},
				Language:    "Go",
			},
			wantErr: nil,
		},
		{
			name: "empty id",
			repo: Repo{
				ID:        "",
				Project:   "dasm2-project",
				Type:      TypeGitHub,
				Name:      "DASM2",
				GitHubURL: "https://github.com/matsen/dasm2",
			},
			wantErr: ErrEmptyID,
		},
		{
			name: "invalid id",
			repo: Repo{
				ID:        "DASM2",
				Project:   "dasm2-project",
				Type:      TypeGitHub,
				Name:      "DASM2",
				GitHubURL: "https://github.com/matsen/dasm2",
			},
			wantErr: ErrInvalidID,
		},
		{
			name: "empty project",
			repo: Repo{
				ID:        "dasm2",
				Project:   "",
				Type:      TypeGitHub,
				Name:      "DASM2",
				GitHubURL: "https://github.com/matsen/dasm2",
			},
			wantErr: ErrEmptyProject,
		},
		{
			name: "empty name",
			repo: Repo{
				ID:        "dasm2",
				Project:   "dasm2-project",
				Type:      TypeGitHub,
				Name:      "",
				GitHubURL: "https://github.com/matsen/dasm2",
			},
			wantErr: ErrEmptyName,
		},
		{
			name: "invalid type",
			repo: Repo{
				ID:      "dasm2",
				Project: "dasm2-project",
				Type:    "gitlab",
				Name:    "DASM2",
			},
			wantErr: ErrInvalidType,
		},
		{
			name: "github type without url",
			repo: Repo{
				ID:      "dasm2",
				Project: "dasm2-project",
				Type:    TypeGitHub,
				Name:    "DASM2",
			},
			wantErr: ErrMissingGitHubURL,
		},
		{
			name: "manual type without url is ok",
			repo: Repo{
				ID:      "internal-tools",
				Project: "dasm2-project",
				Type:    TypeManual,
				Name:    "Internal Tools",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.repo.ValidateForCreate()
			if err != tt.wantErr {
				t.Errorf("ValidateForCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "valid lowercase",
			id:      "dasm2",
			wantErr: nil,
		},
		{
			name:    "valid with hyphen",
			id:      "dasm2-code",
			wantErr: nil,
		},
		{
			name:    "valid with underscore",
			id:      "dasm2_code",
			wantErr: nil,
		},
		{
			name:    "empty",
			id:      "",
			wantErr: ErrEmptyID,
		},
		{
			name:    "uppercase",
			id:      "DASM2",
			wantErr: ErrInvalidID,
		},
		{
			name:    "starts with hyphen",
			id:      "-dasm2",
			wantErr: ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID(tt.id)
			if err != tt.wantErr {
				t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
