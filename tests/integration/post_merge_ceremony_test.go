package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// runPostMergeCeremony sources spawn-intent.sh under shell and calls
// post_merge_ceremony with a fake `gh` first on PATH. The fake prints
// ghOutput and exits ghExit, standing in for `gh pr view --json ...`.
// Returns trimmed stdout and the function's exit code.
func runPostMergeCeremony(t *testing.T, shell, cloneDir, ghOutput string, ghExit int) (string, int) {
	t.Helper()
	binDir := t.TempDir()
	payload := filepath.Join(binDir, "payload.json")
	if err := os.WriteFile(payload, []byte(ghOutput), 0644); err != nil {
		t.Fatal(err)
	}
	fake := "#!/bin/sh\ncat " + shellQuote(payload) + "\nexit " + strconv.Itoa(ghExit) + "\n"
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
	return strings.TrimSpace(string(out)), code
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
	// carries the same header a terminal comment does.
	nonTerminalLead = "🤖 **Issue Lead** (iteration 1)\n\n**Category**: `quality-gate` — gates clean. Not `completed`: humans merge here.\n"
	terminalLead    = "🤖 **Issue Lead** (iteration 3)\n\n**Category**: completed\n**Action**: none\n"
	// phyz#2920 wrote its terminal Category this way.
	terminalLeadBackticked = "🤖 **Issue Lead** (iteration 1) — terminal evaluation, post-land.\n\n**Category**: `completed`\n"
	prLandComment          = "🤖 EPIC worklog preserved to `/x/.preserved/3-2026-09-22` (issue #2216)."
)

func prJSON(state string, comments ...string) string {
	var b strings.Builder
	b.WriteString(`{"state":"` + state + `","closingIssuesReferences":[{"number":3}],"comments":[`)
	for i, c := range comments {
		if i > 0 {
			b.WriteString(",")
		}
		q := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(c)
		b.WriteString(`{"body":"` + q + `"}`)
	}
	b.WriteString("]}")
	return b.String()
}

// TestPostMergeCeremony covers the decision the merged-PR slot cleanup in
// bip-conductor-poll acts on (issue #258).
func TestPostMergeCeremony(t *testing.T) {
	cases := []struct {
		name       string
		withStatus bool
		gh         string
		ghExit     int
		wantPrefix string
		wantCode   int
	}{
		{"terminal comment present", true, prJSON("MERGED", nonTerminalLead, nonTerminalLead, terminalLead), 0, "CEREMONY RAN #7", 0},
		{"backticked terminal comment", false, prJSON("MERGED", terminalLeadBackticked), 0, "CEREMONY RAN #7", 0},
		// Human merge: the worker ended at a clean gate with its state files
		// in place, and the PR carries non-terminal lead comments only.
		{"human merge, lead owed", true, prJSON("MERGED", nonTerminalLead, nonTerminalLead), 0, "CEREMONY OWED #7 ", 0},
		// Worker landed with /bip-pr-land, which deleted the state files;
		// its own final lead call owns the ceremony.
		{"pr-land ran, worker owns it", false, prJSON("MERGED", prLandComment), 0, "CEREMONY WORKER-OWNS #7", 0},
		// Lost race: the state files are gone and nothing else will run it.
		{"state gone, ceremony unrun", false, prJSON("MERGED", nonTerminalLead), 0, "ceremony UNRUN for #3 (PR #7)", 1},
		{"not merged", true, prJSON("OPEN"), 0, "CEREMONY UNKNOWN #7: state is OPEN", 2},
		{"gh fails", true, "HTTP 502", 1, "CEREMONY UNKNOWN #7: gh pr view failed", 2},
		{"gh prints non-JSON", true, "HTTP 502", 0, "CEREMONY UNKNOWN #7: gh output is not JSON", 2},
	}
	shells := []string{"bash"}
	if _, err := exec.LookPath("zsh"); err == nil {
		shells = append(shells, "zsh")
	}
	for _, shell := range shells {
		for _, tc := range cases {
			t.Run(shell+"/"+tc.name, func(t *testing.T) {
				dir := cloneWithStatus(t, tc.withStatus)
				got, code := runPostMergeCeremony(t, shell, dir, tc.gh, tc.ghExit)
				if !strings.HasPrefix(got, tc.wantPrefix) {
					t.Errorf("output = %q, want prefix %q", got, tc.wantPrefix)
				}
				if code != tc.wantCode {
					t.Errorf("exit code = %d, want %d (output %q)", code, tc.wantCode, got)
				}
			})
		}
	}
}
