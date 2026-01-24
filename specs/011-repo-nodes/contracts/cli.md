# CLI Contracts: Projects and Repos

**Feature**: 011-repo-nodes | **Date**: 2026-01-24

## Project Commands

### `bip project add <id>`

Create a new project.

**Arguments**:
- `<id>`: Project identifier (required)

**Flags**:
- `--name, -n <name>`: Display name (required)
- `--description, -d <text>`: Project description

**Output (JSON)**:
```json
{
  "status": "created",
  "project": {
    "id": "dasm2",
    "name": "DASM2",
    "description": "Distance-based antibody sequence modeling",
    "created_at": "2026-01-24T10:00:00Z",
    "updated_at": "2026-01-24T10:00:00Z"
  }
}
```

**Exit Codes**:
- `0`: Success
- `2`: Project ID already exists
- `3`: Validation error (invalid ID format, missing name)
- `4`: Data error (file I/O)

---

### `bip project get <id>`

Retrieve a project by ID.

**Output (JSON)**:
```json
{
  "id": "dasm2",
  "name": "DASM2",
  "description": "Distance-based antibody sequence modeling",
  "created_at": "2026-01-24T10:00:00Z",
  "updated_at": "2026-01-24T10:00:00Z"
}
```

**Exit Codes**:
- `0`: Success
- `2`: Project not found

---

### `bip project list`

List all projects.

**Output (JSON)**:
```json
{
  "projects": [...],
  "count": 3
}
```

---

### `bip project update <id>`

Update a project's metadata.

**Flags**:
- `--name, -n <name>`: New display name
- `--description, -d <text>`: New description

**Exit Codes**:
- `0`: Success
- `2`: Project not found
- `3`: Validation error (no flags provided)

---

### `bip project delete <id>`

Delete a project and all its repos.

**Flags**:
- `--force, -f`: Delete even if edges exist (cascade delete)

**Output (JSON)**:
```json
{
  "status": "deleted",
  "id": "dasm2",
  "repos_removed": 2,
  "edges_removed": 5
}
```

**Exit Codes**:
- `0`: Success
- `2`: Project not found
- `3`: Project has linked edges (use `--force`)

---

### `bip project repos <id>`

List repos belonging to a project.

**Output (JSON)**:
```json
{
  "project_id": "dasm2",
  "repos": [...],
  "count": 2
}
```

---

### `bip project concepts <id>`

List concepts linked to a project.

**Flags**:
- `--type, -t <rel>`: Filter by relationship type

**Output (JSON)**:
```json
{
  "project_id": "dasm2",
  "concepts": [
    {
      "concept_id": "variational-inference",
      "relationship_type": "implemented-in",
      "summary": "DASM2 uses VI for the latent space model"
    }
  ],
  "count": 1
}
```

---

### `bip project papers <id>`

List papers relevant to a project (transitive via concepts).

**Output (JSON)**:
```json
{
  "project_id": "dasm2",
  "papers": [
    {
      "paper_id": "10.1038/...",
      "via_concept": "variational-inference",
      "relationship_type": "introduces",
      "summary": "Foundational VI paper"
    }
  ],
  "count": 1
}
```

---

## Repo Commands

### `bip repo add <github-url-or-org/repo>`

Add a GitHub repository to a project.

**Arguments**:
- `<github-url-or-org/repo>`: GitHub URL or shorthand (e.g., `matsen/bipartite`)

**Flags**:
- `--project, -p <id>`: Project ID (required)
- `--id <id>`: Override derived repo ID

**Output (JSON)**:
```json
{
  "status": "created",
  "repo": {
    "id": "bipartite",
    "project": "bipartite",
    "type": "github",
    "name": "bipartite",
    "github_url": "https://github.com/matsen/bipartite",
    "description": "Agent-first academic reference manager",
    "topics": ["reference-manager", "cli"],
    "language": "Go",
    "created_at": "2026-01-24T10:00:00Z",
    "updated_at": "2026-01-24T10:00:00Z"
  }
}
```

**Exit Codes**:
- `0`: Success
- `2`: Project not found
- `3`: Validation error (missing project, invalid URL)
- `4`: Data error
- `5`: GitHub API error

---

### `bip repo add --manual`

Add a manual (non-GitHub) repository.

**Flags**:
- `--manual`: Create manual repo (no GitHub fetch)
- `--project, -p <id>`: Project ID (required)
- `--id <id>`: Repo ID (required for manual)
- `--name, -n <name>`: Display name (required)
- `--description, -d <text>`: Description
- `--topics <csv>`: Comma-separated topics

---

### `bip repo get <id>`

Retrieve a repo by ID.

---

### `bip repo list`

List all repos.

**Flags**:
- `--project, -p <id>`: Filter by project

---

### `bip repo update <id>`

Update a repo's metadata.

**Flags**:
- `--name, -n <name>`: New name
- `--description, -d <text>`: New description
- `--topics <csv>`: New topics

---

### `bip repo delete <id>`

Delete a repo.

---

### `bip repo refresh <id>`

Re-fetch GitHub metadata for a repo.

**Exit Codes**:
- `0`: Success
- `2`: Repo not found
- `3`: Repo is manual (no GitHub URL)
- `5`: GitHub API error

---

## Edge Command Extensions

### `bip edge add` (existing, extended)

**New validation**:
- Reject if source or target contains `repo:` prefix
- Reject if connecting unprefixed paper ID to `project:` prefix
- Accept `concept:` ↔ `project:` edges

**New error message**:
```
error: cannot create edge to/from repo (repos have no edges)
error: cannot create paper↔project edge directly (must go through concept)
```

### `bip edge list` (existing, extended)

**New flags**:
- `--project, -P <id>`: Filter edges involving a project

---

## Check Command Extensions

### `bip check` (existing, extended)

**New validations**:
- All repos reference valid projects
- No orphaned project edges (project deleted but edge remains)
- No invalid paper↔project or *↔repo edges

**Output additions**:
```json
{
  "orphaned_project_edges": [...],
  "invalid_repo_edges": [...],
  "repos_with_missing_projects": [...]
}
```

---

## Rebuild Command Extensions

### `bip rebuild` (existing, extended)

Rebuilds SQLite index including:
- `projects` table from `projects.jsonl`
- `repos` table from `repos.jsonl`

---

## Common Output Formats

All commands support:
- `--json` (default): JSON output
- Human-readable output when TTY detected

Human-readable format examples:

```
# bip project get dasm2
Project: dasm2
Name:    DASM2
Desc:    Distance-based antibody sequence modeling
Created: 2026-01-24T10:00:00Z

# bip project repos dasm2
Repos for project: dasm2

  dasm2-code (github)
    https://github.com/matsen/dasm2
    Python · antibodies, ml

  dasm2-paper (github)
    https://github.com/matsen/dasm2-paper
    LaTeX · manuscript

Total: 2 repos
```
