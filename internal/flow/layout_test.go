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
	ResetFileCache()
	t.Cleanup(config.ResetGlobalConfigCache)
	t.Cleanup(ResetFileCache)
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

func TestResolveRepoPath_UnknownModeErrors(t *testing.T) {
	// A typo in layout.mode must error rather than silently falling through
	// to the side-effecting worktree path.
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	flowCfgYAML := `paths:
  code: /tmp/code
`
	globalYAML := `layout:
  mode: wirktree
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	_, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{IssueNumber: 1})
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
	if !strings.Contains(err.Error(), "wirktree") {
		t.Errorf("error did not name the bad mode: %v", err)
	}
}

func TestResolveRepoPath_PRWorktreeDefaultSlotErrors(t *testing.T) {
	// Worktree mode + a PR context + the default issue-only slot template
	// would expand to "issue-" (empty issue number). That must be a clear
	// config error, not a silently-created directory named "issue-".
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
	_, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{PRNumber: 99, Slug: "fix"})
	if err == nil {
		t.Fatal("expected error for PR + issue-only slot, got nil")
	}
	if !strings.Contains(err.Error(), "issue") {
		t.Errorf("error should name the empty {issue} variable: %v", err)
	}
}

func TestResolveRepoPath_PRWorktreeBranchSlot(t *testing.T) {
	// With a {branch}-based slot template, a PR spawn resolves cleanly —
	// branch is always populated, so the empty-var guard does not trip.
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
    slot: "{branch}"
`
	nexus := setupResolverFixture(t, sourcesYAML, flowCfgYAML, globalYAML)
	rp, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{PRNumber: 99, Slug: "fix"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/wt/bipartite-workers/pr-99-fix"; rp.Path != want {
		t.Errorf("Path = %q, want %q", rp.Path, want)
	}
	if want := "pr-99-fix"; rp.Branch != want {
		t.Errorf("Branch = %q, want %q", rp.Branch, want)
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

func TestListWorktreeSlots(t *testing.T) {
	// ListWorktreeSlots enumerates issue-* subdirs and ignores everything
	// else (files, non-prefix dirs). Names are sorted for determinism so
	// EPIC's slot list is stable across runs.
	root := t.TempDir()
	for _, sub := range []string{"issue-200", "issue-100", "scratch", "issue-150"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "issue-not-a-dir"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ListWorktreeSlots(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"issue-100", "issue-150", "issue-200"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("slot[%d] = %q, want %q", i, g, want[i])
		}
	}
}

func TestListWorktreeSlots_MissingRoot(t *testing.T) {
	// Reading a missing root is the caller's signal that the EPIC clone_root
	// itself doesn't exist — surface as an error, not an empty slice.
	_, err := ListWorktreeSlots(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected error for missing root, got nil")
	}
}

func TestResolveRepoPath_ParseErrorSurfacesDetail(t *testing.T) {
	// Pre-#149, spawn called GetRepoLocalPath which swallowed config parse
	// errors and reported "repo not found in sources.yml" — confusing when
	// the real problem was malformed YAML. After #149, spawn calls
	// ResolveRepoPath directly so the parse error reaches the user. This
	// test pins that behavior: a malformed sources.yml must yield an error
	// whose message names the file, distinct from ErrRepoNotInSources.
	sourcesYAML := "code: [unterminated\n"
	nexus := setupResolverFixture(t, sourcesYAML, "", "")

	_, err := ResolveRepoPath(nexus, "matsen/bipartite", ResolveContext{})
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if errors.Is(err, ErrRepoNotInSources) {
		t.Errorf("parse error must NOT collapse to ErrRepoNotInSources: %v", err)
	}
	if !strings.Contains(err.Error(), "sources.yml") {
		t.Errorf("error %q should name sources.yml", err)
	}
}

func TestLoadSources_CachedAcrossCalls(t *testing.T) {
	// Repeated LoadSources for the same nexus must not re-parse the file
	// (the resolver calls it on every bip spawn / bip checkin invocation
	// and we want one read per command). We assert this by mutating the
	// file in place to a *broken* YAML and verifying the second call
	// returns the originally-cached value rather than the parse error.
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	nexus := setupResolverFixture(t, sourcesYAML, "", "")

	first, err := LoadSources(nexus)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Code) != 1 || first.Code[0].Repo != "matsen/bipartite" {
		t.Fatalf("first load: %+v", first)
	}

	// Overwrite with broken YAML but preserve mtime+size by writing the
	// same number of bytes and resetting mtime. The cache key matches, so
	// the broken read should never happen.
	path := filepath.Join(nexus, "sources.yml")
	info, _ := os.Stat(path)
	broken := []byte("code: [\n") // 8 bytes; original is 31 bytes
	// Pad to the same size so mtime+size key is unchanged. (Length match
	// matters more than mtime for the cache hit assertion since file
	// systems may coalesce sub-microsecond writes.)
	for len(broken) < int(info.Size()) {
		broken = append(broken, ' ')
	}
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	second, err := LoadSources(nexus)
	if err != nil {
		t.Fatalf("expected cache hit, got parse error: %v", err)
	}
	if len(second.Code) != 1 || second.Code[0].Repo != "matsen/bipartite" {
		t.Fatalf("second load returned stale-but-different parse: %+v", second)
	}

	// Sanity check: ResetFileCache must invalidate, surfacing the bad YAML.
	ResetFileCache()
	if _, err := LoadSources(nexus); err == nil {
		t.Fatal("expected parse error after cache reset, got nil")
	}
}

func TestListWorktreeSlots_EmptyRoot(t *testing.T) {
	// An empty (but existing) root means EPIC has been configured but no
	// workers have spawned yet — empty slice, no error.
	got, err := ListWorktreeSlots(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected zero slots, got %v", got)
	}
}
