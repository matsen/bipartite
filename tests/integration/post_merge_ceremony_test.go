package integration

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	cmd := exec.Command(shell, "-c", spawnIntentCall(t, "post_merge_ceremony", cloneDir, "owner/repo", "7"))
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
	// The shape of #259's skills-steward approval (2026-09-23T00:39:30Z),
	// trimmed: prose quoting the header, and a fenced table whose rows are
	// Category lines. Neither is at the start, so it is not a lead comment
	// (issue #260).
	stewardApproval = "🤖 **Skills-steward approval for `8383219`.** Re-reviewed.\n\n```\n**Category**: completed      -> True\n**Category:** completed      -> True\nCategory: completed          -> False\n```\n\n- Requiring `**Issue Lead**` in the same comment as the `Category` line means a quoted marker no longer counts.\n"
	// The lead's template quoted in a code block: header and Category line
	// both present, header at the start of a LINE but not of the comment.
	// Fails if the anchor is ever made multiline.
	templateInCodeBlock = "Template:\n```\n🤖 **Issue Lead** (iteration N)\n\n**Category**: completed\n```"
	// A real terminal comment with leading whitespace, which pins lstrip/\s*.
	terminalLeadIndented = "\n  " + terminalLead
	prLandComment        = "🤖 EPIC worklog preserved to `/x/.preserved/3-2026-09-22` (issue #2216)."
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

// ceremonyCase is one row of the post_merge_ceremony decision table.
type ceremonyCase struct {
	name       string
	withStatus bool
	gh         func(t *testing.T) string
	ghExit     int
	want       string // exact output; "<dir>" is replaced by the clone dir
	wantPrefix bool   // match want as a prefix only
	wantCode   int
}

// ceremonyCases is shared by TestPostMergeCeremony (the helper) and
// TestLeadGuardAgreesWithHelper (the lead's jq copy of the same guard).
func ceremonyCases() []ceremonyCase {
	closes3 := []int{3}
	return []ceremonyCase{
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
		{"#259 steward approval quoting both markers", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, nonTerminalLead, stewardApproval)
		}, 0, "CEREMONY OWED #7 <dir>", false, 0},
		{"lead template quoted in a code block", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, templateInCodeBlock)
		}, 0, "CEREMONY OWED #7 <dir>", false, 0},
		{"terminal comment with leading whitespace", true, func(t *testing.T) string {
			return prJSON(t, "MERGED", closes3, terminalLeadIndented)
		}, 0, "CEREMONY RAN #7", false, 0},
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
}

// TestPostMergeCeremony covers the decision the merged-PR slot cleanup in
// bip-conductor-poll acts on (issue #258).
func TestPostMergeCeremony(t *testing.T) {
	cases := ceremonyCases()
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

// TestTerminalMarkerPatternsAgree pins the terminal-comment pattern in
// agents/issue-lead.md's Step 8 guard (a jq test() the lead runs) to the one
// post_merge_ceremony uses (a Python regex the poll runs). The lead is an
// agent, not a skill, so it cannot source the helper; two copies in two
// languages is the price, and this test is what keeps them from drifting.
func TestTerminalMarkerPatternsAgree(t *testing.T) {
	root := moduleRoot(t)
	lead, err := os.ReadFile(filepath.Join(root, "agents", "issue-lead.md"))
	if err != nil {
		t.Fatal(err)
	}
	helper, err := os.ReadFile(filepath.Join(root, "skills", "lib", "spawn-intent.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// jq string literal: test("...Category...") with \\ for each regex backslash.
	jq := regexp.MustCompile(`test\("(\\\\\*\\\\\*Category[^"]*)"\)`).FindSubmatch(lead)
	if jq == nil {
		t.Fatal("no Category test() found in agents/issue-lead.md")
	}
	py := regexp.MustCompile(`re\.search\(r"(\\\*\\\*Category[^"]*)"`).FindSubmatch(helper)
	if py == nil {
		t.Fatal("no Category re.search found in skills/lib/spawn-intent.sh")
	}
	jqPattern := strings.ReplaceAll(string(jq[1]), `\\`, `\`)
	if jqPattern != string(py[1]) {
		t.Errorf("patterns differ:\n  issue-lead.md jq: %s\n  spawn-intent.sh:  %s", jqPattern, py[1])
	}
	if !strings.Contains(string(lead), `test("^\\s*🤖 \\*\\*Issue Lead\\*\\*")`) {
		t.Error("issue-lead.md guard no longer requires the comment to begin with the Issue Lead header")
	}
	if !strings.Contains(string(helper), `b.lstrip().startswith("🤖 **Issue Lead**")`) {
		t.Error("post_merge_ceremony no longer requires the comment to begin with the Issue Lead header")
	}
}

// TestLeadGuardAgreesWithHelper runs the jq expression from
// agents/issue-lead.md's Step 8 guard over every MERGED fixture and checks it
// says "the ceremony ran" exactly where post_merge_ceremony prints RAN.
// TestTerminalMarkerPatternsAgree only compares pattern strings, so a change
// made the same wrong way in both copies passes it; this one does not
// (issue #260).
func TestLeadGuardAgreesWithHelper(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not on PATH")
	}
	lead, err := os.ReadFile(filepath.Join(moduleRoot(t), "agents", "issue-lead.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`-q '(\[\.comments\[\]\.body \| select\([^']*\)\] \| length)'`).FindSubmatch(lead)
	if m == nil {
		t.Fatal("no Step 8 guard expression found in agents/issue-lead.md")
	}
	expr := string(m[1])
	checked := 0
	for _, tc := range ceremonyCases() {
		payload := tc.gh(t)
		var pr ghPR
		if json.Unmarshal([]byte(payload), &pr) != nil || pr.State != "MERGED" {
			continue
		}
		checked++
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("jq", "-r", expr)
			cmd.Stdin = strings.NewReader(payload)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("jq: %v", err)
			}
			n, err := strconv.Atoi(strings.TrimSpace(string(out)))
			if err != nil {
				t.Fatalf("jq printed %q, not a count", out)
			}
			if ran := strings.HasPrefix(tc.want, "CEREMONY RAN"); (n > 0) != ran {
				t.Errorf("lead guard count = %d, but the helper's expected output is %q", n, tc.want)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no MERGED fixtures reached the jq control")
	}
}
