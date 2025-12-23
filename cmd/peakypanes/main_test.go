package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestExtractRepoName(t *testing.T) {
	cases := map[string]string{
		"https://github.com/user/repo.git": "repo",
		"git@github.com:user/repo":         "repo",
		"https://example.com/user/repo/":   "repo",
	}
	for input, want := range cases {
		if got := extractRepoName(input); got != want {
			t.Fatalf("extractRepoName(%q) = %q", input, got)
		}
	}
}

func TestGridSplitPercent(t *testing.T) {
	if gridSplitPercent(1) != 0 {
		t.Fatalf("gridSplitPercent(1) expected 0")
	}
	if gridSplitPercent(2) != 50 {
		t.Fatalf("gridSplitPercent(2) expected 50")
	}
	if gridSplitPercent(3) == 0 {
		t.Fatalf("gridSplitPercent(3) expected > 0")
	}
}

func TestResolveGridCommandsAndTitles(t *testing.T) {
	cfg := &layout.LayoutConfig{
		Command:  "echo",
		Commands: []string{"a", "b"},
		Titles:   []string{"one"},
	}
	commands := resolveGridCommands(cfg, 3)
	if !reflect.DeepEqual(commands, []string{"a", "b", "echo"}) {
		t.Fatalf("resolveGridCommands() = %#v", commands)
	}
	titles := resolveGridTitles(cfg, 3)
	if !reflect.DeepEqual(titles, []string{"one", "", ""}) {
		t.Fatalf("resolveGridTitles() = %#v", titles)
	}
}

func TestExpandUserPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := expandUserPath("~"); got != home {
		t.Fatalf("expandUserPath(~) = %q", got)
	}
	if got := expandUserPath("~/projects"); got != filepath.Join(home, "projects") {
		t.Fatalf("expandUserPath(~/projects) = %q", got)
	}
}

func TestSanitizeSessionName(t *testing.T) {
	if got := sanitizeSessionName(" My Project "); got != "my-project" {
		t.Fatalf("sanitizeSessionName() = %q", got)
	}
	if got := sanitizeSessionName(""); got != "session" {
		t.Fatalf("sanitizeSessionName(empty) = %q", got)
	}
}

func TestInsideTmux(t *testing.T) {
	origTTY := currentTTYFn
	origHas := hasClientOnTTYFn
	defer func() {
		currentTTYFn = origTTY
		hasClientOnTTYFn = origHas
	}()
	currentTTYFn = func() string { return "/dev/ttys001" }

	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	if insideTmux() {
		t.Fatalf("insideTmux() should be false")
	}
	t.Setenv("TMUX", "/tmp/tmux")
	t.Setenv("TMUX_PANE", "%1")
	hasClientOnTTYFn = func(ctx context.Context, tty string) (bool, error) { return true, nil }
	if !insideTmux() {
		t.Fatalf("insideTmux() should be true")
	}
	hasClientOnTTYFn = func(ctx context.Context, tty string) (bool, error) { return false, nil }
	if insideTmux() {
		t.Fatalf("insideTmux() should be false when no client matches")
	}
}

func TestCurrentDir(t *testing.T) {
	if dir := currentDir(); strings.TrimSpace(dir) == "" {
		t.Fatalf("currentDir() empty")
	}
}

func TestRunLayoutsHelp(t *testing.T) {
	out := captureStdout(func() {
		runLayouts([]string{"--help"})
	})
	if !strings.Contains(out, "List and manage layouts") {
		t.Fatalf("runLayouts(--help) output = %q", out)
	}
}

func TestRunDashboardCommandHelp(t *testing.T) {
	origRunMenu := runMenuFn
	origPopup := runDashboardPopupFn
	origHosted := runDashboardHostedFn
	defer func() {
		runMenuFn = origRunMenu
		runDashboardPopupFn = origPopup
		runDashboardHostedFn = origHosted
	}()

	var called bool
	runMenuFn = func() { called = true }
	runDashboardPopupFn = func(_ []string) { called = true }
	runDashboardHostedFn = func(_ string) { called = true }

	out := captureStdout(func() {
		runDashboardCommand([]string{"--help"})
	})
	if called {
		t.Fatalf("runDashboardCommand(--help) should not invoke handlers")
	}
	if !strings.Contains(out, "Open the Peaky Panes dashboard UI") {
		t.Fatalf("runDashboardCommand(--help) output = %q", out)
	}
}

