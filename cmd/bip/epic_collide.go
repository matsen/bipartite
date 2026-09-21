package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/gitx"
	"github.com/spf13/cobra"
)

var (
	collideRoot      string
	collideSymbol    string
	collidePathspecs []string
)

var epicCollideCmd = &cobra.Command{
	Use:   "collide",
	Short: "Check a clone pool for live-branch file overlap",
	Long: `Check every clone in one EPIC clone pool for file overlap.

Reports three things, in this order:

  1. every live (non-main) branch in the pool and the files it touches
  2. files touched by MORE THAN ONE live clone       <- permits a collision
  3. each live branch against what LANDED on origin/main since it forked

Section 3 is a different frame from section 2, not a weaker version of it:
section 2 is live-vs-live, section 3 is live-vs-landed. A branch can be
clear of every other clone and still conflict with a commit that merged an
hour ago.

This check cannot be done from an EPIC session. Clone branches are local
under shared_filesystem: false, and remote refs are not a substitute --
under squash-merge every historical branch stays permanently ahead of main.
The decisive case is UNCOMMITTED work, which exists only in the clone.

SCOPE IS DERIVED, NEVER DEFAULTED, AND IT HAS TWO HALVES. With no --root,
the pool root comes from clone_root in .epic-config.json in the current
directory; if that cannot be resolved the command exits "could not check"
rather than scanning anything. An empty --root is an error, not a root.

The second half is the UNIVERSE: which directories under that root are
slots. clone_names from the same config is authoritative where it exists,
because a pool also holds clones nobody spawns into -- CI clones that reset
hard to origin/main at run time, a pinned dependency checkout -- and a
collision reported against one of those cannot involve a worker. They
are listed as unmanaged and never compared.

In WORKTREE mode there is no clone_names, and the slot list comes from the
issue-* subdirectories instead -- the same rule "bip epic watch" applies, so
the two commands agree on what a slot is. Only an explicit --root, which may
not be the pool this config describes, falls back to treating every non-dot
directory as a candidate; there, clones of other repositories are excluded
by comparing origin against the pool's modal origin.

Both halves are on the first line of output -- read it before reading the
report.

Exit codes:

  0   clear         the pool was examined and no overlap was found
  10  found         a real overlap was found
  11  could not check  some part of the pool could not be examined

11 dominates 10: a caller acting on "found" believes the pool was fully
examined. Do not read 11 as clear. These are deliberately outside the 1-6
range used by the rest of bip, where 1 is a general error and 2 is a config
error.

All report lines -- including UNCHECKABLE reasons and the verdict -- go to
stdout, so one redirect captures the whole report. Do not read $? after a
pipeline; redirect to a file and run it un-piped, or the status you read is
the last stage's.

The global --human flag has no effect here: this command's output is a
report in both modes, and the exit code is the machine-readable verdict.

KNOWN LIMITATION, issue #255: liveness is decided from the branch name
alone, so a branch whose PR has already MERGED still counts as live. Under
squash-merge every commit on such a branch reads as unmerged, so the residual
branch in a not-yet-reclaimed clone is reported forever -- as a false
collision in section 2, and as a false OVERLAPS-LANDED in section 3 whose
printed remedy is to rebase a branch that no longer has a purpose. This has
caught two conductors. Until #255 lands, check PR state before acting on any
reported overlap:

  gh pr list --state merged --head <branch> --json number,mergedAt

A non-empty result means that branch is dead and its report lines are
artifacts. Commit-identity checks do not work here: squash-merge guarantees
git cherry confirms the wrong answer with full confidence.

With --symbol, the command reports a symbol's blast radius in the current
repository instead of checking the pool: the tracked files naming it, split
into code, prose and data hits, stamped with the commit and time the count
was taken at. A blast-radius count is a measurement with a date, not a
property of the symbol.

Examples:
  bip epic collide
  bip epic collide --root ~/re/pz
  bip epic collide --symbol MATCHED_PARAMS
  bip epic collide --symbol MATCHED_PARAMS --path experiments/`,
	RunE: runEpicCollide,
}

