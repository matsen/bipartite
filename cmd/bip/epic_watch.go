package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

const (
	epicConfigName        = ".epic-config.json"
	epicStatusName        = ".epic-status.json"
	epicNotificationsName = ".epic-notifications.log"

	defaultEpicWatchPhases = "needs-human,completed,awaiting-results,quality-gate"
	defaultPollInterval    = "2s"
)

var (
	epicWatchPhases string
	epicWatchSince  string
	epicWatchPoll   string
)

var epicWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch .epic-status.json files for phase transitions",
	Long: `Watch .epic-status.json files across all configured slots and emit
phase-transition events to .epic-notifications.log (one JSONL line per
event) and to stdout (one human-readable line per event).

Reads .epic-config.json from the current working directory to discover
slots. Watches each slot's parent directory using fsnotify, or falls back
to stat polling when --poll is set (e.g. for NFS-mounted clone roots
where inotify does not fire on remote writes).

Run as a long-lived process. Exits cleanly on SIGINT or SIGTERM.

Examples:
  bip epic watch
  bip epic watch --phases needs-human,completed
  bip epic watch --since 30m
  bip epic watch --poll=100ms`,
	RunE: runEpicWatch,
}

func init() {
	epicWatchCmd.Flags().StringVar(&epicWatchPhases, "phases", defaultEpicWatchPhases,
		"Comma-separated phases to alert on")
	epicWatchCmd.Flags().StringVar(&epicWatchSince, "since", "",
		"Replay log entries newer than DURATION to stdout, then exit (e.g. 30m, 2h)")
	epicWatchCmd.Flags().StringVar(&epicWatchPoll, "poll", "",
		"Use stat polling at DURATION instead of fsnotify (default 2s)")
	epicWatchCmd.Flags().Lookup("poll").NoOptDefVal = defaultPollInterval
	epicCmd.AddCommand(epicWatchCmd)
}

// epicConfig mirrors the relevant fields of .epic-config.json.
type epicConfig struct {
	CloneRoot      string   `json:"clone_root"`
	CloneNames     []string `json:"clone_names"`
	LocalWorktrees bool     `json:"local_worktrees"`
}

// epicStatus mirrors the .epic-status.json fields the watcher reads.
type epicStatus struct {
	Issue   int    `json:"issue"`
	Phase   string `json:"phase"`
	Summary string `json:"summary"`
}

// epicEvent is the JSONL schema written to .epic-notifications.log.
type epicEvent struct {
	Ts       string  `json:"ts"`
	Slot     string  `json:"slot"`
	Issue    int     `json:"issue"`
	OldPhase *string `json:"old_phase"`
	NewPhase string  `json:"new_phase"`
	Summary  string  `json:"summary"`
}

// slotInfo identifies a single slot the watcher tracks.
type slotInfo struct {
	name       string // e.g. "alpha" (clone mode) or "issue-100" (worktree mode)
	statusPath string // absolute path to .epic-status.json
}

// watchConfig is the testable input to runWatcher.
type watchConfig struct {
	slots        []slotInfo
	phases       map[string]bool
	pollInterval time.Duration // 0 means use fsnotify
	logPath      string
	stdout       io.Writer
	stderr       io.Writer
}

func runEpicWatch(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		exitWithError(ExitError, "getting cwd: %v", err)
	}

	cfg, err := loadEpicConfig(cwd)
	if err != nil {
		exitWithError(ExitConfigError, "%v", err)
	}

	slots, err := resolveSlots(cwd, cfg)
	if err != nil {
		exitWithError(ExitConfigError, "%v", err)
	}

	logPath := filepath.Join(cwd, epicNotificationsName)

	if epicWatchSince != "" {
		dur, err := time.ParseDuration(epicWatchSince)
		if err != nil {
			exitWithError(ExitError, "invalid --since duration %q: %v", epicWatchSince, err)
		}
		cutoff := time.Now().Add(-dur)
		return replaySince(logPath, cutoff, os.Stdout)
	}

	var pollInterval time.Duration
	if epicWatchPoll != "" {
		d, err := time.ParseDuration(epicWatchPoll)
		if err != nil {
			exitWithError(ExitError, "invalid --poll duration %q: %v", epicWatchPoll, err)
		}
		if d <= 0 {
			exitWithError(ExitError, "invalid --poll duration %q: must be positive", epicWatchPoll)
		}
		pollInterval = d
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return runWatcher(ctx, watchConfig{
		slots:        slots,
		phases:       parsePhasesFilter(epicWatchPhases),
		pollInterval: pollInterval,
		logPath:      logPath,
		stdout:       os.Stdout,
		stderr:       os.Stderr,
	})
}

