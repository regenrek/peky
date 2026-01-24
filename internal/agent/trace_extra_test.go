package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTruncateField(t *testing.T) {
	if got := truncateField(" "); got != "" {
		t.Fatalf("expected empty string")
	}
	short := strings.Repeat("a", 10)
	if got := truncateField(short); got != short {
		t.Fatalf("short=%q", got)
	}
	long := strings.Repeat("b", traceFieldLimit+10)
	out := truncateField(long)
	if len(out) <= traceFieldLimit {
		t.Fatalf("expected truncated output")
	}
	if !strings.HasPrefix(out, strings.Repeat("b", traceFieldLimit)) {
		t.Fatalf("unexpected prefix")
	}
}

func TestNewTraceLogger(t *testing.T) {
	logger, closeFn, err := newTraceLogger("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger != nil {
		t.Fatalf("expected nil logger for empty path")
	}
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn error: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.log")
	logger, closeFn, err = newTraceLogger(path)
	if err != nil || logger == nil {
		t.Fatalf("newTraceLogger error: %v", err)
	}
	logger.log(traceEvent{Time: nowRFC3339(), Event: "test"})
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected log file, err=%v", err)
	}
}

func TestNowRFC3339(t *testing.T) {
	if _, err := time.Parse(time.RFC3339Nano, nowRFC3339()); err != nil {
		t.Fatalf("parse error: %v", err)
	}
}
