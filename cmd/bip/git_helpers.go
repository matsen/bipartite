package main

import (
	"errors"

	"github.com/matsen/bipartite/internal/git"
	"github.com/matsen/bipartite/internal/reference"
)

// mustFindGitRepo finds the git repository root, exits on error.
func mustFindGitRepo(bipRoot string) string {
	gitRoot, err := git.FindRepoRoot(bipRoot)
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepo) {
			exitWithError(ExitError, "not in a git repository\n  Hint: Initialize with 'git init' or navigate to a git repository")
		}
		exitWithError(ExitError, "finding git repository: %v", err)
	}
	return gitRoot
}

// mustValidateCommit validates a commit reference, exits on error.
func mustValidateCommit(repoRoot, commitRef string) string {
	sha, err := git.ValidateCommit(repoRoot, commitRef)
	if err != nil {
		if errors.Is(err, git.ErrCommitNotFound) {
			exitWithError(ExitError, "commit not found: %s\n  Hint: Verify the commit exists with 'git log --oneline'", commitRef)
		}
		exitWithError(ExitError, "validating commit: %v", err)
	}
	return sha
}

// mustCheckGitTracking verifies refs.jsonl is tracked, exits on error.
func mustCheckGitTracking(repoRoot string) {
	if !git.IsFileTracked(repoRoot) {
		exitWithError(ExitError, "refs.jsonl not tracked by git\n  Hint: Run 'git add .bipartite/refs.jsonl' to track the file")
	}
}

// refToDiffPaper converts a reference to a DiffPaper for output.
func refToDiffPaper(ref reference.Reference) DiffPaper {
	return DiffPaper{
		ID:      ref.ID,
		Title:   ref.Title,
		Authors: formatAuthorsLastNames(ref.Authors),
		Year:    ref.Published.Year,
	}
}

// refToNewPaper converts a reference and commit info to a NewPaper for output.
func refToNewPaper(ref reference.Reference, commitSHA string) NewPaper {
	return NewPaper{
		ID:        ref.ID,
		Title:     ref.Title,
		Authors:   formatAuthorsLastNames(ref.Authors),
		Year:      ref.Published.Year,
		CommitSHA: commitSHA,
	}
}

// formatAuthorsLastNames formats authors as "Last1, Last2, ..."
func formatAuthorsLastNames(authors []reference.Author) string {
	if len(authors) == 0 {
		return ""
	}
	names := make([]string, 0, len(authors))
	for _, a := range authors {
		names = append(names, a.Last)
	}
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}
