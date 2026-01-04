package sessiond

import (
	"context"
	"runtime"
	"time"
)

const (
	scopeSendConcurrencyMin = 4
	scopeSendConcurrencyMax = 32
)

var scopeSendTimeout = defaultOpTimeout

type scopeSendJob struct {
	Index  int
	PaneID string
}

type scopeSendResult struct {
	Index   int
	PaneID  string
	Status  string
	Message string
}

func scopeSendConcurrency(targets int) int {
	if targets <= 0 {
		return 0
	}
	base := runtime.GOMAXPROCS(0) * 2
	if base < scopeSendConcurrencyMin {
		base = scopeSendConcurrencyMin
	}
	if base > scopeSendConcurrencyMax {
		base = scopeSendConcurrencyMax
	}
	if targets < base {
		return targets
	}
	return base
}

func sendInputToTargetWithTimeout(manager sessionManager, paneID string, input []byte, timeout time.Duration) (string, string) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	status, message := sendInputToTarget(manager, ctx, paneID, input)
	if timeout > 0 && ctx.Err() != nil && status == "failed" {
		return "timeout", "send timed out"
	}
	return status, message
}
