package terminal

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"
)

type fakePty struct {
	writes [][]byte
}

func (f *fakePty) Read(p []byte) (int, error) { return 0, io.EOF }
func (f *fakePty) Write(p []byte) (int, error) {
	f.writes = append(f.writes, append([]byte{}, p...))
	return len(p), nil
}
func (f *fakePty) Close() error { return nil }
func (f *fakePty) Fd() uintptr  { return 0 }
func (f *fakePty) Resize(width, height int) error {
	return nil
}
func (f *fakePty) Size() (width, height int, err error) { return 0, 0, nil }
func (f *fakePty) Name() string                         { return "fake-pty" }
func (f *fakePty) Start(cmd *exec.Cmd) error            { return nil }

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

func TestSendInputClearsMouseSelection(t *testing.T) {
	pty := &fakePty{}
	w := &Window{pty: pty, writeCh: make(chan writeRequest, 1)}
	w.CopyMode = &CopyMode{Active: true, Selecting: true}
	w.mouseSelection = true
	w.ScrollbackMode = true
	w.ScrollbackOffset = 3

	if err := w.SendInput([]byte("h")); err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	if w.CopyMode != nil {
		t.Fatalf("expected copy mode cleared after input")
	}
	if w.ScrollbackOffset != 0 || w.ScrollbackMode {
		t.Fatalf("expected scrollback cleared after input, offset=%d mode=%v", w.ScrollbackOffset, w.ScrollbackMode)
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
