# bipartite — agent guidance

## What this file is

**Facts + guardrails** for working in this repo — and *only* those. There are three layers,
and this is just one of them:

- **Principles / "why"** live in `CONSTITUTION.md` (numbered articles: skills-are-the-product,
  file-based state, scope discipline, quality gates). It is **not** auto-loaded — read it when
  making a design, scope, or quality-gate decision.
- **Procedures / "how to do X"** are the `bip-*` skills, listed with descriptions in my context
  every session. **Never catalog or describe a skill here** — duplicating the auto-injected
  list adds nothing and silently rots (the old `/bip.lit` dot-name drifted for months this
  way). Invoke skills by name; let their descriptions be canonical.

What's left for *this* file: rules that must hold even when no skill is invoked, plus project
facts no skill carries.

## Stack & layout

- Go (min version in `go.mod`). CLI: spf13/cobra. Storage: modernc.org/sqlite (pure Go, no
  CGO). External refs: Semantic Scholar (`internal/s2`).
- Data model: JSONL is the source of truth → ephemeral SQLite, rebuilt on `bip rebuild`.
- `cmd/` CLI commands · `internal/` packages (s2, store, flow, …) · `testdata/`
  fixtures · `tests/` integration tests.

## Build & style

- `go build -o bip ./cmd/bip && ./bip --help`
- Run `go fmt ./...` and `go vet ./...` before any PR. Exported symbols get doc comments.
- `skills/lib/spawn-intent.sh` is **sourced**, so it runs in the caller's shell and must stay
  bash/zsh portable — a shebang would be ignored. The other `.sh` files are executed and keep
  theirs. `docs/guides/shell-assumptions.md`.

## Database location (easy to get wrong)

The DB path comes from `nexus_path` in `~/.config/bip/config.yml`, **not** `~/.bipartite`.
The actual file:

```
$NEXUS_PATH/.bipartite/cache/refs.db
```

SQLite footgun: `CREATE ... IF NOT EXISTS` does **not** alter an existing table. After a schema
change you must rebuild the binary, `rm` that `refs.db`, then `./bip rebuild` — otherwise the
new schema silently won't take.

## Repo facts

- Agents decide and land changes here without asking the user, under downward pressure on size: a peer's suggested addition is a proposal to weigh, not a fix to apply, and deleting beats adding (`CONSTITUTION.md` Article VII).
- Every PR gets `/bip-pr-review` before it lands, skill-only PRs included; peer sessions' acks don't replace it. Skip the steps that don't apply and say so in the report.
- Owner is **`matsen/bipartite`**, not `matsengrp`. Use `matsen` in GitHub URLs and API calls.
- Continuation notes → `_ignore/CONTINUE-<role>.md` (gitignored); never commit. Written by `/bip-tuckin`, read by `/bip-continue`; see `docs/guides/continuation-prompt.md`.
- Secrets: the token and webhook getters in `internal/config/global.go` look up each name (`BIP_GITHUB_TOKEN`, `BIP_SLACK_TOKEN`, `BIP_ASTA_API_KEY`, `BIP_SLACK_WEBHOOK_<CHANNEL>`, plus fallbacks) in the environment, then in `~/.config/bip/secrets.env`, then in `config.yml`. Never print `secrets.env`. `config.yml` should hold no secrets, but an older one may still carry tokens or `slack_webhooks`, so don't print it whole: read the field you need.
- Per-issue git worktrees are opt-in via a `layout:` block in `~/.config/bip/config.yml`
  (per-repo overrides in `sources.yml`). Absent block = today's clone-per-repo behavior.
  Schema and precedence: `docs/guides/layout.md`. The resolver is `flow.ResolveRepoPath`.

## Paper lookups (guardrail)

Search locally first:

```bash
grep -i "name|keyword" "$NEXUS_PATH/.bipartite/refs.jsonl" | jq -r '.id + " - " + .title'
```

~6000 papers are already imported, so most relevant work is present. **Ask before any ASTA MCP
call** — ASTA is only for papers confirmed absent locally, citation/reference discovery, or
topic search with no local hit. To add a paper, ask the user to put it in Paperpile; adding it yourself
(`bip s2 add`) is discouraged.

## Docs conventions

- A skill or agent file carries procedure an agent must follow and facts it can't get by looking — no incident stories, dated measurements, or ⛔/⭐ markers; a check whose correct form looks wrong keeps a one-clause why (#262).
- `README.md` stays short (overview, install, env vars). Detailed guides live in
  `docs/guides/`. Skills live in `./skills/` (not `./.claude/skills/`).
- When a change adds or alters a command, run `./bip <cmd> --help` and make the skill docs
  match the real flags and workflow.
