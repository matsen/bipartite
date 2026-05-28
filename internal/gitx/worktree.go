// Package gitx provides small, tested wrappers over the git porcelain
// commands bip needs for its worktree integration. The point of this
// package is not to reimplement git — it is to keep the dangerous paths
// (worktree creation, --force removal, primary-clone resolution) in tested
// Go so they are not duplicated as untested shell pipelines inside skill
// markdown files.
package gitx

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsInWorktree returns true if dir sits inside a linked worktree (one that
// was added via `git worktree add`). It returns false if dir is the primary
// working tree. An error here means dir is not a git repository at all, or
// git itself failed.
func IsInWorktree(dir string) (bool, error) {
	common, err := runGit(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return false, err
	}
	gitDir, err := runGit(dir, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return false, err
	}
	// In a primary working tree, --git-dir and --git-common-dir point at
	// the same .git directory. In a linked worktree, --git-dir is the
	// per-worktree .git/worktrees/<name> directory, while --git-common-dir
	// is still the primary's .git.
	return filepath.Clean(common) != filepath.Clean(gitDir), nil
}

// PrimaryCloneDir returns the canonical clone directory (the parent of the
// primary .git directory) for dir, which may itself be either the primary
// or a linked worktree.
func PrimaryCloneDir(dir string) (string, error) {
	common, err := runGit(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	// `--git-common-dir` resolves to "<primary>/.git" (or to the bare repo
	// itself for a bare clone). For non-bare clones the primary working
	// tree is its parent.
	return filepath.Dir(filepath.Clean(common)), nil
}

// AddWorktree runs `git worktree add` from primaryClone, creating a new
// worktree at path on a new branch. An empty branch checks out HEAD's
// current branch (uncommon in our usage; we always pass a branch name).
func AddWorktree(primaryClone, path, branch string) error {
	args := []string{"worktree", "add"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, path)
	out, err := runGitCombined(primaryClone, args...)
	if err != nil {
		return fmt.Errorf("git worktree add: %w\n%s", err, out)
	}
	return nil
}

// RemoveWorktree runs `git worktree remove` from primaryClone. force=true
// passes --force, required after a squash merge leaves the worktree with
// commits unreachable from the merged branch.
func RemoveWorktree(primaryClone, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	out, err := runGitCombined(primaryClone, args...)
	if err != nil {
		return fmt.Errorf("git worktree remove: %w\n%s", err, out)
	}
	return nil
}

// WorktreeExists returns true if path is currently registered as a worktree
// of primaryClone. Useful for the "spawn the same issue twice" reuse case.
//
// Comparison is symlink-resolving: macOS reports t.TempDir() paths as
// /var/... while git emits the realpath /private/var/..., so a raw
// filepath.Clean would miss the match.
func WorktreeExists(primaryClone, path string) (bool, error) {
	out, err := runGit(primaryClone, "worktree", "list", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git worktree list: %w", err)
	}
	target := realpath(path)
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		wt := strings.TrimPrefix(line, "worktree ")
		if realpath(wt) == target {
			return true, nil
		}
	}
	return false, nil
}

// realpath resolves symlinks and makes p absolute. On any error it falls
// back to filepath.Clean(p) — callers want a best-effort canonical form,
// not a hard failure.
func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(resolved)
}

// runGit invokes git in dir and returns trimmed stdout, with stderr folded
// into the returned error on failure.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, stderr)
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitCombined returns combined stdout+stderr as a single string, useful
// when reporting the failure context for write operations.
func runGitCombined(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
