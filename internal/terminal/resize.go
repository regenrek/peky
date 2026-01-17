package terminal

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/logging"
)

// Resize resizes both the VT and PTY (PTY resize is best-effort).
func (w *Window) Resize(cols, rows int) error {
	if w == nil {
		return errors.New("terminal: nil window")
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	cols, rows = limits.Clamp(cols, rows)
	if w.closed.Load() {
		return errors.New("terminal: window closed")
	}

	if cols == w.cols && rows == w.rows {
		return nil
	}
	w.cols, w.rows = cols, rows

	w.termMu.Lock()
	if w.term != nil {
		w.term.Resize(cols, rows)
	}
	w.termMu.Unlock()

	w.ptyMu.Lock()
	pty := w.pty
	w.ptyMu.Unlock()
	if pty != nil {
		if err := pty.Resize(cols, rows); err != nil {
			logging.LogEvery(
				context.Background(),
				"terminal.pty.resize",
				2*time.Second,
				slog.LevelDebug,
				"terminal: pty resize failed",
				slog.Any("err", err),
				slog.Int("cols", cols),
				slog.Int("rows", rows),
			)
		} else {
			setPTYSlaveWinsizeBestEffort(pty, cols, rows)
			signalWINCHForPTY(w.PID(), pty)
		}
	}

	w.clampViewState()
	w.markDirty()
	return nil
}
