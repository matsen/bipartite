# bipartite Design Decisions

Technical architecture decisions for the bipartite project. These are
load-bearing choices — changing them has wide impact.

## I. JSONL Source of Truth

All persistent data lives in JSONL files (refs, projects, repos). SQLite indexes are ephemeral — rebuilt from JSONL via
`bip rebuild`. Schema changes require deleting the database file and
rebuilding, never migration scripts.

JSONL is chosen for git-mergeability. This is a load-bearing decision.

## II. Pure Go, No CGO

`go build` MUST work without a C compiler. Use modernc.org/sqlite (pure
Go SQLite), golang.org/x/crypto/ssh (native SSH). No C dependencies.

## III. Local-First, Cloud-Optional

Core functionality works offline with local data. External APIs
(Semantic Scholar, ASTA, GitHub, Slack) are optional enrichment.
Embeddings via local Ollama, not cloud embedding services. bip never
phones home.

## IV. Nexus Pattern

All user data lives in a single git-backed nexus repository. Path comes
from `~/.config/bip/config.yml` → `nexus_path`. The nexus is a git
repo — all data is versioned and shareable. bip MUST NOT write outside
the nexus or its own config directory.

## V. Git-Compatible Collaboration

JSONL format enables standard git merge workflows. `bip resolve` handles
domain-aware conflict resolution (DOI-based deduplication, complementary
metadata merging). `bip dedupe` cleans up post-merge duplicates. Design
for the case where multiple people push to the same nexus.
