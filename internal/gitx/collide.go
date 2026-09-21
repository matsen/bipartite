package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Verdict is the three-valued result of a collision check. Two values are
// not enough: a caller acting on Found believes the pool was fully
// examined, so "I could not look" has to be sayable.
type Verdict int

const (
	// VerdictClear means the pool was examined and no overlap was found.
	VerdictClear Verdict = iota
	// VerdictFound means a real overlap was found.
	VerdictFound
	// VerdictUncheckable means some part of the pool could not be examined.
	// It dominates VerdictFound — see Report.Verdict.
	VerdictUncheckable
)

// String renders a verdict for the report's last line.
func (v Verdict) String() string {
	switch v {
	case VerdictFound:
		return "found"
	case VerdictUncheckable:
		return "could not check"
	default:
		return "clear"
	}
}

// UnreadableCount is the placeholder a clone's file count carries when the
// clone could not be read. It is a string, and deliberately not a number:
// `uncommitted=0` and `files=0` read as "nothing here" on the one line that
// exists to say otherwise, and a numeric sentinel like -1 silently passes a
// `> 0` test. See skills/lib/hazards/counting-a-failed-command.md.
const UnreadableCount = "UNREADABLE"

// CloneTouch is one live clone, its branch, and the files it touches
// relative to its merge base with origin/main.
type CloneTouch struct {
	Name      string
	Branch    string
	Files     []string
	MidRebase bool
	// Unreadable, when non-empty, is why this clone's file set could not be
	// computed. Files is nil in that case and the count must render as
	// UnreadableCount rather than as 0.
	Unreadable string
}

// Count renders the clone's touched-file count, or UnreadableCount when the
// clone could not be read.
func (c CloneTouch) Count() string {
	if c.Unreadable != "" {
		return UnreadableCount
	}
	return fmt.Sprintf("%d", len(c.Files))
}

// Collision is one file touched by more than one live clone.
type Collision struct {
	File   string
	Clones []string
}

// LandedCheck is the live-vs-landed result for one clone: what its branch
// touches since it forked, against what reached origin/main since the same
// point. Mine and Landed are printed even on a clear result — without them
// a trivially clear result, a genuinely clear result and a broken check all
// print the same word.
type LandedCheck struct {
	Name   string
	Branch string
	// Referent is what "mine" was measured from: "origin/<branch>" when the
	// branch is pushed, or "working tree" when it is not. It is printed
	// whenever it is not the remote ref, because the two are different
	// claims — the second includes uncommitted work and is not stable while
	// a rebase runs.
	Referent string
	Mine     int
	Landed   int
	Overlap  []string
	// Reason, when non-empty, is why this clone could not be compared. It
	// always has a matching entry in Report.Problems.
	Reason string
}

// PoolScope is the universe a Collide run examines. It exists because
// "every directory under the clone root" is the WRONG population, and a
// check at full power over the wrong population is worse than no check.
//
// Measured 2026-09-21 on matsengrp/phyz's ~/re/pz: 21 directories against
// 17 entries in clone_names. One was a foreign clone (caught by the origin
// test), one was not a clone at all, and two — `nightly-ci` and
// `beagle-weekly-ci` — were genuine clones of the same origin that no
// worker is ever spawned into. Each is named in a systemd unit and does
// `git fetch && git reset --hard origin/main` at run time, so a moving HEAD
// there is the EXPECTED state. A collision reported against one of them
// cannot involve a worker, and a conductor's action on a reported collision
// is to delay or resequence a spawn — so the false positive costs a stalled
// slot for a reason nobody can reproduce.
//
// `skills/bip-conductor/SKILL.md` already records this for the sibling
// currency sweep: "Its universe is clone_names from .epic-config.json, NOT
// every directory under clone_root, and that is deliberate."
type PoolScope struct {
	// Root is the pool root. Never defaulted; an empty Root is an error.
	Root string
	// Names, when non-empty, is the authoritative slot list. Directories
	// under Root that are not in it are reported as unmanaged and never
	// gated on. When empty, every non-dot directory under Root is a
	// candidate — the only available rule when there is no config to read.
	Names []string
}

