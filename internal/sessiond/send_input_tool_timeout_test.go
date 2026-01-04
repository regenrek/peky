package sessiond

import (
	"context"
	"testing"
	"time"
)

type blockingManager struct {
	fakeManager
	delay time.Duration
}

func (m *blockingManager) SendInput(ctx context.Context, paneID string, input []byte) error {
	if ctx == nil {
		time.Sleep(m.delay)
		return nil
	}
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestSendToolInputCombine(t *testing.T) {
	mgr := &fakeManager{}
	plan := toolSendPlan{Payload: []byte("hi"), Submit: []byte("\r"), Combine: true}
	status, msg := sendToolInput(mgr, "pane-1", plan)
	if status != "ok" || msg != "" {
		t.Fatalf("sendToolInput status=%q msg=%q", status, msg)
	}
	if len(mgr.inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(mgr.inputs))
	}
	if got := string(mgr.inputs[0]); got != "hi\r" {
		t.Fatalf("combined = %q", got)
	}
}

func TestSendToolInputWithTimeout(t *testing.T) {
	orig := scopeSendTimeout
	scopeSendTimeout = 10 * time.Millisecond
	defer func() { scopeSendTimeout = orig }()
	mgr := &blockingManager{delay: 50 * time.Millisecond}
	plan := toolSendPlan{Payload: []byte("hi")}
	status, _ := sendToolInputWithTimeout(mgr, "pane-1", plan)
	if status != "timeout" {
		t.Fatalf("expected timeout, got %q", status)
	}
}
