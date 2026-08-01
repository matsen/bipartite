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

// CreateWindow creates a tmux window and runs Claude Code with the given prompt.
// model, if non-empty, is passed through as claude's --model flag.
func CreateWindow(windowName, repoPath, prompt, url, model string) error {
	// Write prompt to temp file
	promptFile, err := os.CreateTemp("", fmt.Sprintf("review-%s-*.txt", windowName))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer promptFile.Close()

	if _, err := promptFile.WriteString(prompt); err != nil {
		os.Remove(promptFile.Name())
		return fmt.Errorf("writing prompt: %w", err)
	}
	promptPath := promptFile.Name()

	// Write a launcher script that reads the prompt into a bash variable
	// and passes it to claude. This avoids zsh shell expansion issues —
	// the prompt content never gets interpreted by the shell.
	launcherFile, err := os.CreateTemp("", fmt.Sprintf("launcher-%s-*.sh", windowName))
	if err != nil {
		os.Remove(promptPath)
		return fmt.Errorf("creating launcher: %w", err)
	}

	launcherPath := launcherFile.Name()
	var urlLine string
	if url != "" {
		urlLine = fmt.Sprintf("echo '%s'\necho ''\n", url)
	}
	claudeInvocation := buildClaudeInvocation(model)
	launcherContent := fmt.Sprintf(`#!/bin/bash
%scat '%s'
echo '---'
prompt=$(<'%s')
rm -f '%s' '%s'
%s
`, urlLine, promptPath, promptPath, promptPath, launcherPath, claudeInvocation)

	if _, err := launcherFile.WriteString(launcherContent); err != nil {
		os.Remove(promptPath)
		os.Remove(launcherPath)
		launcherFile.Close()
		return fmt.Errorf("writing launcher: %w", err)
	}
	launcherFile.Close()

	if err := os.Chmod(launcherPath, 0755); err != nil {
		os.Remove(promptPath)
		os.Remove(launcherPath)
		return fmt.Errorf("chmod launcher: %w", err)
	}

	// Create tmux window
	cmd := exec.Command("tmux", "new-window", "-n", windowName, "-c", repoPath, "-P")
	output, err := cmd.Output()
	if err != nil {
		os.Remove(promptPath)
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

	return nil
}

// buildClaudeInvocation returns the shell command that launches claude, with
// --model injected only when model is non-empty. Empty model must produce a
// byte-identical command to the pre-existing behavior.
func buildClaudeInvocation(model string) string {
	if model == "" {
		return `claude --dangerously-skip-permissions "$prompt"`
	}
	return fmt.Sprintf(`claude --dangerously-skip-permissions --model '%s' "$prompt"`, model)
}

// BuildWindowName creates a window name from repo and number.
func BuildWindowName(repoPath string, number int) string {
	repoName := filepath.Base(repoPath)
	return fmt.Sprintf("%s#%d", repoName, number)
}
