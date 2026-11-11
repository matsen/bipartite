package spawn

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// requireTmux skips the test if tmux isn't available in the test environment.
func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
}

// withTmuxSessionIn starts a detached tmux session with cwd set to dir,
// waits for its pane to report dir, runs fn, then tears the session down.
// tmux 3.6 starts the pane's shell asynchronously, so right after
// new-session pane_current_path can still be the server's cwd.
func withTmuxSessionIn(t *testing.T, dir, sessionName string, fn func()) {
	t.Helper()
	if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", dir).Run(); err != nil {
		t.Fatalf("starting tmux session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	want, err := canonicalizePath(dir)
	if err != nil {
		t.Fatalf("resolving %q: %v", dir, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		out, _ := exec.Command("tmux", "display-message", "-p", "-t", sessionName, "#{pane_current_path}").Output()
		last := strings.TrimSpace(string(out))
		if got, err := canonicalizePath(last); last != "" && err == nil && got == want {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pane of %s never reported cwd %q (last: %q)", sessionName, dir, last)
		}
		time.Sleep(20 * time.Millisecond)
	}
	fn()
}

func TestOccupyingPane(t *testing.T) {
	requireTmux(t)

	dir := t.TempDir()
	sessionName := "bip-test-occupying-pane"

	withTmuxSessionIn(t, dir, sessionName, func() {
		occupied, err := OccupyingPane(dir)
		if err != nil {
			t.Fatalf("OccupyingPane(%q) error: %v", dir, err)
		}
		if occupied == "" {
			t.Errorf("OccupyingPane(%q) = \"\", want a match for the live pane", dir)
		}
	})

	// After the session is torn down, no pane should occupy dir anymore.
	occupied, err := OccupyingPane(dir)
	if err != nil {
		t.Fatalf("OccupyingPane(%q) error after teardown: %v", dir, err)
	}
	if occupied != "" {
		t.Errorf("OccupyingPane(%q) = %q after session teardown, want \"\"", dir, occupied)
	}
}

func TestOccupyingPane_RelativeAndSymlinkedPathsMatch(t *testing.T) {
	requireTmux(t)

	realDir := t.TempDir()
	linkParent := t.TempDir()
	symlink := filepath.Join(linkParent, "link")
	if err := exec.Command("ln", "-s", realDir, symlink).Run(); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	sessionName := "bip-test-occupying-pane-symlink"
	withTmuxSessionIn(t, realDir, sessionName, func() {
		occupied, err := OccupyingPane(symlink)
		if err != nil {
			t.Fatalf("OccupyingPane(%q) error: %v", symlink, err)
		}
		if occupied == "" {
			t.Errorf("OccupyingPane(%q) = \"\", want a match through the symlink", symlink)
		}
	})
}

func TestBuildWindowName(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		number   int
		want     string
	}{
		{
			name:     "simple repo path",
			repoPath: "/Users/matsen/re/netam",
			number:   123,
			want:     "netam#123",
		},
		{
			name:     "nested repo path",
			repoPath: "/home/user/projects/work/my-repo",
			number:   42,
			want:     "my-repo#42",
		},
		{
			name:     "single directory",
			repoPath: "repo",
			number:   1,
			want:     "repo#1",
		},
		{
			name:     "path with trailing slash",
			repoPath: "/Users/matsen/re/netam/",
			number:   456,
			want:     "netam#456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWindowName(tt.repoPath, tt.number)
			if got != tt.want {
				t.Errorf("BuildWindowName(%q, %d) = %q, want %q", tt.repoPath, tt.number, got, tt.want)
			}
		})
	}
}

func TestIsInTmux(t *testing.T) {
	// This test just verifies the function runs without panic.
	// The actual result depends on the test environment.
	_ = IsInTmux()
}

