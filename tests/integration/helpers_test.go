// Package integration provides integration tests for bipartite commands.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	bpBinary     string
	bpBinaryOnce sync.Once
	bpBinaryErr  error
)

// moduleRoot returns the absolute path to the repository root, derived from
// this file's own location (tests/integration/helpers_test.go).
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

// getBPBinary builds the bp binary once and returns its path.
func getBPBinary(t *testing.T) string {
	t.Helper()
	bpBinaryOnce.Do(func() {
		moduleRoot := moduleRoot(t)

		// Build bp to a temp location
		tmpDir, err := os.MkdirTemp("", "bp-test-*")
		if err != nil {
			bpBinaryErr = err
			return
		}
		bpBinary = filepath.Join(tmpDir, "bp")

		cmd := exec.Command("go", "build", "-o", bpBinary, "./cmd/bip")
		cmd.Dir = moduleRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			bpBinaryErr = &buildError{output: string(output), err: err}
			return
		}
	})
	if bpBinaryErr != nil {
		t.Fatalf("failed to build bp: %v", bpBinaryErr)
	}
	return bpBinary
}

type buildError struct {
	output string
	err    error
}

func (e *buildError) Error() string {
	return e.err.Error() + ": " + e.output
}

// setupTestRepo creates a minimal bipartite repo with test refs.
// Returns the repo directory, which also holds a config directory for XDG_CONFIG_HOME.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create .bipartite directory
	bpDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(filepath.Join(bpDir, "cache"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configContent := "pdf_root: \"\"\npdf_reader: system\n"
	if err := os.WriteFile(filepath.Join(bpDir, "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create refs.jsonl with test papers
	refsContent := `{"id":"PaperA","title":"Paper A","authors":[{"last":"A"}],"published":{"year":2024},"source":{"type":"manual"}}
{"id":"PaperB","title":"Paper B","authors":[{"last":"B"}],"published":{"year":2024},"source":{"type":"manual"}}
{"id":"PaperC","title":"Paper C","authors":[{"last":"C"}],"published":{"year":2024},"source":{"type":"manual"}}
`
	if err := os.WriteFile(filepath.Join(bpDir, "refs.jsonl"), []byte(refsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create global config directory with nexus_path pointing to test repo
	configDir := filepath.Join(tmpDir, "config", "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := "nexus_path: " + tmpDir + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// runBP executes the bp command with given args and returns output.
// Uses XDG_CONFIG_HOME to point to test-specific global config with nexus_path.
func runBP(t *testing.T, repoDir string, args ...string) (string, error) {
	t.Helper()
	bp := getBPBinary(t)
	cmd := exec.Command(bp, args...)
	cmd.Dir = repoDir
	// Set XDG_CONFIG_HOME to the test config directory (parent of bip/)
	configHome := filepath.Join(repoDir, "config")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configHome)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
