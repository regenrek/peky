package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/regenrek/peakypanes/internal/identity"
)

func TestParseWatchDirs(t *testing.T) {
	if got := parseWatchDirs(""); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	if got := parseWatchDirs(" , , "); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	got := parseWatchDirs("cmd, internal ,vendor")
	if len(got) != 3 || got[0] != "cmd" || got[1] != "internal" || got[2] != "vendor" {
		t.Fatalf("unexpected parseWatchDirs result: %v", got)
	}
}

func TestParseConfig_DefaultArgs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "b"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg, err := parseConfig([]string{"--watch", "a,b"})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if len(cfg.watchDirs) != 2 {
		t.Fatalf("watchDirs=%v, want 2 dirs", cfg.watchDirs)
	}
	if cfg.args[0] != "start" {
		t.Fatalf("args=%v, want default start args", cfg.args)
	}
}

func TestParseConfig_RejectsMissingDir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if _, err := parseConfig([]string{"--watch", "a,missing"}); err == nil {
		t.Fatalf("expected error for missing watch dir")
	}
}

func TestParseConfig_RejectsEmptyWatch(t *testing.T) {
	if _, err := parseConfig([]string{"--watch", ""}); err == nil {
		t.Fatalf("expected error for empty watch list")
	}
}

func TestParseConfig_RejectsFileWatch(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if _, err := parseConfig([]string{"--watch", "file.txt"}); err == nil {
		t.Fatalf("expected error for watch dir that is a file")
	}
}

func TestEnsureRepoRoot(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := ensureRepoRoot(); err == nil {
		t.Fatalf("expected error when go.mod missing")
	}

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := ensureRepoRoot(); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestResolvePekyBin_UsesGOBIN(t *testing.T) {
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)

	got, err := resolvePekyBin()
	if err != nil {
		t.Fatalf("resolvePekyBin: %v", err)
	}
	want := filepath.Join(gobin, identity.CLIName)
	if got != want {
		t.Fatalf("bin=%q, want %q", got, want)
	}
}

func TestResolvePekyBin_FallsBackToGoEnv(t *testing.T) {
	t.Setenv("GOBIN", "")
	got, err := resolvePekyBin()
	if err != nil {
		t.Fatalf("resolvePekyBin: %v", err)
	}
	if filepath.Base(got) != identity.CLIName {
		t.Fatalf("bin=%q, want base %q", got, identity.CLIName)
	}
}

func TestIgnoreHelpers(t *testing.T) {
	if !shouldIgnoreDir(".git") || !shouldIgnoreDir("node_modules") || shouldIgnoreDir("src") {
		t.Fatalf("unexpected shouldIgnoreDir results")
	}

	cases := []struct {
		path string
		want bool
	}{
		{path: "/tmp/.#foo", want: true},
		{path: "/tmp/foo~", want: true},
		{path: "/tmp/foo.swp", want: true},
		{path: "/tmp/foo.tmp", want: true},
		{path: "/tmp/foo.go", want: false},
	}
	for _, tc := range cases {
		if got := shouldIgnoreFile(tc.path); got != tc.want {
			t.Fatalf("shouldIgnoreFile(%q)=%v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestAddWatchTree(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("watcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := addWatchTree(w, root); err != nil {
		t.Fatalf("addWatchTree: %v", err)
	}
}

func TestHandleWatchEvent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "newdir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("watcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if got := handleWatchEvent(w, fsnotify.Event{Name: dir, Op: fsnotify.Create}); !got {
		t.Fatalf("expected create dir to trigger reload")
	}
	if got := handleWatchEvent(w, fsnotify.Event{Name: filepath.Join(root, "noop"), Op: fsnotify.Chmod}); got {
		t.Fatalf("expected chmod to be ignored")
	}
	if got := handleWatchEvent(w, fsnotify.Event{Name: filepath.Join(root, "file.tmp"), Op: fsnotify.Write}); got {
		t.Fatalf("expected ignored file to not trigger reload")
	}
}

func TestFlushPending(t *testing.T) {
	ch := make(chan struct{}, 1)
	if flushPending(ch, false) {
		t.Fatalf("expected flushPending false when pending=false")
	}

	if flushPending(ch, true) {
		t.Fatalf("expected flushPending false (always clears)")
	}
	if len(ch) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(ch))
	}

	if flushPending(ch, true) {
		t.Fatalf("expected flushPending false (always clears)")
	}
	if len(ch) != 1 {
		t.Fatalf("expected still 1 signal (non-blocking), got %d", len(ch))
	}
}

func TestResetTimer(t *testing.T) {
	timer := time.NewTimer(0)
	<-timer.C
	resetTimer(timer, time.Hour, true)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func TestWaitWithTimeout(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := waitWithTimeout(cmd, time.Second); err != nil {
		t.Fatalf("waitWithTimeout: %v", err)
	}
}

func TestIsDir(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if isDir(file) {
		t.Fatalf("expected file to not be dir")
	}
	if !isDir(root) {
		t.Fatalf("expected root to be dir")
	}
}