func init() {
	epicCollideCmd.Flags().StringVar(&collideRoot, "root", "",
		"Clone-pool root to check (default: clone_root from .epic-config.json in the current directory)")
	epicCollideCmd.Flags().StringVar(&collideSymbol, "symbol", "",
		"Report this symbol's blast radius in the current repo instead of checking the pool")
	epicCollideCmd.Flags().StringSliceVar(&collidePathspecs, "path", nil,
		"Limit --symbol to these pathspecs (repeatable)")
	epicCmd.AddCommand(epicCollideCmd)
}

func runEpicCollide(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		exitWithError(ExitError, "getting cwd: %v", err)
	}

	// A flag that is accepted and then ignored is worse than one that is
	// rejected: the caller believes it scoped the run. --symbol answers a
	// different question in a different place (this repo, not the pool), so
	// the two modes cannot be combined, and --path only means anything to
	// --symbol.
	if collideSymbol != "" && cmd.Flags().Changed("root") {
		exitWithError(ExitError, "--root and --symbol are different checks in different places: --root scopes the clone pool, --symbol measures this repo. Run them separately.")
	}
	if collideSymbol == "" && cmd.Flags().Changed("path") {
		exitWithError(ExitError, "--path only applies to --symbol; the pool check examines every clone under the root")
	}

	if collideSymbol != "" {
		runCollideSymbol(cwd)
		return nil
	}

	// RunE returns nil on every verdict and the exit code is set here,
	// because main.go maps every returned error to ExitError — so a
	// three-valued verdict cannot travel as an error.
	if code := collideMain(os.Stdout, collideRoot, cmd.Flags().Changed("root"), cwd); code != ExitSuccess {
		os.Exit(code)
	}
	return nil
}

// collideMain runs the pool check and returns the exit code, writing the
// whole report to w. Split out from runEpicCollide so the verdict-to-exit-
// code mapping and the scope line are testable without a subprocess.
func collideMain(w io.Writer, rootFlag string, rootGiven bool, cwd string) int {
	scope, source, err := resolveCollideScope(rootFlag, rootGiven, cwd)
	if err != nil {
		// Scope failures are "could not check", never a scan of some other
		// directory. This is the whole reason the command exists: the shell
		// version's ROOT="${1:-$HOME/re/pz}" substituted another
		// repository's clone pool on an empty or absent argument, and
		// returned a populated, plausible, entirely wrong report.
		fmt.Fprintf(w, "scope: UNRESOLVED -- %v\n", err)
		fmt.Fprintf(w, "verdict: %s\n", gitx.VerdictUncheckable)
		return ExitCollisionUncheckable
	}
	// Printed before the check runs, and first: it is the cheapest possible
	// confirmation that you measured your own fleet. The universe rule is
	// on the same line because "which directories" is half the scope — a
	// right root with the wrong population is still the wrong answer.
	fmt.Fprintf(w, "scope: clone_root=%s (%s); universe=%s\n", scope.Root, source, scope.Rule())

	rep, err := gitx.Collide(scope)
	if err != nil {
		fmt.Fprintf(w, "cannot examine pool: %v\n", err)
		fmt.Fprintf(w, "verdict: %s\n", gitx.VerdictUncheckable)
		return ExitCollisionUncheckable
	}

	renderCollideReport(w, rep)
	switch rep.Verdict() {
	case gitx.VerdictFound:
		return ExitCollisionFound
	case gitx.VerdictUncheckable:
		return ExitCollisionUncheckable
	}
	return ExitSuccess
}