// Rule describes the universe in one line, for the report to print. A
// reader cannot otherwise tell which of the two populations was examined.
func (p PoolScope) Rule() string {
	if len(p.Names) > 0 {
		return fmt.Sprintf("clone_names from .epic-config.json (%d slots)", len(p.Names))
	}
	return "every directory under the root (no clone_names available)"
}

// Report is the full result of one Collide run over one clone pool.
type Report struct {
	// Root is the pool root that was actually examined. It is never
	// defaulted — Collide's caller resolves it, and an empty root is an
	// error rather than a root.
	Root string
	// Empty is true when Root exists but holds no clone directories. That
	// is a legitimate state (a pool nobody has spawned into yet), not an
	// error, and it is reported as "nothing to check" rather than as clear.
	Empty bool
	// Pool is every candidate directory found under Root, in order.
	Pool []string
	// Foreign lists directories excluded because their origin is not this
	// pool's. They appear nowhere else in the report.
	Foreign []string
	// NotClones lists directories with no .git entry at all — pool
	// machinery (.spawn-prompts/, .preserved/) that is not a slot.
	NotClones []string
	// ScopeRule is the universe rule that was applied, printed so a reader
	// can tell which population was examined.
	ScopeRule string
	// Unmanaged lists directories present under Root but absent from
	// clone_names — CI clones and hand checkouts. Informational, never
	// gated on, and never compared against anything.
	Unmanaged []string
	// Missing lists clone_names entries with no directory. Informational:
	// a slot that does not exist cannot hold a branch, and a brand-new
	// pool legitimately has several.
	Missing []string

	Live       []CloneTouch
	Collisions []Collision
	Landed     []LandedCheck

	// Notes are informational lines (a recovered mid-rebase branch name).
	Notes []string
	// Warnings are non-fatal failures that could produce false collisions
	// but not missed ones — a failed fetch leaving origin/main stale.
	Warnings []string
	// Problems are the reasons some part of the pool could not be examined.
	// Any code path that could not look appends here, which is what makes
	// uncheckable dominate structurally rather than by the order sections
	// happen to run in.
	Problems []string
}

// Verdict resolves the report to its three-valued result. Uncheckable
// dominates found, because a caller acting on found believes the pool was
// fully examined.
func (r *Report) Verdict() Verdict {
	if len(r.Problems) > 0 {
		return VerdictUncheckable
	}
	if len(r.Collisions) > 0 {
		return VerdictFound
	}
	for _, l := range r.Landed {
		if len(l.Overlap) > 0 {
			return VerdictFound
		}
	}
	return VerdictClear
}

// LiveCount is the number of live clones whose file set was computed. It is
// the pairwise section's denominator: with fewer than two there is no pair
// to compare, and the report says so rather than printing a bare "none".
func (r *Report) LiveCount() int {
	n := 0
	for _, c := range r.Live {
		if c.Unreadable == "" {
			n++
		}
	}
	return n
}

func (r *Report) problem(format string, args ...interface{}) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

// ListPoolDirs returns the immediate subdirectories of root, excluding
// dot-directories. It is the single source of the pool's universe: the
// emptiness check and the per-clone loop both read this slice, so they
// cannot disagree about what counts as a clone. (The shell version had to
// state that as a rule — `find ... ! -name '.*'` matching the `*/` glob —
// because a real pool root holds .spawn-prompts/, .preserved/ and .mirror/,
// and a guard with a wider universe than its loop passes on a pool with no
// clones at all.)
func ListPoolDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading clone root %s: %w", root, err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, filepath.Join(root, e.Name()))
	}
	return dirs, nil
}

// poolMember is one directory under the pool root that belongs to this
// pool, with its branch already resolved (rebase recovery included) so both
// report sections read the same branch name.
type poolMember struct {
	name, path, gitDir, branch, backend string
	midRebase                           bool
}

