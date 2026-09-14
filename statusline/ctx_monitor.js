#!/usr/bin/env node
"use strict";

const fs = require("fs");
const { execSync } = require("child_process");

// --- input ---
const input = readJSON(0); // stdin
const sessionId = `\x1b[90m${String(input.session_id ?? "")}\x1b[0m`;
const transcript = input.transcript_path;
const model = input.model || {};
const name = `\x1b[95m${String(model.display_name ?? "")}\x1b[0m`.trim();
// input.agent.name: set when claude is launched with --agent <name>.
// input.session_name: set via --name at launch, or /rename mid-session.
// Both are documented in the CLI's own statusline hook payload schema.
const agentName = input.agent?.name || input.session_name || "";
const agentLabel = agentName ? ` \x1b[96m(${agentName})\x1b[0m` : "";
const cwd = input.workspace?.current_dir || input.cwd || process.cwd();
// Context window size. PREFER THE PAYLOAD'S OWN FIGURE -- the CLI reports
// `context_window.context_window_size`, the window actually in force for
// this session. An id-based map is structurally the wrong instrument: it has
// to be updated whenever a model ships or a default changes, and it fails
// silently when it isn't. Read the thing rather than a correlate of it.
//
// MEASURED 2026-09-14 across four live sessions: the payload DOES supply the
// field, and it reports 1_000_000 for BOTH Sonnet 5 and Opus 5 here --
// matching code.claude.com/docs/en/model-config.md ("Fable 5.1, Fable 5,
// Sonnet 5, and Opus 4.7 and later run with the 1M window by default").
//
// The old hard-coded 200_000 for Sonnet was the whole bug, and it was a bad
// one: every Sonnet session read 5x its true usage. Three workers rendered
// 460%, 306% and 236%; against the real window those are 92%, 64% and 50%.
// A conductor read the >100% figures as "four or five compactions" and
// reported that to a user. Nothing supported it -- and note that the
// numerator was never at fault. `usedTotal` of the last assistant message's
// `usage` is current window occupancy and was correct throughout; only the
// denominator was wrong.
//
// TRAP THAT COST AN HOUR, worth knowing before you test a change here: a
// live statusline pane does NOT re-render on demand. Editing this file and
// then reading a pane can show you the PRE-EDIT render, with fresh-looking
// numbers, because the token counts move on their own schedule. That is how
// the 200_000 reading survived one round of "verification" and got written
// into this comment as a measured fact. Test by feeding a payload to the
// script on stdin, or by extracting getContextWindow and calling it -- not
// by looking at a pane.
//
// Fallback direction is deliberate: unknown models get the SMALLER guess.
// Too small over-reports usage (alarming, safe); too large under-reports and
// hides an imminent compaction.
//
// NOT AVAILABLE AT ALL, so do not try to infer it: there is no documented
// way to detect whether -- or how many times -- a session has auto-compacted,
// from this payload or from the transcript jsonl. The payload has no
// compaction field, and the jsonl format is explicitly documented as internal
// and version-fragile. Auto-compact fires near ~967K for a 1M window (see
// model-config.md; `/autocompact`, `--autocompact`, and
// CLAUDE_CODE_AUTO_COMPACT_WINDOW adjust it), so a session below that has
// not compacted for capacity reasons -- but that is a bound, not a count.
function getContextWindow(model, payload) {
  const reported = Number(payload?.context_window?.context_window_size);
  if (Number.isFinite(reported) && reported > 0) return reported;
  const id = String(model?.model_id || model?.display_name || "").toLowerCase();
  if (id.includes("1m") || id.includes("1000k")) return 1_000_000;
  if (id.includes("haiku")) return 200_000;
  if (id.includes("opus") || id.includes("sonnet") || id.includes("fable")) {
    return 1_000_000;
  }
  return 200_000; // unknown: guess small, see note above
}
const CONTEXT_WINDOW = getContextWindow(model, input);

// --- helpers ---
function readJSON(fd) {
  try {
    return JSON.parse(fs.readFileSync(fd, "utf8"));
  } catch {
    return {};
  }
}
function color(p) {
  if (p >= 90) return "\x1b[31m"; // red
  if (p >= 70) return "\x1b[33m"; // yellow
  return "\x1b[32m"; // green
}
const comma = (n) =>
  new Intl.NumberFormat("en-US").format(
    Math.max(0, Math.floor(Number(n) || 0))
  );

function usedTotal(u) {
  return (
    (u?.input_tokens ?? 0) +
    (u?.output_tokens ?? 0) +
    (u?.cache_read_input_tokens ?? 0) +
    (u?.cache_creation_input_tokens ?? 0)
  );
}

