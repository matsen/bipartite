package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a fresh git repo at dir with one commit on a branch
// called "main" and returns the directory. We pin user.name/email and
// init.defaultBranch via local config so the test does not pick up the
// developer's global git settings.
func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}
	dir := t.TempDir()
	mustGit(t, dir, "init", "--quiet", "--initial-branch=main")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test User")
	mustGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README.md")
	mustGit(t, dir, "commit", "--quiet", "-m", "initial")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestIsInWorktree_Primary(t *testing.T) {
	primary := initRepo(t)
	got, err := IsInWorktree(primary)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Errorf("IsInWorktree(primary) = true, want false")
	}
}

func TestIsInWorktree_Linked(t *testing.T) {
	primary := initRepo(t)
	// Place the linked worktree as a sibling of the primary so the worktree
	// path is never a subdirectory of the primary's working tree.
	wt := filepath.Join(filepath.Dir(primary), "wt-feature")
	if err := AddWorktree(primary, wt, "feature"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	got, err := IsInWorktree(wt)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Errorf("IsInWorktree(linked) = false, want true")
	}
}

func TestPrimaryCloneDir(t *testing.T) {
	primary := initRepo(t)
	wt := filepath.Join(filepath.Dir(primary), "wt-feature")
	if err := AddWorktree(primary, wt, "feature"); err != nil {
		t.Fatal(err)
	}

	// Resolve from inside the worktree — must walk back to the primary.
	got, err := PrimaryCloneDir(wt)
	if err != nil {
		t.Fatal(err)
	}
	wantAbs, _ := filepath.EvalSymlinks(primary)
	gotAbs, _ := filepath.EvalSymlinks(got)
	if gotAbs != wantAbs {
		t.Errorf("PrimaryCloneDir(linked) = %q, want %q", gotAbs, wantAbs)
	}

	// Resolve from inside the primary itself — must return the primary.
	got, err = PrimaryCloneDir(primary)
	if err != nil {
		t.Fatal(err)
	}
	gotAbs, _ = filepath.EvalSymlinks(got)
	if gotAbs != wantAbs {
		t.Errorf("PrimaryCloneDir(primary) = %q, want %q", gotAbs, wantAbs)
	}
}

func TestAddAndRemoveWorktree(t *testing.T) {
	primary := initRepo(t)
	wt := filepath.Join(filepath.Dir(primary), "wt-149")
	if err := AddWorktree(primary, wt, "149-test"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("worktree dir not created: %v", err)
	}
	exists, err := WorktreeExists(primary, wt)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Errorf("WorktreeExists after add = false, want true")
	}
	if err := RemoveWorktree(primary, wt, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree dir still exists after remove: err=%v", err)
	}
	exists, err = WorktreeExists(primary, wt)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Errorf("WorktreeExists after remove = true, want false")
	}
}

func TestRemoveWorktree_ForceOnDirty(t *testing.T) {
	// A worktree with uncommitted changes can only be removed with --force.
	// This is the post-squash-merge state pr-land lands in.
	primary := initRepo(t)
	wt := filepath.Join(filepath.Dir(primary), "wt-dirty")
	if err := AddWorktree(primary, wt, "dirty-branch"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "uncommitted.txt"), []byte("dirt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(primary, wt, false); err == nil {
		t.Fatal("expected non-force removal of dirty worktree to fail")
	}
	if err := RemoveWorktree(primary, wt, true); err != nil {
		t.Fatalf("forced removal of dirty worktree failed: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree dir survived forced removal: err=%v", err)
	}
}
