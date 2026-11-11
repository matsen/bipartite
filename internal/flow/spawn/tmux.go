// Package spawn provides tmux window spawning utilities for issue/PR review.
package spawn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// IsInTmux checks if we're running inside a tmux session.
func IsInTmux() bool {
	return os.Getenv("TMUX") != ""
}

// WindowExists checks if a tmux window with the given name exists.
func WindowExists(windowName string) bool {
	cmd := exec.Command("tmux", "list-windows", "-F", "#W")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, w := range windows {
		if w == windowName {
			return true
		}
	}
	return false
}

// OccupyingPane returns the canonicalized cwd of a live tmux pane already
// sitting in repoPath, or "" if none. Both sides are resolved through
// EvalSymlinks + Abs so ~, relative, and symlinked forms compare equal.
func OccupyingPane(repoPath string) (string, error) {
	target, err := canonicalizePath(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		// No tmux server or no panes; nothing is occupying anything.
		return "", nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		paneDir, err := canonicalizePath(line)
		if err != nil {
			continue
		}
		if paneDir == target {
			return line, nil
		}
	}
	return "", nil
}

func canonicalizePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// Path may not exist on disk (e.g. a pane cwd that was since
		// removed); fall back to the absolute form.
		return abs, nil
	}
	return resolved, nil
}

// ValidateAgent validates that agent is an allowed agent runner ("claude" or "agy").
// An empty string is accepted and treated as "claude" by callers.
func ValidateAgent(agent string) error {
	switch agent {
	case "", "claude", "agy":
		return nil
	default:
		return fmt.Errorf("unsupported agent %q: must be 'claude' or 'agy'", agent)
	}
}

// CreateWindow creates a tmux window and runs the configured agent runner
// (claude or agy) with the given prompt. model, if non-empty, is passed through
// to the runner. For claude, windowName also becomes the session's --name, so
// it doubles as the address other sessions use to reach this one via
// ListAgents/SendMessage.
func CreateWindow(windowName, repoPath, prompt, url, model, agent string) error {
	if err := ValidateAgent(agent); err != nil {
		return err
	}

	executable := "claude"
	var invocation string
	switch agent {
	case "", "claude":
		invocation = buildClaudeInvocation(model, windowName)
	case "agy":
		executable = "agy"
		invocation = buildAgyInvocation(model)
	}

	if _, err := exec.LookPath(executable); err != nil {
		return fmt.Errorf("agent runner %q not found in PATH: %w", executable, err)
	}
	if agent == "agy" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		if err := trustAgyWorkspace(filepath.Join(home, agySettingsPath), repoPath); err != nil {
			return fmt.Errorf("pre-trusting %s for agy: %w", repoPath, err)
		}
	}

	// Keep the brief out of argv: a 20 KB prompt there is matched by any
	// other session's `pgrep -f`/`pkill -f`. The runner gets a short pointer
	// and reads the brief from a file that outlives the launcher.
	briefPath, err := writeBrief(windowName, prompt)
	if err != nil {
		return err
	}
	launched := false
	defer func() {
		if !launched {
			os.Remove(briefPath)
		}
	}()
	pointer := strings.ReplaceAll(runnerPointer(agent, prompt, briefPath), "'", `'\''`)

	// Write a launcher script that shows the brief in the pane and passes
	// the pointer to the agent runner through a bash variable.
	launcherFile, err := os.CreateTemp("", fmt.Sprintf("launcher-%s-*.sh", windowName))
	if err != nil {
		return fmt.Errorf("creating launcher: %w", err)
	}

	launcherPath := launcherFile.Name()
	var urlLine string
	if url != "" {
		urlLine = fmt.Sprintf("echo '%s'\necho ''\n", url)
	}
	launcherContent := fmt.Sprintf(`#!/bin/bash
%scat '%s'
echo '---'
prompt='%s'
rm -f '%s'
%s
`, urlLine, briefPath, pointer, launcherPath, invocation)

	if _, err := launcherFile.WriteString(launcherContent); err != nil {
		os.Remove(launcherPath)
		launcherFile.Close()
		return fmt.Errorf("writing launcher: %w", err)
	}
	launcherFile.Close()

	if err := os.Chmod(launcherPath, 0755); err != nil {
		os.Remove(launcherPath)
		return fmt.Errorf("chmod launcher: %w", err)
	}

	// Create tmux window
	cmd := exec.Command("tmux", "new-window", "-d", "-n", windowName, "-c", repoPath, "-P", "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		os.Remove(launcherPath)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("creating window: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("creating window: %w", err)
	}

	// Target the pane by id: tmux reads a name with a leading dash as "previous window".
	pane := strings.TrimSpace(string(output))

	// Run the launcher script — bash reads the file safely
	cmd = exec.Command("tmux", "send-keys", "-t", pane, launcherPath, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sending command to window: %w", err)
	}

	launched = true
	return nil
}

// writeBrief saves the prompt as <state dir>/bip/briefs/<window>.txt and
// returns its path. The state dir is $XDG_STATE_HOME, else ~/.local/state,
// so the file survives a reboot: a worker re-reads its brief after /clear.
// The next spawn of the same window name overwrites it.
func writeBrief(windowName, prompt string) (string, error) {
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home dir: %w", err)
		}
		state = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(state, "bip", "briefs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating briefs dir: %w", err)
	}
	path := filepath.Join(dir, windowName+".txt")
	if err := os.WriteFile(path, []byte(prompt), 0o600); err != nil {
		return "", fmt.Errorf("writing brief: %w", err)
	}
	return path, nil
}

