# The Whole Rig

Tier 4 puts a talks session on top of the [tier 3](where-to-start.md#tier-3-fleet) fleet.
We call the result talk-driven development.
Results go on slides, you present them to the group, and the group's questions drive the next work.
Work is spun out to the main line once a result is worth keeping.
This replaces the manuscript-driven development of the [June post](https://matsen.group/general/2026/06/06/bipartite.html).
Talks match how scientists already work, and they bring people back into the room.

This page has four diagrams. The first shows the whole rig. The other three zoom in on an issue, a worker, and a fix to the skills.
In all four, people are in a humanist face with a purple outline, and agents are in typewriter.
The literature is in a dashed box, because it is knowledge, not a person or an agent.

## The rig

![Talk-driven development: four layers of knowledge, from research up to the field, with a support layer below](../images/tier4/talk-driven-development.svg)

Read upward, each layer is a wider circle that has to understand and accept a result.
These are layers of knowledge, not of authority.

- **Research.** The agents produce results. The epic session (`/bip-epic`) holds the current edge of one research question. The conductor (`/bip-conductor`) spawns a worker for each issue. The manuscript session (`/bip-ms`) stores each result with how it relates to prior work.
- **You.** Most of the work happens here, between you and the talks session (`/bip-talk`). Together you turn results into slides, and you have to understand each result well enough to explain it. A `/bip-helper` does the figure and deck work, so the talks session stays free to answer you.
- **The group.** You present the slides, and the group's questions come back. The talks session writes each question into the talk's `feedback.md`. It answers each one with a reproducible exploratory analysis, or sends it to the epic session as a draft issue.
- **The field.** The literature is the field's state of knowledge, and the goal is to improve it. The manuscript session reads the literature in. People write the paper that goes back out; the manuscript session keeps the results it is built from, not its final prose.

The support layer holds no results. It keeps the other layers running.

- Staff (`/bip-staff`) holds your rulings word for word. It answers what it can and pages you for the rest.
- A sysadmin session watches the machine, and an infra session (`/bip-infra`) watches the cluster.
- A librarian session (`/bip-librarian`) triages literature alerts and asks the epic owners about the few that matter.
- A bipartite session lands fixes to the skills (see [Kaizen](#kaizen)).

[Agent Roles](roles.md) says what each role owns.

## The life of an issue

![The life of an issue: formulate with ms and EPIC, check and file, work, land, and the result goes back](../images/tier4/issue-lifecycle.svg)

Many agents look at one issue before it lands.
An issue starts from a talk question, a PR's `DEFERRED` list, or you.
Before it is filed, its draft goes to two sessions:

- The manuscript session knows what the field knows. It can say whether published work already predicts the answer.
- The epic session knows the project's current edge. It can say how the issue fits the work in flight.

`/bip-issue-check` then reviews the draft, a skeptic tests its plan, and the issue is filed.
The conductor spawns a worker for it.
A self-contained PR lands after the epic session's claims check; a PR that changes shared code needs a review from a group member.
The result goes back to the epic session, which rules on it, to the manuscript session, which records it, and to the talks session, which presents it.
[Issue Lifecycle](issue-lifecycle.md) gives the skills in order.

## Inside a worker

![Inside a worker: the worker loop, the issue-lead, parallel review agents, and comment-check](../images/tier4/inside-a-worker.svg)

A worker owns one issue in one clone.
It loops: implement, run, report.
At each stopping point it spawns an issue-lead, which reads the files cold, with none of the worker's context. The issue-lead judges scope and results, and says whether to continue, fold deferred work in, or escalate.
Before merge, `/bip-pr-review` runs its review agents in parallel. The skeptic runs only when the PR makes a scientific claim, and the cold reader runs only when the PR adds a rendered notebook.
`/bip-comment-check` fact-checks each review comment against the code before the worker acts on it.
The epic session sends corrections through the conductor, so the message and the clone's worklog stay in step.
Once CI is green at the PR's head, the PR goes to its landing signer.

## Kaizen

![Kaizen: a proposal from any session goes to the bipartite session, is reviewed and landed, and reaches every session](../images/tier4/kaizen.svg)

You can be in any session when you hit friction or a bug.
Run `/bip-kaizen` there. It writes what happened, the root cause, and a proposed edit.
For a change to bipartite, the session sends the proposal to the bipartite session and quotes your words.
The bipartite session weighs it, lean by default: a rule must name a failure someone actually hit, and deleting beats adding.
It may push back, or ask for an A/B test first.
A cold reviewer compares the old file with the new one, and the session that sent the proposal acks the diff.
The fix lands once CI is green at the PR's head.
The skills directory is symlinked into `~/.claude/skills`, so every session in every repo uses the fix from its next skill invocation.
