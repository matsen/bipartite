---
name: bip-recap
description: Recap what the user did over the last N days (default 14), from their merged PRs and opened issues across GitHub. A short list for remembering, not a digest to post.
---

# /bip-recap

Set `DAYS` to `$ARGUMENTS`, or 14 if it is empty.
Agents post as the user, so `--author @me` covers their work too.

```bash
d=$(date -d "$DAYS days ago" +%F 2>/dev/null || date -v-"$DAYS"d +%F)   # GNU, then BSD/macOS
gh search prs --author @me --merged-at ">=$d" --limit 1000 --json repository,title,url
gh search issues --author @me --created ">=$d" --limit 1000 --json repository,title,url,state
```

Group by repo, then by theme within a repo, and write at most 15 one-line bullets in the reply, each with a link to the PR or issue that best stands for it. Put the PR and issue counts per repo on one line at the end. Write no file and post nothing.
