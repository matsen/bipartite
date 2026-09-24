package prov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Finding levels.
const (
	LevelError  = "error"
	LevelReview = "review"
	LevelInfo   = "info"
)

// Finding is one result of a check.
type Finding struct {
	Level   string `json:"level"`
	ID      string `json:"id"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

// Options configures Check.
type Options struct {
	PaperDir string // directory holding the ledger and main file; a git work tree
	Main     string // main TeX file, relative to PaperDir
	Ledger   string // ledger file, relative to PaperDir
	// Resolve maps an org/name repo to its local clone.
	Resolve func(repo string) (string, error)
	Fetch   bool // git fetch each repo before reading it
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

type checker struct {
	opts     Options
	ledger   *Ledger
	paper    *Paper
	findings []Finding
	repos    map[string]*repoState
	launches map[string][]string
	commits  map[string]*commitState // by repo + " " + sha
	bad      []*commitState          // unresolvable commits, in first-use order
	dirty    map[string]bool         // entries with a source error or review, or an unresolvable commit
}

// commitState memoises one commit's resolution, so an unreachable commit is
// one finding however many entries use it.
type commitState struct {
	full  string
	err   error
	id    string // first user: the entry, or run:<name> for a launch
	users int
}

type repoState struct {
	git gitRepo
	err error
}

// Check runs every mechanical check. The error return is for failures that
// stop the check as a whole (unreadable ledger or paper); everything else is
// a Finding.
func Check(opts Options) ([]Finding, error) {
	l, err := LoadLedger(filepath.Join(opts.PaperDir, opts.Ledger))
	if err != nil {
		return nil, err
	}
	p, err := ScanPaper(opts.PaperDir, opts.Main)
	if err != nil {
		return nil, err
	}
	c := &checker{opts: opts, ledger: l, paper: p, repos: map[string]*repoState{}, launches: map[string][]string{}, commits: map[string]*commitState{}, dirty: map[string]bool{}}
	c.checkTags()
	ids := make([]string, 0, len(l.Entries))
	for id := range l.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		before := len(c.findings)
		c.checkEntry(id, l.Entries[id])
		for _, f := range c.findings[before:] {
			if f.Level != LevelInfo {
				c.dirty[id] = true
			}
		}
	}
	c.checkDerived(ids)
	c.checkRenderPin()
	for _, b := range c.bad {
		c.add(LevelError, b.id, "%v (used by %d entries)", b.err, b.users)
	}
	rank := map[string]int{LevelError: 0, LevelReview: 1, LevelInfo: 2}
	sort.SliceStable(c.findings, func(i, j int) bool {
		return rank[c.findings[i].Level] < rank[c.findings[j].Level]
	})
	return c.findings, nil
}

func (c *checker) add(level, id, format string, args ...any) *Finding {
	c.findings = append(c.findings, Finding{Level: level, ID: id, Message: fmt.Sprintf(format, args...)})
	return &c.findings[len(c.findings)-1]
}

func (c *checker) repo(name string) (gitRepo, error) {
	if s, ok := c.repos[name]; ok {
		return s.git, s.err
	}
	s := &repoState{}
	c.repos[name] = s
	dir, err := c.opts.Resolve(name)
	if err != nil {
		s.err = fmt.Errorf("repo %s: %w", name, err)
		return s.git, s.err
	}
	s.git = gitRepo{dir: dir}
	if c.opts.Fetch {
		if err := s.git.fetch(); err != nil {
			s.err = fmt.Errorf("repo %s: %w", name, err)
		}
	}
	return s.git, s.err
}

func (c *checker) checkTags() {
	for _, t := range c.paper.Malformed {
		f := c.add(LevelError, t.ID, "tag has no space before %%PROV; a bare %% swallows the line break")
		f.File, f.Line = t.File, t.Line
	}
	old, oldErr := c.auditedLines()
	for _, t := range c.paper.Tags {
		e, ok := c.ledger.Entries[t.ID]
		if !ok {
			f := c.add(LevelError, t.ID, "tag has no ledger entry")
			f.File, f.Line = t.File, t.Line
			continue
		}
		if e.Token != "" && !strings.Contains(t.Sentence, e.Token) {
			f := c.add(LevelError, t.ID, "token %q not in its sentence", e.Token)
			f.File, f.Line, f.Scope = t.File, t.Line, e.Scope
		}
		var f *Finding
		switch {
		case c.ledger.AuditedThrough == "":
			f = c.add(LevelReview, t.ID, "tagged sentence not yet audited (no audited_through)")
		case oldErr != nil:
			continue // reported once by auditedLines
		case !old[t.File][t.Text]:
			f = c.add(LevelReview, t.ID, "tagged sentence changed since %s", c.ledger.AuditedThrough)
		default:
			continue
		}
		f.File, f.Line, f.Scope = t.File, t.Line, e.Scope
	}
}

// auditedLines returns, per scanned file, the set of its lines at
// audited_through. A tagged line absent from that set has changed.
func (c *checker) auditedLines() (map[string]map[string]bool, error) {
	at := c.ledger.AuditedThrough
	if at == "" {
		return nil, nil
	}
	g := gitRepo{dir: c.opts.PaperDir}
	prefix, err := g.run("rev-parse", "--show-prefix")
	if err == nil {
		_, err = g.run("rev-parse", "--verify", "--quiet", at+"^{commit}")
		if err != nil {
			err = fmt.Errorf("audited_through %s is not a commit of the paper repo", at)
		}
	}
	if err != nil {
		c.add(LevelError, "audited_through", "%v", err)
		return nil, err
	}
	sets := map[string]map[string]bool{}
	for _, rel := range c.paper.Files {
		set := map[string]bool{}
		// A file absent at audited_through leaves an empty set: every tag in it changed.
		if data, err := g.show(at, filepath.ToSlash(filepath.Join(strings.TrimSpace(string(prefix)), rel))); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				set[strings.TrimRight(line, "\r")] = true
			}
		}
		sets[rel] = set
	}
	return sets, nil
}

// runLaunches returns the launch commits of a run: run_facts.json's
// launches[].commit (or its pipeline_sha when that list is absent), plus
// the ledger's own launches. "unknown" is dropped.
func (c *checker) runLaunches(name string) ([]string, error) {
	if l, ok := c.launches[name]; ok {
		return l, nil
	}
	run, ok := c.ledger.Runs[name]
	if !ok {
		return nil, fmt.Errorf("run %q not in runs", name)
	}
	var commits []string
	if run.Facts != nil {
		fromFacts, err := c.factsLaunches(name, run.Facts)
		if err != nil {
			c.add(LevelError, "run:"+name, "%v", err)
		}
		commits = fromFacts
	}
	for _, sha := range run.Launches {
		if sha == "unknown" || containsPrefix(commits, sha) {
			continue
		}
		commits = append(commits, sha)
	}
	c.launches[name] = commits
	return commits, nil
}

func (c *checker) factsLaunches(name string, pin *Pin) ([]string, error) {
	g, err := c.repo(pin.Repo)
	if err != nil {
		return nil, err
	}
	full, err := g.commit(pin.SHA)
	if err != nil {
		return nil, fmt.Errorf("facts: %w", err)
	}
	data, err := g.show(full, pin.Path)
	if err != nil {
		return nil, fmt.Errorf("facts: %w", err)
	}
	var facts struct {
		Launches []struct {
			Commit string    `json:"commit"`
			Stages *[]string `json:"stages"`
		} `json:"launches"`
		PipelineSHA string `json:"pipeline_sha"`
	}
	if err := json.Unmarshal(data, &facts); err != nil {
		return nil, fmt.Errorf("facts %s: %w", pin.Path, err)
	}
	var commits []string
	for _, l := range facts.Launches {
		if l.Stages != nil && len(*l.Stages) == 0 {
			continue // submitted no tasks, so no claim describes it
		}
		commits = append(commits, l.Commit)
	}
	if facts.Launches == nil && facts.PipelineSHA != "" {
		c.add(LevelInfo, "run:"+name, "producing commit = HEAD at collection, not per-launch (run_facts.json has pipeline_sha, no launches)")
		commits = []string{facts.PipelineSHA}
	}
	return commits, nil
}

// commit resolves sha in repo once. An unresolvable commit is reported at
// the end of Check; callers skip it silently.
func (c *checker) commit(repo string, g gitRepo, sha, id string) (string, bool) {
	key := repo + " " + sha
	st, ok := c.commits[key]
	if !ok {
		st = &commitState{id: id}
		st.full, st.err = g.commit(sha)
		c.commits[key] = st
		if st.err != nil {
			c.bad = append(c.bad, st)
		}
	}
	st.users++
	return st.full, st.err == nil
}

func containsPrefix(commits []string, sha string) bool {
	for _, c := range commits {
		if strings.HasPrefix(c, sha) || strings.HasPrefix(sha, c) {
			return true
		}
	}
	return false
}

func (c *checker) checkEntry(id string, e Entry) {
	used := false
	for _, t := range c.paper.Tags {
		if t.ID == id {
			used = true
			break
		}
	}
	if !used {
		c.add(LevelInfo, id, "no tag uses this entry")
	}
	n := 0
	for _, s := range []string{e.Key, e.Pattern, e.Absent, e.Blob, e.Unsourced} {
		if s != "" {
			n++
		}
	}
	if len(e.From) > 0 {
		n++
	}
	if n != 1 {
		c.add(LevelError, id, "entry has %d extractors; want exactly one of key, pattern, absent, blob, from, unsourced", n)
		return
	}
	if len(e.From) > 0 {
		if e.Scope == "" {
			c.add(LevelError, id, "derived entry needs a scope")
		}
		return // checked by checkDerived once every input is checked
	}
	if e.Unsourced != "" {
		c.add(LevelInfo, id, "unsourced: %s", e.Unsourced)
		return
	}
	if (e.Value != nil || e.Blob != "") && e.Scope == "" {
		c.add(LevelError, id, "entry with value or blob needs a scope")
	}
	if e.Key != "" && e.Value == nil {
		c.add(LevelError, id, "key entry needs a value")
		return
	}
	if e.SHA != "" && !shaRe.MatchString(e.SHA) {
		c.add(LevelError, id, "sha %q is not a commit SHA; pin a commit, never a branch", e.SHA)
		return
	}
	if (e.Key != "" || e.Blob != "") && e.SHA == "" {
		c.add(LevelError, id, "key and blob entries need the storage sha they are read at")
		return
	}
	repoName := e.Repo
	if repoName == "" && e.Run != "" {
		if r, ok := c.ledger.Runs[e.Run]; ok && r.Facts != nil {
			repoName = r.Facts.Repo
		}
	}
	if repoName == "" {
		c.add(LevelError, id, "entry names no repo, and its run has no facts repo")
		return
	}
	g, err := c.repo(repoName)
	if err != nil {
		c.add(LevelError, id, "%v", err)
		return
	}
	commits, user := []string{e.SHA}, id
	if e.SHA == "" {
		user = "run:" + e.Run
		if e.Run == "" {
			c.add(LevelError, id, "entry has neither sha nor run")
			return
		}
		commits, err = c.runLaunches(e.Run)
		if err != nil {
			c.add(LevelError, id, "%v", err)
			return
		}
		if len(commits) == 0 {
			c.add(LevelError, id, "run %q has no known launch commit to check at", e.Run)
			return
		}
	}

	var holds, fails []string
	for _, sha := range commits {
		full, ok := c.commit(repoName, g, sha, user)
		if !ok {
			c.dirty[id] = true
			continue
		}
		switch {
		case e.Blob != "":
			got, err := g.blobID(full, e.Path)
			if err != nil {
				c.add(LevelError, id, "%v", err)
			} else if !strings.HasPrefix(got, e.Blob) {
				c.add(LevelReview, id, "blob of %s at %s is %s, ledger has %s", e.Path, short(sha), short(got), e.Blob).Scope = e.Scope
			}
		case e.Key != "":
			c.checkKey(id, e, g, full)
		default:
			data, err := g.show(full, e.Path)
			if err != nil {
				c.add(LevelError, id, "%v", err)
				continue
			}
			found := containsNormalized(data, e.Pattern+e.Absent) // one of the two is empty
			if found == (e.Pattern != "") {
				holds = append(holds, short(full))
			} else {
				fails = append(fails, short(full))
			}
		}
	}
	if len(fails) > 0 {
		what := "pattern absent from"
		if e.Absent != "" {
			what = "absent literal present in"
		}
		msg := fmt.Sprintf("%s %s at %s", what, e.Path, strings.Join(fails, ", "))
		if len(holds) > 0 {
			msg += fmt.Sprintf(" (holds at %s; name the launch in the prose and pin its sha)", strings.Join(holds, ", "))
		}
		c.add(LevelError, id, "%s", msg)
	}
	if e.Key != "" || e.Blob != "" {
		c.checkMain(id, e, g)
	}
}

// checkDerived sends a from: entry to review when any input, transitively,
// has a source error or review. The arithmetic itself stays in the prose.
func (c *checker) checkDerived(ids []string) {
	state := map[string]int{} // 0 unvisited, 1 visiting, 2 done
	var visit func(id string)
	visit = func(id string) {
		if state[id] == 1 {
			c.add(LevelError, id, "from: cycle through this entry")
			return
		}
		if state[id] == 2 {
			return
		}
		state[id] = 1
		for _, in := range c.ledger.Entries[id].From {
			if _, ok := c.ledger.Entries[in]; !ok {
				c.add(LevelError, id, "from: %q is not a ledger entry", in)
				continue
			}
			visit(in)
			if c.dirty[in] {
				c.dirty[id] = true
			}
		}
		state[id] = 2
	}
	for _, id := range ids {
		visit(id)
	}
	for _, id := range ids {
		e := c.ledger.Entries[id]
		var why []string
		for _, in := range e.From {
			if c.dirty[in] {
				why = append(why, in)
			}
		}
		if len(why) > 0 {
			c.add(LevelReview, id, "input changed: %s; recompute from the prose", strings.Join(why, ", ")).Scope = e.Scope
		}
	}
}

// containsNormalized reports whether lit occurs in data, ignoring CRLF vs LF.
func containsNormalized(data []byte, lit string) bool {
	crlf := []byte("\r\n")
	return bytes.Contains(bytes.ReplaceAll(data, crlf, []byte("\n")), bytes.ReplaceAll([]byte(lit), crlf, []byte("\n")))
}

func (c *checker) checkKey(id string, e Entry, g gitRepo, full string) {
	data, err := g.show(full, e.Path)
	if err != nil {
		c.add(LevelError, id, "%v", err)
		return
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		c.add(LevelError, id, "%s at %s: %v", e.Path, short(full), err)
		return
	}
	for _, part := range strings.Split(e.Key, ".") {
		switch node := v.(type) {
		case map[string]any:
			v = node[part]
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(node) {
				v = nil
			} else {
				v = node[i]
			}
		default:
			v = nil
		}
		if v == nil {
			c.add(LevelError, id, "key %s not in %s at %s", e.Key, e.Path, short(full))
			return
		}
	}
	if !sameValue(v, e.Value) {
		c.add(LevelError, id, "%s = %v at %s, ledger value %v", e.Key, v, short(full), e.Value).Scope = e.Scope
	}
}

func sameValue(got, want any) bool {
	gf, gok := toFloat(got)
	wf, wok := toFloat(want)
	if gok && wok {
		return gf == wf
	}
	return fmt.Sprint(got) == fmt.Sprint(want)
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case int:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

// checkMain flags a result file that origin/main holds in a different
// version: committed results get overwritten in place.
func (c *checker) checkMain(id string, e Entry, g gitRepo) {
	pinned, err := g.blobID(e.SHA, e.Path)
	if err != nil {
		return // reported by the extractor
	}
	onMain, err := g.blobID("origin/main", e.Path)
	if err != nil {
		c.add(LevelReview, id, "%s is gone at origin/main", e.Path).Scope = e.Scope
	} else if onMain != pinned {
		c.add(LevelReview, id, "%s at origin/main differs from %s; a newer version may exist", e.Path, e.SHA).Scope = e.Scope
	}
}

// checkRenderPin asserts \renderpin equals the storage sha of every entry
// whose path the paper links to with \render.
func (c *checker) checkRenderPin() {
	rendered := map[string]bool{}
	for _, r := range c.paper.Renders {
		rendered[r] = true
	}
	ids := make([]string, 0)
	for id, e := range c.ledger.Entries {
		if e.SHA != "" && rendered[e.Path] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := c.ledger.Entries[id]
		if c.paper.RenderPin == "" {
			c.add(LevelError, id, "paper links %s with \\render but defines no \\renderpin", e.Path)
			continue
		}
		if strings.HasPrefix(e.SHA, c.paper.RenderPin) || strings.HasPrefix(c.paper.RenderPin, e.SHA) {
			continue
		}
		c.add(LevelError, id, "\\renderpin %s differs from the entry's sha %s for rendered %s", c.paper.RenderPin, e.SHA, e.Path)
	}
}
