---
name: bip-talk
description: Cold-start for a talks session — the live layer of a project. It presents current results to the group as talks, collects the questions raised, answers them with reproducible exploratory analyses, and, once the group is satisfied with an analysis, has it formalized on the main line of the codebase. Like a continuing group meeting.
---

# /bip-talk

Cold-start for a talks session.
Run from the clone that holds the project's `talks/` directory, on the current talk's branch.
The session is live: results go on slides, the group questions them, and the answers settle on a slide, become issues, or are formalized on main.
The deck is how current results are presented and made understandable to the group. The manuscript session keeps the project's long-term memory and its connection to the literature, and the EPIC session is the current edge of the work; the talks session presents what they hold to the group.

## Role

The session owns `talks/` and the talk branch: decks, figures and their scripts, `feedback.md`, and `feedback-scripts/`; no one else commits there.
There is one talk branch and PR at a time, holding every talk since the last landing; it stays draft until the user asks to land it, and its body is the session's to keep current. Landing steps are in the repo's `talks/AGENTS.md`; land through `/bip-pr-land`, whose Step 5.5 waits for a build or test check to succeed at the head after the PR leaves draft. The user's word to land, `--admin` included, waives review, not CI.
The project's EPIC session rules on numbers and files issues (it runs `/bip-issue-check`); send it a draft path, not a filed issue.
Unlike `/bip-ms`, this session runs the computations behind its responses, itself or through a helper, under the compute rules in `~/.claude/CLAUDE.md` and any gate its conductor relays.
Questions for the user go as `~/.claude/CLAUDE.md` says ("Asking the user").

## Conventions

The repo's own `talks/AGENTS.md` holds what is particular to it: the build command, where decks are served, the title scheme, the slide style for figures, render gotchas.
Everything here holds across repos.

- **One dated directory per talk** (`YYYY-MM-DD-<name>/`): slide source, built deck and PDF, `figures/` (copies, so the record stands alone), `notes/`, `feedback.md`, `feedback-scripts/`.
  A deck is the record of the talk as given, apart from notes marked as added afterwards; later work goes into the next talk.
- **Define each new term on a slide, before its first use.** A new statistic, model or term gets its own lead-in slide. Write a regression out in full, with one bullet per symbol. Explain a derived quantity (such as a per-target level) as step-by-step bullets revealed one at a time, each with a one-sentence gloss. Name each quantity for what it computes ("the slope on log codon count", not "tilt"), and settle the names before building the slides that use them. Speaker notes are not enough for a definition; they hold how to read each figure, the numbers behind it, and links to scripts and issues.
- **`feedback.md`** has two `#` groups, **Quick answers** and **For follow-up**.
  One `##` per point, phrased as a question, with the asker in parentheses (the user's own points last); then a `**Response.**` paragraph in full prose, which may push back: the take-home and "see slide N", with the long version in `feedback-scripts/`.
  Each follow-up ends with `**For an issue.**`, saying what work would answer it, for the EPIC session to file.
- **`feedback-scripts/qN-<topic>/`** keeps every computed response reproducible: the scripts that ran, small outputs with `outputs.sha256`, `run.sh` (host or job id, code commit, input paths, as runnable commands), and `evidence.md` (the question, the response, and the source of each number). No home or scratchpad paths, no large files; scripts need to run, not to be polished.
- **Analyses on the talk branch are exploratory**: `feedback-scripts/` and figure scripts run unreviewed so they iterate fast, and they are not the codebase.

## Discipline

- Put a skeptic (`surprising-conclusion-skeptic`) on each follow-up candidate before drafting its issue: it catches misattributions and duplicates of open issues.
- A new or changed figure is drawn in the slide style the repo's `talks/AGENTS.md` names, and gets a `plot-reviewer` pass before it goes in a deck. Give the reviewer the script, the rendered figure, and the slide as rendered in the deck PDF, since it has to judge the size the figure is projected at.
- Re-derive every number a peer sends before it reaches a slide, and keep PROVISIONAL numbers off slides.
- Keep a hold list (never present X until issue Y reads), each hold with its FIRST CHECK command, in the continuation file.
- When a port's review changes a number, the slide and its notes change only after the EPIC session rules on it.

## Figure and deck work

All figure and deck work goes through one long-lived `/bip-helper`, never inline, so the session stays free to answer the user.

- Start its worktree off `origin/<talk-branch>` in place of `origin/main`.
- Each round, the helper runs `git fetch && git reset --hard origin/<talk-branch>`, does the work, and pushes a fresh helper branch.
- The session fast-forwards or cherry-picks that branch onto the talk branch, pushes, rebuilds the served deck, and deletes the helper's remote branch. Deleting it lets the next round push without force after the session commits on its own.
- The helper's worktree is under `~/.local`, which snap chromium cannot read, so a DeckTape PDF build there fails. The helper renders from a copy under `~/`.
- For a clean context, run `/bip-helper` stop steps 1–2 and 6, so the home and worktree stay but the old record goes. Then run its start steps 3–5 on the same `DIR` with the brief.

## The cycle

1. After a talk, the user relays the audience's questions, and the session writes `feedback.md`: quick answers first.
2. Then the follow-ups, slowly, one candidate at a time, with a skeptic early. Each ends filed (through the EPIC session), answered by an existing issue or PR, converged to no experiment (the user confirms no-file), dropped as a duplicate, or gated on another issue's read.
3. What is settled goes into the next talk's deck, and the user reviews the rebuilt deck where it is served.
4. The user signs off on `feedback.md` and on the figures.
5. Once the group is satisfied with an analysis (the user's call), it is formalized on main: ask the EPIC session for an issue that ports its script and analysis with tests and PR review, and a worker does the port. The talk keeps its copy as the record, and its notes point at main once the port lands.

## Cold start

1. Check out and pull the talk branch; read `talks/AGENTS.md`, the current talk's source and `feedback.md`, and the talk PR's body.
2. Read the EPIC body for the project's current state.
3. Run each hold's FIRST CHECK; report what has read and what still holds.
4. Take the continuation file's in-flight items, if any, as the user's direction; otherwise list the follow-ups not yet settled and what the next talk needs.

Tuck in with `/bip-tuckin` (the generic fallback), naming `/bip-talk` as the cold-start in the continuation file.

## A new project's talks/

Copy `talks/tools/` from a repo that has one (matsengrp/dasm2-experiments: a pandoc reveal.js build with a DeckTape PDF and an index page), and write `talks/AGENTS.md` with that repo's build, serving, title and slide-style facts.
