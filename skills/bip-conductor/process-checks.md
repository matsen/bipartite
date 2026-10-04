# /bip-conductor process checks

A `bip spawn` worker's entire spawn prompt is its argv, so **never match argv** (`pgrep -f`, `ps | grep`) for a question about processes: it matches every worker whose prompt mentions the string, and the asking shell itself. Enumerate by exact name and attribute by working directory:

```bash
# Which clone is each live Claude session in?
for pid in $(pgrep -x claude); do
  printf '%s\t%s\n' "$pid" "$(readlink /proc/$pid/cwd)"
done
```

- `pgrep -x` is not fleet-scoped; filter on the cwd being under `$CLONE_ROOT`.
- To ask "what is running in my pool", don't hand-list names: take `/proc/<pid>/comm` for every pid whose cwd is under `$CLONE_ROOT`, `sort | uniq -c` (test runners are named neither `zig` nor after the product).
- On shared hosts, scope with `-u $(id -u)`. You cannot read another user's `/proc/<pid>/cwd`; treat an unreadable cwd as foreign and report own and foreign counts separately.
- To wait on something you launched, poll its `$!`, never a pattern — or background it and let the harness re-invoke you.
- Load average answers "is this host contended", not "is my job still running": enumerate processes for occupancy.

**A slot blocked on a foreground shell wait** reports `shell`, never `idle`, and cannot drain a `SendMessage`. Sweep whenever a slot reads `shell`, and on any full reconciliation:

```bash
for pid in $(pgrep -x zsh; pgrep -x bash; pgrep -x sh); do
  [ "$pid" = "$$" ] && continue
  cwd=$(readlink /proc/$pid/cwd 2>/dev/null); case "$cwd" in "$CLONE_ROOT"/*) ;; *) continue;; esac
  ppid=$(ps -o ppid= -p $pid | tr -d ' '); et=$(ps -o etimes= -p $pid | tr -d ' ')
  [ "$(ps -o comm= -p $ppid | tr -d ' ')" = "claude" ] || continue
  [ "$et" -gt 600 ] || continue
  echo "$et|$pid|${cwd##*/}"; tr '\0' '\n' < /proc/$pid/cmdline | tail -1 | /usr/bin/grep -o "eval '.*' < /dev/null"
done | sort -rn
```

Before killing one, confirm no real process of that clone is being waited on (`pgrep -x make` / `-x zig` plus a cwd match).

