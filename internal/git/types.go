// Package git provides git integration for tracking repository history.
package git

import "github.com/matsen/bipartite/internal/reference"

// GitDiff represents changes to refs.jsonl between two git states.
type GitDiff struct {
	Added   []reference.Reference
	Removed []reference.Reference
}

// CommitInfo represents information about a git commit.
type CommitInfo struct {
	SHA     string
	Message string
}
