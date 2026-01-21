package git

import (
	"github.com/matsen/bipartite/internal/reference"
)

// DiffWorkingTree compares the working tree refs.jsonl to HEAD.
// Returns papers added and removed since the last commit.
func DiffWorkingTree(repoRoot string) (*GitDiff, error) {
	// Get refs at HEAD
	headRefs, err := GetRefsJSONLAtCommit(repoRoot, "HEAD")
	if err != nil {
		return nil, err
	}

	// Get current working tree refs
	currentRefs, err := GetCurrentRefs(repoRoot)
	if err != nil {
		return nil, err
	}

	return diffRefs(headRefs, currentRefs), nil
}

// DiffSince compares current refs.jsonl to a specific commit.
// Returns papers added and removed since that commit.
func DiffSince(repoRoot, commitRef string) (*GitDiff, error) {
	// Get refs at the specified commit
	oldRefs, err := GetRefsJSONLAtCommit(repoRoot, commitRef)
	if err != nil {
		return nil, err
	}

	// Get current working tree refs
	currentRefs, err := GetCurrentRefs(repoRoot)
	if err != nil {
		return nil, err
	}

	return diffRefs(oldRefs, currentRefs), nil
}

// diffRefs computes the difference between two sets of references.
// Returns papers in current but not old (added), and papers in old but not current (removed).
func diffRefs(oldRefs, currentRefs []reference.Reference) *GitDiff {
	// Build maps keyed by ID for efficient lookup
	oldMap := make(map[string]reference.Reference, len(oldRefs))
	for _, ref := range oldRefs {
		oldMap[ref.ID] = ref
	}

	currentMap := make(map[string]reference.Reference, len(currentRefs))
	for _, ref := range currentRefs {
		currentMap[ref.ID] = ref
	}

	// Find added papers (in current but not old)
	var added []reference.Reference
	for id, ref := range currentMap {
		if _, exists := oldMap[id]; !exists {
			added = append(added, ref)
		}
	}

	// Find removed papers (in old but not current)
	var removed []reference.Reference
	for id, ref := range oldMap {
		if _, exists := currentMap[id]; !exists {
			removed = append(removed, ref)
		}
	}

	return &GitDiff{
		Added:   added,
		Removed: removed,
	}
}