// dirKind classifies a directory under the pool root.
type dirKind int

const (
	kindMember dirKind = iota
	kindForeign
	kindNotClone
	kindUnreadable
)

// classifyPoolDir decides whether dir is a slot in this pool. The
// three-way split matters: the shell version had one predicate (does origin
// match?) and so treated "a clone whose git failed" exactly like "a clone
// of another repository" — a silent skip in the direction this whole check
// exists to close.
//
//   - no .git entry at all  -> pool machinery, not a slot, skip silently
//   - .git present, git fails -> UNREADABLE, uncheckable; fail toward checking
//   - origin resolves and differs from the pool's modal origin -> foreign
//   - anything else (including an origin-less repo, and any repo when the
//     modal origin could not be determined) -> a member
func classifyPoolDir(dir, modalOrigin string) (dirKind, string, string) {
	gitDir, err := runGit(dir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		if _, lerr := os.Lstat(filepath.Join(dir, ".git")); lerr != nil {
			return kindNotClone, "", ""
		}
		return kindUnreadable, "", fmt.Sprintf("git rev-parse failed but .git exists: %v", err)
	}
	url, err := runGit(dir, "remote", "get-url", "origin")
	if err == nil && url != "" && modalOrigin != "" && url != modalOrigin {
		return kindForeign, gitDir, url
	}
	return kindMember, gitDir, ""
}

// modalOrigin returns the most common origin URL across dirs, or "" when no
// directory has one. Returning "" makes classifyPoolDir include everything:
// an unknown pool must not silently exclude every clone in it. Ties break
// lexicographically so a run is reproducible.
func modalOrigin(dirs []string) string {
	counts := map[string]int{}
	for _, d := range dirs {
		if url, err := runGit(d, "remote", "get-url", "origin"); err == nil && url != "" {
			counts[url]++
		}
	}
	best, bestN := "", 0
	for url, n := range counts {
		if n > bestN || (n == bestN && url < best) {
			best, bestN = url, n
		}
	}
	return best
}

// resolveBranch returns the branch a clone is on. When HEAD is detached it
// recovers the name from either rebase backend: git stores head-name under
// rebase-merge/ (the merge backend) and rebase-apply/ (--apply, am-based),
// and covering only one leaves the other a silent skip. A mid-rebase clone
// is the one you most want checked, so it must not fall out here.
//
// Returns "" when HEAD is detached and no rebase is in progress.
func resolveBranch(dir, gitDir string) (branch string, midRebase bool, backend string) {
	if b, err := runGit(dir, "branch", "--show-current"); err == nil && b != "" {
		return b, false, ""
	}
	if gitDir == "" {
		return "", false, ""
	}
	for _, rd := range []string{"rebase-merge", "rebase-apply"} {
		data, err := os.ReadFile(filepath.Join(gitDir, rd, "head-name"))
		if err != nil {
			continue
		}
		b := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "refs/heads/"))
		if b != "" {
			return b, true, rd
		}
	}
	return "", false, ""
}

