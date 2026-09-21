package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matsen/bipartite/internal/gitx"
)

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, epicConfigName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestResolveCollideScope_NeverDefaults is the fix this command exists for.
// `ROOT="${1:-$HOME/re/pz}"` substitutes on unset OR EMPTY, so an
// unparseable .epic-config.json made an explicitly-passed root evaporate
// into another repository's clone pool — and returned a populated,
// well-formed, entirely plausible report about the wrong fleet. Every case
// below must fail to resolve rather than resolve to something else.
func TestResolveCollideScope_NeverDefaults(t *testing.T) {
	t.Run("unparseable config", func(t *testing.T) {
		cwd := t.TempDir()
		writeConfig(t, cwd, "{ this is not json")
		scope, _, err := resolveCollideScope("", false, cwd)
		if err == nil {
			t.Fatalf("resolved %q from an unparseable config; want could-not-check", scope.Root)
		}
	})

	t.Run("no config at all", func(t *testing.T) {
		cwd := t.TempDir()
		if scope, _, err := resolveCollideScope("", false, cwd); err == nil {
			t.Fatalf("resolved %q with no config present", scope.Root)
		}
	})

	t.Run("config with no clone_root", func(t *testing.T) {
		cwd := t.TempDir()
		writeConfig(t, cwd, `{"clone_names": ["alpha"]}`)
		if scope, _, err := resolveCollideScope("", false, cwd); err == nil {
			t.Fatalf("resolved %q from a config with no clone_root", scope.Root)
		}
	})

	t.Run("explicit empty root", func(t *testing.T) {
		cwd := t.TempDir()
		// A config IS present and valid here: an empty explicit argument
		// must not silently fall through to it either.
		writeConfig(t, cwd, `{"clone_root": "/somewhere/else", "clone_names": ["alpha"]}`)
		for _, given := range []string{"", "   "} {
			scope, _, err := resolveCollideScope(given, true, cwd)
			if err == nil {
				t.Errorf("--root %q resolved to %q; an empty root is not a root", given, scope.Root)
			}
		}
	})

	t.Run("config clone_root is used", func(t *testing.T) {
		cwd := t.TempDir()
		pool := filepath.Join(cwd, "pool")
		writeConfig(t, cwd, `{"clone_root": "`+pool+`", "clone_names": ["alpha"]}`)
		scope, source, err := resolveCollideScope("", false, cwd)
		if err != nil {
			t.Fatal(err)
		}
		if scope.Root != pool {
			t.Errorf("root = %q, want %q", scope.Root, pool)
		}
		if !strings.Contains(source, epicConfigName) {
			t.Errorf("source = %q, want it to name %s", source, epicConfigName)
		}
		// clone_names is the authoritative slot list, and it must reach the
		// scope: every directory under the root is the WRONG population.
		if strings.Join(scope.Names, " ") != "alpha" {
			t.Errorf("scope.Names = %v, want the config's clone_names", scope.Names)
		}
		if !strings.Contains(scope.Rule(), "clone_names") {
			t.Errorf("scope.Rule() = %q, want it to name the universe applied", scope.Rule())
		}
	})

	t.Run("explicit root wins", func(t *testing.T) {
		cwd := t.TempDir()
		writeConfig(t, cwd, `{"clone_root": "/from/config", "clone_names": ["alpha"]}`)
		scope, source, err := resolveCollideScope("/from/flag", true, cwd)
		if err != nil {
			t.Fatal(err)
		}
		if scope.Root != "/from/flag" || source != "--root" {
			t.Errorf("root = %q source = %q, want /from/flag from --root", scope.Root, source)
		}
		// An explicit root may not be the pool this config describes, so
		// carrying its clone_names across would be a worse error than
		// having none.
		if len(scope.Names) != 0 {
			t.Errorf("scope.Names = %v, want empty for an explicit --root", scope.Names)
		}
	})
}

