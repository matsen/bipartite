// Package integration provides integration tests for bipartite commands.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// spawnIntentScriptPath returns the absolute path to skills/lib/spawn-intent.sh.
func spawnIntentScriptPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return filepath.Join(moduleRoot, "skills", "lib", "spawn-intent.sh")
}

// runSpawnIntentShell sources spawn-intent.sh and runs the given shell
// snippet, returning trimmed stdout.
func runSpawnIntentShell(t *testing.T, script string) string {
	t.Helper()
	full := "source " + spawnIntentScriptPath(t) + "\n" + script
	cmd := exec.Command("bash", "-c", full)
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

	got := runSpawnIntentShell(t, "resolve_clone_root "+configPath)
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

	got := runSpawnIntentShell(t, "resolve_clone_root "+configPath)
	if got != absPath {
		t.Fatalf("resolve_clone_root(%s) = %q, want unchanged %q", absPath, got, absPath)
	}
}

func TestFindSpawnIntentMdOnly(t *testing.T) {
	cloneRoot := t.TempDir()
	promptsDir := filepath.Join(cloneRoot, ".spawn-prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(promptsDir, "302.md")
	if err := os.WriteFile(mdPath, []byte("intent"), 0644); err != nil {
		t.Fatal(err)
	}

	got := runSpawnIntentShell(t, "find_spawn_intent "+cloneRoot+" 302")
	if got != mdPath {
		t.Fatalf("find_spawn_intent = %q, want %q", got, mdPath)
	}
}

func TestFindSpawnIntentTxtOnly(t *testing.T) {
	cloneRoot := t.TempDir()
	promptsDir := filepath.Join(cloneRoot, ".spawn-prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	txtPath := filepath.Join(promptsDir, "spawn-302.txt")
	if err := os.WriteFile(txtPath, []byte("intent"), 0644); err != nil {
		t.Fatal(err)
	}

	got := runSpawnIntentShell(t, "find_spawn_intent "+cloneRoot+" 302")
	if got != txtPath {
		t.Fatalf("find_spawn_intent = %q, want %q", got, txtPath)
	}
}

func TestFindSpawnIntentPrefersMdWhenBothPresent(t *testing.T) {
	cloneRoot := t.TempDir()
	promptsDir := filepath.Join(cloneRoot, ".spawn-prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(promptsDir, "302.md")
	txtPath := filepath.Join(promptsDir, "spawn-302.txt")
	if err := os.WriteFile(mdPath, []byte("intent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txtPath, []byte("intent"), 0644); err != nil {
		t.Fatal(err)
	}

	got := runSpawnIntentShell(t, "find_spawn_intent "+cloneRoot+" 302")
	if got != mdPath {
		t.Fatalf("find_spawn_intent with both present = %q, want %q (should prefer <N>.md)", got, mdPath)
	}
}

func TestFindSpawnIntentNoneFound(t *testing.T) {
	cloneRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cloneRoot, ".spawn-prompts"), 0755); err != nil {
		t.Fatal(err)
	}

	got := runSpawnIntentShell(t, "find_spawn_intent "+cloneRoot+" 302")
	if got != "" {
		t.Fatalf("find_spawn_intent with no intent files = %q, want empty", got)
	}
}