// resolveCollideScope resolves BOTH halves of the scope: the pool root and
// the universe of directories under it. It is a pure function of its
// arguments so the no-default guarantee is testable.
//
// Precedence: an explicit --root, then .epic-config.json in cwd. There is no
// third case — no hardcoded pool, and no fallback when the config is
// unreadable.
//
// An explicit --root implies directory discovery, because that root may not
// be the pool the local config describes and clone_names from a different
// pool would be a worse error than no clone_names at all. The no-argument
// form is the one to prefer.
func resolveCollideScope(rootFlag string, rootGiven bool, cwd string) (scope gitx.PoolScope, source string, err error) {
	if rootGiven {
		// Checked separately from "not given" on purpose. The shell
		// `${1:-default}` form substitutes on unset OR EMPTY, so an
		// unparseable config made an explicitly-passed root evaporate into
		// the default — and the reassuring answer about the wrong fleet is
		// what came back.
		if strings.TrimSpace(rootFlag) == "" {
			return scope, "", fmt.Errorf("--root was given an empty value; an empty root is not a root")
		}
		root := config.ExpandTilde(rootFlag)
		if !filepath.IsAbs(root) {
			root = filepath.Join(cwd, root)
		}
		return gitx.PoolScope{Root: root}, "--root", nil
	}

	cfg, cfgErr := loadEpicConfig(cwd)
	if cfgErr != nil {
		return scope, "", fmt.Errorf("no --root and cannot read a clone_root: %w", cfgErr)
	}
	root := config.ExpandTilde(cfg.CloneRoot)
	if !filepath.IsAbs(root) {
		root = filepath.Join(cwd, root)
	}
	if strings.TrimSpace(root) == "" {
		return scope, "", fmt.Errorf("clone_root in %s resolved to an empty path", epicConfigName)
	}
	// In WORKTREE mode there is no clone_names, and directory discovery is
	// NOT correct there either: a slot is an `issue-*` subdirectory, so a
	// stray checkout sharing the pool's origin would otherwise become a
	// collision candidate. flow.ListWorktreeSlots applies the same
	// slot-prefix rule the sibling `bip epic watch` uses (epic_watch.go's
	// resolveSlots), so the two commands agree on what a slot is.
	if cfg.LocalWorktrees {
		names, err := flow.ListWorktreeSlots(root)
		if err != nil {
			return scope, "", fmt.Errorf("reading clone_root %s: %w", root, err)
		}
		// An empty result is a real answer — a worktree pool with no slots
		// yet — and Collide reports it as "nothing to check". It must not
		// fall through to discovery, which would widen the universe.
		return gitx.PoolScope{Root: root, Names: names, Empty: len(names) == 0},
			"from " + epicConfigName + " (worktree mode)", nil
	}
	// Clone mode: clone_names is the authoritative slot list.
	return gitx.PoolScope{Root: root, Names: cfg.CloneNames}, "from " + epicConfigName, nil
}

