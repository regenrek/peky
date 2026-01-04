package logging

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestLogEveryPrunesOldKeys(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	origLogger := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(origLogger) })

	logEveryMu.Lock()
	orig := logEveryLast
	origMax := maxLogEveryKeys
	logEveryLast = map[string]time.Time{}
	maxLogEveryKeys = 3
	logEveryLast["a"] = time.Unix(1, 0)
	logEveryLast["b"] = time.Unix(2, 0)
	logEveryLast["c"] = time.Unix(3, 0)
	logEveryLast["d"] = time.Unix(4, 0)
	logEveryMu.Unlock()
	t.Cleanup(func() {
		logEveryMu.Lock()
		logEveryLast = orig
		maxLogEveryKeys = origMax
		logEveryMu.Unlock()
	})

	LogEvery(context.Background(), "e", time.Millisecond, 0, "msg")

	logEveryMu.Lock()
	defer logEveryMu.Unlock()
	if len(logEveryLast) > maxLogEveryKeys {
		t.Fatalf("expected map size <= %d, got %d", maxLogEveryKeys, len(logEveryLast))
	}
	if _, ok := logEveryLast["a"]; ok {
		t.Fatalf("expected oldest key pruned")
	}
}

func TestLogEverySkipsWhenDisabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
	orig := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(orig) })

	logEveryMu.Lock()
	logEveryLast = map[string]time.Time{}
	logEveryMu.Unlock()

	LogEvery(context.Background(), "key", time.Minute, slog.LevelInfo, "msg")

	logEveryMu.Lock()
	defer logEveryMu.Unlock()
	if len(logEveryLast) != 0 {
		t.Fatalf("expected no entries when level disabled, got %d", len(logEveryLast))
	}
}
