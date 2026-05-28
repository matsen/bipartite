package flow

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/config"
)

// setupResolverFixture writes a minimal nexus dir + global config dir and
// returns the nexus path. The global config is written iff globalYAML != "";
// XDG_CONFIG_HOME is pointed at the temp config dir so LoadGlobalConfig sees
// it. The cache is reset on entry and on cleanup so back-to-back tests don't
// see each other's config.
func setupResolverFixture(t *testing.T, sourcesYAML, flowConfigYAML, globalYAML string) string {
	t.Helper()
	root := t.TempDir()
	nexus := filepath.Join(root, "nexus")
	if err := os.MkdirAll(nexus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nexus, "sources.yml"), []byte(sourcesYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if flowConfigYAML != "" {
		if err := os.WriteFile(filepath.Join(nexus, "config.yml"), []byte(flowConfigYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfgHome := filepath.Join(root, "xdg")
	if err := os.MkdirAll(filepath.Join(cfgHome, "bip"), 0o755); err != nil {
		t.Fatal(err)
	}
	if globalYAML != "" {
		if err := os.WriteFile(filepath.Join(cfgHome, "bip", "config.yml"), []byte(globalYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	config.ResetGlobalConfigCache()
	t.Cleanup(config.ResetGlobalConfigCache)
	return nexus
}

func TestResolveRepoPath_AbsentLayoutMatchesGetRepoLocalPath(t *testing.T) {
	// The headline backward-compat guarantee: with no `layout:` block
	// anywhere, ResolveRepoPath returns byte-identical paths to the
	// pre-issue-149 GetRepoLocalPath for both code and writing repos.
	sourcesYAML := `code:
  - repo: matsen/bipartite
writing:
  - repo: matsen/notes
`
	flowCfgYAML := `paths:
  code: ~/myre
  writing: ~/mywriting
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		repo string
		want string
	}{
		{"matsen/bipartite", filepath.Join(home, "myre", "bipartite")},
		{"matsen/notes", filepath.Join(home, "mywriting", "notes")},
	} {
		rp, err := ResolveRepoPath(nexus, c.repo, ResolveContext{})
		if err != nil {
			t.Fatalf("ResolveRepoPath(%q): %v", c.repo, err)
		}
		if rp.Path != c.want {
			t.Errorf("ResolveRepoPath(%q).Path = %q, want %q", c.repo, rp.Path, c.want)
		}
		if rp.Mode != config.LayoutModeClone {
			t.Errorf("ResolveRepoPath(%q).Mode = %q, want clone", c.repo, rp.Mode)
		}
		// And the wrapper must agree.
		if got, ok := GetRepoLocalPath(nexus, c.repo); !ok || got != c.want {
			t.Errorf("GetRepoLocalPath(%q) = %q,%v want %q,true", c.repo, got, ok, c.want)
		}
	}
}

func TestResolveRepoPath_WritingTakesPrecedenceOverCode(t *testing.T) {
	// A repo listed under writing must resolve to paths.writing, not
	// paths.code, even when also (hypothetically) shadowed by a code entry.
	// This preserves the GetRepoLocalPath ordering that read-only callers
	// (checkin, board, scout) depend on.
	sourcesYAML := `code:
  - repo: matsen/notes
writing:
  - repo: matsen/notes
`
	flowCfgYAML := `paths:
  code: /tmp/code
  writing: /tmp/writing
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, "")
	rp, err := ResolveRepoPath(nexus, "matsen/notes", ResolveContext{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rp.Path, "/tmp/writing/notes"; got != want {
		t.Errorf("writing-first precedence broken: got %q want %q", got, want)
	}
}

func TestResolveRepoPath_GlobalWorktreeMode(t *testing.T) {
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	flowCfgYAML := `paths:
  code: /tmp/code
`
	globalYAML := `layout:
  mode: worktree
  worktree:
    root: "/tmp/wt/{repo}-workers"
    slot: "issue-{issue}"
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	rp, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{IssueNumber: 123, Slug: "fix-thing"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/wt/bipartite-workers/issue-123"; rp.Path != want {
		t.Errorf("Path = %q, want %q", rp.Path, want)
	}
	if rp.Mode != config.LayoutModeWorktree {
		t.Errorf("Mode = %q, want worktree", rp.Mode)
	}
	if !rp.IsNew {
		t.Errorf("IsNew = false, want true for a path that does not exist")
	}
	if want := "123-fix-thing"; rp.Branch != want {
		t.Errorf("Branch = %q, want %q", rp.Branch, want)
	}
}

func TestResolveRepoPath_WorktreeEmptyContextFallsBack(t *testing.T) {
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	flowCfgYAML := `paths:
  code: /tmp/code
`
	globalYAML := `layout:
  mode: worktree
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	rp, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !rp.FellBack {
		t.Errorf("FellBack = false, want true")
	}
	if rp.Mode != config.LayoutModeClone {
		t.Errorf("Mode = %q, want clone (post-fallback)", rp.Mode)
	}
	if want := "/tmp/code/bipartite"; rp.Path != want {
		t.Errorf("Path = %q, want %q", rp.Path, want)
	}
}

func TestResolveRepoPath_PerRepoOverrideOptOut(t *testing.T) {
	// Global says worktree, per-repo says clone — per-repo wins.
	sourcesYAML := `code:
  - repo: matsen/bipartite
    layout:
      mode: clone
`
	flowCfgYAML := `paths:
  code: /tmp/code
`
	globalYAML := `layout:
  mode: worktree
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	rp, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{IssueNumber: 42})
	if err != nil {
		t.Fatal(err)
	}
	if rp.Mode != config.LayoutModeClone {
		t.Errorf("per-repo opt-out failed: Mode = %q, want clone", rp.Mode)
	}
	if want := "/tmp/code/bipartite"; rp.Path != want {
		t.Errorf("Path = %q, want %q", rp.Path, want)
	}
}

func TestResolveRepoPath_PerRepoPartialOverride(t *testing.T) {
	// Global sets mode and slot; per-repo overrides only worktree.root.
	// Slot template must come from global; root must come from per-repo.
	sourcesYAML := `code:
  - repo: matsen/big-thing
    layout:
      worktree:
        root: "/tmp/special/{repo}"
`
	flowCfgYAML := `paths:
  code: /tmp/code
`
	globalYAML := `layout:
  mode: worktree
  worktree:
    slot: "wt-{issue}"
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	rp, err := ResolveRepoPath(nexus, "matsen/big-thing", ResolveContext{IssueNumber: 7})
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/special/big-thing/wt-7"; rp.Path != want {
		t.Errorf("Path = %q, want %q", rp.Path, want)
	}
}

func TestResolveRepoPath_RepoNotInSources(t *testing.T) {
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	nexus := setupResolverFixture(t, sourcesYAML, "", "")
	_, err := ResolveRepoPath(nexus, "someone-else/nope", ResolveContext{})
	if !errors.Is(err, ErrRepoNotInSources) {
		t.Errorf("expected ErrRepoNotInSources, got %v", err)
	}
}

func TestExpandTemplate(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "two"}
	if got, _ := expandTemplate("x{a}-{b}", vars); got != "x1-two" {
		t.Errorf("substitution: got %q", got)
	}
	if _, err := expandTemplate("{a}-{nope}-{b}", vars); err == nil {
		t.Errorf("expected error for unknown var")
	} else if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error did not mention unknown var: %v", err)
	}
	// Identical unknown vars should not be duplicated in the message.
	_, err := expandTemplate("{x}-{x}-{y}-{y}", map[string]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Count(msg, "x") != 1 || strings.Count(msg, "y") != 1 {
		t.Errorf("error did not dedupe unknown vars: %v", err)
	}
}

func TestParseLayoutInSourcesYAML(t *testing.T) {
	// Round-trip an entry with a nested layout block; verify the struct.
	sourcesYAML := `code:
  - repo: matsen/big-thing
    channel: bt
    layout:
      mode: worktree
      worktree:
        root: ~/re/bt-workers
        slot: "issue-{issue}"
`
	nexus := setupResolverFixture(t, sourcesYAML, "", "")
	sources, err := LoadSources(nexus)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources.Code) != 1 {
		t.Fatalf("expected 1 code entry, got %d", len(sources.Code))
	}
	entry := sources.Code[0]
	if entry.Layout == nil {
		t.Fatalf("Layout was nil; per-repo block did not round-trip")
	}
	if entry.Layout.Mode != "worktree" {
		t.Errorf("Mode = %q, want worktree", entry.Layout.Mode)
	}
	if entry.Layout.Worktree == nil || entry.Layout.Worktree.Root != "~/re/bt-workers" {
		t.Errorf("Worktree.Root not parsed: %+v", entry.Layout.Worktree)
	}
}

func TestParseLayoutMalformed(t *testing.T) {
	// A layout: that is not a mapping (e.g. a bare string) must produce a
	// clear parse error rather than silently being ignored.
	sourcesYAML := `code:
  - repo: matsen/bipartite
    layout: "worktree"
`
	nexus := setupResolverFixture(t, sourcesYAML, "", "")
	_, err := LoadSources(nexus)
	if err == nil {
		t.Fatal("expected error for non-mapping layout, got nil")
	}
	if !strings.Contains(err.Error(), "layout") {
		t.Errorf("error did not mention layout: %v", err)
	}
}

func TestParseLayoutUnknownField(t *testing.T) {
	// KnownFields(true) in the decoder catches typos in the layout block.
	sourcesYAML := `code:
  - repo: matsen/bipartite
    layout:
      mode: worktree
      worktrees: {root: "/oops"}
`
	nexus := setupResolverFixture(t, sourcesYAML, "", "")
	_, err := LoadSources(nexus)
	if err == nil {
		t.Fatal("expected error for unknown layout field, got nil")
	}
}
