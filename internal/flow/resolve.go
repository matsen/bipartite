package flow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/matsen/bipartite/internal/config"
)

// ErrRepoNotInSources is returned by ResolveRepoPath when orgRepo is not
// listed in sources.yml under code or writing. Callers use errors.Is.
var ErrRepoNotInSources = errors.New("repo not in sources.yml")

// ResolveContext carries the optional caller info that ResolveRepoPath uses
// to build a worktree slot path. Zero values mean "not applicable."
type ResolveContext struct {
	IssueNumber int
	PRNumber    int
	Slug        string
	Branch      string
}

// ResolvedPath is the result of ResolveRepoPath.
type ResolvedPath struct {
	// Path is the absolute working directory the caller should use.
	Path string
	// Mode is the effective layout mode after any fallback ("clone" or "worktree").
	Mode string
	// IsNew is true when Path does not yet exist on disk and the caller
	// must create it (e.g. via `git worktree add` for worktree mode).
	IsNew bool
	// Branch is the branch name to use when creating a new worktree.
	// Empty for clone-mode resolutions.
	Branch string
	// FellBack is true when worktree mode was configured but the resolver
	// fell back to the canonical clone (e.g. ResolveContext had no
	// IssueNumber or PRNumber). Callers can use this to print a one-line note.
	FellBack bool
}

// ResolveRepoPath maps a GitHub repo (org/name) plus optional context to a
// working-directory layout decision. It is the single source of truth for
// "where does this repo's working tree live?"
//
// Precedence, each leaf field independently: per-repo sources.yml layout >
// global ~/.config/bip/config.yml layout > built-in default (clone mode).
//
// In clone mode (the default) the returned Path matches GetRepoLocalPath
// byte-for-byte. In worktree mode with a non-empty ResolveContext, Path is
// the expanded worktree.root/worktree.slot template. Worktree mode with an
// empty context falls back to the canonical clone and sets FellBack=true.
func ResolveRepoPath(nexusPath, orgRepo string, ctx ResolveContext) (ResolvedPath, error) {
	sources, err := LoadSources(nexusPath)
	if err != nil {
		return ResolvedPath{}, err
	}
	cfg, err := LoadConfig(nexusPath)
	if err != nil {
		return ResolvedPath{}, err
	}
	// Tolerate a missing/unreadable global config — that's the "absent
	// block = today's behavior" guarantee.
	gcfg, _ := config.LoadGlobalConfig()
	if gcfg == nil {
		gcfg = &config.GlobalConfig{}
	}

	repoName := ExtractRepoName(orgRepo)

	// Locate the repo entry. Writing is checked first to match the existing
	// GetRepoLocalPath order — a regression here would silently break
	// every read-only consumer (bip checkin, bip board, bip scout).
	entry, categoryRoot, ok := findRepoEntry(sources, cfg, orgRepo)
	if !ok {
		return ResolvedPath{}, fmt.Errorf("%w: %s", ErrRepoNotInSources, orgRepo)
	}

	layout := mergeLayouts(gcfg.Layout, entry.Layout)
	canonical := filepath.Join(categoryRoot, repoName)

	mode := layout.Mode
	if mode == "" {
		mode = config.LayoutModeClone
	}

	if mode == config.LayoutModeClone {
		return ResolvedPath{
			Path:  canonical,
			Mode:  config.LayoutModeClone,
			IsNew: !pathExists(canonical),
		}, nil
	}

	// Worktree mode. Empty context falls back to the canonical clone —
	// callers print a note so the degradation is visible.
	if ctx.IssueNumber == 0 && ctx.PRNumber == 0 {
		return ResolvedPath{
			Path:     canonical,
			Mode:     config.LayoutModeClone,
			IsNew:    !pathExists(canonical),
			FellBack: true,
		}, nil
	}

	rootTmpl := config.DefaultRootTemplate
	if layout.Worktree != nil && layout.Worktree.Root != "" {
		rootTmpl = layout.Worktree.Root
	}
	rootVars := map[string]string{
		"repo": repoName,
		"code": config.ExpandTilde(cfg.Paths.Code),
	}
	rootExpanded, err := expandTemplate(rootTmpl, rootVars)
	if err != nil {
		return ResolvedPath{}, fmt.Errorf("expanding worktree.root template %q: %w", rootTmpl, err)
	}
	root := config.ExpandTilde(rootExpanded)

	slotTmpl := config.DefaultSlotTemplate
	if layout.Worktree != nil && layout.Worktree.Slot != "" {
		slotTmpl = layout.Worktree.Slot
	}
	issueStr, prStr := "", ""
	if ctx.IssueNumber != 0 {
		issueStr = strconv.Itoa(ctx.IssueNumber)
	}
	if ctx.PRNumber != 0 {
		prStr = strconv.Itoa(ctx.PRNumber)
	}
	branch := ctx.Branch
	if branch == "" {
		switch {
		case ctx.IssueNumber != 0 && ctx.Slug != "":
			branch = fmt.Sprintf("%d-%s", ctx.IssueNumber, ctx.Slug)
		case ctx.IssueNumber != 0:
			branch = strconv.Itoa(ctx.IssueNumber)
		case ctx.PRNumber != 0 && ctx.Slug != "":
			branch = fmt.Sprintf("pr-%d-%s", ctx.PRNumber, ctx.Slug)
		case ctx.PRNumber != 0:
			branch = fmt.Sprintf("pr-%d", ctx.PRNumber)
		}
	}
	slotVars := map[string]string{
		"issue":  issueStr,
		"pr":     prStr,
		"slug":   ctx.Slug,
		"branch": branch,
	}
	slotExpanded, err := expandTemplate(slotTmpl, slotVars)
	if err != nil {
		return ResolvedPath{}, fmt.Errorf("expanding worktree.slot template %q: %w", slotTmpl, err)
	}

	path := filepath.Join(root, slotExpanded)
	return ResolvedPath{
		Path:   path,
		Mode:   config.LayoutModeWorktree,
		IsNew:  !pathExists(path),
		Branch: branch,
	}, nil
}

