package git

import (
	"bufio"
	"bytes"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/reference"
)

// RecentPaper represents a recently added paper with its commit info.
type RecentPaper struct {
	Reference reference.Reference
	CommitSHA string
	CommitMsg string
}

// GetRecentlyAddedPapers returns the N most recently added papers.
// "Recently added" is determined by git commit history on refs.jsonl.
func GetRecentlyAddedPapers(repoRoot string, n int) ([]RecentPaper, error) {
	// Get commits that touched refs.jsonl
	commits, err := getCommitsTouchingRefs(repoRoot)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	// Track papers we've seen and their commit info
	seen := make(map[string]RecentPaper)
	var result []RecentPaper

	// Walk through commits from newest to oldest
	for i := 0; i < len(commits); i++ {
		commit := commits[i]

		// Get refs at this commit
		refsAtCommit, err := GetRefsJSONLAtCommit(repoRoot, commit.SHA)
		if err != nil {
			continue
		}

		// Get refs at parent commit (if any)
		var refsAtParent []reference.Reference
		if i+1 < len(commits) {
			refsAtParent, _ = GetRefsJSONLAtCommit(repoRoot, commits[i+1].SHA)
		}

		// Build parent map for efficient lookup
		parentMap := make(map[string]bool, len(refsAtParent))
		for _, ref := range refsAtParent {
			parentMap[ref.ID] = true
		}

		// Find papers added in this commit
		for _, ref := range refsAtCommit {
			if _, wasSeen := seen[ref.ID]; !wasSeen {
				if !parentMap[ref.ID] {
					// This paper was added in this commit
					rp := RecentPaper{
						Reference: ref,
						CommitSHA: shortSHA(commit.SHA),
						CommitMsg: commit.Message,
					}
					seen[ref.ID] = rp
					result = append(result, rp)
					if len(result) >= n {
						return result, nil
					}
				}
			}
		}
	}

	return result, nil
}

// GetPapersAddedSince returns papers added since a specific commit.
func GetPapersAddedSince(repoRoot, commitRef string) ([]RecentPaper, error) {
	// Validate commit
	_, err := ValidateCommit(repoRoot, commitRef)
	if err != nil {
		return nil, err
	}

	// Get commits between commitRef and HEAD that touched refs.jsonl
	commits, err := getCommitsSince(repoRoot, commitRef)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	// Get refs at the starting commit
	refsAtStart, err := GetRefsJSONLAtCommit(repoRoot, commitRef)
	if err != nil {
		return nil, err
	}
	startMap := make(map[string]bool, len(refsAtStart))
	for _, ref := range refsAtStart {
		startMap[ref.ID] = true
	}

	// Get current refs
	currentRefs, err := GetCurrentRefs(repoRoot)
	if err != nil {
		return nil, err
	}

	// Find papers added since the starting commit
	var result []RecentPaper
	seen := make(map[string]bool)

	for _, ref := range currentRefs {
		if !startMap[ref.ID] && !seen[ref.ID] {
			seen[ref.ID] = true
			// Find which commit added this paper
			commitSHA := findCommitThatAdded(repoRoot, ref.ID, commits)
			result = append(result, RecentPaper{
				Reference: ref,
				CommitSHA: commitSHA,
			})
		}
	}

	return result, nil
}

// GetPapersAddedInDays returns papers that exist in the current refs
// and were added to the repository within the last N days.
func GetPapersAddedInDays(repoRoot string, days int) ([]RecentPaper, error) {
	// Calculate the cutoff time
	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	// Get commits since cutoff that touched refs.jsonl
	commits, err := getCommitsSinceTime(repoRoot, cutoff)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	// Get refs at the earliest commit in our range (or empty if none before)
	var refsAtStart []reference.Reference
	// Find earliest commit before our range
	earliestCommit, err := getEarliestCommitBefore(repoRoot, cutoff)
	if err == nil && earliestCommit != "" {
		refsAtStart, _ = GetRefsJSONLAtCommit(repoRoot, earliestCommit)
	}

	startMap := make(map[string]bool, len(refsAtStart))
	for _, ref := range refsAtStart {
		startMap[ref.ID] = true
	}

	// Get current refs
	currentRefs, err := GetCurrentRefs(repoRoot)
	if err != nil {
		return nil, err
	}

	// Find papers added since the cutoff
	var result []RecentPaper
	seen := make(map[string]bool)

	for _, ref := range currentRefs {
		if !startMap[ref.ID] && !seen[ref.ID] {
			seen[ref.ID] = true
			commitSHA := findCommitThatAdded(repoRoot, ref.ID, commits)
			result = append(result, RecentPaper{
				Reference: ref,
				CommitSHA: commitSHA,
			})
		}
	}

	return result, nil
}

