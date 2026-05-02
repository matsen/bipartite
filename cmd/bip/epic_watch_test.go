package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- helpers ---

// statusJSON returns a serialized .epic-status.json body with the given fields.
func statusJSON(t *testing.T, issue int, phase, summary string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"issue":   issue,
		"phase":   phase,
		"summary": summary,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// writeStatus atomically replaces a slot's .epic-status.json.
func writeStatus(t *testing.T, path string, issue int, phase, summary string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, statusJSON(t, issue, phase, summary), 0o644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatalf("rename: %v", err)
	}
}

// readLogEvents parses every JSONL line from .epic-notifications.log.
func readLogEvents(t *testing.T, logPath string) []epicEvent {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log: %v", err)
	}
	var out []epicEvent
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var ev epicEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("parse log line %q: %v", line, err)
		}
		out = append(out, ev)
	}
	return out
}

// stdoutLines returns non-empty lines from a captured stdout buffer.
func stdoutLines(buf *syncBuffer) []string {
	raw := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	var out []string
	for _, l := range raw {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// syncBuffer is a goroutine-safe bytes.Buffer wrapper for capturing watcher output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// runWatcherAsync starts runWatcher on a background goroutine and returns a
// cancel func that stops it cleanly. The watcher uses poll mode by default
// to avoid fsnotify-related test flakiness; pass pollInterval=0 to use fsnotify.
type watcherHandle struct {
	cancel context.CancelFunc
	done   chan error
	stdout *syncBuffer
	stderr *syncBuffer
}

func startWatcher(t *testing.T, slots []slotInfo, phases map[string]bool, logPath string, pollInterval time.Duration) *watcherHandle {
	t.Helper()
	stdout := &syncBuffer{}
	stderr := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runWatcher(ctx, watchConfig{
			slots:        slots,
			phases:       phases,
			pollInterval: pollInterval,
			logPath:      logPath,
			stdout:       stdout,
			stderr:       stderr,
		})
	}()
	return &watcherHandle{cancel: cancel, done: done, stdout: stdout, stderr: stderr}
}

func (h *watcherHandle) stop(t *testing.T) {
	t.Helper()
	h.cancel()
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not exit after cancel")
	}
}

// waitFor polls until cond returns true or the deadline elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition after %v", timeout)
}

// --- core transition behavior ---

func TestPhaseTransitionEmits(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 42, "coding", "hacking away")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	defer h.stop(t)

	// Wait for baseline read to complete.
	time.Sleep(150 * time.Millisecond)

	writeStatus(t, slot.statusPath, 42, "quality-gate", "PR open")

	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})

	events := readLogEvents(t, logPath)
	if len(events) != 1 {
		t.Fatalf("got %d log events, want 1: %+v", len(events), events)
	}
	got := events[0]
	if got.Slot != "alpha" || got.NewPhase != "quality-gate" || got.OldPhase == nil || *got.OldPhase != "coding" {
		t.Errorf("unexpected event: %+v", got)
	}
	if got.Issue != 42 || got.Summary != "PR open" {
		t.Errorf("unexpected fields: %+v", got)
	}

	lines := stdoutLines(h.stdout)
	if len(lines) != 1 {
		t.Fatalf("got %d stdout lines, want 1: %v", len(lines), lines)
	}
	wantPrefix := "alpha (i42): coding → quality-gate"
	if !strings.HasPrefix(lines[0], wantPrefix) {
		t.Errorf("stdout %q does not start with %q", lines[0], wantPrefix)
	}
}

func TestFirstReadIsBaselineNoEmit(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "needs-human", "stuck")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	h.stop(t)

	if events := readLogEvents(t, logPath); len(events) != 0 {
		t.Fatalf("expected zero events, got: %+v", events)
	}
	if lines := stdoutLines(h.stdout); len(lines) != 0 {
		t.Fatalf("expected zero stdout lines, got: %v", lines)
	}
}

func TestNonMilestoneSuppressed(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "exploring", "looking around")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	defer h.stop(t)

	time.Sleep(150 * time.Millisecond)
	writeStatus(t, slot.statusPath, 1, "coding", "making changes")
	time.Sleep(250 * time.Millisecond)

	if events := readLogEvents(t, logPath); len(events) != 0 {
		t.Fatalf("expected zero events for non-milestone transition, got: %+v", events)
	}
}

