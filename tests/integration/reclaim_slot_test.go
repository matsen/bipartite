package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// reclaimFixture is a slot clone on branch feat, pushed to a bare origin,
// with EPIC state files, inside a clone root.
type reclaimFixture struct {
	root, clone, origin, bin string
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// newReclaimFixture builds the slot, plus a bin dir holding a fake gh and a
// fake tmux. gh answers the ceremony query with ceremony, the metadata query
// with a PR from feat to main closing one issue, and issue view with
// issueState. tmux reports one pane in the clone whose cursor is at cursor
// ("" for no pane) and records kill-window calls.
func newReclaimFixture(t *testing.T, ceremony, issueState, cursor string) reclaimFixture {
	t.Helper()
	f := reclaimFixture{root: t.TempDir(), bin: t.TempDir()}
	f.origin = filepath.Join(f.root, "origin.git")
	f.clone = filepath.Join(f.root, "slot")
	runGit(t, f.root, "init", "-q", "--bare", f.origin)
	runGit(t, f.root, "clone", "-q", f.origin, f.clone)
	writeFile(t, filepath.Join(f.clone, ".gitignore"), ".epic-*\n.claude/\n", 0644)
	runGit(t, f.clone, "add", ".gitignore")
	runGit(t, f.clone, "commit", "-qm", "base")
	runGit(t, f.clone, "push", "-q", "origin", "HEAD:main")
	runGit(t, f.clone, "checkout", "-qb", "feat")
	writeFile(t, filepath.Join(f.clone, "x"), "x\n", 0644)
	runGit(t, f.clone, "add", "x")
	runGit(t, f.clone, "commit", "-qm", "feat")
	runGit(t, f.clone, "push", "-q", "origin", "feat", "feat:refs/pull/7/head")
	writeFile(t, filepath.Join(f.clone, ".epic-status.json"), `{"issue": 3, "phase": "quality-gate"}`, 0644)
	writeFile(t, filepath.Join(f.clone, ".epic-worklog.md"), "worklog\n", 0644)

	writeFile(t, filepath.Join(f.bin, "ceremony.json"), ceremony, 0644)
	writeFile(t, filepath.Join(f.bin, "meta.json"),
		`{"baseRefName":"main","headRefName":"feat","headRefOid":"`+runGit(t, f.clone, "rev-parse", "HEAD")+
			`","closingIssuesReferences":[{"number":3,"url":"https://github.com/owner/repo/issues/3"}]}`, 0644)
	gh := "#!/bin/sh\ncase \"$*\" in\n" +
		"*state,comments*) cat " + shellQuote(filepath.Join(f.bin, "ceremony.json")) + " ;;\n" +
		"*baseRefName*) cat " + shellQuote(filepath.Join(f.bin, "meta.json")) + " ;;\n" +
		"'issue view https://github.com/owner/repo/issues/3 '*) echo " + issueState + " ;;\n" +
		"*) echo \"unexpected gh $*\" >&2; exit 1 ;;\nesac\n"
	writeFile(t, filepath.Join(f.bin, "gh"), gh, 0755)
	pane := ""
	if cursor != "" {
		pane = "echo '%9 " + f.clone + "/sub'"
	}
	tmux := "#!/bin/sh\ncase \"$1\" in\n" +
		"list-panes) " + pane + " ;;\n" +
		"display-message) echo " + shellQuote(cursor) + " ;;\n" +
		"kill-window) echo \"$*\" >> " + shellQuote(filepath.Join(f.bin, "killed")) + " ;;\n" +
		"esac\n"
	writeFile(t, filepath.Join(f.bin, "tmux"), tmux, 0755)
	return f
}

func (f reclaimFixture) run(t *testing.T, shell string) (string, int) {
	t.Helper()
	return f.runAs(t, shell, "idle")
}

func (f reclaimFixture) runAs(t *testing.T, shell, agentState string) (string, int) {
	t.Helper()
	cmd := exec.Command(shell, "-c", spawnIntentCall(t, "reclaim_slot", f.clone, "owner/repo", "7", agentState))
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+f.bin+":"+os.Getenv("PATH"),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.Output()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %s: %v", shell, err)
	}
	return strings.TrimSpace(string(out)), code
}

func (f reclaimFixture) killed() string {
	b, _ := os.ReadFile(filepath.Join(f.bin, "killed"))
	return strings.TrimSpace(string(b))
}

// assertUntouched checks a HOLD left the slot exactly as it was.
func (f reclaimFixture) assertUntouched(t *testing.T) {
	t.Helper()
	if br := runGit(t, f.clone, "branch", "--show-current"); br != "feat" {
		t.Errorf("branch = %q, want feat", br)
	}
	if _, err := os.Stat(filepath.Join(f.clone, ".epic-status.json")); err != nil {
		t.Errorf("status file gone after a hold: %v", err)
	}
	if k := f.killed(); k != "" {
		t.Errorf("window killed after a hold: %q", k)
	}
}

