---
name: bip-inbox
description: A conductor's GitHub inbox — find what on its fleet's repo needs action and act on it. Spawns a review window for each PR whose review is requested of the user, and forwards each human comment or review on a fleet PR to the slot that owns it, or to its EPIC owner. Run from a conductor's directory, on demand or from /bip-conductor's scan and monitor.
---

# /bip-inbox

Run from the conductor's directory:

```bash
<this skill's dir>/inbox .epic-config.json
```

It prints one JSON line per item (its header has the fields) and records what it has seen in `$CLONE_ROOT/.inbox-seen.json`, so each comment or review comes out once. Act on every line:

- **`review-request`**: spawn a review window, `r<N>-<slot>`, as `/bip-conductor`'s intake says for a code review of someone else's PR. The request is the user's go, so don't ask first. With no free clone, make one as `/bip-conductor-spawn`'s "Creating new slots" says. It comes back on each run until that window exists.
- **`comment` or `review` with a `slot`**: append it to that slot's `.epic-worklog.md`, under the time it was posted, giving the author, `url`, `state` and the `body` verbatim (its first 2000 characters; `url` has the rest). When `slot` names several clones on the same branch, the one with a live window owns it. Then message the slot's session with the URL and one line on what it asks. A review's inline comments are not in `body`: `gh api repos/<repo>/pulls/<N>/comments`.
  An `APPROVED` or `CHANGES_REQUESTED` review also goes to the EPIC owner, since it can open or close the landing gate.
- **`comment` or `review` with no slot** (the PR landed, or its slot was reclaimed): send it to the owner of the EPIC whose body links the PR's issue, to fold in or file as a follow-up. Never drop it.

A 🤖 comment from another person's account is that person's agent writing for them; forward it like their own.
