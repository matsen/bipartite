package prov

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Tag is one %PROV[id] occurrence.
type Tag struct {
	ID       string
	File     string // relative to the paper directory
	Line     int
	Sentence string   // the line's text before its comment
	Text     string   // the whole line
	Para     []string // the other prose lines of its paragraph
}

// Paper is what the scanner extracts from main.tex and its \input files.
type Paper struct {
	Tags      []Tag
	Malformed []Tag // %PROV[ with no whitespace before it
	RenderPin string
	Renders   []string // paths passed to \render
	Files     []string // scanned files, relative to the paper directory
}

var (
	inputRe     = regexp.MustCompile(`\\input\{([^}]+)\}`)
	renderpinRe = regexp.MustCompile(`\\newcommand\{\\renderpin\}\{([^}]*)\}`)
	renderRe    = regexp.MustCompile(`\\render\{([^}]*)\}`)
	tagRe       = regexp.MustCompile(`%PROV\[([^\]]*)\]`)
)

// ScanPaper reads main (relative to dir) and every file it \inputs.
func ScanPaper(dir, main string) (*Paper, error) {
	p := &Paper{}
	seen := map[string]bool{}
	if err := p.scan(dir, main, seen); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Paper) scan(dir, rel string, seen map[string]bool) error {
	if seen[rel] {
		return nil
	}
	seen[rel] = true
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return err
	}
	p.Files = append(p.Files, rel)
	lines := strings.Split(string(data), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	for i, line := range lines {
		body := line[:commentStart(line)]
		if m := renderpinRe.FindStringSubmatch(body); m != nil {
			p.RenderPin = m[1]
		}
		for _, m := range renderRe.FindAllStringSubmatch(body, -1) {
			p.Renders = append(p.Renders, m[1])
		}
		for _, loc := range tagRe.FindAllStringSubmatchIndex(line, -1) {
			if loc[0] > 0 && line[loc[0]-1] == '\\' {
				continue // \%PROV is text, not a tag
			}
			t := Tag{ID: line[loc[2]:loc[3]], File: rel, Line: i + 1, Sentence: body, Text: line, Para: paragraph(lines, i)}
			if loc[0] == 0 || (line[loc[0]-1] != ' ' && line[loc[0]-1] != '\t') {
				p.Malformed = append(p.Malformed, t)
				continue
			}
			p.Tags = append(p.Tags, t)
		}
		for _, m := range inputRe.FindAllStringSubmatch(body, -1) {
			child := m[1]
			if filepath.Ext(child) == "" {
				child += ".tex"
			}
			child = filepath.Join(filepath.Dir(rel), child)
			if err := p.scan(dir, child, seen); err != nil {
				return fmt.Errorf("%s:%d: \\input: %w", rel, i+1, err)
			}
		}
	}
	return nil
}

// paragraph returns the prose lines around lines[i], up to the nearest blank
// line on each side, excluding lines[i] and comment-only lines.
func paragraph(lines []string, i int) []string {
	blank := func(j int) bool { return strings.TrimSpace(lines[j]) == "" }
	lo, hi := i, i
	for lo > 0 && !blank(lo-1) {
		lo--
	}
	for hi < len(lines)-1 && !blank(hi+1) {
		hi++
	}
	var para []string
	for j := lo; j <= hi; j++ {
		if j != i && strings.TrimSpace(lines[j][:commentStart(lines[j])]) != "" {
			para = append(para, lines[j])
		}
	}
	return para
}

// commentStart returns the index of the first unescaped %, or len(line).
func commentStart(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' {
			i++
			continue
		}
		if line[i] == '%' {
			return i
		}
	}
	return len(line)
}
