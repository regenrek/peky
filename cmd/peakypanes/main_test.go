package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/app"
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

func TestResolveGridCommandsAndTitles(t *testing.T) {
	cfg := &layout.LayoutConfig{
		Command:  "echo",
		Commands: []string{"a", "b"},
		Titles:   []string{"one"},
	}
	commands := layout.ResolveGridCommands(cfg, 3)
	if !reflect.DeepEqual(commands, []string{"a", "b", "echo"}) {
		t.Fatalf("resolveGridCommands() = %#v", commands)
	}
	titles := layout.ResolveGridTitles(cfg, 3)
	if !reflect.DeepEqual(titles, []string{"one", "", ""}) {
		t.Fatalf("resolveGridTitles() = %#v", titles)
	}
}

func TestSanitizeSessionName(t *testing.T) {
	if got := layout.SanitizeSessionName(" My Project "); got != "my-project" {
		t.Fatalf("SanitizeSessionName() = %q", got)
	}
	if got := layout.SanitizeSessionName(""); got != "session" {
		t.Fatalf("SanitizeSessionName(empty) = %q", got)
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
	defer func() {
		runMenuFn = origRunMenu
	}()

	called := false
	runMenuFn = func(_ *app.AutoStartSpec) { called = true }

	out := captureStdout(func() {
		runDashboardCommand([]string{"--help"})
	})
	if called {
		t.Fatalf("runDashboardCommand(--help) should not invoke runMenu")
	}
	if !strings.Contains(out, "Open the Peaky Panes dashboard UI") {
		t.Fatalf("runDashboardCommand(--help) output = %q", out)
	}
}

func TestRunDashboardCommandRunsMenu(t *testing.T) {
	origRunMenu := runMenuFn
	defer func() {
		runMenuFn = origRunMenu
	}()

	called := false
	runMenuFn = func(spec *app.AutoStartSpec) {
		called = true
		if spec != nil {
			t.Fatalf("runDashboardCommand should not pass autoStart")
		}
	}

	runDashboardCommand(nil)
	if !called {
		t.Fatalf("runDashboardCommand should invoke runMenu")
	}
}

func TestRunStartAutoStartSpec(t *testing.T) {
	origRunMenu := runMenuFn
	defer func() {
		runMenuFn = origRunMenu
	}()

	var got *app.AutoStartSpec
	runMenuFn = func(spec *app.AutoStartSpec) { got = spec }

	runStart([]string{"--layout", "dev-3", "--session", "sess", "--path", "/tmp"})
	if got == nil {
		t.Fatalf("runStart should pass autoStart spec")
	}
	if got.Layout != "dev-3" || got.Session != "sess" || got.Path != "/tmp" || !got.Focus {
		t.Fatalf("autoStart = %#v", got)
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

func TestParseStartArgs(t *testing.T) {
	opts := parseStartArgs([]string{"--layout", "dev-3", "--session", "sess", "--path", "/tmp"})
	if opts.layoutName != "dev-3" || opts.session != "sess" || opts.path != "/tmp" {
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
