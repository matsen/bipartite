package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures here are real git repositories, not mocks. Every behaviour
// this file pins down was a measured failure of the shell implementation it
// replaces (skills/lib/fleet-collisions.sh), and every one of them is
// invisible to a fake: merge-base versus two-dot, rename old paths,
// untracked files, a `.git` that is a file, two rebase backends.
//
// Each test names the defect it fails against in its doc comment.

// pool is one fixture: a bare upstream, a seed clone used to move main, and
// an empty clone-pool root.
type pool struct {
	base     string
	root     string
	upstream string
	seed     string
}

// newPool builds an upstream repo with one commit on main, plus an empty
// pool root beside it.
func newPool(t *testing.T) *pool {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}
	base := t.TempDir()
	p := &pool{
		base:     base,
		root:     filepath.Join(base, "pool"),
		upstream: filepath.Join(base, "up.git"),
		seed:     filepath.Join(base, "seed"),
	}
	if err := os.MkdirAll(p.root, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, base, "init", "--bare", "--quiet", "--initial-branch=main", "up.git")
	mustGit(t, base, "clone", "--quiet", p.upstream, "seed")
	configureRepo(t, p.seed)
	writeAt(t, p.seed, "README.md", "hi\n")
	mustGit(t, p.seed, "add", "-A")
	mustGit(t, p.seed, "commit", "--quiet", "-m", "M1")
	mustGit(t, p.seed, "push", "--quiet", "-u", "origin", "main")
	return p
}

// clone adds a clone of the pool's upstream at <root>/<name>.
func (p *pool) clone(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(p.root, name)
	mustGit(t, p.base, "clone", "--quiet", p.upstream, dir)
	configureRepo(t, dir)
	return dir
}

// landOnMain commits the given files in the seed clone and pushes main.
func (p *pool) landOnMain(t *testing.T, msg string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		writeAt(t, p.seed, path, content)
	}
	mustGit(t, p.seed, "add", "-A")
	mustGit(t, p.seed, "commit", "--quiet", "-m", msg)
	mustGit(t, p.seed, "push", "--quiet", "origin", "main")
}

// branchWith creates branch in dir, commits files, and pushes it so the
// live-vs-landed section has an origin/<branch> to compare.
func branchWith(t *testing.T, dir, branch string, files map[string]string) {
	t.Helper()
	mustGit(t, dir, "checkout", "--quiet", "-b", branch)
	for path, content := range files {
		writeAt(t, dir, path, content)
	}
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "--quiet", "-m", "work on "+branch)
	mustGit(t, dir, "push", "--quiet", "-u", "origin", branch)
}

func configureRepo(t *testing.T, dir string) {
	t.Helper()
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test User")
	mustGit(t, dir, "config", "commit.gpgsign", "false")
}

func writeAt(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// touchedBy returns the report's file set for one clone, and whether it was
// present at all.
func touchedBy(rep *Report, name string) ([]string, bool) {
	for _, c := range rep.Live {
		if c.Name == name {
			return c.Files, true
		}
	}
	return nil, false
}

func landedFor(t *testing.T, rep *Report, name string) LandedCheck {
	t.Helper()
	for _, l := range rep.Landed {
		if l.Name == name {
			return l
		}
	}
	t.Fatalf("no live-vs-landed entry for %q; report has %+v", name, rep.Landed)
	return LandedCheck{}
}

func mustCollide(t *testing.T, root string, names ...string) *Report {
	t.Helper()
	rep, err := Collide(PoolScope{Root: root, Names: names})
	if err != nil {
		t.Fatalf("Collide(%s, %v): %v", root, names, err)
	}
	return rep
}

// TestCollide_MergeBaseNotTwoDot fails against an implementation that diffs
// against origin/main rather than the merge base. Measured 2026-09-14: a
// slot one commit behind reported 15 touched files having touched 6, and the
// 9 phantoms produced a false collision.
func TestCollide_MergeBaseNotTwoDot(t *testing.T) {
	p := newPool(t)
	// A forks at M1 and touches only a.txt.
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"a.txt": "a\n"})
	// main moves twice after A forked.
	p.landOnMain(t, "M2", map[string]string{"c.txt": "c\n"})
	p.landOnMain(t, "M3", map[string]string{"d.txt": "d\n"})
	// B forks at M3 and touches only b.txt.
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"b.txt": "b\n"})

	rep := mustCollide(t, p.root)

	files, ok := touchedBy(rep, "alpha")
	if !ok {
		t.Fatalf("alpha missing from report: %+v", rep.Live)
	}
	if got, want := strings.Join(files, " "), "a.txt"; got != want {
		t.Errorf("alpha touched files = %q, want %q (a two-dot diff would add main's c.txt and d.txt)", got, want)
	}
	if len(rep.Collisions) != 0 {
		t.Errorf("collisions = %+v, want none", rep.Collisions)
	}
	if v := rep.Verdict(); v != VerdictClear {
		t.Errorf("verdict = %v, want clear; problems=%v", v, rep.Problems)
	}
	// The denominators are the point of the clear result, not decoration.
	if l := landedFor(t, rep, "alpha"); l.Mine != 1 || l.Landed != 2 {
		t.Errorf("alpha landed check = mine=%d landed=%d, want mine=1 landed=2", l.Mine, l.Landed)
	}
}

