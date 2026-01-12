# Continuation Prompt for Bipartite

## What is Bipartite?

An agent-first academic reference manager. See `VISION.md` for the complete design document.

Key points:
- CLI: `bp` (two letters, fast)
- Git-versionable: JSONL source of truth, ephemeral database rebuilt on demand (beads-inspired)
- Reference-manager agnostic: Paperpile JSON import first, extensible to others
- PDF access is a key design goal: agents can open papers for humans via `bp open <id>`
- BibTeX export for LaTeX integration

## Current State

- **Spec-kit installed**: `.claude/commands/` has speckit slash commands, `.specify/` has templates
- **VISION.md written**: Complete design document covering concept, architecture, CLI, phases, code quality
- **Ready for**: `/speckit.constitution` to formalize project principles

## Phased Roadmap

1. **Phase I**: Core reference manager (JSONL, CLI, Paperpile import, BibTeX export, PDF linking)
2. **Phase II**: RAG over abstracts
3. **Phase III**: Knowledge graph with relational summaries
4. **Phase IV**: ASTA/Semantic Scholar integration (including `bp add paper.pdf`)
5. **Phase V**: Discovery tracking ("discovered-from" provenance)

## Key Decisions Made

- **Internal schema**: Normalized JSONL, not raw Paperpile format. `source.type` and `source.id` for tracking origin.
- **DOI is primary deduplication key**: On re-import, match by DOI and replace. Paperpile `source.id` can change on re-import.
- **Citekeys from source**: Use Paperpile's citekeys. Citekey generation deferred until after Paperpile import works.
- **Supersedes relationship**: Published papers can mark `supersedes: <preprint-doi>`. Manual for Phase I, ASTA auto-detection in Phase IV.
- **Multi-PDF support**: `pdf_path` for main PDF, `supplement_paths` array for supplementary materials.
- **No `bp add <doi>`**: Publishers block automated access (403s). Add papers via reference manager, then import.
- **BibTeX over BibLaTeX**: Universal compatibility
- **Beads for orchestration**: Use actual beads for bulk import tasks
- **PDF readers**: Skim on macOS (with page targeting), Zathura/Evince/Okular on Linux
- **Compiled language preferred**: Fast CLI startup (Rust or Go likely)
- **Fully agentic TDD**: After spec is complete, agents write tests and code autonomously; human reviews at PR level
- **Beads for development**: Agentic loop orchestrated via beads

## Code Quality Standards

From user's other projects (see VISION.md for full details):
- Clean architecture (SRP, DIP, composition over configuration)
- Fail-fast philosophy (no silent defaults)
- Naming excellence (names reveal intent)
- Real testing (no fake mocks)
- Minimal dependencies

## Files to Reference

- `VISION.md`: Complete design document
- `.specify/memory/constitution.md`: Will contain project constitution after `/speckit.constitution`
- User's Paperpile export: `_ignore/paperpile-export-jan-12.json` (32MB, not committed)
- User's PDF folder: `/Users/matsen/Library/CloudStorage/GoogleDrive-ematsen@gmail.com/My Drive/Paperpile`

## Next Steps

1. Run `/speckit.constitution` to establish project principles
2. Run `/speckit.specify` to create baseline specification
3. Run `/speckit.plan` to plan Phase I implementation
4. Choose implementation language (fast CLI startup, good JSON handling)
5. Build Phase I
