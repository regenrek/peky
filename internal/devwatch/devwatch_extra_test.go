package devwatch

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type logSink struct {
	calls []string
}

func (l *logSink) Printf(format string, args ...any) {
	l.calls = append(l.calls, format)
}

func TestPrepareConfigDefaults(t *testing.T) {
	cfg, err := prepareConfig(Config{Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatalf("prepareConfig error: %v", err)
	}
	if len(cfg.BuildCmd) == 0 || len(cfg.RunCmd) == 0 || len(cfg.WatchPaths) == 0 {
		t.Fatalf("defaults missing: %#v", cfg)
	}
	if cfg.Interval <= 0 || cfg.Debounce < 0 || cfg.ShutdownTimeout <= 0 {
		t.Fatalf("timers=%#v", cfg)
	}
}

func TestValidateConfigErrors(t *testing.T) {
	if err := validateConfig(Config{}); err == nil {
		t.Fatalf("expected build command error")
	}
	if err := validateConfig(Config{BuildCmd: []string{"go"}, RunCmd: []string{}, WatchPaths: []string{"."}}); err == nil {
		t.Fatalf("expected run command error")
	}
	if err := validateConfig(Config{BuildCmd: []string{"go"}, RunCmd: []string{"go"}, WatchPaths: nil}); err == nil {
		t.Fatalf("expected watch paths error")
	}
}

func TestCollectFilesIncludesExplicit(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	goFile := filepath.Join(dir, "app.go")
	if err := os.WriteFile(goFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	logger := &logSink{}
	cfg := Config{
		WatchPaths: []string{file, dir},
		Extensions: []string{".go"},
		Logger:     log.New(io.Discard, "", 0),
		Interval:   time.Millisecond,
		Debounce:   0,
	}
	watcher := newPollWatcher(cfg)
	watcher.logger = logger
	files, err := watcher.collectFiles()
	if err != nil {
		t.Fatalf("collectFiles error: %v", err)
	}
	foundFile := false
	foundGo := false
	for _, path := range files {
		if path == file {
			foundFile = true
		}
		if path == goFile {
			foundGo = true
		}
	}
	if !foundFile || !foundGo {
		t.Fatalf("files=%v", files)
	}
}

func TestTimerChanAndSignal(t *testing.T) {
	if timerChan(nil) != nil {
		t.Fatalf("expected nil timer channel")
	}
	ch := make(chan struct{}, 1)
	w := &pollWatcher{}
	w.signalChange(ch)
	w.signalChange(ch)
	select {
	case <-ch:
	default:
		t.Fatalf("expected change signal")
	}
}

func TestDrainChangesAndWait(t *testing.T) {
	ch := make(chan struct{}, 2)
	ch <- struct{}{}
	ch <- struct{}{}
	drainChanges(ch)
	select {
	case <-ch:
		t.Fatalf("expected drained channel")
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitForChange(ctx, ch) {
		t.Fatalf("expected false on canceled context")
	}
	ch <- struct{}{}
	if !waitForChange(context.Background(), ch) {
		t.Fatalf("expected change")
	}
}
