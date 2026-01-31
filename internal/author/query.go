// Package author provides author name parsing and matching for search queries.
package author

import (
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// Query represents a parsed author search query.
type Query struct {
	First string // First name (may be empty for last-name-only queries)
	Last  string // Last name (required)
}

// ParseQuery parses an author search string into a structured Query.
//
// Supported formats:
//   - "Yu"           → last="Yu" (single word = last name only)
//   - "Timothy Yu"   → first="Timothy", last="Yu" (space-separated = First Last)
//   - "Yu, Timothy"  → first="Timothy", last="Yu" (comma = Last, First)
//
// Names are trimmed but case is preserved (matching is case-insensitive).
func ParseQuery(input string) Query {
	input = strings.TrimSpace(input)
	if input == "" {
		return Query{}
	}

	// Check for comma format: "Last, First"
	if idx := strings.Index(input, ","); idx > 0 {
		last := strings.TrimSpace(input[:idx])
		first := strings.TrimSpace(input[idx+1:])
		return Query{First: first, Last: last}
	}

	// Check for space format: "First Last"
	parts := strings.Fields(input)
	if len(parts) == 1 {
		// Single word = last name only
		return Query{Last: parts[0]}
	}

	// Multiple words: last word is last name, rest is first name
	// e.g., "Timothy C Yu" → first="Timothy C", last="Yu"
	last := parts[len(parts)-1]
	first := strings.Join(parts[:len(parts)-1], " ")
	return Query{First: first, Last: last}
}

// Matches checks if the query matches a given author.
//
// Matching rules:
//   - Last name: case-insensitive exact match (required)
//   - First name: case-insensitive prefix match (if query has first name)
//
// This enables "Tim Yu" to match "Timothy C Yu" while preventing
// "Yu" from matching "Yujia" (since "Yu" is not Yujia's last name).
func (q Query) Matches(a reference.Author) bool {
	// Last name must match exactly (case-insensitive)
	if !strings.EqualFold(q.Last, a.Last) {
		return false
	}

	// If no first name in query, we're done
	if q.First == "" {
		return true
	}

	// First name uses prefix matching (case-insensitive)
	// "Tim" matches "Timothy", "Timothy C", etc.
	return strings.HasPrefix(
		strings.ToLower(a.First),
		strings.ToLower(q.First),
	)
}

// MatchesAny checks if the query matches any author in the list.
func (q Query) MatchesAny(authors []reference.Author) bool {
	for _, a := range authors {
		if q.Matches(a) {
			return true
		}
	}
	return false
}

// AllMatch checks if all queries match at least one author each.
// This implements AND logic for multiple author filters.
func AllMatch(queries []Query, authors []reference.Author) bool {
	for _, q := range queries {
		if !q.MatchesAny(authors) {
			return false
		}
	}
	return true
}
