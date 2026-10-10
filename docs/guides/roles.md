# Agent Roles

A bipartite fleet is a set of Claude sessions, each with one role; every role but the worker is long-lived.
A role fixes what a session owns, whom it answers, and which skill starts it.
This page names the roles; the skills carry the procedures.

| Role | How many | Owns | Starts with |
|---|---|---|---|
| Conductor | one per fleet (repo + clone pool) | ops: slots, spawns, hosts, landing mechanics | `/bip-conductor` |
| Epic | one per live EPIC, one EPIC each | the EPIC: its body, its direction, the current edge of its topic | `/bip-epic <N>` |
| Manuscript (ms) | one per paper | the paper: broad perspective, literature, what the results mean for it | `/bip-ms` |
| Staff | one per site | adjudication, the user's rulings, paging the user | `/bip-staff` |
| Worker | one per issue in flight | one issue's branch and PR | spawned by the conductor |

Discussants (an interpretation or literature pal, a brainstorm partner) are generic sessions attached to an epic or ms session; they tuck in with the generic fallback, put their reasoning on the issues it bears on, and escalate to staff directly.
A `/bip-helper` session is an extension of its primary, not a role.

## Conductor: ops

The conductor runs the fleet: which clone holds which issue, what runs on which host, what is free, what lands.
It narrates fleet state and nothing else; why an issue matters is the epic's to say.
Its reports never ask the user a question: each question goes to the session that owns its subject, and the report lists it as `pending with <owner>`.
Workers reach it through `$CLONE_ROOT/.conductor-session`, which it rewrites every cycle.

## Epic: one EPIC, the current edge

An epic session owns exactly one EPIC.
It keeps the EPIC body current, drafts spawn briefs for its EPIC, and judges whether its workers' results answer the question the EPIC asked.
Its questions to the user go directly when the user is in its window, otherwise through staff.

**One EPIC per epic session is a bright line.**
A session that finds itself wanting a second EPIC has found either two EPICs that should merge, or a second topic that needs its own session.
Issues already filed under the wrong EPIC stay there until the user names an owner for their topic.

The EPIC body names its owner and what it feeds, each entry a `repo#N` or an exact ms session name:

```
Owner: <session name>
Feeds: <repo#N>, <ms session name>
```

The owner re-asserts `Owner:` at each cold start and pushes a body only while `Owner:` names it.

## Routing notices

A worker's brief freezes `EPIC: <owner/repo>#<N>`, or `EPIC: none`, and may add `Notify:` with the exact names of the sessions that requested or framed the issue.
A follow-up issue inherits both lines from its parent.
Notices go to the EPIC's `Owner:`, read when the notice is sent, plus every `Notify:` name.
The conductor hears every phase transition through `.epic-notifications.log` either way.

Names match exactly.
A missing `Owner:` line, or a name with no live session, is never routed to some other session: the owner's copy stays in `.epic-notifications.log`, and the conductor reports the EPIC to staff. `Notify:` names still receive theirs.
Staff proposes an owner to the user, usually the session already producing the EPIC's edits; staff never stands in as the owner.

The 🤖 landing signer is landing policy, set in the brief's `LANDING DELEGATION:` line; it defaults to the EPIC's owner but need not be. A brief with `EPIC: none` names its signer, or its PR waits for a human merge.

## Claims: no self-judging

An epic rules on how to test — design, procedure, the calls inside a pre-registration — and on whether a result answers the question asked.
It does not rule on whether its own hypothesis holds.
A result claim goes into the EPIC body, or to the user, only after a session that did not frame it has re-derived the decisive fact from code or data and named the files or lines: `surprising-conclusion-skeptic` by default, a worker's PR-level skeptic, a reviewer session the EPIC body names, or the user.
Agreeing with the reasoning is not a check, and neither is relaying another session's reading.
The body entry names its checker.
A hypothesis labelled unmeasured needs no check of its inference, but any checkable fact it rests on is re-derived the same way before it reaches the EPIC body or the user.

A session that receives a user ruling on an EPIC's subject relays it to that EPIC's `Owner:`, quoted with window and date, and the owner writes it into the body.

### From brainstorm to epic

Many threads start as a brainstorm in an ad-hoc session.
When one becomes a Thing, file or revive its EPIC, set `Owner:` to that session, and run `/bip-epic <N>` in it.
The session keeps its context, which is why the no-self-judging rule matters most here.

## Manuscript: the broad view

An ms session owns one paper.
It is the project's resource for questions and guidance, from what the project and the field know: it holds the perspective across EPICs and repos, knows the literature (`/bip-lit`), and decides what results mean for the paper, which follows from that perspective.
An epic consults the papers in its `Feeds:` line before it changes direction; they are consulted, not a gate.
An ms session owns no EPIC; when a paper session finds itself running one, that EPIC needs its own epic session.

## Staff: adjudication

Staff is the last layer before the user: it rules on worth and on cleanliness, records the user's rulings verbatim, and is the one session that pages the user.
Process and ops questions are settled among the agents, by their owner or by staff; design, claims, and what gets built go to the user.
A staff call is labelled a staff ruling: it is never recorded as the user's and never authorizes a land.

## Posting to GitHub

Every agent posts as the user, so what it may post depends on whose work it is, for PRs and issues alike (user, 2026-10-10):

- **Work by an agent working as the user**: post freely.
- **Another human's operational work** (tooling, pipelines, CI, docs): post directly, reviews and approvals included, typically after `/bip-comment-check`. An approval from the user's account never counts as the group review (`/bip-conductor`'s "Who may land").
- **Another human's scientific work**, which produces or changes a result, claim or analysis that could reach a paper: post nothing until the user has discussed it. A mixed or unclear case is scientific.

These are defaults: an explicit instruction in the request or brief, such as a landing-gate brief's "don't post", wins.

## Names

Session names describe topics, not EPIC numbers, so they survive an EPIC being superseded: `phyz-search`, `dasm2-neutral`, `pcp-ms`.
Each name is unique among the sessions `ListAgents` shows, because routing matches it exactly, and the session's tmux window carries the same name.
Renaming a session edits every `Owner:` line that names it, in the same step.
