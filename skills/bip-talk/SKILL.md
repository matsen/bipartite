---
name: bip-talk
description: Cold-start for a talks session — talk-driven development, the stage before a paper. The session owns a project's talk decks, the questions raised at each talk with their responses, and the scripts behind those responses, so preliminary results get presented, questioned and made understandable like a continuing group meeting. A lighter companion to /bip-ms.
---

# /bip-talk

Cold-start for a talks session.
Run from the clone that holds the project's `talks/` directory, on the current talk's branch.
The group writes a talk before a paper: results go on slides, the audience questions them, and the answers either settle on a slide or become issues.
The current deck is the project's best statement of where things stand, the way the paper is for `/bip-ms`.

## Role

The session owns `talks/` and the talk branch: decks, figures and their scripts, `feedback.md`, and `feedback-scripts/`; no one else commits there.
The project's EPIC session rules on numbers and files issues (it runs `/bip-issue-check`); send it a draft path, not a filed issue.
Unlike `/bip-ms`, this session runs the computations behind its responses, itself or through a helper, under the compute rules in `~/.claude/CLAUDE.md` and any gate its conductor relays.
Questions for the user go as `~/.claude/CLAUDE.md` says ("Asking the user").

## Conventions

The repo's own `talks/AGENTS.md` holds what is particular to it: the build command, where decks are served, the title scheme, render gotchas.
Everything here holds across repos.

- **One dated directory per talk** (`YYYY-MM-DD-<name>/`): slide source, built deck and PDF, `figures/` (copies, so the record stands alone), `notes/`, `feedback.md`, `feedback-scripts/`.
  A deck is the record of the talk as given, apart from notes marked as added afterwards; later work goes into the next talk.
- **Speaker notes cover each slide completely**: how to read the figure, how each statistic is computed, the numbers behind it, and links to scripts and issues. The slides carry little text.
- **`feedback.md`** has two `#` groups, **Quick answers** and **For follow-up**.
  One `##` per point, phrased as a question, with the asker in parentheses (the user's own points last); then a `**Response.**` paragraph in full prose, which may push back.
  Each follow-up ends with `**For an issue.**`, saying what work would answer it, for the EPIC session to file.
- **`feedback-scripts/qN-<topic>/`** keeps every computed response reproducible: the scripts that ran, small outputs with `outputs.sha256`, `run.sh` (host or job id, code commit, input paths, as runnable commands), and `evidence.md` (the question, the response, and the source of each number). No home or scratchpad paths, no large files; scripts need to run, not to be polished.
- **Figures iterate on the talk branch without review.** Once one carries a claim the project will keep making, ask the EPIC session for an issue that ports its script to main with tests and review; the talk keeps its copy, and its notes point at main once the port lands.

## Discipline

- Put a skeptic (`surprising-conclusion-skeptic`) on each follow-up candidate before drafting its issue: it catches misattributions and duplicates of open issues.
- Re-derive every number a peer sends before it reaches a slide, and keep PROVISIONAL numbers off slides.
- Keep a hold list (never present X until issue Y reads), each hold with its FIRST CHECK command, in the continuation file.
- A number that touches a review finding goes to this session and the EPIC session before it changes.

## Cold start

1. Read `talks/AGENTS.md`, the current talk's source and `feedback.md`, and the talk PR's body.
2. Read the EPIC body for the project's current state.
3. Run each hold's FIRST CHECK; report what has read and what still holds.
4. List the open follow-up points without a filed issue, and what the next talk needs.

Tuck in with `/bip-tuckin` (the generic fallback), naming `/bip-talk` as the cold-start in the continuation file.

## A new project's talks/

Copy `talks/tools/` from a repo that has one (matsengrp/dasm2-experiments: a pandoc reveal.js build with a DeckTape PDF and an index page), and write `talks/AGENTS.md` with that repo's build, serving and title facts.
