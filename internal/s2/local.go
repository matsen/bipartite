// Package s2 local.go provides local reference resolution for S2 commands.
// It enables looking up papers in the local collection by various identifiers
// (local ID, DOI, S2 paper ID) and resolving them to Semantic Scholar API IDs.
package s2

import (
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// LocalResolver resolves local paper IDs to S2 identifiers and provides
// lookup capabilities for the local reference collection. It maintains
// indexes by ID, DOI, and S2 paper ID for efficient lookups.
type LocalResolver struct {
	refs   []reference.Reference
	byID   map[string]*reference.Reference
	byDOI  map[string]*reference.Reference
	byS2ID map[string]*reference.Reference
}

// NewLocalResolver creates a LocalResolver by loading references from a refs.jsonl file.
// Returns an error if the file cannot be read or parsed.
func NewLocalResolver(refsPath string) (*LocalResolver, error) {
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return nil, err
	}
	return NewLocalResolverFromRefs(refs), nil
}

// NewLocalResolverFromRefs creates a LocalResolver from a pre-loaded slice of references.
// This is useful when references are already in memory.
func NewLocalResolverFromRefs(refs []reference.Reference) *LocalResolver {
	r := &LocalResolver{
		refs:   refs,
		byID:   make(map[string]*reference.Reference),
		byDOI:  make(map[string]*reference.Reference),
		byS2ID: make(map[string]*reference.Reference),
	}

	for i := range refs {
		ref := &refs[i]
		r.byID[ref.ID] = ref
		if ref.DOI != "" {
			r.byDOI[NormalizeDOI(ref.DOI)] = ref
		}
		// Index by S2 ID if source is s2
		if ref.Source.Type == "s2" && ref.Source.ID != "" {
			r.byS2ID[ref.Source.ID] = ref
		}
	}

	return r
}

// ResolveToS2ID resolves a paper identifier to an S2 API-compatible format.
// If the input is already an external ID (DOI:, ARXIV:, etc.), it returns as-is.
// For local IDs, it looks up the reference and returns its DOI or S2 ID.
// Returns the S2 API identifier string, the matched reference (if local), and any error.
func (r *LocalResolver) ResolveToS2ID(id string) (string, *reference.Reference, error) {
	parsed := ParsePaperID(id)

	// If it's already an external ID, return as-is
	if parsed.IsExternalID() {
		return parsed.String(), nil, nil
	}

	// Look up local ID
	ref, found := r.byID[id]
	if !found {
		return "", nil, ErrNotFound
	}

	// Prefer DOI, then S2 ID
	if ref.DOI != "" {
		return "DOI:" + ref.DOI, ref, nil
	}
	if ref.Source.Type == "s2" && ref.Source.ID != "" {
		return ref.Source.ID, ref, nil
	}

	return "", nil, ErrNotFound
}

// FindByDOI finds a local reference by DOI. The DOI is normalized before lookup.
func (r *LocalResolver) FindByDOI(doi string) (*reference.Reference, bool) {
	ref, ok := r.byDOI[NormalizeDOI(doi)]
	return ref, ok
}

// FindByID finds a local reference by ID.
func (r *LocalResolver) FindByID(id string) (*reference.Reference, bool) {
	ref, ok := r.byID[id]
	return ref, ok
}

// FindByS2ID finds a local reference by S2 paper ID.
func (r *LocalResolver) FindByS2ID(s2ID string) (*reference.Reference, bool) {
	ref, ok := r.byS2ID[s2ID]
	return ref, ok
}

// ExistsLocally checks if an S2Paper exists in the local collection.
// It checks by DOI first, then by S2 paper ID. Returns the matching
// reference and true if found, nil and false otherwise.
func (r *LocalResolver) ExistsLocally(paper S2Paper) (*reference.Reference, bool) {
	// Check by DOI
	if paper.ExternalIDs.DOI != "" {
		if ref, ok := r.FindByDOI(paper.ExternalIDs.DOI); ok {
			return ref, true
		}
	}

	// Check by S2 ID
	if paper.PaperID != "" {
		if ref, ok := r.FindByS2ID(paper.PaperID); ok {
			return ref, true
		}
	}

	return nil, false
}

// AllRefs returns all references.
func (r *LocalResolver) AllRefs() []reference.Reference {
	return r.refs
}

// Count returns the number of references.
func (r *LocalResolver) Count() int {
	return len(r.refs)
}

// RefsWithDOI returns all references that have a non-empty DOI field.
// This is useful for operations that require DOI-based S2 API lookups.
func (r *LocalResolver) RefsWithDOI() []reference.Reference {
	var result []reference.Reference
	for _, ref := range r.refs {
		if ref.DOI != "" {
			result = append(result, ref)
		}
	}
	return result
}
