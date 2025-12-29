package devwatch

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanFingerprintDetectsChange(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Config{
		WatchPaths: []string{root},
		Extensions: []string{".go"},
		Interval:   10 * time.Millisecond,
		Debounce:   0,
		Logger:     log.New(os.Stdout, "", 0),
	}
	watcher := newPollWatcher(cfg)
	first, err := watcher.scanFingerprint()
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if err := os.WriteFile(file, []byte("package main\n// changed\n"), 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if err := os.Chtimes(file, time.Now().Add(1*time.Second), time.Now().Add(1*time.Second)); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	second, err := watcher.scanFingerprint()
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if first == second {
		t.Fatalf("expected fingerprint to change")
	}
}

func TestIncludeFileExplicit(t *testing.T) {
	cfg := Config{
		Extensions: []string{".go"},
		Logger:     log.New(io.Discard, "", 0),
	}
	watcher := newPollWatcher(cfg)
	if !watcher.includeFile("/tmp/README.md", true) {
		t.Fatalf("explicit file should be included")
	}
	if watcher.includeFile("/tmp/README.md", false) {
		t.Fatalf("non-matching extension should be excluded")
	}
}