func TestReclaimSlot(t *testing.T) {
	ran := prJSON(t, "MERGED", []int{3}, terminalLead)
	shells := []string{"bash"}
	if _, err := exec.LookPath("zsh"); err == nil {
		shells = append(shells, "zsh")
	}
	for _, shell := range shells {
		t.Run(shell+"/reclaimed", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "2")
			got, code := f.run(t, shell)
			if code != 0 || !strings.HasPrefix(got, "RECLAIMED "+f.clone+" (feat -> main ") {
				t.Fatalf("got %q (exit %d)", got, code)
			}
			if k := f.killed(); k != "kill-window -t %9" {
				t.Errorf("kill-window calls = %q", k)
			}
			if br := runGit(t, f.clone, "branch", "--show-current"); br != "main" {
				t.Errorf("branch = %q, want main", br)
			}
			if b := runGit(t, f.clone, "branch", "--list", "feat"); b != "" {
				t.Errorf("local feat not deleted: %q", b)
			}
			if b := runGit(t, f.root, "--git-dir", f.origin, "branch", "--list", "feat"); b != "" {
				t.Errorf("remote feat not deleted: %q", b)
			}
			for _, name := range []string{".epic-status.json", ".epic-worklog.md"} {
				if _, err := os.Stat(filepath.Join(f.clone, name)); !os.IsNotExist(err) {
					t.Errorf("%s not deleted", name)
				}
			}
			preserved, _ := filepath.Glob(filepath.Join(f.root, ".preserved", "3-*", "i3-slot.worklog.md"))
			if len(preserved) != 1 {
				t.Errorf("preserved worklogs = %v, want one", preserved)
			}
		})
		holds := []struct {
			name, ceremony, issue, cursor, want string
		}{
			{"ceremony owed", prJSON(t, "MERGED", []int{3}, nonTerminalLead), "CLOSED", "", "CEREMONY OWED #7 <clone>"},
			{"issue still open", ran, "OPEN", "", "HOLD <clone>: https://github.com/owner/repo/issues/3 is OPEN"},
			{"typed input at the prompt", ran, "CLOSED", "9", "HOLD <clone>: composer cursor at 9, typed input?"},
		}
		for _, h := range holds {
			t.Run(shell+"/"+h.name, func(t *testing.T) {
				f := newReclaimFixture(t, h.ceremony, h.issue, h.cursor)
				got, code := f.run(t, shell)
				if want := strings.ReplaceAll(h.want, "<clone>", f.clone); got != want || code != 1 {
					t.Errorf("got %q (exit %d), want %q (exit 1)", got, code, want)
				}
				f.assertUntouched(t)
			})
		}
		t.Run(shell+"/local commit the merge never saw", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "")
			writeFile(t, filepath.Join(f.clone, "y"), "y\n", 0644)
			runGit(t, f.clone, "add", "y")
			runGit(t, f.clone, "commit", "-qm", "unpushed")
			got, code := f.run(t, shell)
			if want := "HOLD " + f.clone + ": local feat has commits the merged head lacks"; got != want || code != 1 {
				t.Errorf("got %q (exit %d), want %q", got, code, want)
			}
			f.assertUntouched(t)
		})
		t.Run(shell+"/busy session", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "2")
			got, code := f.runAs(t, shell, "busy")
			if want := "HOLD " + f.clone + ": session state is 'busy', not idle"; got != want || code != 1 {
				t.Errorf("got %q (exit %d), want %q", got, code, want)
			}
			f.assertUntouched(t)
		})
		// bip-pr-land's moved-base default rebases the PR elsewhere and
		// force-pushes, so the merged head is a new SHA with the same patch.
		t.Run(shell+"/head rebased elsewhere", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "")
			other := filepath.Join(f.root, "other")
			runGit(t, f.root, "clone", "-q", "-b", "main", f.origin, other)
			writeFile(t, filepath.Join(other, "z"), "z\n", 0644)
			runGit(t, other, "add", "z")
			runGit(t, other, "commit", "-qm", "moved base")
			runGit(t, other, "push", "-q", "origin", "main")
			runGit(t, other, "fetch", "-q", "origin", "feat")
			runGit(t, other, "checkout", "-q", "-b", "feat", "FETCH_HEAD")
			runGit(t, other, "rebase", "-q", "main")
			runGit(t, other, "push", "-q", "-f", "origin", "feat", "feat:refs/pull/7/head")
			writeFile(t, filepath.Join(f.bin, "meta.json"),
				`{"baseRefName":"main","headRefName":"feat","headRefOid":"`+runGit(t, other, "rev-parse", "HEAD")+
					`","closingIssuesReferences":[{"number":3,"url":"https://github.com/owner/repo/issues/3"}]}`, 0644)
			got, code := f.run(t, shell)
			if code != 0 || !strings.HasPrefix(got, "RECLAIMED ") {
				t.Errorf("got %q (exit %d), want RECLAIMED", got, code)
			}
		})
		t.Run(shell+"/PR closes no issue", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "")
			writeFile(t, filepath.Join(f.bin, "meta.json"), `{"baseRefName":"main","headRefName":"feat","headRefOid":"x","closingIssuesReferences":[]}`, 0644)
			got, code := f.run(t, shell)
			if want := "HOLD " + f.clone + ": PR #7 closes no issue; reclaim by hand"; got != want || code != 1 {
				t.Errorf("got %q (exit %d), want %q", got, code, want)
			}
			f.assertUntouched(t)
		})
		t.Run(shell+"/uncommitted changes", func(t *testing.T) {
			f := newReclaimFixture(t, ran, "CLOSED", "")
			writeFile(t, filepath.Join(f.clone, "x"), "edited\n", 0644)
			got, code := f.run(t, shell)
			if want := "HOLD " + f.clone + ": uncommitted changes in x"; got != want || code != 1 {
				t.Errorf("got %q (exit %d), want %q", got, code, want)
			}
			f.assertUntouched(t)
		})
	}
}
