// Package edge defines the core domain types for knowledge graph edges.
package edge

import (
	"errors"
	"time"
)

// Edge represents a directed relationship between two papers.
type Edge struct {
	// Identity: (SourceID, TargetID, RelationshipType) tuple
	SourceID         string `json:"source_id"`
	TargetID         string `json:"target_id"`
	RelationshipType string `json:"relationship_type"`

	// Metadata
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Validation errors.
var (
	ErrEmptySourceID         = errors.New("source_id is required")
	ErrEmptyTargetID         = errors.New("target_id is required")
	ErrEmptyRelationshipType = errors.New("relationship_type is required")
	ErrEmptySummary          = errors.New("summary is required")
	ErrSelfEdge              = errors.New("source_id and target_id cannot be the same")
)

// ValidateForCreate validates an edge for creation.
// Returns an error if any required field is missing or invalid.
func (e *Edge) ValidateForCreate() error {
	if e.SourceID == "" {
		return ErrEmptySourceID
	}
	if e.TargetID == "" {
		return ErrEmptyTargetID
	}
	if e.RelationshipType == "" {
		return ErrEmptyRelationshipType
	}
	if e.Summary == "" {
		return ErrEmptySummary
	}
	if e.SourceID == e.TargetID {
		return ErrSelfEdge
	}
	return nil
}

// SetCreatedAt sets the CreatedAt timestamp to the current time if not already set.
func (e *Edge) SetCreatedAt() {
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

// Key returns the unique identity tuple for this edge.
func (e *Edge) Key() EdgeKey {
	return EdgeKey{
		SourceID:         e.SourceID,
		TargetID:         e.TargetID,
		RelationshipType: e.RelationshipType,
	}
}

// EdgeKey represents the unique identity of an edge.
type EdgeKey struct {
	SourceID         string
	TargetID         string
	RelationshipType string
}
