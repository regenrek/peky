package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/tui/app"
)

func TestRunStartUsesCwdWhenPathMissing(t *testing.T) {
	origRunMenu := runMenuFn
	defer func() { runMenuFn = origRunMenu }()

	var got *app.AutoStartSpec
	runMenuFn = func(spec *app.AutoStartSpec) { got = spec }

	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	runStart([]string{"--session", "sess"})
	if got == nil {
		t.Fatalf("expected autoStart spec")
	}
	wantPath, _ := filepath.EvalSymlinks(dir)
	gotPath, _ := filepath.EvalSymlinks(got.Path)
	if gotPath != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, gotPath)
	}
}

func TestRunCloneExistingDir(t *testing.T) {
	origRunMenu := runMenuFn
	defer func() { runMenuFn = origRunMenu }()

	var got *app.AutoStartSpec
	runMenuFn = func(spec *app.AutoStartSpec) { got = spec }

	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "projects", "repo")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	out := captureStdout(func() {
		runClone([]string{"user/repo"})
	})
	if !strings.Contains(out, "Directory exists") {
		t.Fatalf("expected existing directory output, got %q", out)
	}
	if got == nil || got.Path != target {
		t.Fatalf("expected autoStart path %q, got %#v", target, got)
	}
}

func TestRunCloneNewRepoWithFakeGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git helper uses sh")
	}
	origRunMenu := runMenuFn
	defer func() { runMenuFn = origRunMenu }()

	var got *app.AutoStartSpec
	runMenuFn = func(spec *app.AutoStartSpec) { got = spec }

	home := t.TempDir()
	t.Setenv("HOME", home)

	bin := t.TempDir()
	gitPath := filepath.Join(bin, "git")
	script := "#!/bin/sh\nmkdir -p \"$3\"\nexit 0\n"
	if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	runClone([]string{"user/repo"})
	target := filepath.Join(home, "projects", "repo")
	if got == nil || got.Path != target {
		t.Fatalf("expected autoStart path %q, got %#v", target, got)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target dir, err=%v", err)
	}
}

func TestListLayoutsAndExport(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	out := captureStdout(func() {
		listLayouts()
	})
	if !strings.Contains(out, "Available Layouts") {
		t.Fatalf("listLayouts output = %q", out)
	}

	export := captureStdout(func() {
		exportLayout("dev-3")
	})
	if !strings.Contains(export, "Peaky Panes Layout") {
		t.Fatalf("exportLayout output = %q", export)
	}
}

func TestRunStartHelp(t *testing.T) {
	out := captureStdout(func() {
		runStart([]string{"--help"})
	})
	if !strings.Contains(out, "Start a session") {
		t.Fatalf("expected start help output, got %q", out)
	}
}

func TestRunLayoutsExport(t *testing.T) {
	out := captureStdout(func() {
		runLayouts([]string{"export", "dev-3"})
	})
	if !strings.Contains(out, "Peaky Panes Layout") {
		t.Fatalf("expected export output, got %q", out)
	}
}

func TestRunDaemonHelp(t *testing.T) {
	out := captureStdout(func() {
		runDaemon([]string{"--help"})
	})
	if !strings.Contains(out, "session daemon") {
		t.Fatalf("runDaemon help output = %q", out)
	}
}

func TestInitGlobalCreatesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	out := captureStdout(func() {
		initGlobal("dev-3", true)
	})
	if !strings.Contains(out, "Initialized Peaky Panes") {
		t.Fatalf("initGlobal output = %q", out)
	}

	out = captureStdout(func() {
		initGlobal("dev-3", false)
	})
	if !strings.Contains(out, "Config already exists") {
		t.Fatalf("expected existing config message, got %q", out)
	}
}