func TestRunDashboardCommandPopup(t *testing.T) {
	origRunMenu := runMenuFn
	origPopup := runDashboardPopupFn
	origHosted := runDashboardHostedFn
	defer func() {
		runMenuFn = origRunMenu
		runDashboardPopupFn = origPopup
		runDashboardHostedFn = origHosted
	}()

	var called string
	runMenuFn = func() { called = "menu" }
	runDashboardPopupFn = func(_ []string) { called = "popup" }
	runDashboardHostedFn = func(_ string) { called = "hosted" }

	runDashboardCommand([]string{"--popup"})
	if called != "popup" {
		t.Fatalf("runDashboardCommand(--popup) called %q", called)
	}
}

func TestRunDashboardCommandHostedSession(t *testing.T) {
	origRunMenu := runMenuFn
	origPopup := runDashboardPopupFn
	origHosted := runDashboardHostedFn
	defer func() {
		runMenuFn = origRunMenu
		runDashboardPopupFn = origPopup
		runDashboardHostedFn = origHosted
	}()

	var session string
	runMenuFn = func() {}
	runDashboardPopupFn = func(_ []string) {}
	runDashboardHostedFn = func(name string) { session = name }

	runDashboardCommand([]string{"--tmux-session", "--session", "my-dash"})
	if session != "my-dash" {
		t.Fatalf("runDashboardCommand(--tmux-session) session = %q", session)
	}
}

func TestRunDashboardCommandConflict(t *testing.T) {
	origFatal := fatalFn
	origRunMenu := runMenuFn
	origPopup := runDashboardPopupFn
	origHosted := runDashboardHostedFn
	defer func() {
		fatalFn = origFatal
		runMenuFn = origRunMenu
		runDashboardPopupFn = origPopup
		runDashboardHostedFn = origHosted
	}()

	var got string
	fatalFn = func(format string, args ...interface{}) {
		got = format
	}
	runMenuFn = func() {}
	runDashboardPopupFn = func(_ []string) {}
	runDashboardHostedFn = func(_ string) {}

	runDashboardCommand([]string{"--popup", "--tmux-session"})
	if !strings.Contains(got, "choose either --popup or --tmux-session") {
		t.Fatalf("fatalFn message = %q", got)
	}
}

func TestRunInitLocal(t *testing.T) {
	root := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir() error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	out := captureStdout(func() {
		runInit([]string{"--local", "--layout", "dev-3", "--force"})
	})
	if !strings.Contains(out, "Created") {
		t.Fatalf("runInit(local) output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".peakypanes.yml")); err != nil {
		t.Fatalf(".peakypanes.yml missing: %v", err)
	}
}

func TestParseInitArgs(t *testing.T) {
	opts := parseInitArgs([]string{})
	if opts.layout != "dev-3" {
		t.Fatalf("parseInitArgs() default layout = %q", opts.layout)
	}
	opts = parseInitArgs([]string{"--local", "--layout", "custom", "--force"})
	if !opts.local || !opts.force || opts.layout != "custom" {
		t.Fatalf("parseInitArgs() = %#v", opts)
	}
	opts = parseInitArgs([]string{"--help"})
	if !opts.showHelp {
		t.Fatalf("parseInitArgs(--help) should set showHelp")
	}
}

func TestParseKillArgs(t *testing.T) {
	opts := parseKillArgs([]string{"--help"})
	if !opts.showHelp {
		t.Fatalf("parseKillArgs(--help) should set showHelp")
	}
	opts = parseKillArgs([]string{"myapp"})
	if opts.session != "myapp" {
		t.Fatalf("parseKillArgs(session) = %q", opts.session)
	}
}

func TestParseStartArgs(t *testing.T) {
	opts := parseStartArgs([]string{"--layout", "dev-3", "--session", "sess", "--path", "/tmp", "--detach"})
	if opts.layoutName != "dev-3" || opts.session != "sess" || opts.path != "/tmp" || !opts.detach {
		t.Fatalf("parseStartArgs() = %#v", opts)
	}
	opts = parseStartArgs([]string{"fullstack"})
	if opts.layoutName != "fullstack" {
		t.Fatalf("parseStartArgs(positional) = %#v", opts)
	}
	opts = parseStartArgs([]string{"--help"})
	if !opts.showHelp {
		t.Fatalf("parseStartArgs(--help) should set showHelp")
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	return string(data)
}
