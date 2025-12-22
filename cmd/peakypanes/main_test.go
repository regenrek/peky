package main

import (
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
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	if insideTmux() {
		t.Fatalf("insideTmux() should be false")
	}
	t.Setenv("TMUX", "/tmp/tmux")
	if !insideTmux() {
		t.Fatalf("insideTmux() should be true")
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
