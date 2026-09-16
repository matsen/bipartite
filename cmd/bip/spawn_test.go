package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/gitx"
)

// TestSpawnWorktreeIntegration exercises the flow.ResolveRepoPath +
// gitx.AddWorktree composition that bip spawn uses in worktree mode,
// without invoking runSpawn (which shells out to gh and tmux). It covers
// the two end-to-end behaviors the issue's test plan calls out:
//
//   - Worktree creation: resolver returns IsNew=true on a fresh path, and
//     gitx.AddWorktree actually creates the directory on the named branch.
//   - Reuse: resolving the same issue twice has IsNew=false on the second
//     call, so spawn would skip the `git worktree add` invocation.
func TestSpawnWorktreeIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	root := t.TempDir()
	nexus := filepath.Join(root, "nexus")
	if err := os.MkdirAll(nexus, 0o755); err != nil {
		t.Fatal(err)
	}

	// Initialize a primary clone with one commit so `git worktree add` works.
	codeDir := filepath.Join(root, "code")
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(codeDir, "bipartite")
	mustRun(t, "", "git", "init", "--quiet", "--initial-branch=main", primary)
	mustRun(t, primary, "git", "config", "user.email", "test@example.com")
	mustRun(t, primary, "git", "config", "user.name", "Test")
	mustRun(t, primary, "git", "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(primary, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, primary, "git", "add", "README.md")
	mustRun(t, primary, "git", "commit", "--quiet", "-m", "initial")

	// Wire up sources.yml + flow config + global config in worktree mode.
	wtRoot := filepath.Join(root, "workers")
	sourcesYAML := `code:
  - repo: matsen/bipartite
`
	if err := os.WriteFile(filepath.Join(nexus, "sources.yml"), []byte(sourcesYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	flowCfg := "paths:\n  code: " + codeDir + "\n"
	if err := os.WriteFile(filepath.Join(nexus, "config.yml"), []byte(flowCfg), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgHome := filepath.Join(root, "xdg")
	if err := os.MkdirAll(filepath.Join(cfgHome, "bip"), 0o755); err != nil {
		t.Fatal(err)
	}
	globalYAML := "layout:\n  mode: worktree\n  worktree:\n    root: " + wtRoot + "/{repo}-workers\n"
	if err := os.WriteFile(filepath.Join(cfgHome, "bip", "config.yml"), []byte(globalYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	config.ResetGlobalConfigCache()
	t.Cleanup(config.ResetGlobalConfigCache)

	ctx := flow.ResolveContext{IssueNumber: 149, Slug: "worktrees"}
	resolved, err := flow.ResolveRepoPath(nexus, "matsen/bipartite", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Mode != config.LayoutModeWorktree {
		t.Fatalf("Mode = %q, want worktree", resolved.Mode)
	}
	if !resolved.IsNew {
		t.Errorf("IsNew = false, want true for a fresh path")
	}
	if resolved.Branch != "149-worktrees" {
		t.Errorf("Branch = %q, want 149-worktrees", resolved.Branch)
	}

	if err := gitx.AddWorktree(primary, resolved.Path, resolved.Branch); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if _, err := os.Stat(resolved.Path); err != nil {
		t.Fatalf("worktree not created at %s: %v", resolved.Path, err)
	}

	// Reuse: second resolve must report IsNew=false now that the worktree
	// is on disk. spawn keys off this to skip a redundant `git worktree add`.
	resolved2, err := flow.ResolveRepoPath(nexus, "matsen/bipartite", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resolved2.IsNew {
		t.Errorf("second resolve: IsNew = true, want false (worktree already exists)")
	}
	if resolved2.Path != resolved.Path {
		t.Errorf("second resolve path drift: %q vs %q", resolved2.Path, resolved.Path)
	}
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func TestResolveAgent(t *testing.T) {
	tests := []struct {
		name      string
		flagVal   string
		configVal string
		want      string
		wantErr   string
	}{
		{
			name:      "CLI flag --agent agy overrides config and defaults to agy",
			flagVal:   "agy",
			configVal: "claude",
			want:      "agy",
		},
		{
			name:      "Empty CLI flag falls back to global config spawn_agent: agy",
			flagVal:   "",
			configVal: "agy",
			want:      "agy",
		},
		{
			name:      "Empty CLI flag and empty config defaults to claude",
			flagVal:   "",
			configVal: "",
			want:      "claude",
		},
		{
			name:      "Invalid agent returns an error without invoking external commands",
			flagVal:   "unsupported",
			configVal: "",
			wantErr:   `unsupported agent "unsupported": must be 'claude' or 'agy'`,
		},
		{
			name:      "Invalid config agent returns an error when flag is empty",
			flagVal:   "",
			configVal: "invalid",
			wantErr:   `unsupported agent "invalid": must be 'claude' or 'agy'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAgent(tt.flagVal, tt.configVal)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveAgent(%q, %q) expected error, got nil", tt.flagVal, tt.configVal)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("resolveAgent(%q, %q) error = %q, want containing %q", tt.flagVal, tt.configVal, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveAgent(%q, %q) unexpected error: %v", tt.flagVal, tt.configVal, err)
			}
			if got != tt.want {
				t.Errorf("resolveAgent(%q, %q) = %q, want %q", tt.flagVal, tt.configVal, got, tt.want)
			}
		})
	}
}

func TestResolveAgent_EnvVarIntegration(t *testing.T) {
	t.Setenv("BIP_SPAWN_AGENT", "agy")
	config.ResetGlobalConfigCache()
	defer config.ResetGlobalConfigCache()

	got, err := resolveAgent("", config.GetSpawnAgent())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "agy" {
		t.Errorf("got %q, want agy", got)
	}
}