// loadEpicConfig reads and validates .epic-config.json from dir.
func loadEpicConfig(dir string) (*epicConfig, error) {
	path := filepath.Join(dir, epicConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("missing %s in %s", epicConfigName, dir)
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg epicConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.CloneRoot == "" {
		return nil, fmt.Errorf("%s: clone_root is required", path)
	}
	if cfg.LocalWorktrees && len(cfg.CloneNames) > 0 {
		return nil, fmt.Errorf("%s: local_worktrees and clone_names are mutually exclusive", path)
	}
	if !cfg.LocalWorktrees && len(cfg.CloneNames) == 0 {
		return nil, fmt.Errorf("%s: must set local_worktrees: true or clone_names: [...]", path)
	}
	return &cfg, nil
}

// resolveSlots returns the slots to watch given the config.
// Clone mode: one slot per entry in clone_names.
// Worktree mode: one slot per existing issue-* subdirectory of clone_root.
func resolveSlots(repoDir string, cfg *epicConfig) ([]slotInfo, error) {
	cloneRoot := expandPath(cfg.CloneRoot)
	if !filepath.IsAbs(cloneRoot) {
		cloneRoot = filepath.Join(repoDir, cloneRoot)
	}

	if cfg.LocalWorktrees {
		entries, err := os.ReadDir(cloneRoot)
		if err != nil {
			return nil, fmt.Errorf("reading clone_root %s: %w", cloneRoot, err)
		}
		var slots []slotInfo
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || !strings.HasPrefix(name, "issue-") {
				continue
			}
			slots = append(slots, slotInfo{
				name:       name,
				statusPath: filepath.Join(cloneRoot, name, epicStatusName),
			})
		}
		return slots, nil
	}

	var slots []slotInfo
	for _, name := range cfg.CloneNames {
		slots = append(slots, slotInfo{
			name:       name,
			statusPath: filepath.Join(cloneRoot, name, epicStatusName),
		})
	}
	return slots, nil
}

// expandPath expands a leading ~/ to the user's home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// parsePhasesFilter parses the comma-separated --phases value.
// An empty string yields an empty filter (matches no phase).
func parsePhasesFilter(s string) map[string]bool {
	out := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out[p] = true
	}
	return out
}

// readStatus loads and validates a slot's .epic-status.json.
func readStatus(path string) (*epicStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s epicStatus
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Phase == "" {
		return nil, errors.New("missing phase")
	}
	return &s, nil
}

// seedLastPhases reads the notifications log and returns the most recent
// new_phase per slot. Missing log returns an empty map.
func seedLastPhases(logPath string) (map[string]string, error) {
	last := map[string]string{}
	f, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return last, nil
		}
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for dec.More() {
		var ev epicEvent
		if err := dec.Decode(&ev); err != nil {
			break
		}
		last[ev.Slot] = ev.NewPhase
	}
	return last, nil
}

// replaySince writes log entries with timestamp >= now-DURATION to w.
// It does not append to the log.
func replaySince(logPath string, cutoff time.Time, w io.Writer) error {
	f, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for dec.More() {
		var ev epicEvent
		if err := dec.Decode(&ev); err != nil {
			break
		}
		ts, err := time.Parse(time.RFC3339, ev.Ts)
		if err != nil {
			continue
		}
		if ts.Before(cutoff) {
			continue
		}
		fmt.Fprintln(w, formatEventLine(ev))
	}
	return nil
}