// Collide runs the live-clone file-overlap check over the clone pool at
// root. root must already be resolved and non-empty; Collide never
// substitutes a default for it.
//
// Errors are returned only when the pool root itself cannot be read.
// Everything else — an unreadable clone, a missing remote ref, a branch
// with nothing to compare — lands in Report.Problems, so the caller gets a
// verdict rather than an error for conditions that mean "could not check".
func Collide(scope PoolScope) (*Report, error) {
	root := scope.Root
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("empty clone root: an empty root is not a root")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("no clone root at %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("clone root %s is not a directory", root)
	}

	rep := &Report{Root: root, ScopeRule: scope.Rule()}
	dirs, err := ListPoolDirs(root)
	if err != nil {
		return nil, err
	}
	// Narrow to the declared slots when there are any. The emptiness check
	// below reads the SAME slice as the loops, so the guard and the loop
	// cannot disagree about what counts as a clone.
	if len(scope.Names) > 0 {
		declared := make(map[string]bool, len(scope.Names))
		for _, n := range scope.Names {
			declared[n] = true
		}
		present := map[string]bool{}
		var kept []string
		for _, d := range dirs {
			name := filepath.Base(d)
			present[name] = true
			if declared[name] {
				kept = append(kept, d)
			} else {
				rep.Unmanaged = append(rep.Unmanaged, name)
			}
		}
		for _, n := range scope.Names {
			if !present[n] {
				rep.Missing = append(rep.Missing, n)
			}
		}
		dirs = kept
	}
	rep.Pool = dirs
	if len(dirs) == 0 {
		rep.Empty = true
		return rep, nil
	}

	modal := modalOrigin(dirs)

	var members []poolMember
	for _, d := range dirs {
		name := filepath.Base(d)
		kind, gitDir, detail := classifyPoolDir(d, modal)
		switch kind {
		case kindForeign:
			rep.Foreign = append(rep.Foreign, fmt.Sprintf("%s (origin %s)", name, detail))
			continue
		case kindNotClone:
			rep.NotClones = append(rep.NotClones, name)
			continue
		case kindUnreadable:
			rep.Live = append(rep.Live, CloneTouch{Name: name, Branch: UnreadableCount, Unreadable: detail})
			rep.problem("%s: cannot examine clone — %s", name, detail)
			continue
		}
		b, mid, backend := resolveBranch(d, gitDir)
		members = append(members, poolMember{name: name, path: d, gitDir: gitDir, branch: b, backend: backend, midRebase: mid})
	}

	for _, m := range members {
		if m.midRebase {
			rep.Notes = append(rep.Notes,
				fmt.Sprintf("%s is mid-rebase; using branch %q from %s/head-name", m.name, m.branch, m.backend))
		}
		if m.branch == "" || m.branch == "main" {
			continue
		}
		// A failed fetch warns and continues on purpose: a stale
		// origin/main produces FALSE collisions, not missed ones, and the
		// asymmetry is the decision rather than something to rediscover.
		if _, err := runGit(m.path, "fetch", "-q", "origin", "main"); err != nil {
			rep.Warnings = append(rep.Warnings,
				fmt.Sprintf("%s: fetch failed; origin/main may be stale -> false collisions", m.name))
		}
	}

	rep.sectionTouched(members)
	rep.sectionCollisions()
	rep.sectionLanded(members)
	return rep, nil
}

// sectionTouched fills Live: every non-main branch in the pool and the
// files it touches since its merge base with origin/main.
func (r *Report) sectionTouched(members []poolMember) {
	for _, m := range members {
		if m.branch == "" || m.branch == "main" {
			continue
		}
		touch := CloneTouch{Name: m.name, Branch: m.branch, MidRebase: m.midRebase}
		// Diff against the MERGE BASE, not origin/main. A branch even one
		// commit behind otherwise reports every file main changed since the
		// branch point as its own: measured 2026-09-14, a slot one commit
		// behind reported 15 touched files having touched 6, and the 9
		// phantoms produced a false collision.
		base, err := runGit(m.path, "merge-base", "HEAD", "origin/main")
		if err != nil {
			// The shell version fell back to origin/main here, silently
			// reinstating the two-dot behaviour the merge-base exists to
			// avoid. A fallback that reinstates the defect is worse than
			// no answer, so this is uncheckable instead.
			touch.Unreadable = fmt.Sprintf("no merge-base for HEAD and origin/main: %v", err)
			r.problem("%s: cannot compute touched files — %s", m.name, touch.Unreadable)
			r.Live = append(r.Live, touch)
			continue
		}
		files, err := diffNames(m.path, base)
		if err != nil {
			touch.Unreadable = fmt.Sprintf("git diff against %s failed: %v", base, err)
			r.problem("%s: cannot compute touched files — %s", m.name, touch.Unreadable)
			r.Live = append(r.Live, touch)
			continue
		}
		// The merge-base diff already covers uncommitted TRACKED changes —
		// it diffs the working tree against a commit. This pass is for
		// UNTRACKED files, which no diff can see.
		untracked, err := statusPaths(m.path)
		if err != nil {
			touch.Unreadable = fmt.Sprintf("git status failed: %v", err)
			r.problem("%s: cannot compute touched files — %s", m.name, touch.Unreadable)
			r.Live = append(r.Live, touch)
			continue
		}
		touch.Files = dedupeSorted(append(files, untracked...))
		r.Live = append(r.Live, touch)
	}
}

