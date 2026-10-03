# Context additions for a worker brief

Snippets `/bip-conductor-spawn` Step 4 appends under `IMPORTANT CONTEXT:` when the issue calls for them.

**Filesystem mode** — always include this block when the issue involves running jobs on remote compute nodes. Check `shared_filesystem` in `.epic-config.json`:

*When `shared_filesystem: false` (laptop — files must be synced):*
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip-scout to find an available server before remote operations
- REMOTE_DIR: read the repo's Makefile first. If it already derives
  REMOTE_DIR per slot (e.g. `REMOTE_DIR ?= ~/re/pz/$(notdir $(CURDIR))`),
  do not pass it. If you must override it, pass a full path: a bare slot
  name is resolved against the remote HOME and the run lands in the
  wrong place without error.
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... CMD='zig build -Doptimize=ReleaseFast'
- Wrap the experiment in a Snakemake workflow
```

*When `shared_filesystem: true` (NFS — files already visible on all nodes):*
```
- Use /bip-scout to find an available server before remote operations
- For short/medium jobs (< ~30 min): block on SSH
    ssh <host> "cd <absolute_clone_path> && <command>"
  Results appear on NFS immediately — no sync or polling needed.
- For long jobs (hours): background the SSH call, then poll local NFS paths
    ssh <host> "cd <absolute_clone_path> && nohup <command> > out.log 2>&1 &"
  Use the awaiting-results phase with check_files pointing to local NFS
  output paths — no SSH needed to poll, just test -f /nfs/path/output.
- Never use make remote-sync or make remote-tmux in NFS mode.
- SSH quoting tip: if a command has complex quoting or special characters,
  write it to a temp file (e.g. /tmp/run-<N>.sh), then:
    ssh <host> "bash /nfs/path/to/run-<N>.sh"
  Clean up the temp file when the command finishes.
- Use the absolute clone path in SSH commands (expand ~ from clone_root
  before embedding — remote shells resolve ~ relative to the SSH user's
  home, which may differ from the NFS path).
```

**For experiments (Snakemake workflows):**
```
- SSF143587 data is at ~/re/superfamily-pcp/results/SSF143587/
- Wrap the experiment in a Snakemake workflow
```

**Next to any build or run command you put in the prompt** (not in a separate warnings section), add:
```
- This exact command string is in YOUR OWN argv (a bip spawn worker's prompt IS
  its command line), so `pgrep -f` or `ps | grep` on ANY fragment of it matches
  your own session and a wait loop built from it can never exit. Poll a PID you
  captured at launch (`$!`), or just let a backgrounded Bash re-invoke you.
```

**If the issue adds a build-system target**, list as a deliverable whatever makes that target discoverable in the repo (a hand-maintained test-target table, a `make help` entry, a CI matrix row). On `matsengrp/phyz` the source→target table is the only lookup that exists, so a step absent from it is unreachable. If nothing makes it discoverable, say so in the brief.

**For code changes:**
```
- Run zig build test before committing
- Run make parity if touching shared alignment code
- Check PRE-MERGE-CHECKLIST.md
- NEVER trust a pre-existing zig-out/bin/phyz. Before you measure ANYTHING
  with it, run this from the clone root; if it prints STALE or UNDETERMINED,
  rebuild first. A stale binary does not error -- it returns a confident
  wrong number under newer-looking provenance.
      # `--version` prints `phyz <version> (<commit>)`; older binaries print
      # the describe string alone, hence the `-g` fallback.
      vl=$(zig-out/bin/phyz --version)
      bc=$(printf '%s' "$vl" | sed -n 's/.*(\([0-9a-f]\{7,40\}\))[[:space:]]*$/\1/p')
      # Validate: an unresolvable SHA otherwise falls through to "ok".
      if [ -n "$bc" ]; then bc=$(git rev-parse --verify --quiet "${bc}^{commit}") || bc=""; fi
      if [ -z "$bc" ]; then
        vs=$(printf '%s' "$vl" | awk '{print $NF}'); vs=${vs%-dirty}
        case "$vs" in *-g*) ref="${vs##*-g}" ;; *) ref="$vs" ;; esac
        bc=$(git rev-parse --verify --quiet "${ref}^{commit}") || bc=""
      fi
      hc=$(git rev-parse --verify HEAD)
      if [ -z "$bc" ] || [ "$bc" = "unknown" ]; then
        echo "UNDETERMINED: no commit recoverable from '$vl' (tag not fetched, or built with no git) -- do NOT assume stale"
      elif [ "$bc" != "$hc" ] && [ -n "$(git diff --name-only "$bc" "$hc" -- src build.zig build.zig.zon)" ]; then
        echo "STALE: binary=$bc HEAD=$hc, and code differs -- rebuild before measuring"
      fi
      # A commit match does NOT establish content identity on a dirty tree:
      [ -n "$(git status --porcelain)" ] && echo "NOTE: tree dirty -- commit matches, content identity not established"
```

The check compares code between the two commits (`src build.zig build.zig.zon`, deliberately not `tests`) so a results-only commit after a build is not STALE. It resolves both sides to a commit rather than comparing `git describe` strings, because phyz tags every commit and a describe string changes when its tag lands. It parses `phyz --version`, so re-run it against a fresh binary whenever that line's format changes.

**For phased work:**
```
- This issue has multiple phases. Start with Phase 1 only.
- Phase 1: <describe scope and gate criteria>
- Only proceed to Phase 2 if the gate passes.
```