// TestCollide_RenameOldPathAndSpaceInPath fails against an implementation
// that lets git's rename detection drop the old path, and against one that
// splits paths on whitespace. Both paths of a rename are real and either can
// collide with another clone still editing it.
func TestCollide_RenameOldPathAndSpaceInPath(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed files", map[string]string{
		"x/common.py": "shared\n",
		"docs/a b.md": "doc\n",
	})

	a := p.clone(t, "alpha")
	mustGit(t, a, "checkout", "--quiet", "-b", "a-work")
	mustGit(t, a, "mv", "x/common.py", "x/shared.py")
	writeAt(t, a, "docs/a b.md", "doc from a\n")
	mustGit(t, a, "add", "-A")
	mustGit(t, a, "commit", "--quiet", "-m", "rename and edit")
	mustGit(t, a, "push", "--quiet", "-u", "origin", "a-work")

	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{
		"x/common.py": "edited by b\n",
		"docs/a b.md": "doc from b\n",
	})

	rep := mustCollide(t, p.root)

	var got []string
	for _, c := range rep.Collisions {
		got = append(got, c.File)
	}
	want := []string{"docs/a b.md", "x/common.py"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("collisions = %v, want %v", got, want)
	}
	if v := rep.Verdict(); v != VerdictFound {
		t.Errorf("verdict = %v, want found; problems=%v", v, rep.Problems)
	}
}

// TestCollide_UntrackedFileCollides is the case the status pass exists for.
// The merge-base diff already covers uncommitted TRACKED changes; an
// untracked file is invisible to every diff, and it is the earliest state a
// collision appears in.
func TestCollide_UntrackedFileCollides(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"a.txt": "a\n"})
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"b.txt": "b\n"})
	// Uncommitted and never added, in both clones.
	writeAt(t, a, "scratch/notes.md", "a notes\n")
	writeAt(t, b, "scratch/notes.md", "b notes\n")

	rep := mustCollide(t, p.root)

	found := false
	for _, c := range rep.Collisions {
		if c.File == "scratch/notes.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("collisions = %+v, want scratch/notes.md (an -unormal status pass reports only scratch/)", rep.Collisions)
	}
}

// TestCollide_DegenerateRefIsUncheckable fails against an implementation
// that reports a branch with nothing to compare as clear. The denominators
// alone made this visible to a human who reads and thinks; they did not
// make the check say so, which is the fail-open one level in.
func TestCollide_DegenerateRefIsUncheckable(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	// A live branch with no commits of its own, pushed.
	mustGit(t, a, "checkout", "--quiet", "-b", "a-work")
	mustGit(t, a, "push", "--quiet", "-u", "origin", "a-work")
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"b.txt": "b\n"})

	rep := mustCollide(t, p.root)

	if v := rep.Verdict(); v != VerdictUncheckable {
		t.Errorf("verdict = %v, want could-not-check; problems=%v", v, rep.Problems)
	}
	l := landedFor(t, rep, "alpha")
	if l.Reason == "" {
		t.Errorf("alpha reported without a reason: %+v", l)
	}
	if !strings.Contains(strings.Join(rep.Problems, "\n"), "nothing to compare") {
		t.Errorf("problems = %v, want one naming alpha as having nothing to compare", rep.Problems)
	}
}