// TestCollideMain_PrintsScopeAndExitCode pins two contract points: the
// resolved scope is the first line of output on every run, and the verdict
// reaches the caller as an exit code rather than as a returned error
// (main.go maps every RunE error to 1, so a three-valued verdict cannot
// travel that way).
func TestCollideMain_PrintsScopeAndExitCode(t *testing.T) {
	t.Run("unresolvable scope exits uncheckable", func(t *testing.T) {
		cwd := t.TempDir()
		writeConfig(t, cwd, "{ not json")
		var buf bytes.Buffer
		code := collideMain(&buf, "", false, cwd)
		if code != ExitCollisionUncheckable {
			t.Errorf("exit code = %d, want %d", code, ExitCollisionUncheckable)
		}
		out := buf.String()
		if !strings.HasPrefix(out, "scope: UNRESOLVED") {
			t.Errorf("output does not lead with the unresolved scope:\n%s", out)
		}
		if strings.Contains(out, "clear") {
			t.Errorf("output contains the word clear:\n%s", out)
		}
	})

	t.Run("empty pool prints scope and nothing to check", func(t *testing.T) {
		cwd := t.TempDir()
		pool := filepath.Join(cwd, "pool")
		if err := os.MkdirAll(filepath.Join(pool, ".spawn-prompts"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, cwd, `{"clone_root": "`+pool+`", "clone_names": ["alpha"]}`)
		var buf bytes.Buffer
		code := collideMain(&buf, "", false, cwd)
		if code != ExitSuccess {
			t.Errorf("exit code = %d, want %d", code, ExitSuccess)
		}
		out := buf.String()
		if !strings.HasPrefix(out, "scope: clone_root="+pool) {
			t.Errorf("first line is not the resolved scope:\n%s", out)
		}
		if !strings.Contains(out, "nothing to check") {
			t.Errorf("output does not say nothing to check:\n%s", out)
		}
		// "nothing to check" and "clear" are different claims, and a pool
		// root holding only .spawn-prompts/ is the first one.
		if strings.Contains(out, "clear") {
			t.Errorf("an empty pool reported as clear:\n%s", out)
		}
	})
}

// TestRenderCollideReport_UnreadableCloneNeverPrintsZero: `files=0` reads as
// "nothing here" on the one line that exists to say otherwise, and a
// numeric sentinel would silently pass a `> 0` test. See
// skills/lib/hazards/counting-a-failed-command.md.
func TestRenderCollideReport_UnreadableCloneNeverPrintsZero(t *testing.T) {
	rep := &gitx.Report{
		Root: "/pool",
		Pool: []string{"/pool/alpha", "/pool/broken"},
		Live: []gitx.CloneTouch{
			{Name: "alpha", Branch: "a-work", Files: []string{"CLAUDE.md"}},
			{Name: "broken", Unreadable: "git rev-parse failed but .git exists"},
		},
		Collisions: []gitx.Collision{{File: "CLAUDE.md", Clones: []string{"alpha", "bravo"}}},
		Problems:   []string{"broken: cannot examine clone — git rev-parse failed but .git exists"},
	}
	var buf bytes.Buffer
	renderCollideReport(&buf, rep)
	out := buf.String()

	var brokenLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "broken") && strings.Contains(line, "files=") {
			brokenLine = line
		}
	}
	if brokenLine == "" {
		t.Fatalf("no per-clone line for the unreadable clone:\n%s", out)
	}
	if !strings.Contains(brokenLine, "files="+gitx.UnreadableCount) {
		t.Errorf("unreadable clone line = %q, want files=%s", brokenLine, gitx.UnreadableCount)
	}
	if strings.Contains(brokenLine, "=0") {
		t.Errorf("unreadable clone line = %q, which reads as a real count of zero", brokenLine)
	}
	if !strings.Contains(brokenLine, "rev-parse") {
		t.Errorf("unreadable clone line = %q, want the reason named", brokenLine)
	}
	if !strings.Contains(out, "verdict: could not check") {
		t.Errorf("verdict line missing or wrong; uncheckable must dominate the found collision:\n%s", out)
	}
	// The pairwise section's denominator, so a bare "none" is never the
	// whole answer.
	if !strings.Contains(out, "live clones=1, pairs=0") {
		t.Errorf("output does not print the pairwise denominator:\n%s", out)
	}
}

// TestRenderBlastRadius_CarriesCommitAndHedge: a blast-radius count is a
// measurement with a date, not a property of the symbol, and `git grep`
// cannot see files that import the defining module without naming it.
func TestRenderBlastRadius_CarriesCommitAndHedge(t *testing.T) {
	br := &gitx.BlastRadius{
		Symbol: "MATCHED_PARAMS",
		Repo:   "/repo",
		Commit: "47a35e7c",
		Code:   []string{"a.py", "b.py"},
		Prose:  []string{"README.md"},
	}
	var buf bytes.Buffer
	taken := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	renderBlastRadius(&buf, br, taken)
	out := buf.String()

	for _, want := range []string{
		"commit: 47a35e7c",
		"2026-09-21T10:00:00Z",
		"code hits: 2 files",
		"prose hits: 1 files",
		"indirect consumers: NOT MEASURED",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// The sum must not be the headline: "26 consumers" tells a worker one
	// kind of work where "17 code plus 9 prose" tells it two.
	if strings.Contains(out, "3 files naming") || strings.Contains(out, "total: 3") {
		t.Errorf("output reports a combined total:\n%s", out)
	}
}
