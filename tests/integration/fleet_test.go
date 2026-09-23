package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fleetConductor returns a conductor dir whose .epic-config.json names one
// clone under a clone root, plus a fake HOME. extra is merged into the
// config's JSON object.
func fleetConductor(t *testing.T, cloneRoot, extra string) (conductor, home string) {
	t.Helper()
	base := t.TempDir()
	conductor = filepath.Join(base, "conductor")
	home = filepath.Join(base, "home")
	for _, d := range []string{conductor, filepath.Join(home, ".claude", "skills", "lib")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := `{"clone_root": "` + cloneRoot + `", "clone_names": ["a"]` + extra + `}`
	writeFile(t, filepath.Join(conductor, ".epic-config.json"), cfg, 0644)
	return conductor, home
}

// stubScript installs home/.claude/skills/lib/<name> printing its
// arguments and exiting code.
func stubScript(t *testing.T, home, name string, code int) {
	t.Helper()
	body := "echo \"args: $*\"\nexit " + strconv.Itoa(code) + "\n"
	writeFile(t, filepath.Join(home, ".claude", "skills", "lib", name), body, 0644)
}

func runFleet(t *testing.T, dir, home string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(getBPBinary(t), append([]string{"fleet"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestFleetScriptExitCodes(t *testing.T) {
	for _, tc := range []struct{ sub, script string }{
		{"collisions", "fleet-collisions.sh"},
		{"currency", "clone-currency.sh"},
	} {
		for _, code := range []int{0, 1, 2} {
			t.Run(tc.sub+"/"+strconv.Itoa(code), func(t *testing.T) {
				conductor, home := fleetConductor(t, "/pool", "")
				stubScript(t, home, tc.script, code)
				if _, got := runFleet(t, conductor, home, tc.sub); got != code {
					t.Errorf("exit = %d, want %d", got, code)
				}
			})
		}
		t.Run(tc.sub+"/no script", func(t *testing.T) {
			conductor, home := fleetConductor(t, "/pool", "")
			out, got := runFleet(t, conductor, home, tc.sub)
			if got != 2 || !strings.Contains(out, "make symlink-skills") {
				t.Errorf("exit = %d, output %q; want 2 naming make symlink-skills", got, out)
			}
		})
		t.Run(tc.sub+"/no config", func(t *testing.T) {
			_, home := fleetConductor(t, "/pool", "")
			stubScript(t, home, tc.script, 0)
			if _, got := runFleet(t, t.TempDir(), home, tc.sub); got != 2 {
				t.Errorf("exit = %d, want 2", got)
			}
		})
	}
}

func TestFleetScriptScope(t *testing.T) {
	t.Run("tilde clone_root is expanded", func(t *testing.T) {
		conductor, home := fleetConductor(t, "~/pool", "")
		stubScript(t, home, "fleet-collisions.sh", 0)
		out, _ := runFleet(t, conductor, home, "collisions")
		if want := "args: " + filepath.Join(home, "pool"); !strings.Contains(out, want) {
			t.Errorf("output %q, want %q", out, want)
		}
	})
	t.Run("currency gets the conductor and the root", func(t *testing.T) {
		conductor, home := fleetConductor(t, "/pool", "")
		stubScript(t, home, "clone-currency.sh", 0)
		out, _ := runFleet(t, conductor, home, "currency")
		if want := "args: " + conductor + " /pool"; !strings.Contains(out, want) {
			t.Errorf("output %q, want %q", out, want)
		}
	})
	t.Run("frozen worker copy is refused", func(t *testing.T) {
		conductor, home := fleetConductor(t, "/pool", `, "main_checkout": "/elsewhere"`)
		stubScript(t, home, "fleet-collisions.sh", 0)
		out, got := runFleet(t, conductor, home, "collisions")
		if got != 2 || strings.Contains(out, "args:") {
			t.Errorf("exit = %d, output %q; want 2 without running the script", got, out)
		}
	})
	t.Run("main_checkout naming this dir passes", func(t *testing.T) {
		conductor, home := fleetConductor(t, "/pool", "")
		writeFile(t, filepath.Join(conductor, ".epic-config.json"),
			`{"clone_root": "/pool", "clone_names": ["a"], "main_checkout": "`+conductor+`"}`, 0644)
		stubScript(t, home, "fleet-collisions.sh", 0)
		if out, got := runFleet(t, conductor, home, "collisions"); got != 0 {
			t.Errorf("exit = %d, output %q; want 0", got, out)
		}
	})
}

// TestFleetWatchers runs real watchers and checks fleet_watchers finds
// exactly the ones in its conductor dir, under either command name.
func TestFleetWatchers(t *testing.T) {
	bin := t.TempDir()
	if err := os.Symlink(getBPBinary(t), filepath.Join(bin, "bip")); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a"), 0755); err != nil {
		t.Fatal(err)
	}
	here, _ := fleetConductor(t, root, "")
	other, _ := fleetConductor(t, root, "")

	bip := filepath.Join(bin, "bip")
	start := func(dir string, args ...string) int {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
		return cmd.Process.Pid
	}
	want := []string{
		strconv.Itoa(start(here, bip, "fleet", "watch")),
		strconv.Itoa(start(here, bip, "epic", "watch", "--poll")),
	}
	sort.Strings(want)
	start(other, bip, "fleet", "watch")
	// A worker whose command line quotes the watcher is not one.
	start(here, "sh", "-c", "sleep 30 # bip fleet watch")
	time.Sleep(300 * time.Millisecond)

	shells := []string{"bash"}
	if _, err := exec.LookPath("zsh"); err == nil {
		shells = append(shells, "zsh")
	}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			out, err := exec.Command(shell, "-c", spawnIntentCall(t, "fleet_watchers", here)).Output()
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Fields(string(out))
			sort.Strings(got)
			if strings.Join(got, " ") != strings.Join(want, " ") {
				t.Errorf("fleet_watchers = %v, want %v", got, want)
			}
		})
	}
}
