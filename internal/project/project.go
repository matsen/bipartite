// Package project defines the core domain types for project nodes.
package project

import (
	"errors"
	"regexp"
)

// Project represents a logical unit of research work (e.g., a paper being written, a software tool).
type Project struct {
	ID          string `json:"id"`                    // Required: unique identifier
	Name        string `json:"name"`                  // Required: human-readable display name
	Description string `json:"description,omitempty"` // Optional
	CreatedAt   string `json:"created_at,omitempty"`  // RFC3339, auto-set on create
	UpdatedAt   string `json:"updated_at,omitempty"`  // RFC3339, auto-set on update
}

// IDPattern is the regex pattern for valid project IDs.
// Must start with alphanumeric, followed by alphanumeric, hyphens, or underscores.
var IDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Validation errors.
var (
	ErrEmptyID         = errors.New("id is required")
	ErrInvalidID       = errors.New("id must match pattern: lowercase alphanumeric, hyphens, underscores; must start with alphanumeric")
	ErrEmptyName       = errors.New("name is required")
	ErrDuplicateID     = errors.New("project with this id already exists")
	ErrProjectNotFound = errors.New("project not found")
)

// ValidateForCreate validates a project for creation.
// Returns an error if any required field is missing or invalid.
func (p *Project) ValidateForCreate() error {
	if p.ID == "" {
		return ErrEmptyID
	}
	if !IDPattern.MatchString(p.ID) {
		return ErrInvalidID
	}
	if p.Name == "" {
		return ErrEmptyName
	}
	return nil
}

// ValidateID validates just the ID field (useful for lookup operations).
func ValidateID(id string) error {
	if id == "" {
		return ErrEmptyID
	}
	if !IDPattern.MatchString(id) {
		return ErrInvalidID
	}
	return nil
}