func TestBuildClaudeInvocation(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		windowName string
		want       string
	}{
		{
			name:       "empty model passes --name only",
			model:      "",
			windowName: "bipartite#281",
			want:       `claude --dangerously-skip-permissions --name 'bipartite#281' "$prompt"`,
		},
		{
			name:       "model is passed through as --model, alongside --name",
			model:      "opus",
			windowName: "bipartite#281",
			want:       `claude --dangerously-skip-permissions --model 'opus' --name 'bipartite#281' "$prompt"`,
		},
		{
			name:       "single quotes in windowName and model are escaped",
			model:      "test'model",
			windowName: "repo#'test'",
			want:       `claude --dangerously-skip-permissions --model 'test'\''model' --name 'repo#'\''test'\''' "$prompt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClaudeInvocation(tt.model, tt.windowName)
			if got != tt.want {
				t.Errorf("buildClaudeInvocation(%q, %q) = %q, want %q", tt.model, tt.windowName, got, tt.want)
			}
		})
	}
}

func TestBuildAgyInvocation(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "empty model passes -i only",
			model: "",
			want:  `agy --dangerously-skip-permissions -i "$prompt"`,
		},
		{
			name:  "model is passed through as --model",
			model: "gemini-1.5-pro",
			want:  `agy --dangerously-skip-permissions --model 'gemini-1.5-pro' -i "$prompt"`,
		},
		{
			name:  "single quotes in model are escaped",
			model: "gemini'pro",
			want:  `agy --dangerously-skip-permissions --model 'gemini'\''pro' -i "$prompt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAgyInvocation(tt.model)
			if got != tt.want {
				t.Errorf("buildAgyInvocation(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		agent   string
		wantErr bool
	}{
		{"", false},
		{"claude", false},
		{"agy", false},
		{"unsupported", true},
		{"CLAUDE", true},
		{"AGY", true},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			err := ValidateAgent(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgent(%q) err = %v, wantErr = %v", tt.agent, err, tt.wantErr)
			}
		})
	}
}

func TestCreateWindow_UnsupportedAgent(t *testing.T) {
	err := CreateWindow("test-window", "/tmp", "prompt", "", "", "invalid")
	if err == nil {
		t.Fatal("CreateWindow with invalid agent should return error")
	}
	if !strings.Contains(err.Error(), "unsupported agent") {
		t.Errorf("CreateWindow error = %q, want containing 'unsupported agent'", err.Error())
	}
}

func TestCreateWindow_NotFoundInPATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := CreateWindow("test-window", "/tmp", "prompt", "", "", "agy")
	if err == nil {
		t.Fatal("CreateWindow with nonexistent executable should return error")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("CreateWindow error = %q, want containing 'not found in PATH'", err.Error())
	}
}

func TestBriefPointer(t *testing.T) {
	brief := "You are working on GitHub issue #1416 build (ubuntu-latest, 3.11).\n\nIMPORTANT CONTEXT:\n" + strings.Repeat("x", 20000)
	got := briefPointer(brief, "/s/bip/briefs/w.txt")
	if !strings.HasPrefix(got, "You are working on GitHub issue #1416") {
		t.Errorf("pointer should start with the brief's first line, got %q", got)
	}
	if !strings.Contains(got, "/s/bip/briefs/w.txt") {
		t.Errorf("pointer should name the brief path, got %q", got)
	}
	if strings.Contains(got, "IMPORTANT CONTEXT") || len(got) > 400 {
		t.Errorf("pointer should carry only the first line, got %d bytes", len(got))
	}

	long := briefPointer(strings.Repeat("é", 300), "/p")
	if first, _, _ := strings.Cut(long, "\n"); len([]rune(first)) != 201 {
		t.Errorf("first line should be cut to 200 runes plus an ellipsis, got %d runes", len([]rune(first)))
	}
}

func TestRunnerPointer(t *testing.T) {
	if got := runnerPointer("claude", "brief", "/p"); strings.Contains(got, "~/.claude/CLAUDE.md") {
		t.Errorf("claude pointer should not carry the agy note, got %q", got)
	}
	got := runnerPointer("agy", "brief", "/p")
	for _, want := range []string{"/p", "~/.claude/CLAUDE.md", "~/.claude/skills/bip-<name>/SKILL.md", "PR body"} {
		if !strings.Contains(got, want) {
			t.Errorf("agy pointer should contain %q, got %q", want, got)
		}
	}
}