// TestCollide_SingleClonePoolDoesNotSelfCollide fails against an
// implementation that intersects a clone's file set with itself: a pool of
// one clone has no pair to compare, and every file it touches must not be
// reported as a collision.
func TestCollide_SingleClonePoolDoesNotSelfCollide(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"a.txt": "a\n", "b.txt": "b\n"})

	rep := mustCollide(t, p.root)

	if len(rep.Collisions) != 0 {
		t.Errorf("collisions = %+v, want none in a one-clone pool", rep.Collisions)
	}
	if got := rep.LiveCount(); got != 1 {
		t.Errorf("LiveCount = %d, want 1 (the pairwise section's printed denominator)", got)
	}
	if v := rep.Verdict(); v != VerdictClear {
		t.Errorf("verdict = %v, want clear; problems=%v", v, rep.Problems)
	}
}

// TestCollide_UnreadableCloneIsUncheckableAndNotZero pins two things at
// once: a clone that cannot be read is never counted as zero touched files,
// and could-not-check dominates found even when a real collision is also
// present. A caller acting on "found" believes the pool was fully examined.
func TestCollide_UnreadableCloneIsUncheckableAndNotZero(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed", map[string]string{"CLAUDE.md": "rules\n"})
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"CLAUDE.md": "rules from a\n"})
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"CLAUDE.md": "rules from b\n"})

	// A clone whose .git is a file pointing nowhere: git fails, but .git
	// exists, so this is "could not look" rather than "not a clone".
	broken := filepath.Join(p.root, "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAt(t, broken, ".git", "gitdir: /nonexistent/definitely-not-here\n")

	rep := mustCollide(t, p.root)

	if len(rep.Collisions) == 0 {
		t.Fatalf("expected the CLAUDE.md collision to be found as well: %+v", rep)
	}
	if v := rep.Verdict(); v != VerdictUncheckable {
		t.Errorf("verdict = %v, want could-not-check (it must dominate found)", v)
	}
	var entry *CloneTouch
	for i := range rep.Live {
		if rep.Live[i].Name == "broken" {
			entry = &rep.Live[i]
		}
	}
	if entry == nil {
		t.Fatalf("broken clone absent from the report; a clone that cannot be read must not be silently skipped: %+v", rep.Live)
	}
	if entry.Unreadable == "" {
		t.Error("broken clone reported without a reason")
	}
	if got := entry.Count(); got != UnreadableCount {
		t.Errorf("broken clone Count() = %q, want %q — a failed read must not render as a number", got, UnreadableCount)
	}
}

// TestCollide_DenominatorsOnClearResult fails against an implementation
// that prints "clear" with no numbers. `clear (mine=8 landed=0)` says
// nothing has landed since that branch forked; `clear (mine=36 landed=44)`
// says the check had real data on both sides.
func TestCollide_DenominatorsOnClearResult(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{
		"one.txt": "1\n", "two.txt": "2\n", "three.txt": "3\n",
	})

	rep := mustCollide(t, p.root)
	l := landedFor(t, rep, "alpha")
	if l.Mine != 3 || l.Landed != 0 || l.Reason != "" {
		t.Errorf("first run = mine=%d landed=%d reason=%q, want mine=3 landed=0 clear", l.Mine, l.Landed, l.Reason)
	}
	if v := rep.Verdict(); v != VerdictClear {
		t.Errorf("first run verdict = %v, want clear; problems=%v", v, rep.Problems)
	}

	// Two files land on main past the branch point, touching neither of
	// the branch's files: the landed denominator must move, the verdict
	// must not.
	p.landOnMain(t, "M2", map[string]string{"four.txt": "4\n", "five.txt": "5\n"})

	rep = mustCollide(t, p.root)
	l = landedFor(t, rep, "alpha")
	if l.Mine != 3 || l.Landed != 2 || l.Reason != "" {
		t.Errorf("second run = mine=%d landed=%d reason=%q, want mine=3 landed=2 clear", l.Mine, l.Landed, l.Reason)
	}
	if v := rep.Verdict(); v != VerdictClear {
		t.Errorf("second run verdict = %v, want clear; problems=%v", v, rep.Problems)
	}
}

