package integration

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// runPostMergeCeremony sources spawn-intent.sh under shell and calls
// post_merge_ceremony with a fake `gh` first on PATH. The fake records its
// arguments, prints ghOutput and exits ghExit, standing in for
// `gh pr view --json ...`. Returns trimmed stdout, the function's exit
// code, and the arguments gh was called with.
func runPostMergeCeremony(t *testing.T, shell, cloneDir, ghOutput string, ghExit int) (string, int, string) {
	t.Helper()
	binDir := t.TempDir()
	payload := filepath.Join(binDir, "payload.json")
	if err := os.WriteFile(payload, []byte(ghOutput), 0644); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(binDir, "args")
	// The stderr line stands in for gh's upgrade and auth notices, which
	// must not reach the JSON parser on a successful call.
	fake := "#!/bin/sh\necho \"$@\" > " + shellQuote(argsFile) + "\necho 'A new release of gh is available' >&2\ncat " + shellQuote(payload) + "\nexit " + strconv.Itoa(ghExit) + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(fake), 0755); err != nil {
		t.Fatal(err)
	}
	script := "source " + shellQuote(spawnIntentScriptPath(t)) + "\n" +
		"post_merge_ceremony " + shellQuote(cloneDir) + " owner/repo 7"
	cmd := exec.Command(shell, "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	out, err := cmd.Output()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %s: %v", shell, err)
	}
	args, _ := os.ReadFile(argsFile)
	return strings.TrimSpace(string(out)), code, strings.TrimSpace(string(args))
}

