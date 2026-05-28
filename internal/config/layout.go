package config

// LayoutConfig describes how bip resolves working directories for repos.
// It can appear in two places:
//   - ~/.config/bip/config.yml under `layout:` (per-machine default; applies
//     to every repo unless overridden).
//   - sources.yml on a `RepoEntry` (per-repo override).
//
// Each leaf field resolves independently: per-repo settings override global,
// and any field left zero inherits from the next tier. The block is opt-in;
// when absent everywhere the resolver behaves identically to pre-issue-149.
type LayoutConfig struct {
	// Mode selects the layout: LayoutModeClone (default) or LayoutModeWorktree.
	Mode string `yaml:"mode,omitempty"`

	// Worktree configures worktree-mode resolution.
	Worktree *WorktreeLayout `yaml:"worktree,omitempty"`

	// Clone configures clone-mode resolution.
	Clone *CloneLayout `yaml:"clone,omitempty"`
}

// WorktreeLayout configures worktree-mode resolution.
//
// Root is a template for the directory under which per-issue worktrees live.
// Slot is a template for the per-issue directory name. See the package doc
// on expandTemplate (in internal/flow) for the supported variables.
type WorktreeLayout struct {
	Root string `yaml:"root,omitempty"`
	Slot string `yaml:"slot,omitempty"`
}

// CloneLayout configures clone-mode resolution.
//
// Root, when set, overrides paths.code as the parent directory of the
// canonical clone. Names is an EPIC-style multi-clone pool (used by
// epic_watch); an empty list means "single canonical clone."
type CloneLayout struct {
	Root  string   `yaml:"root,omitempty"`
	Names []string `yaml:"names,omitempty"`
}

// Layout mode values.
const (
	LayoutModeClone    = "clone"
	LayoutModeWorktree = "worktree"
)

// DefaultSlotTemplate is the slot template used when worktree.slot is unset.
// It matches the EPIC convention of issue-<N> directories.
const DefaultSlotTemplate = "issue-{issue}"

// DefaultRootTemplate is the worktree.root template used when worktree.root
// is unset. {code} expands to paths.code from $NEXUS_PATH/config.yml.
const DefaultRootTemplate = "{code}/{repo}-workers"

// WorktreeSlotPrefix is the directory-name prefix used by the default slot
// template. EPIC's worktree-discovery scan (cmd/bip/epic_watch.go) keys off
// this prefix when filtering issue-* subdirectories.
const WorktreeSlotPrefix = "issue-"
