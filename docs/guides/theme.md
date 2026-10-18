# Light and dark panes

Skills that show the user a document open it with `skills/lib/glow-pane <file>`, a glow split beside the calling tmux pane. Its style is tmux's `@theme` option (`light` or `dark`), or glow's `auto` when `@theme` is unset. bip-staff's queue pane uses it too.

## Following a switch

`skills/lib/theme-follow` respawns each of those panes when `@theme` changes. Run it from a tmux hook in `~/.tmux.conf`:

```tmux
set -g @theme-bg-light '#ffffff'   # your terminal theme's backgrounds; optional
set -g @theme-bg-dark  '#282c34'
set-hook -g after-set-option 'run-shell -b ~/.claude/skills/lib/theme-follow'
```

With the two background options set, it also sets `window-style` to the matching background. Then tmux answers a pane's colour queries itself (OSC 11, DSR 996), which it can't do while the client's colours never reach it. With those answers, glow's `auto` and Claude Code's `auto` theme both follow, Claude Code live in running sessions. Without the background options, theme-follow only respawns panes. It acts only when `@theme` changes, so after changing a background option, run `tmux set -gu @theme-applied` and set `@theme` again.

## Setting `@theme`

On a direct terminal, tmux (3.6 and later) learns the mode itself, through its `client-light-theme` and `client-dark-theme` hooks. A command run from a hook fires no further hooks, so `after-set-option` misses the `set` there and each hook calls theme-follow too:

```tmux
set-hook -g client-light-theme 'set -g @theme light ; run-shell -b ~/.claude/skills/lib/theme-follow'
set-hook -g client-dark-theme  'set -g @theme dark ; run-shell -b ~/.claude/skills/lib/theme-follow'
```

Under mosh, the terminal's colour queries never arrive, so something on the laptop has to push the mode. On a Mac, [`dark-notify`](https://github.com/cormacrelf/dark-notify) runs a command on every appearance change. Point it at:

```sh
ssh -o BatchMode=yes <host> "tmux -S <socket> set -g @theme $1 \; set -g theme $1"
```

Name the socket (`tmux display -p '#{socket_path}'`), because a non-interactive ssh shell may not see the `TMUX_TMPDIR` your shell profile sets. Run it as a LaunchAgent with `RunAtLoad` and `KeepAlive`.
