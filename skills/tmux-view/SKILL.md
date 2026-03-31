---
name: tmux-view
description: Open a file in a tmux popup window using less
allowed-tools: Bash
---

# /tmux-view

Open a file in an 80%×80% tmux popup using `less` for scrolling. Press `q` to close.

## Usage

```
/tmux-view <file>
```

## Behavior

Run:

```bash
tmux display-popup -w 80% -h 80% -E -- less <file>
```

That's it. No confirmation needed — just run it.
