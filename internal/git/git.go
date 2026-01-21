package git

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// ErrNotGitRepo indicates the directory is not a git repository.
var ErrNotGitRepo = errors.New("not a git repository")

// ErrCommitNotFound indicates the specified commit does not exist.
var ErrCommitNotFound = errors.New("commit not found")

// ErrFileNotTracked indicates refs.jsonl is not tracked by git.
var ErrFileNotTracked = errors.New("refs.jsonl not tracked by git")

// FindRepoRoot finds the root of the git repository containing the given path.
// Returns ErrNotGitRepo if not in a git repository.
func FindRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", ErrNotGitRepo
	}
	return strings.TrimSpace(string(output)), nil
}

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(path string) bool {
	_, err := FindRepoRoot(path)
	return err == nil
}

// ValidateCommit verifies that a commit reference exists.
// Supports SHA, HEAD, HEAD~N, branch names, tags, etc.
// Returns the resolved full SHA or ErrCommitNotFound.
func ValidateCommit(repoRoot, commitRef string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "--verify", commitRef+"^{commit}")
	output, err := cmd.Output()
	if err != nil {
		return "", ErrCommitNotFound
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRefsJSONLPath returns the path to refs.jsonl relative to repo root.
func GetRefsJSONLPath() string {
	return ".bipartite/refs.jsonl"
}

// GetRefsJSONLAtCommit retrieves the contents of refs.jsonl at a specific commit.
// Returns ErrCommitNotFound if commit doesn't exist, or empty slice if file didn't exist at that commit.
func GetRefsJSONLAtCommit(repoRoot, commitRef string) ([]reference.Reference, error) {
	// First validate the commit exists
	sha, err := ValidateCommit(repoRoot, commitRef)
	if err != nil {
		return nil, err
	}

	// Get file contents at that commit
	refsPath := GetRefsJSONLPath()
	cmd := exec.Command("git", "-C", repoRoot, "show", sha+":"+refsPath)
	output, err := cmd.Output()
	if err != nil {
		// File might not exist at that commit - return empty slice
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return []reference.Reference{}, nil
		}
		return nil, fmt.Errorf("getting refs.jsonl at %s: %w", commitRef, err)
	}

	return parseRefsJSONL(output)
}

// GetCurrentRefs reads the current refs.jsonl from the working tree.
func GetCurrentRefs(repoRoot string) ([]reference.Reference, error) {
	refsPath := filepath.Join(repoRoot, GetRefsJSONLPath())
	data, err := os.ReadFile(refsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []reference.Reference{}, nil
		}
		return nil, fmt.Errorf("reading refs.jsonl: %w", err)
	}

	return parseRefsJSONL(data)
}

// parseRefsJSONL parses JSONL content into a slice of references.
func parseRefsJSONL(data []byte) ([]reference.Reference, error) {
	var refs []reference.Reference
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ref reference.Reference
		if err := json.Unmarshal([]byte(line), &ref); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		refs = append(refs, ref)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning refs: %w", err)
	}
	return refs, nil
}

// IsFileTracked checks if refs.jsonl is tracked by git.
func IsFileTracked(repoRoot string) bool {
	refsPath := GetRefsJSONLPath()
	cmd := exec.Command("git", "-C", repoRoot, "ls-files", refsPath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}
