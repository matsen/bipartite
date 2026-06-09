# bipartite — Claude guidance

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
  CGO). Embeddings: Ollama (local), pure-Go vector storage. External refs: Semantic Scholar
  (`internal/s2`).
- Data model: JSONL is the source of truth → ephemeral SQLite, rebuilt on `bip rebuild`. The
  vector index is GOB-serialized, ephemeral, gitignored.
- `cmd/` CLI commands · `internal/` packages (s2, store, index, flow, …) · `testdata/`
  fixtures · `tests/` integration tests.

## Build & style

- `go build -o bip ./cmd/bip && ./bip --help`
- Run `go fmt ./...` and `go vet ./...` before any PR. Exported symbols get doc comments.

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

- Owner is **`matsen/bipartite`**, not `matsengrp`. Use `matsen` in GitHub URLs and API calls.
- Continuation notes → `_ignore/CONTINUE.md` (gitignored); never commit.
- Secrets: `config.GetGitHubToken()` / `config.GetSlackBotToken()` consult env vars before
  `~/.config/bip/config.yml`. Order — GitHub: `BIP_GITHUB_TOKEN` → `GITHUB_TOKEN` → `GH_TOKEN`
  → config; Slack: `BIP_SLACK_TOKEN` → `SLACK_BOT_TOKEN` → config. Prefer the `BIP_*` names,
  sourced from a secrets manager (e.g. 1Password `op run` / `op read`).
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
topic search with no local hit. Add via `./bip s2 add DOI:...` when rate limits allow.

## Docs conventions

- `README.md` stays short (overview, install, env vars). Detailed guides live in
  `docs/guides/`. Skills live in `./skills/` (not `./.claude/skills/`).
- When a change adds or alters a command, run `./bip <cmd> --help` and make the skill docs
  match the real flags and workflow.
- The pre-PR process *is* the `/bip-pr-review` skill (with `/bip-pr-check` as a quick gate) —
  don't restate the checklist here.
