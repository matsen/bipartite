package conflict

import (
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// Field completeness weights (higher = more important)
const (
	weightAbstract  = 5
	weightAuthors   = 4
	weightVenue     = 3
	weightPublished = 2
	weightDOI       = 1
)

// Resolve determines the resolution plan for a matched paper pair.
func Resolve(match PaperMatch) ResolutionPlan {
	plan := ResolutionPlan{
		PaperID:      match.Ours.ID,
		DOI:          match.Ours.DOI,
		FieldSources: make(map[string]string),
	}

	// Try to merge and check for conflicts
	merged, conflicts := MergeReferences(match.Ours, match.Theirs)
	_ = merged // Used when action is merge

	if len(conflicts) > 0 {
		plan.Action = ActionConflict
		plan.Conflicts = conflicts
		plan.Reason = "true conflicts on: " + conflictFieldNames(conflicts)
		return plan
	}

	// No conflicts - determine best action
	oursScore := ComputeCompleteness(match.Ours)
	theirsScore := ComputeCompleteness(match.Theirs)

	// Check if it's a complementary merge situation
	if isComplementary(match.Ours, match.Theirs) {
		plan.Action = ActionMerge
		plan.Reason = "complementary metadata merged"
		return plan
	}

	// Special case: check author list length when scores are tied
	// Longer author list is more complete
	if oursScore == theirsScore {
		oursAuthors := len(match.Ours.Authors)
		theirsAuthors := len(match.Theirs.Authors)
		if theirsAuthors > oursAuthors {
			plan.Action = ActionKeepTheirs
			plan.Reason = "theirs has more authors"
			return plan
		} else if oursAuthors > theirsAuthors {
			plan.Action = ActionKeepOurs
			plan.Reason = "ours has more authors"
			return plan
		}
	}

	if oursScore >= theirsScore {
		plan.Action = ActionKeepOurs
		if oursScore > theirsScore {
			plan.Reason = "ours is more complete"
		} else {
			plan.Reason = "identical content, keeping ours"
		}
	} else {
		plan.Action = ActionKeepTheirs
		plan.Reason = "theirs is more complete"
	}

	return plan
}

// isComplementary returns true if the two references have complementary fields
// (each has fields the other lacks, and no conflicts).
func isComplementary(ours, theirs reference.Reference) bool {
	oursHasExtra := false
	theirsHasExtra := false

	// Check string fields
	if ours.Abstract != "" && theirs.Abstract == "" {
		oursHasExtra = true
	}
	if theirs.Abstract != "" && ours.Abstract == "" {
		theirsHasExtra = true
	}

	if ours.Venue != "" && theirs.Venue == "" {
		oursHasExtra = true
	}
	if theirs.Venue != "" && ours.Venue == "" {
		theirsHasExtra = true
	}

	if ours.PDFPath != "" && theirs.PDFPath == "" {
		oursHasExtra = true
	}
	if theirs.PDFPath != "" && ours.PDFPath == "" {
		theirsHasExtra = true
	}

	// Check authors
	if len(ours.Authors) > 0 && len(theirs.Authors) == 0 {
		oursHasExtra = true
	}
	if len(theirs.Authors) > 0 && len(ours.Authors) == 0 {
		theirsHasExtra = true
	}

	// Check publication date specificity
	oursDateScore := dateSpecificity(ours.Published)
	theirsDateScore := dateSpecificity(theirs.Published)
	if oursDateScore > theirsDateScore {
		oursHasExtra = true
	}
	if theirsDateScore > oursDateScore {
		theirsHasExtra = true
	}

	return oursHasExtra && theirsHasExtra
}

// MergeReferences merges two references, returning the merged result and any conflicts.
func MergeReferences(ours, theirs reference.Reference) (reference.Reference, []FieldConflict) {
	merged := reference.Reference{
		ID:  ours.ID,
		DOI: nonEmpty(ours.DOI, theirs.DOI),
	}
	var conflicts []FieldConflict

	// Merge string fields using helper to reduce duplication
	mergeField := func(fieldName, oursVal, theirsVal string, target *string) {
		val, conflict := mergeString(fieldName, oursVal, theirsVal)
		*target = val
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	mergeField("title", ours.Title, theirs.Title, &merged.Title)
	mergeField("abstract", ours.Abstract, theirs.Abstract, &merged.Abstract)
	mergeField("venue", ours.Venue, theirs.Venue, &merged.Venue)
	mergeField("pdf_path", ours.PDFPath, theirs.PDFPath, &merged.PDFPath)
	mergeField("supersedes", ours.Supersedes, theirs.Supersedes, &merged.Supersedes)

	// Authors - special handling
	authors, authorsConflict := mergeAuthors(ours.Authors, theirs.Authors)
	merged.Authors = authors
	if authorsConflict != nil {
		conflicts = append(conflicts, *authorsConflict)
	}

	// Published - take most specific
	merged.Published = mergePublicationDate(ours.Published, theirs.Published)

	// SupplementPaths - union
	merged.SupplementPaths = unionStrings(ours.SupplementPaths, theirs.SupplementPaths)

	// Source - keep original (from ours)
	merged.Source = ours.Source

	return merged, conflicts
}

// mergeString merges two string values, returning the merged value and any conflict.
func mergeString(fieldName, ours, theirs string) (string, *FieldConflict) {
	if ours == "" {
		return theirs, nil
	}
	if theirs == "" {
		return ours, nil
	}
	if ours == theirs {
		return ours, nil
	}
	// True conflict
	return "", &FieldConflict{
		FieldName:   fieldName,
		OursValue:   truncate(ours, 50),
		TheirsValue: truncate(theirs, 50),
	}
}

// mergeAuthors merges two author lists.
// Returns the longer list, or a conflict if same length with different content.
func mergeAuthors(ours, theirs []reference.Author) ([]reference.Author, *FieldConflict) {
	if len(ours) == 0 {
		return theirs, nil
	}
	if len(theirs) == 0 {
		return ours, nil
	}

	// Longer list wins
	if len(ours) > len(theirs) {
		return ours, nil
	}
	if len(theirs) > len(ours) {
		return theirs, nil
	}

	// Same length - check if content matches
	if authorsEqual(ours, theirs) {
		// Merge ORCIDs into combined list
		merged := mergeAuthorORCIDs(ours, theirs)
		return merged, nil
	}

	// Same length, different content = true conflict
	return nil, &FieldConflict{
		FieldName:   "authors",
		OursValue:   formatAuthorsShort(ours),
		TheirsValue: formatAuthorsShort(theirs),
	}
}

// authorsEqual checks if two author lists have the same names (case-insensitive).
func authorsEqual(a, b []reference.Author) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i].First, b[i].First) ||
			!strings.EqualFold(a[i].Last, b[i].Last) {
			return false
		}
	}
	return true
}

