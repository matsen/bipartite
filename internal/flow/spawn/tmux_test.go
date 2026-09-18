package spawn

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireTmux skips the test if tmux isn't available in the test environment.
func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
}

// withTmuxSessionIn starts a detached tmux session with cwd set to dir,
// runs fn, then tears the session down.
func withTmuxSessionIn(t *testing.T, dir, sessionName string, fn func()) {
	t.Helper()
	if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", dir).Run(); err != nil {
		t.Fatalf("starting tmux session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()
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