func TestRepeatedWriteSamePhase(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "v1")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	defer h.stop(t)

	time.Sleep(150 * time.Millisecond)

	writeStatus(t, slot.statusPath, 1, "needs-human", "v1")
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})

	writeStatus(t, slot.statusPath, 1, "needs-human", "v2")
	writeStatus(t, slot.statusPath, 1, "needs-human", "v3")
	time.Sleep(200 * time.Millisecond)

	if got := len(readLogEvents(t, logPath)); got != 1 {
		t.Fatalf("expected exactly one transition event, got %d", got)
	}
}

// --- log format and persistence ---

func TestLogLineIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 7, "coding", "")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	defer h.stop(t)

	time.Sleep(150 * time.Millisecond)
	gnarlySummary := "line1\nline2\twith\ttabs\nquoted: \"hello\" — emoji 🎉"
	writeStatus(t, slot.statusPath, 7, "completed", gnarlySummary)
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})

	events := readLogEvents(t, logPath)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	got := events[0]
	if _, err := time.Parse(time.RFC3339, got.Ts); err != nil {
		t.Errorf("ts %q is not RFC3339: %v", got.Ts, err)
	}
	if got.Summary != gnarlySummary {
		t.Errorf("summary did not round-trip: got %q want %q", got.Summary, gnarlySummary)
	}

	// The stdout line must be a single line: tabs/newlines collapsed.
	lines := stdoutLines(h.stdout)
	if len(lines) != 1 {
		t.Fatalf("expected single stdout line, got %d: %v", len(lines), lines)
	}
	if strings.ContainsAny(lines[0], "\t") {
		t.Errorf("stdout line still contains tab: %q", lines[0])
	}
}

func TestNotificationLogPersists(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	writeStatus(t, slot.statusPath, 1, "completed", "done")
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})
	h.stop(t)

	// After clean shutdown, the file is on disk and parses cleanly.
	events := readLogEvents(t, logPath)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].NewPhase != "completed" {
		t.Errorf("unexpected phase: %s", events[0].NewPhase)
	}
}

func TestStateRecoveryFromLog(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	// Pre-seed the log with a quality-gate transition for alpha.
	old := "coding"
	seed := epicEvent{
		Ts:       time.Now().UTC().Format(time.RFC3339),
		Slot:     "alpha",
		Issue:    99,
		OldPhase: &old,
		NewPhase: "quality-gate",
		Summary:  "from prior watcher run",
	}
	b, _ := json.Marshal(seed)
	if err := os.WriteFile(logPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	// Status file says the same phase the log already recorded.
	writeStatus(t, slot.statusPath, 99, "quality-gate", "still here")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	h.stop(t)

	events := readLogEvents(t, logPath)
	if len(events) != 1 {
		t.Fatalf("expected exactly the seeded event (no new ones), got %d: %+v", len(events), events)
	}
}

// --- --since replay ---

func TestSinceFlagReplaysFromLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, epicNotificationsName)

	now := time.Now().UTC()
	mkLine := func(offset time.Duration, phase string) string {
		old := "coding"
		ev := epicEvent{
			Ts:       now.Add(offset).Format(time.RFC3339),
			Slot:     "alpha",
			Issue:    1,
			OldPhase: &old,
			NewPhase: phase,
			Summary:  "x",
		}
		b, _ := json.Marshal(ev)
		return string(b) + "\n"
	}
	body := mkLine(-2*time.Hour, "needs-human") +
		mkLine(-30*time.Minute, "quality-gate") +
		mkLine(-1*time.Minute, "completed")
	if err := os.WriteFile(logPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cutoff := time.Now().Add(-1 * time.Hour)
	if err := replaySince(logPath, cutoff, &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 replayed lines, got %d: %v", len(lines), lines)
	}

	// Log file is unchanged.
	after, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != body {
		t.Errorf("log file was modified by replay")
	}
}

