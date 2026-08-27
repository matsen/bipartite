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
