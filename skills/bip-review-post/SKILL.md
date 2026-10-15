---
name: bip-review-post
description: Post a review window's findings as the user's GitHub review — approve unless the user already has, distill _ignore/REVIEW-p<N>.md into a comment, fact-check it with /bip-comment-check, post it, and tell the conductor to reclaim the slot. Run in an r<N>-<slot> review window once the user has read the review.
---

# /bip-review-post

Run in an `r<N>-<slot>` review window that `/bip-conductor` spawned.
The user invoking it is their go to post, including on another human's scientific PR (`docs/guides/roles.md`, "Posting to GitHub").

1. **Find the PR.**
   `<N>` comes from this session's name, `r<N>-<slot>`, not from `ls _ignore/`, since a clone can hold reviews of several PRs.
   Read `_ignore/REVIEW-p<N>.md` and the head it says it reviewed.
   If `gh pr view <N> --json headRefOid` has moved past that head, stop and tell the user: an approval would cover code nobody reviewed.

2. **Draft** `_ignore/REVIEW-p<N>-post.md`: what the author needs, written to them.
   Open with `🤖`, then the verdict in a sentence or two, then the findings grouped by kind, each with its `file:line` and the fix.
   Leave out the process notes (agents run, paths read, "nothing posted"), and say which head the line numbers refer to.
   Apply `PROSE-DISCIPLINE.md` and `EVIDENCE-DISCIPLINE.md` (bipartite repo root).

3. **Check it** with `/bip-comment-check` on the draft, against the PR branch.
   Cut every claim it finds FALSE, and correct every PARTIALLY TRUE one to what the code shows.

4. **Post** as one review:
   ```bash
   gh pr view <N> --json reviews --jq '[.reviews[] | select(.author.login == "<user>" and .state == "APPROVED")] | length'
   gh pr review <N> --approve --body-file _ignore/REVIEW-p<N>-post.md   # if that printed 0
   gh pr review <N> --comment --body-file _ignore/REVIEW-p<N>-post.md   # if the user had already approved
   ```
   `<user>` is `gh api user --jq .login`.

5. **Hand back the slot.**
   Message the conductor named in `$CLONE_ROOT/.conductor-session` (the clone's parent directory) with the review's URL, and ask it to preserve `REVIEW-p<N>.md` and `REVIEW-p<N>-post.md`, close this window, and reclaim `<slot>`.
   Tell the user the URL and what changed between the review and the post.
