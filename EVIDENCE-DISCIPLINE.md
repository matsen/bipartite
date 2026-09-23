# Evidence discipline

Apply these whenever you cite a number or write a correction: in an issue, PR, commit message, cross-session message, or doc. Re-derive a number before it ships, including one from a subagent or from your own earlier round once the design has changed. Quote the command that produced it. With tiny degrees of freedom, give a range, not a point estimate.

## Before citing a measurement

- A number means something only when it is bound to a population, a unit, and a predicate. State which range, files, or runs you measured in the same sentence as the number.
- For a fraction, state the denominator and check that a member of every category the claim covers could have entered it.
- Check that it is derived from the data and does not just restate the configuration or a struct default. What the program actually runs is a separate claim.
- Check that it reaches a file a reader can open. Gitignored or local-only paths don't count.
- For "which items have property P", derive the population from P, not from the items you already had. Re-derivation cannot find a missing member.
- A partial result licenses no direction. Wait for the whole result, or report only what has arrived and say what hasn't.

## Name the question your check answers, not the one you asked

- A check that succeeds on an adjacent question is worse than one that fails. Before reporting a question as settled, write down the question the check answers. If that is narrower than the claim, the check hasn't been run yet.
- A negative result from a detector is evidence only once you've seen the detector fire. Give every zero a positive control. Show a new guard can both fire and pass, against the live system.
- Make sure the pattern matches the source's form: quoting, anchors, format strings. Inside Claude Code, bare `grep` is ugrep. Use `/usr/bin/grep` for `$`, alternation, or anchors, and use one pattern per `-F`.
- A match on a mention is not a match on a call. Compare content, not containers. A state observed mid-transition is not a state.
- Never echo a conclusion ("clean", "no drift") next to a command whose output you didn't read. Compute the conclusion, or leave it out.
- Check the scaffolding as well as the artifact: the probe, the comparator, the control arm. Care goes to the artifact, and the check gets none.
- Run a check on a second arm whose conventions differ, and smoke-test a parser against the data actually on disk.
- A validity gate is a contract: read what it actually tests.
- A failure that moves when you intervene means the intervention changed more than one thing.
- A self-correction inherits no credibility from being a correction.

## A source read that stops at the first plausible boundary

- Say where you stopped: "read X and Y; did not follow Z's callers".
- For a claim that something is absent, search for the thing you say is missing instead of reading where it would be.
- Mark an unverified verdict as provisional in the text itself, not only in a column heading.

## A verdict built from background tasks does not survive the session that launched it

- A conclusion from a background task is provisional until its output is in a durable artifact you can re-read.
- On resuming a session, treat every unverified verdict in its worklog as provisional.
- When relaying a peer's verdict, relay its basis: "it reports its suite passed", not "its suite passed".

## A success criterion is an unreviewed claim about the right measurement

- Read a criterion against the code before satisfying it. It was written before the measurement existed.
- "A default run" must name the dispatch path wherever paths differ in their defaults.
- Check any population a criterion prescribes.

## Before explaining a discrepancy, establish that there is one

- When two figures for "the same thing" disagree, name the command behind each before accounting for the gap. If they measure different quantities, you have two results, not an error.

## A prohibition is the one claim nobody checks

- Write a "do not re-run" together with the archive line it rests on, quoted, with the file named.
- When you strike a prohibition, grep every downstream artifact for citations of it: briefs, spawn prompts, issue bodies, continuation files.
- State separately what a ruled-out entry measured and what question it closes. If those differ, it isn't ruled out.

## Before writing a correction or a unification

- Before retracting something down to no information, check whether a weaker claim that is still true survives.
- Before accepting a critique, ask whether it would have been raised if the result had come out the other way.
- Two agreeing checkers are not independent if they tested the same sub-claim, or the same artifact under another name.
- Merge two findings into one story only if their remedies land in the same subsystem.