// formatEventLine renders an event as a single human-readable stdout line.
// Tabs and newlines in summary are collapsed to spaces.
func formatEventLine(ev epicEvent) string {
	summary := strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, ev.Summary)
	old := "<none>"
	if ev.OldPhase != nil {
		old = *ev.OldPhase
	}
	return fmt.Sprintf("%s (i%d): %s → %s — %s",
		ev.Slot, ev.Issue, old, ev.NewPhase, summary)
}

// runWatcher seeds state from the log, performs an initial baseline
// read of every slot, and then either polls or uses fsnotify to detect
// phase transitions until ctx is cancelled.
func runWatcher(ctx context.Context, cfg watchConfig) error {
	lastPhase, err := seedLastPhases(cfg.logPath)
	if err != nil {
		fmt.Fprintf(cfg.stderr, "warning: reading notifications log: %v\n", err)
		lastPhase = map[string]string{}
	}

	logFile, err := os.OpenFile(cfg.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening notifications log: %w", err)
	}
	defer logFile.Close()

	for _, s := range cfg.slots {
		processStatus(s, lastPhase, cfg, logFile)
	}

	if cfg.pollInterval > 0 {
		return runPollLoop(ctx, cfg, lastPhase, logFile)
	}
	return runFsnotifyLoop(ctx, cfg, lastPhase, logFile)
}

// processStatus reads the slot's status file and emits a transition event
// when the phase has changed and matches the filter. A slot first observed
// without a prior phase has its phase recorded as a baseline (no emission).
func processStatus(s slotInfo, lastPhase map[string]string, cfg watchConfig, logFile *os.File) {
	status, err := readStatus(s.statusPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(cfg.stderr, "warning: reading %s: %v\n", s.statusPath, err)
		}
		return
	}
	prev, hadPrev := lastPhase[s.name]
	if !hadPrev {
		lastPhase[s.name] = status.Phase
		return
	}
	if prev == status.Phase {
		return
	}
	if !cfg.phases[status.Phase] {
		// Non-milestone transition: silently update state.
		lastPhase[s.name] = status.Phase
		return
	}
	old := prev
	ev := epicEvent{
		Ts:       time.Now().UTC().Format(time.RFC3339),
		Slot:     s.name,
		Issue:    status.Issue,
		OldPhase: &old,
		NewPhase: status.Phase,
		Summary:  status.Summary,
	}
	emitEvent(ev, logFile, cfg.stdout, cfg.stderr)
	lastPhase[s.name] = status.Phase
}

// emitEvent appends a JSONL entry to the log and prints a human-readable
// line to stdout. A failure to write either channel is reported to stderr
// but never aborts the watcher.
func emitEvent(ev epicEvent, logFile *os.File, stdout, stderr io.Writer) {
	line, err := json.Marshal(ev)
	if err != nil {
		fmt.Fprintf(stderr, "warning: marshaling event: %v\n", err)
		return
	}
	line = append(line, '\n')
	if _, err := logFile.Write(line); err != nil {
		fmt.Fprintf(stderr, "warning: writing notifications log: %v\n", err)
	}
	fmt.Fprintln(stdout, formatEventLine(ev))
}

func runFsnotifyLoop(ctx context.Context, cfg watchConfig, lastPhase map[string]string, logFile *os.File) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	defer w.Close()

	parentToSlots := map[string][]slotInfo{}
	for _, s := range cfg.slots {
		parent := filepath.Dir(s.statusPath)
		parentToSlots[parent] = append(parentToSlots[parent], s)
	}
	for parent := range parentToSlots {
		if err := w.Add(parent); err != nil {
			fmt.Fprintf(cfg.stderr, "warning: watching %s: %v\n", parent, err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if filepath.Base(ev.Name) != epicStatusName {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			for _, s := range parentToSlots[filepath.Dir(ev.Name)] {
				if s.statusPath == ev.Name {
					processStatus(s, lastPhase, cfg, logFile)
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(cfg.stderr, "warning: fsnotify error: %v\n", err)
		}
	}
}

func runPollLoop(ctx context.Context, cfg watchConfig, lastPhase map[string]string, logFile *os.File) error {
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, s := range cfg.slots {
				processStatus(s, lastPhase, cfg, logFile)
			}
		}
	}
}
