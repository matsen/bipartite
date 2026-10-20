---
name: bip-lit-import
description: Import Paperpile JSON export into bip library, rebuild database, and clean up.
---

# Import Paperpile Library

Import a Paperpile JSON export from `~/Downloads/` into the bip reference library.

## Usage

```bash
/bip-lit-import              # Auto-detect latest Paperpile JSON in ~/Downloads
/bip-lit-import ~/path.json  # Explicit file path
```

## Workflow

### 0. Refuse a stale binary

A stale `bip` may replace refs instead of merging them, silently dropping fields such as `pmcid`, and nothing records which binary ran. Go only when the first command prints `vcs.revision=<SHA>` matching the second command's SHA, and also prints `vcs.modified=false`; a binary with no `vcs` lines counts as stale. If they differ, rebuild from main in the clone the binary links to (`readlink -f "$(command -v bip)"`), then re-run both.

```bash
go version -m "$(command -v bip)" | grep -E 'vcs\.(revision|modified)'
git ls-remote https://github.com/matsen/bipartite refs/heads/main | cut -f1
```

### 1. Find the export file

If no path argument given, look for `~/Downloads/Paperpile*.json` or `~/Downloads/*.json`:

```bash
ls -lt ~/Downloads/Paperpile*.json 2>/dev/null || ls -lt ~/Downloads/*.json
```

If multiple files match, ask the user which one.

### 2. Dry-run import

Always dry-run first to show what will change:

```bash
bip import --format paperpile "<file>" --dry-run --human
```

Report the counts (new, updated, skipped, warnings) to the user.
By default, entries missing required fields (title, author, published.year) are imported with sentinel placeholders (`[no title]`, `Unknown`, year `0`) and listed under `warnings` so they're not silently dropped — review the warning list and fix the affected Paperpile records when convenient.
Entries with no useful metadata at all (no title, author, year, or DOI — typically Paperpile auto-stubs for unparsed web pages) are still skipped and counted under `skipped`.
Pass `--strict` to revert to the old behavior of dropping any entry with a missing required field.

### 3. Import

```bash
bip import --format paperpile "<file>" --human
```

### 4. Rebuild database

```bash
bip rebuild --human
```

### 5. Delete the export file

```bash
rm "<file>"
```

### 6. Report

Summarize: new refs added, total count, file deleted.
Notes from Paperpile are preserved and searchable via `bip search`.

### 7. Recommend cleanup if warnings fired

If the import reported a non-zero `warnings` count, recommend the cleanup query so the user can fix the upstream Paperpile records when convenient:

```bash
bip search --tag paperpile:incomplete
```

Every reference imported with a sentinel field (`[no title]`, `Unknown`, year `0`) is auto-tagged `paperpile:incomplete`.
Once the user fixes the entry in Paperpile and re-imports, the next import replaces the stored reference and the tag disappears automatically, so the list shrinks until empty without any manual untagging step.