// findRepoEntry returns a pointer to the matching RepoEntry plus the
// category root (paths.writing for writing, paths.code for code). Writing is
// checked first to match the existing GetRepoLocalPath order.
func findRepoEntry(sources *Sources, cfg *Config, orgRepo string) (*RepoEntry, string, bool) {
	for i := range sources.Writing {
		if sources.Writing[i].Repo == orgRepo {
			return &sources.Writing[i], config.ExpandTilde(cfg.Paths.Writing), true
		}
	}
	for i := range sources.Code {
		if sources.Code[i].Repo == orgRepo {
			return &sources.Code[i], config.ExpandTilde(cfg.Paths.Code), true
		}
	}
	return nil, "", false
}

// mergeLayouts overlays a per-repo layout on top of a global layout, leaf by
// leaf. The result is always non-nil and never aliases either input.
func mergeLayouts(global, perRepo *config.LayoutConfig) config.LayoutConfig {
	var out config.LayoutConfig
	if global != nil {
		out.Mode = global.Mode
		if global.Worktree != nil {
			wt := *global.Worktree
			out.Worktree = &wt
		}
		if global.Clone != nil {
			cl := *global.Clone
			cl.Names = append([]string(nil), global.Clone.Names...)
			out.Clone = &cl
		}
	}
	if perRepo == nil {
		return out
	}
	if perRepo.Mode != "" {
		out.Mode = perRepo.Mode
	}
	if perRepo.Worktree != nil {
		if out.Worktree == nil {
			out.Worktree = &config.WorktreeLayout{}
		}
		if perRepo.Worktree.Root != "" {
			out.Worktree.Root = perRepo.Worktree.Root
		}
		if perRepo.Worktree.Slot != "" {
			out.Worktree.Slot = perRepo.Worktree.Slot
		}
	}
	if perRepo.Clone != nil {
		if out.Clone == nil {
			out.Clone = &config.CloneLayout{}
		}
		if perRepo.Clone.Root != "" {
			out.Clone.Root = perRepo.Clone.Root
		}
		if len(perRepo.Clone.Names) > 0 {
			out.Clone.Names = append([]string(nil), perRepo.Clone.Names...)
		}
	}
	return out
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ListWorktreeSlots scans root and returns the names of immediate
// subdirectories that match the worktree slot convention
// (config.WorktreeSlotPrefix, e.g. "issue-"). Results are sorted for
// deterministic test ordering.
//
// This is the inverse of the forward mapping ResolveRepoPath performs
// in worktree mode: ResolveRepoPath builds <root>/<slot> for a known
// issue, ListWorktreeSlots enumerates what is actually on disk. EPIC
// watch composes both: ResolveRepoPath (eventually, once .epic-config.json
// is deprecated) provides the root, ListWorktreeSlots enumerates the
// active issues.
//
// Returns an empty slice (no error) when root has no matching subdirs;
// returns an error when root cannot be read.
func ListWorktreeSlots(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), config.WorktreeSlotPrefix) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// templateVarRE matches {name} placeholders where name is a Go-identifier.
var templateVarRE = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// expandTemplate substitutes {name} placeholders using vars. Unknown
// placeholders yield an error; this is the lazy validation hook — callers
// only invoke expandTemplate when a template is actually used, so a typo in
// an opt-in worktree.slot does not break read-only commands.
func expandTemplate(tmpl string, vars map[string]string) (string, error) {
	var unknown []string
	out := templateVarRE.ReplaceAllStringFunc(tmpl, func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := vars[name]
		if !ok {
			unknown = append(unknown, name)
			return m
		}
		return v
	})
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return "", fmt.Errorf("unknown template variable(s): %s", strings.Join(dedupSorted(unknown), ", "))
	}
	return out, nil
}

// dedupSorted removes consecutive duplicates from an already-sorted slice.
func dedupSorted(s []string) []string {
	if len(s) <= 1 {
		return s
	}
	out := s[:1]
	for _, x := range s[1:] {
		if x != out[len(out)-1] {
			out = append(out, x)
		}
	}
	return out
}
