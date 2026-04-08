---
name: bip.lit.import
description: Import references from Paperpile or Zotero into bip library, rebuild database, and clean up.
---

# Import Reference Library

Import references from Paperpile JSON or Zotero CSL-JSON exports into the bip reference library.

## Usage

```bash
/bip.lit.import                     # Auto-detect latest export in ~/Downloads
/bip.lit.import ~/path.json         # Explicit file path
/bip.lit.import zotero ~/path.json  # Explicit format + file
```

## Format Detection

If the format is not specified, inspect the JSON to determine it:
- Contains `_id` and `citekey` fields → **Paperpile**
- Contains `type` and `issued` fields → **Zotero** (CSL-JSON)

## Workflow

### 1. Find the export file

If no path argument given, look for recent JSON exports:

```bash
ls -lt ~/Downloads/Paperpile*.json ~/Downloads/*.json 2>/dev/null | head -5
```

If multiple files match, ask the user which one.

### 2. Dry-run import

Always dry-run first to show what will change:

```bash
# Paperpile
bip import --format paperpile "<file>" --dry-run --human

# Zotero CSL-JSON
bip import --format zotero "<file>" --dry-run --human

# Zotero with PDF resolution (reads Zotero's local database for PDF paths)
bip import --format zotero --zotero-db ~/Zotero/zotero.sqlite "<file>" --dry-run --human
```

Report the counts (new, updated, skipped) to the user.

### 3. Import

```bash
# Paperpile
bip import --format paperpile "<file>" --human

# Zotero
bip import --format zotero "<file>" --human
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

Summarize: new refs added, total count, file deleted. Notes are preserved and searchable via `bip search`.

## Zotero-Specific Notes

- Export from Zotero: right-click a collection → Export Collection → CSL-JSON format
- Install the Better BibTeX plugin for cleaner citation keys in exports
- PDF resolution requires `--zotero-db` pointing to `~/Zotero/zotero.sqlite` (close Zotero first)
- Set `pdf_root: ~/Zotero/storage` in `.bipartite/config.yml` for PDF access
