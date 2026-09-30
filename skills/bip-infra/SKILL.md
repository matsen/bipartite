---
name: bip-infra
description: Cold-start a long-lived infra session that tracks compute hosts and outages for the whole fleet — verifies outage reports independently, broadcasts status and policy to conductors, answers footprint questions, and keeps the site's stable host facts in its docs. Start it in the repo that holds the site's cluster docs.
---

# /bip-infra

The infra session is where other sessions send host trouble, and where the user asks "what's up, what's down, where can this run".
It is long-lived: start it once, in the repo that documents the site's compute (a wiki, an ops repo), and leave it running.
Other sessions reach it by name, so ask the user to `/rename infra` if it is not already named that.

## Where site facts come from

This skill carries no site facts. The session learns the site from:

- **`servers.yml`** (`$NEXUS_PATH/servers.yml`): the hosts and the jump host. `bip scout` reads it (`bip scout --help`).
- **The site doc**: the page in this repo that describes those hosts. Rank pages by how many distinct `servers.yml` hosts they mention (`git grep -l <host>` per host, counted; not `-w`, since docs write names as `__orca01__` and `_` is a word character).
  Read the top page fully before answering anything, and skim the runners-up for stale duplicates.
  If none exists, say so and offer to start one.
- **Live probes**: `bip scout`, plus the checks below. Load and memory are measured, never written down; hardware may go in the site doc with an as-of date.

## Checking a report

Other sessions report outages; confirm each one independently before relaying it.

1. **Probe through the jump host**, every host in parallel, each bounded: `timeout 25 ssh -o BatchMode=yes <host> uptime`.
   `bip scout` reports a jump-host throttle as `OFFLINE`, so never report an outage from scout alone (`/bip-scout`).
2. **Probe from a live host inside the network** (`ping`, and `timeout 5 bash -c '</dev/tcp/<host>/22'`).
   The second view separates *host down* from *path through the jump host broken*, and *rejects auth* from *unreachable*.
3. A host the user or the site doc names that is missing from `servers.yml` gets added, whether or not it is up (`getent hosts <host>` on an inside host confirms it exists); a host in `servers.yml` that no longer exists is reported to the user.

## Telling the fleet

Broadcast only when the user asks. Send one notice to each conductor — the `ListAgents` sessions named `*-conductor` — with:
what is down, what is up and how loaded, the policy the user set (e.g. pause big work, don't crowd the remaining shared hosts), and "pass this to your workers; send questions to infra".

Keep a list of who was told and who acknowledged.
A conductor that doesn't acknowledge is reported to the user by name, with an offer to message its workers directly — conductors own their workers, so infra doesn't go around one on its own.
When the situation changes (a host returns, the pause lifts), notify the same list.

## Footprint reports

Conductors report what they run where. Spot-check the target host's load and memory, reply only when something must change, and keep the report in the notify list.

## Rechecking

While a host is down, recheck it on a schedule (`/loop`, a few hours apart). Stop once the cause and a return estimate are known, and say so.

## Finding other capacity

When asked where else work could run, use read-only queries and dry-run submission only.
For SLURM: `sinfo`, `scontrol show partition`, `sacctmgr show qos`, `sshare`, `sacct` history, and `sbatch --test-only`, which is accepted and returns a start estimate without queueing anything.

## Stable facts and live status

- **Stable facts go in the site doc:** which hosts exist, who owns them, whose capacity is borrowed, what is shared with other groups, how to reach each, and the queue or partition rules.
  Edit it on a branch or as a draft, and have the user review before it lands.
- **Live status stays in the conversation:** what is down right now, load, who is paused.
  It is stale within days, and a stale status in a shared doc misleads everyone who reads it.
