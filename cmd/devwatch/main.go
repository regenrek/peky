package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/regenrek/peakypanes/internal/identity"
)

const (
	defaultWatchDirs = "cmd,internal"
	defaultDebounce  = 250 * time.Millisecond
	stopTimeout      = 2 * time.Second
)

type config struct {
	watchDirs []string
	debounce  time.Duration
	args      []string
}

type runner struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "devwatch:", err)
		os.Exit(1)
	}
	if err := ensureRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "devwatch:", err)
		os.Exit(1)
	}

	bin, err := resolvePekyBin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "devwatch:", err)
		os.Exit(1)
	}

	r := &runner{}
	if err := runCycle(r, bin, cfg.args); err != nil {
		fmt.Fprintln(os.Stderr, "devwatch:", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintln(os.Stderr, "devwatch:", err)
		os.Exit(1)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "devwatch:", err)
		}
	}()

	for _, dir := range cfg.watchDirs {
		if err := addWatchTree(watcher, dir); err != nil {
			fmt.Fprintln(os.Stderr, "devwatch:", err)
			os.Exit(1)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	reloadCh := make(chan struct{}, 1)
	go debounceEvents(watcher, cfg.debounce, reloadCh)

	for {
		select {
		case <-reloadCh:
			if err := runCycle(r, bin, cfg.args); err != nil {
				fmt.Fprintln(os.Stderr, "devwatch:", err)
			}
		case <-sigCh:
			_ = r.Stop(stopTimeout)
			return
		}
	}
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("devwatch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	watch := fs.String("watch", defaultWatchDirs, "comma-separated watch roots")
	debounce := fs.Duration("debounce", defaultDebounce, "debounce duration")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	watchDirs := parseWatchDirs(*watch)
	if len(watchDirs) == 0 {
		return config{}, errors.New("no watch directories configured")
	}
	for _, dir := range watchDirs {
		info, err := os.Stat(dir)
		if err != nil {
			return config{}, fmt.Errorf("watch dir %q: %w", dir, err)
		}
		if !info.IsDir() {
			return config{}, fmt.Errorf("watch dir %q: not a directory", dir)
		}
	}

	cmdArgs := fs.Args()
	if len(cmdArgs) == 0 {
		cmdArgs = []string{"start", "-y"}
	}

	return config{
		watchDirs: watchDirs,
		debounce:  *debounce,
		args:      cmdArgs,
	}, nil
}

func parseWatchDirs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		dir := strings.TrimSpace(part)
		if dir == "" {
			continue
		}
		out = append(out, dir)
	}
	return out
}

func ensureRepoRoot() error {
	if _, err := os.Stat("go.mod"); err == nil {
		return nil
	}
	return errors.New("run from repo root (go.mod not found)")
}

func resolvePekyBin() (string, error) {
	gobin := strings.TrimSpace(os.Getenv("GOBIN"))
	if gobin == "" {
		out, err := exec.Command("go", "env", "GOPATH").Output()
		if err != nil {
			return "", fmt.Errorf("go env GOPATH: %w", err)
		}
		gobin = filepath.Join(strings.TrimSpace(string(out)), "bin")
	}
	bin := filepath.Join(gobin, identity.CLIName)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	return bin, nil
}

func runCycle(r *runner, bin string, args []string) error {
	if err := r.Stop(stopTimeout); err != nil {
		return err
	}
	if err := goInstall(); err != nil {
		return err
	}
	if err := restartDaemon(bin); err != nil {
		return err
	}
	return r.Start(bin, args)
}

func goInstall() error {
	cmd := exec.Command("go", "install", "./cmd/peky")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func restartDaemon(bin string) error {
	cmd := exec.Command(bin, "daemon", "restart", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *runner) Start(bin string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	r.cmd = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

func (r *runner) Stop(timeout time.Duration) error {
	r.mu.Lock()
	cmd := r.cmd
	r.cmd = nil
	r.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
		return nil
	}
	return waitWithTimeout(cmd, timeout)
}

func waitWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return nil
	}
}

func addWatchTree(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if shouldIgnoreDir(d.Name()) {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

func shouldIgnoreDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "bin", "dist", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func debounceEvents(w *fsnotify.Watcher, debounce time.Duration, out chan<- struct{}) {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	pending := false

	reset := func() { resetTimer(timer, debounce, pending) }

	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if handleWatchEvent(w, ev) {
				pending = true
				reset()
			}
		case <-timer.C:
			pending = flushPending(out, pending)
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
		}
	}
}

func resetTimer(timer *time.Timer, debounce time.Duration, pending bool) {
	if !timer.Stop() && pending {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(debounce)
}

func handleWatchEvent(w *fsnotify.Watcher, ev fsnotify.Event) bool {
	if ev.Op&fsnotify.Create != 0 {
		if isDir(ev.Name) {
			_ = addWatchTree(w, ev.Name)
		}
	}
	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	if shouldIgnoreFile(ev.Name) {
		return false
	}
	return true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func flushPending(out chan<- struct{}, pending bool) bool {
	if !pending {
		return false
	}
	select {
	case out <- struct{}{}:
	default:
	}
	return false
}

func shouldIgnoreFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".#") || strings.HasSuffix(base, "~") {
		return true
	}
	if strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, ".tmp") {
		return true
	}
	return false
}
