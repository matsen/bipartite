// Package spawn provides tmux window spawning utilities for issue/PR review.
package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	pointer := strings.ReplaceAll(briefPointer(prompt, briefPath), "'", `'\''`)

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
	cmd := exec.Command("tmux", "new-window", "-d", "-n", windowName, "-c", repoPath, "-P")
	output, err := cmd.Output()
	if err != nil {
		os.Remove(launcherPath)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("creating window: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("creating window: %w", err)
	}

	// Window created successfully
	_ = output // Contains the window target, not needed

	// Run the launcher script — bash reads the file safely
	cmd = exec.Command("tmux", "send-keys", "-t", windowName, launcherPath, "Enter")
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