func TestTrustAgyWorkspace(t *testing.T) {
	settings := filepath.Join(t.TempDir(), "agy", "settings.json")
	for range 2 { // the second call must not add a duplicate
		if err := trustAgyWorkspace(settings, "/clones/a"); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := os.ReadFile(settings); err != nil || strings.Count(string(data), "/clones/a") != 1 {
		t.Errorf("a second call should not add a duplicate, got %s", data)
	}
	if err := os.WriteFile(settings, []byte(`{"other": 1, "trustedWorkspaces": ["/home/x"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := trustAgyWorkspace(settings, "/clones/b"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["other"] != float64(1) {
		t.Errorf("other keys should be kept, got %v", got)
	}
	trusted := got["trustedWorkspaces"].([]any)
	if len(trusted) != 2 || trusted[0] != "/home/x" || trusted[1] != "/clones/b" {
		t.Errorf("trustedWorkspaces = %v, want [/home/x /clones/b]", trusted)
	}

	if err := os.WriteFile(settings, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := trustAgyWorkspace(settings, "/clones/c"); err == nil {
		t.Error("a settings file that is not JSON should be an error, not overwritten")
	}
}

func TestTrustAgyWorkspaceKeepsUserData(t *testing.T) {
	dir := t.TempDir()
	write := func(path, body string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	notList := filepath.Join(dir, "notlist.json")
	write(notList, `{"trustedWorkspaces": "/home/x"}`, 0o600)
	if err := trustAgyWorkspace(notList, "/clones/a"); err == nil {
		t.Error("a trustedWorkspaces that is not a list should be an error")
	}
	if got := read(notList); got != `{"trustedWorkspaces": "/home/x"}` {
		t.Errorf("file should be unchanged on error, got %s", got)
	}

	null := filepath.Join(dir, "null.json")
	write(null, "null", 0o600)
	if err := trustAgyWorkspace(null, "/clones/a"); err != nil {
		t.Errorf("a file holding null should be treated as empty, got %v", err)
	}

	big := filepath.Join(dir, "big.json")
	write(big, `{"n": 9007199254740993, "s": "<a&b>"}`, 0o644)
	if err := trustAgyWorkspace(big, "/clones/a"); err != nil {
		t.Fatal(err)
	}
	if got := read(big); !strings.Contains(got, "9007199254740993") || !strings.Contains(got, "<a&b>") {
		t.Errorf("numbers and strings should be kept as written, got %s", got)
	}
	if info, err := os.Stat(big); err != nil || info.Mode().Perm() != 0o644 {
		t.Errorf("file mode should be kept at 0644, got %v (%v)", info.Mode().Perm(), err)
	}

	trailing := filepath.Join(dir, "trailing.json")
	write(trailing, "{\"a\":1}\n{\"b\":2}\n", 0o600)
	if err := trustAgyWorkspace(trailing, "/clones/a"); err == nil {
		t.Error("data after the settings object should be an error")
	}
	if got := read(trailing); got != "{\"a\":1}\n{\"b\":2}\n" {
		t.Errorf("file should be unchanged on error, got %s", got)
	}

	dangling := filepath.Join(dir, "dangling.json")
	if err := os.Symlink(filepath.Join(dir, "missing", "settings.json"), dangling); err != nil {
		t.Fatal(err)
	}
	if err := trustAgyWorkspace(dangling, "/clones/a"); err == nil {
		t.Error("a symlink with a missing target should be an error")
	}
	if info, err := os.Lstat(dangling); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("a dangling symlink should not be replaced")
	}

	realClone := filepath.Join(dir, "clone")
	if err := os.Mkdir(realClone, 0o755); err != nil {
		t.Fatal(err)
	}
	cloneLink := filepath.Join(dir, "clone-link")
	if err := os.Symlink(realClone, cloneLink); err != nil {
		t.Fatal(err)
	}
	forms := filepath.Join(dir, "forms.json")
	if err := trustAgyWorkspace(forms, cloneLink); err != nil {
		t.Fatal(err)
	}
	resolvedClone, _ := filepath.EvalSymlinks(realClone)
	if got := read(forms); !strings.Contains(got, cloneLink) || !strings.Contains(got, resolvedClone) {
		t.Errorf("both the link and resolved forms of the clone should be trusted, got %s", got)
	}

	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	write(target, `{}`, 0o600)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := trustAgyWorkspace(link, "/clones/a"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("a symlinked settings file should stay a symlink")
	}
	if !strings.Contains(read(target), "/clones/a") {
		t.Error("the symlink's target should get the new entry")
	}
}

func TestWriteBrief(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	brief := "line one\nline two"
	path, err := writeBrief("dasm2#1416", brief)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(state, "bip", "briefs", "dasm2#1416.txt"); path != want {
		t.Errorf("brief path = %s, want %s", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != brief {
		t.Errorf("brief file = %q, want %q", data, brief)
	}
}
