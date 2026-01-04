package logging

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"
)

var (
	logEveryMu      sync.Mutex
	logEveryLast    = map[string]time.Time{}
	maxLogEveryKeys = 1024
)

// LogEvery emits a log entry at most once per interval for a key.
func LogEvery(ctx context.Context, key string, interval time.Duration, level slog.Level, msg string, attrs ...slog.Attr) {
	if !slog.Default().Enabled(ctx, level) {
		return
	}
	if key == "" {
		slog.LogAttrs(ctx, level, msg, attrs...)
		return
	}
	if interval <= 0 {
		slog.LogAttrs(ctx, level, msg, attrs...)
		return
	}
	now := time.Now()
	logEveryMu.Lock()
	last := logEveryLast[key]
	if !last.IsZero() && now.Sub(last) < interval {
		logEveryMu.Unlock()
		return
	}
	logEveryLast[key] = now
	if len(logEveryLast) > maxLogEveryKeys {
		pruneLogEvery()
	}
	logEveryMu.Unlock()
	slog.LogAttrs(ctx, level, msg, attrs...)
}

func pruneLogEvery() {
	if len(logEveryLast) <= maxLogEveryKeys {
		return
	}
	type entry struct {
		key string
		t   time.Time
	}
	entries := make([]entry, 0, len(logEveryLast))
	for key, t := range logEveryLast {
		entries = append(entries, entry{key: key, t: t})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].t.Before(entries[j].t)
	})
	remove := len(entries) - maxLogEveryKeys
	for i := 0; i < remove; i++ {
		delete(logEveryLast, entries[i].key)
	}
}