// renderCollideReport writes the human-readable report. Every section
// prints its own denominator: without them a trivially clear result, a
// genuinely clear result and a broken check all print the same word.
func renderCollideReport(w io.Writer, rep *gitx.Report) {
	fmt.Fprintf(w, "pool: %d directories -> %d clones",
		len(rep.Pool), len(rep.Pool)-len(rep.Foreign)-len(rep.NotClones))
	if len(rep.Foreign) > 0 {
		fmt.Fprintf(w, ", %d excluded (not this pool's origin: %s)",
			len(rep.Foreign), strings.Join(rep.Foreign, " "))
	}
	if len(rep.NotClones) > 0 {
		fmt.Fprintf(w, ", %d not clones (%s)", len(rep.NotClones), strings.Join(rep.NotClones, " "))
	}
	fmt.Fprintln(w)
	// Informational, never gated on: these are outside the universe, so a
	// report that did not mention them would look like it had examined
	// them. Named after the sibling currency sweep's own label.
	if len(rep.Unmanaged) > 0 {
		fmt.Fprintf(w, "  (unmanaged, not compared: %s)\n", strings.Join(rep.Unmanaged, " "))
	}
	if len(rep.Missing) > 0 {
		fmt.Fprintf(w, "  (declared but absent: %s)\n", strings.Join(rep.Missing, " "))
	}

	if rep.Empty {
		// A pool root that exists and holds no clones is a legitimate state
		// — nobody has spawned into it yet — and saying "nothing to check"
		// is not the same claim as "clear".
		fmt.Fprintf(w, "no clones under %s yet -- nothing to check\n", rep.Root)
		fmt.Fprintf(w, "verdict: nothing to check (no clones in the pool)\n")
		return
	}

	for _, warn := range rep.Warnings {
		fmt.Fprintf(w, "  WARN %s\n", warn)
	}

	fmt.Fprintln(w, "\n=== live branches and their touched files ===")
	if len(rep.Live) == 0 {
		fmt.Fprintln(w, "  none")
	}
	for _, c := range rep.Live {
		branch := c.Branch
		if branch == "" {
			branch = "?"
		}
		fmt.Fprintf(w, "  %-12s %-34s files=%s", c.Name, branch, c.Count())
		if c.Unreadable != "" {
			fmt.Fprintf(w, "  (%s)", c.Unreadable)
		} else if c.MidRebase {
			fmt.Fprint(w, "  (mid-rebase)")
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "\n=== files touched by MORE THAN ONE live clone ===")
	for _, col := range rep.Collisions {
		fmt.Fprintf(w, "  COLLISION %s <- %s\n", col.File, strings.Join(col.Clones, " "))
	}
	if len(rep.Collisions) == 0 {
		fmt.Fprintln(w, "  none")
	}
	// The denominator: with fewer than two readable live clones there is no
	// pair to compare, so "none" above is arithmetic rather than evidence.
	live := rep.LiveCount()
	fmt.Fprintf(w, "  (live clones=%d, pairs=%d)\n", live, live*(live-1)/2)

	fmt.Fprintln(w, "\n=== live branch vs. what LANDED since it forked ===")
	for _, note := range rep.Notes {
		fmt.Fprintf(w, "  NOTE %s\n", note)
	}
	if len(rep.Landed) == 0 {
		fmt.Fprintln(w, "  (no pushed live branches to check)")
	}
	for _, l := range rep.Landed {
		switch {
		case l.Reason != "":
			fmt.Fprintf(w, "  UNCHECKABLE %s: %s\n", l.Name, l.Reason)
		case len(l.Overlap) > 0:
			fmt.Fprintf(w, "  OVERLAPS-LANDED %s (from %s): %s\n", l.Name, l.Referent, strings.Join(l.Overlap, " "))
			fmt.Fprintln(w, "    -> rebase and take main's content for those files; your copy predates the merge.")
			fmt.Fprintf(w, "       Verify with: git -C %s diff origin/main -- <file>   (should show ONLY your additions)\n",
				filepath.Join(rep.Root, l.Name))
		default:
			referent := ""
			if l.Referent != "" && !strings.HasPrefix(l.Referent, "origin/") {
				// Say which referent produced the number: "mine" from a
				// working tree and "mine" from a pushed branch are
				// different claims.
				referent = fmt.Sprintf(", mine from %s: %s not pushed", l.Referent, l.Branch)
			}
			fmt.Fprintf(w, "  %s clear (mine=%d landed=%d%s)\n", l.Name, l.Mine, l.Landed, referent)
		}
	}

	if len(rep.Problems) > 0 {
		fmt.Fprintln(w, "\n=== could not check ===")
		for _, p := range rep.Problems {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	fmt.Fprintf(w, "\nverdict: %s\n", rep.Verdict())
}

// runCollideSymbol reports a symbol's blast radius and exits.
func runCollideSymbol(cwd string) {
	br, err := gitx.Symbol(cwd, collideSymbol, collidePathspecs)
	if err != nil {
		fmt.Printf("symbol: %s\ncannot measure: %v\nverdict: %s\n",
			collideSymbol, err, gitx.VerdictUncheckable)
		os.Exit(ExitCollisionUncheckable)
	}
	renderBlastRadius(os.Stdout, br, time.Now())
}

// renderBlastRadius prints the three buckets and the hedge. The commit and
// the timestamp are part of the figure, not decoration: the same grep at a
// commit 14 hours later returned 39 consumers where it had returned 26, so
// a count quoted without them is not re-derivable.
func renderBlastRadius(w io.Writer, br *gitx.BlastRadius, taken time.Time) {
	fmt.Fprintf(w, "symbol: %s\n", br.Symbol)
	fmt.Fprintf(w, "repo: %s  commit: %s  taken: %s\n", br.Repo, br.Commit, taken.Format(time.RFC3339))
	if len(br.Pathspecs) > 0 {
		fmt.Fprintf(w, "pathspec: %s\n", strings.Join(br.Pathspecs, " "))
	}
	writeHits := func(label string, files []string, note string) {
		fmt.Fprintf(w, "%s: %d files%s\n", label, len(files), note)
		for _, f := range files {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
	// Reported separately, never as a total: "26 consumers" tells a worker
	// one kind of work where "17 code plus 9 prose" tells it two.
	writeHits("code hits", br.Code, "")
	writeHits("prose hits", br.Prose, "")
	writeHits("data hits", br.Data, " (committed artifacts: these record what a landed result was computed from -- usually do NOT edit)")
	fmt.Fprintf(w, "%s\n", gitx.IndirectHedge)
}
