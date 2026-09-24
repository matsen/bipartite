package prov

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitFiles(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	for name, body := range files {
		if body == "" {
			git(t, dir, "rm", "-q", name)
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, dir, "add", name)
	}
	git(t, dir, "commit", "-q", "-m", "c")
	return git(t, dir, "rev-parse", "HEAD")
}

// codeRepo builds the fixture code repo and returns its dir and the ledger
// placeholders: C1, C2, C3 on origin/main, and D, a commit no ref contains.
func codeRepo(t *testing.T) (string, map[string]string) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	shas := map[string]string{}
	shas["C1"] = commitFiles(t, dir, map[string]string{
		"results.json":    `{"corpus": {"pcps_changed": 0.020769}}`,
		"nextflow.config": "joint = true\r\nother = 1\r\n",
		"old.txt":         "so it says\n",
	})
	shas["C2"] = commitFiles(t, dir, map[string]string{
		"results.json":    "{\n  \"corpus\": {\n    \"pcps_changed\": 2.0769e-2\n  }\n}\n",
		"nextflow.config": "joint = false\nother = 1\n",
		"old.txt":         "",
		"new.txt":         "so it says\n",
	})
	shas["C4"] = commitFiles(t, dir, map[string]string{"nextflow.config": "nothing here\nother = 1\n"})
	shas["C3"] = commitFiles(t, dir, map[string]string{
		"results.json":        `{"corpus": {"pcps_changed": 0.021}}`,
		"facts_pipeline.json": `{"pipeline_sha": "` + shas["C1"] + `"}`,
		"facts_launches.json": `{"launches": [{"commit": "` + shas["C1"] + `", "stages": ["ASR"]}, {"commit": "` + shas["C2"][:7] + `"}, {"commit": "` + shas["C4"] + `", "stages": []}, {"run_name": "lost_launch", "commit": null, "stages": ["ASR"]}]}`,
	})
	shas["D"] = commitFiles(t, dir, map[string]string{"nextflow.config": "joint = true\n"})
	git(t, dir, "reset", "-q", "--hard", shas["C3"])
	git(t, dir, "update-ref", "refs/remotes/origin/main", shas["C3"])
	return dir, shas
}

// paperDir copies testdata/prov/paper into a git repo with the ledger's
// placeholders filled in, and commits it.
func paperDir(t *testing.T, shas map[string]string) string {
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "prov", "paper")
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(data)
		for k, v := range shas {
			s = strings.ReplaceAll(s, "@"+k+"@", v)
		}
		dst := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, []byte(s), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "paper")
	return dir
}