// mergeAuthorORCIDs merges ORCID information from two equal author lists.
func mergeAuthorORCIDs(ours, theirs []reference.Author) []reference.Author {
	merged := make([]reference.Author, len(ours))
	for i := range ours {
		merged[i] = ours[i]
		if merged[i].ORCID == "" && theirs[i].ORCID != "" {
			merged[i].ORCID = theirs[i].ORCID
		}
	}
	return merged
}

// formatAuthorsShort formats an author list for display.
func formatAuthorsShort(authors []reference.Author) string {
	if len(authors) == 0 {
		return ""
	}
	var names []string
	for _, a := range authors {
		names = append(names, a.Last)
	}
	result := strings.Join(names, ", ")
	if len(result) > 50 {
		return result[:47] + "..."
	}
	return result
}

// mergePublicationDate returns the more specific publication date.
func mergePublicationDate(ours, theirs reference.PublicationDate) reference.PublicationDate {
	oursScore := dateSpecificity(ours)
	theirsScore := dateSpecificity(theirs)

	if theirsScore > oursScore {
		return theirs
	}
	return ours
}

// dateSpecificity returns a score for how specific a date is.
func dateSpecificity(d reference.PublicationDate) int {
	score := 0
	if d.Year != 0 {
		score += 1
	}
	if d.Month != 0 {
		score += 1
	}
	if d.Day != 0 {
		score += 1
	}
	return score
}

// unionStrings returns the union of two string slices, preserving order.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// nonEmpty returns the first non-empty string.
func nonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// conflictFieldNames returns a comma-separated list of field names with conflicts.
func conflictFieldNames(conflicts []FieldConflict) string {
	var names []string
	for _, c := range conflicts {
		names = append(names, c.FieldName)
	}
	return strings.Join(names, ", ")
}

// ComputeCompleteness returns a completeness score for a reference.
// Higher scores indicate more complete metadata.
func ComputeCompleteness(ref reference.Reference) int {
	score := 0

	if ref.Abstract != "" {
		score += weightAbstract
	}
	if len(ref.Authors) > 0 {
		score += weightAuthors
	}
	if ref.Venue != "" {
		score += weightVenue
	}
	if ref.Published.Year != 0 {
		score += weightPublished
	}
	if ref.DOI != "" {
		score += weightDOI
	}

	return score
}

// ApplyResolution applies a resolution plan to produce the resolved reference.
func ApplyResolution(match PaperMatch, plan ResolutionPlan) reference.Reference {
	switch plan.Action {
	case ActionKeepOurs:
		return match.Ours
	case ActionKeepTheirs:
		return match.Theirs
	case ActionMerge:
		merged, _ := MergeReferences(match.Ours, match.Theirs)
		return merged
	default:
		// For conflicts, return ours as default (will be overridden by interactive)
		return match.Ours
	}
}