function syntheticModel(j) {
  const m = String(j?.message?.model ?? "").toLowerCase();
  return m === "<synthetic>" || m.includes("synthetic");
}

function assistantMessage(j) {
  return j?.message?.role === "assistant";
}

function subContext(j) {
  return j?.isSidechain === true;
}

function contentNoResponse(j) {
  const c = j?.message?.content;
  return (
    Array.isArray(c) &&
    c.some(
      (x) =>
        x &&
        x.type === "text" &&
        /no\s+response\s+requested/i.test(String(x.text))
    )
  );
}

function parseTs(j) {
  const t = j?.timestamp;
  const n = Date.parse(t);
  return Number.isFinite(n) ? n : -Infinity;
}

// Get working directory basename
function getDirName(path) {
  const parts = path.split("/").filter(Boolean);
  return parts[parts.length - 1] || "/";
}

// Get git branch
function getGitBranch() {
  try {
    const branch = execSync("git rev-parse --abbrev-ref HEAD 2>/dev/null", {
      cwd: cwd,
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
    return branch ? `\x1b[36m[${branch}]\x1b[0m` : "";
  } catch {
    return "";
  }
}

// Read the transcript file into lines. Returns null when not configured or
// unreadable; callers treat that the same as an empty transcript.
function readTranscriptLines() {
  if (!transcript) return null;
  try {
    return fs.readFileSync(transcript, "utf8").split(/\r?\n/);
  } catch {
    return null;
  }
}

// Timestamp of the most recent compaction summary in the transcript, or
// -Infinity if none. The summary appears as a user-role entry with
// `isCompactSummary: true` at the top level (sibling of `type`/`message`).
// Returning -Infinity rather than null lets callers use it as a plain
// numeric threshold with the file's `parseTs` sentinel convention.
function latestCompactTs(lines) {
  let result = -Infinity;
  for (const line of lines) {
    const s = line.trim();
    if (!s) continue;
    let j;
    try { j = JSON.parse(s); } catch { continue; }
    if (j.isCompactSummary === true) {
      const ts = parseTs(j);
      if (ts > result) result = ts;
    }
  }
  return result;
}

// Newest main-context assistant `usage` with timestamp strictly after `minTs`
// (not file order). Pass `-Infinity` to keep the pre-compaction-aware behavior.
// The `ts > minTs` guard is what stops a pre-compact assistant turn from being
// reported after `/compact` runs but before the next user prompt.
function newestMainUsageAfter(lines, minTs) {
  let latestTs = -Infinity;
  let latestUsage = null;
  for (let i = lines.length - 1; i >= 0; i--) {
    const line = lines[i].trim();
    if (!line) continue;

    let j;
    try {
      j = JSON.parse(line);
    } catch {
      continue;
    }
    const u = j.message?.usage;
    if (
      subContext(j) ||
      syntheticModel(j) ||
      j.isApiErrorMessage === true ||
      usedTotal(u) === 0 ||
      contentNoResponse(j) ||
      !assistantMessage(j)
    )
      continue;

    const ts = parseTs(j);
    if (ts <= minTs) continue;

    if (ts > latestTs) {
      latestTs = ts;
      latestUsage = u;
    }
    else if (ts == latestTs && usedTotal(u) > usedTotal(latestUsage)) {
      latestUsage = u;
    }
  }
  return latestUsage;
}

// --- compute/print ---
const dirName = `\x1b[94m${getDirName(cwd)}\x1b[0m`;
const gitBranch = getGitBranch();
const dirInfo = gitBranch ? `${dirName} ${gitBranch}` : dirName;

const lines = readTranscriptLines();
const compactTs = lines ? latestCompactTs(lines) : -Infinity;
const usage = lines ? newestMainUsageAfter(lines, compactTs) : null;
if (!usage) {
  // Compaction with no later assistant turn = the user just ran `/compact`
  // and hasn't sent the next prompt; the empty-state message reflects that.
  const msg = compactTs > -Infinity
    ? "post-compact: usage refreshes on next turn."
    : "context window usage starts after your first question.";
  console.log(`${name}${agentLabel} | ${dirInfo} | \x1b[36m${msg}\x1b[0m`);
  process.exit(0);
}

const used = usedTotal(usage);
const pct = CONTEXT_WINDOW > 0 ? Math.round((used * 1000) / CONTEXT_WINDOW) / 10 : 0;

const usagePercentLabel = `${color(pct)}context used ${pct.toFixed(1)}%\x1b[0m`;
const usageCountLabel = `\x1b[33m(${comma(used)}/${comma(
  CONTEXT_WINDOW
)})\x1b[0m`;

console.log(
  `${name}${agentLabel} | ${dirInfo} | ${usagePercentLabel} - ${usageCountLabel}`
);
