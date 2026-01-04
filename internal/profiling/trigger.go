//go:build profiler
// +build profiler

package profiling

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	triggerOnce sync.Once
	triggered   atomic.Bool
	triggerCh   = make(chan struct{})
)

// Trigger signals that profiling should start.
func Trigger(reason string) {
	if triggered.Load() {
		return
	}
	if !triggered.CompareAndSwap(false, true) {
		return
	}
	triggerOnce.Do(func() {
		close(triggerCh)
		if strings.TrimSpace(reason) == "" {
			reason = "trigger"
		}
		slog.Info("sessiond: profiler trigger", slog.String("reason", reason))
	})
}

// Wait blocks until Trigger is called, the timeout elapses, or ctx is done.
func Wait(ctx context.Context, timeout time.Duration) bool {
	if triggered.Load() {
		return true
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		select {
		case <-triggerCh:
			return true
		case <-ctx.Done():
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-triggerCh:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}
