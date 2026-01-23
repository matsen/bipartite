"""Tmux window spawning utilities."""

from __future__ import annotations

import os
import subprocess
import tempfile
from pathlib import Path


def is_in_tmux() -> bool:
    """Check if we're running inside a tmux session."""
    return os.environ.get("TMUX") is not None


def tmux_window_exists(window_name: str) -> bool:
    """Check if a tmux window with the given name exists."""
    result = subprocess.run(
        ["tmux", "list-windows", "-F", "#W"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        return False
    windows = result.stdout.strip().split("\n")
    return window_name in windows


def create_tmux_window(
    window_name: str, repo_path: Path, prompt: str, url: str
) -> bool:
    """Create a tmux window and run Claude Code with the given prompt.

    Args:
        window_name: Name for the tmux window.
        repo_path: Working directory for the window.
        prompt: Prompt text to pass to Claude Code.
        url: URL to display before the prompt (for reference).

    Returns:
        True if window was created successfully, False otherwise.
    """
    prompt_file = tempfile.NamedTemporaryFile(
        mode="w", prefix=f"review-{window_name}-", suffix=".txt", delete=False
    )
    prompt_file.write(prompt)
    prompt_file.close()
    prompt_path = prompt_file.name

    create_result = subprocess.run(
        ["tmux", "new-window", "-n", window_name, "-c", str(repo_path), "-P"],
        capture_output=True,
        text=True,
    )
    if create_result.returncode != 0:
        print(f"  Error creating window for {window_name}: {create_result.stderr}")
        os.unlink(prompt_path)
        return False

    display_cmd = (
        f'echo "\\n{url}\\n" && cat {prompt_path} && '
        f'claude --dangerously-skip-permissions "$(cat {prompt_path})"; '
        f"rm -f {prompt_path}"
    )

    subprocess.run(
        ["tmux", "send-keys", "-t", window_name, display_cmd, "Enter"],
        capture_output=True,
    )

    print(f"  Created window: {window_name}")
    return True
