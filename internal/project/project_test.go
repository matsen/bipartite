package project

import (
	"testing"
)

func TestProject_ValidateForCreate(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		wantErr error
	}{
		{
			name:    "valid project",
			project: Project{ID: "dasm2", Name: "DASM2"},
			wantErr: nil,
		},
		{
			name:    "valid project with description",
			project: Project{ID: "phylo-review", Name: "Phylo Review", Description: "A review paper"},
			wantErr: nil,
		},
		{
			name:    "valid project with underscore",
			project: Project{ID: "my_project", Name: "My Project"},
			wantErr: nil,
		},
		{
			name:    "valid project with hyphen",
			project: Project{ID: "my-project", Name: "My Project"},
			wantErr: nil,
		},
		{
			name:    "empty id",
			project: Project{ID: "", Name: "DASM2"},
			wantErr: ErrEmptyID,
		},
		{
			name:    "empty name",
			project: Project{ID: "dasm2", Name: ""},
			wantErr: ErrEmptyName,
		},
		{
			name:    "id with uppercase",
			project: Project{ID: "DASM2", Name: "DASM2"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "id starting with hyphen",
			project: Project{ID: "-dasm2", Name: "DASM2"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "id starting with underscore",
			project: Project{ID: "_dasm2", Name: "DASM2"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "id with space",
			project: Project{ID: "my project", Name: "My Project"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "id with special characters",
			project: Project{ID: "my@project", Name: "My Project"},
			wantErr: ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.ValidateForCreate()
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
			id:      "my-project",
			wantErr: nil,
		},
		{
			name:    "valid with underscore",
			id:      "my_project",
			wantErr: nil,
		},
		{
			name:    "valid numeric start",
			id:      "123project",
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
			id:      "-project",
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

func TestIDPattern(t *testing.T) {
	validIDs := []string{
		"a",
		"abc",
		"123",
		"a1",
		"1a",
		"my-project",
		"my_project",
		"a-b-c",
		"a_b_c",
		"abc123-def_456",
	}

	invalidIDs := []string{
		"",
		"-abc",
		"_abc",
		"ABC",
		"Abc",
		"abc def",
		"abc@def",
		"abc.def",
	}

	for _, id := range validIDs {
		if !IDPattern.MatchString(id) {
			t.Errorf("IDPattern should match %q", id)
		}
	}

	for _, id := range invalidIDs {
		if IDPattern.MatchString(id) {
			t.Errorf("IDPattern should not match %q", id)
		}
	}
}