func TestSinceBoundaryFuture(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, epicNotificationsName)

	old := "coding"
	ev := epicEvent{
		Ts:       time.Now().UTC().Format(time.RFC3339),
		Slot:     "alpha",
		Issue:    1,
		OldPhase: &old,
		NewPhase: "completed",
		Summary:  "x",
	}
	b, _ := json.Marshal(ev)
	if err := os.WriteFile(logPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	// --since=-1h pushes the cutoff one hour into the future.
	dur, _ := time.ParseDuration("-1h")
	cutoff := time.Now().Add(-dur)
	var buf bytes.Buffer
	if err := replaySince(logPath, cutoff, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for future cutoff, got: %q", buf.String())
	}
}

func TestSinceBoundaryAll(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, epicNotificationsName)

	now := time.Now().UTC()
	var body strings.Builder
	for i, off := range []time.Duration{-3 * time.Hour, -2 * time.Hour, -1 * time.Hour} {
		old := "coding"
		ev := epicEvent{
			Ts:       now.Add(off).Format(time.RFC3339),
			Slot:     fmt.Sprintf("slot%d", i),
			Issue:    i,
			OldPhase: &old,
			NewPhase: "completed",
			Summary:  "x",
		}
		b, _ := json.Marshal(ev)
		body.Write(b)
		body.WriteByte('\n')
	}
	if err := os.WriteFile(logPath, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := replaySince(logPath, time.Now().Add(-1000*time.Hour), &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 replayed lines, got %d: %v", len(lines), lines)
	}
}

// --- config discovery and error handling ---

func TestConfigDiscoveryCloneMode(t *testing.T) {
	dir := t.TempDir()
	cfg := map[string]any{
		"clone_root":  filepath.Join(dir, "clones"),
		"clone_names": []string{"alpha", "beta"},
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(dir, epicConfigName), cfgBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := loadEpicConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := resolveSlots(dir, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	wantPaths := map[string]string{
		"alpha": filepath.Join(dir, "clones", "alpha", epicStatusName),
		"beta":  filepath.Join(dir, "clones", "beta", epicStatusName),
	}
	for _, s := range slots {
		if want := wantPaths[s.name]; s.statusPath != want {
			t.Errorf("slot %q path = %q, want %q", s.name, s.statusPath, want)
		}
	}
}

func TestConfigDiscoveryWorktreeMode(t *testing.T) {
	dir := t.TempDir()
	cloneRoot := filepath.Join(dir, "workers")
	for _, sub := range []string{"issue-100", "issue-200", "scratch"} {
		if err := os.MkdirAll(filepath.Join(cloneRoot, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := map[string]any{
		"clone_root":      cloneRoot,
		"local_worktrees": true,
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(dir, epicConfigName), cfgBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := loadEpicConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := resolveSlots(dir, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 issue-* slots, got %d (%+v)", len(slots), slots)
	}
	names := map[string]bool{}
	for _, s := range slots {
		names[s.name] = true
	}
	if !names["issue-100"] || !names["issue-200"] {
		t.Errorf("expected issue-100 and issue-200, got %+v", names)
	}
	if names["scratch"] {
		t.Errorf("non-issue subdirectory was included")
	}
}

func TestMissingConfigFailsFast(t *testing.T) {
	dir := t.TempDir()
	_, err := loadEpicConfig(dir)
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
	if !strings.Contains(err.Error(), epicConfigName) {
		t.Errorf("error %q does not name %q", err, epicConfigName)
	}
}

func TestMutuallyExclusiveModes(t *testing.T) {
	dir := t.TempDir()
	cfg := map[string]any{
		"clone_root":      "/tmp",
		"clone_names":     []string{"a"},
		"local_worktrees": true,
	}
	b, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(dir, epicConfigName), b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEpicConfig(dir); err == nil {
		t.Fatal("expected error for clone_names + local_worktrees, got nil")
	}
}

func TestMalformedStatusFileSkipped(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	if err := os.MkdirAll(filepath.Dir(slot.statusPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(slot.statusPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

	// Now write a valid file: baseline read happens, then a transition.
	writeStatus(t, slot.statusPath, 1, "coding", "")
	time.Sleep(150 * time.Millisecond)
	writeStatus(t, slot.statusPath, 1, "completed", "done")
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})

	h.stop(t)

	if h.stderr.String() == "" {
		t.Errorf("expected a parse warning on stderr, got empty")
	}
	if events := readLogEvents(t, logPath); len(events) != 1 {
		t.Fatalf("expected exactly 1 event after recovery, got %d", len(events))
	}
}

func TestStatusFileDeletedDuringWatch(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "")

	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 50*time.Millisecond)
	defer h.stop(t)

	time.Sleep(150 * time.Millisecond)
	if err := os.Remove(slot.statusPath); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)

	// Recreate with a milestone phase and verify a transition is emitted.
	writeStatus(t, slot.statusPath, 1, "completed", "back")
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})
	events := readLogEvents(t, logPath)
	if len(events) != 1 || events[0].NewPhase != "completed" {
		t.Fatalf("unexpected events after recreate: %+v", events)
	}
}

func TestParentDirectoryWatched(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "")

	// Use fsnotify (pollInterval=0).
	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 0)
	defer h.stop(t)

	time.Sleep(200 * time.Millisecond)

	// Delete and recreate: a watch on the file alone would now be dead.
	if err := os.Remove(slot.statusPath); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	writeStatus(t, slot.statusPath, 1, "completed", "back")

	waitFor(t, 3*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})
}