// TestCollide_ForeignCloneExcluded fails against the measured regression
// where one permanently-detached clone of ANOTHER repository (a pinned
// iqtree2 checkout living in the phyz pool) reported UNCHECKABLE on every
// run and made every run exit non-zero — destroying the signal the
// stickiness fix had been added to protect, one commit after adding it.
func TestCollide_ForeignCloneExcluded(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"a.txt": "a\n"})
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"b.txt": "b\n"})

	// A clone of a different upstream, parked at a detached tag the way a
	// pinned dependency checkout is.
	other := filepath.Join(p.base, "other.git")
	mustGit(t, p.base, "init", "--bare", "--quiet", "--initial-branch=main", "other.git")
	otherSeed := filepath.Join(p.base, "other-seed")
	mustGit(t, p.base, "clone", "--quiet", other, "other-seed")
	configureRepo(t, otherSeed)
	writeAt(t, otherSeed, "tool.c", "int main(){}\n")
	mustGit(t, otherSeed, "add", "-A")
	mustGit(t, otherSeed, "commit", "--quiet", "-m", "v1")
	mustGit(t, otherSeed, "tag", "v1")
	mustGit(t, otherSeed, "push", "--quiet", "-u", "origin", "main")
	mustGit(t, otherSeed, "push", "--quiet", "origin", "v1")
	pinned := filepath.Join(p.root, "pinned-tool")
	mustGit(t, p.base, "clone", "--quiet", other, pinned)
	configureRepo(t, pinned)
	mustGit(t, pinned, "checkout", "--quiet", "--detach", "v1")

	rep := mustCollide(t, p.root)

	if _, ok := touchedBy(rep, "pinned-tool"); ok {
		t.Error("the foreign clone appears in the live-branch section; it must be excluded, not reported")
	}
	for _, l := range rep.Landed {
		if l.Name == "pinned-tool" {
			t.Error("the foreign clone appears in the live-vs-landed section")
		}
	}
	if len(rep.Foreign) != 1 || !strings.HasPrefix(rep.Foreign[0], "pinned-tool") {
		t.Errorf("Foreign = %v, want exactly the pinned-tool clone", rep.Foreign)
	}
	if v := rep.Verdict(); v != VerdictClear {
		t.Errorf("verdict = %v, want clear; problems=%v", v, rep.Problems)
	}
}

// TestCollide_MidRebaseWorktreeIsChecked fails against `[ -d "$d/.git" ]`:
// in a linked worktree `.git` is a FILE, so a directory test is false and
// both weird-state paths silently skip it, reinstating in a supported mode
// the exact gap they exist to close. It also fails against an
// implementation that covers only one rebase backend — this fixture uses
// the --apply backend, whose state lives in rebase-apply/, not
// rebase-merge/.
func TestCollide_MidRebaseWorktreeIsChecked(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed", map[string]string{"f.txt": "base\n"})

	midRebaseWorktree(t, p, true)

	rep := mustCollide(t, p.root)

	if !strings.Contains(strings.Join(rep.Notes, "\n"), "mid-rebase") {
		t.Errorf("Notes = %v, want one recording the recovered mid-rebase branch", rep.Notes)
	}
	l := landedFor(t, rep, "wt")
	if l.Branch != "wt-work" {
		t.Errorf("recovered branch = %q, want %q (from rebase-apply/head-name)", l.Branch, "wt-work")
	}
	if l.Reason != "" {
		t.Errorf("wt reported as uncheckable (%q); the comparison must run for a clone in a weird state", l.Reason)
	}
	if strings.Join(l.Overlap, " ") != "f.txt" {
		t.Errorf("overlap = %v, want f.txt — its branch edits a file that landed on main since it forked", l.Overlap)
	}
}

// TestCollide_EmptyPoolRoot fails against an implementation that reports an
// empty or machinery-only pool root as clear. Measured 2026-09-14: globbing
// an empty directory errors in zsh and does one silent iteration on a
// nonexistent path in bash — and a real pool root holds .spawn-prompts/,
// .preserved/ and .mirror/, so the dotfile filter is load-bearing.
func TestCollide_EmptyPoolRoot(t *testing.T) {
	for _, tc := range []struct {
		name     string
		makeDirs []string
	}{
		{"empty", nil},
		{"machinery only", []string{".spawn-prompts", ".preserved"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, d := range tc.makeDirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			rep := mustCollide(t, root)
			if !rep.Empty {
				t.Errorf("Empty = false, want true; pool = %v", rep.Pool)
			}
			if len(rep.Live) != 0 || len(rep.Collisions) != 0 {
				t.Errorf("empty pool produced report content: %+v", rep)
			}
		})
	}
}

