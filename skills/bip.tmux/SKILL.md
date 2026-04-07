---
name: bip.tmux
description: Open a file in a tmux popup window using less
allowed-tools: Bash
---

# /bip.tmux

Open a file in an 80%×80% tmux popup using `less` for scrolling. Press `q` to close.

## Usage

```
/bip.tmux <file>
```

## Behavior

Run:

```bash
tmux display-popup -w 80% -h 80% -E -- less <file>
```

That's it. No confirmation needed — just run it. After the command completes, stop and wait for the user to respond. Do not summarize the file contents, ask follow-up questions, or take any further action.