// getCommitsTouchingRefs returns commits that touched refs.jsonl, newest first.
func getCommitsTouchingRefs(repoRoot string) ([]CommitInfo, error) {
	refsPath := GetRefsJSONLPath()
	cmd := exec.Command("git", "-C", repoRoot, "log", "--oneline", "--follow", "--", refsPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // No commits or file not tracked
	}

	return parseGitLogOneline(output), nil
}

// getCommitsSince returns commits between commitRef and HEAD that touched refs.jsonl.
func getCommitsSince(repoRoot, commitRef string) ([]CommitInfo, error) {
	refsPath := GetRefsJSONLPath()
	cmd := exec.Command("git", "-C", repoRoot, "log", "--oneline", commitRef+"..HEAD", "--", refsPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return parseGitLogOneline(output), nil
}

// getCommitsSinceTime returns commits since a specific time that touched refs.jsonl.
func getCommitsSinceTime(repoRoot string, since time.Time) ([]CommitInfo, error) {
	refsPath := GetRefsJSONLPath()
	sinceStr := since.Format("2006-01-02")
	cmd := exec.Command("git", "-C", repoRoot, "log", "--oneline", "--since="+sinceStr, "--", refsPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return parseGitLogOneline(output), nil
}

// getEarliestCommitBefore returns the earliest commit touching refs.jsonl before the given time.
func getEarliestCommitBefore(repoRoot string, before time.Time) (string, error) {
	refsPath := GetRefsJSONLPath()
	beforeStr := before.Format("2006-01-02")
	cmd := exec.Command("git", "-C", repoRoot, "log", "--oneline", "--until="+beforeStr, "-1", "--", refsPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	commits := parseGitLogOneline(output)
	if len(commits) == 0 {
		return "", nil
	}
	return commits[0].SHA, nil
}

// findCommitThatAdded finds the commit that added a specific paper ID.
func findCommitThatAdded(repoRoot, paperID string, commits []CommitInfo) string {
	// Walk commits from newest to oldest
	for i := 0; i < len(commits); i++ {
		commit := commits[i]
		refsAtCommit, err := GetRefsJSONLAtCommit(repoRoot, commit.SHA)
		if err != nil {
			continue
		}

		// Check if paper exists at this commit
		hasAtCommit := false
		for _, ref := range refsAtCommit {
			if ref.ID == paperID {
				hasAtCommit = true
				break
			}
		}

		if !hasAtCommit {
			// Paper doesn't exist at this commit, so it was added after
			if i > 0 {
				return shortSHA(commits[i-1].SHA)
			}
			return ""
		}
	}

	// Paper existed at earliest commit in range
	if len(commits) > 0 {
		return shortSHA(commits[len(commits)-1].SHA)
	}
	return ""
}

// shortSHA returns a short version of a SHA (up to 8 chars).
func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// parseGitLogOneline parses git log --oneline output.
func parseGitLogOneline(data []byte) []CommitInfo {
	var commits []CommitInfo
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}
		ci := CommitInfo{SHA: parts[0]}
		if len(parts) > 1 {
			ci.Message = parts[1]
		}
		commits = append(commits, ci)
	}
	return commits
}

// SortRefsAlphabetically sorts references by ID for deterministic output.
func SortRefsAlphabetically(refs []reference.Reference) {
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ID < refs[j].ID
	})
}
