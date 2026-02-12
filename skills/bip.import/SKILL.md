---
name: bip.import
description: Import Paperpile JSON export into bip library, rebuild database, and clean up.
---

# Import Paperpile Library

Import a Paperpile JSON export from `~/Downloads/` into the bip reference library.

## Usage

```bash
/bip.import              # Auto-detect latest Paperpile JSON in ~/Downloads
/bip.import ~/path.json  # Explicit file path
```

## Workflow

### 1. Find the export file

If no path argument given, look for `~/Downloads/Paperpile*.json` or `~/Downloads/*.json`:

```bash
ls -lt ~/Downloads/Paperpile*.json 2>/dev/null || ls -lt ~/Downloads/*.json
```

If multiple files match, ask the user which one.

### 2. Dry-run import

Always dry-run first to show what will change:

```bash
cd ~/re/nexus
bip import --format paperpile "<file>" --dry-run --human
```

Report the counts (new, updated, skipped) to the user. The skipped entries are typically old Paperpile records missing year or author — expected and not actionable.

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