// --- filter and poll ---

func TestPhasesFlagOverride(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "exploring", "")
	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter("coding,testing"), logPath, 50*time.Millisecond)
	defer h.stop(t)

	time.Sleep(150 * time.Millisecond)

	// Transition to coding (in the override list) — should emit.
	writeStatus(t, slot.statusPath, 1, "coding", "")
	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})

	// Transition to needs-human (NOT in the override list) — should NOT emit.
	writeStatus(t, slot.statusPath, 1, "needs-human", "")
	time.Sleep(250 * time.Millisecond)

	events := readLogEvents(t, logPath)
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 event (coding only), got %d: %+v", len(events), events)
	}
	if events[0].NewPhase != "coding" {
		t.Errorf("unexpected event phase: %s", events[0].NewPhase)
	}
}

func TestPollFallback(t *testing.T) {
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "")
	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 100*time.Millisecond)
	defer h.stop(t)

	time.Sleep(250 * time.Millisecond)
	writeStatus(t, slot.statusPath, 1, "completed", "done")

	waitFor(t, 2*time.Second, func() bool {
		return len(readLogEvents(t, logPath)) >= 1
	})
}

func TestSigtermFlushesLog(t *testing.T) {
	// We can't actually send SIGTERM to the test process. Instead, simulate
	// the same path: cancel the context mid-stream and confirm the log file
	// has no torn lines and parses cleanly.
	dir := t.TempDir()
	slot := slotInfo{name: "alpha", statusPath: filepath.Join(dir, "alpha", epicStatusName)}
	logPath := filepath.Join(dir, epicNotificationsName)

	writeStatus(t, slot.statusPath, 1, "coding", "")
	h := startWatcher(t, []slotInfo{slot}, parsePhasesFilter(defaultEpicWatchPhases), logPath, 30*time.Millisecond)
	time.Sleep(80 * time.Millisecond)

	// Trigger a transition then cancel almost immediately.
	writeStatus(t, slot.statusPath, 1, "completed", "rapid shutdown")
	time.Sleep(60 * time.Millisecond)
	h.stop(t)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var ev epicEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("torn or unparseable line %q: %v", line, err)
		}
	}
}

// --- formatting helpers ---

func TestFormatEventLineCollapsesWhitespace(t *testing.T) {
	old := "coding"
	ev := epicEvent{
		Slot:     "alpha",
		Issue:    1,
		OldPhase: &old,
		NewPhase: "completed",
		Summary:  "line1\nline2\twith\ttabs",
	}
	got := formatEventLine(ev)
	if strings.ContainsAny(got, "\n\t") {
		t.Errorf("output still contains tab/newline: %q", got)
	}
}

// Ensure io.Writer interface compliance for syncBuffer at compile time.
var _ io.Writer = (*syncBuffer)(nil)
