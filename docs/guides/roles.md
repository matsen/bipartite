# Agent Roles

A bipartite fleet is a set of long-lived Claude sessions, each with one role.
A role fixes what a session owns, whom it answers, and which skill starts it.
This page names the roles; the skills carry the procedures.

| Role | One per | Owns | Starts with |
|---|---|---|---|
| Conductor | fleet (repo + clone pool) | ops: slots, spawns, hosts, landing mechanics | `/bip-conductor` |
| Epic | live EPIC | the EPIC: its body, its direction, the current edge of its topic | `/bip-epic <N>` |
| Manuscript (ms) | paper | the paper: broad perspective, literature, what the results mean for it | `/bip-ms` |
| Staff | site | adjudication, the user's rulings, paging the user | `/bip-staff` |
| Worker | issue | one issue's branch and PR | spawned by the conductor |

Discussants (an interpretation or literature pal, a brainstorm partner) are generic sessions attached to an epic or ms session; they tuck in with the generic fallback and put their reasoning on the issues it bears on.

## Conductor: ops

The conductor runs the fleet: which clone holds which issue, what runs on which host, what is free, what lands.
It narrates fleet state and nothing else; why an issue matters is the epic's to say.
Its reports never ask the user a question: each question goes to the session that owns its subject, and the report lists it as `pending with <owner>`.

## Epic: one EPIC, the current edge

An epic session owns exactly one EPIC.
It keeps the EPIC body current, drafts spawn briefs for the conductor, judges whether its workers' results answer the EPIC, and is where the topic's questions to the user come from (through staff).

**One EPIC per epic session is a bright line.**
A session that finds itself wanting a second EPIC has found either two EPICs that should merge, or a second topic that needs its own session.

The EPIC body names its owner and what it feeds:

```
Owner: <session name>
Feeds: <repo#N>, <paper>
```

Workers and the conductor read `Owner:` when they send a notice, not when the worker spawns, so an ownership change takes effect at once.
The match is on the exact session name; a name with no live session is reported, never routed to some other session.
An EPIC with no live owner keeps its notices on file, and the conductor reports it to staff, who finds or names an owner; staff never stands in as the owner.

An epic does not rule on its own hypotheses.
A result claim goes into the EPIC body only after someone other than the session that framed it has checked it: `surprising-conclusion-skeptic`, a paper the EPIC feeds, or the user.

### From brainstorm to epic

Many threads start as a brainstorm in an ad-hoc session.
When one becomes a Thing, file or revive its EPIC, set `Owner:` to that session, and run `/bip-epic <N>` in it.
The session keeps its context; nothing is handed over.

## Manuscript: the broad view

An ms session owns one paper.
It holds the perspective across EPICs and repos, knows the literature (`/bip-lit`), and decides what results mean for the paper.
An epic consults the papers in its `Feeds:` line before it changes direction, and a paper may check an epic's result claim.
An ms session owns no EPIC; when a paper session finds itself running one, that EPIC needs its own epic session.

## Staff: adjudication

Staff is the last layer before the user: it rules on worth and on cleanliness, records the user's rulings verbatim, and is the one session that pages the user.
Every session escalates a user-owned question to staff; staff settles it or pages.

## Names

Session names describe topics, not EPIC numbers, so they survive an EPIC being superseded: `phyz-search`, `dasm2-neutral`, `pcp-ms`.
Each name is unique across the machine, because routing matches it exactly, and the session's tmux window carries the same name.
