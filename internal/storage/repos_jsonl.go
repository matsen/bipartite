package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matsen/bipartite/internal/repo"
)

// ReadAllRepos reads all repos from a JSONL file.
// Returns an error if any repo fails structural validation (fail-fast).
func ReadAllRepos(path string) ([]repo.Repo, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening repos file: %w", err)
	}
	defer f.Close()

	var repos []repo.Repo
	scanner := bufio.NewScanner(f)

	// Increase buffer size for long lines
	buf := make([]byte, MaxJSONLLineCapacity)
	scanner.Buffer(buf, MaxJSONLLineCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var r repo.Repo
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}

		// Fail fast: validate repo structure before adding to collection
		if err := r.ValidateForCreate(); err != nil {
			return nil, fmt.Errorf("invalid repo at line %d: %w", lineNum, err)
		}

		repos = append(repos, r)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading repos file: %w", err)
	}

	return repos, nil
}

// writeRepoJSONL marshals a repo to JSON and writes it as a JSONL line.
func writeRepoJSONL(w io.Writer, r repo.Repo) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encoding repo: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing repo: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	return nil
}

// AppendRepo adds a repo to the end of a JSONL file.
func AppendRepo(path string, r repo.Repo) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening repos file for append: %w", err)
	}
	defer f.Close()

	return writeRepoJSONL(f, r)
}

// WriteAllRepos writes all repos to a JSONL file, replacing existing content.
func WriteAllRepos(path string, repos []repo.Repo) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating repos file: %w", err)
	}
	defer f.Close()

	for _, r := range repos {
		if err := writeRepoJSONL(f, r); err != nil {
			return err
		}
	}

	return nil
}

// FindRepoByID searches for a repo by its ID in an in-memory slice.
// Returns the index and true if found, -1 and false otherwise.
func FindRepoByID(repos []repo.Repo, id string) (int, bool) {
	for i, r := range repos {
		if r.ID == id {
			return i, true
		}
	}
	return -1, false
}

// FindRepoByGitHubURL searches for a repo by its GitHub URL in an in-memory slice.
// Returns the index and true if found, -1 and false otherwise.
func FindRepoByGitHubURL(repos []repo.Repo, url string) (int, bool) {
	if url == "" {
		return -1, false
	}
	for i, r := range repos {
		if r.GitHubURL == url {
			return i, true
		}
	}
	return -1, false
}

// UpsertRepoInSlice adds or updates a repo in an in-memory slice.
// Returns the updated slice and true if the repo was updated, false if added.
func UpsertRepoInSlice(repos []repo.Repo, newRepo repo.Repo) ([]repo.Repo, bool) {
	idx, found := FindRepoByID(repos, newRepo.ID)
	if found {
		repos[idx] = newRepo
		return repos, true
	}
	return append(repos, newRepo), false
}

// DeleteRepoFromSlice removes a repo from an in-memory slice.
// Returns the updated slice and true if the repo was found and removed, false otherwise.
// Note: This operation does not preserve slice order; it swaps the deleted element with
// the last element for O(1) performance. If ordering matters, callers should sort after deletion.
func DeleteRepoFromSlice(repos []repo.Repo, id string) ([]repo.Repo, bool) {
	idx, found := FindRepoByID(repos, id)
	if !found {
		return repos, false
	}
	// Remove by replacing with last element and truncating (O(1) but changes order)
	repos[idx] = repos[len(repos)-1]
	return repos[:len(repos)-1], true
}

// LoadRepoIDSet loads all repo IDs and returns them as a set for O(1) lookup.
func LoadRepoIDSet(path string) (map[string]bool, error) {
	repos, err := ReadAllRepos(path)
	if err != nil {
		return nil, err
	}

	idSet := make(map[string]bool, len(repos))
	for _, r := range repos {
		idSet[r.ID] = true
	}
	return idSet, nil
}

// GetReposByProject filters repos by project ID.
func GetReposByProject(repos []repo.Repo, projectID string) []repo.Repo {
	var filtered []repo.Repo
	for _, r := range repos {
		if r.Project == projectID {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
