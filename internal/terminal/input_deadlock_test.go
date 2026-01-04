package terminal

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/vt"
)

type stubPty struct{}

func (p *stubPty) Read([]byte) (int, error)    { return 0, io.EOF }
func (p *stubPty) Write(b []byte) (int, error) { return len(b), nil }
func (p *stubPty) Close() error                { return nil }
func (p *stubPty) Fd() uintptr                 { return 0 }
func (p *stubPty) Resize(int, int) error       { return nil }
func (p *stubPty) Size() (int, int, error)     { return 0, 0, nil }
func (p *stubPty) Name() string                { return "stub" }
func (p *stubPty) Start(*exec.Cmd) error       { return nil }

type lockingEmu struct {
	readCh   chan []byte
	lockMu   *sync.Mutex
	lockNext atomic.Bool
	lockedCh chan struct{}
}

func (e *lockingEmu) Read(p []byte) (int, error) {
	data, ok := <-e.readCh
	if !ok {
		return 0, io.EOF
	}
	if e.lockMu != nil && e.lockNext.Swap(false) {
		e.lockMu.Lock()
		if e.lockedCh != nil {
			close(e.lockedCh)
		}
	}
	n := copy(p, data)
	return n, nil
}

func (e *lockingEmu) Write(p []byte) (int, error) { return len(p), nil }
func (e *lockingEmu) Close() error {
	if e.readCh != nil {
		close(e.readCh)
	}
	return nil
}
func (e *lockingEmu) Resize(int, int)              {}
func (e *lockingEmu) Render() string               { return "" }
func (e *lockingEmu) CellAt(int, int) *uv.Cell     { return nil }
func (e *lockingEmu) CursorPosition() uv.Position  { return uv.Position{} }
func (e *lockingEmu) SendMouse(uv.MouseEvent)      {}
func (e *lockingEmu) SetCallbacks(vt.Callbacks)    {}
func (e *lockingEmu) Height() int                  { return 0 }
func (e *lockingEmu) Width() int                   { return 0 }
func (e *lockingEmu) IsAltScreen() bool            { return false }
func (e *lockingEmu) Cwd() string                  { return "" }
func (e *lockingEmu) ScrollbackLen() int           { return 0 }
func (e *lockingEmu) ScrollbackLine(int) []uv.Cell { return nil }
func (e *lockingEmu) ClearScrollback()             {}
func (e *lockingEmu) SetScrollbackMaxLines(int)    {}

func TestStartVtToPtyDoesNotDependOnTermMu(t *testing.T) {
	emu := &lockingEmu{
		readCh:   make(chan []byte, 2),
		lockedCh: make(chan struct{}),
	}
	w := &Window{
		term:    emu,
		pty:     &stubPty{},
		writeCh: make(chan writeRequest, 2),
		updates: make(chan struct{}, 1),
	}
	emu.lockMu = &w.termMu

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.startVtToPty(ctx)

	emu.lockNext.Store(true)
	emu.readCh <- []byte("a")

	select {
	case <-w.writeCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("vt->pty did not forward initial data")
	}

	select {
	case <-emu.lockedCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected termMu to be locked after read")
	}
	defer w.termMu.Unlock()

	emu.readCh <- []byte("b")
	select {
	case <-w.writeCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("vt->pty blocked when termMu is held")
	}
}
