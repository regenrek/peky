//go:build !profiler
// +build !profiler

package profiling

import (
	"context"
	"testing"
	"time"
)

func TestTriggerNoop(t *testing.T) {
	Trigger("stub")
	if !Wait(context.Background(), 10*time.Millisecond) {
		t.Fatalf("expected Wait to return true in stub build")
	}
}
