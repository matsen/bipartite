package export

import "testing"

// lexiconTitles stand in for a library's sentence-case titles.
var lexiconTitles = []string{
	"The posterior and the prior in Bayesian phylogenetics",
	"Bayesian inference of phylogeny",
	"Fast Markov chain Monte Carlo without likelihoods",
	"Phylogenetic inference via sequential Monte Carlo",
	"Deep learning for protein design",
	"Learning a Bayesian model of evolution", // "Learning" starts the title: no evidence
}

func TestProtectTitle(t *testing.T) {
	lex := NewLexicon(lexiconTitles)
	tests := []struct {
		name, title, want string
		allCaps           bool
	}{
		{"sentence case keeps its own capitals",
			"A nascent peptide code for translational control of mRNA stability in Drosophila cells",
			"A nascent peptide code for translational control of {mRNA} stability in {Drosophila} cells", false},
		{"word after a colon is left to the style",
			"Decision Transformer: Reinforcement learning via sequence modeling",
			"Decision {Transformer}: Reinforcement learning via sequence modeling", false},
		{"capital after a question mark is kept",
			"Why are there so few clones? The influence of selection",
			"Why are there so few clones? {The} influence of selection", false},
		{"title case braces acronyms and lexicon words only",
			"Variational Inference with Node Embeddings (VINE) for Scalable Bayesian Phylogenetics",
			"Variational Inference with Node Embeddings ({VINE}) for Scalable {Bayesian} Phylogenetics", false},
		{"title case: mixed-case name and Monte Carlo",
			"SaProt: Protein Language Modeling via Markov Chain Monte Carlo",
			"{SaProt}: Protein Language Modeling via {Markov} Chain {Monte} {Carlo}", false},
		{"title case word seen only lowercase is not braced",
			"Deep Learning",
			"Deep Learning", false},
		{"letters with digits",
			"Antibody responses to H3N2 and SARS-CoV-2",
			"Antibody responses to {H3N2} and {SARS-CoV-2}", false},
		{"all caps is left alone and reported",
			"A SIGNIFICANCE TEST FOR THE LASSO",
			"A SIGNIFICANCE TEST FOR THE LASSO", true},
		{"math passes through, braced",
			`Nested $\hat R$: Assessing convergence for Markov chain Monte Carlo`,
			`Nested {$\hat R$}: Assessing convergence for {Markov} chain {Monte} {Carlo}`, false},
		{"math without a backslash",
			"Impossibility of phylogeny reconstruction from $k$-mer counts",
			"Impossibility of phylogeny reconstruction from {$k$}-mer counts", false},
		{"an existing brace group passes through",
			"{Trend Filtering on Graphs}",
			"{Trend Filtering on Graphs}", false},
		{"special characters are escaped",
			"A Study of α & β: 100% Complete",
			`A Study of α \& β: 100\% Complete`, false},
		{"unmatched dollar and brace are escaped",
			"Costs $5 and $10 { per test",
			`Costs \$5 and \$10 \{ per test`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, allCaps := ProtectTitle(tt.title, lex)
			if got != tt.want || allCaps != tt.allCaps {
				t.Errorf("ProtectTitle(%q) = %q, %v; want %q, %v", tt.title, got, allCaps, tt.want, tt.allCaps)
			}
		})
	}
}

func TestNewLexicon(t *testing.T) {
	lex := NewLexicon(append(lexiconTitles,
		"Network analysis of Learning curves",              // one capitalized "Learning" against one lowercase
		"Variational Inference for Scalable Phylogenetics", // Title Case: not evidence
	))
	for word, want := range map[string]bool{
		"Bayesian": true, "Markov": true, "Monte": true, "Carlo": true,
		"Learning": false, "Inference": false, "Scalable": false,
	} {
		if lex.proper[word] != want {
			t.Errorf("lexicon[%q] = %v, want %v", word, lex.proper[word], want)
		}
	}
}

func TestEscapeField(t *testing.T) {
	tests := []struct{ input, want string }{
		{"Journal of {R} & Statistics", `Journal of {R} \& Statistics`},
		{`We estimate $\theta$ for 50% of cases`, `We estimate $\theta$ for 50\% of cases`},
		{"$100 price", `\$100 price`},
	}
	for _, tt := range tests {
		if got := escapeField(tt.input); got != tt.want {
			t.Errorf("escapeField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
