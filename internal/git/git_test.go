package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseRefsJSONL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		data := `{"id":"Zhang2018-vi","title":"Variational Inference"}
{"id":"Doe2020-nn","title":"Neural Networks"}
`
		refs, err := parseRefsJSONL([]byte(data))
		if err != nil {
			t.Fatalf("parseRefsJSONL: %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("len = %d, want 2", len(refs))
		}
		if refs[0].ID != "Zhang2018-vi" || refs[1].ID != "Doe2020-nn" {
			t.Errorf("ids = %q, %q", refs[0].ID, refs[1].ID)
		}
	})

	t.Run("blank lines tolerated", func(t *testing.T) {
		data := "\n{\"id\":\"a\"}\n\n   \n{\"id\":\"b\"}\n\n"
		refs, err := parseRefsJSONL([]byte(data))
		if err != nil {
			t.Fatalf("parseRefsJSONL: %v", err)
		}
		if len(refs) != 2 {
			t.Fatalf("len = %d, want 2", len(refs))
		}
	})

	t.Run("malformed line errors", func(t *testing.T) {
		data := "{\"id\":\"a\"}\nnot json\n"
		_, err := parseRefsJSONL([]byte(data))
		if err == nil {
			t.Fatal("expected error for malformed line, got nil")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		refs, err := parseRefsJSONL([]byte(""))
		if err != nil {
			t.Fatalf("parseRefsJSONL: %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("len = %d, want 0", len(refs))
		}
	})
}

// newTestRepo creates a throwaway git repo with one commit that adds
// .bipartite/refs.jsonl. It skips the test if git is not available.
// Returns the repo root and the full SHA of the commit.
func newTestRepo(t *testing.T) (root, sha string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	root = t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return string(out)
	}

	run("init", "-q")
	refsPath := filepath.Join(root, GetRefsJSONLPath())
	if err := os.MkdirAll(filepath.Dir(refsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refsPath, []byte(`{"id":"Zhang2018-vi","title":"Variational Inference"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", GetRefsJSONLPath())
	run("commit", "-q", "-m", "Add first paper")

	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	sha = trimNL(string(out))
	return root, sha
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestFindRepoRootAndIsGitRepo(t *testing.T) {
	root, _ := newTestRepo(t)

	got, err := FindRepoRoot(root)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	// Resolve symlinks since macOS TempDir lives under /var -> /private/var.
	wantResolved, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("FindRepoRoot = %q, want %q", gotResolved, wantResolved)
	}

	if !IsGitRepo(root) {
		t.Error("IsGitRepo = false, want true")
	}

	// A bare temp dir (no git) should not be a repo.
	notRepo := t.TempDir()
	if IsGitRepo(notRepo) {
		t.Error("IsGitRepo on non-repo = true, want false")
	}
	if _, err := FindRepoRoot(notRepo); err != ErrNotGitRepo {
		t.Errorf("FindRepoRoot on non-repo err = %v, want ErrNotGitRepo", err)
	}
}

func TestIsFileTracked(t *testing.T) {
	root, _ := newTestRepo(t)
	if !IsFileTracked(root) {
		t.Error("IsFileTracked = false, want true (refs.jsonl committed)")
	}

	// Fresh repo with no commits: file not tracked.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	empty := t.TempDir()
	if err := exec.Command("git", "-C", empty, "init", "-q").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if IsFileTracked(empty) {
		t.Error("IsFileTracked on empty repo = true, want false")
	}
}

func TestValidateCommit(t *testing.T) {
	root, sha := newTestRepo(t)

	resolved, err := ValidateCommit(root, "HEAD")
	if err != nil {
		t.Fatalf("ValidateCommit(HEAD): %v", err)
	}
	if resolved != sha {
		t.Errorf("ValidateCommit(HEAD) = %q, want %q", resolved, sha)
	}

	if _, err := ValidateCommit(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); err != ErrCommitNotFound {
		t.Errorf("ValidateCommit(bogus) err = %v, want ErrCommitNotFound", err)
	}
}

func TestGetRefsJSONLAtCommit(t *testing.T) {
	root, sha := newTestRepo(t)
	refs, err := GetRefsJSONLAtCommit(root, sha)
	if err != nil {
		t.Fatalf("GetRefsJSONLAtCommit: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != "Zhang2018-vi" {
		t.Errorf("refs = %+v, want one Zhang2018-vi", refs)
	}
}

func TestFindCommitThatAdded(t *testing.T) {
	root, sha := newTestRepo(t)
	commits := []CommitInfo{{SHA: sha}}

	// Paper present at the only commit -> returns shortSHA of earliest commit.
	got := findCommitThatAdded(root, "Zhang2018-vi", commits)
	if got != shortSHA(sha) {
		t.Errorf("findCommitThatAdded(present) = %q, want %q", got, shortSHA(sha))
	}

	// Unknown paper never present -> empty (no commit lacked it before its add).
	if got := findCommitThatAdded(root, "Nobody1999-xx", commits); got != "" {
		t.Errorf("findCommitThatAdded(absent) = %q, want empty", got)
	}
}
