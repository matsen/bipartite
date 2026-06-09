# Repo layout configuration

The `layout:` block tells `bip` *where on disk a repo's working tree
lives*. It is opt-in: with no block configured anywhere, `bip` behaves
exactly as it did before issue #149 — one canonical clone per repo,
under `paths.code` (or `paths.writing`).

This guide covers the schema, precedence, template variables, and a
migration tip for EPIC users.

## Schema

There are two places `layout:` may appear:

### 1. Global default — `~/.config/bip/config.yml`

This is per-machine — your personal preference, not shared with others
on the same nexus.

```yaml
nexus_path: ~/re/nexus
# ...other global keys (api keys, tokens)...

# Optional. Absent block = clone mode (today's behavior).
layout:
  mode: clone                       # clone | worktree   (default: clone)
  worktree:
    root: "~/re/{repo}-workers"     # template; {repo} = matsen/bipartite → "bipartite"
    slot: "issue-{issue}"           # template; {issue}, {pr}, {branch}, {slug}
  clone:
    root: "~/re"                    # defaults to paths.code if absent
    names: []                       # EPIC-style multi-clone pool; empty = single canonical clone
```

The nested-per-mode shape is deliberate: with `mode: clone` the
`worktree:` block is inert, and vice versa. There is no
conditional-field anti-pattern; both blocks may be present and only the
selected mode's settings are consulted.

### 2. Per-repo override — `sources.yml`

`sources.yml` lives in the nexus (shared across users of that nexus),
so per-repo overrides express **repo-specific** facts (this repo needs
a different worktree root because it's huge, this repo opts out of
worktree mode entirely), not personal preferences.

```yaml
code:
  - repo: matsen/bipartite
    channel: bip
    # inherits global layout — most repos look like this

  - repo: matsen/tiny-docs
    channel: docs
    layout:
      mode: clone        # opt OUT of global worktree default for this repo

  - repo: matsen/big-thing
    channel: bt
    layout:
      worktree:
        root: ~/re/bt-special-workers   # bigger drive
```

## Precedence

Each leaf field resolves independently:

```
per-repo sources.yml layout  >  global config.yml layout  >  built-in default (clone)
```

Example. Global sets `mode: worktree` and `worktree.slot:
"issue-{issue}"`. A per-repo entry overrides only `worktree.root`. The
result for that repo:

- `mode` — `worktree` (inherited from global)
- `worktree.root` — the per-repo value
- `worktree.slot` — `"issue-{issue}"` (inherited from global)

## Template variables

`worktree.root` supports:

| Variable | Meaning |
|----------|---------|
| `{repo}` | The repo name (e.g. `matsen/bipartite` → `bipartite`) |
| `{code}` | Expanded value of `paths.code` from `$NEXUS_PATH/config.yml` |

`worktree.slot` supports:

| Variable | Meaning |
|----------|---------|
| `{issue}` | Issue number (empty when called for a PR ref) |
| `{pr}`    | PR number (empty when called for an issue ref) |
| `{branch}` | Branch name (`<N>-<slug>` by default) |
| `{slug}`  | Slugified GitHub issue/PR title (lowercase, ASCII, `-` separated, 40 chars max) |

Unknown placeholders error at **resolve time** (not config-load time),
so a typo in an opt-in template will not break read-only commands like
`bip checkin` for users who never opt in.

Tilde expansion runs after template substitution.

## Defaults

If `layout.mode: worktree` is set without further detail:

- `worktree.root` defaults to `{code}/{repo}-workers`
- `worktree.slot` defaults to `issue-{issue}`

So a minimal opt-in is just:

```yaml
layout:
  mode: worktree
```

## What happens when `bip spawn` runs in worktree mode

1. `bip spawn matsen/bipartite#149` resolves the repo's canonical
   clone (used for `git worktree add`).
2. It fetches the issue title and slugifies it.
3. It resolves the worktree path:
   `worktree.root / worktree.slot` (with the variables filled in).
4. If that path does not yet exist, it runs
   `git worktree add <path> -b <N>-<slug>` from the canonical clone.
5. The tmux window opens in the worktree.

If `bip spawn` is invoked without an issue or PR ref (an adhoc session
or one of the manual-`--dir` paths), worktree mode falls back to the
canonical clone and prints a one-line note — there is no silent
empty-string substitution into the slot template.

When `bip pr-land` later runs from that worktree, it detects the
worktree via `bip worktree primary`, runs the squash merge from the
primary clone, and removes the worktree via `bip worktree remove`
before deleting the branch.

## Migration from `.epic-config.json`

EPIC orchestration still reads `.epic-config.json` (its three
fields — `clone_root`, `clone_names`, `local_worktrees` — are
unchanged). The global `layout:` block is the preferred way to
configure worktree mode for *non-EPIC* `bip spawn`. Existing EPICs keep
working without any change. The EPIC config will be deprecated in favor
of the YAML form in a future PR once the new form has been used in
anger.
