package terminal

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/regenrek/peakypanes/internal/logging"
)

const (
	ptyWriteQueueSize = 128
)

var errWriteQueueFull = errors.New("terminal: write queue full")

type writeRequest struct {
	data      []byte
	markDirty bool
}

// SendInput writes bytes to the underlying PTY.
// This is what your Bubble Tea model should call for focused pane input.
func (w *Window) SendInput(ctx context.Context, input []byte) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if len(input) == 0 {
		return nil
	}
	if w.CopySelectionFromMouseActive() {
		w.ExitCopyMode()
		if w.ScrollbackModeActive() || w.GetScrollbackOffset() > 0 {
			w.ExitScrollback()
		}
	}
	if w.closed.Load() {
		return &PaneClosedError{Reason: PaneClosedWindowClosed}
	}
	if w.exited.Load() {
		return &PaneClosedError{Reason: PaneClosedProcessExited}
	}
	if w.inputClosed.Load() {
		return &PaneClosedError{Reason: w.inputClosedReasonValue()}
	}
	w.ptyMu.Lock()
	pty := w.pty
	w.ptyMu.Unlock()
	if pty == nil {
		return &PaneClosedError{Reason: PaneClosedPTYClosed}
	}
	return w.enqueueWrite(ctx, input, true)
}

func isPTYClosedWriteError(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, syscall.EIO):
		return true
	case errors.Is(err, syscall.EPIPE):
		return true
	case errors.Is(err, syscall.EBADF):
		return true
	case errors.Is(err, os.ErrClosed):
		return true
	case errors.Is(err, io.ErrClosedPipe):
		return true
	default:
		return false
	}
}

func (w *Window) startIO(ctx context.Context) {
	w.ioStartedAt.CompareAndSwap(0, time.Now().UnixNano())
	w.startPtyWriter(ctx)
	w.startPtyToVt(ctx)
	w.startVtToPty(ctx)
}

func (w *Window) startPtyToVt(ctx context.Context) {
	// PTY -> VT (screen updates)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		buf := make([]byte, 32*1024)
		for {
			pty := w.currentPTY()
			if pty == nil {
				return
			}

			n, err := pty.Read(buf)
			if n > 0 {
				w.handleTerminalWrite(buf[:n])
			}
			if err != nil {
				// Best-effort: treat read errors as exit.
				return
			}
			if ctxDone(ctx) {
				return
			}
		}
	}()
}

func (w *Window) startVtToPty(ctx context.Context) {
	// VT -> PTY (terminal query responses like DSR/DA)
	// This is critical for apps like vim/htop/ncurses that query terminal state.
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		term := w.currentTerminal()
		if term == nil {
			return
		}

		buf := make([]byte, 4096)
		for {
			if w.currentPTY() == nil {
				return
			}

			n, err := term.Read(buf)
			if n > 0 {
				_ = w.enqueueWriteBestEffort(buf[:n], false)
			}
			if err != nil {
				// Best-effort: treat read errors as exit.
				return
			}
			if ctxDone(ctx) {
				return
			}
		}
	}()
}

func (w *Window) currentPTY() io.ReadWriter {
	w.ptyMu.Lock()
	defer w.ptyMu.Unlock()
	return w.pty
}

func (w *Window) currentTerminal() vtEmulator {
	w.termMu.Lock()
	defer w.termMu.Unlock()
	return w.term
}