// sectionCollisions fills Collisions: files touched by more than one live
// clone. Its universe is live-vs-live.
func (r *Report) sectionCollisions() {
	byFile := map[string][]string{}
	for _, c := range r.Live {
		if c.Unreadable != "" {
			continue
		}
		for _, f := range c.Files {
			byFile[f] = append(byFile[f], c.Name)
		}
	}
	var files []string
	for f, clones := range byFile {
		if len(clones) > 1 {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	for _, f := range files {
		clones := byFile[f]
		sort.Strings(clones)
		r.Collisions = append(r.Collisions, Collision{File: f, Clones: clones})
	}
}

// sectionLanded fills Landed: each live branch against what reached
// origin/main since that branch forked. Section 2's universe is
// live-vs-live; this is live-vs-landed, a different frame rather than a
// weaker version of the same one — measured 2026-09-14, a PR went
// CONFLICTING against one that had merged two hours earlier while the
// live-vs-live section reported clean and was correct about what it
// measures.
func (r *Report) sectionLanded(members []poolMember) {
	for _, m := range members {
		if m.branch == "" {
			// Detached and not rebasing — a bisect, or a hand checkout of a
			// candidate commit. Skipping it silently would repeat this
			// section's own mistake one notch along.
			r.problem("%s: detached HEAD, no rebase in progress — cannot determine its branch", m.name)
			r.Landed = append(r.Landed, LandedCheck{Name: m.name,
				Reason: "detached HEAD, no rebase in progress"})
			continue
		}
		if m.branch == "main" {
			continue
		}
		check := LandedCheck{Name: m.name, Branch: m.branch}
		// Prefer the REMOTE ref: a local rebase rewrites HEAD underneath
		// us, so a local HEAD is not a stable referent while another
		// session is rebasing.
		//
		// But an UNPUSHED branch is the normal state in this workflow --
		// a worker's branch lives in its own clone and is not pushed until
		// the PR opens -- so refusing to check it (what the shell version
		// did) would make almost every real run exit "could not check" and
		// destroy the signal, the same way one permanently-detached
		// foreign clone once did. Fail toward CHECKING: fall back to the
		// local working tree and say which referent was used.
		//
		// The one case with no usable referent is a MID-REBASE clone with
		// no remote branch: its HEAD is a partially-replayed commit, which
		// is neither the branch's content nor a stable point.
		remote := "origin/" + m.branch
		haveRemote := true
		if _, err := runGit(m.path, "rev-parse", "--verify", "--quiet", remote); err != nil {
			haveRemote = false
		}
		if !haveRemote && m.midRebase {
			check.Reason = fmt.Sprintf("no %s and mid-rebase — HEAD is a partially-replayed commit, not a referent", remote)
			r.problem("%s: %s", m.name, check.Reason)
			r.Landed = append(r.Landed, check)
			continue
		}
		mineRef, baseRef := remote, remote
		check.Referent = remote
		if !haveRemote {
			// The working tree, not HEAD: uncommitted work is the decisive
			// case this whole check exists for, and it is the earliest
			// state in which an overlap with landed work is visible.
			mineRef, baseRef = "", "HEAD"
			check.Referent = "working tree"
		}
		base, err := runGit(m.path, "merge-base", baseRef, "origin/main")
		if err != nil {
			check.Reason = fmt.Sprintf("no merge-base for %s and origin/main", baseRef)
			r.problem("%s: %s", m.name, check.Reason)
			r.Landed = append(r.Landed, check)
			continue
		}
		var mine []string
		var err1 error
		if mineRef == "" {
			mine, err1 = diffNames(m.path, base)
			if err1 == nil {
				var untracked []string
				untracked, err1 = statusPaths(m.path)
				mine = append(mine, untracked...)
			}
		} else {
			mine, err1 = diffNames(m.path, base, mineRef)
		}
		landed, err2 := diffNames(m.path, base, "origin/main")
		if err1 != nil || err2 != nil {
			check.Reason = fmt.Sprintf("git diff against %s failed: %v %v", base, err1, err2)
			r.problem("%s: %s", m.name, check.Reason)
			r.Landed = append(r.Landed, check)
			continue
		}
		mine = dedupeSorted(mine)
		landed = dedupeSorted(landed)
		check.Mine = len(mine)
		check.Landed = len(landed)
		check.Overlap = intersect(mine, landed)
		if len(check.Overlap) == 0 && check.Mine == 0 {
			// A live branch that has changed nothing relative to its fork
			// point is either not started or not measurable, and both mean
			// a clear result here is worthless. The denominators alone made
			// this visible to a human who reads and thinks; they did not
			// make the check say so, which is the fail-open one level in.
			check.Reason = fmt.Sprintf("nothing to compare (mine=0 landed=%d) — do NOT read this as clear", check.Landed)
			r.problem("%s: %s", m.name, check.Reason)
		}
		r.Landed = append(r.Landed, check)
	}
}

// diffNames runs `git diff -z --no-renames --name-only <args>` in dir.
//
// -z because git quotes paths with unusual characters in its default
// output, and a whitespace split mangles a path containing a space into a
// silent non-match. --no-renames because git's rename detection reports
// only the NEW path for a rename, and the OLD path is exactly what collides
// with another clone still editing it; with --no-renames a rename arrives
// as a delete plus an add and both paths are reported.
func diffNames(dir string, args ...string) ([]string, error) {
	full := append([]string{"diff", "-z", "--no-renames", "--name-only"}, args...)
	out, err := runGitRaw(dir, full...)
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// statusPaths returns every path named by `git status --porcelain -z -uall`,
// including the old path of a rename or copy.
//
// -uall rather than the default: `-unormal` collapses an untracked
// directory to `dir/`, so two clones each holding an untracked
// scratch/notes.md would collide on "scratch/" and the report would not
// name the file. -z for the same quoting reason as diffNames.
func statusPaths(dir string) ([]string, error) {
	out, err := runGitRaw(dir, "status", "--porcelain", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	fields := strings.Split(out, "\x00")
	var paths []string
	for i := 0; i < len(fields); i++ {
		e := fields[i]
		// "XY path": a shorter field is the trailing empty one, or noise.
		if len(e) < 4 {
			continue
		}
		paths = append(paths, e[3:])
		// R/C entries emit a SECOND NUL field holding the old path, whole,
		// with no 3-character status prefix. Both paths are real and either
		// can collide, so emit both rather than stripping the second.
		if (e[0] == 'R' || e[0] == 'C') && i+1 < len(fields) && fields[i+1] != "" {
			i++
			paths = append(paths, fields[i])
		}
	}
	return paths, nil
}

// intersect returns the sorted intersection of two sorted, deduped slices.
func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	var out []string
	for _, s := range a {
		if set[s] {
			out = append(out, s)
		}
	}
	return out
}

// dedupeSorted sorts, drops empties, and removes duplicates.
func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// splitNUL splits NUL-delimited git output, dropping the trailing empty
// field.
func splitNUL(s string) []string {
	var out []string
	for _, f := range strings.Split(s, "\x00") {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// runGitRaw is runGit without the whitespace trim, for the -z forms whose
// delimiters and paths must survive intact.
func runGitRaw(dir string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(fullArgs, " "), err, stderr)
	}
	return string(out), nil
}
