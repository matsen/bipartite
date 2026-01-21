// Package conflict provides domain-aware conflict resolution for refs.jsonl.
package conflict

import (
	"fmt"

	"github.com/matsen/bipartite/internal/reference"
)

// ConflictRegion represents a single git conflict region in a JSONL file.
type ConflictRegion struct {
	// Line numbers in original file (1-indexed)
	StartLine int // Line of <<<<<<< marker
	EndLine   int // Line of >>>>>>> marker

	// Parsed content from each side
	OursRefs   []reference.Reference // Papers from "ours" (HEAD) side
	TheirsRefs []reference.Reference // Papers from "theirs" side

	// Raw content (for error messages)
	OursRaw   string // Raw JSONL content from ours
	TheirsRaw string // Raw JSONL content from theirs
}

// PaperMatch represents a paper that appears on both sides of a conflict.
type PaperMatch struct {
	Ours      reference.Reference // Version from ours side
	Theirs    reference.Reference // Version from theirs side
	MatchedBy string              // "doi" or "id" - how they were matched
}

// FieldConflict represents a true conflict for a specific field.
// Values are stored in full; truncation happens only at display time.
type FieldConflict struct {
	FieldName   string // e.g., "abstract", "title", "venue"
	OursValue   string // Full value from ours side
	TheirsValue string // Full value from theirs side
}

// ResolutionPlan describes how a matched paper pair will be resolved.
type ResolutionPlan struct {
	PaperID string // ID of the paper
	DOI     string // DOI if available

	// Resolution action
	Action ResolutionAction // See enum below
	Reason string           // Human-readable explanation

	// For merge actions, which fields come from where
	FieldSources map[string]string // field name -> "ours" | "theirs" | "merged"

	// True conflicts requiring interactive resolution
	Conflicts []FieldConflict // Empty if auto-resolvable
}

// ResolutionAction indicates the type of resolution applied.
type ResolutionAction string

const (
	ActionKeepOurs   ResolutionAction = "keep_ours"   // Ours is more complete
	ActionKeepTheirs ResolutionAction = "keep_theirs" // Theirs is more complete
	ActionMerge      ResolutionAction = "merge"       // Complementary metadata merged
	ActionAddOurs    ResolutionAction = "add_ours"    // Paper only in ours
	ActionAddTheirs  ResolutionAction = "add_theirs"  // Paper only in theirs
	ActionConflict   ResolutionAction = "conflict"    // True conflict, needs interactive
)

// ParseError represents an error while parsing conflict markers or JSONL.
type ParseError struct {
	Line    int    // Line number where error occurred (1-indexed)
	Message string // Description of the error
	Context string // Surrounding content for debugging
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// ParseResult contains the result of parsing a conflicted file.
type ParseResult struct {
	// Lines outside conflict regions (clean content)
	CleanLines []CleanLine

	// Conflict regions found
	Conflicts []ConflictRegion
}

// CleanLine represents a line outside of any conflict region.
type CleanLine struct {
	LineNum int    // Line number in original file (1-indexed)
	Content string // Line content
}

// MatchResult contains the result of matching papers in a conflict region.
type MatchResult struct {
	// Papers matched between both sides (same DOI or ID)
	Matches []PaperMatch

	// Papers only on ours side (no match on theirs)
	OursOnly []reference.Reference

	// Papers only on theirs side (no match on ours)
	TheirsOnly []reference.Reference
}
