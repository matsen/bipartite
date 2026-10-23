package export

import (
	"strings"
	"unicode"
)

// Lexicon holds the words that keep their capitals in titles, learned from a
// library's sentence-case titles: a word capitalized mid-title there is a
// proper noun or acronym (Bayesian, Markov, DNA). Title Case titles carry no
// such evidence, so they are protected by looking their words up here.
type Lexicon struct {
	proper map[string]bool // exact capitalized forms
}

// NewLexicon learns proper nouns from titles. A capitalized form counts when it
// appears mid-title in a sentence-case title and its lowercase form appears
// there at most a quarter as often.
func NewLexicon(titles []string) *Lexicon {
	capForms := map[string]map[string]int{} // lowercase -> capitalized form -> count
	lowCount := map[string]int{}
	for _, t := range titles {
		segs := splitTitle(t)
		if classifyTitle(segs) != sentenceCase {
			continue
		}
		for _, s := range segs {
			if s.verbatim || s.word == "" || s.sentStart {
				continue
			}
			key := strings.ToLower(s.word)
			if isCapitalized(s.word) {
				if capForms[key] == nil {
					capForms[key] = map[string]int{}
				}
				capForms[key][s.word]++
			} else {
				lowCount[key]++
			}
		}
	}
	lex := &Lexicon{proper: map[string]bool{}}
	for key, forms := range capForms {
		total := 0
		for _, n := range forms {
			total += n
		}
		if lowCount[key] > total/4 {
			continue
		}
		for form := range forms {
			lex.proper[form] = true
		}
	}
	return lex
}

type titleCase int

const (
	sentenceCase titleCase = iota
	titleCaseWords
	allCaps
)

// span is one piece of a title: a word, the text between words, or a $...$ or
// {...} group that passes through verbatim.
type span struct {
	text     string
	word     string // set when the span is a word
	verbatim bool
	// segStart marks the first word of the title or after ": ", whose first
	// letter a BibTeX style keeps; sentStart also covers after "? " and ". ".
	segStart, sentStart bool
}

// splitTitle cuts a title into spans. Balanced {...} groups and $...$ math
// (no spaces, or containing a backslash) are verbatim; everything else is
// words and the text between them.
func splitTitle(t string) []span {
	var spans []span
	r := []rune(t)
	segStart, sentStart := true, true
	var gap strings.Builder
	flushGap := func() {
		if gap.Len() > 0 {
			spans = append(spans, span{text: gap.String()})
			gap.Reset()
		}
	}
	for i := 0; i < len(r); {
		if end := verbatimEnd(r, i); end > i {
			flushGap()
			spans = append(spans, span{text: string(r[i:end]), verbatim: true})
			segStart, sentStart = false, false
			i = end
			continue
		}
		if isWordRune(r[i]) {
			j := i
			for j < len(r) && (isWordRune(r[j]) || (r[j] == '-' || r[j] == '\'') && j+1 < len(r) && isWordRune(r[j+1])) {
				j++
			}
			flushGap()
			w := string(r[i:j])
			spans = append(spans, span{text: w, word: w, segStart: segStart, sentStart: sentStart})
			segStart, sentStart = false, false
			i = j
			continue
		}
		if strings.ContainsRune(":?.", r[i]) && i+1 < len(r) && unicode.IsSpace(r[i+1]) {
			segStart = segStart || r[i] == ':'
			sentStart = true
		}
		gap.WriteRune(r[i])
		i++
	}
	flushGap()
	return spans
}

