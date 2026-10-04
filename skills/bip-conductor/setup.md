# /bip-conductor setup

Run when `.epic-config.json` is missing.
Ask the user:

1. Are you using local git worktrees or separate clones for parallel work?
2. Where should slots live?
   (e.g.
   `~/re/pz-workers` for worktrees, or `~/re/pz` for clones)
3. (Clone mode only) What are the clone directory names?
   Which is the conductor?
4. What is the GitHub repo (`org/repo`)?
5. Are compute nodes on a shared NFS filesystem?
   (sets `shared_filesystem`)

**Note (worktree mode)**: The skill is run from the main repo itself, which acts as the conductor.
There is no separate conductor clone — `clone_root` is just where worktrees are placed.

Then create `.epic-config.json` with their answers and proceed.

## Configuration

The conductor skill reads `.epic-config.json` from the repo root — the same file `/bip-epic` reads.
This file is gitignored and must exist before either skill can operate; the conductor owns creating it.

**Clone mode** (remote compute or pre-existing clones):
```json
{
  "clone_root": "~/re/myproject",
  "clone_names": ["alpha", "beta", "gamma"],
  "new_clone_names": ["delta", "epsilon", "zeta"],
  "github_repo": "org/repo",
  "conductor": "alpha",
  "max_lead_iterations": 8
}
```

**Worktree mode** (local parallel work only):
```json
{
  "clone_root": "~/re/myproject-workers",
  "local_worktrees": true,
  "github_repo": "org/repo",
  "max_lead_iterations": 8
}
```

**Validation**: If `local_worktrees: true` and `clone_names` are both present, **stop and report an error** — they are mutually exclusive.
`clone_names` is meaningless in worktree mode because slots are created on demand and named after the issue.

Fields:
- **clone_root**: Parent directory containing all clones or worktrees.
  Also where the shared `.spawn-prompts/` intent directory, `.preserved/`, and any unfiled `ISSUE-*.md` drafts live.
- **clone_names**: (clone mode only) Existing clone directory names.
  This is the pool's universe; other directories under `clone_root` (e.g.
  CI clones) are unmanaged.
- **new_clone_names**: (clone mode only) Names available for creating new clones
- **local_worktrees**: (worktree mode) If `true`, use `git worktree` for local slots named `issue-N`
- **github_repo**: `org/repo` for `gh` commands
- **conductor**: (clone mode only) Which clone is the orchestrator (stays on main)
- **max_lead_iterations**: Max issue-lead evaluations before escalating to `needs-human` (default: 8)
- **shared_filesystem**: (optional, default `false`) Set to `true` when the conductor and all compute nodes share an NFS filesystem; the conductor composes direct SSH execution commands instead of `make remote-sync` calls, and experiment results are immediately visible on local NFS paths.
  Each machine sets this flag for itself.

`.epic-status.json` must be gitignored, along with `.epic-worklog.md`, `.epic-decisions.md`, and `.epic-notifications.log`.
Any file this skill writes to the **conductor cwd** needs a `.gitignore` entry in the consuming repo; files at `$CLONE_ROOT` (`.epic-session`, `.conductor-session`, `.spawn-prompts/`, `.preserved/`) do not, since that path is outside every clone's git tree.

EPIC orchestration reads `.epic-config.json`, not the `layout:` block in `~/.config/bip/config.yml` (which configures non-EPIC `bip spawn`).