// TestCollide_EmptyRootIsNotARoot pins the fix rather than the port: an
// empty scope is an error, never a default. `${1:-$HOME/re/pz}` substitutes
// on unset OR EMPTY, which is how an explicitly-passed root evaporated into
// another repository's clone pool.
func TestCollide_EmptyRootIsNotARoot(t *testing.T) {
	for _, root := range []string{"", "   "} {
		if _, err := Collide(PoolScope{Root: root}); err == nil {
			t.Errorf("Collide(%q) succeeded; an empty root is not a root", root)
		}
	}
	if _, err := Collide(PoolScope{Root: filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Error("Collide on a nonexistent root succeeded; want an error the caller maps to could-not-check")
	}
}

// TestStatusPaths_RenameEmitsBothPaths covers the NUL framing directly: an
// R entry's old path arrives as a second NUL field with no status prefix,
// and either path can collide.
func TestStatusPaths_RenameEmitsBothPaths(t *testing.T) {
	dir := initRepo(t)
	writeAt(t, dir, "old name.txt", "x\n")
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "--quiet", "-m", "add")
	mustGit(t, dir, "mv", "old name.txt", "new name.txt")

	paths, err := statusPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(dedupeSorted(paths), "|")
	if joined != "new name.txt|old name.txt" {
		t.Errorf("statusPaths = %q, want both paths of the rename", joined)
	}
}

// TestCollide_NoMergeBaseIsUncheckableNotTwoDot pins the one place this port
// deliberately diverges from the script it replaces. The shell version fell
// back to `origin/main` when `merge-base` failed, silently reinstating the
// two-dot behaviour the merge base exists to avoid — a fallback that
// reinstates the defect is worse than no answer.
func TestCollide_NoMergeBaseIsUncheckableNotTwoDot(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "M2", map[string]string{"landed.txt": "x\n"})
	a := p.clone(t, "alpha")
	// An orphan branch shares no history with main, so HEAD and
	// origin/main have no merge base at all.
	mustGit(t, a, "checkout", "--quiet", "--orphan", "a-work")
	mustGit(t, a, "rm", "-rq", "--cached", ".")
	writeAt(t, a, "a.txt", "a\n")
	mustGit(t, a, "add", "-A")
	mustGit(t, a, "commit", "--quiet", "-m", "orphan work")

	rep := mustCollide(t, p.root)

	var entry *CloneTouch
	for i := range rep.Live {
		if rep.Live[i].Name == "alpha" {
			entry = &rep.Live[i]
		}
	}
	if entry == nil {
		t.Fatalf("alpha absent from the report: %+v", rep.Live)
	}
	if !strings.Contains(entry.Unreadable, "merge-base") {
		t.Errorf("alpha Unreadable = %q, want it to name the missing merge base rather than falling back to origin/main", entry.Unreadable)
	}
	if entry.Files != nil {
		t.Errorf("alpha reported files %v; a fallback to origin/main would report main's files as its own", entry.Files)
	}
	if v := rep.Verdict(); v != VerdictUncheckable {
		t.Errorf("verdict = %v, want could-not-check", v)
	}
}

// TestCollide_FailedFetchWarnsAndContinues states the asymmetry as a
// decision rather than leaving it to be rediscovered: a stale origin/main
// produces FALSE collisions, not missed ones, so a failed fetch must warn
// and keep checking rather than abandon the run.
func TestCollide_FailedFetchWarnsAndContinues(t *testing.T) {
	p := newPool(t)
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"shared.txt": "a\n"})
	b := p.clone(t, "bravo")
	branchWith(t, b, "b-work", map[string]string{"shared.txt": "b\n"})
	// Both origins break the same way, so the broken URL is the pool's
	// modal origin and neither clone is excluded as foreign. The
	// remote-tracking refs from the original clone survive, so the check
	// can still run against a stale origin/main.
	gone := filepath.Join(p.base, "gone.git")
	mustGit(t, a, "remote", "set-url", "origin", gone)
	mustGit(t, b, "remote", "set-url", "origin", gone)

	rep := mustCollide(t, p.root)

	if len(rep.Warnings) != 2 {
		t.Errorf("warnings = %v, want one per clone naming the failed fetch", rep.Warnings)
	}
	if !strings.Contains(strings.Join(rep.Warnings, "\n"), "origin/main may be stale") {
		t.Errorf("warnings = %v, want them to say why a stale origin/main matters", rep.Warnings)
	}
	if v := rep.Verdict(); v != VerdictFound {
		t.Errorf("verdict = %v, want found — the check continued past the fetch failure; problems=%v", v, rep.Problems)
	}
}