// cloneWithStatus returns a temp clone dir, with an .epic-status.json in it
// when withStatus is set.
func cloneWithStatus(t *testing.T, withStatus bool) string {
	t.Helper()
	dir := t.TempDir()
	if withStatus {
		body := `{"issue": 3, "phase": "quality-gate", "stop_reason": "awaiting-human-merge"}`
		if err := os.WriteFile(filepath.Join(dir, ".epic-status.json"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const (
	// A clean-gate comment from a repo where humans merge. It names
	// `completed` on its Category line, but not as the first word, and it
	// carries the same header a terminal comment does. The quotes and
	// backslash exercise JSON escaping on the way through the fake gh.
	nonTerminalLead = "🤖 **Issue Lead** (iteration 1)\n\n**Category**: `quality-gate` — gates clean. Not `completed`: \"humans merge\" here (C:\\no).\n"
	terminalLead    = "🤖 **Issue Lead** (iteration 3)\n\n**Category**: completed\n**Action**: none\n"
	// phyz#2920 wrote its terminal Category this way.
	terminalLeadBackticked = "🤖 **Issue Lead** (iteration 1) — terminal evaluation, post-land.\n\n**Category**: `completed`\n"
	// phyz#2909 wrote this one; text after `completed` is allowed.
	terminalLeadTrailing = "🤖 **Issue Lead** (iteration 1)\n\n**Category**: completed — post-landing review.\n"
	// The label's bold can also close after the colon.
	terminalLeadColonInBold = "🤖 **Issue Lead** (iteration 2)\n\n**Category:** completed\n"
	// A review comment quoting the marker is not a lead comment.
	quotesMarker = "Nit: the guard keys on `**Category**: completed`, see spawn-intent.sh."
	// phyz#2817: a conductor's hand-posted note saying /bip-pr-land did NOT run.
	handPostedPreserved = "🤖 **EPIC worklog preserved** — `/x/.preserved/2816/`\n\nPosted by the conductor, not by `/bip-pr-land`."
	prLandComment       = "🤖 EPIC worklog preserved to `/x/.preserved/3-2026-09-22` (issue #2216)."
)

type ghComment struct {
	Body string `json:"body"`
}

type ghIssueRef struct {
	Number int `json:"number"`
}

type ghPR struct {
	State                   string       `json:"state"`
	ClosingIssuesReferences []ghIssueRef `json:"closingIssuesReferences"`
	Comments                []ghComment  `json:"comments"`
}

// prJSON renders what `gh pr view --json state,comments,closingIssuesReferences`
// prints for a PR that closes the given issues (none if nil).
func prJSON(t *testing.T, state string, closes []int, comments ...string) string {
	t.Helper()
	pr := ghPR{State: state, ClosingIssuesReferences: []ghIssueRef{}, Comments: []ghComment{}}
	for _, n := range closes {
		pr.ClosingIssuesReferences = append(pr.ClosingIssuesReferences, ghIssueRef{Number: n})
	}
	for _, c := range comments {
		pr.Comments = append(pr.Comments, ghComment{Body: c})
	}
	b, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestPostMergeCeremony covers the decision the merged-PR slot cleanup in
// bip-conductor-poll acts on (issue #258).
func TestPostMergeCeremony(t *testing.T) {
	closes3 := []int{3}
	cases := []struct {
		name       string
		withStatus bool
		gh         func(t *testing.T) string
		ghExit     int
		want       string // exact output; "<dir>" is replaced by the clone dir
		wantPrefix bool   // match want as a prefix only
		wantCode   int
	}{
		{"terminal comment present", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, nonTerminalLead, nonTerminalLead, terminalLead)
		}, 0, "CEREMONY RAN #7", false, 0},
		{"backticked terminal comment", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, terminalLeadBackticked)
		}, 0, "CEREMONY RAN #7", false, 0},
		{"text after completed", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, terminalLeadTrailing)
		}, 0, "CEREMONY RAN #7", false, 0},
		{"colon inside the bold", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, terminalLeadColonInBold)
		}, 0, "CEREMONY RAN #7", false, 0},
		{"review comment quoting the marker", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, nonTerminalLead, quotesMarker)
		}, 0, "CEREMONY OWED #7 <dir>", false, 0},
		{"hand-posted preserved note is not pr-land's", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, handPostedPreserved)
		}, 0, "ceremony UNRUN for #3 (PR #7)", false, 1},
		// Human merge: the worker ended at a clean gate with its state files
		// in place, and the PR carries non-terminal lead comments only.
		{"human merge, lead owed", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, nonTerminalLead, nonTerminalLead)
		}, 0, "CEREMONY OWED #7 <dir>", false, 0},
		// A /bip-pr-land that posted its marker (Step 6a) and died before
		// deleting the state files (Step 9.5): the file wins, so the
		// ceremony is still owed rather than handed to an ended worker.
		{"pr-land marker but state still on disk", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, prLandComment)
		}, 0, "CEREMONY OWED #7 <dir>", false, 0},
		// Worker landed with /bip-pr-land, which deleted the state files;
		// its own final lead call owns the ceremony.
		{"pr-land ran, worker owns it", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, prLandComment)
		}, 0, "CEREMONY WORKER-OWNS #7", false, 0},
		// Lost race: the state files are gone and nothing else will run it.
		{"state gone, ceremony unrun", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, nonTerminalLead)
		}, 0, "ceremony UNRUN for #3 (PR #7)", false, 1},
		{"unrun, PR names no closing issue", false, func(t *testing.T) string {
			return prJSON(t, "MERGED", nil)
		}, 0, "ceremony UNRUN for ? (PR #7)", false, 1},
		{"not merged", true, func(t *testing.T) string {
			return prJSON(t, "OPEN", closes3)
		}, 0, "CEREMONY UNKNOWN #7: state is OPEN, not MERGED", false, 2},
		{"gh fails", true, func(*testing.T) string { return "HTTP 502" }, 1,
			"CEREMONY UNKNOWN #7: gh pr view failed: A new release of gh is available", true, 2},
		{"gh prints non-JSON", true, func(*testing.T) string { return "HTTP 502" }, 0,
			"CEREMONY UNKNOWN #7: gh output is not JSON", true, 2},
	}
	shells := []string{"bash"}
	if _, err := exec.LookPath("zsh"); err == nil {
		shells = append(shells, "zsh")
	} else {
		t.Log("zsh not on PATH: running the bash half only")
	}
	for _, shell := range shells {
		for _, tc := range cases {
			t.Run(shell+"/"+tc.name, func(t *testing.T) {
				dir := cloneWithStatus(t, tc.withStatus)
				got, code, args := runPostMergeCeremony(t, shell, dir, tc.gh(t), tc.ghExit)
				want := strings.ReplaceAll(tc.want, "<dir>", dir)
				if tc.wantPrefix && !strings.HasPrefix(got, want) {
					t.Errorf("output = %q, want prefix %q", got, want)
				} else if !tc.wantPrefix && got != want {
					t.Errorf("output = %q, want %q", got, want)
				}
				if code != tc.wantCode {
					t.Errorf("exit code = %d, want %d (output %q)", code, tc.wantCode, got)
				}
				if wantArgs := "pr view 7 -R owner/repo --json state,comments,closingIssuesReferences"; args != wantArgs {
					t.Errorf("gh called with %q, want %q", args, wantArgs)
				}
			})
		}
	}
}
