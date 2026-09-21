package gitx

import (
	"strings"
	"testing"
)

// TestSymbol_SplitsCodeFromProse fails against an implementation that
// reports one total. Measured on matsengrp/phyz: "26 consumers" was 17 code
// files plus 9 READMEs stating the value in prose — the prose hits go stale
// on the edit exactly like the code hits, but they are different work, and
// a single number tells a worker one kind where the answer is two.
func TestSymbol_SplitsCodeFromProse(t *testing.T) {
	dir := initRepo(t)
	writeAt(t, dir, "src/common.py", "MATCHED_PARAMS = [1, 2]\n")
	writeAt(t, dir, "src/user.py", "from common import MATCHED_PARAMS\n")
	writeAt(t, dir, "docs/README.md", "MATCHED_PARAMS is the list of matched params.\n")
	writeAt(t, dir, "results/sweep.tsv", "run\tMATCHED_PARAMS\n1\t2\n")
	writeAt(t, dir, "src/unrelated.py", "OTHER = 3\n")
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "--quiet", "-m", "add consumers")

	br, err := Symbol(dir, "MATCHED_PARAMS", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(br.Code, " "); got != "src/common.py src/user.py" {
		t.Errorf("code hits = %q, want the two .py files", got)
	}
	if got := strings.Join(br.Prose, " "); got != "docs/README.md" {
		t.Errorf("prose hits = %q, want docs/README.md", got)
	}
	if got := strings.Join(br.Data, " "); got != "results/sweep.tsv" {
		t.Errorf("data hits = %q, want results/sweep.tsv — a committed artifact is a third kind, not a code hit", got)
	}
	if len(br.Code) == br.Total() {
		t.Error("the code bucket equals the total, so the split is not happening")
	}
	// The commit is part of the figure: the same grep 14 hours later
	// returned 39 consumers where it had returned 26.
	if br.Commit == "" {
		t.Error("BlastRadius.Commit is empty; a count without its commit is not re-derivable")
	}
}

// TestSymbol_PathspecLimitsPopulation covers the --path form. The `-e`
// pattern form exists so a pathspec list cannot be misparsed as part of the
// pattern, which would silently change the population being counted.
func TestSymbol_PathspecLimitsPopulation(t *testing.T) {
	dir := initRepo(t)
	writeAt(t, dir, "a/x.py", "SYM = 1\n")
	writeAt(t, dir, "b/y.py", "SYM = 2\n")
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "--quiet", "-m", "two dirs")

	br, err := Symbol(dir, "SYM", []string{"a/"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(br.Code, " "); got != "a/x.py" {
		t.Errorf("code hits = %q, want only a/x.py", got)
	}
}

// TestSymbol_NoMatchIsZeroNotAFailure keeps two verdicts apart: `git grep`
// exits 1 when nothing matched, and "zero consumers" is a real answer
// rather than "could not check".
func TestSymbol_NoMatchIsZeroNotAFailure(t *testing.T) {
	dir := initRepo(t)
	br, err := Symbol(dir, "NOT_PRESENT_ANYWHERE", nil)
	if err != nil {
		t.Fatalf("no-match grep returned an error: %v", err)
	}
	if br.Total() != 0 {
		t.Errorf("total = %d, want 0", br.Total())
	}
}

// TestSymbol_EmptySymbolRefused: an empty pattern matches every file, so
// the blast radius of "" would be the whole repo — the same empty-argument
// fail-open the pool check's scope resolution exists to close.
func TestSymbol_EmptySymbolRefused(t *testing.T) {
	dir := initRepo(t)
	for _, sym := range []string{"", "  "} {
		if _, err := Symbol(dir, sym, nil); err == nil {
			t.Errorf("Symbol(%q) succeeded; an empty symbol matches everything", sym)
		}
	}
}

func TestClassifyHit(t *testing.T) {
	cases := map[string]string{
		"src/a.go":            "code",
		"build.zig":           "code",
		"Makefile":            "code",
		"docs/guide.md":       "prose",
		"README":              "prose",
		"notes.txt":           "prose",
		"results/out.tsv":     "data",
		"logs/run.log":        "data",
		"experiments/x.jsonl": "data",
	}
	for path, want := range cases {
		if got := classifyHit(path); got != want {
			t.Errorf("classifyHit(%q) = %q, want %q", path, got, want)
		}
	}
}
