package sessiond

import (
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
	if timeout <= 0 {
		return sendInputToTarget(manager, paneID, input)
	}
	done := make(chan struct{})
	var status string
	var message string
	go func() {
		status, message = sendInputToTarget(manager, paneID, input)
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return status, message
	case <-timer.C:
		return "timeout", "send timed out"
	}
}