// briefPointer is the short prompt the runner gets in argv: the brief's
// first line, so session listings still show the topic, and the brief's path.
func briefPointer(prompt, path string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(prompt), "\n")
	if r := []rune(first); len(r) > 200 {
		first = string(r[:200]) + "…"
	}
	return fmt.Sprintf("%s\n\nYour full brief is in %s. Read all of it before doing anything else, and follow it.", first, path)
}

// runnerPointer is briefPointer plus, for agy, agyNote.
func runnerPointer(agent, prompt, path string) string {
	if agent == "agy" {
		return briefPointer(prompt, path) + agyNote
	}
	return briefPointer(prompt, path)
}

// agyNote tells an agy runner where to find what Claude Code would have
// loaded for it: agy reads neither ~/.claude/CLAUDE.md nor the skills, and it
// cannot message other sessions.
const agyNote = "\n\nThis runner does not load Claude Code's instructions or skills. Before you act on the brief, read ~/.claude/CLAUDE.md and the repo's CLAUDE.md and AGENTS.md where they exist, and follow them. For each /bip-<name> skill the brief names, read and follow ~/.claude/skills/bip-<name>/SKILL.md. You cannot message other sessions: report through the PR body and issue comments."

// agySettingsPath is agy's settings file, relative to the home directory.
const agySettingsPath = ".gemini/antigravity-cli/settings.json"

// trustAgyWorkspace adds dir to the trustedWorkspaces list in agy's settings
// file, keeping its other keys. agy trusts exact paths only, and an untrusted
// directory gets a trust dialog that hides the runner's finished work.
// Numbers are kept exactly; key order and formatting are not.
func trustAgyWorkspace(settingsPath, dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	resolved, err := canonicalizePath(abs)
	if err != nil {
		return err
	}
	// Write through a symlinked settings file rather than replacing the link.
	if info, err := os.Lstat(settingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if settingsPath, err = filepath.EvalSymlinks(settingsPath); err != nil {
			return fmt.Errorf("settings file is a symlink whose target is missing: %w", err)
		}
	}
	mode := os.FileMode(0o600)
	settings := map[string]any{}
	data, err := os.ReadFile(settingsPath)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return err
	default:
		if info, err := os.Stat(settingsPath); err == nil {
			mode = info.Mode().Perm()
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if len(bytes.TrimSpace(data)) > 0 {
			if err := dec.Decode(&settings); err != nil {
				return fmt.Errorf("parsing %s: %w", settingsPath, err)
			}
			if _, err := dec.Token(); err != io.EOF {
				return fmt.Errorf("parsing %s: data after the settings object", settingsPath)
			}
		}
		if settings == nil { // the file held null
			settings = map[string]any{}
		}
	}
	var trusted []any
	if v, ok := settings["trustedWorkspaces"]; ok && v != nil {
		if trusted, ok = v.([]any); !ok {
			return fmt.Errorf("%s: trustedWorkspaces is not a list", settingsPath)
		}
	}
	// Trust both the path as given and its resolved form: which one agy
	// compares is not known.
	added := false
	for _, d := range []string{abs, resolved} {
		if !slices.Contains(trusted, any(d)) {
			trusted = append(trusted, d)
			added = true
		}
	}
	if !added {
		return nil
	}
	settings["trustedWorkspaces"] = trusted

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	// A unique temp file, so two spawns at once cannot interleave writes.
	tmp, err := os.CreateTemp(filepath.Dir(settingsPath), "settings-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(out.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), settingsPath)
}

// buildClaudeInvocation returns the shell command that launches claude, with
// --model injected only when model is non-empty. windowName is always passed
// via --name so the claude session is addressable by ListAgents/SendMessage
// under the same name as its tmux window, instead of a default that gives
// every worker sharing a clone root the same unhelpful prefix.
func buildClaudeInvocation(model, windowName string) string {
	escapedName := strings.ReplaceAll(windowName, "'", `'\''`)
	if model == "" {
		return fmt.Sprintf(`claude --dangerously-skip-permissions --name '%s' "$prompt"`, escapedName)
	}
	escapedModel := strings.ReplaceAll(model, "'", `'\''`)
	return fmt.Sprintf(`claude --dangerously-skip-permissions --model '%s' --name '%s' "$prompt"`, escapedModel, escapedName)
}

// buildAgyInvocation returns the shell command that launches agy in interactive
// mode (-i), with --model injected only when model is non-empty.
func buildAgyInvocation(model string) string {
	if model == "" {
		return `agy --dangerously-skip-permissions -i "$prompt"`
	}
	escapedModel := strings.ReplaceAll(model, "'", `'\''`)
	return fmt.Sprintf(`agy --dangerously-skip-permissions --model '%s' -i "$prompt"`, escapedModel)
}

// BuildWindowName creates a window name from repo and number.
func BuildWindowName(repoPath string, number int) string {
	repoName := filepath.Base(repoPath)
	return fmt.Sprintf("%s#%d", repoName, number)
}
