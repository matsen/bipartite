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
// VERIFIED 2026-09-14, on this machine, by editing this function and
// observing live panes: the payload DOES supply the field, and for Sonnet 5
// sessions here it reports 200000. A fallback of 1_000_000 for Sonnet was
// tried and is wrong for this configuration -- `code.claude.com`'s
// model-config page says Sonnet 5 runs the 1M window by default on the
// Anthropic API, and that does not describe these sessions. So the map stays
// conservative and the payload decides.
//
// Fallback direction is deliberate: unknown models get the SMALLER guess.
// Too small over-reports usage (alarming, safe); too large under-reports and
// hides an imminent compaction.
//
// UNRESOLVED, and do not paper over it: the numerator (`usedTotal` of the
// last assistant message's `usage`) routinely EXCEEDS this denominator on
// long sessions -- three live Sonnet workers read 306%, 236% and 158% the
// same day. A single API call's prompt cannot exceed the window, so the two
// figures are not the same quantity: either that `usage` object is
// cumulative across the session rather than per-call, or the window is
// larger than reported. Until that is settled, TREAT A FIGURE OVER 100% AS
// UNINTERPRETABLE -- in particular it is NOT evidence that the session has
// auto-compacted, and there is no documented way to detect compaction count
// from the payload or the transcript at all. A conductor read 460%/304%/158%
// as "four or five compactions" and reported that to a user; nothing
// supported it.
function getContextWindow(model, payload) {
  const reported = Number(payload?.context_window?.context_window_size);
  if (Number.isFinite(reported) && reported > 0) return reported;
  const id = String(model?.model_id || model?.display_name || "").toLowerCase();
  if (id.includes("1m") || id.includes("1000k")) return 1_000_000;
  if (id.includes("opus")) return 1_000_000;
  return 200_000; // sonnet/haiku/unknown: guess small, see note above
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
