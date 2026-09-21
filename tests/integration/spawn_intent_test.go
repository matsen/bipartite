// Package integration provides integration tests for bipartite commands.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spawnIntentScriptPath returns the absolute path to skills/lib/spawn-intent.sh.
func spawnIntentScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), "skills", "lib", "spawn-intent.sh")
}

// shellQuote wraps s in single quotes for safe interpolation into a shell
// command string, escaping any single quotes it contains.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// callSpawnIntentFunc sources spawn-intent.sh and calls fn with the given
// arguments, returning trimmed stdout.
func callSpawnIntentFunc(t *testing.T, fn string, args ...string) string {
	t.Helper()
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	script := "source " + shellQuote(spawnIntentScriptPath(t)) + "\n" +
		fn + " " + strings.Join(quoted, " ")
	cmd := exec.Command("bash", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell snippet failed: %v\nOutput: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestSpawnIntentSourcesUnderZsh guards the shell this helper is actually used
// from. The skills tell an agent to `source` spawn-intent.sh, so it runs in
// that session's login shell -- zsh on these boxes -- while every other test
// here invokes it under bash. A bash-only idiom would pass CI and break every
// real caller silently; zsh and bash disagree on parameter expansion in ways
// that mangle values rather than erroring (skills/lib/hazards/zsh-colon-modifier.md).
//
// Read-only functions only: mark_spawn_intent_consumed moves a file, so it
// cannot be run twice against one fixture.
func TestSpawnIntentSourcesUnderZsh(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".epic-config.json")
	if err := os.WriteFile(configPath, []byte(`{"clone_root": "~/re/pz"}`), 0644); err != nil {
		t.Fatal(err)
	}

	script := "source " + shellQuote(spawnIntentScriptPath(t)) + "\n" +
		"resolve_clone_root " + shellQuote(configPath)

	outBash, errBash := exec.Command("bash", "-c", script).CombinedOutput()
	if errBash != nil {
		t.Fatalf("sourcing under bash failed: %v\nOutput: %s", errBash, outBash)
	}
	outZsh, errZsh := exec.Command("zsh", "-c", script).CombinedOutput()
	if errZsh != nil {
		t.Fatalf("sourcing under zsh failed: %v\nOutput: %s", errZsh, outZsh)
	}

	if got, want := strings.TrimSpace(string(outZsh)), strings.TrimSpace(string(outBash)); got != want {
		t.Fatalf("zsh and bash disagree: zsh=%q bash=%q", got, want)
	}
}

func TestResolveCloneRootTildeForm(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".epic-config.json")
	if err := os.WriteFile(configPath, []byte(`{"clone_root": "~/re/pz"}`), 0644); err != nil {
		t.Fatal(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "re", "pz")

	got := callSpawnIntentFunc(t, "resolve_clone_root", configPath)
	if got != want {
		t.Fatalf("resolve_clone_root(~/re/pz) = %q, want %q", got, want)
	}
}

func TestResolveCloneRootAbsoluteForm(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".epic-config.json")
	absPath := "/opt/repos/pz"
	if err := os.WriteFile(configPath, []byte(`{"clone_root": "`+absPath+`"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got := callSpawnIntentFunc(t, "resolve_clone_root", configPath)
	if got != absPath {
		t.Fatalf("resolve_clone_root(%s) = %q, want unchanged %q", absPath, got, absPath)
	}
}

func TestResolveCloneRootMissingFailsFast(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".epic-config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	script := "source " + shellQuote(spawnIntentScriptPath(t)) + "\n" +
		"resolve_clone_root " + shellQuote(configPath)
	cmd := exec.Command("bash", "-c", script)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("resolve_clone_root with no .clone_root succeeded, want non-zero exit; output: %s", out)
	}
	if !strings.Contains(string(out), "no .clone_root in") {
		t.Fatalf("resolve_clone_root error output = %q, want mention of missing .clone_root", out)
	}
}

func TestFindSpawnIntent(t *testing.T) {
	tests := []struct {
		name       string
		createMd   bool
		createTxt  bool
		wantSuffix string // relative to .spawn-prompts/, or "" for none found
	}{
		{name: "md only", createMd: true, wantSuffix: "302.md"},
		{name: "txt only", createTxt: true, wantSuffix: "spawn-302.txt"},
		{name: "both present prefers md", createMd: true, createTxt: true, wantSuffix: "302.md"},
		{name: "none found", wantSuffix: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cloneRoot := t.TempDir()
			promptsDir := filepath.Join(cloneRoot, ".spawn-prompts")
			if err := os.MkdirAll(promptsDir, 0755); err != nil {
				t.Fatal(err)
			}
			if tc.createMd {
				if err := os.WriteFile(filepath.Join(promptsDir, "302.md"), []byte("intent"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.createTxt {
				if err := os.WriteFile(filepath.Join(promptsDir, "spawn-302.txt"), []byte("intent"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			want := ""
			if tc.wantSuffix != "" {
				want = filepath.Join(promptsDir, tc.wantSuffix)
			}

			got := callSpawnIntentFunc(t, "find_spawn_intent", cloneRoot, "302")
			if got != want {
				t.Fatalf("find_spawn_intent = %q, want %q", got, want)
			}
		})
	}
}

func TestMarkSpawnIntentConsumed(t *testing.T) {
	cloneRoot := t.TempDir()
	promptsDir := filepath.Join(cloneRoot, ".spawn-prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	intentPath := filepath.Join(promptsDir, "302.md")
	if err := os.WriteFile(intentPath, []byte("intent"), 0644); err != nil {
		t.Fatal(err)
	}

	callSpawnIntentFunc(t, "mark_spawn_intent_consumed", intentPath)

	if _, err := os.Stat(intentPath); !os.IsNotExist(err) {
		t.Fatalf("intent file still present at original path after consuming: err=%v", err)
	}

	consumedPath := filepath.Join(promptsDir, "consumed", "302.md")
	data, err := os.ReadFile(consumedPath)
	if err != nil {
		t.Fatalf("expected consumed file at %s: %v", consumedPath, err)
	}
	if string(data) != "intent" {
		t.Fatalf("consumed file contents = %q, want %q", data, "intent")
	}

	// A file distinguishable as consumed must not still resolve as queued.
	got := callSpawnIntentFunc(t, "find_spawn_intent", cloneRoot, "302")
	if got != "" {
		t.Fatalf("find_spawn_intent after consuming = %q, want empty (not still queued)", got)
	}
}

func TestMarkSpawnIntentConsumedMissingSourceFailsFast(t *testing.T) {
	cloneRoot := t.TempDir()
	missingPath := filepath.Join(cloneRoot, ".spawn-prompts", "302.md")

	script := "source " + shellQuote(spawnIntentScriptPath(t)) + "\n" +
		"mark_spawn_intent_consumed " + shellQuote(missingPath)
	cmd := exec.Command("bash", "-c", script)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("mark_spawn_intent_consumed on missing source succeeded, want non-zero exit; output: %s", out)
	}
}
