// Package concept defines the core domain types for concept nodes.
package concept

import (
	"errors"
	"regexp"
)

// Concept represents a named idea, method, or phenomenon that papers can relate to.
type Concept struct {
	ID          string   `json:"id"`                    // Required, unique, lowercase alphanumeric + hyphens/underscores
	Name        string   `json:"name"`                  // Required, human-readable display name
	Aliases     []string `json:"aliases,omitempty"`     // Optional, alternative names
	Description string   `json:"description,omitempty"` // Optional, longer explanation
}

// IDPattern is the regex pattern for valid concept IDs.
// Must start with alphanumeric, followed by alphanumeric, hyphens, or underscores.
var IDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Validation errors.
var (
	ErrEmptyID         = errors.New("id is required")
	ErrInvalidID       = errors.New("id must match pattern: lowercase alphanumeric, hyphens, underscores; must start with alphanumeric")
	ErrEmptyName       = errors.New("name is required")
	ErrDuplicateID     = errors.New("concept with this id already exists")
	ErrConceptNotFound = errors.New("concept not found")
	ErrSameIDMerge     = errors.New("source and target concepts cannot be the same")
)

// ValidateForCreate validates a concept for creation.
// Returns an error if any required field is missing or invalid.
func (c *Concept) ValidateForCreate() error {
	if c.ID == "" {
		return ErrEmptyID
	}
	if !IDPattern.MatchString(c.ID) {
		return ErrInvalidID
	}
	if c.Name == "" {
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

// MergeAliases adds aliases from another concept, avoiding duplicates.
// Returns the list of aliases that were added. If other is nil, returns nil.
func (c *Concept) MergeAliases(other *Concept) []string {
	if other == nil {
		return nil
	}
	existingAliases := make(map[string]bool)
	for _, a := range c.Aliases {
		existingAliases[a] = true
	}

	var added []string
	for _, a := range other.Aliases {
		if !existingAliases[a] {
			c.Aliases = append(c.Aliases, a)
			existingAliases[a] = true
			added = append(added, a)
		}
	}

	// Also add the other concept's name as an alias if not already present
	if other.Name != "" && other.Name != c.Name && !existingAliases[other.Name] {
		c.Aliases = append(c.Aliases, other.Name)
		added = append(added, other.Name)
	}

	return added
}
