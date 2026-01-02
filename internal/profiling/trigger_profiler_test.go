//go:build profiler
// +build profiler

package profiling

import (
	"context"
	"testing"
	"time"
)

func TestWaitTimesOutWithoutTrigger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if Wait(ctx, 20*time.Millisecond) {
		t.Fatalf("expected Wait to return false without trigger")
	}
}

func TestTriggerUnblocksWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		time.Sleep(10 * time.Millisecond)
		Trigger("test")
	}()
	if !Wait(ctx, 500*time.Millisecond) {
		t.Fatalf("expected Wait to return true after trigger")
	}
}
