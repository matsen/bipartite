package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollideHelpMatchesSkillDoc makes CONSTITUTION.md's "Skill behavior MUST
// match --help output exactly" mechanical for `bip epic collide`, instead of
// a promise someone has to re-check by eye.
//
// skills/bip-conductor/SKILL.md pastes the command's --help into its
// invocation block. That paste is the conductor's contract with the command
// — the exit codes and the no-default scope rule are both in it — and the
// skills are symlinked live, so a drifted paste is what a running session
// reads. This test fails the moment the two diverge in either direction.
func TestCollideHelpMatchesSkillDoc(t *testing.T) {
	bp := getBPBinary(t)
	out, err := exec.Command(bp, "epic", "collide", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("bip epic collide --help: %v\n%s", err, out)
	}
	help := strings.TrimRight(string(out), "\n")

	skillPath := filepath.Join(moduleRoot(t), "skills", "bip-conductor", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, block := range fencedBlocks(string(data)) {
		if block == help {
			return
		}
	}

	// Report the closest thing we found, so the failure says what drifted
	// rather than only that something did.
	var closest string
	for _, block := range fencedBlocks(string(data)) {
		if strings.HasPrefix(block, "Check every clone in one EPIC clone pool") {
			closest = block
		}
	}
	if closest == "" {
		t.Fatalf("no fenced block in %s holds `bip epic collide --help`; paste it there (see CONSTITUTION.md section III)", skillPath)
	}
	t.Fatalf("the --help block in %s has drifted from the command.\n--- skill has ---\n%s\n--- command prints ---\n%s",
		skillPath, closest, help)
}

// fencedBlocks returns the contents of every ``` fenced block in a markdown
// document, with the opening fence's info string dropped.
func fencedBlocks(doc string) []string {
	var blocks []string
	var cur []string
	inBlock := false
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				blocks = append(blocks, strings.Join(cur, "\n"))
				cur = nil
			}
			inBlock = !inBlock
			continue
		}
		if inBlock {
			cur = append(cur, line)
		}
	}
	return blocks
}

// TestCollideRejectsIgnoredFlagCombinations: a flag that is accepted and
// then ignored is worse than one that is rejected, because the caller
// believes it scoped the run. --symbol measures this repo and --root scopes
// a clone pool; --path means nothing without --symbol.
func TestCollideRejectsIgnoredFlagCombinations(t *testing.T) {
	bp := getBPBinary(t)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"root with symbol", []string{"epic", "collide", "--root", t.TempDir(), "--symbol", "Foo"}, "different checks"},
		{"path without symbol", []string{"epic", "collide", "--path", "x/"}, "only applies to --symbol"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := exec.Command(bp, tc.args...).CombinedOutput()
			if err == nil {
				t.Fatalf("expected a non-zero exit, got success:\n%s", out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Errorf("output %q does not name why the combination is refused", out)
			}
		})
	}
}
