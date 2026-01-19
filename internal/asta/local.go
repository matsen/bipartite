package asta

import (
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// LocalResolver resolves local paper IDs to S2 identifiers.
type LocalResolver struct {
	refs   []reference.Reference
	byID   map[string]*reference.Reference
	byDOI  map[string]*reference.Reference
	byS2ID map[string]*reference.Reference
}

// NewLocalResolver creates a resolver from a refs.jsonl path.
func NewLocalResolver(refsPath string) (*LocalResolver, error) {
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return nil, err
	}
	return NewLocalResolverFromRefs(refs), nil
}

// NewLocalResolverFromRefs creates a resolver from a slice of references.
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
		// Index by S2 ID if source is asta
		if ref.Source.Type == "asta" && ref.Source.ID != "" {
			r.byS2ID[ref.Source.ID] = ref
		}
	}

	return r
}

// ResolveToS2ID resolves a local ID to an S2-compatible identifier.
// Returns the paper identifier string to use with the S2 API.
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
	if ref.Source.Type == "asta" && ref.Source.ID != "" {
		return ref.Source.ID, ref, nil
	}

	return "", nil, ErrNotFound
}

// FindByDOI finds a local reference by DOI.
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

// ExistsLocally checks if a paper exists in the local collection.
// Checks by DOI, S2 ID, and local ID.
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

// RefsWithDOI returns references that have a DOI.
func (r *LocalResolver) RefsWithDOI() []reference.Reference {
	var result []reference.Reference
	for _, ref := range r.refs {
		if ref.DOI != "" {
			result = append(result, ref)
		}
	}
	return result
}
