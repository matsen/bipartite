package gitx

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// BlastRadius is the consumer set of a named symbol: the tracked files that
// mention it, split by what editing them would involve.
//
// The split is the point. A single total tells a worker one kind of work
// where the real answer is two: prose hits go stale on the edit exactly
// like code hits do, but fixing them is a different job, and data hits
// often must not be edited at all. Measured on matsengrp/phyz: "26
// consumers" was 17 code files plus 9 READMEs stating the value in prose.
type BlastRadius struct {
	Symbol string
	// Repo is the repository the grep ran in, and Commit is the commit it
	// ran at. Both are part of the figure: the same grep at a commit ~14
	// hours later returned 39 consumers where it had returned 26, so a
	// blast-radius count is a measurement with a date, not a property of
	// the symbol.
	Repo   string
	Commit string
	// Pathspecs, when non-empty, is the subtree the grep was limited to.
	Pathspecs []string

	Code  []string
	Prose []string
	Data  []string
}

// Total is the number of files naming the symbol. It exists for
// completeness; report the three buckets rather than this.
func (b *BlastRadius) Total() int { return len(b.Code) + len(b.Prose) + len(b.Data) }

// IndirectHedge is the third line every blast-radius report carries. A
// `git grep` cannot see files that import the defining module without ever
// naming the symbol, and a worker who greps gets the direct set and
// believes it has the set. Measured on matsengrp/phyz at three commits: 14
// indirect consumers, unchanged, while the direct count moved 26 -> 39.
const IndirectHedge = "indirect consumers: NOT MEASURED. `git grep` cannot see files that import the " +
	"defining module without naming the symbol; the direct count above is not the consumer set. " +
	"Read the importers of the defining module too."

// proseExts are documentation and prose file types. A hit here goes stale
// on the edit like a code hit, but fixing it is a different job.
var proseExts = map[string]bool{
	".md": true, ".markdown": true, ".rst": true, ".txt": true,
	".tex": true, ".org": true, ".adoc": true,
}

// dataExts are committed results and data artifacts. A hit here often must
// NOT be edited — it is a record of what a landed result was computed from,
// and "fix and regenerate" is the wrong remedy even when the fix is right.
var dataExts = map[string]bool{
	".tsv": true, ".csv": true, ".jsonl": true, ".ndjson": true,
	".log": true, ".out": true, ".err": true, ".parquet": true,
	".npy": true, ".npz": true, ".pkl": true,
}

// classifyHit buckets one path. Unknown extensions fall to code
// deliberately: a new source extension misfiled as code overstates the code
// bucket, which is the bucket a worker is already going to read.
func classifyHit(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)
	switch {
	case proseExts[ext]:
		return "prose"
	case dataExts[ext]:
		return "data"
	case ext == "" && (strings.HasPrefix(base, "README") || strings.HasPrefix(base, "CHANGELOG")):
		return "prose"
	default:
		return "code"
	}
}

// Symbol reports the blast radius of symbol in the repository at repoDir,
// optionally limited to pathspecs.
//
// The grep is fixed-string and whole-file-list (`git grep -l --fixed-strings`):
// a symbol name is a literal, and regex metacharacters in one would
// otherwise silently change the population being counted.
func Symbol(repoDir, symbol string, pathspecs []string) (*BlastRadius, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("empty symbol: nothing to measure")
	}
	commit, err := runGit(repoDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolving HEAD in %s: %w", repoDir, err)
	}
	br := &BlastRadius{Symbol: symbol, Repo: repoDir, Commit: commit, Pathspecs: pathspecs}

	// `-e <symbol>` rather than a bare positional pattern: with a pathspec
	// list following, a positional pattern plus two `--` separators is
	// ambiguous, and the failure would be a silently different population
	// rather than an error.
	args := append([]string{"grep", "-l", "-z", "--fixed-strings", "-e", symbol}, "--")
	args = append(args, pathspecs...)
	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	// cmd.Output returns stdout even on a non-zero exit, which matters
	// because `git grep` exits 1 with no output when nothing matched. That
	// is a real answer (zero consumers), not a failure, and reporting it as
	// "could not check" would conflate two different verdicts.
	out, err := cmd.Output()
	if err != nil {
		if isExitCode(err, 1) && len(out) == 0 {
			return br, nil
		}
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		return nil, fmt.Errorf("git grep for %q in %s: %w (%s)", symbol, repoDir, err, stderr)
	}
	for _, f := range splitNUL(string(out)) {
		switch classifyHit(f) {
		case "prose":
			br.Prose = append(br.Prose, f)
		case "data":
			br.Data = append(br.Data, f)
		default:
			br.Code = append(br.Code, f)
		}
	}
	sort.Strings(br.Code)
	sort.Strings(br.Prose)
	sort.Strings(br.Data)
	return br, nil
}

// isExitCode reports whether err is a git exit with the given status.
func isExitCode(err error, code int) bool {
	var ee *exec.ExitError
	for e := err; e != nil; {
		if x, ok := e.(*exec.ExitError); ok {
			ee = x
			break
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			break
		}
		e = u.Unwrap()
	}
	return ee != nil && ee.ExitCode() == code
}