func (w *Window) handleTerminalWrite(data []byte) {
	// Track scrollback growth so scrollback view stays stable.
	oldSB := 0
	newSB := 0
	if len(data) > 0 {
		w.bytesOut.Add(uint64(len(data)))
		if w.outputFn != nil {
			w.outputFn(data)
		}
	}

	perf := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	var start time.Time
	if perf {
		start = time.Now()
	}

	w.termMu.Lock()
	lockWait := time.Duration(0)
	if perf {
		lockWait = time.Since(start)
	}
	if w.term != nil {
		oldSB = w.term.ScrollbackLen()
		if perf {
			writeStart := time.Now()
			_, _ = w.term.Write(data)
			writeDur := time.Since(writeStart)
			if writeDur > perfSlowWrite {
				logging.LogEvery(
					context.Background(),
					"term.write.apply",
					perfLogInterval,
					slog.LevelDebug,
					"terminal: term.Write slow",
					slog.Duration("dur", writeDur),
					slog.Int("bytes", len(data)),
				)
			}
		} else {
			_, _ = w.term.Write(data)
		}
		newSB = w.term.ScrollbackLen()
	}
	w.termMu.Unlock()

	nowNano := time.Now().UnixNano()
	if w.firstReadAt.CompareAndSwap(0, nowNano) {
		if w.onFirstRead != nil {
			w.onFirstRead()
		}
	}
	lastWrite := w.lastWriteAt.Load()
	if lastWrite != 0 {
		w.firstReadAfterWriteAt.CompareAndSwap(0, nowNano)
	}

	if perf {
		total := time.Since(start)
		if lockWait > perfSlowLock {
			logging.LogEvery(
				context.Background(),
				"term.write.lock",
				perfLogInterval,
				slog.LevelDebug,
				"terminal: termMu slow",
				slog.Duration("wait", lockWait),
				slog.Int("bytes", len(data)),
			)
		}
		if total > perfSlowWrite {
			logging.LogEvery(
				context.Background(),
				"term.write.total",
				perfLogInterval,
				slog.LevelDebug,
				"terminal: handleTerminalWrite slow",
				slog.Duration("dur", total),
				slog.Int("bytes", len(data)),
			)
		}
	}

	if newSB > oldSB {
		w.onScrollbackGrew(newSB - oldSB)
	}
	w.markDirty()
}

func (w *Window) enqueueWrite(ctx context.Context, data []byte, markDirty bool) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if len(data) == 0 {
		return nil
	}
	if w.closed.Load() {
		return &PaneClosedError{Reason: PaneClosedWindowClosed}
	}
	if w.exited.Load() {
		return &PaneClosedError{Reason: PaneClosedProcessExited}
	}
	if w.inputClosed.Load() {
		return &PaneClosedError{Reason: w.inputClosedReasonValue()}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	buf := append([]byte(nil), data...)
	req := writeRequest{data: buf, markDirty: markDirty}
	select {
	case w.writeCh <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Window) enqueueWriteBestEffort(data []byte, markDirty bool) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if len(data) == 0 {
		return nil
	}
	if w.closed.Load() {
		return &PaneClosedError{Reason: PaneClosedWindowClosed}
	}
	if w.exited.Load() {
		return &PaneClosedError{Reason: PaneClosedProcessExited}
	}
	if w.inputClosed.Load() {
		return &PaneClosedError{Reason: w.inputClosedReasonValue()}
	}
	buf := append([]byte(nil), data...)
	req := writeRequest{data: buf, markDirty: markDirty}
	select {
	case w.writeCh <- req:
		return nil
	default:
		return errWriteQueueFull
	}
}

func (w *Window) startPtyWriter(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-w.writeCh:
				if len(req.data) == 0 {
					continue
				}
				pty := w.currentPTY()
				if pty == nil {
					return
				}
				w.writeMu.Lock()
				n, err := pty.Write(req.data)
				w.writeMu.Unlock()
				if err != nil {
					if isPTYClosedWriteError(err) {
						w.markInputClosed(PaneClosedPTYClosed)
						return
					}
					return
				}
				if n != len(req.data) {
					w.markInputClosed(PaneClosedPTYClosed)
					return
				}
				w.bytesIn.Add(uint64(n))
				now := time.Now().UnixNano()
				w.lastWriteAt.Store(now)
				w.firstWriteAt.CompareAndSwap(0, now)
				if req.markDirty {
					w.markDirty()
				}
			}
		}
	}()
}

func ctxDone(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
