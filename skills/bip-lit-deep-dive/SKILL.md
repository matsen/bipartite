---
name: bip-lit-deep-dive
description: Back each item of a plan or design (a list of choices, claims or settings) with papers read from their PDFs — search, get the missing papers into the library, check each claim against its paper with a verdict, fold the verdicts into the plan, and, in a manuscript session, put the result in the paper. Use when the user wants the full literature pass on a list, not a single lookup.
---

# /bip-lit-deep-dive

`/bip-lit-deep-dive <plan, or the list in it>`

Every lookup, PDF read and import below follows `/bip-lit`; load it first.
Step 3 matters most: a paper picked from its abstract can cut against the claim, or support it only in a regime far from yours.

## Steps

1. **Search.** Number the items if the plan does not. Send agents over the items (a few items each). Tell each agent to load `/bip-lit` and search the library, then ASTA.
   Write a notes file in the project (for example `notes/literature-<topic>.md`). For each item, list the candidate papers and the claim each would carry. End it with the papers not in the library, each as a `https://doi.org/<doi>` link. A paper goes on that list only after `bip search --doi <doi>` and an author-and-title search (`/bip-lit` step 3) both miss; a hit means it is in the library, often under another year or version, so list its key instead.
2. **Get the papers** into the library by `/bip-lit`'s "Adding to the library". Repeat until every paper is in, or the user drops it.
   If work is blocked on the papers and the user doesn't respond after the DOI list is up, page the user through `staff`.
3. **Read the load-bearing papers.** Send 2–4 papers to each reading agent, in parallel. Tell each agent to load `/bip-lit`, follow its PDF steps, and write or update the nexus note. For each claim it returns the quoted page, table or figure and a verdict: SUPPORTED, QUALIFIED (say how) or NOT SUPPORTED.
   A setting the paper does not state (a default, a ratio you derive from a table) gets its source named, for example the tool's code at a commit.
4. **Fold the verdicts back** into the notes file and the plan or decision record. A NOT SUPPORTED claim comes out of the plan's reasoning, or goes to the user as a gap.
5. **In a manuscript session, put the result in the paper** where and in the form `/bip-ms`'s rules decide, with each item's evidence and gap. Add each cited paper with `bip export --bibtex --append <bib> <id>`, where `<bib>` is the file `\bibliography` names. Build the PDF and give the user the diff.
   A paper cited while writing, or after the user reads the section, goes through step 3 before it goes in.

Report each item's verdicts in a table, the papers still missing (each as its `https://doi.org/<doi>` link, in every message that asks for one or reports it missing), and what changed in the plan.