// verbatimEnd returns the end of a verbatim group starting at i, or i if none starts there.
func verbatimEnd(r []rune, i int) int {
	switch r[i] {
	case '{':
		depth := 0
		for j := i; j < len(r); j++ {
			switch r[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return j + 1
				}
			}
		}
	case '$':
		for j := i + 1; j < len(r); j++ {
			if r[j] == '$' {
				body := string(r[i+1 : j])
				if j > i+1 && (!strings.ContainsAny(body, " \t") || strings.Contains(body, `\`)) {
					return j + 1
				}
				return i
			}
		}
	}
	return i
}

func isWordRune(c rune) bool { return unicode.IsLetter(c) || unicode.IsDigit(c) }

func isCapitalized(w string) bool {
	for _, c := range w {
		return unicode.IsUpper(c)
	}
	return false
}

// needsBraces reports whether a word's capitals survive no style: an
// uppercase letter after the first (DNA, SaProt, mRNA), or letters mixed with
// digits (H3N2).
func needsBraces(w string) bool {
	hasLetter, hasDigit := false, false
	for i, c := range []rune(w) {
		if i > 0 && unicode.IsUpper(c) {
			return true
		}
		hasLetter = hasLetter || unicode.IsLetter(c)
		hasDigit = hasDigit || unicode.IsDigit(c)
	}
	return hasLetter && hasDigit
}

// minorWord lists the words Title Case leaves lowercase, so they say nothing
// about which case a title is in.
var minorWord = map[string]bool{}

func init() {
	for _, w := range strings.Fields("a an the and but or nor for so yet as at by in of off on per to up via with from into onto over than that upon vs versus is are be its") {
		minorWord[w] = true
	}
}

func classifyTitle(spans []span) titleCase {
	letters, upper := 0, 0
	major, majorCap := 0, 0
	for _, s := range spans {
		if s.verbatim || s.word == "" {
			continue
		}
		for _, c := range s.word {
			if unicode.IsLetter(c) {
				letters++
				if unicode.IsUpper(c) {
					upper++
				}
			}
		}
		// Only a major word of two or more letters, not at a segment start and
		// with no capitals of its own, says whether the author capitalized words.
		if w := []rune(s.word); len(w) > 1 && unicode.IsLetter(w[0]) && !minorWord[strings.ToLower(s.word)] && !s.segStart && !needsBraces(s.word) {
			major++
			if isCapitalized(s.word) {
				majorCap++
			}
		}
	}
	switch {
	case letters > 15 && upper*10 > letters*9:
		return allCaps
	case major >= 1 && majorCap*4 >= major*3:
		return titleCaseWords
	}
	return sentenceCase
}

// ProtectTitle escapes a title for BibTeX and braces the words whose capitals
// a lowercasing style must keep. lex may be nil. allCapsTitle reports a title
// written in capitals, which carries no evidence of which words to keep and is
// left for the style to lowercase.
func ProtectTitle(title string, lex *Lexicon) (out string, allCapsTitle bool) {
	spans := splitTitle(title)
	kind := classifyTitle(spans)
	var b strings.Builder
	for _, s := range spans {
		switch {
		case s.verbatim && strings.HasPrefix(s.text, "$"):
			// A style lowercases math too ($\hat R$ becomes $\hat r$).
			b.WriteString("{" + s.text + "}")
		case s.verbatim:
			b.WriteString(s.text)
		case s.word != "" && kind != allCaps && protect(s, kind, lex):
			b.WriteString("{" + escapeLatex(s.word) + "}")
		default:
			b.WriteString(escapeLatex(s.text))
		}
	}
	return b.String(), kind == allCaps
}

func protect(s span, kind titleCase, lex *Lexicon) bool {
	if needsBraces(s.word) {
		return true
	}
	if !isCapitalized(s.word) || s.segStart {
		return false
	}
	if kind == sentenceCase {
		return true
	}
	return lex != nil && lex.proper[s.word]
}

// escapeField escapes a field's text for BibTeX, passing balanced {...} groups
// and $...$ math through unchanged.
func escapeField(s string) string {
	var b strings.Builder
	for _, sp := range splitTitle(s) {
		if sp.verbatim {
			b.WriteString(sp.text)
		} else {
			b.WriteString(escapeLatex(sp.text))
		}
	}
	return b.String()
}
