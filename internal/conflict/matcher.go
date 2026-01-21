package conflict

import (
	"github.com/matsen/bipartite/internal/reference"
)

// MatchPapers matches papers between ours and theirs sides of a conflict region.
// It matches by DOI first (primary), then by ID (fallback).
func MatchPapers(region ConflictRegion) MatchResult {
	result := MatchResult{}

	// Build maps for quick lookup
	oursByDOI := make(map[string]reference.Reference)
	oursByID := make(map[string]reference.Reference)
	oursMatched := make(map[string]bool) // Track which ours papers have been matched

	for _, ref := range region.OursRefs {
		if ref.DOI != "" {
			oursByDOI[ref.DOI] = ref
		}
		if ref.ID != "" {
			oursByID[ref.ID] = ref
		}
	}

	// Track which theirs papers have been matched
	theirsMatched := make(map[string]bool)

	// First pass: match by DOI
	for _, theirs := range region.TheirsRefs {
		if theirs.DOI != "" {
			if ours, ok := oursByDOI[theirs.DOI]; ok {
				result.Matches = append(result.Matches, PaperMatch{
					Ours:      ours,
					Theirs:    theirs,
					MatchedBy: "doi",
				})
				oursMatched[ours.ID] = true
				theirsMatched[theirs.ID] = true
			}
		}
	}

	// Second pass: match by ID for papers not yet matched
	for _, theirs := range region.TheirsRefs {
		if theirsMatched[theirs.ID] {
			continue // Already matched by DOI
		}
		if theirs.ID != "" {
			if ours, ok := oursByID[theirs.ID]; ok {
				if !oursMatched[ours.ID] {
					result.Matches = append(result.Matches, PaperMatch{
						Ours:      ours,
						Theirs:    theirs,
						MatchedBy: "id",
					})
					oursMatched[ours.ID] = true
					theirsMatched[theirs.ID] = true
				}
			}
		}
	}

	// Collect unmatched papers from ours
	for _, ref := range region.OursRefs {
		if !oursMatched[ref.ID] {
			result.OursOnly = append(result.OursOnly, ref)
		}
	}

	// Collect unmatched papers from theirs
	for _, ref := range region.TheirsRefs {
		if !theirsMatched[ref.ID] {
			result.TheirsOnly = append(result.TheirsOnly, ref)
		}
	}

	return result
}
