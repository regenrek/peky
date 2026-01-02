//go:build !profiler
// +build !profiler

package profiling

import (
	"context"
	"time"
)

// Trigger is a no-op in non-profiler builds.
func Trigger(reason string) {}

// Wait returns immediately in non-profiler builds.
func Wait(ctx context.Context, timeout time.Duration) bool { return true }
