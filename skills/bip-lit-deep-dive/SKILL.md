---
name: bip-lit-deep-dive
description: Back each item of a plan or design (a list of choices, claims or settings) with papers read from their PDFs — search, get the missing papers into the library, check each claim against its paper with a verdict, fold the verdicts into the plan, and, in a manuscript session, write the section. Use when the user wants the full literature pass on a list, not a single lookup.
---

# /bip-lit-deep-dive

`/bip-lit-deep-dive <plan, or the list in it>`

Every lookup, PDF read and import below follows `/bip-lit`; load it first.
The step that pays is step 3: a paper picked from its abstract can cut against the claim, or support it only in a regime far from yours.

## Steps

1. **Search.** Number the items if the plan does not. Send agents over the items (a few items each), each searching the library, then ASTA.
   Write a notes file in the project (for example `notes/literature-<topic>.md`). For each item, list the candidate papers and the claim each would carry. End it with the papers not in the library, each as a `https://doi.org/<doi>` link.
2. **Get the papers.** Give the user the DOI links to add to Paperpile, run `/bip-lit-import`, and confirm each paper arrived (`/bip-lit`, "Adding to the library"). Repeat until every paper is in, or the user drops it.
3. **Read the load-bearing papers.** Send 2–4 papers to each reading agent, in parallel. Tell each agent to load `/bip-lit`, follow its PDF steps, and write or update the nexus note. For each claim it returns the quoted page, table or figure and a verdict: SUPPORTED, QUALIFIED (say how) or NOT SUPPORTED.
   A setting the paper does not state (a default, a ratio you derive from a table) gets its source named, for example the tool's code at a commit.
4. **Fold the verdicts back** into the notes file and the plan or decision record. A NOT SUPPORTED claim comes out of the plan's reasoning, or goes to the user as a gap.
5. **In a manuscript session, write it up.** Write the affected section by `/bip-ms`'s rules, with each item, its evidence and its gap. Add the cited papers with `bip export --bibtex --append <bib> <id>`, build the PDF, and give the section to the user before any ruling.

Report each item's verdicts in a table, the papers still missing, and what changed in the plan.