// midRebaseWorktree builds a linked worktree (so its `.git` is a FILE) that
// is stopped mid-rebase on the --apply backend (so its state lives in
// rebase-apply/, not rebase-merge/), with main having moved conflictingly
// underneath it. push controls whether the branch reached origin.
func midRebaseWorktree(t *testing.T, p *pool, push bool) string {
	t.Helper()
	// The primary clone lives OUTSIDE the pool; the pool holds the linked
	// worktree, so the thing being checked has a .git file.
	primary := filepath.Join(p.base, "primary")
	mustGit(t, p.base, "clone", "--quiet", p.upstream, "primary")
	configureRepo(t, primary)
	wt := filepath.Join(p.root, "wt")
	if err := AddWorktree(primary, wt, "wt-work"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	configureRepo(t, wt)
	writeAt(t, wt, "f.txt", "mine\n")
	mustGit(t, wt, "commit", "--quiet", "-am", "wt change")
	if push {
		mustGit(t, wt, "push", "--quiet", "-u", "origin", "wt-work")
	}

	// main moves under it, conflictingly.
	p.landOnMain(t, "M2", map[string]string{"f.txt": "theirs\n"})

	mustGit(t, wt, "fetch", "--quiet", "origin", "main")
	// Expected to fail: it stops mid-rebase, which is the state under test.
	_ = exec.Command("git", "-C", wt, "rebase", "--apply", "origin/main").Run()
	// FATAL, not SKIP. Everything above is under this fixture's own control,
	// so failing to reach rebase-apply state is either a harness bug or a git
	// version whose layout differs — and the production code reads these exact
	// paths (resolveBranch), so a layout change breaks the command too. A skip
	// here would hide that: `go test`'s summary does not distinguish a skipped
	// test from a passing one, and these two lines are the only coverage of the
	// worktree-plus-apply-backend scenario this file exists to pin.
	if _, err := os.Stat(filepath.Join(primary, ".git", "worktrees", "wt", "rebase-apply", "head-name")); err != nil {
		t.Fatalf("fixture did not produce rebase-apply state, so the mid-rebase paths are untested: %v", err)
	}
	if b, _ := runGit(wt, "branch", "--show-current"); b != "" {
		t.Fatalf("fixture is not detached mid-rebase (branch=%q); the recovery path is untested", b)
	}
	return wt
}

// TestCollide_UnpushedBranchChecksTheWorkingTree is the one place this port
// deliberately checks MORE than the script it replaces. A worker's branch
// lives in its own clone and is not pushed until the PR opens, so the
// script's "no origin/<branch>, cannot compare" made the normal state
// uncheckable — and a verdict that is uncheckable on almost every run is
// the signal-destruction failure this file's foreign-clone exclusion exists
// to prevent. Fail toward CHECKING: use the working tree, and say so.
func TestCollide_UnpushedBranchChecksTheWorkingTree(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed", map[string]string{"shared.md": "base\n"})
	a := p.clone(t, "alpha")
	mustGit(t, a, "checkout", "--quiet", "-b", "a-work")
	// Never committed, never pushed: the earliest state a collision with
	// landed work exists in, and the decisive case for this whole check.
	writeAt(t, a, "shared.md", "edited by a\n")
	// main moves, touching the same file.
	p.landOnMain(t, "M2", map[string]string{"shared.md": "landed\n"})

	rep := mustCollide(t, p.root)

	l := landedFor(t, rep, "alpha")
	if l.Reason != "" {
		t.Fatalf("alpha reported as uncheckable (%q); an unpushed branch is the normal state, not an unexaminable one", l.Reason)
	}
	if l.Referent != "working tree" {
		t.Errorf("referent = %q, want %q so the reader knows which number they have", l.Referent, "working tree")
	}
	if strings.Join(l.Overlap, " ") != "shared.md" {
		t.Errorf("overlap = %v, want shared.md", l.Overlap)
	}
	if v := rep.Verdict(); v != VerdictFound {
		t.Errorf("verdict = %v, want found; problems=%v", v, rep.Problems)
	}
}

// TestCollide_MidRebaseUnpushedIsUncheckable is the one state with no
// usable referent: HEAD is a partially-replayed commit, which is neither
// the branch's content nor a stable point, and there is no remote branch to
// fall back to.
func TestCollide_MidRebaseUnpushedIsUncheckable(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed", map[string]string{"f.txt": "base\n"})
	midRebaseWorktree(t, p, false)

	rep := mustCollide(t, p.root)

	l := landedFor(t, rep, "wt")
	if l.Reason == "" {
		t.Fatalf("mid-rebase unpushed branch reported a verdict: %+v", l)
	}
	if !strings.Contains(l.Reason, "partially-replayed") {
		t.Errorf("reason = %q, want it to name why HEAD is not a referent here", l.Reason)
	}
	if v := rep.Verdict(); v != VerdictUncheckable {
		t.Errorf("verdict = %v, want could-not-check", v)
	}
}

// TestCollide_ScopedToCloneNames fails against a check that examines every
// directory under the pool root. Measured 2026-09-21 on matsengrp/phyz's
// ~/re/pz: 21 directories against 17 clone_names, and two of the extras
// (`nightly-ci`, `beagle-weekly-ci`) are GENUINE clones of the same origin
// that no worker is ever spawned into — each is named in a systemd unit and
// does `git fetch && git reset --hard origin/main` at run time, so a moving
// HEAD there is the expected state. A conductor's action on a reported
// collision is to delay or resequence a spawn, so a false positive against
// one of them costs a stalled slot for a reason nobody can reproduce.
//
// The FIRST subtest is the regression guard — verified by mutation: stubbing
// out the narrowing block fails `declared slots only` and leaves `no
// clone_names` passing, because that second case supplies no names and never
// reaches the narrowing at all. The second subtest is a CONTROL: it rules out
// "the fixture's edits do not really overlap" as an alternative explanation
// for the first one's clear verdict. An earlier version of this comment called
// the second one load-bearing, which was backwards.
func TestCollide_ScopedToCloneNames(t *testing.T) {
	p := newPool(t)
	p.landOnMain(t, "seed", map[string]string{"shared.md": "base\n"})
	a := p.clone(t, "alpha")
	branchWith(t, a, "a-work", map[string]string{"shared.md": "from a\n"})
	p.clone(t, "bravo") // declared, on main, touches nothing
	// A real clone of the same origin, on a live branch touching the same
	// file — and deliberately not a slot.
	ci := p.clone(t, "nightly-ci")
	branchWith(t, ci, "ci-work", map[string]string{"shared.md": "from ci\n"})

	t.Run("declared slots only", func(t *testing.T) {
		rep := mustCollide(t, p.root, "alpha", "bravo", "charlie")
		if len(rep.Collisions) != 0 {
			t.Errorf("collisions = %+v, want none: nightly-ci is not a slot and cannot collide with a worker", rep.Collisions)
		}
		if strings.Join(rep.Unmanaged, " ") != "nightly-ci" {
			t.Errorf("Unmanaged = %v, want nightly-ci reported informationally rather than compared", rep.Unmanaged)
		}
		if _, ok := touchedBy(rep, "nightly-ci"); ok {
			t.Error("nightly-ci appears in the live-branch section; it is outside the universe")
		}
		// A declared slot with no directory is informational: a slot that
		// does not exist cannot hold a branch, and a new pool has several.
		if strings.Join(rep.Missing, " ") != "charlie" {
			t.Errorf("Missing = %v, want charlie named so the reader can see the declared-but-absent slot", rep.Missing)
		}
		if !strings.Contains(rep.ScopeRule, "declared slot list") {
			t.Errorf("ScopeRule = %q, want it to say the universe is a declared list", rep.ScopeRule)
		}
		if v := rep.Verdict(); v != VerdictClear {
			t.Errorf("verdict = %v, want clear; problems=%v", v, rep.Problems)
		}
	})

	t.Run("no clone_names falls back to discovery", func(t *testing.T) {
		rep := mustCollide(t, p.root)
		if len(rep.Collisions) != 1 || rep.Collisions[0].File != "shared.md" {
			t.Fatalf("collisions = %+v, want shared.md — without clone_names every directory is a candidate", rep.Collisions)
		}
		if !strings.Contains(rep.ScopeRule, "every directory") {
			t.Errorf("ScopeRule = %q, want it to say the universe was directory discovery", rep.ScopeRule)
		}
	})
}