func runCheck(t *testing.T, paper, code string) []Finding {
	t.Helper()
	fs, err := Check(Options{
		PaperDir: paper,
		Main:     "main.tex",
		Ledger:   "provenance.yaml",
		Resolve:  func(string) (string, error) { return code, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

// byID groups findings as id → "level: message" lines.
func byID(fs []Finding) map[string][]string {
	m := map[string][]string{}
	for _, f := range fs {
		m[f.ID] = append(m[f.ID], f.Level+": "+f.Message)
	}
	return m
}

func has(t *testing.T, m map[string][]string, id, level, substr string) {
	t.Helper()
	for _, s := range m[id] {
		if strings.HasPrefix(s, level+": ") && strings.Contains(s, substr) {
			return
		}
	}
	t.Errorf("%s: no %s containing %q; got %q", id, level, substr, m[id])
}

func hasNo(t *testing.T, m map[string][]string, id, level string) {
	t.Helper()
	for _, s := range m[id] {
		if strings.HasPrefix(s, level+": ") {
			t.Errorf("%s: unexpected %s", id, s)
		}
	}
}

func TestCheck(t *testing.T) {
	code, shas := codeRepo(t)
	fs := runCheck(t, paperDir(t, shas), code)
	m := byID(fs)

	// A JSON file re-rendered with the same value passes; a changed value errors.
	hasNo(t, m, "same1", LevelError)
	hasNo(t, m, "same2", LevelError)
	has(t, m, "changed", LevelError, "pcps_changed = 0.021")
	// The file at origin/main differs from the pinned one.
	has(t, m, "same1", LevelReview, "origin/main differs")
	for _, s := range m["changed"] {
		if strings.Contains(s, "origin/main") {
			t.Errorf("changed is pinned at origin/main: %s", s)
		}
	}

	// A path renamed at the pin errors.
	has(t, m, "renamed", LevelError, "old.txt missing")

	// D is unreachable, pinned by unreach and a launch of run multi: one error.
	var unreachable []Finding
	for _, f := range fs {
		if strings.Contains(f.Message, "unreachable") {
			unreachable = append(unreachable, f)
		}
	}
	if len(unreachable) != 1 || unreachable[0].ID != "run:multi" || !strings.Contains(unreachable[0].Message, "used by 2 entries") {
		t.Errorf("want one unreachable error for run:multi used by 2 entries, got %v", unreachable)
	}

	// Three launches: holds at C1, absent at C2, D skipped (reported above).
	has(t, m, "multi", LevelError, "pattern absent from nextflow.config at "+shas["C2"][:8])
	has(t, m, "multi", LevelError, "holds at "+shas["C1"][:8])
	// A launch with a null commit is one info, not an error.
	has(t, m, "run:multi", LevelInfo, "lost_launch has no attested commit")
	// A launch that submitted no tasks (stages: []) is not checked.
	for _, s := range m["multi"] {
		if strings.Contains(s, shas["C4"][:8]) {
			t.Errorf("multi checked at the no-task launch C4: %s", s)
		}
	}

	// A CRLF file matches an LF pattern.
	hasNo(t, m, "crlf", LevelError)
	// It holds at its pin but not at origin/main: one info, not an error.
	has(t, m, "crlf", LevelInfo, "no longer holds at origin/main")
	for _, s := range m["pipe"] {
		if strings.Contains(s, "origin/main") {
			t.Errorf("pipe still holds at origin/main: %s", s)
		}
	}

	// pipeline_sha fallback: one info for the run, and the claim still checks.
	if got := len(m["run:pipe"]); got != 1 {
		t.Errorf("run:pipe: want 1 finding, got %q", m["run:pipe"])
	}
	has(t, m, "run:pipe", LevelInfo, "HEAD at collection")
	hasNo(t, m, "pipe", LevelError)
	for _, s := range m["run:multi"] { // reads launches[], not pipeline_sha
		if strings.Contains(s, "HEAD at collection") {
			t.Errorf("run:multi fell back to pipeline_sha: %s", s)
		}
	}

	// Tags: no entry, no leading space, the \input file is scanned.
	has(t, m, "nosuch", LevelError, "no ledger entry")
	has(t, m, "glued", LevelError, "no space before")
	has(t, m, "spare", LevelInfo, "unsourced")
	has(t, m, "spare", LevelInfo, "no tag uses")

	// With no audited_through, every tagged sentence is review.
	for _, id := range []string{"same1", "same2", "changed", "crlf", "multi", "renamed", "unreach", "pipe"} {
		has(t, m, id, LevelReview, "not yet audited")
	}
}

func TestDerived(t *testing.T) {
	code, shas := codeRepo(t)
	m := byID(runCheck(t, paperDir(t, shas), code))

	has(t, m, "d_changed", LevelReview, "input changed: changed")
	hasNo(t, m, "d_clean", LevelReview)
	has(t, m, "d_trans", LevelReview, "input changed: d_changed")
	has(t, m, "d_unreach", LevelReview, "input changed: unreach")
	has(t, m, "d_missing", LevelError, `"nosuch_entry" is not a ledger entry`)
	cycles := 0
	for _, id := range []string{"cyc_a", "cyc_b"} {
		for _, s := range m[id] {
			if strings.Contains(s, "cycle") {
				cycles++
			}
		}
	}
	if cycles != 1 {
		t.Errorf("want one cycle error, got %d: %q %q", cycles, m["cyc_a"], m["cyc_b"])
	}
}

func TestChangedSentenceIsReview(t *testing.T) {
	code, shas := codeRepo(t)
	paper := paperDir(t, shas)
	ledger := filepath.Join(paper, "provenance.yaml")
	data, _ := os.ReadFile(ledger)
	head := git(t, paper, "rev-parse", "HEAD")
	if err := os.WriteFile(ledger, []byte("audited_through: "+head+"\n"+string(data)), 0o644); err != nil {
		t.Fatal(err)
	}
	before := runCheck(t, paper, code)

	methods := filepath.Join(paper, "sections", "methods.tex")
	data, _ = os.ReadFile(methods)
	edited := strings.Replace(string(data), "at every launch", "at each launch", 1)
	if err := os.WriteFile(methods, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	after := runCheck(t, paper, code)

	reviews := map[string]string{}
	for _, f := range after {
		if f.Level == LevelReview && strings.Contains(f.Message, "since") {
			reviews[f.ID] = f.Message
		}
	}
	// The edited line's tag, then the other tags of its paragraph.
	want := map[string]string{"multi": "tagged sentence changed", "renamed": "in its paragraph", "unreach": "in its paragraph", "pipe": "in its paragraph"}
	if len(reviews) != len(want) {
		t.Errorf("want reviews for %v, got %v", want, reviews)
	}
	for id, sub := range want {
		if !strings.Contains(reviews[id], sub) {
			t.Errorf("%s: want review containing %q, got %q", id, sub, reviews[id])
		}
	}
	count := func(fs []Finding) (n int) {
		for _, f := range fs {
			if f.Level == LevelError {
				n++
			}
		}
		return n
	}
	if count(before) != count(after) {
		t.Errorf("editing a sentence changed the error count: %d → %d", count(before), count(after))
	}
}

func TestLedgerEditIsReview(t *testing.T) {
	code, shas := codeRepo(t)
	paper := paperDir(t, shas)
	ledger := filepath.Join(paper, "provenance.yaml")
	data, _ := os.ReadFile(ledger)
	head := git(t, paper, "rev-parse", "HEAD")
	edited := strings.Replace(string(data), `scope: "tree edges of the sample"`, `scope: "tree edges of the pilot"`, 1)
	if err := os.WriteFile(ledger, []byte("audited_through: "+head+"\n"+edited), 0o644); err != nil {
		t.Fatal(err)
	}
	m := byID(runCheck(t, paper, code))
	has(t, m, "same1", LevelReview, "ledger entry changed")
	for _, id := range []string{"same2", "crlf"} {
		for _, s := range m[id] {
			if strings.Contains(s, "since") {
				t.Errorf("%s reviewed with no change: %s", id, s)
			}
		}
	}
}

func TestFromCountsAsUse(t *testing.T) {
	code, shas := codeRepo(t)
	m := byID(runCheck(t, paperDir(t, shas), code))
	// cyc_a is untagged, but cyc_b uses it as an input.
	for _, s := range m["cyc_a"] {
		if strings.Contains(s, "no tag uses") {
			t.Errorf("cyc_a used by from: but reported unused: %s", s)
		}
	}
	has(t, m, "d_trans", LevelInfo, "no tag uses") // nothing tags or uses it
}

func TestDuplicateID(t *testing.T) {
	p := filepath.Join(t.TempDir(), "provenance.yaml")
	os.WriteFile(p, []byte("entries:\n  a: {unsourced: x}\n  a: {unsourced: y}\n"), 0o644)
	if _, err := LoadLedger(p); err == nil {
		t.Error("duplicate id loaded without error")
	}
}

func TestScanPaper(t *testing.T) {
	p, err := ScanPaper(filepath.Join("..", "..", "testdata", "prov", "paper"), "main.tex")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, tag := range p.Tags {
		ids[tag.ID] = true
	}
	if !ids["multi"] || ids["glued"] || len(p.Tags) != 9 {
		t.Errorf("tags = %v", p.Tags)
	}
	if len(p.Malformed) != 1 || p.Malformed[0].ID != "glued" {
		t.Errorf("malformed = %v", p.Malformed)
	}
	if p.Tags[0].Sentence != "Rerooting changed at least 2.1\\% of tree edges. " {
		t.Errorf("sentence = %q", p.Tags[0].Sentence)
	}
}
