# Erick's writing standards

Read this at the start of every `/bip-ms-edit` session.
These standards come before any style guide a specialist agent applies.

## Order and scope

- Revise the section first, then the paragraph, then the sentence, then the word.
  Working out of order throws away carefully crafted prose.
- Make no major change to structure or notation without discussing it first.
- Late in revision, add less.
  A side idea earns one sentence in the right place, or none.
- Don't write anything you don't expect people to read, in tables and figures too.

## Sections and paragraphs

- The work is clearly motivated, and the sections flow logically.
- Introduction: introduce background objects before any paragraph about them, prior-work paragraphs included.
  The last paragraph(s) open with "In this paper", and no prior-work paragraph comes after.
  The final paragraph gives the results, then the roadmap.
- Methods open with "we have data of this form", say what is wanted from that data, and then unwind the method from that goal.
- Each paragraph opens with a topic sentence that makes a complete claim, which the rest of the paragraph defends.
  Semicolons are fine in a complex topic sentence.
- One idea per paragraph, and almost always at least three sentences.
- Rather than ending a paragraph with a problem and opening the next with the solution, end with a hint of the solution and open the next with "Indeed, …" (Harmit Malik).

## Claims

- Understatement: overstating annoys reviewers, and real advances get noticed.
- Scope a claim precisely instead of hedging it ("the data alone cannot tell…", not "do not explicitly distinguish").
- Test every claim, especially one you wrote, with: does this also apply to the methods the paper favors?
  If so, find the real distinction.
- Watch for words that promise more than you believe ("conflate" implies that perfect separation is possible).
- Check every number and model description against the tables and methods; a claim often holds for one setting only.
- Don't cite or characterize work that neither author has read.

## Sentences

- The main noun and verb are the most important actor and action.
- No zombie nouns ("the fact that" is a red flag) and no meta-speak about the process ("We note that").
- "We find that X and Y (Figure 1)", never "We ran an analysis and the results are in Figure 1" or "as shown in".
- Alternate short and long sentences.
  No comma splices or run-ons.
- Count subordinate clauses: an interrupting "such as…" aside plus a trailing "so…" clause is too many, so split the sentence.
- Repeat the exact noun phrase instead of a vague referent or a synonym, and keep one umbrella term across the paper.
  Every "this X", "it", "both", or "them" must have one obvious antecedent.
- Watch for collisions between the paper's terms and similar terms in cited work.

## Over-clever wording

Erick has a sharp eye for this; check each sentence, including plain-looking ones.

- clefts and pseudo-clefts ("What is missing is…", "It is X that…");
- set-up frames ("Two kinds of tool exist, and neither…", "The first is… The second is…");
- metaphors and stale figures of speech ("an honest sum", "the payoff is visible");
- colon reveals ("Our networks posit no generative process: …");
- fancy verbs ("posit", "couples", "rest on", "strictly contains");
- analogies that restate the point, "X rather than Y" contrasts that add nothing, and punchy closing lines.

He does like em dashes (`---`) around a list appositive in a topic sentence.

## Words

- Simple, short, everyday words over fancy ones or jargon.
  Active voice.
- Cut any word you can without hurting flow or meaning.
- Cut contentless phrases: "Of course", "Note that", "Interestingly", "very", "nice", "We can see that", "It is important to note that".
- which vs. that: dated, but still observe it.
- No contractions in the paper body.
- Define each acronym once in the abstract, once in the main text, once in the figures, and once in the tables.
- Break any of these rules sooner than say anything barbarous (Orwell).

## Verb tense

Pick one strategy and keep it.

- **Simple**: present tense everywhere; suits methods papers.
- **Complex**:
  accepted facts, present ("DNA is composed of…");
  prior work with current relevance, present perfect ("Studies have shown");
  specific prior methods, past ("Smith sampled 96 swamps");
  our contribution, present ("In this paper, we devise").
  Methods: present for method development, past for experiments.
  Results: past for completed experiments, present for the manuscript itself ("Figure 1 shows") and definitions.
  Discussion: past for specific results, present for conclusions, future for future work.

## Figures

- Captions open with a headline stating the key finding, then the details (relaxed for supplementary figures).
- Figure descriptions belong in the legend.
- State the claim, then cite the figure in parentheses; better still, put the reference in a topic sentence that states the finding.

## LaTeX

- One sentence per line.
- amsmath alignment, never `eqnarray`.
- `\eqref{}` alone, never "Equation \eqref{}".
- Labels `eq:…`, `lem:…`, and so on; never refer to section numbers.
- TeX quotes: ` `` ` and `''`.
- Keep compilation simple (no `\input`), and compile cleanly, references included.

## `%EM` comments

- `%EM` refers to the line directly after it.
- If an edit clearly addresses a comment, delete the comment.
  If you disagree, add a rebuttal comment just after it.
  Never delete one silently.
- A note to a coauthor is `%EM @Name-- …`: short, saying what changed and why, never asking permission.

## Response to reviewers

- The editor should understand the response without reading the manuscript.
- Nearly every change points to new text, including terminology and definition changes.
- Ask no questions; go with your best reading.
- If a reviewer found something unclear, rewrite it.
- Don't compliment reviewers by default.
- Erick's lean: simplify and delete.
