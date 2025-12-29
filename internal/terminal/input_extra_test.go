package terminal

import (
	"context"
	"errors"
	"testing"
)

func TestSendInputErrorPaths(t *testing.T) {
	var nilWindow *Window
	if err := nilWindow.SendInput([]byte("hi")); err == nil {
		t.Fatalf("expected error for nil window")
	}

	w := &Window{}
	if err := w.SendInput(nil); err != nil {
		t.Fatalf("expected nil error for empty input, got %v", err)
	}
	w.closed.Store(true)
	if err := w.SendInput([]byte("hi")); err == nil || !errors.Is(err, ErrPaneClosed) {
		t.Fatalf("expected pane closed error for closed window, got %v", err)
	}
	w.closed.Store(false)
	if err := w.SendInput([]byte("hi")); err == nil || !errors.Is(err, ErrPaneClosed) {
		t.Fatalf("expected pane closed error for missing pty, got %v", err)
	}
}

func TestCtxDone(t *testing.T) {
	if ctxDone(context.Background()) {
		t.Fatalf("expected background context to be not done")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !ctxDone(ctx) {
		t.Fatalf("expected context done")
	}
}
